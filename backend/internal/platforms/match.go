// Package platforms is the trusted external-registration platform whitelist:
// URL normalization, domain-suffix matching, and the trusted_platforms store.
// Spec: docs/superpowers/specs/2026-08-12-external-registration-design.md.
package platforms

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"

	"golang.org/x/net/idna"
)

// ErrBadURL marks an external-registration URL that fails hard validation
// (non-https, userinfo, IP host, unparseable). Everything else — including an
// unknown domain — is NOT an error; unknown domains take the moderation path.
var ErrBadURL = errors.New("bad external registration url")

// NormalizeHost parses a raw URL and returns its lowercased punycode host.
func NormalizeHost(rawURL string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil || u.Scheme != "https" || u.Host == "" {
		return "", fmt.Errorf("%w: %q", ErrBadURL, rawURL)
	}
	if u.User != nil {
		return "", fmt.Errorf("%w: userinfo in %q", ErrBadURL, rawURL)
	}
	host := strings.ToLower(u.Hostname())
	if net.ParseIP(host) != nil {
		return "", fmt.Errorf("%w: ip host in %q", ErrBadURL, rawURL)
	}
	ascii, err := idna.Lookup.ToASCII(host)
	if err != nil {
		return "", fmt.Errorf("%w: idna %q", ErrBadURL, rawURL)
	}
	return ascii, nil
}

// MatchesSuffix reports whether host equals suffix or ends with "."+suffix
// (dot-boundary check defeats "timepad.ru.evil.com").
func MatchesSuffix(host, suffix string) bool {
	if host == suffix {
		return true
	}
	return strings.HasSuffix(host, "."+suffix)
}
