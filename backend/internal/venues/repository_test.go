package venues

import "testing"

func TestSearchPattern_WrapsInWildcards(t *testing.T) {
	if got, want := searchPattern("Дом Радио"), "%Дом Радио%"; got != want {
		t.Fatalf("searchPattern = %q, want %q", got, want)
	}
}

func TestSearchPattern_TrimsSurroundingSpace(t *testing.T) {
	if got, want := searchPattern("  Манеж \n"), "%Манеж%"; got != want {
		t.Fatalf("searchPattern = %q, want %q", got, want)
	}
}

// A user typing "%" means the character, not "match everything". Same for "_",
// which LIKE reads as "any single character".
func TestSearchPattern_EscapesLikeMetacharacters(t *testing.T) {
	tests := []struct {
		name string
		q    string
		want string
	}{
		{"percent", "50%", `%50\%%`},
		{"underscore", "a_b", `%a\_b%`},
		{"backslash", `a\b`, `%a\\b%`},
		{"bare percent", "%", `%\%%`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := searchPattern(tt.q); got != tt.want {
				t.Fatalf("searchPattern(%q) = %q, want %q", tt.q, got, tt.want)
			}
		})
	}
}
