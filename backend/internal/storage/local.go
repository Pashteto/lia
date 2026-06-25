package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type local struct {
	baseDir    string
	publicBase string // no trailing slash
}

// NewLocal returns a Storage that writes blobs under baseDir and serves them
// at publicBase/<key> (a backend route — see the files HTTP handler).
func NewLocal(baseDir, publicBase string) (Storage, error) {
	if baseDir == "" {
		return nil, fmt.Errorf("storage: baseDir is required")
	}
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		return nil, fmt.Errorf("storage: mkdir %q: %w", baseDir, err)
	}
	return &local{baseDir: baseDir, publicBase: strings.TrimRight(publicBase, "/")}, nil
}

// resolve guards against path traversal and returns the on-disk path for key.
func (l *local) resolve(key string) (string, error) {
	// Reject keys that contain ".." path components.
	for _, part := range strings.Split(filepath.ToSlash(key), "/") {
		if part == ".." {
			return "", fmt.Errorf("storage: invalid key %q", key)
		}
	}
	clean := filepath.Clean("/" + key) // force absolute, normalize slashes
	p := filepath.Join(l.baseDir, clean)
	if !strings.HasPrefix(p, filepath.Clean(l.baseDir)+string(os.PathSeparator)) {
		return "", fmt.Errorf("storage: invalid key %q", key)
	}
	return p, nil
}

func (l *local) Put(_ context.Context, key string, r io.Reader, _ int64, _ string) error {
	// size and contentType are unused by the local backend (the S3 backend uses them).
	p, err := l.resolve(key)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return fmt.Errorf("storage: mkdir: %w", err)
	}
	f, err := os.Create(p)
	if err != nil {
		return fmt.Errorf("storage: create: %w", err)
	}
	defer f.Close()
	if _, err := io.Copy(f, r); err != nil {
		return fmt.Errorf("storage: write: %w", err)
	}
	return nil
}

func (l *local) Get(_ context.Context, key string) (io.ReadCloser, error) {
	p, err := l.resolve(key)
	if err != nil {
		return nil, err
	}
	f, err := os.Open(p)
	if os.IsNotExist(err) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("storage: open: %w", err)
	}
	return f, nil
}

func (l *local) Delete(_ context.Context, key string) error {
	p, err := l.resolve(key)
	if err != nil {
		return err
	}
	if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("storage: remove: %w", err)
	}
	return nil
}

func (l *local) Exists(_ context.Context, key string) (bool, error) {
	p, err := l.resolve(key)
	if err != nil {
		return false, err
	}
	if _, err := os.Stat(p); err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (l *local) URL(key string) string {
	return l.publicBase + "/" + strings.TrimLeft(key, "/")
}
