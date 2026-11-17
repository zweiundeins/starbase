package templgen

import (
	"regexp"
	"strings"
	"testing"

	templparser "github.com/a-h/templ/parser/v2"
)

func load(t *testing.T) *Model {
	t.Helper()
	m, err := Load("../..")
	if err != nil {
		t.Fatal(err)
	}
	return m
}

// TestWrappersUpToDate fails when a manifest (or a component file, which
// changes its pinned version) changed without regenerating the wrappers.
func TestWrappersUpToDate(t *testing.T) {
	stale, err := Refresh("../..", false)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range stale {
		t.Errorf("%s is stale: run go tool task templ-wrappers", p)
	}
}

// TestTemplSnippetsParse: the templ tab of every component page, comments
// included, is templ that templ's own parser reads.
func TestTemplSnippetsParse(t *testing.T) {
	m := load(t)
	for _, c := range m.Components {
		snippet, err := m.Install(c, c.InstallMarkup(), "https://example.com")
		if err != nil {
			t.Errorf("%s: %v", c.Slug, err)
			continue
		}
		src := "package x\n\nimport \"github.com/zweiundeins/starbase/templ/sb\"\n\ntempl Usage() {\n\t" + strings.ReplaceAll(snippet, "\n", "\n\t") + "\n}\n"
		f, err := templparser.ParseString(src)
		if err != nil {
			t.Errorf("%s: %v\n%s", c.Slug, err, src)
			continue
		}
		if n := strings.Count(snippet, "@sb."); n == 0 || !strings.Contains(src, "@sb."+c.GoName+"(") {
			t.Errorf("%s: no call of sb.%s in\n%s", c.Slug, c.GoName, snippet)
		}
		comments := 0
		for _, n := range f.Nodes {
			if tt, ok := n.(*templparser.HTMLTemplate); ok {
				for _, ch := range tt.Children {
					if _, ok := ch.(*templparser.GoComment); ok {
						comments++
					}
				}
			}
		}
		if comments != 3 || !strings.Contains(snippet, "@sb.Imports(sb.ImportsProps{Base: \"https://example.com\", Components: ") {
			t.Errorf("%s: %d Go comments, want 3, in\n%s", c.Slug, comments, snippet)
		}
	}
}

func TestTemplSnippet(t *testing.T) {
	m := load(t)
	for markup, want := range map[string]string{
		`<sb-toggle label="Notifications"></sb-toggle>`: `@sb.Toggle(sb.ToggleProps{Label: "Notifications"})`,
		`<sb-slider label="Volume" value="60" unit="%" show-value="false" data-on:sb-change="@post('/v')"></sb-slider>`: `@sb.Slider(sb.SliderProps{
	Label:     "Volume",
	Value:     sb.Num(60),
	Unit:      "%",
	ShowValue: sb.Off,
	Attrs:     templ.Attributes{"data-on:sb-change": "@post('/v')"},
})`,
		"<sb-modal data-ref:_confirm heading=\"Delete?\">\n  Gone {for good}.\n  <button slot=\"footer\" data-sb-close>Cancel</button>\n</sb-modal>": `@sb.Modal(sb.ModalProps{
	Heading: "Delete?",
	Attrs:   templ.Attributes{"data-ref:_confirm": true},
}) {
	{ "Gone {for good}." }
	<button slot="footer" data-sb-close>Cancel</button>
}`,
		`<sb-dropdown items='[{"value":"a","label":"A"},"-"]'></sb-dropdown>`: `@sb.Dropdown(sb.DropdownProps{Items: []sb.DropdownItem{{Value: "a", Label: "A"}, {Divider: true}}})`,
	} {
		got, err := m.Templ(markup)
		if err != nil {
			t.Fatal(err)
		}
		if got != want {
			t.Errorf("%s:\n got %s\nwant %s", markup, got, want)
		}
	}
	for _, bad := range []string{
		`<sb-toggle size="xl"></sb-toggle>`,
		`<sb-meter max="lots"></sb-meter>`,
		`<sb-dropdown items='[{"value":"a","colour":"red"}]'></sb-dropdown>`,
		`<sb-toggle>`,
	} {
		if got, err := m.Templ(bad); err == nil {
			t.Errorf("%s: no error, got %s", bad, got)
		}
	}
}

var hostAttrRe = regexp.MustCompile(`host\.(?:has|get)Attribute\('([a-z-]+)'\)`)

// TestPresencePropsArePointers: a component that looks at whether its
// attribute is there (host.hasAttribute, host.getAttribute) tells "" or 0
// from no attribute, so the wrapper's field must too: booleans and flags by
// their own rules, strings and numbers as pointers (serverWord, presence,
// or a number with a default other than 0), JSON as nil or not.
func TestPresencePropsArePointers(t *testing.T) {
	m := load(t)
	for _, c := range m.Components {
		src, err := m.Catalog.ModuleFiles(c.Component)
		if err != nil {
			t.Fatal(err)
		}
		for _, match := range hostAttrRe.FindAllStringSubmatch(string(src[strings.TrimPrefix(c.Script, c.Slug+"/")]), -1) {
			f, ok := c.Field(match[1])
			if !ok {
				continue // aria-*, role: not props
			}
			switch f.Kind {
			case KindString, KindURL, KindNumber:
				t.Errorf("%s reads %s by presence, but %s is a %s, whose zero value leaves it out: add it to serverWord or presence", c.Tag, match[1], f.GoName, f.GoType)
			}
		}
	}
}

func TestGoNames(t *testing.T) {
	for in, want := range map[string]string{"QR Code": "QRCode", "ECharts": "ECharts", "Table of Contents": "TableOfContents", "bottom-start": "BottomStart", "M": "M", "deep-space": "DeepSpace"} {
		if got := goName(in); got != want {
			t.Errorf("goName(%q) = %q, want %q", in, got, want)
		}
	}
	for in, want := range map[string]string{"rowKey": "RowKey", "resetMs": "ResetMS", "ariaLabel": "AriaLabel", "for": "For", "ecc": "ECC"} {
		if got := fieldName(in); got != want {
			t.Errorf("fieldName(%q) = %q, want %q", in, got, want)
		}
	}
}
