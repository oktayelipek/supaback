package backup

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/url"
	"os"
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

	if err := checkIPv4Reachable(ctx, d.cfg.DatabaseURL); err != nil {
		return 0, err
	}

	connDSN, password := buildConnDSN(ctx, d.cfg.DatabaseURL)

	pr, pw := io.Pipe()

	var stderrBuf bytes.Buffer
	cmd := exec.CommandContext(ctx, "pg_dump",
		"--format=custom",
		"--no-owner",
		"--no-privileges",
		"-d", connDSN,
	)
	cmd.Stdout = pw
	cmd.Stderr = &stderrBuf
	// PGPASSWORD keeps the password out of the process list.
	cmd.Env = append(os.Environ(), "PGPASSWORD="+password)

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

// buildConnDSN converts a PostgreSQL connection URI into the libpq
// keyword-value DSN format and resolves the hostname to IPv4.
//
// Using the keyword-value format lets us set both "host" (for TLS SNI /
// certificate verification) and "hostaddr" (the actual TCP address) as
// separate keys.  This forces libpq to dial IPv4 while still validating
// the server certificate against the real hostname — important for Docker /
// Alpine where IPv6 routing is absent but the DNS returns AAAA records first.
//
// The password is returned separately so the caller can pass it via
// PGPASSWORD instead of embedding it in the DSN (keeps it off the process list).
func buildConnDSN(ctx context.Context, rawURL string) (dsn, password string) {
	u, err := url.Parse(rawURL)
	if err != nil || u.Hostname() == "" {
		return rawURL, ""
	}

	host := u.Hostname()
	port := u.Port()
	if port == "" {
		port = "5432"
	}
	user := u.User.Username()
	password, _ = u.User.Password()
	dbname := strings.TrimPrefix(u.Path, "/")
	if dbname == "" {
		dbname = "postgres"
	}
	sslmode := u.Query().Get("sslmode")
	if sslmode == "" {
		sslmode = "require" // Supabase mandates TLS
	}

	// Resolve hostname to an IPv4 address.
	hostaddr := ""
	if net.ParseIP(host).To4() == nil { // skip if already an IPv4 literal
		addrs, err := net.DefaultResolver.LookupHost(ctx, host)
		if err == nil {
			for _, a := range addrs {
				if net.ParseIP(a).To4() != nil {
					hostaddr = a
					slog.Info("pg_dump: resolved host to IPv4", "host", host, "addr", a)
					break
				}
			}
		}
		if hostaddr == "" {
			slog.Warn("pg_dump: no IPv4 address found, falling back to hostname", "host", host)
		}
	}

	parts := []string{
		"host=" + dsnEscape(host),
		"port=" + port,
		"dbname=" + dsnEscape(dbname),
		"user=" + dsnEscape(user),
		"sslmode=" + sslmode,
	}
	if hostaddr != "" {
		parts = append(parts, "hostaddr="+hostaddr)
	}
	return strings.Join(parts, " "), password
}

// checkIPv4Reachable verifies that the database hostname resolves to at least
// one IPv4 address. Newer Supabase projects use IPv6-only direct connections;
// those fail in Docker where IPv6 routing is absent.  Returning a clear error
// early is better than a cryptic pg_dump "Network unreachable" message.
func checkIPv4Reachable(ctx context.Context, rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil // let pg_dump surface the URL error itself
	}
	host := u.Hostname()
	if host == "" || net.ParseIP(host).To4() != nil {
		return nil // literal IPv4 or no host to check
	}
	addrs, err := net.DefaultResolver.LookupHost(ctx, host)
	if err != nil {
		return nil // DNS error — let pg_dump handle it
	}
	for _, a := range addrs {
		if net.ParseIP(a).To4() != nil {
			return nil
		}
	}
	return fmt.Errorf(
		"database host %q resolves to IPv6 only — "+
			"Docker containers cannot reach it without IPv6 routing. "+
			"Use the Supabase Session Pooler URL instead: "+
			"Project Settings → Database → Connection Pooling → Session mode (port 5432)",
		host,
	)
}

// dsnEscape quotes a DSN value if it contains spaces or special characters.
func dsnEscape(s string) string {
	if !strings.ContainsAny(s, " \\'=") {
		return s
	}
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `'`, `\'`)
	return "'" + s + "'"
}
