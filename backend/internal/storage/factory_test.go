package storage

import "testing"

func TestNew_LocalBackend(t *testing.T) {
	s, err := New(StorageSettings{Backend: "local", LocalDir: t.TempDir(), PublicBase: "https://x/f"})
	if err != nil || s == nil {
		t.Fatalf("New local: s=%v err=%v", s, err)
	}
}

func TestNew_UnknownBackend(t *testing.T) {
	if _, err := New(StorageSettings{Backend: "weird"}); err == nil {
		t.Fatal("expected error for unknown backend")
	}
}

func TestNew_DefaultsToLocal(t *testing.T) {
	if _, err := New(StorageSettings{Backend: "", LocalDir: t.TempDir(), PublicBase: "https://x/f"}); err != nil {
		t.Fatalf("empty backend should default to local: %v", err)
	}
}
