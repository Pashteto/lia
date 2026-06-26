//go:build integration

package organizers

import (
	"context"
	"os"
	"testing"

	"github.com/go-pg/pg/v10"
	"github.com/gofrs/uuid"
)

// requires a migrated test DB; DSN from TEST_DATABASE_URL. Mirrors the
// moderation integration tests (still not wired into local CI — see roadmap).
func testDB(t *testing.T) *pg.DB {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	opt, err := pg.ParseURL(dsn)
	if err != nil {
		t.Fatal(err)
	}
	return pg.Connect(opt)
}

func TestSubmitThenVerifyLifecycle(t *testing.T) {
	db := testDB(t)
	defer db.Close()
	repo := NewRepository(db)
	ctx := context.Background()
	owner := uuid.Must(uuid.NewV4())

	org, err := repo.Upsert(ctx, owner, Input{Name: "Acme"})
	if err != nil {
		t.Fatal(err)
	}
	if org.VerificationStatus != "draft" {
		t.Fatalf("status = %q; want draft", org.VerificationStatus)
	}
	status, err := repo.Submit(ctx, org.ID, owner, false)
	if err != nil || status != "pending" {
		t.Fatalf("submit = %q, %v; want pending", status, err)
	}
	if err := repo.Verify(ctx, org.ID, owner); err != nil {
		t.Fatal(err)
	}
	// Verifying a non-pending org now fails.
	if err := repo.Verify(ctx, org.ID, owner); err != ErrInvalidTransition {
		t.Fatalf("re-verify err = %v; want ErrInvalidTransition", err)
	}
}

func TestSubmitAutoVerifyShortCircuits(t *testing.T) {
	db := testDB(t)
	defer db.Close()
	repo := NewRepository(db)
	ctx := context.Background()
	owner := uuid.Must(uuid.NewV4())
	org, err := repo.Upsert(ctx, owner, Input{Name: "Trusted"})
	if err != nil {
		t.Fatal(err)
	}
	status, err := repo.Submit(ctx, org.ID, owner, true)
	if err != nil || status != "verified" {
		t.Fatalf("submit auto = %q, %v; want verified", status, err)
	}
}

// TestUpsertPreservesLogoOnOmit proves that re-upserting with a zero LogoFileID
// (i.e. the client omitted the logo on an edit-profile save) does NOT wipe the
// previously stored logo_file_id. Mirrors the preserve-on-omit convention used
// for venue coordinates in the events package.
func TestUpsertPreservesLogoOnOmit(t *testing.T) {
	db := testDB(t)
	defer db.Close()
	repo := NewRepository(db)
	ctx := context.Background()
	owner := uuid.Must(uuid.NewV4())
	originalLogo := uuid.Must(uuid.NewV4())

	// First upsert: create the organizer with a real logo file id.
	org, err := repo.Upsert(ctx, owner, Input{Name: "Logo Test Org", LogoFileID: originalLogo})
	if err != nil {
		t.Fatal(err)
	}
	if org.LogoFileID != originalLogo {
		t.Fatalf("initial LogoFileID = %v; want %v", org.LogoFileID, originalLogo)
	}

	// Second upsert: edit the name but omit the logo (zero uuid = unchanged).
	updated, err := repo.Upsert(ctx, owner, Input{Name: "Logo Test Org Renamed", LogoFileID: uuid.Nil})
	if err != nil {
		t.Fatal(err)
	}
	if updated.LogoFileID != originalLogo {
		t.Fatalf("after omit-logo upsert: LogoFileID = %v; want original %v (logo was wiped)", updated.LogoFileID, originalLogo)
	}
}

// TestListMapsWebsiteURLAndVerifiedAt proves that List correctly maps the
// website_url / logo_file_id / owner_user_id columns now that the Organizer
// struct carries explicit pg tags. A draft org should have WebsiteURL populated
// and VerifiedAt == nil (never verified).
func TestListMapsWebsiteURLAndVerifiedAt(t *testing.T) {
	db := testDB(t)
	defer db.Close()
	repo := NewRepository(db)
	ctx := context.Background()
	owner := uuid.Must(uuid.NewV4())

	_, err := repo.Upsert(ctx, owner, Input{Name: "Acme List", WebsiteURL: "https://acme.test"})
	if err != nil {
		t.Fatal(err)
	}

	orgs, err := repo.List(ctx, ListFilter{Status: "draft"})
	if err != nil {
		t.Fatal(err)
	}

	var found *Organizer
	for i := range orgs {
		if orgs[i].OwnerUserID == owner {
			found = &orgs[i]
			break
		}
	}
	if found == nil {
		t.Fatal("org not found in List result")
	}
	if found.WebsiteURL != "https://acme.test" {
		t.Errorf("WebsiteURL = %q; want %q", found.WebsiteURL, "https://acme.test")
	}
	if found.VerifiedAt != nil {
		t.Errorf("VerifiedAt = %v; want nil (draft, never verified)", found.VerifiedAt)
	}
}
