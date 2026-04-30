package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

type JobStatus string

const (
	StatusRunning JobStatus = "running"
	StatusSuccess JobStatus = "success"
	StatusFailed  JobStatus = "failed"
)

type Job struct {
	ID          int64
	StartedAt   time.Time
	FinishedAt  *time.Time
	Status      JobStatus
	Type        string // "database", "storage", "full"
	Destination string
	SizeBytes   int64
	ErrorMsg    string
}

type Store struct {
	db *sql.DB
}

func New(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return s, nil
}

func (s *Store) migrate() error {
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS jobs (
			id          INTEGER PRIMARY KEY AUTOINCREMENT,
			started_at  DATETIME NOT NULL,
			finished_at DATETIME,
			status      TEXT NOT NULL,
			type        TEXT NOT NULL,
			destination TEXT NOT NULL,
			size_bytes  INTEGER NOT NULL DEFAULT 0,
			error_msg   TEXT NOT NULL DEFAULT ''
		);
		CREATE INDEX IF NOT EXISTS idx_jobs_started_at ON jobs(started_at DESC);

		CREATE TABLE IF NOT EXISTS schedules (
			id          INTEGER PRIMARY KEY AUTOINCREMENT,
			name        TEXT NOT NULL,
			cron_expr   TEXT NOT NULL,
			type        TEXT NOT NULL DEFAULT 'full',
			enabled     INTEGER NOT NULL DEFAULT 1,
			created_at  DATETIME NOT NULL,
			last_run_at DATETIME
		);

		CREATE TABLE IF NOT EXISTS settings (
			key   TEXT PRIMARY KEY,
			value TEXT NOT NULL
		);
	`)
	return err
}

func (s *Store) CreateJob(ctx context.Context, jobType, destination string) (int64, error) {
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO jobs (started_at, status, type, destination) VALUES (?, ?, ?, ?)`,
		time.Now().UTC(), StatusRunning, jobType, destination,
	)
	if err != nil {
		return 0, fmt.Errorf("create job: %w", err)
	}
	return res.LastInsertId()
}

func (s *Store) CompleteJob(ctx context.Context, id int64, status JobStatus, sizeBytes int64, errMsg string) error {
	now := time.Now().UTC()
	_, err := s.db.ExecContext(ctx,
		`UPDATE jobs SET finished_at=?, status=?, size_bytes=?, error_msg=? WHERE id=?`,
		now, status, sizeBytes, errMsg, id,
	)
	return err
}

func (s *Store) ListJobs(ctx context.Context, limit int) ([]Job, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, started_at, finished_at, status, type, destination, size_bytes, error_msg
		 FROM jobs ORDER BY started_at DESC LIMIT ?`, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []Job
	for rows.Next() {
		var j Job
		var finishedAt sql.NullTime
		if err := rows.Scan(&j.ID, &j.StartedAt, &finishedAt, &j.Status, &j.Type, &j.Destination, &j.SizeBytes, &j.ErrorMsg); err != nil {
			return nil, err
		}
		if finishedAt.Valid {
			j.FinishedAt = &finishedAt.Time
		}
		jobs = append(jobs, j)
	}
	return jobs, rows.Err()
}

func (s *Store) Close() error {
	return s.db.Close()
}
