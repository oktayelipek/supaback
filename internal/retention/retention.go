package retention

import (
	"context"
	"log/slog"
	"sort"
	"time"
)

type Config struct {
	KeepLast int // keep N most recent backup dates; 0 = disabled
	KeepDays int // keep backups newer than N days; 0 = disabled
}

// Dest is the subset of destination.Destination needed for retention.
type Dest interface {
	List(ctx context.Context, prefix string) ([]string, error)
	Delete(ctx context.Context, key string) error
}

// Apply deletes backup date-directories that fall outside the retention policy.
// It only acts on top-level entries that match the YYYY-MM-DD format.
func Apply(ctx context.Context, dest Dest, cfg Config) error {
	if cfg.KeepLast == 0 && cfg.KeepDays == 0 {
		return nil
	}

	entries, err := dest.List(ctx, "")
	if err != nil {
		return err
	}

	// Keep only valid date directories.
	var dates []string
	for _, e := range entries {
		if _, err := time.Parse("2006-01-02", e); err == nil {
			dates = append(dates, e)
		}
	}

	// Sort newest → oldest.
	sort.Sort(sort.Reverse(sort.StringSlice(dates)))

	keep := buildKeepSet(dates, cfg)

	deleted := 0
	for _, date := range dates {
		if keep[date] {
			continue
		}
		slog.Info("retention: removing old backup", "date", date)
		if err := dest.Delete(ctx, date); err != nil {
			slog.Warn("retention: delete failed", "date", date, "err", err)
		} else {
			deleted++
		}
	}
	if deleted > 0 {
		slog.Info("retention: cleanup complete", "removed", deleted)
	}
	return nil
}

func buildKeepSet(dates []string, cfg Config) map[string]bool {
	keep := make(map[string]bool)

	// Rule 1: keep the N most recent.
	if cfg.KeepLast > 0 {
		for i := 0; i < cfg.KeepLast && i < len(dates); i++ {
			keep[dates[i]] = true
		}
	}

	// Rule 2: keep anything newer than N days.
	if cfg.KeepDays > 0 {
		cutoff := time.Now().UTC().AddDate(0, 0, -cfg.KeepDays)
		for _, d := range dates {
			t, _ := time.Parse("2006-01-02", d)
			if !t.Before(cutoff) {
				keep[d] = true
			}
		}
	}

	return keep
}
