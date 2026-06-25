package storage

import "fmt"

// StorageSettings is a transport-agnostic snapshot of storage config. The config
// package converts its StorageConfig into this to avoid a config→storage import cycle.
type StorageSettings struct {
	Backend    string
	LocalDir   string
	PublicBase string
	S3         S3Config
}

// New builds the configured Storage. Backend "" or "local" → local disk; "s3" → minio-go.
func New(cfg StorageSettings) (Storage, error) {
	switch cfg.Backend {
	case "", "local":
		return NewLocal(cfg.LocalDir, cfg.PublicBase)
	case "s3":
		s := cfg.S3
		if s.PublicBase == "" {
			s.PublicBase = cfg.PublicBase
		}
		return NewS3(s)
	default:
		return nil, fmt.Errorf("storage: unknown backend %q", cfg.Backend)
	}
}
