package venues

import (
	"reflect"
	"testing"
)

// The venues these cases target are real rows in prod, and the Latin spellings
// are the ones people actually type.
func TestLatinToCyrillic_LandsExactlyOnRealVenueNames(t *testing.T) {
	tests := []struct {
		q    string
		want string
	}{
		{"Erarta", "эрарта"},       // «Эрарта»  — word-initial e → э
		{"Ermitazh", "эрмитаж"},    // «Эрмитаж»
		{"Garazh", "гараж"},        // «Гараж»
		{"Artmuza", "артмуза"},     // «Артмуза»
		{"Kosmonavt", "космонавт"}, // Клуб «Космонавт»
		{"Manezh", "манеж"},        // ЦВЗ «Манеж»
	}
	for _, tt := range tests {
		t.Run(tt.q, func(t *testing.T) {
			if got := latinToCyrillic(tt.q); got != tt.want {
				t.Fatalf("latinToCyrillic(%q) = %q, want %q", tt.q, got, tt.want)
			}
		})
	}
}

// Digraphs have to win over their leading single letter, or "zh" becomes "зх".
func TestLatinToCyrillic_PrefersDigraphsOverSingleLetters(t *testing.T) {
	tests := []struct {
		q    string
		want string
	}{
		{"zh", "ж"},
		{"sh", "ш"},
		{"shch", "щ"},
		{"ch", "ч"},
		{"kh", "х"},
		{"ts", "ц"},
		{"ya", "я"},
		{"yu", "ю"},
	}
	for _, tt := range tests {
		t.Run(tt.q, func(t *testing.T) {
			if got := latinToCyrillic(tt.q); got != tt.want {
				t.Fatalf("latinToCyrillic(%q) = %q, want %q", tt.q, got, tt.want)
			}
		})
	}
}

// Only a word-initial "e" is э; inside a word it stays е.
func TestLatinToCyrillic_InitialEOnly(t *testing.T) {
	if got, want := latinToCyrillic("Neva"), "нева"; got != want {
		t.Fatalf("latinToCyrillic(\"Neva\") = %q, want %q", got, want)
	}
	if got, want := latinToCyrillic("Dom Eleny"), "дом элены"; got != want {
		t.Fatalf("latinToCyrillic(\"Dom Eleny\") = %q, want %q", got, want)
	}
}

// A standalone "y" is й after a vowel and ы otherwise.
func TestLatinToCyrillic_PositionalY(t *testing.T) {
	tests := []struct {
		q    string
		want string
	}{
		{"Sergey", "сергей"}, // after a vowel
		{"Krasny", "красны"}, // after a consonant
		// Romanization drops the soft sign, so «Большой» cannot be rebuilt
		// exactly — "болшой" is one character short. That is the lossiness the
		// trigram branch in Search exists to absorb.
		{"Bolshoy", "болшой"},
		{"Yar", "яр"}, // ya digraph still wins at word start
		{"Novy Arbat", "новы арбат"},
	}
	for _, tt := range tests {
		t.Run(tt.q, func(t *testing.T) {
			if got := latinToCyrillic(tt.q); got != tt.want {
				t.Fatalf("latinToCyrillic(%q) = %q, want %q", tt.q, got, tt.want)
			}
		})
	}
}

func TestLatinToCyrillic_PassesThroughNonLetters(t *testing.T) {
	if got, want := latinToCyrillic("Loft 812"), "лофт 812"; got != want {
		t.Fatalf("latinToCyrillic = %q, want %q", got, want)
	}
}

func TestCyrillicToLatin_Romanizes(t *testing.T) {
	tests := []struct {
		q    string
		want string
	}{
		{"Эрарта", "erarta"},
		{"Ноодоме", "noodome"},
		{"Гараж", "garazh"},
		{"Тверская", "tverskaya"}, // soft sign drops out
		{"Щукин", "shchukin"},
	}
	for _, tt := range tests {
		t.Run(tt.q, func(t *testing.T) {
			if got := cyrillicToLatin(tt.q); got != tt.want {
				t.Fatalf("cyrillicToLatin(%q) = %q, want %q", tt.q, got, tt.want)
			}
		})
	}
}

func TestSearchVariants_AddsTheOtherAlphabet(t *testing.T) {
	// Romanizing an already-Latin query just lowercases it, which matching is
	// insensitive to anyway, so it collapses into the original.
	got := searchVariants("Erarta")
	if want := []string{"Erarta", "эрарта"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("searchVariants(\"Erarta\") = %#v, want %#v", got, want)
	}

	got = searchVariants("Эрарта")
	if want := []string{"Эрарта", "erarta"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("searchVariants(\"Эрарта\") = %#v, want %#v", got, want)
	}
}

// A query with no letters has nothing to transliterate, so Search should not
// pay for extra SQL branches.
func TestSearchVariants_DigitsOnlyYieldsOneVariant(t *testing.T) {
	if got := searchVariants("812"); !reflect.DeepEqual(got, []string{"812"}) {
		t.Fatalf("searchVariants(\"812\") = %#v, want one variant", got)
	}
}

func TestSearchVariants_Deduplicates(t *testing.T) {
	for _, q := range []string{"Erarta", "Эрарта", "50%", "Дом Радио"} {
		seen := map[string]bool{}
		for _, v := range searchVariants(q) {
			if seen[v] {
				t.Fatalf("searchVariants(%q) repeated %q", q, v)
			}
			seen[v] = true
		}
	}
}
