package templates

import (
	"context"

	"github.com/gateway-fm/scriptorium/clog"
)

const emailVerificationTemplateName = "email_verification"

// EmailVerification is the transactional email carrying a 6-digit code.
type EmailVerification struct {
	Code string

	log *clog.CustomLogger
}

func NewEmailVerification(log *clog.CustomLogger, code string) *EmailVerification {
	return &EmailVerification{Code: code, log: log}
}

func (t EmailVerification) GetTemplateAsString(ctx context.Context) (string, error) {
	return parseNamedTemplate(ctx, emailVerificationTemplateName, htmlVerificationBody(), parseTemplateIn{
		log:      t.log,
		metadata: t,
	})
}

func (t EmailVerification) TemplateName() string { return emailVerificationTemplateName }

func (t EmailVerification) Subject() string {
	return "Subject: Presence: код подтверждения почты"
}
