package uploads

import (
	"bytes"
	"context"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"testing"

	"github.com/go-openapi/strfmt"
	"github.com/gofrs/uuid"

	apimodels "github.com/Pashteto/lia/internal/http/models"
	"github.com/Pashteto/lia/internal/models"
	"github.com/Pashteto/lia/internal/storage"
)

// strfmtUUID returns a valid *apimodels.User principal for tests.
func okPrincipal() *apimodels.User {
	email := strfmt.Email("test@example.com")
	name := "Test User"
	status := "active"
	return &apimodels.User{
		UUID:   strfmt.UUID(uuid.Must(uuid.NewV4()).String()),
		Email:  &email,
		Name:   &name,
		Status: &status,
	}
}

func okAuth(string) (*apimodels.User, error) {
	return okPrincipal(), nil
}

// multipartFile builds a multipart/form-data body with one file field.
// It sets the part's Content-Type header so the handler can read it.
func multipartFile(t *testing.T, field, filename, contentType string, data []byte) (*bytes.Buffer, string) {
	t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", `form-data; name="`+field+`"; filename="`+filename+`"`)
	h.Set("Content-Type", contentType)
	part, err := w.CreatePart(h)
	if err != nil {
		t.Fatalf("create part: %v", err)
	}
	if _, err := part.Write(data); err != nil {
		t.Fatalf("write part: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}
	return &buf, w.FormDataContentType()
}

// memStore creates a local storage backed by a temp directory.
func memStore(t *testing.T) storage.Storage {
	t.Helper()
	s, err := storage.NewLocal(t.TempDir(), "http://localhost:8080/api/v1/files")
	if err != nil {
		t.Fatalf("create local storage: %v", err)
	}
	return s
}

// fakeFiles implements files.Service for tests.
type fakeFiles struct {
	registered *models.File
}

func (f *fakeFiles) Register(_ context.Context, key, ct string, size int64, owner uuid.UUID) (*models.File, error) {
	f.registered = &models.File{
		ID:          uuid.Must(uuid.NewV4()),
		StorageKey:  key,
		ContentType: ct,
		Size:        size,
		OwnerUserID: owner,
	}
	return f.registered, nil
}

func (f *fakeFiles) Get(_ context.Context, id uuid.UUID) (*models.File, error) {
	if f.registered != nil && f.registered.ID == id {
		return f.registered, nil
	}
	return nil, nil
}

// --- Tests ---

func TestUpload_RejectsAnonymous(t *testing.T) {
	h := NewHandler(memStore(t), &fakeFiles{}, func(string) (*apimodels.User, error) {
		return nil, http.ErrNoCookie
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/uploads", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("want 401, got %d", rec.Code)
	}
}

func TestUpload_RejectsNonImage(t *testing.T) {
	body, ct := multipartFile(t, "file", "x.txt", "text/plain", []byte("nope"))
	h := NewHandler(memStore(t), &fakeFiles{}, okAuth)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/uploads", body)
	req.Header.Set("Content-Type", ct)
	req.Header.Set("Authorization", "Bearer t")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("want 415, got %d", rec.Code)
	}
}

func TestUpload_AcceptsImage_Returns201WithURL(t *testing.T) {
	// Minimal 4-byte PNG magic bytes
	body, ct := multipartFile(t, "file", "a.png", "image/png", []byte{0x89, 0x50, 0x4e, 0x47})
	h := NewHandler(memStore(t), &fakeFiles{}, okAuth)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/uploads", body)
	req.Header.Set("Content-Type", ct)
	req.Header.Set("Authorization", "Bearer t")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("want 201, got %d body=%s", rec.Code, rec.Body.String())
	}
	body2 := rec.Body.String()
	if len(body2) == 0 {
		t.Fatal("empty response body")
	}
	// Must contain "id" and "url" keys
	if !bytes.Contains([]byte(body2), []byte("\"id\"")) {
		t.Fatalf("response missing 'id': %s", body2)
	}
	if !bytes.Contains([]byte(body2), []byte("\"url\"")) {
		t.Fatalf("response missing 'url': %s", body2)
	}
}

func TestServe_Returns404ForMissingKey(t *testing.T) {
	h := NewHandler(memStore(t), &fakeFiles{}, okAuth)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/files/uploads/nonexistent.png", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("want 404, got %d", rec.Code)
	}
}
