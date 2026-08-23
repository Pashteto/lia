package geocode

import "testing"

func TestBiasFor(t *testing.T) {
	if biasFor("spb").ll != "30.314997,59.938784" {
		t.Fatalf("spb bias wrong: %+v", biasFor("spb"))
	}
	msk := biasFor("msk")
	for _, fallback := range []string{"", "ekb", "МСК"} {
		if biasFor(fallback) != msk {
			t.Fatalf("expected %q to fall back to msk bias", fallback)
		}
	}
}
