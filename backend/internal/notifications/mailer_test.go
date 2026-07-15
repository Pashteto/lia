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

func TestRenderInvitationEmail_EscapesHTML(t *testing.T) {
	_, body := notifications.RenderInvitationEmail(`<script>alert(1)</script> & "quoted"`, "https://x.test/a?b=1&c=2")
	if strings.Contains(body, "<script>") {
		t.Fatalf("body contains unescaped <script>: %s", body)
	}
	if !strings.Contains(body, "&lt;script&gt;") {
		t.Fatalf("body missing escaped title: %s", body)
	}
	if !strings.Contains(body, "&amp;") {
		t.Fatalf("body missing escaped ampersand: %s", body)
	}
}
