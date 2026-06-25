package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// S3Config configures an S3-compatible backend. Works against AWS S3, MinIO,
// and RU-zone providers (Yandex Object Storage, Selectel, VK Cloud, Cloud.ru):
// set Endpoint to the provider host and Region to "us-east-1".
type S3Config struct {
	Endpoint   string
	Region     string
	Bucket     string
	AccessKey  string
	SecretKey  string
	PublicBase string // optional; if empty, URL() derives endpoint/bucket/key
	UseSSL     bool
}

type s3store struct {
	client     *minio.Client
	bucket     string
	publicBase string
}

// NewS3 dials the S3-compatible endpoint and verifies the bucket exists.
func NewS3(cfg S3Config) (Storage, error) {
	if cfg.Endpoint == "" || cfg.Bucket == "" {
		return nil, fmt.Errorf("storage(s3): endpoint and bucket are required")
	}
	cli, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
		Region: cfg.Region,
	})
	if err != nil {
		return nil, fmt.Errorf("storage(s3): new client: %w", err)
	}
	ok, err := cli.BucketExists(context.Background(), cfg.Bucket)
	if err != nil {
		return nil, fmt.Errorf("storage(s3): bucket check: %w", err)
	}
	if !ok {
		return nil, fmt.Errorf("storage(s3): bucket %q does not exist", cfg.Bucket)
	}
	base := strings.TrimRight(cfg.PublicBase, "/")
	if base == "" {
		scheme := "https"
		if !cfg.UseSSL {
			scheme = "http"
		}
		base = fmt.Sprintf("%s://%s/%s", scheme, cfg.Endpoint, cfg.Bucket)
	}
	return &s3store{client: cli, bucket: cfg.Bucket, publicBase: base}, nil
}

func (s *s3store) Put(ctx context.Context, key string, r io.Reader, size int64, contentType string) error {
	_, err := s.client.PutObject(ctx, s.bucket, key, r, size, minio.PutObjectOptions{ContentType: contentType})
	if err != nil {
		return fmt.Errorf("storage(s3): put %q: %w", key, err)
	}
	return nil
}

func (s *s3store) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	obj, err := s.client.GetObject(ctx, s.bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("storage(s3): get %q: %w", key, err)
	}
	if _, err := obj.Stat(); err != nil {
		obj.Close()
		var e minio.ErrorResponse
		if errors.As(err, &e) && e.Code == "NoSuchKey" {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("storage(s3): stat %q: %w", key, err)
	}
	return obj, nil
}

func (s *s3store) Delete(ctx context.Context, key string) error {
	if err := s.client.RemoveObject(ctx, s.bucket, key, minio.RemoveObjectOptions{}); err != nil {
		return fmt.Errorf("storage(s3): delete %q: %w", key, err)
	}
	return nil
}

func (s *s3store) Exists(ctx context.Context, key string) (bool, error) {
	_, err := s.client.StatObject(ctx, s.bucket, key, minio.StatObjectOptions{})
	if err != nil {
		var e minio.ErrorResponse
		if errors.As(err, &e) && e.Code == "NoSuchKey" {
			return false, nil
		}
		return false, fmt.Errorf("storage(s3): stat %q: %w", key, err)
	}
	return true, nil
}

func (s *s3store) URL(key string) string {
	return s.publicBase + "/" + strings.TrimLeft(key, "/")
}
