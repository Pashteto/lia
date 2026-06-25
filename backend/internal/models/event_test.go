package models

import (
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
