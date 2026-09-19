package models

import (
	"strings"
	"testing"
	"time"
)

func TestEventValidateSignupMode(t *testing.T) {
	base := func() *Event {
		return &Event{Title: "x", StartsAt: time.Now(), Status: EventPublished, SignupMode: "open"}
	}
	if err := base().Validate(); err != nil {
		t.Fatalf("open mode should be valid: %v", err)
	}

	app := base()
	app.SignupMode = "application"
	app.CuratorQuestion = ""
	if err := app.Validate(); err == nil {
		t.Fatal("application mode without curator_question should fail")
	}
	app.CuratorQuestion = "почему вам интересно?"
	if err := app.Validate(); err != nil {
		t.Fatalf("application mode with question should pass: %v", err)
	}

	ext := base()
	ext.SignupMode = "external"
	if err := ext.Validate(); err == nil {
		t.Fatal("external mode without url should fail")
	}
	ext.ExternalRegistrationURL = "https://org.example/signup"
	if err := ext.Validate(); err != nil {
		t.Fatalf("external mode with url should pass: %v", err)
	}

	bad := base()
	bad.SignupMode = "bogus"
	if err := bad.Validate(); err == nil {
		t.Fatal("unknown signup_mode should fail")
	}
}

func TestEventValidate_SignupMessages(t *testing.T) {
	e := &Event{Title: "T", StartsAt: time.Now(), Status: EventPublished, SignupMode: "application"}
	if err := e.Validate(); err == nil || !strings.Contains(err.Error(), "вопрос") {
		t.Fatalf("want curator-question message, got %v", err)
	}
	cap0 := 0
	e2 := &Event{Title: "T", StartsAt: time.Now(), Status: EventPublished, SignupMode: "open", Capacity: &cap0}
	if err := e2.Validate(); err == nil || !strings.Contains(err.Error(), "больше нуля") {
		t.Fatalf("want capacity message, got %v", err)
	}
}

// Attribution: the venues that let us republish their announcements did so on
// the condition that every imported event credits its source with a link. The
// model keeps that link honest — never a label with nothing to click, never a
// scheme a browser would not open.
func TestEventValidate_Source(t *testing.T) {
	base := func() *Event {
		return &Event{Title: "x", StartsAt: time.Now(), Status: EventPublished, SignupMode: "open"}
	}

	none := base()
	if err := none.Validate(); err != nil {
		t.Fatalf("no source at all is fine — organizers post their own events: %v", err)
	}

	ok := base()
	ok.SourceURL = "https://t.me/nefiktiv/512"
	ok.SourceLabel = "Телеграм-канал «Нефиктивное образование»"
	if err := ok.Validate(); err != nil {
		t.Fatalf("url + label should pass: %v", err)
	}

	labelOnly := base()
	labelOnly.SourceLabel = "Телеграм-канал «Нефиктивное образование»"
	if err := labelOnly.Validate(); err == nil {
		t.Fatal("a source credit with no link is not a credit — must fail")
	}

	for _, bad := range []string{"t.me/nefiktiv", "javascript:alert(1)", "ftp://example.org/x", "://broken"} {
		e := base()
		e.SourceURL = bad
		e.SourceLabel = "источник"
		if err := e.Validate(); err == nil {
			t.Fatalf("source_url %q should be rejected", bad)
		}
	}
}

// A link with no label would render as a bare URL, so the host stands in. The
// importer always sets a label; a human editing by hand often will not.
func TestNormalizeSource_FillsLabelFromHost(t *testing.T) {
	e := &Event{SourceURL: "https://eusp.timepad.ru/event/4188233/"}
	e.NormalizeSource()
	if e.SourceLabel != "eusp.timepad.ru" {
		t.Fatalf("label = %q, want the host", e.SourceLabel)
	}

	kept := &Event{SourceURL: "https://t.me/nefiktiv/512", SourceLabel: "Телеграм-канал «Нефиктивное образование»"}
	kept.NormalizeSource()
	if kept.SourceLabel != "Телеграм-канал «Нефиктивное образование»" {
		t.Fatalf("an explicit label must survive, got %q", kept.SourceLabel)
	}

	blank := &Event{}
	blank.NormalizeSource()
	if blank.SourceLabel != "" {
		t.Fatalf("no url → no invented label, got %q", blank.SourceLabel)
	}

	spaced := &Event{SourceURL: "  https://eusp.timepad.ru/event/1/  ", SourceLabel: "  "}
	spaced.NormalizeSource()
	if spaced.SourceURL != "https://eusp.timepad.ru/event/1/" || spaced.SourceLabel != "eusp.timepad.ru" {
		t.Fatalf("whitespace must be trimmed, got url=%q label=%q", spaced.SourceURL, spaced.SourceLabel)
	}
}
