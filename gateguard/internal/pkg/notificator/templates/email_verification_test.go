package templates_test

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/gateway-fm/scriptorium/clog"

	"gateguard/internal/pkg/notificator/templates"
)

func Test_EmailVerification_RendersCode(t *testing.T) {
	log := clog.NewCustomLogger(os.Stdout, clog.LevelDebug, false)
	tmpl := templates.NewEmailVerification(log, "042173")

	body, err := tmpl.GetTemplateAsString(context.Background())
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if !strings.Contains(body, "042173") {
		t.Fatalf("rendered body does not contain the code; got: %s", body)
	}
	if tmpl.TemplateName() != "email_verification" {
		t.Fatalf("unexpected template name %q", tmpl.TemplateName())
	}
	if !strings.HasPrefix(tmpl.Subject(), "Subject:") {
		t.Fatalf("subject must start with 'Subject:'; got %q", tmpl.Subject())
	}
}
