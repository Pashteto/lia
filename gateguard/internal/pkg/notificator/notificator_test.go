package notificator

import (
	"net/mail"
	"testing"
)

func Test_fromHeader_NamesTheSender(t *testing.T) {
	h := fromHeader("info@tarski.ru")
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
