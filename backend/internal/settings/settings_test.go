package settings

import (
	"context"
	"testing"

	"github.com/gofrs/uuid"
)

type fakeRepo struct {
	store    map[string]bool
	setCalls int
	lastKey  string
	lastVal  bool
	lastBy   uuid.UUID
}

func (f *fakeRepo) GetBool(_ context.Context, key string) (bool, error) { return f.store[key], nil }
func (f *fakeRepo) All(_ context.Context) (map[string]bool, error)      { return f.store, nil }
func (f *fakeRepo) SetBool(_ context.Context, key string, by uuid.UUID, v bool) error {
	f.setCalls++
	f.lastKey, f.lastVal, f.lastBy = key, v, by
	f.store[key] = v
	return nil
}

func TestServiceBoolReadsRepo(t *testing.T) {
	r := &fakeRepo{store: map[string]bool{KeyAutoVerifyAll: true}}
	got, err := NewService(r).Bool(context.Background(), KeyAutoVerifyAll)
	if err != nil || !got {
		t.Fatalf("Bool = %v, %v; want true, nil", got, err)
	}
}

func TestServiceSetBoolDelegates(t *testing.T) {
	r := &fakeRepo{store: map[string]bool{}}
	actor := uuid.Must(uuid.NewV4())
	if err := NewService(r).SetBool(context.Background(), KeyAutoVerifyAll, actor, true); err != nil {
		t.Fatal(err)
	}
	if r.setCalls != 1 || r.lastKey != KeyAutoVerifyAll || !r.lastVal || r.lastBy != actor {
		t.Fatalf("unexpected SetBool args: calls=%d key=%s val=%v by=%v", r.setCalls, r.lastKey, r.lastVal, r.lastBy)
	}
}
