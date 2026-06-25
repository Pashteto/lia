package cleanup_test

import (
	"testing"

	"github.com/Pashteto/lia/cmd/cleanup"
)

func TestCleanupCmd_WiringAndRunE(t *testing.T) {
	cmd := cleanup.Cmd()
	if cmd.Use != "files:cleanup" {
		t.Fatalf("want Use=%q, got %q", "files:cleanup", cmd.Use)
	}
	if cmd.RunE == nil {
		t.Fatal("RunE must not be nil")
	}
}
