package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

type Schedule struct {
	ID        int64
	Name      string
	CronExpr  string
	Type      string // "full", "database", "storage"
	Enabled   bool
	CreatedAt time.Time
	LastRunAt *time.Time
}

func (s *Store) CreateSchedule(ctx context.Context, name, cronExpr, jobType string) (*Schedule, error) {
	now := time.Now().UTC()
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO schedules (name, cron_expr, type, enabled, created_at) VALUES (?, ?, ?, 1, ?)`,
		name, cronExpr, jobType, now,
	)
	if err != nil {
		return nil, fmt.Errorf("create schedule: %w", err)
	}
	id, _ := res.LastInsertId()
	return &Schedule{
		ID:        id,
		Name:      name,
		CronExpr:  cronExpr,
		Type:      jobType,
		Enabled:   true,
		CreatedAt: now,
	}, nil
}

func (s *Store) ListSchedules(ctx context.Context) ([]Schedule, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, name, cron_expr, type, enabled, created_at, last_run_at FROM schedules ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var schedules []Schedule
	for rows.Next() {
		var sc Schedule
		var enabled int
		var lastRun sql.NullTime
		if err := rows.Scan(&sc.ID, &sc.Name, &sc.CronExpr, &sc.Type, &enabled, &sc.CreatedAt, &lastRun); err != nil {
			return nil, err
		}
		sc.Enabled = enabled == 1
		if lastRun.Valid {
			sc.LastRunAt = &lastRun.Time
		}
		schedules = append(schedules, sc)
	}
	return schedules, rows.Err()
}

func (s *Store) GetSchedule(ctx context.Context, id int64) (*Schedule, error) {
	var sc Schedule
	var enabled int
	var lastRun sql.NullTime
	err := s.db.QueryRowContext(ctx,
		`SELECT id, name, cron_expr, type, enabled, created_at, last_run_at FROM schedules WHERE id=?`, id,
	).Scan(&sc.ID, &sc.Name, &sc.CronExpr, &sc.Type, &enabled, &sc.CreatedAt, &lastRun)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	sc.Enabled = enabled == 1
	if lastRun.Valid {
		sc.LastRunAt = &lastRun.Time
	}
	return &sc, nil
}

func (s *Store) DeleteSchedule(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM schedules WHERE id=?`, id)
	return err
}

func (s *Store) ToggleSchedule(ctx context.Context, id int64, enabled bool) error {
	v := 0
	if enabled {
		v = 1
	}
	_, err := s.db.ExecContext(ctx, `UPDATE schedules SET enabled=? WHERE id=?`, v, id)
	return err
}

func (s *Store) TouchSchedule(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `UPDATE schedules SET last_run_at=? WHERE id=?`, time.Now().UTC(), id)
	return err
}
