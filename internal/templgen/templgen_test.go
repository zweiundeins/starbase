package templgen

import (
	"io/fs"
	"os"
	"regexp"
	"strconv"
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

// TestRelease: sb.Release is the first released version in the changelog,
// never [Unreleased], and the changelog has one.
func TestRelease(t *testing.T) {
	for in, want := range map[string]string{
		"# Changelog\n\n## [Unreleased]\n\n### Added\n\n## [0.9.1] - 2026-11-17\n\n## [0.9.0] - 2026-11-16\n": "0.9.1",
		"## [Unreleased]\n## [1.10.0]\n":                "1.10.0",
		"## [Unreleased]\n- see [0.9.0]\n### [0.8.0]\n": "",
	} {
		got, err := Release([]byte(in))
		if got != want || (err == nil) != (want != "") {
			t.Errorf("Release(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
	b, err := os.ReadFile("../../CHANGELOG.md")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Release(b); err != nil {
		t.Error(err)
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
		`<sb-dropdown items='[{"value":"a","label":"A"},"-"]'></sb-dropdown>`:                    `@sb.Dropdown(sb.DropdownProps{Items: []sb.DropdownItem{{Value: "a", Label: "A"}, {Divider: true}}})`,
		`<sb-dropdown items='[{"label":"Sort","type":"radio","children":["a"]}]'></sb-dropdown>`: `@sb.Dropdown(sb.DropdownProps{Items: []sb.DropdownItem{{Label: "Sort", Children: []sb.DropdownItem{{Value: "a"}}, Type: sb.DropdownItemTypeRadio}}})`,
		`<sb-tabs selected="0" labels='["A"]'></sb-tabs>`:                                        `@sb.Tabs(sb.TabsProps{Selected: sb.Int(0), Labels: []string{"A"}})`,
		`<sb-meter segments="10" decimals="0"></sb-meter>`:                                       `@sb.Meter(sb.MeterProps{Segments: sb.Int(10)})`,
		`<sb-virtual-scroll offset="0" total="40"></sb-virtual-scroll>`:                          `@sb.VirtualScroll(sb.VirtualScrollProps{Offset: sb.Int(0), Total: sb.Int(40)})`,
		`<sb-tree value="" expanded="a  b"></sb-tree>`:                                           `@sb.Tree(sb.TreeProps{Value: []string{}, Expanded: []string{"a", "b"}})`,
		`<sb-select value="Mars, Venus"></sb-select>`:                                            `@sb.Select(sb.SelectProps{Value: []string{"Mars, Venus"}})`,
		`<sb-select multiple value="Mars, Venus"></sb-select>`:                                   `@sb.Select(sb.SelectProps{Multiple: true, Value: []string{"Mars", "Venus"}})`,
		`<sb-select multiple value='["a,b"]'></sb-select>`:                                       `@sb.Select(sb.SelectProps{Multiple: true, Value: []string{"a,b"}})`,
		`<sb-toast toasts='[{"id":"a","text":"b","duration":0}]'></sb-toast>`:                    `@sb.Toast(sb.ToastProps{Toasts: []sb.ToastMessage{{ID: "a", Text: "b", Duration: sb.Int(0)}}})`,
		`<sb-toast toasts='[{"id":"a","text":"b","href":"/m","label":"Go"}]'></sb-toast>`:        `@sb.Toast(sb.ToastProps{Toasts: []sb.ToastMessage{{ID: "a", Text: "b", Href: "/m", Label: "Go"}}})`,
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
		`<sb-meter segments="2.5"></sb-meter>`,
		`<sb-dropdown items='[{"value":"a","colour":"red"}]'></sb-dropdown>`,
		`<sb-toggle>`,
	} {
		if got, err := m.Templ(bad); err == nil {
			t.Errorf("%s: no error, got %s", bad, got)
		}
	}
}

var (
	hostAttrRe  = regexp.MustCompile(`host\.(?:has|get)Attribute\(['"]([a-z-]+)['"]\)`)
	hostAttrVar = regexp.MustCompile(`host\.getAttribute\([a-zA-Z_]\w*\)`)
	observeRe   = regexp.MustCompile(`\.observe\(host, \{[^}]*attributeFilter: \[([^\]]*)\]`)
	quotedRe    = regexp.MustCompile(`['"]([a-z-]+)['"]`)
	serverDocRe = regexp.MustCompile(`(?i)^server data|from the server|server's word`)
	serverMdRe  = regexp.MustCompile(`(?i)server owns|server data|server's word`)
	backtickRe  = regexp.MustCompile("`([a-z-]+)`")
)

// TestPresencePropsArePointers: a prop whose attribute the component tells
// apart from no attribute, and server data, must tell "" or 0 from no
// attribute in Go too: booleans and flags by their own rules, strings and
// numbers as pointers (serverWord, presence, or a number with a default
// other than 0), lists and JSON as nil or not. It finds them in the
// component's code (host.hasAttribute('x'), host.getAttribute('x'), and
// every attribute a MutationObserver on the host watches when the code reads
// host.getAttribute(name)), in the prop's docs ("Server data", "from the
// server", "the server's word") and in README sentences that say the server
// owns a `prop`.
func TestPresencePropsArePointers(t *testing.T) {
	m := load(t)
	found := 0
	for _, c := range m.Components {
		files, err := m.Catalog.ModuleFiles(c.Component)
		if err != nil {
			t.Fatal(err)
		}
		src := string(files[strings.TrimPrefix(c.Script, c.Slug+"/")])
		why := map[string]string{}
		for _, match := range hostAttrRe.FindAllStringSubmatch(src, -1) {
			why[match[1]] = "the code reads host." + strings.SplitN(match[0], ".", 2)[1]
		}
		if hostAttrVar.MatchString(src) {
			for _, obs := range observeRe.FindAllStringSubmatch(src, -1) {
				for _, q := range quotedRe.FindAllStringSubmatch(obs[1], -1) {
					why[q[1]] = "the code reads it with host.getAttribute(name) in a MutationObserver"
				}
			}
		}
		for _, f := range c.Fields {
			if serverDocRe.MatchString(f.description()) {
				why[f.Prop.Attribute] = "its docs say it is the server's"
			}
		}
		readme, err := fs.ReadFile(m.Catalog.FS, c.Slug+"/README.md")
		if err != nil {
			t.Fatal(err)
		}
		for _, sentence := range regexp.MustCompile(`[.!?]\s|\n\n`).Split(string(readme), -1) {
			if serverMdRe.MatchString(sentence) {
				for _, b := range backtickRe.FindAllStringSubmatch(sentence, -1) {
					why[b[1]] = "its README says the server owns it"
				}
			}
		}
		for attr, reason := range why {
			f, ok := c.Field(attr)
			if !ok {
				continue // aria-*, role: not props
			}
			found++
			switch f.Kind {
			case KindString, KindURL, KindNumber, KindInt:
				t.Errorf("%s %s: %s, but %s is a %s, whose zero value leaves it out: add it to serverWord or presence", c.Tag, attr, reason, f.GoName, f.GoType)
			}
		}
	}
	t.Logf("%d props told apart from no attribute", found)
}

// intCueRe are words in a number prop's docs that say it holds a whole
// number, and intCodecRe a codec that rounds.
var (
	intCueRe   = regexp.MustCompile(`(?i)\b(index|how many|number of|ms|milliseconds|seconds|decimals|digits|characters|items|lines|blocks|cells|points kept|levels|modules)\b`)
	intCodecRe = regexp.MustCompile(`^\s*([a-zA-Z]+): number(?:\.[a-z]+(?:\([^)]*\))?)*?\.(?:step\(1\)|round\b|floor\(\)|ceil\(\))`)
)

// notInt are number props whose docs read like a count but whose value
// isn't one.
var notInt = map[string]string{
	"sb-odometer.value": "the number its digits roll to, with decimals",
}

// TestIntegerProps finds number props that hold whole numbers by their own
// cues, apart from intProps: a codec that rounds (step(1), round, floor(),
// ceil()) or docs that count, index or time in ms. Each must be an int.
func TestIntegerProps(t *testing.T) {
	m := load(t)
	for _, c := range m.Components {
		files, err := m.Catalog.ModuleFiles(c.Component)
		if err != nil {
			t.Fatal(err)
		}
		codecs := map[string]bool{}
		for _, line := range strings.Split(string(files[strings.TrimPrefix(c.Script, c.Slug+"/")]), "\n") {
			if match := intCodecRe.FindStringSubmatch(line); match != nil {
				codecs[match[1]] = true
			}
		}
		for _, f := range c.Fields {
			if f.Prop.Type != "number" || notInt[f.key()] != "" {
				continue
			}
			cue := ""
			if codecs[f.Prop.Name] {
				cue = "its codec rounds"
			} else if w := intCueRe.FindString(f.description()); w != "" {
				cue = "its docs say " + strconv.Quote(w)
			}
			isInt := f.Kind == KindInt || f.Kind == KindIntPtr
			if cue != "" && !isInt {
				t.Errorf("%s %s: %s, but it is a %s: add it to intProps (or notInt, with a reason)", c.Tag, f.Prop.Attribute, cue, f.GoType)
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
