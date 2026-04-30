package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/supaback/supaback/internal/api"
	"github.com/supaback/supaback/internal/appstate"
	"github.com/supaback/supaback/internal/config"
	"github.com/supaback/supaback/internal/destination"
	"github.com/supaback/supaback/internal/scheduler"
	"github.com/supaback/supaback/internal/store"
)

func main() {
	configPath := flag.String("config", "config.yaml", "path to config file")
	flag.Parse()

	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))

	// Load base config (file + env vars) — no validation, may be incomplete
	cfg, err := config.Load(*configPath)
	if err != nil {
		slog.Error("failed to load config", "err", err)
		os.Exit(1)
	}

	db, err := store.New(cfg.Store.Path)
	if err != nil {
		slog.Error("failed to open store", "err", err)
		os.Exit(1)
	}
	defer db.Close()

	// Merge settings from DB on top of file/env config
	dbSettings, err := db.GetAllSettings(context.Background())
	if err != nil {
		slog.Warn("could not load settings from DB", "err", err)
	} else {
		cfg = config.MergeFromMap(cfg, dbSettings)
	}

	// Build destination (use local fallback if dest config is incomplete)
	dest, err := destination.New(cfg.Destination)
	if err != nil {
		slog.Warn("destination init failed, using local fallback", "err", err)
		fallbackCfg := config.DestinationConfig{Type: "local", LocalPath: "./backups"}
		dest, _ = destination.New(fallbackCfg)
	}

	state := appstate.New(cfg, dest)
	sched := scheduler.New(state, db)

	if err := sched.LoadAndStart(context.Background()); err != nil {
		slog.Error("failed to start scheduler", "err", err)
		os.Exit(1)
	}
	defer sched.Stop()

	if config.IsConfigured(cfg) {
		slog.Info("supaback configured", "url", cfg.Supabase.URL)
	} else {
		slog.Warn("supaback not configured — open the UI to set credentials")
	}

	srv := api.NewServer(cfg, state, db, sched)
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)

	go func() {
		slog.Info("server listening", "addr", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "err", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
}
