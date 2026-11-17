package sb

import (
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/a-h/templ"
)

// Merge returns one set of attributes holding all of attrs; for a name in
// more than one, the later wins (names compare without case, as Attrs do
// against the props). Classes and styles are not joined: a later "class"
// replaces an earlier one.
//
//	Attrs: sb.Merge(base, templ.Attributes{"data-on:sb-change": "@post('/cmd/x')"})
func Merge(attrs ...templ.Attributes) templ.Attributes {
	out := templ.Attributes{}
	for _, a := range attrs {
		for k, v := range a {
			for old := range out {
				if old != k && strings.EqualFold(old, k) {
					delete(out, old)
				}
			}
			out[k] = v
		}
	}
	return out
}

// TabSlot is the slot sb-tabs gives the panel of labels[i]: the label in
// lower case, every run of anything but letters and digits a hyphen, no
// hyphen at either end; a slot an earlier label already has gets "-i"
// appended (until it is free). A label without letters or digits gives "":
// its panel shows the content that has no slot attribute. It returns "" for
// an i outside labels too.
//
//	@sb.Tabs(sb.TabsProps{Labels: labels}) {
//		for i := range labels {
//			<div slot={ sb.TabSlot(labels, i) }>…</div>
//		}
//	}
func TabSlot(labels []string, i int) string {
	if i < 0 || i >= len(labels) {
		return ""
	}
	seen := map[string]bool{}
	slot := ""
	for j := 0; j <= i; j++ {
		slot = TabSlug(labels[j])
		for seen[slot] {
			slot += "-" + strconv.Itoa(j)
		}
		seen[slot] = true
	}
	return slot
}

// TabSlug is the slug sb-tabs makes of one label, without TabSlot's
// de-duplication: for a panel that knows its own label only. It is the
// panel's slot as long as no earlier label has the same slug. sb-tabs'
// slug(): String(s).toLowerCase(), then
// /[^\p{L}\p{N}]+/gu → "-" and one hyphen off each end.
func TabSlug(s string) string {
	var b strings.Builder
	hyphen := false
	for i, r := range s {
		low := jsLower(s, i, r)
		for _, c := range low {
			if unicode.IsLetter(c) || unicode.IsNumber(c) {
				b.WriteRune(c)
				hyphen = false
			} else if !hyphen {
				b.WriteByte('-')
				hyphen = true
			}
		}
	}
	out := strings.TrimPrefix(b.String(), "-")
	return strings.TrimSuffix(out, "-")
}

// jsLower is JavaScript's toLowerCase for the rune r at byte i of s, where
// it differs from Go's: İ becomes i and a combining dot, and a capital sigma
// that ends a word becomes ς.
func jsLower(s string, i int, r rune) string {
	switch r {
	case 'İ':
		return "i̇"
	case 'Σ':
		if finalSigma(s, i) {
			return "ς"
		}
		return "σ"
	}
	return string(unicode.ToLower(r))
}

// finalSigma is Unicode's Final_Sigma condition for the sigma at byte i: a
// cased letter before it and none after it, skipping case-ignorable runes.
func finalSigma(s string, i int) bool {
	before := false
	for j := i; j > 0; {
		r, n := utf8.DecodeLastRuneInString(s[:j])
		j -= n
		if caseIgnorable(r) {
			continue
		}
		before = cased(r)
		break
	}
	if !before {
		return false
	}
	for j := i + len("Σ"); j < len(s); {
		r, n := utf8.DecodeRuneInString(s[j:])
		j += n
		if caseIgnorable(r) {
			continue
		}
		return !cased(r)
	}
	return true
}

func cased(r rune) bool {
	return unicode.IsUpper(r) || unicode.IsLower(r) || unicode.IsTitle(r) ||
		unicode.Is(unicode.Other_Lowercase, r) || unicode.Is(unicode.Other_Uppercase, r)
}

func caseIgnorable(r rune) bool {
	switch r {
	case '\'', '.', ':', '^', '`', '·', '·', '״', '‘', '’', '․', '‧', '︓', '﹒', '﹕', '＇', '．', '：':
		return true
	}
	return unicode.In(r, unicode.Mn, unicode.Me, unicode.Cf, unicode.Lm, unicode.Sk)
}
