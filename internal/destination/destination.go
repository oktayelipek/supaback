package destination

import (
	"context"
	"fmt"
	"io"

	"github.com/supaback/supaback/internal/config"
)

// FileInfo holds metadata about a single backup file.
type FileInfo struct {
	Key  string `json:"key"`  // full path relative to destination root
	Size int64  `json:"size"` // bytes
}

type Destination interface {
	Write(ctx context.Context, key string, r io.Reader) (int64, error)
	// List returns top-level entries under prefix ("" = root).
	List(ctx context.Context, prefix string) ([]string, error)
	// ListFiles returns all files (recursively) under prefix ("" = root).
	ListFiles(ctx context.Context, prefix string) ([]FileInfo, error)
	// Read opens the file at key for reading. Caller must close.
	Read(ctx context.Context, key string) (io.ReadCloser, error)
	// Delete removes all files under key recursively.
	Delete(ctx context.Context, key string) error
	String() string
}

func New(cfg config.DestinationConfig) (Destination, error) {
	t := cfg.Type
	if t == "" {
		t = "local"
	}
	switch t {
	case "local":
		return newLocal(cfg.LocalPath)
	case "s3":
		return newS3(cfg.S3)
	case "sftp":
		return newSFTP(cfg.SFTP)
	default:
		return nil, fmt.Errorf("unknown destination type: %q", t)
	}
}
