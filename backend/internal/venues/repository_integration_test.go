//go:build integration

package venues

import (
	"os"
	"testing"

	"github.com/go-pg/pg/v10"
	"github.com/gofrs/uuid"
)

// Run with a migrated Postgres (needs migration 000022 for unaccent/pg_trgm):
//
//	TEST_DATABASE_URL=postgres://lia:lia@localhost:5432/lia_test?sslmode=disable \
//	  go test -tags=integration ./internal/venues/ -v
func openTestDB(t *testing.T) *pg.DB {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set — skipping integration test")
	}
	opts, err := pg.ParseURL(dsn)
	if err != nil {
		t.Fatalf("parse TEST_DATABASE_URL: %v", err)
	}
	db := pg.Connect(opts)
	if _, err := db.Exec("SELECT 1"); err != nil {
		db.Close()
		t.Fatalf("connect to test DB: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

// seedSearchFixtures inserts the venues these tests search over and removes
// them again afterwards, so the suite is re-runnable against a shared DB.
func seedSearchFixtures(t *testing.T, db *pg.DB) {
	t.Helper()
	rows := []struct{ name, address, metro string }{
		{"Noôdome", "Романов переулок, 2, строение 1", "Библиотека имени Ленина"},
		{"Голицын Лофт", "набережная реки Фонтанки, 20", "Гостиный двор"},
		{"Музей Фаберже", "набережная реки Фонтанки, 21", "Гостиный двор"},
		{"Клуб «Космонавт»", "Бронницкая улица, 24", "Технологический институт"},
		{"Скидка 50% на всё", "Тестовая улица, 1", "Тестовая"},
	}
	ids := make([]uuid.UUID, 0, len(rows))
	for _, r := range rows {
		id := uuid.Must(uuid.NewV4())
		ids = append(ids, id)
		if _, err := db.Exec(
			`INSERT INTO venues (id, name, address, metro) VALUES (?, ?, ?, ?)`,
			id, r.name, r.address, r.metro,
		); err != nil {
			t.Fatalf("seed venue %q: %v", r.name, err)
		}
	}
	t.Cleanup(func() {
		_, _ = db.Exec(`DELETE FROM venues WHERE id IN (?)`, pg.In(ids))
	})
}

func names(t *testing.T, repo Repository, q string) []string {
	t.Helper()
	list, err := repo.Search(q, 20)
	if err != nil {
		t.Fatalf("Search(%q): %v", q, err)
	}
	out := make([]string, 0, len(list))
	for _, v := range list {
		out = append(out, v.Name)
	}
	return out
}

func contains(got []string, want string) bool {
	for _, g := range got {
		if g == want {
			return true
		}
	}
	return false
}

// The QA report: searching the street found none of the venues on it, because
// Search only ever looked at the name.
func TestSearch_MatchesAddress(t *testing.T) {
	db := openTestDB(t)
	seedSearchFixtures(t, db)
	got := names(t, NewRepository(db), "Фонтанки")
	if !contains(got, "Голицын Лофт") || !contains(got, "Музей Фаберже") {
		t.Fatalf("Search(\"Фонтанки\") = %v, want both Fontanka venues", got)
	}
}

func TestSearch_MatchesMetro(t *testing.T) {
	db := openTestDB(t)
	seedSearchFixtures(t, db)
	got := names(t, NewRepository(db), "Технологический")
	if !contains(got, "Клуб «Космонавт»") {
		t.Fatalf("Search(\"Технологический\") = %v, want Космонавт", got)
	}
}

// "Noodome" is what a user types; the venue is spelled «Noôdome».
func TestSearch_FoldsDiacritics(t *testing.T) {
	db := openTestDB(t)
	seedSearchFixtures(t, db)
	got := names(t, NewRepository(db), "Noodome")
	if !contains(got, "Noôdome") {
		t.Fatalf("Search(\"Noodome\") = %v, want Noôdome", got)
	}
}

// The literal string from the QA report — a near-miss that substring matching
// cannot reach, so it has to come from the trigram branch.
func TestSearch_FuzzyMatchesNearMiss(t *testing.T) {
	db := openTestDB(t)
	seedSearchFixtures(t, db)
	got := names(t, NewRepository(db), "NoDom")
	if !contains(got, "Noôdome") {
		t.Fatalf("Search(\"NoDom\") = %v, want Noôdome", got)
	}
}

// A name hit is a stronger signal than an address hit and must sort first.
func TestSearch_RanksNameMatchesAboveAddressMatches(t *testing.T) {
	db := openTestDB(t)
	seedSearchFixtures(t, db)
	got := names(t, NewRepository(db), "Фаберже")
	if len(got) == 0 || got[0] != "Музей Фаберже" {
		t.Fatalf("Search(\"Фаберже\") = %v, want Музей Фаберже first", got)
	}
}

// "%" must search for the character, not match every row.
func TestSearch_TreatsWildcardsAsLiterals(t *testing.T) {
	db := openTestDB(t)
	seedSearchFixtures(t, db)
	got := names(t, NewRepository(db), "50%")
	if !contains(got, "Скидка 50% на всё") {
		t.Fatalf("Search(\"50%%\") = %v, want the literal-percent venue", got)
	}
	if contains(got, "Музей Фаберже") {
		t.Fatalf("Search(\"50%%\") = %v, wildcard leaked and matched everything", got)
	}
}

func TestSearch_EmptyQueryStillListsVenues(t *testing.T) {
	db := openTestDB(t)
	seedSearchFixtures(t, db)
	if got := names(t, NewRepository(db), "   "); len(got) == 0 {
		t.Fatal("Search(\"   \") returned nothing, want the first page of venues")
	}
}

// Latin↔Cyrillic: neither unaccent nor trigrams cross alphabets, so these can
// only pass via the transliterated query variants.
func TestSearch_MatchesAcrossAlphabets(t *testing.T) {
	db := openTestDB(t)
	seedTranslitFixtures(t, db)
	repo := NewRepository(db)

	tests := []struct{ q, want string }{
		{"Erarta", "Музей современного искусства «Эрарта»"},
		{"Garazh", "Музей «Гараж»"},
		{"Artmuza", "Музей современного искусства «Артмуза»"},
		{"Manezh", "ЦВЗ «Манеж»"},
		// Romanization drops the soft sign ("bolshoy" → "болшой"), so this one
		// only lands through the trigram branch.
		{"Bolshoy", "Большой театр"},
		// The other direction: a Cyrillic query for a Latin-spelled venue.
		{"Ноодоме", "Noôdome"},
	}
	for _, tt := range tests {
		t.Run(tt.q, func(t *testing.T) {
			if got := names(t, repo, tt.q); !contains(got, tt.want) {
				t.Fatalf("Search(%q) = %v, want %q", tt.q, got, tt.want)
			}
		})
	}
}

func seedTranslitFixtures(t *testing.T, db *pg.DB) {
	t.Helper()
	rows := []string{
		"Музей современного искусства «Эрарта»",
		"Музей «Гараж»",
		"Музей современного искусства «Артмуза»",
		"ЦВЗ «Манеж»",
		"Большой театр",
		"Noôdome",
	}
	ids := make([]uuid.UUID, 0, len(rows))
	for _, name := range rows {
		id := uuid.Must(uuid.NewV4())
		ids = append(ids, id)
		if _, err := db.Exec(`INSERT INTO venues (id, name) VALUES (?, ?)`, id, name); err != nil {
			t.Fatalf("seed venue %q: %v", name, err)
		}
	}
	t.Cleanup(func() { _, _ = db.Exec(`DELETE FROM venues WHERE id IN (?)`, pg.In(ids)) })
}
