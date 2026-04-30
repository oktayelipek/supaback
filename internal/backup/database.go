package backup

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os/exec"
	"strings"
	"time"

	"github.com/supaback/supaback/internal/config"
	"github.com/supaback/supaback/internal/destination"
)

type DatabaseBackup struct {
	cfg  config.SupabaseConfig
	dest destination.Destination
	gzip bool
}

func NewDatabaseBackup(cfg config.SupabaseConfig, dest destination.Destination, compress bool) *DatabaseBackup {
	return &DatabaseBackup{cfg: cfg, dest: dest, gzip: compress}
}

func (d *DatabaseBackup) Run(ctx context.Context) (int64, error) {
	if _, err := exec.LookPath("pg_dump"); err != nil {
		return 0, fmt.Errorf("pg_dump not found in PATH — install postgresql-client")
	}

	key := fmt.Sprintf("%s/database/postgres_%s.dump",
		time.Now().UTC().Format("2006-01-02"),
		time.Now().UTC().Format("20060102_150405"),
	)
	if d.gzip {
		key += ".gz"
	}

	slog.Info("starting database backup", "key", key)

	pr, pw := io.Pipe()

	var stderrBuf bytes.Buffer
	cmd := exec.CommandContext(ctx, "pg_dump",
		"--format=custom",
		"--no-owner",
		"--no-privileges",
		d.cfg.DatabaseURL,
	)
	cmd.Stdout = pw
	cmd.Stderr = &stderrBuf

	var cmdErr error
	go func() {
		cmdErr = cmd.Run()
		if cmdErr != nil {
			if msg := strings.TrimSpace(stderrBuf.String()); msg != "" {
				slog.Error("pg_dump stderr", "output", msg)
				cmdErr = fmt.Errorf("%w: %s", cmdErr, msg)
			}
		}
		pw.CloseWithError(cmdErr)
	}()

	var reader io.Reader = pr
	if d.gzip {
		pr2, pw2 := io.Pipe()
		go func() {
			gz := gzip.NewWriter(pw2)
			if _, err := io.Copy(gz, pr); err != nil {
				pw2.CloseWithError(err)
				return
			}
			gz.Close()
			pw2.Close()
		}()
		reader = pr2
	}

	n, err := d.dest.Write(ctx, key, reader)
	if err != nil {
		return 0, fmt.Errorf("write database backup: %w", err)
	}

	if cmdErr != nil {
		return 0, fmt.Errorf("pg_dump failed: %w", cmdErr)
	}

	slog.Info("database backup complete", "key", key, "bytes", n)
	return n, nil
}
