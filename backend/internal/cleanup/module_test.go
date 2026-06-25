package cleanup_test

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Pashteto/lia/internal/cleanup"
)

// fakeCleaner counts how many times Run is called.
type fakeCleaner struct {
	calls atomic.Int64
}

func (f *fakeCleaner) Run(_ context.Context) (int, error) {
	f.calls.Add(1)
	return 0, nil
}

func TestModule_StartRunsAtLeastOnce(t *testing.T) {
	fc := &fakeCleaner{}
	mod := cleanup.NewModule(fc, 20*time.Millisecond)

	ctx := context.Background()
	if err := mod.Init(ctx); err != nil {
		t.Fatalf("Init: %v", err)
	}
	if err := mod.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}

	// Poll until at least one call or timeout.
	deadline := time.Now().Add(200 * time.Millisecond)
	for time.Now().Before(deadline) {
		if fc.calls.Load() >= 1 {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if fc.calls.Load() < 1 {
		t.Fatalf("want cleaner called ≥1 within 200ms, got %d", fc.calls.Load())
	}

	stopCtx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := mod.Stop(stopCtx); err != nil {
		t.Fatalf("Stop: %v", err)
	}

	// Record the call count after Stop and wait a bit; no new calls should arrive.
	afterStop := fc.calls.Load()
	time.Sleep(60 * time.Millisecond)
	if fc.calls.Load() > afterStop {
		t.Fatalf("cleaner called after Stop: before=%d after=%d", afterStop, fc.calls.Load())
	}
}

func TestModule_Name(t *testing.T) {
	mod := cleanup.NewModule(nil, time.Hour)
	if mod.Name() != "cleanup" {
		t.Fatalf("want Name()=\"cleanup\", got %q", mod.Name())
	}
}

func TestModule_StopBeforeStart(t *testing.T) {
	mod := cleanup.NewModule(nil, time.Hour)
	// Stop must be safe even if Start was never called.
	if err := mod.Stop(context.Background()); err != nil {
		t.Fatalf("Stop before Start: %v", err)
	}
}
