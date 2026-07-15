package service

import (
	"regexp"
	"testing"
)

func Test_newVerificationCode_format(t *testing.T) {
	re := regexp.MustCompile(`^[0-9]{6}$`)
	seen := map[string]int{}
	for i := 0; i < 1000; i++ {
		code := newVerificationCode()
		if !re.MatchString(code) {
			t.Fatalf("code %q is not 6 digits", code)
		}
		seen[code]++
	}
	// Sanity: not all identical (would indicate a broken RNG).
	if len(seen) < 100 {
		t.Fatalf("suspiciously low entropy: only %d distinct codes in 1000", len(seen))
	}
}
