package platforms

import (
	"errors"
	"testing"
)

func TestNormalizeHost(t *testing.T) {
	cases := []struct {
		name, raw, want string
		wantErr         bool
	}{
		{"plain https", "https://timepad.ru/event/123", "timepad.ru", false},
		{"subdomain", "https://org.timepad.ru/event/123", "org.timepad.ru", false},
		{"uppercase host", "https://TimePad.RU/e", "timepad.ru", false},
		{"unicode idn to punycode", "https://культура.рф/afisha", "xn--80atdujec4e.xn--p1ai", false},
		{"http rejected", "http://timepad.ru/e", "", true},
		{"userinfo rejected", "https://user@timepad.ru/e", "", true},
		{"ipv4 rejected", "https://1.2.3.4/e", "", true},
		{"ipv6 rejected", "https://[2001:db8::1]/e", "", true},
		{"garbage rejected", "not a url", "", true},
		{"empty rejected", "", "", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := NormalizeHost(c.raw)
			if c.wantErr {
				if !errors.Is(err, ErrBadURL) {
					t.Fatalf("want ErrBadURL, got %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != c.want {
				t.Fatalf("got %q, want %q", got, c.want)
			}
		})
	}
}

func TestMatchesSuffix(t *testing.T) {
	cases := []struct {
		name, host, suffix string
		want               bool
	}{
		{"exact", "timepad.ru", "timepad.ru", true},
		{"subdomain", "org.timepad.ru", "timepad.ru", true},
		{"city subdomain", "msk.kassir.ru", "kassir.ru", true},
		{"bypass attempt", "timepad.ru.evil.com", "timepad.ru", false},
		{"partial label", "evilTimepad.ru", "timepad.ru", false},
		{"unrelated", "evil.com", "timepad.ru", false},
		{"suffix longer than host", "ru", "timepad.ru", false},
		{"deep gov path host", "afisha.yandex.ru", "afisha.yandex.ru", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := MatchesSuffix(c.host, c.suffix); got != c.want {
				t.Fatalf("MatchesSuffix(%q,%q) = %v, want %v", c.host, c.suffix, got, c.want)
			}
		})
	}
}
