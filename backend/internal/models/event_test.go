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
