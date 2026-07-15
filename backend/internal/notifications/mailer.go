package notifications

import (
	"context"
	"errors"
	"fmt"
	"net/smtp"
	"strings"
)

// Mailer sends transactional email for the Lia app.
type Mailer interface {
	SendEventInvitation(ctx context.Context, to, eventTitle, acceptURL string) error
}

type smtpMailer struct {
	addr, from, username, password string
}

// NewSMTPMailer builds an SMTP mailer (SendPulse). A blank addr yields a no-op
// mailer so local/dev runs without SMTP config don't fail invites.
func NewSMTPMailer(addr, username, password, from string) Mailer {
	if addr == "" {
		return noopMailer{}
	}
	return &smtpMailer{addr: addr, from: from, username: username, password: password}
}

func (m *smtpMailer) Start(_ *smtp.ServerInfo) (string, []byte, error) { return "LOGIN", []byte{}, nil }
func (m *smtpMailer) Next(fromServer []byte, more bool) ([]byte, error) {
	if !more {
		return nil, nil
	}
	switch string(fromServer) {
	case "Username:":
		return []byte(m.username), nil
	case "Password:":
		return []byte(m.password), nil
	default:
		return nil, errors.New("unexpected SMTP server challenge")
	}
}

func (m *smtpMailer) SendEventInvitation(_ context.Context, to, eventTitle, acceptURL string) error {
	subject, body := RenderInvitationEmail(eventTitle, acceptURL)
	headers := []string{
		"MIME-version: 1.0;",
		`Content-Type: text/html; charset="UTF-8";`,
		fmt.Sprintf("From: %s", m.from),
		fmt.Sprintf("To: %s", to),
		subject,
	}
	msg := []byte(strings.Join(headers, "\r\n") + "\r\n\r\n" + body)
	if err := smtp.SendMail(m.addr, m, m.from, []string{to}, msg); err != nil {
		return fmt.Errorf("send invitation email: %w", err)
	}
	return nil
}

// RenderInvitationEmail builds the subject line and HTML body (pure/testable).
func RenderInvitationEmail(eventTitle, acceptURL string) (string, string) {
	subject := "Subject: Presence: приглашение на событие"
	body := fmt.Sprintf(`<!DOCTYPE html><html lang="ru"><body style="font-family:Arial,sans-serif;background:#f4f4f4;padding:20px;">
<div style="max-width:600px;margin:0 auto;background:#fff;border-radius:8px;padding:24px;line-height:1.5;">
<h2>Вас пригласили на событие</h2>
<p>Вас пригласили на «%s».</p>
<p><a href="%s" style="display:inline-block;padding:10px 20px;background:#8950fa;color:#fff;text-decoration:none;border-radius:20px;">Открыть приглашение</a></p>
<p style="color:#666;font-size:13px;">Если ссылка не открывается, скопируйте её в браузер: %s</p>
</div></body></html>`, eventTitle, acceptURL, acceptURL)
	return subject, body
}

type noopMailer struct{}

func (noopMailer) SendEventInvitation(context.Context, string, string, string) error { return nil }
