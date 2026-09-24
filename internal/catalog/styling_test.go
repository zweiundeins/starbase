package catalog_test

import (
	"io/fs"
	"path"
	"regexp"
	"strings"
	"testing"

	"starbase/components"
	"starbase/internal/catalog"
)

var (
	stylingPartRe  = regexp.MustCompile(`::part\(([a-z0-9 -]+)\)`)
	stylingTokenRe = regexp.MustCompile(`(var\(\s*)?(--sb-[a-z0-9]+(?:-[a-z0-9]+)*)`)
	sourcePartRe   = regexp.MustCompile(`\bpart="([^"]*)"`)
	partNameRe     = regexp.MustCompile(`[a-z][a-z0-9-]*`)
)

// stylingSection returns the body of a README's "## Styling" section.
func stylingSection(readme string) (string, bool) {
	_, rest, ok := strings.Cut(readme, "\n## Styling\n")
	if !ok {
		return "", false
	}
	if i := strings.Index(rest, "\n## "); i >= 0 {
		rest = rest[:i]
	}
	return rest, true
}

// componentSource concatenates a component's own scripts: every .js file in
// its folder except minified siblings and vendored code.
func componentSource(t *testing.T, slug string) string {
	t.Helper()
	var b strings.Builder
	err := fs.WalkDir(components.FS, slug, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && d.Name() == "vendor" {
			return fs.SkipDir
		}
		if d.IsDir() || path.Ext(p) != ".js" || strings.HasSuffix(p, ".min.js") {
			return nil
		}
		data, err := fs.ReadFile(components.FS, p)
		b.Write(data)
		b.WriteByte('\n')
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return b.String()
}

// TestStylingDocs keeps every README's Styling section honest: it must exist,
// every ::part(x) it names must be a part the component (or a component it
// renders) exposes, and every --sb-… custom property it names or sets must
// appear in that source (a var(--sb-…) value in an example is not checked).
func TestStylingDocs(t *testing.T) {
	cat, err := catalog.Load(components.FS)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range cat.Components {
		readme, err := fs.ReadFile(components.FS, c.Slug+"/README.md")
		if err != nil {
			t.Fatal(err)
		}
		section, ok := stylingSection(string(readme))
		if !ok || strings.TrimSpace(section) == "" {
			t.Errorf("%s: README.md has no ## Styling section (see components/gauge/README.md)", c.Slug)
			continue
		}
		src := componentSource(t, c.Slug)
		for _, u := range cat.Uses(c) {
			src += componentSource(t, u.Slug)
		}
		parts := map[string]bool{}
		for _, m := range sourcePartRe.FindAllStringSubmatch(src, -1) {
			// A static list (part="tab") or an expression (data-attr:part="… ? 'tab selected' : 'tab'").
			for _, p := range partNameRe.FindAllString(m[1], -1) {
				parts[p] = true
			}
		}
		for _, m := range stylingPartRe.FindAllStringSubmatch(section, -1) {
			for _, p := range strings.Fields(m[1]) {
				if !parts[p] {
					t.Errorf("%s: Styling mentions ::part(%s), but no element has part %q", c.Slug, m[1], p)
				}
			}
		}
		seen := map[string]bool{}
		for _, m := range stylingTokenRe.FindAllStringSubmatch(section, -1) {
			tok := m[2]
			if m[1] != "" || seen[tok] {
				continue // var(--sb-x) is a value an example uses, not a claim about the component
			}
			seen[tok] = true
			if !regexp.MustCompile(regexp.QuoteMeta(tok) + `(?:[^a-z0-9-]|$)`).MatchString(src) {
				t.Errorf("%s: Styling mentions %s, which the component's source never uses", c.Slug, tok)
			}
		}
	}
}
