package formatter

import (
	"testing"

	"github.com/gofrs/uuid"

	domainModels "github.com/Pashteto/lia/internal/models"
)

func TestRsvpToAPI_MapsApplicantName(t *testing.T) {
	uid := uuid.Must(uuid.NewV4())
	in := &domainModels.Rsvp{
		ID: uuid.Must(uuid.NewV4()), EventID: uuid.Must(uuid.NewV4()), UserID: uid,
		Status: domainModels.RsvpApplied, ApplicantName: "Иван Петров",
	}
	out := RsvpToAPI(in)
	if out.Applicant == nil {
		t.Fatal("expected applicant to be set")
	}
	if out.Applicant.Name != "Иван Петров" {
		t.Errorf("name = %q, want %q", out.Applicant.Name, "Иван Петров")
	}
	if out.Applicant.UUID.String() != uid.String() {
		t.Errorf("uuid = %q, want %q", out.Applicant.UUID, uid)
	}
}

func TestRsvpToAPI_OmitsApplicantWhenNameEmpty(t *testing.T) {
	out := RsvpToAPI(&domainModels.Rsvp{
		ID: uuid.Must(uuid.NewV4()), EventID: uuid.Must(uuid.NewV4()), UserID: uuid.Must(uuid.NewV4()),
		Status: domainModels.RsvpApplied,
	})
	if out.Applicant != nil {
		t.Error("expected applicant nil when name empty")
	}
}
