// Package follows provides plain net/http handlers for the user-facing
// organizer subscriptions (/api/v1/me/follows) and the personal calendar
// (/api/v1/me/calendar). Mounted ahead of the go-swagger mux in
// internal/http/module.go (mirrors internal/http/organizers).
package follows

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gofrs/uuid"

	eventsdomain "github.com/Pashteto/lia/internal/events"
	followsdomain "github.com/Pashteto/lia/internal/follows"
	"github.com/Pashteto/lia/internal/http/formatter"
	apimodels "github.com/Pashteto/lia/internal/http/models"
	domain "github.com/Pashteto/lia/internal/models"
	rsvpdomain "github.com/Pashteto/lia/internal/rsvp"
	"github.com/Pashteto/lia/internal/storage"
)

// defaultCalendarWindow is how far ahead /me/calendar looks when `to` is omitted.
const defaultCalendarWindow = 90 * 24 * time.Hour

// maxCalendarWindow caps the queried range to bound the work per request.
const maxCalendarWindow = 366 * 24 * time.Hour

// errInvalidRange is returned when the calendar `to` is not after `from`.
var errInvalidRange = errors.New("calendar: to must be after from")

// Deps are the collaborators the handler needs.
type Deps struct {
	Authenticate func(token string) (*domain.User, error)
	Follows      followsdomain.Service
	Rsvp         rsvpdomain.Service
	Events       eventsdomain.Service
	Store        storage.Storage // may be nil; logo URLs omitted when nil
}

type handler struct {
	deps Deps
	mux  *http.ServeMux
}

// NewHandler returns the mounted follows + calendar handler.
func NewHandler(deps Deps) http.Handler {
	h := &handler{deps: deps, mux: http.NewServeMux()}
	h.mux.HandleFunc("POST /api/v1/me/follows/{organizerId}", h.follow)
	h.mux.HandleFunc("DELETE /api/v1/me/follows/{organizerId}", h.unfollow)
	h.mux.HandleFunc("GET /api/v1/me/follows", h.list)
	h.mux.HandleFunc("GET /api/v1/me/calendar", h.calendar)
	return h
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) { h.mux.ServeHTTP(w, r) }

func (h *handler) principal(r *http.Request) *domain.User {
	authHeader := r.Header.Get("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		return nil
	}
	u, err := h.deps.Authenticate(strings.TrimPrefix(authHeader, "Bearer "))
	if err != nil || u == nil {
		return nil
	}
	return u
}

func (h *handler) follow(w http.ResponseWriter, r *http.Request) {
	u := h.principal(r)
	if u == nil {
		writeErr(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	id, err := uuid.FromString(r.PathValue("organizerId"))
	if err != nil {
		writeErr(w, http.StatusBadRequest, "invalid id")
		return
	}
	switch err := h.deps.Follows.Follow(r.Context(), u.UUID, id); err {
	case nil:
		writeJSON(w, http.StatusOK, map[string]bool{"following": true})
	case followsdomain.ErrNotFound:
		writeErr(w, http.StatusNotFound, "not found")
	default:
		writeErr(w, http.StatusInternalServerError, "follow failed")
	}
}

func (h *handler) unfollow(w http.ResponseWriter, r *http.Request) {
	u := h.principal(r)
	if u == nil {
		writeErr(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	id, err := uuid.FromString(r.PathValue("organizerId"))
	if err != nil {
		writeErr(w, http.StatusBadRequest, "invalid id")
		return
	}
	switch err := h.deps.Follows.Unfollow(r.Context(), u.UUID, id); err {
	case nil:
		writeJSON(w, http.StatusOK, map[string]bool{"following": false})
	case followsdomain.ErrNotFound:
		writeErr(w, http.StatusNotFound, "not found")
	default:
		writeErr(w, http.StatusInternalServerError, "unfollow failed")
	}
}

type followedOrgJSON struct {
	ProfileID string `json:"profile_id"`
	Name      string `json:"name"`
	LogoURL   string `json:"logo_url,omitempty"`
}

func (h *handler) list(w http.ResponseWriter, r *http.Request) {
	u := h.principal(r)
	if u == nil {
		writeErr(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	rows, err := h.deps.Follows.ListFollowed(r.Context(), u.UUID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "load failed")
		return
	}
	out := make([]followedOrgJSON, 0, len(rows))
	for _, row := range rows {
		// logo_url resolution (logo_file_id -> files.storage_key -> URL) is a
		// shared follow-up with the organizer page; omitted until wired there.
		out = append(out, followedOrgJSON{ProfileID: row.ProfileID.String(), Name: row.Name})
	}
	writeJSON(w, http.StatusOK, out)
}

// calendarEventJSON is an enriched event plus the two source flags. An event can
// be both attending and from a followed organizer.
type calendarEventJSON struct {
	*apimodels.Event
	Attending    bool `json:"attending"`
	FromFollowed bool `json:"from_followed"`
}

type calendarFlags struct {
	attending    bool
	fromFollowed bool
}

func (h *handler) calendar(w http.ResponseWriter, r *http.Request) {
	u := h.principal(r)
	if u == nil {
		writeErr(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	from, to, err := parseRange(r)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "invalid range")
		return
	}

	flags := make(map[uuid.UUID]*calendarFlags)
	mark := func(id uuid.UUID) *calendarFlags {
		f, ok := flags[id]
		if !ok {
			f = &calendarFlags{}
			flags[id] = f
		}
		return f
	}

	// Branch A: events from followed organizers.
	followed, err := h.deps.Follows.ListEventsFromFollowed(r.Context(), u.UUID, from, to)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "calendar failed")
		return
	}
	for _, e := range followed {
		mark(e.ID).fromFollowed = true
	}

	// Branch B: events the user has an active RSVP to.
	attending, err := h.deps.Rsvp.ListActiveEventsInRange(r.Context(), u.UUID, from, to)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "calendar failed")
		return
	}
	for _, e := range attending {
		mark(e.ID).attending = true
	}

	if len(flags) == 0 {
		writeJSON(w, http.StatusOK, []calendarEventJSON{})
		return
	}

	// Re-enrich the merged id set so every row is shaped identically.
	ids := make([]uuid.UUID, 0, len(flags))
	for id := range flags {
		ids = append(ids, id)
	}
	enriched, err := h.deps.Events.GetEnriched(r.Context(), ids)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "calendar failed")
		return
	}

	out := make([]calendarEventJSON, 0, len(enriched))
	for _, e := range enriched {
		f := flags[e.ID]
		out = append(out, calendarEventJSON{
			Event:        formatter.EventToAPI(e),
			Attending:    f.attending,
			FromFollowed: f.fromFollowed,
		})
	}
	writeJSON(w, http.StatusOK, out)
}

// parseRange reads from/to RFC3339 query params. from defaults to now; to
// defaults to from + defaultCalendarWindow. The window is capped at
// maxCalendarWindow. An inverted range (to <= from) is an error.
func parseRange(r *http.Request) (time.Time, time.Time, error) {
	now := time.Now()
	from := now
	if v := r.URL.Query().Get("from"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			return time.Time{}, time.Time{}, err
		}
		from = t
	}
	to := from.Add(defaultCalendarWindow)
	if v := r.URL.Query().Get("to"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			return time.Time{}, time.Time{}, err
		}
		to = t
	}
	if !to.After(from) {
		return time.Time{}, time.Time{}, errInvalidRange
	}
	if to.Sub(from) > maxCalendarWindow {
		to = from.Add(maxCalendarWindow)
	}
	return from, to, nil
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}
