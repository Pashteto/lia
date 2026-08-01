package adminusers

import (
	"context"
	"testing"
)

type recordingRepo struct {
	got  Filter
	rows []Row
	err  error
}

func (r *recordingRepo) List(_ context.Context, f Filter) ([]Row, error) {
	r.got = f
	return r.rows, r.err
}

func (r *recordingRepo) Count(context.Context) (int, error) { return len(r.rows), r.err }

func TestList_DefaultsLimitAndTrimsQuery(t *testing.T) {
	repo := &recordingRepo{}
	if _, err := NewService(repo).List(context.Background(), Filter{Query: "  анна  "}); err != nil {
		t.Fatalf("List: %v", err)
	}
	if repo.got.Limit != DefaultLimit {
		t.Fatalf("Limit = %d, want %d", repo.got.Limit, DefaultLimit)
	}
	if repo.got.Query != "анна" {
		t.Fatalf("Query = %q, want %q", repo.got.Query, "анна")
	}
}

func TestList_ClampsLimitAndFloorsOffset(t *testing.T) {
	repo := &recordingRepo{}
	if _, err := NewService(repo).List(context.Background(), Filter{Limit: 5000, Offset: -3}); err != nil {
		t.Fatalf("List: %v", err)
	}
	if repo.got.Limit != MaxLimit {
		t.Fatalf("Limit = %d, want %d", repo.got.Limit, MaxLimit)
	}
	if repo.got.Offset != 0 {
		t.Fatalf("Offset = %d, want 0", repo.got.Offset)
	}
}

func TestList_PassesThroughRows(t *testing.T) {
	repo := &recordingRepo{rows: []Row{{Name: "Анна", Bookings: 14}}}
	rows, err := NewService(repo).List(context.Background(), Filter{Limit: 10, Offset: 20})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(rows) != 1 || rows[0].Bookings != 14 {
		t.Fatalf("rows = %+v, want one row with 14 bookings", rows)
	}
	if repo.got.Limit != 10 || repo.got.Offset != 20 {
		t.Fatalf("filter = %+v, want limit 10 offset 20", repo.got)
	}
}
