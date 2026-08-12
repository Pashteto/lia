package platforms

import (
	"context"
	"errors"
	"testing"

	"github.com/gofrs/uuid"
)

type fakeRepo struct{ rows []*TrustedPlatform }

func (f *fakeRepo) ListActive(context.Context) ([]*TrustedPlatform, error) {
	out := []*TrustedPlatform{}
	for _, r := range f.rows {
		if r.IsActive {
			out = append(out, r)
		}
	}
	return out, nil
}
func (f *fakeRepo) List(context.Context) ([]*TrustedPlatform, error) { return f.rows, nil }
func (f *fakeRepo) Create(_ context.Context, p *TrustedPlatform) error {
	f.rows = append(f.rows, p)
	return nil
}
func (f *fakeRepo) SetActive(_ context.Context, id uuid.UUID, active bool) error {
	for _, r := range f.rows {
		if r.ID == id {
			r.IsActive = active
			return nil
		}
	}
	return ErrNotFound
}

func newTestService() Service {
	return NewService(&fakeRepo{rows: []*TrustedPlatform{
		{DomainSuffix: "timepad.ru", DisplayName: "TimePad", Category: "ticketing", IsActive: true},
		{DomainSuffix: "xn--80atdujec4e.xn--p1ai", DisplayName: "Культура.РФ", Category: "gov", IsActive: true},
		{DomainSuffix: "dead.ru", DisplayName: "Dead", Category: "ticketing", IsActive: false},
	}})
}

func TestCheck(t *testing.T) {
	s := newTestService()
	ctx := context.Background()

	name, trusted, err := s.Check(ctx, "https://org.timepad.ru/event/123")
	if err != nil || !trusted || name != "TimePad" {
		t.Fatalf("subdomain match: name=%q trusted=%v err=%v", name, trusted, err)
	}

	name, trusted, err = s.Check(ctx, "https://культура.рф/afisha")
	if err != nil || !trusted || name != "Культура.РФ" {
		t.Fatalf("punycode match: name=%q trusted=%v err=%v", name, trusted, err)
	}

	_, trusted, err = s.Check(ctx, "https://unknown-site.ru/e")
	if err != nil || trusted {
		t.Fatalf("unknown domain must be trusted=false, err=nil; got trusted=%v err=%v", trusted, err)
	}

	_, trusted, err = s.Check(ctx, "https://x.dead.ru/e")
	if err != nil || trusted {
		t.Fatalf("inactive row must not match; got trusted=%v err=%v", trusted, err)
	}

	if _, _, err = s.Check(ctx, "http://timepad.ru/e"); !errors.Is(err, ErrBadURL) {
		t.Fatalf("http must be ErrBadURL, got %v", err)
	}
}

func TestAddValidation(t *testing.T) {
	s := newTestService()
	if _, err := s.Add(context.Background(), "", "X", "ticketing"); err == nil || !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("empty suffix must fail with ErrInvalidInput, got %v", err)
	}
	if _, err := s.Add(context.Background(), "new.ru", "New", "bogus"); err == nil || !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("bogus category must fail with ErrInvalidInput, got %v", err)
	}
	p, err := s.Add(context.Background(), "Новый.РФ", "Новый", "afisha")
	if err != nil {
		t.Fatalf("add: %v", err)
	}
	if p.DomainSuffix != "xn--b1aoke0e.xn--p1ai" {
		t.Fatalf("suffix must be stored in punycode, got %q", p.DomainSuffix)
	}
}

func TestDeactivate_ErrNotFound(t *testing.T) {
	s := newTestService()
	if err := s.Deactivate(context.Background(), uuid.Must(uuid.NewV4())); !errors.Is(err, ErrNotFound) {
		t.Fatalf("unknown id must fail with ErrNotFound, got %v", err)
	}
}
