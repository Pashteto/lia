package platforms

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
)

type Service interface {
	Check(ctx context.Context, rawURL string) (displayName string, trusted bool, err error)
	ListActive(ctx context.Context) ([]*TrustedPlatform, error)
	List(ctx context.Context) ([]*TrustedPlatform, error)
	Add(ctx context.Context, domainSuffix, displayName, category string) (*TrustedPlatform, error)
	Deactivate(ctx context.Context, id uuid.UUID) error
}

var validCategories = map[string]bool{"ticketing": true, "afisha": true, "gov": true, "social": true}

type service struct{ repo Repository }

func NewService(repo Repository) Service { return &service{repo: repo} }

func (s *service) Check(ctx context.Context, rawURL string) (string, bool, error) {
	host, err := NormalizeHost(rawURL)
	if err != nil {
		return "", false, err
	}
	rows, err := s.repo.ListActive(ctx)
	if err != nil {
		return "", false, fmt.Errorf("check platform: %w", err)
	}
	for _, row := range rows {
		if MatchesSuffix(host, row.DomainSuffix) {
			return row.DisplayName, true, nil
		}
	}
	return "", false, nil
}

func (s *service) ListActive(ctx context.Context) ([]*TrustedPlatform, error) {
	return s.repo.ListActive(ctx)
}
func (s *service) List(ctx context.Context) ([]*TrustedPlatform, error) { return s.repo.List(ctx) }

func (s *service) Add(ctx context.Context, domainSuffix, displayName, category string) (*TrustedPlatform, error) {
	if domainSuffix == "" || displayName == "" {
		return nil, fmt.Errorf("domain_suffix and display_name are required")
	}
	if !validCategories[category] {
		return nil, fmt.Errorf("invalid category %q", category)
	}
	// Normalize the suffix the same way hosts are normalized (punycode,
	// lowercase) so matching stays consistent.
	ascii, err := NormalizeHost("https://" + domainSuffix)
	if err != nil {
		return nil, fmt.Errorf("invalid domain_suffix %q", domainSuffix)
	}
	p := &TrustedPlatform{DomainSuffix: ascii, DisplayName: displayName, Category: category, IsActive: true}
	if err := s.repo.Create(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}

func (s *service) Deactivate(ctx context.Context, id uuid.UUID) error {
	return s.repo.SetActive(ctx, id, false)
}
