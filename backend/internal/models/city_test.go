package models

import "testing"

func TestValidCity(t *testing.T) {
	for _, ok := range []string{"msk", "spb"} {
		if !ValidCity(ok) {
			t.Errorf("ValidCity(%q) = false, want true", ok)
		}
	}
	for _, bad := range []string{"", "МСК", "SPB", "ekb", "moscow"} {
		if ValidCity(bad) {
			t.Errorf("ValidCity(%q) = true, want false", bad)
		}
	}
}
