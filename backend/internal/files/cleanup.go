package files

import (
	"context"
	"time"

	"github.com/Pashteto/lia/internal/storage"
	"github.com/Pashteto/lia/pkg/logger"
)

// Cleaner deletes orphaned file blobs and their metadata rows.
// A file is an orphan when it is not referenced by any event cover or user
// avatar and is older than the configured grace period.
type Cleaner struct {
	repo  Repository
	store storage.Storage
	grace time.Duration
}

// NewCleaner creates a Cleaner that deletes orphans older than grace.
func NewCleaner(repo Repository, store storage.Storage, grace time.Duration) *Cleaner {
	return &Cleaner{repo: repo, store: store, grace: grace}
}

// Run lists orphan files, deletes each blob then its row, and returns the
// number of files successfully deleted.  A per-file storage error is logged
// and skipped (not fatal); Run returns (deleted, nil) in that case.
// The result is idempotent: calling Run again on an already-clean set is a
// no-op.
func (c *Cleaner) Run(ctx context.Context) (int, error) {
	candidates, err := c.repo.ListOrphansOlderThan(c.grace)
	if err != nil {
		return 0, err
	}
	logger.Log().Infof("cleanup: found %d orphan file(s) older than %s", len(candidates), c.grace)

	deleted := 0
	for _, f := range candidates {
		if err := c.store.Delete(ctx, f.StorageKey); err != nil {
			logger.Log().Errorf("cleanup: delete blob %q for file %s: %v (skipped)", f.StorageKey, f.ID, err)
			continue
		}
		if err := c.repo.Delete(f.ID); err != nil {
			logger.Log().Errorf("cleanup: delete file row %s: %v", f.ID, err)
			// Blob is already gone; still count as partial failure but continue.
			continue
		}
		deleted++
	}

	logger.Log().Infof("cleanup: deleted %d orphan file(s)", deleted)
	return deleted, nil
}
