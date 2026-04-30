package backup

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/supaback/supaback/internal/config"
	"github.com/supaback/supaback/internal/destination"
	"github.com/supaback/supaback/internal/store"
)

type Runner struct {
	cfg   *config.Config
	store *store.Store
	dest  destination.Destination
}

func NewRunner(cfg *config.Config, s *store.Store, dest destination.Destination) *Runner {
	return &Runner{cfg: cfg, store: s, dest: dest}
}

func (r *Runner) Run(ctx context.Context) error {
	if err := config.Validate(r.cfg); err != nil {
		return fmt.Errorf("not configured: %w", err)
	}

	jobType := r.jobType()
	jobID, err := r.store.CreateJob(ctx, jobType, r.dest.String())
	if err != nil {
		return fmt.Errorf("create job record: %w", err)
	}

	slog.Info("backup job started", "id", jobID, "type", jobType, "destination", r.dest.String())

	var totalBytes int64
	var runErr error

	if r.cfg.Backup.IncludeDatabase {
		db := NewDatabaseBackup(r.cfg.Supabase, r.dest, r.cfg.Backup.Compress)
		n, err := db.Run(ctx)
		if err != nil {
			runErr = fmt.Errorf("database backup: %w", err)
		} else {
			totalBytes += n
		}
	}

	if runErr == nil && r.cfg.Backup.IncludeStorage {
		st := NewStorageBackup(r.cfg.Supabase, r.dest, r.cfg.Backup.Buckets)
		n, err := st.Run(ctx)
		if err != nil {
			runErr = fmt.Errorf("storage backup: %w", err)
		} else {
			totalBytes += n
		}
	}

	status := store.StatusSuccess
	errMsg := ""
	if runErr != nil {
		status = store.StatusFailed
		errMsg = runErr.Error()
		slog.Error("backup job failed", "id", jobID, "err", runErr)
	} else {
		slog.Info("backup job succeeded", "id", jobID, "bytes", totalBytes)
	}

	if err := r.store.CompleteJob(ctx, jobID, status, totalBytes, errMsg); err != nil {
		slog.Warn("failed to update job record", "id", jobID, "err", err)
	}

	return runErr
}

func (r *Runner) jobType() string {
	switch {
	case r.cfg.Backup.IncludeDatabase && r.cfg.Backup.IncludeStorage:
		return "full"
	case r.cfg.Backup.IncludeDatabase:
		return "database"
	default:
		return "storage"
	}
}
