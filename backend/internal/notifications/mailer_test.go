package notifications_test

import (
	"net/mail"
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

func TestRenderInvitationEmail_Brand(t *testing.T) {
	subject, _ := notifications.RenderInvitationEmail("Лекция", "https://example.test/invite/x")
	if subject != "Subject: Сообща: приглашение на событие" {
		t.Fatalf("unexpected subject %q", subject)
	}
}

func TestFromHeader_NamesTheSender(t *testing.T) {
	h := notifications.FromHeader("info@tarski.ru")
	for _, r := range h {
		if r > 127 {
			t.Fatalf("From header must be ASCII (RFC 2047-encoded), got %q", h)
		}
	}
	addr, err := mail.ParseAddress(h)
	if err != nil {
		t.Fatalf("From header %q does not parse: %v", h, err)
	}
	if addr.Name != "Сообща" || addr.Address != "info@tarski.ru" {
		t.Fatalf("got name %q address %q", addr.Name, addr.Address)
	}
}
