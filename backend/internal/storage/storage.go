// Package storage is a swappable blob store. The local impl backs the demo;
// the s3 impl (config-gated) targets any S3-compatible RU-zone provider.
package storage

import (
	"context"
	"errors"
	"io"
)

// ErrNotFound is returned by Get/Exists when an object is absent.
var ErrNotFound = errors.New("storage: object not found")

// Storage is a content-addressed-by-key blob store.
type Storage interface {
	Put(ctx context.Context, key string, r io.Reader, size int64, contentType string) error
	Get(ctx context.Context, key string) (io.ReadCloser, error)
	Delete(ctx context.Context, key string) error
	// URL returns a publicly fetchable URL for key.
	URL(key string) string
	Exists(ctx context.Context, key string) (bool, error)
}
