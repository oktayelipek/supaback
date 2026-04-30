package destination

import (
	"context"
	"fmt"
	"io"
	"io/fs"
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
	return io.Copy(f, r)
}

func (l *localDest) List(_ context.Context, prefix string) ([]string, error) {
	dir := l.basePath
	if prefix != "" {
		dir = filepath.Join(l.basePath, prefix)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() {
			names = append(names, e.Name())
		}
	}
	return names, nil
}

func (l *localDest) ListFiles(_ context.Context, prefix string) ([]FileInfo, error) {
	dir := l.basePath
	if prefix != "" {
		dir = filepath.Join(l.basePath, prefix)
	}
	var files []FileInfo
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}
		if d.IsDir() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(l.basePath, path)
		files = append(files, FileInfo{Key: filepath.ToSlash(rel), Size: info.Size()})
		return nil
	})
	return files, err
}

func (l *localDest) Read(_ context.Context, key string) (io.ReadCloser, error) {
	return os.Open(filepath.Join(l.basePath, filepath.FromSlash(key)))
}

func (l *localDest) Delete(_ context.Context, key string) error {
	return os.RemoveAll(filepath.Join(l.basePath, key))
}

func (l *localDest) String() string {
	return "local:" + l.basePath
}
