package notifications_test

import (
	"strings"
	"testing"

	"github.com/Pashteto/lia/internal/notifications"
)

func TestRenderInvitationEmail(t *testing.T) {
	subject, body := notifications.RenderInvitationEmail("Йога в парке", "https://presence.tarski.ru/invite/abc")
	if !strings.HasPrefix(subject, "Subject:") {
		t.Fatalf("subject must start with 'Subject:', got %q", subject)
	}
	if !strings.Contains(body, "Йога в парке") || !strings.Contains(body, "https://presence.tarski.ru/invite/abc") {
		t.Fatalf("body missing title or link: %s", body)
	}
}
