package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/runtime/middleware"
	"github.com/gofrs/uuid"

	"github.com/Pashteto/lia/internal/http/formatter"
	apimodels "github.com/Pashteto/lia/internal/http/models"
	rsvpops "github.com/Pashteto/lia/internal/http/server/operations/rsvp"
	rsvpdomain "github.com/Pashteto/lia/internal/rsvp"
	"github.com/Pashteto/lia/pkg/logger"
)

// rsvpSvcUnavailable writes a plain 503 JSON response when no generated 503 responder exists.
func rsvpSvcUnavailable(err error) middleware.Responder {
	payload := DefaultError(http.StatusServiceUnavailable, err, nil)
	return middleware.ResponderFunc(func(w http.ResponseWriter, _ runtime.Producer) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(payload)
	})
}

// SignUp handles POST /events/{id}/rsvp.
type SignUp struct{ rsvp rsvpdomain.Service }

// NewSignUp constructs a SignUp handler.
func NewSignUp(svc rsvpdomain.Service) *SignUp { return &SignUp{rsvp: svc} }

// Handle registers the caller on the event.
func (h *SignUp) Handle(params rsvpops.SignUpParams, principal *apimodels.User) middleware.Responder {
	if principal == nil {
		return rsvpops.NewSignUpUnauthorized().
			WithPayload(DefaultError(http.StatusUnauthorized, errors.New("authentication required"), nil))
	}
	userID, err := uuid.FromString(principal.UUID.String())
	if err != nil {
		return rsvpops.NewSignUpUnauthorized().
			WithPayload(DefaultError(http.StatusUnauthorized, err, nil))
	}
	eventID, err := uuid.FromString(params.ID.String())
	if err != nil {
		return rsvpops.NewSignUpBadRequest().
			WithPayload(DefaultError(http.StatusBadRequest, err, nil))
	}
	answer := ""
	if params.Body != nil {
		answer = params.Body.ApplicationAnswer
	}
	row, err := h.rsvp.SignUp(params.HTTPRequest.Context(), eventID, userID, answer)
	if err != nil {
		logger.Log().Errorf("signup event %s: %s", eventID, err.Error())
		switch {
		case errors.Is(err, rsvpdomain.ErrNotFound):
			return rsvpops.NewSignUpNotFound().WithPayload(DefaultError(http.StatusNotFound, err, nil))
		case errors.Is(err, rsvpdomain.ErrConflict):
			return rsvpops.NewSignUpConflict().WithPayload(DefaultError(http.StatusConflict, err, nil))
		case errors.Is(err, rsvpdomain.ErrInvalidInput):
			return rsvpops.NewSignUpBadRequest().WithPayload(DefaultError(http.StatusBadRequest, err, nil))
		case errors.Is(err, rsvpdomain.ErrExternal):
			url := strings.TrimPrefix(err.Error(), rsvpdomain.ErrExternal.Error()+": ")
			return rsvpops.NewSignUpUnprocessableEntity().WithPayload(
				DefaultError(http.StatusUnprocessableEntity, errors.New(url), nil))
		default:
			return rsvpSvcUnavailable(err)
		}
	}
	return rsvpops.NewSignUpCreated().WithPayload(formatter.RsvpToAPI(row))
}

// CancelRsvp handles DELETE /events/{id}/rsvp.
type CancelRsvp struct{ rsvp rsvpdomain.Service }

// NewCancelRsvp constructs a CancelRsvp handler.
func NewCancelRsvp(svc rsvpdomain.Service) *CancelRsvp { return &CancelRsvp{rsvp: svc} }

// Handle cancels the caller's registration for the event.
func (h *CancelRsvp) Handle(params rsvpops.CancelRsvpParams, principal *apimodels.User) middleware.Responder {
	if principal == nil {
		return rsvpops.NewCancelRsvpUnauthorized().
			WithPayload(DefaultError(http.StatusUnauthorized, errors.New("authentication required"), nil))
	}
	userID, err := uuid.FromString(principal.UUID.String())
	if err != nil {
		return rsvpops.NewCancelRsvpUnauthorized().
			WithPayload(DefaultError(http.StatusUnauthorized, err, nil))
	}
	eventID, err := uuid.FromString(params.ID.String())
	if err != nil {
		return rsvpops.NewCancelRsvpNotFound().
			WithPayload(DefaultError(http.StatusNotFound, err, nil))
	}
	if err := h.rsvp.Cancel(params.HTTPRequest.Context(), eventID, userID); err != nil {
		logger.Log().Errorf("cancel rsvp event %s: %s", eventID, err.Error())
		switch {
		case errors.Is(err, rsvpdomain.ErrNotFound):
			return rsvpops.NewCancelRsvpNotFound().WithPayload(DefaultError(http.StatusNotFound, err, nil))
		default:
			return rsvpSvcUnavailable(err)
		}
	}
	return rsvpops.NewCancelRsvpNoContent()
}

