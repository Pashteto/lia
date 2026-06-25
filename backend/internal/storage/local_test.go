package storage

import (
	"bytes"
	"context"
	"io"
	"testing"
)

func TestLocal_PutGetExistsDelete(t *testing.T) {
	dir := t.TempDir()
	s, err := NewLocal(dir, "https://x/api/v1/files")
	if err != nil {
		t.Fatalf("NewLocal: %v", err)
	}
	ctx := context.Background()
	body := []byte("hello-image-bytes")
	if err := s.Put(ctx, "uploads/a.png", bytes.NewReader(body), int64(len(body)), "image/png"); err != nil {
		t.Fatalf("Put: %v", err)
	}
	ok, err := s.Exists(ctx, "uploads/a.png")
	if err != nil || !ok {
		t.Fatalf("Exists: ok=%v err=%v", ok, err)
	}
	rc, err := s.Get(ctx, "uploads/a.png")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	got, _ := io.ReadAll(rc)
	rc.Close()
	if !bytes.Equal(got, body) {
		t.Fatalf("Get body mismatch: %q", got)
	}
	if s.URL("uploads/a.png") != "https://x/api/v1/files/uploads/a.png" {
		t.Fatalf("URL: %q", s.URL("uploads/a.png"))
	}
	if err := s.Delete(ctx, "uploads/a.png"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if ok, _ := s.Exists(ctx, "uploads/a.png"); ok {
		t.Fatalf("Exists after delete = true")
	}
}

func TestLocal_GetMissing_ReturnsErrNotFound(t *testing.T) {
	s, _ := NewLocal(t.TempDir(), "https://x/f")
	if _, err := s.Get(context.Background(), "nope.png"); err == nil {
		t.Fatal("expected error for missing object")
	}
}

func TestLocal_RejectsPathTraversal(t *testing.T) {
	s, _ := NewLocal(t.TempDir(), "https://x/f")
	if err := s.Put(context.Background(), "../escape.png", nil, 0, "image/png"); err == nil {
		t.Fatal("expected path-traversal rejection")
	}
}
