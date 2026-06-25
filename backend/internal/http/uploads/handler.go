// Package uploads provides plain net/http handlers for image upload and blob serving.
// These are mounted ahead of the go-swagger mux in internal/http/module.go.
package uploads

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/gofrs/uuid"

	apimodels "github.com/Pashteto/lia/internal/http/models"
	filesdomain "github.com/Pashteto/lia/internal/files"
	"github.com/Pashteto/lia/internal/storage"
)

const maxBytes = 5 << 20 // 5 MiB

// allowedMIME maps accepted image content types to their file extensions.
var allowedMIME = map[string]string{
	"image/png":  ".png",
	"image/jpeg": ".jpg",
	"image/webp": ".webp",
}

// uploadResponse is the 201 JSON payload.
type uploadResponse struct {
	ID  string `json:"id"`
	URL string `json:"url"`
}

type handler struct {
	store  storage.Storage
	files  filesdomain.Service
	authFn func(token string) (*apimodels.User, error)
	mux    *http.ServeMux
}

// NewHandler returns an http.Handler that handles:
//
//	POST /api/v1/uploads      — authenticated image upload → 201 {"id","url"}
//	GET  /api/v1/files/{key}  — public blob stream
func NewHandler(store storage.Storage, files filesdomain.Service, authFn func(token string) (*apimodels.User, error)) http.Handler {
	h := &handler{
		store:  store,
		files:  files,
		authFn: authFn,
		mux:    http.NewServeMux(),
	}
	h.mux.HandleFunc("/api/v1/uploads", h.upload)
	h.mux.HandleFunc("/api/v1/files/", h.serve)
	return h
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.mux.ServeHTTP(w, r)
}

// upload handles POST /api/v1/uploads.
func (h *handler) upload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Auth: require Bearer token with explicit "Bearer " prefix.
	authHeader := r.Header.Get("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	token := strings.TrimPrefix(authHeader, "Bearer ")
	principal, err := h.authFn(token)
	if err != nil || principal == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse owner UUID from principal.
	ownerID, err := uuid.FromString(string(principal.UUID))
	if err != nil {
		ownerID = uuid.Nil
	}

	// Hard cap: abort oversized requests at the network level before any disk spill.
	r.Body = http.MaxBytesReader(w, r.Body, maxBytes+1)

	// Parse multipart (cap at maxBytes + 1 to detect oversize).
	if err := r.ParseMultipartForm(maxBytes); err != nil {
		if errors.Is(err, http.ErrNotMultipart) {
			http.Error(w, "multipart/form-data required", http.StatusBadRequest)
			return
		}
		// MaxBytesReader signals an oversized body via a non-standard error value;
		// treat any parse failure other than ErrNotMultipart as too large.
		http.Error(w, "request too large", http.StatusRequestEntityTooLarge)
		return
	}

	part, _, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "missing file field", http.StatusBadRequest)
		return
	}
	defer part.Close()

	// Read up to maxBytes + 1 to detect oversize.
	buf := &bytes.Buffer{}
	n, err := io.CopyN(buf, part, maxBytes+1)
	if err != nil && !errors.Is(err, io.EOF) {
		http.Error(w, "read error", http.StatusInternalServerError)
		return
	}
	if n > maxBytes {
		http.Error(w, "file too large (max 5 MB)", http.StatusRequestEntityTooLarge)
		return
	}
	size := n
	data := buf.Bytes()

	// Determine content type by sniffing the actual bytes — never trust the
	// declared Content-Type header from the multipart part (MIME spoofing defence).
	sniff := data
	if len(sniff) > 512 {
		sniff = sniff[:512]
	}
	ct := http.DetectContentType(sniff)
	// Normalise (strip parameters like "; charset=utf-8").
	if idx := strings.Index(ct, ";"); idx >= 0 {
		ct = strings.TrimSpace(ct[:idx])
	}

	ext, ok := allowedMIME[ct]
	if !ok {
		http.Error(w, fmt.Sprintf("unsupported media type %q; allowed: image/png, image/jpeg, image/webp", ct), http.StatusUnsupportedMediaType)
		return
	}

	// Generate storage key.
	fileID := uuid.Must(uuid.NewV4())
	key := "uploads/" + fileID.String() + ext

	// Store the bytes.
	if err := h.store.Put(r.Context(), key, bytes.NewReader(data), size, ct); err != nil {
		http.Error(w, "storage error", http.StatusInternalServerError)
		return
	}

	// Register metadata.
	registered, err := h.files.Register(r.Context(), key, ct, size, ownerID)
	if err != nil {
		// Best-effort cleanup; ignore error.
		_ = h.store.Delete(r.Context(), key)
		http.Error(w, "metadata error", http.StatusInternalServerError)
		return
	}

	resp := uploadResponse{
		ID:  registered.ID.String(),
		URL: h.store.URL(key),
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(resp)
}

// serve handles GET /api/v1/files/{key...}.
func (h *handler) serve(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Strip the /api/v1/files/ prefix to get the storage key.
	key := strings.TrimPrefix(r.URL.Path, "/api/v1/files/")
	if key == "" {
		http.Error(w, "missing key", http.StatusBadRequest)
		return
	}

	rc, err := h.store.Get(r.Context(), key)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		http.Error(w, "storage error", http.StatusInternalServerError)
		return
	}
	defer rc.Close()

	// Sniff content type from first 512 bytes.
	var sniff [512]byte
	nr, _ := rc.Read(sniff[:])
	ct := http.DetectContentType(sniff[:nr])
	w.Header().Set("Content-Type", ct)

	// Write the sniffed bytes then stream the rest.
	if _, err := w.Write(sniff[:nr]); err != nil {
		return
	}
	_, _ = io.Copy(w, rc)
}
