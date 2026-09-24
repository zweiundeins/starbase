package catalog_test

import (
	"io/fs"
	"regexp"
	"sort"
	"strings"
	"testing"

	"starbase/components"
	"starbase/internal/catalog"
)

var fallbackRe = regexp.MustCompile(`var\(\s*(--sb-[a-z0-9-]+)\s*,\s*(#[0-9A-Fa-f]{3,8})\s*\)`)

// A token's hex fallback (what a component looks like on a page without
// Starbase's tokens) is the same in every component: two copies of one
// token that disagree mean one of them drifted.
func TestTokenFallbacksAgree(t *testing.T) {
	cat, err := catalog.Load(components.FS)
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]map[string][]string{} // token → fallback → components
	for _, c := range cat.Components {
		src, _ := fs.ReadFile(cat.FS, c.Script)
		for _, m := range fallbackRe.FindAllStringSubmatch(string(src), -1) {
			tok, hex := m[1], strings.ToUpper(m[2])
			if seen[tok] == nil {
				seen[tok] = map[string][]string{}
			}
			if l := seen[tok][hex]; len(l) == 0 || l[len(l)-1] != c.Slug {
				seen[tok][hex] = append(l, c.Slug)
			}
		}
	}
	for tok, byHex := range seen {
		if len(byHex) < 2 {
			continue
		}
		var parts []string
		for hex, slugs := range byHex {
			parts = append(parts, hex+" in "+strings.Join(slugs, ", "))
		}
		sort.Strings(parts)
		t.Errorf("%s has different fallbacks: %s", tok, strings.Join(parts, "; "))
	}
}
