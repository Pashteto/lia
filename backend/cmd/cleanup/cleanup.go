// Package cleanup defines the "files:cleanup" CLI subcommand.
package cleanup

import (
	"context"
	"fmt"
	"time"

	"github.com/go-pg/pg/v10"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/Pashteto/lia/config"
	"github.com/Pashteto/lia/internal/files"
	"github.com/Pashteto/lia/internal/storage"
	"github.com/Pashteto/lia/pkg/logger"
)

// Cmd returns the "files:cleanup" subcommand.
// It bootstraps the database and storage independently (no HTTP/gRPC), runs
// the orphan-file cleaner once, logs the deleted count, and exits.
func Cmd() *cobra.Command {
	return &cobra.Command{
		Use:   "files:cleanup",
		Short: "Delete orphaned file blobs and metadata rows (one-shot)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			// Config defaults + env vars are already registered by config.init()
			// and bound by root.Cmd's PersistentPreRunE. Unmarshal the final state.
			cfg := &config.Scheme{}
			if err := viper.Unmarshal(cfg); err != nil {
				return fmt.Errorf("load config: %w", err)
			}
			return run(cmd.Context(), cfg)
		},
	}
}

// run is the implementation: it builds the DB + storage + cleaner and runs once.
func run(ctx context.Context, cfg *config.Scheme) error {
	// Database is required.
	if cfg.Database == nil || !cfg.Database.Enabled {
		return fmt.Errorf("database is not enabled; set DATABASE_ENABLED=true")
	}

	db := pg.Connect(&pg.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Database.Host, cfg.Database.Port),
		User:     cfg.Database.User,
		Password: cfg.Database.Password,
		Database: cfg.Database.Name,
	})
	defer func() { _ = db.Close() }()

	if _, err := db.Exec("SELECT 1"); err != nil {
		return fmt.Errorf("database ping: %w", err)
	}

	// Storage is required for blob deletion.
	if cfg.Storage == nil {
		return fmt.Errorf("storage is not configured; set STORAGE_BACKEND")
	}
	bs, err := storage.New(toStorageSettings(cfg.Storage))
	if err != nil {
		return fmt.Errorf("init storage: %w", err)
	}

	grace := parseDurationDefault(cfg.Cleanup, 24*time.Hour)

	cleaner := files.NewCleaner(files.NewRepository(db), bs, grace)
	deleted, err := cleaner.Run(ctx)
	if err != nil {
		return fmt.Errorf("cleanup run: %w", err)
	}

	logger.Log().Infof("files:cleanup done deleted=%d", deleted)
	return nil
}

// parseDurationDefault parses CleanupConfig.Grace; falls back to fallback on
// parse failure or nil config.
func parseDurationDefault(cfg *config.CleanupConfig, fallback time.Duration) time.Duration {
	if cfg == nil {
		return fallback
	}
	if d, err := time.ParseDuration(cfg.Grace); err == nil {
		return d
	}
	return fallback
}

// toStorageSettings mirrors the helper in internal/application.go.
func toStorageSettings(cfg *config.StorageConfig) storage.StorageSettings {
	if cfg == nil {
		return storage.StorageSettings{}
	}
	ss := storage.StorageSettings{
		Backend:    cfg.Backend,
		LocalDir:   cfg.LocalDir,
		PublicBase: cfg.PublicBase,
	}
	if cfg.S3 != nil {
		ss.S3 = storage.S3Config{
			Endpoint:  cfg.S3.Endpoint,
			Region:    cfg.S3.Region,
			Bucket:    cfg.S3.Bucket,
			AccessKey: cfg.S3.AccessKey,
			SecretKey: cfg.S3.SecretKey,
			UseSSL:    cfg.S3.UseSSL,
		}
	}
	return ss
}
