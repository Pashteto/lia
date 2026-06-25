package password_test

import (
	"testing"

	"gateguard/internal/pkg/password"
)

func TestHashAndCompare(t *testing.T) {
	const pw = "correct horse battery staple"

	h, err := password.Hash(pw)
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	if h == pw {
		t.Fatal("hash must not equal plaintext")
	}
	if err := password.Compare(h, pw); err != nil {
		t.Fatalf("expected match, got %v", err)
	}
	if err := password.Compare(h, "wrong"); err == nil {
		t.Fatal("expected mismatch error for wrong password")
	}
}

func TestHashIsSalted(t *testing.T) {
	a, _ := password.Hash("same")
	b, _ := password.Hash("same")
	if a == b {
		t.Fatal("two hashes of the same password must differ (salt)")
	}
}
