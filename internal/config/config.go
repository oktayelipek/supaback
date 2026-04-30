package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Supabase    SupabaseConfig    `yaml:"supabase"`
	Backup      BackupConfig      `yaml:"backup"`
	Destination DestinationConfig `yaml:"destination"`
	Store       StoreConfig       `yaml:"store"`
	Server      ServerConfig      `yaml:"server"`
}

// ForType returns a config copy scoped to a specific backup type.
func (c *Config) ForType(jobType string) *Config {
	cp := *c
	backup := c.Backup
	switch jobType {
	case "database":
		backup.IncludeStorage = false
	case "storage":
		backup.IncludeDatabase = false
	}
	cp.Backup = backup
	return &cp
}

type SupabaseConfig struct {
	URL        string `yaml:"url"`
	ServiceKey string `yaml:"service_key"`
	DatabaseURL string `yaml:"database_url"`
}

type BackupConfig struct {
	IncludeDatabase bool     `yaml:"include_database"`
	IncludeStorage  bool     `yaml:"include_storage"`
	Buckets         []string `yaml:"buckets"` // empty = all buckets
	Compress        bool     `yaml:"compress"`
}

type DestinationConfig struct {
	Type      string   `yaml:"type"` // "local" or "s3"
	LocalPath string   `yaml:"local_path"`
	S3        S3Config `yaml:"s3"`
}

type S3Config struct {
	Endpoint        string `yaml:"endpoint"`
	Region          string `yaml:"region"`
	Bucket          string `yaml:"bucket"`
	Prefix          string `yaml:"prefix"`
	AccessKeyID     string `yaml:"access_key_id"`
	SecretAccessKey string `yaml:"secret_access_key"`
	ForcePathStyle  bool   `yaml:"force_path_style"` // required for MinIO
}

type StoreConfig struct {
	Path string `yaml:"path"`
}

type ServerConfig struct {
	Port      int    `yaml:"port"`
	Host      string `yaml:"host"`
	StaticDir string `yaml:"static_dir"` // path to built frontend (apps/web/dist)
}

func Load(path string) (*Config, error) {
	cfg := defaults()

	if path != "" {
		data, err := os.ReadFile(path)
		if err != nil && !os.IsNotExist(err) {
			return nil, fmt.Errorf("read config: %w", err)
		}
		if err == nil {
			if err := yaml.Unmarshal(data, cfg); err != nil {
				return nil, fmt.Errorf("parse config: %w", err)
			}
		}
	}

	applyEnv(cfg)

	return cfg, nil
}

func defaults() *Config {
	return &Config{
		Backup: BackupConfig{
			IncludeDatabase: true,
			IncludeStorage:  true,
			Compress:        true,
		},
		Destination: DestinationConfig{
			Type:      "local",
			LocalPath: "./backups",
		},
		Store: StoreConfig{
			Path: "./supaback.db",
		},
		Server: ServerConfig{
			Host: "0.0.0.0",
			Port: 8080,
		},
	}
}

func applyEnv(cfg *Config) {
	// Supabase
	if v := os.Getenv("SUPABASE_URL"); v != "" {
		cfg.Supabase.URL = v
	}
	if v := os.Getenv("SUPABASE_SERVICE_KEY"); v != "" {
		cfg.Supabase.ServiceKey = v
	}
	if v := os.Getenv("SUPABASE_DB_URL"); v != "" {
		cfg.Supabase.DatabaseURL = v
	}
	// S3
	if v := os.Getenv("S3_ENDPOINT"); v != "" {
		cfg.Destination.S3.Endpoint = v
	}
	if v := os.Getenv("S3_BUCKET"); v != "" {
		cfg.Destination.S3.Bucket = v
	}
	if v := os.Getenv("S3_ACCESS_KEY_ID"); v != "" {
		cfg.Destination.S3.AccessKeyID = v
	}
	if v := os.Getenv("S3_SECRET_ACCESS_KEY"); v != "" {
		cfg.Destination.S3.SecretAccessKey = v
	}
	if v := os.Getenv("S3_REGION"); v != "" {
		cfg.Destination.S3.Region = v
	}
	// Server / paths
	if v := os.Getenv("PORT"); v != "" {
		if n, err := fmt.Sscanf(v, "%d", &cfg.Server.Port); n == 0 || err != nil {
			_ = err
		}
	}
	if v := os.Getenv("STATIC_DIR"); v != "" {
		cfg.Server.StaticDir = v
	}
	if v := os.Getenv("STORE_PATH"); v != "" {
		cfg.Store.Path = v
	}
	if v := os.Getenv("LOCAL_BACKUP_PATH"); v != "" {
		cfg.Destination.LocalPath = v
	}
}

// Validate checks if the config has all required fields to run a backup.
func Validate(cfg *Config) error {
	if cfg.Supabase.URL == "" {
		return fmt.Errorf("Supabase URL is required")
	}
	if cfg.Supabase.ServiceKey == "" {
		return fmt.Errorf("Supabase service key is required")
	}
	if cfg.Backup.IncludeDatabase && cfg.Supabase.DatabaseURL == "" {
		return fmt.Errorf("database URL is required when database backup is enabled")
	}
	if cfg.Destination.Type == "s3" {
		s3 := cfg.Destination.S3
		if s3.Bucket == "" {
			return fmt.Errorf("S3 bucket is required")
		}
		if s3.AccessKeyID == "" || s3.SecretAccessKey == "" {
			return fmt.Errorf("S3 credentials are required")
		}
	}
	return nil
}
