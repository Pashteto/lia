// Package organizers provides plain net/http handlers for the user-facing
// organizer profile (/api/v1/me/organizer) and the public organizer page
// (/api/v1/organizers/{id}). Mounted ahead of the go-swagger mux in
// internal/http/module.go (mirrors internal/http/uploads).
package organizers

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/gofrs/uuid"

	domain "github.com/Pashteto/lia/internal/models"
	orgdomain "github.com/Pashteto/lia/internal/organizers"
	"github.com/Pashteto/lia/internal/storage"
)

// Deps are the collaborators the handler needs.
type Deps struct {
	Authenticate func(token string) (*domain.User, error)
	Organizers   orgdomain.Service
	Store        storage.Storage // may be nil; logo URLs omitted when nil
}

type handler struct {
	deps Deps
	mux  *http.ServeMux
}

// NewHandler returns the mounted organizers handler.
func NewHandler(deps Deps) http.Handler {
	h := &handler{deps: deps, mux: http.NewServeMux()}
	h.mux.HandleFunc("GET /api/v1/me/organizer", h.getMine)
	h.mux.HandleFunc("PUT /api/v1/me/organizer", h.putMine)
	h.mux.HandleFunc("POST /api/v1/me/organizer/submit", h.submit)
	h.mux.HandleFunc("GET /api/v1/organizers/{id}", h.getPublic)
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

type organizerJSON struct {
	ID                 string `json:"id"`
	Name               string `json:"name"`
	Description        string `json:"description"`
	WebsiteURL         string `json:"website_url"`
	LogoURL            string `json:"logo_url,omitempty"`
	VerificationStatus string `json:"verification_status"`
	AutoVerify         bool   `json:"auto_verify"`
	LatestReason       string `json:"latest_reason,omitempty"`
}

func (h *handler) toJSON(o *orgdomain.Organizer) organizerJSON {
	j := organizerJSON{
		ID:                 o.ID.String(),
		Name:               o.Name,
		Description:        o.Description,
		WebsiteURL:         o.WebsiteURL,
		VerificationStatus: o.VerificationStatus,
		AutoVerify:         o.AutoVerify,
		LatestReason:       o.LatestReason,
	}
	// logo_url resolution (logo_file_id -> files.storage_key -> URL) is a follow-up; the client holds the uploaded file id.
	return j
}

func (h *handler) getMine(w http.ResponseWriter, r *http.Request) {
	u := h.principal(r)
	if u == nil {
		writeErr(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	o, err := h.deps.Organizers.GetByOwner(r.Context(), u.UUID)
	if err == orgdomain.ErrNotFound {
		writeErr(w, http.StatusNotFound, "Профиль организатора не создан")
		return
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "load failed")
		return
	}
	writeJSON(w, http.StatusOK, h.toJSON(o))
}

func (h *handler) putMine(w http.ResponseWriter, r *http.Request) {
	u := h.principal(r)
	if u == nil {
		writeErr(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var body struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		WebsiteURL  string `json:"website_url"`
		LogoFileID  string `json:"logo_file_id"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	logo := uuid.Nil
	if body.LogoFileID != "" {
		if id, err := uuid.FromString(body.LogoFileID); err == nil {
			logo = id
		}
	}
	o, err := h.deps.Organizers.Upsert(r.Context(), u.UUID, orgdomain.Input{
		Name: body.Name, Description: body.Description, WebsiteURL: body.WebsiteURL, LogoFileID: logo,
	})
	switch err {
	case nil:
		writeJSON(w, http.StatusOK, h.toJSON(o))
	case orgdomain.ErrNameRequired:
		writeErr(w, http.StatusBadRequest, "Укажите название организатора")
	default:
		writeErr(w, http.StatusInternalServerError, "save failed")
	}
}

func (h *handler) submit(w http.ResponseWriter, r *http.Request) {
	u := h.principal(r)
	if u == nil {
		writeErr(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	status, err := h.deps.Organizers.Submit(r.Context(), u.UUID)
	switch err {
	case nil:
		writeJSON(w, http.StatusOK, map[string]string{"status": status})
	case orgdomain.ErrNotFound:
		writeErr(w, http.StatusNotFound, "Сначала создайте профиль организатора")
	case orgdomain.ErrInvalidTransition:
		writeErr(w, http.StatusConflict, "Профиль нельзя отправить из текущего статуса")
	default:
		writeErr(w, http.StatusInternalServerError, "submit failed")
	}
}

type publicOrganizerJSON struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	WebsiteURL  string `json:"website_url"`
	Verified    bool   `json:"verified"`
}

func (h *handler) getPublic(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.FromString(r.PathValue("id"))
	if err != nil {
		writeErr(w, http.StatusBadRequest, "invalid id")
		return
	}
	o, err := h.deps.Organizers.GetByID(r.Context(), id)
	if err != nil || o.VerificationStatus != "verified" {
		// Don't leak pending/rejected/draft profiles.
		writeErr(w, http.StatusNotFound, "not found")
		return
	}
	writeJSON(w, http.StatusOK, publicOrganizerJSON{
		ID: o.ID.String(), Name: o.Name, Description: o.Description,
		WebsiteURL: o.WebsiteURL, Verified: true,
	})
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}
