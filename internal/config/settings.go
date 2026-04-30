package config

import (
	"strconv"
	"strings"
)

// Keys used in the settings DB table.
const (
	KeySupabaseURL        = "supabase_url"
	KeySupabaseServiceKey = "supabase_service_key"
	KeySupabaseDBURL      = "supabase_db_url"
	KeyIncludeDatabase    = "backup_include_database"
	KeyIncludeStorage     = "backup_include_storage"
	KeyCompress           = "backup_compress"
	KeyBuckets            = "backup_buckets"
	KeyDestType           = "destination_type"
	KeyLocalPath          = "destination_local_path"
	KeyS3Endpoint         = "destination_s3_endpoint"
	KeyS3Region           = "destination_s3_region"
	KeyS3Bucket           = "destination_s3_bucket"
	KeyS3Prefix           = "destination_s3_prefix"
	KeyS3AccessKeyID      = "destination_s3_access_key_id"
	KeyS3SecretAccessKey  = "destination_s3_secret_access_key"
	KeyS3ForcePathStyle   = "destination_s3_force_path_style"
)

// MergeFromMap applies settings from the DB map on top of an existing Config.
// Only non-empty values override.
func MergeFromMap(cfg *Config, m map[string]string) *Config {
	cp := *cfg
	sup := cp.Supabase
	bak := cp.Backup
	dest := cp.Destination
	s3 := dest.S3

	if v, ok := m[KeySupabaseURL]; ok && v != "" {
		sup.URL = v
	}
	if v, ok := m[KeySupabaseServiceKey]; ok && v != "" {
		sup.ServiceKey = v
	}
	if v, ok := m[KeySupabaseDBURL]; ok && v != "" {
		sup.DatabaseURL = v
	}
	if v, ok := m[KeyIncludeDatabase]; ok && v != "" {
		bak.IncludeDatabase = v == "true"
	}
	if v, ok := m[KeyIncludeStorage]; ok && v != "" {
		bak.IncludeStorage = v == "true"
	}
	if v, ok := m[KeyCompress]; ok && v != "" {
		bak.Compress = v == "true"
	}
	if v, ok := m[KeyBuckets]; ok {
		if v == "" {
			bak.Buckets = nil
		} else {
			bak.Buckets = strings.Split(v, ",")
		}
	}
	if v, ok := m[KeyDestType]; ok && v != "" {
		dest.Type = v
	}
	if v, ok := m[KeyLocalPath]; ok && v != "" {
		dest.LocalPath = v
	}
	if v, ok := m[KeyS3Endpoint]; ok {
		s3.Endpoint = v
	}
	if v, ok := m[KeyS3Region]; ok && v != "" {
		s3.Region = v
	}
	if v, ok := m[KeyS3Bucket]; ok && v != "" {
		s3.Bucket = v
	}
	if v, ok := m[KeyS3Prefix]; ok {
		s3.Prefix = v
	}
	if v, ok := m[KeyS3AccessKeyID]; ok && v != "" {
		s3.AccessKeyID = v
	}
	if v, ok := m[KeyS3SecretAccessKey]; ok && v != "" {
		s3.SecretAccessKey = v
	}
	if v, ok := m[KeyS3ForcePathStyle]; ok && v != "" {
		s3.ForcePathStyle, _ = strconv.ParseBool(v)
	}

	dest.S3 = s3
	cp.Supabase = sup
	cp.Backup = bak
	cp.Destination = dest
	return &cp
}

// ToMap converts a Config to a flat settings map for storage.
func ToMap(cfg *Config) map[string]string {
	buckets := ""
	if len(cfg.Backup.Buckets) > 0 {
		buckets = strings.Join(cfg.Backup.Buckets, ",")
	}
	return map[string]string{
		KeySupabaseURL:        cfg.Supabase.URL,
		KeySupabaseServiceKey: cfg.Supabase.ServiceKey,
		KeySupabaseDBURL:      cfg.Supabase.DatabaseURL,
		KeyIncludeDatabase:    strconv.FormatBool(cfg.Backup.IncludeDatabase),
		KeyIncludeStorage:     strconv.FormatBool(cfg.Backup.IncludeStorage),
		KeyCompress:           strconv.FormatBool(cfg.Backup.Compress),
		KeyBuckets:            buckets,
		KeyDestType:           cfg.Destination.Type,
		KeyLocalPath:          cfg.Destination.LocalPath,
		KeyS3Endpoint:         cfg.Destination.S3.Endpoint,
		KeyS3Region:           cfg.Destination.S3.Region,
		KeyS3Bucket:           cfg.Destination.S3.Bucket,
		KeyS3Prefix:           cfg.Destination.S3.Prefix,
		KeyS3AccessKeyID:      cfg.Destination.S3.AccessKeyID,
		KeyS3SecretAccessKey:  cfg.Destination.S3.SecretAccessKey,
		KeyS3ForcePathStyle:   strconv.FormatBool(cfg.Destination.S3.ForcePathStyle),
	}
}

// IsConfigured returns true if minimum required fields are present.
func IsConfigured(cfg *Config) bool {
	return cfg.Supabase.URL != "" && cfg.Supabase.ServiceKey != ""
}
