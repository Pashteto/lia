package templates

import (
	"bytes"
	"context"
	"fmt"
	"html/template"

	"github.com/gateway-fm/scriptorium/clog"

	htmlpkg "gateguard/internal/pkg/notificator/templates/html"
)

const templateName = "organization_invite"

type parseTemplateIn struct {
	log      *clog.CustomLogger
	metadata any
}

// parseTemplate renders the org-invite template (kept for the existing caller).
func parseTemplate(ctx context.Context, in parseTemplateIn) (string, error) {
	return parseNamedTemplate(ctx, templateName, htmlpkg.EmailTemplate, in)
}

// htmlVerificationBody returns the verification template body (indirection keeps
// the html import in one place).
func htmlVerificationBody() string { return htmlpkg.VerificationEmailTemplate }

// parseNamedTemplate parses and executes an arbitrary named html/template body.
func parseNamedTemplate(ctx context.Context, name, body string, in parseTemplateIn) (string, error) {
	ctx = in.log.AddKeysValuesToCtx(ctx, map[string]interface{}{"template_name": name})

	tmpl, err := template.New(name).Parse(body)
	if err != nil {
		in.log.ErrorCtx(ctx, err, "failed to parse template")
		return "", fmt.Errorf("failed to parse %s template: %w", name, err)
	}

	buf := new(bytes.Buffer)
	if err = tmpl.Execute(buf, in.metadata); err != nil {
		in.log.ErrorCtx(ctx, err, "failed to execute template")
		return "", fmt.Errorf("failed to execute %s template: %w", name, err)
	}
	return buf.String(), nil
}
