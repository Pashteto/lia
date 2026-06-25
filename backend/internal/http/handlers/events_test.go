package handlers

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/go-openapi/strfmt"
	"github.com/gofrs/uuid"

	eventsdomain "github.com/Pashteto/lia/internal/events"
	"github.com/Pashteto/lia/internal/http/models"
	eventsops "github.com/Pashteto/lia/internal/http/server/operations/events"
	domainmodels "github.com/Pashteto/lia/internal/models"
)

// mockEventsService captures the event passed to Create.
type mockEventsService struct {
	created    *domainmodels.Event
	createErr  error
}

func (m *mockEventsService) Create(_ context.Context, e *domainmodels.Event) error {
	m.created = e
	return m.createErr
}
func (m *mockEventsService) GetByID(context.Context, string) (*domainmodels.Event, error) {
	return nil, nil
}
func (m *mockEventsService) List(context.Context, string) ([]*domainmodels.Event, error) {
	return nil, nil
}
func (m *mockEventsService) Nearby(context.Context, *float64, *float64, int) ([]*eventsdomain.NearbyResult, error) {
	return nil, nil
}

func TestCreateEvent_SetsOrganizerFromPrincipal(t *testing.T) {
	svc := &mockEventsService{}
	h := NewCreateEvent(svc)

	title := "Test Event"
	starts := strfmt.DateTime(time.Now())
	// A client-supplied organizer_id must be IGNORED in favor of the principal.
	bodyOrganizer := strfmt.UUID(uuid.Must(uuid.NewV4()).String())
	params := eventsops.CreateEventParams{
		Body: &models.EventInput{
			Title:       &title,
			StartsAt:    &starts,
			OrganizerID: bodyOrganizer,
		},
	}
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/events", nil)
	params.HTTPRequest = req

	principalUUID := uuid.Must(uuid.NewV4())
	pu := strfmt.UUID(principalUUID.String())
	email := strfmt.Email("u@example.com")
	name := "U"
	status := "active"
	principal := &models.User{UUID: pu, Email: &email, Name: &name, Status: &status}

	resp := h.Handle(params, principal)
	if resp == nil {
		t.Fatal("nil responder")
	}
	if svc.created == nil {
		t.Fatal("Create was not called")
	}
	if svc.created.OrganizerID != principalUUID {
		t.Errorf("expected organizer %s (from principal), got %s", principalUUID, svc.created.OrganizerID)
	}
}
