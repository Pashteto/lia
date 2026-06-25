//go:build s3smoke

package storage

import (
	"bytes"
	"context"
	"io"
	"os"
	"testing"
)

// Run with a local MinIO:
//
//	docker run -d -p 9000:9000 -e MINIO_ROOT_USER=minioadmin -e MINIO_ROOT_PASSWORD=minioadmin minio/minio server /data
//	docker run --rm --network host minio/mc alias set m http://127.0.0.1:9000 minioadmin minioadmin && mc mb m/testbucket
//	S3_ENDPOINT=127.0.0.1:9000 go test -tags s3smoke ./internal/storage/ -run TestS3Smoke -v
func TestS3Smoke(t *testing.T) {
	ep := os.Getenv("S3_ENDPOINT")
	if ep == "" {
		t.Skip("S3_ENDPOINT not set")
	}
	s, err := NewS3(S3Config{
		Endpoint: ep, Region: "us-east-1", Bucket: "testbucket",
		AccessKey: "minioadmin", SecretKey: "minioadmin", UseSSL: false,
	})
	if err != nil {
		t.Fatalf("NewS3: %v", err)
	}
	ctx := context.Background()
	body := []byte("s3-smoke")
	if err := s.Put(ctx, "uploads/smoke.png", bytes.NewReader(body), int64(len(body)), "image/png"); err != nil {
		t.Fatalf("Put: %v", err)
	}
	rc, err := s.Get(ctx, "uploads/smoke.png")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	got, _ := io.ReadAll(rc)
	rc.Close()
	if !bytes.Equal(got, body) {
		t.Fatalf("body mismatch")
	}
	if err := s.Delete(ctx, "uploads/smoke.png"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if ok, _ := s.Exists(ctx, "uploads/smoke.png"); ok {
		t.Fatal("exists after delete")
	}
}
