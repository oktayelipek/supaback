package destination

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type localDest struct {
	basePath string
}

func newLocal(basePath string) (*localDest, error) {
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return nil, fmt.Errorf("create local destination dir: %w", err)
	}
	return &localDest{basePath: basePath}, nil
}

func (l *localDest) Write(_ context.Context, key string, r io.Reader) (int64, error) {
	dest := filepath.Join(l.basePath, key)

	if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
		return 0, fmt.Errorf("create dir: %w", err)
	}

	f, err := os.Create(dest)
	if err != nil {
		return 0, fmt.Errorf("create file %s: %w", dest, err)
	}
	defer f.Close()

	n, err := io.Copy(f, r)
	if err != nil {
		return 0, fmt.Errorf("write file: %w", err)
	}
	return n, nil
}

func (l *localDest) String() string {
	return "local:" + l.basePath
}
