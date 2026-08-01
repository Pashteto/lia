package venues

import (
	"strings"
	"unicode/utf8"
)

// Transliteration between Latin and Cyrillic for venue search. «Эрарта» is
// widely written "Erarta", and the reverse happens too, so a query in one
// alphabet has to be able to reach a name stored in the other. unaccent() and
// trigram similarity cannot bridge alphabets — they operate on characters that
// have no relationship across scripts.
//
// The mappings below are deliberately plain: they reproduce the common
// romanization rather than a reversible standard. Transliteration is lossy in
// both directions (Russian "ье" and "е" both romanize toward "e"; a bare Latin
// "c" has no single Cyrillic answer), so a variant is expected to land close
// rather than exact. The trigram branch in Search absorbs the remainder.

// cyrillicToLatinMap romanizes one Cyrillic rune at a time. Soft and hard signs
// drop out, matching how names are written in practice ("Тверская" → "tverskaya").
var cyrillicToLatinMap = map[rune]string{
	'а': "a", 'б': "b", 'в': "v", 'г': "g", 'д': "d", 'е': "e", 'ё': "e",
	'ж': "zh", 'з': "z", 'и': "i", 'й': "y", 'к': "k", 'л': "l", 'м': "m",
	'н': "n", 'о': "o", 'п': "p", 'р': "r", 'с': "s", 'т': "t", 'у': "u",
	'ф': "f", 'х': "kh", 'ц': "ts", 'ч': "ch", 'ш': "sh", 'щ': "shch",
	'ъ': "", 'ы': "y", 'ь': "", 'э': "e", 'ю': "yu", 'я': "ya",
}

// latinToCyrillicDigraphs must be tried before single letters, longest first,
// or "shch" degrades into "сх ч" and "zh" into "зх".
var latinToCyrillicDigraphs = []digraph{
	{"shch", "щ"}, {"sch", "щ"},
	{"zh", "ж"}, {"kh", "х"}, {"ts", "ц"}, {"ch", "ч"}, {"sh", "ш"},
	{"yo", "ё"}, {"yu", "ю"}, {"ya", "я"}, {"ye", "е"},
}

var latinToCyrillicMap = map[rune]string{
	'a': "а", 'b': "б", 'c': "к", 'd': "д", 'e': "е", 'f': "ф", 'g': "г",
	'h': "х", 'i': "и", 'j': "дж", 'k': "к", 'l': "л", 'm': "м", 'n': "н",
	'o': "о", 'p': "п", 'q': "к", 'r': "р", 's': "с", 't': "т", 'u': "у",
	'v': "в", 'w': "в", 'x': "кс", 'z': "з",
	// 'y' is handled positionally in latinToCyrillic (й after a vowel, else ы).
}

// cyrillicToLatin romanizes every Cyrillic rune and passes everything else
// through, so "Клуб «Космонавт»" keeps its punctuation and spacing.
func cyrillicToLatin(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range strings.ToLower(s) {
		if latin, ok := cyrillicToLatinMap[r]; ok {
			b.WriteString(latin)
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

// latinToCyrillic is the direction that matters most in practice: a user types
// "Erarta" and the venue is stored as «Эрарта».
//
// Word-initial "e" becomes "э" rather than "е". That single positional rule is
// what makes the common cases land exactly — "Erarta" → "эрарта", "Ermitazh" →
// "эрмитаж" — because Russian spells a word-initial /e/ with э and unaccent()
// does not fold э to е the way it folds ё.
func latinToCyrillic(s string) string {
	lower := strings.ToLower(s)
	out := make([]rune, 0, len(lower))
	atWordStart := true
	for i := 0; i < len(lower); {
		if matched := matchDigraph(lower[i:]); matched.latin != "" {
			out = append(out, []rune(matched.cyr)...)
			i += len(matched.latin)
			atWordStart = false
			continue
		}
		r, size := utf8.DecodeRuneInString(lower[i:])
		i += size
		switch {
		case r == 'e' && atWordStart:
			out = append(out, 'э')
			atWordStart = false
		case r == 'y':
			// A standalone "y" is й after a vowel ("Sergey" → сергей) and ы
			// otherwise ("Krasny" → красны). The digraph table has already
			// consumed ya/yu/yo/ye by this point.
			if endsWithCyrillicVowel(out) {
				out = append(out, 'й')
			} else {
				out = append(out, 'ы')
			}
			atWordStart = false
		default:
			if cyr, ok := latinToCyrillicMap[r]; ok {
				out = append(out, []rune(cyr)...)
				atWordStart = false
			} else {
				out = append(out, r)
				atWordStart = !isLatinLetter(r)
			}
		}
	}
	return string(out)
}

const cyrillicVowels = "аеёиоуыэюя"

func endsWithCyrillicVowel(out []rune) bool {
	if len(out) == 0 {
		return false
	}
	return strings.ContainsRune(cyrillicVowels, out[len(out)-1])
}

type digraph struct {
	latin string
	cyr   string
}

func matchDigraph(s string) digraph {
	for _, d := range latinToCyrillicDigraphs {
		if strings.HasPrefix(s, d.latin) {
			return d
		}
	}
	return digraph{}
}

func isLatinLetter(r rune) bool { return r >= 'a' && r <= 'z' }

// searchVariants returns the query plus its transliterations into the other
// alphabet, deduplicated. A query with nothing to transliterate (digits, or a
// name already written in the target script) yields just itself, so callers do
// not pay for redundant SQL branches.
func searchVariants(q string) []string {
	out := []string{q}
	seen := map[string]bool{strings.ToLower(q): true}
	for _, v := range []string{latinToCyrillic(q), cyrillicToLatin(q)} {
		if v == "" || seen[v] {
			continue
		}
		seen[v] = true
		out = append(out, v)
	}
	return out
}