// MyPractices handles GET /me/practices.
type MyPractices struct{ rsvp rsvpdomain.Service }

// NewMyPractices constructs a MyPractices handler.
func NewMyPractices(svc rsvpdomain.Service) *MyPractices { return &MyPractices{rsvp: svc} }

// Handle returns the caller's practice attendances.
func (h *MyPractices) Handle(params rsvpops.MyPracticesParams, principal *apimodels.User) middleware.Responder {
	if principal == nil {
		return rsvpops.NewMyPracticesUnauthorized().
			WithPayload(DefaultError(http.StatusUnauthorized, errors.New("authentication required"), nil))
	}
	userID, err := uuid.FromString(principal.UUID.String())
	if err != nil {
		return rsvpops.NewMyPracticesUnauthorized().
			WithPayload(DefaultError(http.StatusUnauthorized, err, nil))
	}
	tab := "upcoming"
	if params.Tab != nil {
		tab = *params.Tab
	}
	rows, err := h.rsvp.MyPractices(params.HTTPRequest.Context(), userID, tab)
	if err != nil {
		logger.Log().Errorf("my practices: %s", err.Error())
		return rsvpSvcUnavailable(err)
	}
	payload := make([]*apimodels.Rsvp, 0, len(rows))
	for _, pr := range rows {
		if pr.Rsvp == nil {
			continue
		}
		if pr.Event != nil {
			pr.Rsvp.Event = pr.Event
		}
		payload = append(payload, formatter.RsvpToAPI(pr.Rsvp))
	}
	return rsvpops.NewMyPracticesOK().WithPayload(payload)
}

// MyApplications handles GET /me/applications.
type MyApplications struct{ rsvp rsvpdomain.Service }

// NewMyApplications constructs a MyApplications handler.
func NewMyApplications(svc rsvpdomain.Service) *MyApplications { return &MyApplications{rsvp: svc} }

// Handle returns the caller's applications.
func (h *MyApplications) Handle(params rsvpops.MyApplicationsParams, principal *apimodels.User) middleware.Responder {
	if principal == nil {
		return rsvpops.NewMyApplicationsUnauthorized().
			WithPayload(DefaultError(http.StatusUnauthorized, errors.New("authentication required"), nil))
	}
	userID, err := uuid.FromString(principal.UUID.String())
	if err != nil {
		return rsvpops.NewMyApplicationsUnauthorized().
			WithPayload(DefaultError(http.StatusUnauthorized, err, nil))
	}
	status := ""
	if params.Status != nil {
		status = *params.Status
	}
	rows, err := h.rsvp.MyApplications(params.HTTPRequest.Context(), userID, status)
	if err != nil {
		logger.Log().Errorf("my applications: %s", err.Error())
		return rsvpSvcUnavailable(err)
	}
	payload := make([]*apimodels.Rsvp, 0, len(rows))
	for _, r := range rows {
		payload = append(payload, formatter.RsvpToAPI(r))
	}
	return rsvpops.NewMyApplicationsOK().WithPayload(payload)
}

// ListEventApplications handles GET /events/{id}/applications.
type ListEventApplications struct{ rsvp rsvpdomain.Service }

// NewListEventApplications constructs a ListEventApplications handler.
func NewListEventApplications(svc rsvpdomain.Service) *ListEventApplications {
	return &ListEventApplications{rsvp: svc}
}

