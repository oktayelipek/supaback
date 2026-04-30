package backup

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/supaback/supaback/internal/config"
	"github.com/supaback/supaback/internal/destination"
)

type StorageBackup struct {
	cfg        config.SupabaseConfig
	dest       destination.Destination
	buckets    []string // empty = all
	httpClient *http.Client
}

func NewStorageBackup(cfg config.SupabaseConfig, dest destination.Destination, buckets []string) *StorageBackup {
	return &StorageBackup{
		cfg:        cfg,
		dest:       dest,
		buckets:    buckets,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

func (s *StorageBackup) Run(ctx context.Context) (int64, error) {
	buckets, err := s.listBuckets(ctx)
	if err != nil {
		return 0, fmt.Errorf("list buckets: %w", err)
	}

	if len(s.buckets) > 0 {
		buckets = filter(buckets, s.buckets)
	}

	slog.Info("storage backup started", "buckets", buckets)

	var totalBytes int64
	datePrefix := time.Now().UTC().Format("2006-01-02")

	for _, bucket := range buckets {
		objects, err := s.listObjects(ctx, bucket, "")
		if err != nil {
			return totalBytes, fmt.Errorf("list objects in %s: %w", bucket, err)
		}

		for _, obj := range objects {
			key := fmt.Sprintf("%s/storage/%s/%s", datePrefix, bucket, obj)
			n, err := s.downloadObject(ctx, bucket, obj, key)
			if err != nil {
				slog.Warn("failed to backup object", "bucket", bucket, "object", obj, "err", err)
				continue
			}
			totalBytes += n
		}
		slog.Info("bucket backup complete", "bucket", bucket, "objects", len(objects))
	}

	return totalBytes, nil
}

type bucket struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func (s *StorageBackup) listBuckets(ctx context.Context) ([]string, error) {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet,
		s.cfg.URL+"/storage/v1/bucket", nil)
	s.setHeaders(req)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("list buckets %d: %s", resp.StatusCode, body)
	}

	var buckets []bucket
	if err := json.NewDecoder(resp.Body).Decode(&buckets); err != nil {
		return nil, err
	}

	names := make([]string, len(buckets))
	for i, b := range buckets {
		names[i] = b.Name
	}
	return names, nil
}

type storageObject struct {
	Name string `json:"name"`
	ID   *string `json:"id"` // nil = folder
}

type listRequest struct {
	Prefix string `json:"prefix"`
	Limit  int    `json:"limit"`
	Offset int    `json:"offset"`
}

// listObjects recursively resolves all file paths in a bucket.
func (s *StorageBackup) listObjects(ctx context.Context, bucket, prefix string) ([]string, error) {
	const pageSize = 100
	var all []string
	offset := 0

	for {
		body, _ := json.Marshal(listRequest{Prefix: prefix, Limit: pageSize, Offset: offset})
		req, _ := http.NewRequestWithContext(ctx, http.MethodPost,
			s.cfg.URL+"/storage/v1/object/list/"+bucket,
			strings.NewReader(string(body)),
		)
		s.setHeaders(req)
		req.Header.Set("Content-Type", "application/json")

		resp, err := s.httpClient.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			b, _ := io.ReadAll(resp.Body)
			return nil, fmt.Errorf("list objects %d: %s", resp.StatusCode, b)
		}

		var objects []storageObject
		if err := json.NewDecoder(resp.Body).Decode(&objects); err != nil {
			return nil, err
		}

		for _, obj := range objects {
			path := obj.Name
			if prefix != "" {
				path = prefix + "/" + obj.Name
			}
			if obj.ID == nil {
				// folder — recurse
				sub, err := s.listObjects(ctx, bucket, path)
				if err != nil {
					return nil, err
				}
				all = append(all, sub...)
			} else {
				all = append(all, path)
			}
		}

		if len(objects) < pageSize {
			break
		}
		offset += pageSize
	}

	return all, nil
}

func (s *StorageBackup) downloadObject(ctx context.Context, bucket, objectPath, destKey string) (int64, error) {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet,
		s.cfg.URL+"/storage/v1/object/"+bucket+"/"+objectPath, nil)
	s.setHeaders(req)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("download %s/%s %d: %s", bucket, objectPath, resp.StatusCode, b)
	}

	return s.dest.Write(ctx, destKey, resp.Body)
}

func (s *StorageBackup) setHeaders(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+s.cfg.ServiceKey)
	req.Header.Set("apikey", s.cfg.ServiceKey)
}

func filter(all, keep []string) []string {
	set := make(map[string]struct{}, len(keep))
	for _, k := range keep {
		set[k] = struct{}{}
	}
	var out []string
	for _, v := range all {
		if _, ok := set[v]; ok {
			out = append(out, v)
		}
	}
	return out
}
