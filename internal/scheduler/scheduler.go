package scheduler

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/robfig/cron/v3"
	"github.com/supaback/supaback/internal/appstate"
	"github.com/supaback/supaback/internal/backup"
	"github.com/supaback/supaback/internal/store"
)

type Scheduler struct {
	cron  *cron.Cron
	state *appstate.State
	store *store.Store

	mu      sync.Mutex
	entries map[int64]cron.EntryID
}

func New(state *appstate.State, s *store.Store) *Scheduler {
	return &Scheduler{
		cron:    cron.New(cron.WithSeconds()),
		state:   state,
		store:   s,
		entries: make(map[int64]cron.EntryID),
	}
}

func (s *Scheduler) LoadAndStart(ctx context.Context) error {
	schedules, err := s.store.ListSchedules(ctx)
	if err != nil {
		return fmt.Errorf("load schedules: %w", err)
	}
	for _, sc := range schedules {
		if !sc.Enabled {
			continue
		}
		if err := s.register(sc.ID, sc.CronExpr, sc.Type); err != nil {
			slog.Warn("failed to register schedule", "id", sc.ID, "name", sc.Name, "err", err)
		} else {
			slog.Info("schedule registered", "id", sc.ID, "name", sc.Name, "expr", sc.CronExpr)
		}
	}
	s.cron.Start()
	return nil
}

func (s *Scheduler) Stop() { s.cron.Stop() }

func (s *Scheduler) Add(scheduleID int64, cronExpr, jobType string) error {
	return s.register(scheduleID, cronExpr, jobType)
}

func (s *Scheduler) Remove(scheduleID int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if id, ok := s.entries[scheduleID]; ok {
		s.cron.Remove(id)
		delete(s.entries, scheduleID)
	}
}

func (s *Scheduler) SetEnabled(scheduleID int64, cronExpr, jobType string, enabled bool) error {
	if enabled {
		return s.register(scheduleID, cronExpr, jobType)
	}
	s.Remove(scheduleID)
	return nil
}

func (s *Scheduler) register(scheduleID int64, cronExpr, jobType string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if id, ok := s.entries[scheduleID]; ok {
		s.cron.Remove(id)
	}
	entryID, err := s.cron.AddFunc(cronExpr, s.makeJob(scheduleID, jobType))
	if err != nil {
		return fmt.Errorf("invalid cron expression %q: %w", cronExpr, err)
	}
	s.entries[scheduleID] = entryID
	return nil
}

func (s *Scheduler) makeJob(scheduleID int64, jobType string) func() {
	return func() {
		ctx := context.Background()
		slog.Info("scheduled backup triggered", "scheduleID", scheduleID, "type", jobType)
		_ = s.store.TouchSchedule(ctx, scheduleID)

		cfg, dest := s.state.Get()
		runner := backup.NewRunner(cfg.ForType(jobType), s.store, dest)
		if err := runner.Run(ctx); err != nil {
			slog.Error("scheduled backup failed", "scheduleID", scheduleID, "err", err)
		}
	}
}
