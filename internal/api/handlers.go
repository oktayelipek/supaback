package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/supaback/supaback/internal/appstate"
	"github.com/supaback/supaback/internal/backup"
	"github.com/supaback/supaback/internal/config"
	"github.com/supaback/supaback/internal/destination"
	"github.com/supaback/supaback/internal/scheduler"
	"github.com/supaback/supaback/internal/store"
)

type Handler struct {
	state     *appstate.State
	store     *store.Store
	scheduler *scheduler.Scheduler
}

// ── helpers ──────────────────────────────────────────────────────────────────

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func parseID(r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	return id, err == nil
}

// ── health ────────────────────────────────────────────────────────────────────

func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	cfg, _ := h.state.Get()
	writeJSON(w, http.StatusOK, map[string]any{
		"status":     "ok",
		"configured": config.IsConfigured(cfg),
	})
}

// ── settings ──────────────────────────────────────────────────────────────────

func (h *Handler) GetSettings(w http.ResponseWriter, r *http.Request) {
	cfg, _ := h.state.Get()
	writeJSON(w, http.StatusOK, map[string]any{
		"configured":                    config.IsConfigured(cfg),
		config.KeySupabaseURL:           cfg.Supabase.URL,
		config.KeySupabaseServiceKey:    cfg.Supabase.ServiceKey,
		config.KeySupabaseDBURL:         cfg.Supabase.DatabaseURL,
		config.KeyIncludeDatabase:       strconv.FormatBool(cfg.Backup.IncludeDatabase),
		config.KeyIncludeStorage:        strconv.FormatBool(cfg.Backup.IncludeStorage),
		config.KeyCompress:              strconv.FormatBool(cfg.Backup.Compress),
		config.KeyBuckets:               strings.Join(cfg.Backup.Buckets, ","),
		config.KeyDestType:              cfg.Destination.Type,
		config.KeyLocalPath:             cfg.Destination.LocalPath,
		config.KeyS3Endpoint:            cfg.Destination.S3.Endpoint,
		config.KeyS3Region:              cfg.Destination.S3.Region,
		config.KeyS3Bucket:              cfg.Destination.S3.Bucket,
		config.KeyS3Prefix:              cfg.Destination.S3.Prefix,
		config.KeyS3AccessKeyID:         cfg.Destination.S3.AccessKeyID,
		config.KeyS3SecretAccessKey:     cfg.Destination.S3.SecretAccessKey,
		config.KeyS3ForcePathStyle:      strconv.FormatBool(cfg.Destination.S3.ForcePathStyle),
	})
}

func (h *Handler) UpdateSettings(w http.ResponseWriter, r *http.Request) {
	var incoming map[string]string
	if err := json.NewDecoder(r.Body).Decode(&incoming); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	// Remove meta keys that shouldn't be stored
	delete(incoming, "configured")

	if err := h.store.SetSettings(r.Context(), incoming); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Reload config from DB and rebuild destination
	dbSettings, err := h.store.GetAllSettings(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	currentCfg, _ := h.state.Get()
	newCfg := config.MergeFromMap(currentCfg, dbSettings)

	newDest, err := destination.New(newCfg.Destination)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid destination config: "+err.Error())
		return
	}

	h.state.Update(newCfg, newDest)

	writeJSON(w, http.StatusOK, map[string]any{"status": "saved", "configured": config.IsConfigured(newCfg)})
}

// ── jobs ──────────────────────────────────────────────────────────────────────

func (h *Handler) ListJobs(w http.ResponseWriter, r *http.Request) {
	limit := 50
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}
	jobs, err := h.store.ListJobs(r.Context(), limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if jobs == nil {
		jobs = []store.Job{}
	}
	writeJSON(w, http.StatusOK, jobs)
}

type triggerRequest struct {
	Type string `json:"type"`
}

func (h *Handler) TriggerJob(w http.ResponseWriter, r *http.Request) {
	var req triggerRequest
	_ = json.NewDecoder(r.Body).Decode(&req)
	if req.Type == "" {
		req.Type = "full"
	}
	if req.Type != "full" && req.Type != "database" && req.Type != "storage" {
		writeError(w, http.StatusBadRequest, "type must be full, database, or storage")
		return
	}

	cfg, dest := h.state.Get()
	if !config.IsConfigured(cfg) {
		writeError(w, http.StatusBadRequest, "app is not configured — set Supabase credentials in Settings first")
		return
	}

	runner := backup.NewRunner(cfg.ForType(req.Type), h.store, dest)
	go func() { _ = runner.Run(context.Background()) }()

	writeJSON(w, http.StatusAccepted, map[string]string{"status": "backup started", "type": req.Type})
}

// ── schedules ─────────────────────────────────────────────────────────────────

func (h *Handler) ListSchedules(w http.ResponseWriter, r *http.Request) {
	schedules, err := h.store.ListSchedules(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if schedules == nil {
		schedules = []store.Schedule{}
	}
	writeJSON(w, http.StatusOK, schedules)
}

type createScheduleRequest struct {
	Name     string `json:"name"`
	CronExpr string `json:"cron_expr"`
	Type     string `json:"type"`
}

func (h *Handler) CreateSchedule(w http.ResponseWriter, r *http.Request) {
	var req createScheduleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	req.CronExpr = strings.TrimSpace(req.CronExpr)
	if req.Name == "" || req.CronExpr == "" {
		writeError(w, http.StatusBadRequest, "name and cron_expr are required")
		return
	}
	if req.Type == "" {
		req.Type = "full"
	}

	sc, err := h.store.CreateSchedule(r.Context(), req.Name, req.CronExpr, req.Type)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if err := h.scheduler.Add(sc.ID, sc.CronExpr, sc.Type); err != nil {
		_ = h.store.DeleteSchedule(r.Context(), sc.ID)
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, sc)
}

func (h *Handler) DeleteSchedule(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r)
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	h.scheduler.Remove(id)
	if err := h.store.DeleteSchedule(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type toggleRequest struct {
	Enabled bool `json:"enabled"`
}

func (h *Handler) ToggleSchedule(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r)
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	var req toggleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	sc, err := h.store.GetSchedule(r.Context(), id)
	if err != nil || sc == nil {
		writeError(w, http.StatusNotFound, "schedule not found")
		return
	}
	if err := h.store.ToggleSchedule(r.Context(), id, req.Enabled); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := h.scheduler.SetEnabled(id, sc.CronExpr, sc.Type, req.Enabled); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	sc.Enabled = req.Enabled
	writeJSON(w, http.StatusOK, sc)
}
