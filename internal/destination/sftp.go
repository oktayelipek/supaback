package destination

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	supconfig "github.com/supaback/supaback/internal/config"
)

type sftpDest struct {
	cfg        supconfig.SFTPConfig
	mu         sync.Mutex
	sshConn    *ssh.Client
	sftpClient *sftp.Client
}

func newSFTP(cfg supconfig.SFTPConfig) (*sftpDest, error) {
	d := &sftpDest{cfg: cfg}
	if err := d.connect(); err != nil {
		return nil, err
	}
	return d, nil
}

func (d *sftpDest) connect() error {
	port := d.cfg.Port
	if port == 0 {
		port = 22
	}

	var authMethods []ssh.AuthMethod

	if d.cfg.Password != "" {
		authMethods = append(authMethods, ssh.Password(d.cfg.Password))
	}
	if d.cfg.KeyPath != "" {
		key, err := os.ReadFile(d.cfg.KeyPath)
		if err != nil {
			return fmt.Errorf("read ssh key: %w", err)
		}
		signer, err := ssh.ParsePrivateKey(key)
		if err != nil {
			return fmt.Errorf("parse ssh key: %w", err)
		}
		authMethods = append(authMethods, ssh.PublicKeys(signer))
	}
	if len(authMethods) == 0 {
		return fmt.Errorf("sftp: no auth method configured (set password or key_path)")
	}

	sshCfg := &ssh.ClientConfig{
		User:            d.cfg.User,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), //nolint:gosec — user-controlled server
		Timeout:         15 * time.Second,
	}

	conn, err := ssh.Dial("tcp", net.JoinHostPort(d.cfg.Host, fmt.Sprintf("%d", port)), sshCfg)
	if err != nil {
		return fmt.Errorf("ssh dial %s:%d: %w", d.cfg.Host, port, err)
	}

	client, err := sftp.NewClient(conn)
	if err != nil {
		conn.Close()
		return fmt.Errorf("sftp client: %w", err)
	}

	if d.sftpClient != nil {
		_ = d.sftpClient.Close()
	}
	if d.sshConn != nil {
		_ = d.sshConn.Close()
	}
	d.sshConn = conn
	d.sftpClient = client
	return nil
}

func (d *sftpDest) client() (*sftp.Client, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	// Ping with a cheap stat; reconnect on failure.
	if d.sftpClient != nil {
		if _, err := d.sftpClient.Stat("."); err == nil {
			return d.sftpClient, nil
		}
	}
	if err := d.connect(); err != nil {
		return nil, err
	}
	return d.sftpClient, nil
}

func (d *sftpDest) Write(_ context.Context, key string, r io.Reader) (int64, error) {
	cl, err := d.client()
	if err != nil {
		return 0, err
	}
	remote := filepath.Join(d.cfg.RemotePath, key)
	if err := cl.MkdirAll(filepath.Dir(remote)); err != nil {
		return 0, fmt.Errorf("sftp mkdir: %w", err)
	}
	f, err := cl.Create(remote)
	if err != nil {
		return 0, fmt.Errorf("sftp create %s: %w", remote, err)
	}
	defer f.Close()
	return io.Copy(f, r)
}

func (d *sftpDest) List(_ context.Context, prefix string) ([]string, error) {
	cl, err := d.client()
	if err != nil {
		return nil, err
	}
	dir := d.cfg.RemotePath
	if prefix != "" {
		dir = filepath.Join(dir, prefix)
	}
	entries, err := cl.ReadDir(dir)
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

func (d *sftpDest) Delete(_ context.Context, key string) error {
	cl, err := d.client()
	if err != nil {
		return err
	}
	return removeAllSFTP(cl, filepath.Join(d.cfg.RemotePath, key))
}

func removeAllSFTP(cl *sftp.Client, path string) error {
	entries, err := cl.ReadDir(path)
	if err != nil {
		return cl.Remove(path)
	}
	for _, e := range entries {
		full := filepath.Join(path, e.Name())
		if e.IsDir() {
			if err := removeAllSFTP(cl, full); err != nil {
				return err
			}
		} else {
			if err := cl.Remove(full); err != nil {
				return err
			}
		}
	}
	return cl.RemoveDirectory(path)
}

func (d *sftpDest) String() string {
	return fmt.Sprintf("sftp://%s@%s%s", d.cfg.User, d.cfg.Host, d.cfg.RemotePath)
}
