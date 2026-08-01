// Package venues is the venue domain module of the Lia monolith. It owns the
// venue entity, search, and find-or-create. Identity only — geo arrives later.
package venues

import (
	"errors"
	"fmt"
	"strings"

	"github.com/go-pg/pg/v10"
	"github.com/gofrs/uuid"

	"github.com/Pashteto/lia/internal/models"
)

// DefaultSearchLimit caps Search results when no limit is given.
const DefaultSearchLimit = 20

// Repository defines venue persistence operations.
type Repository interface {
	// Search returns venues whose name, address or metro matches q
	// (case-insensitive substring, diacritics folded), plus name near-misses
	// by trigram similarity. Name hits rank first, then closer trigram
	// matches, then name order. Empty q returns the first `limit` venues by
	// name.
	Search(q string, limit int) ([]*models.Venue, error)
	// GetByID returns a single venue by primary key.
	GetByID(id uuid.UUID) (*models.Venue, error)
	// GetByIDs returns the venues matching the given ids.
	GetByIDs(ids []uuid.UUID) ([]*models.Venue, error)
	// FindOrCreateByName returns an existing venue whose lower(name) matches
	// v.Name, else inserts v and returns it.
	FindOrCreateByName(v *models.Venue) (*models.Venue, error)
	// Update persists name, address, metro, district, lat, lon and updated_at
	// for an existing venue (identified by primary key). Returns pg.ErrNoRows
	// if no row was matched.
	Update(v *models.Venue) error
}

type pgRepository struct {
	db *pg.DB
}

// NewRepository creates a PostgreSQL-backed venue repository.
func NewRepository(db *pg.DB) Repository {
	return &pgRepository{db: db}
}

// likeEscaper neutralizes the three characters LIKE reads as syntax, so a user
// searching for "50%" gets the venue with a percent sign in its name rather
// than every row in the table. The backslash case comes first by construction:
// strings.NewReplacer matches replacements left to right and never rescans its
// own output.
var likeEscaper = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

// searchPattern turns a raw user query into a safe ILIKE substring pattern.
func searchPattern(q string) string {
	return "%" + likeEscaper.Replace(strings.TrimSpace(q)) + "%"
}

// trigramThreshold is the cutoff for the fuzzy branch, applied to
// word_similarity rather than plain similarity.
//
// similarity() compares whole strings, so it collapses as soon as the query is
// short relative to the name: "эрарта" against «Музей современного искусства
// «Эрарта»» scores 0.194, and "заряде" against «Концертный зал «Зарядье»»
// scores 0.217 — both under the 0.3 default, both obviously the right answer.
// word_similarity() scores the query against the best-matching extent of the
// name instead, giving 1.000 and 0.714.
//
// 0.45 rather than the 0.6 word_similarity default: measured against the real
// 174-row table, 0.6 drops both "nodom" → «Noôdome» and "bolshoy" → «Большой
// театр» (each 0.500), while 0.45 keeps hit counts tight (1 and 4).
//
// Inlined as a literal, not a bind parameter, so the planner sees a constant.
// It is a package constant and never user input.
const trigramThreshold = "0.45"

