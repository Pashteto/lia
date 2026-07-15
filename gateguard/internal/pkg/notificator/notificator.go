package notificator

import (
	"context"
	"errors"
	"fmt"
	"net/smtp"
	"strings"

	"github.com/gateway-fm/scriptorium/clog"

	"gateguard/internal/pkg/notificator/templates"
)

type SMTPNotificator struct {
	log      *clog.CustomLogger
	addr     string
	from     string
	username string
	password string
}

func NewSMTPNotificator(
	username, password, from, addr, organizationTemplateLink string,
	log *clog.CustomLogger,
) INotificator {
	return &SMTPNotificator{
		log:      log,
		addr:     addr,
		username: username,
		password: password,
		from:     from,
	}
}

func (s *SMTPNotificator) Start(_ *smtp.ServerInfo) (string, []byte, error) {
	return "LOGIN", []byte{}, nil
}

func (s *SMTPNotificator) Next(fromServer []byte, more bool) ([]byte, error) {
	if more {
		switch string(fromServer) {
		case "Username:":
			return []byte(s.username), nil
		case "Password:":
			return []byte(s.password), nil
		default:
			return nil, errors.New("unknown error fromServer")
		}
	}
	return nil, nil
}

func (s *SMTPNotificator) sendTemplate(ctx context.Context, to string, template templates.ITemplate) error {
	toList := []string{to}

	body, err := template.GetTemplateAsString(ctx)
	if err != nil {
		s.log.ErrorCtx(ctx, err, "failed to parse email template")
		return fmt.Errorf("failed to parse email template: %w", err)
	}

	headers := []string{
		"MIME-version: 1.0;",
		"Content-Type: text/html; charset=\"UTF-8\";",
		fmt.Sprintf("From: %s", s.from),
		fmt.Sprintf("To: %s", to),
		template.Subject(),
	}

	headersJoined := strings.Join(headers, "\r\n")

	msg := []byte(headersJoined + "\r\n\r\n" + body)

	if err = smtp.SendMail(s.addr, s, s.from, toList, msg); err != nil {
		s.log.ErrorCtx(ctx, err, "failed to send email")
		return fmt.Errorf("failed to send email: %w", err)
	}

	return nil
}

// SendEmailVerification emails a 6-digit verification code.
func (s *SMTPNotificator) SendEmailVerification(ctx context.Context, to, code string) error {
	tmpl := templates.NewEmailVerification(s.log, code)
	return s.sendTemplate(ctx, to, tmpl)
}