// Handle returns the list of applications for an event (organizer only).
func (h *ListEventApplications) Handle(params rsvpops.ListEventApplicationsParams, principal *apimodels.User) middleware.Responder {
	if principal == nil {
		return rsvpops.NewListEventApplicationsUnauthorized().
			WithPayload(DefaultError(http.StatusUnauthorized, errors.New("authentication required"), nil))
	}
	organizerID, err := uuid.FromString(principal.UUID.String())
	if err != nil {
		return rsvpops.NewListEventApplicationsUnauthorized().
			WithPayload(DefaultError(http.StatusUnauthorized, err, nil))
	}
	eventID, err := uuid.FromString(params.ID.String())
	if err != nil {
		return rsvpops.NewListEventApplicationsNotFound().
			WithPayload(DefaultError(http.StatusNotFound, err, nil))
	}
	rows, err := h.rsvp.ListApplications(params.HTTPRequest.Context(), eventID, organizerID)
	if err != nil {
		logger.Log().Errorf("list applications event %s: %s", eventID, err.Error())
		switch {
		case errors.Is(err, rsvpdomain.ErrForbidden):
			return rsvpops.NewListEventApplicationsForbidden().WithPayload(DefaultError(http.StatusForbidden, err, nil))
		case errors.Is(err, rsvpdomain.ErrNotFound):
			return rsvpops.NewListEventApplicationsNotFound().WithPayload(DefaultError(http.StatusNotFound, err, nil))
		default:
			return rsvpSvcUnavailable(err)
		}
	}
	payload := make([]*apimodels.Rsvp, 0, len(rows))
	for _, r := range rows {
		payload = append(payload, formatter.RsvpToAPI(r))
	}
	return rsvpops.NewListEventApplicationsOK().WithPayload(payload)
}

// DecideApplication handles POST /events/{id}/applications/{rsvpId}/decision.
type DecideApplication struct{ rsvp rsvpdomain.Service }

// NewDecideApplication constructs a DecideApplication handler.
func NewDecideApplication(svc rsvpdomain.Service) *DecideApplication {
	return &DecideApplication{rsvp: svc}
}

// Handle accepts or declines an RSVP application.
func (h *DecideApplication) Handle(params rsvpops.DecideApplicationParams, principal *apimodels.User) middleware.Responder {
	if principal == nil {
		return rsvpops.NewDecideApplicationUnauthorized().
			WithPayload(DefaultError(http.StatusUnauthorized, errors.New("authentication required"), nil))
	}
	organizerID, err := uuid.FromString(principal.UUID.String())
	if err != nil {
		return rsvpops.NewDecideApplicationUnauthorized().
			WithPayload(DefaultError(http.StatusUnauthorized, err, nil))
	}
	eventID, err := uuid.FromString(params.ID.String())
	if err != nil {
		return rsvpops.NewDecideApplicationNotFound().
			WithPayload(DefaultError(http.StatusNotFound, err, nil))
	}
	rsvpID, err := uuid.FromString(params.RsvpID.String())
	if err != nil {
		return rsvpops.NewDecideApplicationNotFound().
			WithPayload(DefaultError(http.StatusNotFound, err, nil))
	}
	if params.Body == nil || params.Body.Decision == nil {
		payload := DefaultError(http.StatusBadRequest, errors.New("decision is required"), nil)
		return middleware.ResponderFunc(func(w http.ResponseWriter, _ runtime.Producer) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(payload)
		})
	}
	accept := *params.Body.Decision == "accept"
	row, err := h.rsvp.Decide(params.HTTPRequest.Context(), eventID, organizerID, rsvpID, accept)
	if err != nil {
		logger.Log().Errorf("decide application rsvp %s: %s", rsvpID, err.Error())
		switch {
		case errors.Is(err, rsvpdomain.ErrNotFound):
			return rsvpops.NewDecideApplicationNotFound().WithPayload(DefaultError(http.StatusNotFound, err, nil))
		case errors.Is(err, rsvpdomain.ErrForbidden):
			return rsvpops.NewDecideApplicationForbidden().WithPayload(DefaultError(http.StatusForbidden, err, nil))
		case errors.Is(err, rsvpdomain.ErrConflict):
			return rsvpops.NewDecideApplicationConflict().WithPayload(DefaultError(http.StatusConflict, err, nil))
		default:
			return rsvpSvcUnavailable(err)
		}
	}
	return rsvpops.NewDecideApplicationOK().WithPayload(formatter.RsvpToAPI(row))
}
