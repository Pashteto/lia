// Package cleanup provides a daily in-process orphan-file cleanup module.
package cleanup

import (
	"context"
	"time"

	"github.com/Pashteto/lia/pkg/logger"
)

// RunnerFunc is the minimal interface the Module requires from the file
// cleaner.  *files.Cleaner satisfies it.
type RunnerFunc interface {
	Run(ctx context.Context) (int, error)
}

// Module implements module.Module for periodic orphan-file cleanup.
type Module struct {
	cleaner  RunnerFunc
	interval time.Duration
	cancel   context.CancelFunc
	done     chan struct{}
}

// NewModule creates a cleanup module that calls cleaner.Run every interval.
// Pass a nil cleaner to create a disabled (no-op) module.
func NewModule(cleaner RunnerFunc, interval time.Duration) *Module {
	return &Module{
		cleaner:  cleaner,
		interval: interval,
	}
}

// Name returns the module identifier.
func (m *Module) Name() string { return "cleanup" }

// Init is a no-op for this module (nothing to prepare).
func (m *Module) Init(_ context.Context) error { return nil }

// Start launches the background cleanup ticker.  It runs an initial cleanup
// immediately, then repeats every m.interval.  Start is non-blocking.
func (m *Module) Start(_ context.Context) error {
	if m.cleaner == nil {
		logger.Log().Info("cleanup module: no cleaner configured, skipping start")
		return nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	m.cancel = cancel
	m.done = make(chan struct{})

	go func() {
		defer close(m.done)

		// Run once immediately on start.
		m.runOnce(ctx)

		ticker := time.NewTicker(m.interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				m.runOnce(ctx)
			}
		}
	}()

	logger.Log().Infof("cleanup module started (interval=%s)", m.interval)
	return nil
}

// Stop cancels the background goroutine and waits for it to exit.
func (m *Module) Stop(_ context.Context) error {
	if m.cancel == nil {
		return nil // Start was never called or cleaner was nil
	}
	m.cancel()
	if m.done != nil {
		<-m.done
	}
	logger.Log().Info("cleanup module stopped")
	return nil
}

// HealthCheck always returns nil; the cleanup job is best-effort.
func (m *Module) HealthCheck(_ context.Context) error { return nil }

func (m *Module) runOnce(ctx context.Context) {
	deleted, err := m.cleaner.Run(ctx)
	if err != nil {
		logger.Log().Errorf("cleanup module: run error: %v", err)
		return
	}
	logger.Log().Infof("cleanup module: deleted=%d orphan file(s)", deleted)
}
