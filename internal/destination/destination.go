package destination

import (
	"context"
	"fmt"
	"io"

	"github.com/supaback/supaback/internal/config"
)

type Destination interface {
	Write(ctx context.Context, key string, r io.Reader) (int64, error)
	String() string
}

func New(cfg config.DestinationConfig) (Destination, error) {
	switch cfg.Type {
	case "local":
		return newLocal(cfg.LocalPath)
	case "s3":
		return newS3(cfg.S3)
	default:
		return nil, fmt.Errorf("unknown destination type: %q", cfg.Type)
	}
}
