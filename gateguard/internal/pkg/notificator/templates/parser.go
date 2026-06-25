package templates

import (
	"bytes"
	"context"
	"fmt"
	"html/template"

	"github.com/gateway-fm/scriptorium/clog"

	"gateguard/internal/pkg/notificator/templates/html"
)

const templateName = "organization_invite"

type parseTemplateIn struct {
	log      *clog.CustomLogger
	metadata any
}

func parseTemplate(ctx context.Context, in parseTemplateIn) (string, error) {
	ctx = in.log.AddKeysValuesToCtx(ctx, map[string]interface{}{
		"template_name": templateName,
	})

	tmpl, err := template.New(templateName).Parse(html.EmailTemplate)
	if err != nil {
		in.log.ErrorCtx(ctx, err, "failed to parse template")
		return "", fmt.Errorf("failed to parse %s template: %w", templateName, err)
	}

	buf := new(bytes.Buffer)
	if err = tmpl.Execute(buf, in.metadata); err != nil {
		in.log.ErrorCtx(ctx, err, "failed to execute template")
		return "", fmt.Errorf("failed to execute %s template: %w", templateName, err)
	}

	return buf.String(), nil
}