// Search matches q against the name, address and metro, all with diacritics
// folded (unaccent), and falls back to trigram similarity on the name so a
// near-miss spelling still finds the venue. Every branch is repeated for each
// transliteration of the query (see searchVariants), because neither unaccent
// nor trigrams can cross between Latin and Cyrillic. Name hits outrank
// address/metro hits, then closer trigram matches, then alphabetical order.
//
// unaccent() and the pg_trgm `%` operator come from migration 000022.
func (r *pgRepository) Search(q string, limit int) ([]*models.Venue, error) {
	if limit <= 0 {
		limit = DefaultSearchLimit
	}
	var list []*models.Venue
	query := r.db.Model(&list)
	if trimmed := strings.TrimSpace(q); trimmed != "" {
		var (
			match      []string
			matchArgs  []interface{}
			nameHit    []string
			nameArgs   []interface{}
			substrHit  []string
			substrArgs []interface{}
			sim        []string
			simArgs    []interface{}
		)
		for _, v := range searchVariants(trimmed) {
			pattern := searchPattern(v)
			match = append(match,
				"unaccent(name) ILIKE unaccent(?)",
				"unaccent(address) ILIKE unaccent(?)",
				"unaccent(metro) ILIKE unaccent(?)",
				"word_similarity(unaccent(?), unaccent(name)) > "+trigramThreshold,
			)
			matchArgs = append(matchArgs, pattern, pattern, pattern, v)
			nameHit = append(nameHit, "unaccent(name) ILIKE unaccent(?)")
			nameArgs = append(nameArgs, pattern)
			substrHit = append(substrHit,
				"unaccent(name) ILIKE unaccent(?)",
				"unaccent(address) ILIKE unaccent(?)",
				"unaccent(metro) ILIKE unaccent(?)",
			)
			substrArgs = append(substrArgs, pattern, pattern, pattern)
			sim = append(sim, "word_similarity(unaccent(?), unaccent(name))")
			simArgs = append(simArgs, v)
		}
		// Three tiers: the name literally contains the query, then any field
		// does, then how close the fuzzy branch got. Without the middle tier a
		// venue matched on its metro sorts below unrelated names that merely
		// scored above the trigram threshold.
		query = query.
			Where(strings.Join(match, " OR "), matchArgs...).
			OrderExpr("("+strings.Join(nameHit, " OR ")+") DESC", nameArgs...).
			OrderExpr("("+strings.Join(substrHit, " OR ")+") DESC", substrArgs...).
			OrderExpr("GREATEST("+strings.Join(sim, ", ")+") DESC", simArgs...)
	}
	if err := query.Order("name ASC").Limit(limit).Select(); err != nil {
		return nil, fmt.Errorf("search venues from db: %w", err)
	}
	return list, nil
}

func (r *pgRepository) GetByID(id uuid.UUID) (*models.Venue, error) {
	venue := &models.Venue{ID: id}
	if err := r.db.Model(venue).WherePK().Select(); err != nil {
		return nil, fmt.Errorf("get venue %s from db: %w", id, err)
	}
	return venue, nil
}

func (r *pgRepository) GetByIDs(ids []uuid.UUID) ([]*models.Venue, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var list []*models.Venue
	if err := r.db.Model(&list).Where("id IN (?)", pg.In(ids)).Select(); err != nil {
		return nil, fmt.Errorf("get venues by ids from db: %w", err)
	}
	return list, nil
}

// FindOrCreateByName is non-atomic (SELECT then INSERT): two concurrent calls
// with the same normalized name can create two rows. Acceptable per spec (no
// unique constraint on name); a future migration may add a partial unique index
// if dedup needs hardening.
func (r *pgRepository) FindOrCreateByName(v *models.Venue) (*models.Venue, error) {
	existing := new(models.Venue)
	err := r.db.Model(existing).
		Where("lower(name) = lower(?)", v.Name).
		Limit(1).
		Select()
	if err == nil {
		return existing, nil
	}
	if !errors.Is(err, pg.ErrNoRows) {
		return nil, fmt.Errorf("find venue by name: %w", err)
	}
	if _, err := r.db.Model(v).Insert(); err != nil {
		return nil, fmt.Errorf("insert venue %q: %w", v.Name, err)
	}
	return v, nil
}

func (r *pgRepository) Update(v *models.Venue) error {
	res, err := r.db.Model(v).
		Column("name", "address", "metro", "district", "lat", "lon", "updated_at").
		WherePK().
		Update()
	if err != nil {
		return fmt.Errorf("update venue %s: %w", v.ID, err)
	}
	if res.RowsAffected() == 0 {
		return pg.ErrNoRows
	}
	return nil
}
