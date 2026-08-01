// Package adminusers implements the staff-only user registry behind the A4
// admin screen (Пользователи и контент-гигиена). It reads the local users
// table joined with RSVP counts and organizer ownership; it performs no
// writes — A4 has no per-user actions by design.
package adminusers

import (
	"context"
	"strings"
	"time"

	"github.com/gofrs/uuid"
)

// Page size bounds for the registry listing.
const (
	DefaultLimit = 100
	MaxLimit     = 500
)

// Row is one registry line: the A4 table needs exactly these fields.
type Row struct {
	ID          uuid.UUID
	Name        string
	Email       string
	Role        string // users.role: "common" | "admin"
	CreatedAt   time.Time
	Bookings    int  // RSVPs in going|applied|accepted|waitlist
	IsOrganizer bool // has an organizers row as owner
}

// Filter is the listing query. Query matches name or email (case-insensitive,
// substring); Limit/Offset paginate over created_at DESC.
type Filter struct {
	Query  string
	Limit  int
	Offset int
}

// Repository reads registry rows.
type Repository interface {
	List(ctx context.Context, f Filter) ([]Row, error)
	// Count returns every active user, unpaginated — the «Пользователей» tile.
	Count(ctx context.Context) (int, error)
}

// Service is the registry use-case layer.
type Service interface {
	List(ctx context.Context, f Filter) ([]Row, error)
	Count(ctx context.Context) (int, error)
}

type service struct{ repo Repository }

// NewService returns a registry Service backed by repo.
func NewService(repo Repository) Service { return &service{repo: repo} }

func (s *service) List(ctx context.Context, f Filter) ([]Row, error) {
	f.Query = strings.TrimSpace(f.Query)
	if f.Limit <= 0 {
		f.Limit = DefaultLimit
	}
	if f.Limit > MaxLimit {
		f.Limit = MaxLimit
	}
	if f.Offset < 0 {
		f.Offset = 0
	}
	return s.repo.List(ctx, f)
}

func (s *service) Count(ctx context.Context) (int, error) {
	return s.repo.Count(ctx)
}
