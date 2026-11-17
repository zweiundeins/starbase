package sb

import (
	"context"
	"encoding/json"
	"go/ast"
	"go/parser"
	gotoken "go/token"
	"io"
	"io/fs"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"unicode"

	"github.com/a-h/templ"
)

func TestZeroPropsRenderTheBareElement(t *testing.T) {
	for _, e := range Entries {
		if got, want := render(t, e.Render(e.New())), "<"+e.Tag+"></"+e.Tag+">"; got != want {
			t.Errorf("%s with zero props = %s, want %s", e.Tag, got, want)
		}
	}
}

func TestBooleanByPresence(t *testing.T) {
	got := render(t, Toggle(ToggleProps{Checked: true, Disabled: false, Confirm: true}))
	if want := `<sb-toggle checked confirm></sb-toggle>`; got != want {
		t.Errorf("got %s, want %s", got, want)
	}
	if strings.Contains(got, "false") {
		t.Error("a false boolean was written")
	}
}

func TestFlag(t *testing.T) {
	for _, tc := range []struct {
		flag Flag
		want string
	}{
		{Default, `<sb-meter></sb-meter>`},
		{On, `<sb-meter show-value="true"></sb-meter>`},
		{Off, `<sb-meter show-value="false"></sb-meter>`},
	} {
		if got := render(t, Meter(MeterProps{ShowValue: tc.flag})); got != tc.want {
			t.Errorf("ShowValue %v: got %s, want %s", tc.flag, got, tc.want)
		}
	}
	var b strings.Builder
	if err := Meter(MeterProps{ShowValue: Flag(9)}).Render(context.Background(), &b); err == nil {
		t.Error("Flag(9) rendered")
	}
	for _, f := range []Flag{Default, On, Off} {
		j, _ := json.Marshal(struct {
			F Flag `json:"f,omitempty"`
		}{f})
		var back struct{ F Flag }
		if err := json.Unmarshal(j, &back); err != nil || back.F != f {
			t.Errorf("Flag %v through JSON %s = %v, %v", f, j, back.F, err)
		}
	}
	if string(must(json.Marshal(Column{Key: "a"}))) != `{"key":"a"}` {
		t.Error("a Default Flag in a struct is not left out")
	}
}

func TestEnum(t *testing.T) {
	if got, want := render(t, Toggle(ToggleProps{Size: ToggleSizeLg})), `<sb-toggle size="lg"></sb-toggle>`; got != want {
		t.Errorf("got %s, want %s", got, want)
	}
	var b strings.Builder
	err := Toggle(ToggleProps{Size: "xl"}).Render(context.Background(), &b)
	if err == nil || !strings.Contains(err.Error(), `size="xl" is none of sm, md, lg`) {
		t.Errorf("an unknown value: %v", err)
	}
	if b.Len() != 0 {
		t.Errorf("wrote %q before failing", b.String())
	}
}

func TestNumbers(t *testing.T) {
	for _, tc := range []struct {
		p    MeterProps
		want string
	}{
		{MeterProps{}, `<sb-meter></sb-meter>`},
		{MeterProps{Value: 0, Min: 0}, `<sb-meter></sb-meter>`}, // 0 is the default
		{MeterProps{Value: 72.5, Min: -10}, `<sb-meter value="72.5" min="-10"></sb-meter>`},
		{MeterProps{Max: Num(0)}, `<sb-meter max="0"></sb-meter>`}, // set, so written: the default is 100
		{MeterProps{Max: Num(1e21), Segments: Int(12), Decimals: 2}, `<sb-meter max="1000000000000000000000" segments="12" decimals="2"></sb-meter>`},
	} {
		if got := render(t, Meter(tc.p)); got != tc.want {
			t.Errorf("got %s, want %s", got, tc.want)
		}
	}
	// The server's word: nil leaves it out, Num(0) sends 0.
	if got := render(t, Slider(SliderProps{})); got != `<sb-slider></sb-slider>` {
		t.Errorf("got %s", got)
	}
	if got := render(t, Slider(SliderProps{Value: Num(0)})); got != `<sb-slider value="0"></sb-slider>` {
		t.Errorf("got %s", got)
	}
}

func TestStringPointers(t *testing.T) {
	if got := render(t, Input(InputProps{Value: Str("")})); got != `<sb-input value=""></sb-input>` {
		t.Errorf("Str(\"\"): got %s", got)
	}
	if got := render(t, Input(InputProps{})); got != `<sb-input></sb-input>` {
		t.Errorf("nil: got %s", got)
	}
	if got := render(t, Starfield(StarfieldProps{Label: Str("")})); got != `<sb-starfield label=""></sb-starfield>` {
		t.Errorf("a decorative starfield: got %s", got)
	}
}

const nasty = `Say "hi" <b>&amp; it's 😀 é {x}`

func TestEscaping(t *testing.T) {
	out := render(t, Toggle(ToggleProps{Label: nasty, ID: `a"b`, Class: "c <d>"}))
	if strings.Contains(out, "<b>") || strings.Count(out, `"`) != 6 {
		t.Fatalf("not escaped: %s", out)
	}
	h := host(t, out, "sb-toggle")
	for name, want := range map[string]string{"label": nasty, "id": `a"b`, "class": "c <d>"} {
		if a, _ := h.attr(name); a.value != want {
			t.Errorf("%s = %q, want %q", name, a.value, want)
		}
	}
	// JSON: escaped for HTML, and the same value after a round trip.
	opts := []Option{{Value: nasty, Label: "</script><script>alert(1)</script>"}, {Value: "plain"}}
	out = render(t, Select(SelectProps{Options: opts}))
	if strings.Contains(out, "<script>") || strings.Contains(out, "</script>") {
		t.Fatalf("not escaped: %s", out)
	}
	a, _ := host(t, out, "sb-select").attr("options")
	var back []Option
	if err := json.Unmarshal([]byte(a.value), &back); err != nil || !reflect.DeepEqual(back, opts) {
		t.Errorf("options %s came back as %#v (%v)", a.value, back, err)
	}
	if !strings.Contains(a.value, `"plain"`) {
		t.Errorf("an Option with only a value is not a string: %s", a.value)
	}
}

func TestJSONNilAndEmpty(t *testing.T) {
	if got := render(t, CheckboxGroup(CheckboxGroupProps{})); got != `<sb-checkbox-group></sb-checkbox-group>` {
		t.Errorf("nil: got %s", got)
	}
	// An empty list is a value: the server clears the selection.
	if got := render(t, CheckboxGroup(CheckboxGroupProps{Value: []string{}})); got != `<sb-checkbox-group value="[]"></sb-checkbox-group>` {
		t.Errorf("empty: got %s", got)
	}
	if got := render(t, Range(RangeProps{Value: &RangeValue{Start: 0, End: 60}})); got != `<sb-range value="{&#34;start&#34;:0,&#34;end&#34;:60}"></sb-range>` {
		t.Errorf("range: got %s", got)
	}
	var b strings.Builder
	err := ECharts(EChartsProps{Option: func() {}}).Render(context.Background(), &b)
	if err == nil || !strings.Contains(err.Error(), "option") {
		t.Errorf("an option that is no JSON: %v", err)
	}
}

func TestAttrs(t *testing.T) {
	out := render(t, Toggle(ToggleProps{
		ID:    "warp",
		Label: "Warp",
		Size:  ToggleSizeSm,
		Attrs: templ.Attributes{
			"data-on:sb-change":  "@post('/cmd/warp')",
			"data-bind:warp":     true,
			"data-preserve-attr": "checked",
			"aria-describedby":   "hint",
			"LABEL":              "Attrs win",
			"size":               false, // removes the field's attribute
			"hidden":             false,
		},
	}))
	want := `<sb-toggle id="warp" LABEL="Attrs win" aria-describedby="hint" data-bind:warp data-on:sb-change="@post(&#39;/cmd/warp&#39;)" data-preserve-attr="checked"></sb-toggle>`
	if out != want {
		t.Errorf("got  %s\nwant %s", out, want)
	}
	// Datastar's attributes are not URLs or event handlers to templ, but
	// these are: templ's rules apply to Attrs (a templ.SafeURL passes).
	out = render(t, Card(CardProps{Attrs: templ.Attributes{"data-attr:src": "$img", "data-url": "/x", "onclick": "alert(1)"}}))
	if strings.Contains(out, `data-attr:src="about:invalid`) || strings.Contains(out, `data-url="about:invalid`) {
		t.Errorf("relative values sanitized: %s", out)
	}
	if !strings.Contains(out, `onclick="`+templ.FailedSanitizationJS) {
		t.Errorf("an onclick string passed: %s", out)
	}
}

func TestURLs(t *testing.T) {
	for _, u := range []string{"/fleet", "https://example.com/a?b=1&c=2#d", "mailto:crew@example.com", "tel:+41", "#top", "?page=2", "relative/path", "/a:b"} {
		out := render(t, Button(ButtonProps{Href: u}))
		out += render(t, Card(CardProps{Href: u}))
		out += render(t, Dropdown(DropdownProps{Items: []DropdownItem{{Label: "Go", Href: u}}}))
		out += render(t, DataTable(DataTableProps{Rows: []map[string]any{{"a": Cell{Value: 1, Href: u}}}}))
		if strings.Contains(out, "TemplFailedSanitization") {
			t.Errorf("%q was sanitized: %s", u, out)
		}
	}
	for _, u := range []string{"javascript:alert(1)", "JavaScript:x", "data:text/html,<b>"} {
		for _, c := range []templ.Component{
			Button(ButtonProps{Href: u}),
			Card(CardProps{Href: u}),
			Dropdown(DropdownProps{Items: []DropdownItem{{Label: "Go", Href: u}}}),
			DataTable(DataTableProps{Rows: []map[string]any{{"a": Cell{Value: 1, Href: u}}}}),
		} {
			if out := render(t, c); !strings.Contains(out, "about:invalid#TemplFailedSanitizationURL") {
				t.Errorf("%q was not sanitized: %s", u, out)
			}
		}
	}
	// Text that templ would read as a URL by its attribute's name stays as it is.
	for _, c := range []templ.Component{
		Details(DetailsProps{Icon: "🛰️: x"}),
		QRCode(QRCodeProps{Value: "WIFI:S:Starbase;T:WPA;P:secret;;"}),
		Autoloader(AutoloaderProps{Base: "javascript-free/", Modules: map[string]string{"x-a": "data:text/javascript,1"}}),
		DataTable(DataTableProps{Rows: []map[string]any{{"url": "javascript:not-a-link"}}}),
	} {
		if out := render(t, c); strings.Contains(out, "TemplFailedSanitization") {
			t.Errorf("sanitized: %s", out)
		}
	}
}

func TestChildren(t *testing.T) {
	inner := Button(ButtonProps{Variant: ButtonVariantPixel})
	out := render(t, withChildren(Modal(ModalProps{Heading: "Launch?"}),
		templ.Raw("Ready? "),
		withChildren(inner, templ.Raw("Go")),
		inner, // no children: must not get the modal's
	))
	want := `<sb-modal heading="Launch?">Ready? <sb-button variant="pixel">Go</sb-button><sb-button variant="pixel"></sb-button></sb-modal>`
	if out != want {
		t.Errorf("got  %s\nwant %s", out, want)
	}
}

func TestRenderErrorsStop(t *testing.T) {
	fail := templ.ComponentFunc(func(context.Context, io.Writer) error { return io.ErrClosedPipe })
	var b strings.Builder
	if err := withChildren(Card(CardProps{}), fail).Render(context.Background(), &b); err != io.ErrClosedPipe {
		t.Errorf("a child's error: %v", err)
	}
}

func TestTypesThroughJSON(t *testing.T) {
	for _, tc := range []struct {
		in   string
		into any
		want any
	}{
		{`["Rename", "-", "---", {"value":"del","label":"Delete","danger":true}, {"group":"Move"}, {"label":"Sub","children":["a"]}]`, &[]DropdownItem{},
			&[]DropdownItem{{Value: "Rename"}, {Divider: true}, {Divider: true}, {Value: "del", Label: "Delete", Danger: true}, {Group: "Move"}, {Label: "Sub", Children: []DropdownItem{{Value: "a"}}}}},
		{`["A", 2, {"value":"b","label":"B","disabled":true}]`, &[]Option{}, &[]Option{{Value: "A"}, {Value: "2"}, {Value: "b", Label: "B", Disabled: true}}},
		{`[{"key":"n","align":"end","resizable":false,"min":60}]`, &[]Column{}, &[]Column{{Key: "n", Align: ColumnAlignEnd, Resizable: Off, Min: 60}}},
		{`[{"id":"a","text":"Hi","duration":0}]`, &[]ToastMessage{}, &[]ToastMessage{{ID: "a", Text: "Hi", Duration: Int(0)}}},
		{`[{"label":"Sort","type":"radio","children":["a"]}]`, &[]DropdownItem{}, &[]DropdownItem{{Label: "Sort", Type: DropdownItemTypeRadio, Children: []DropdownItem{{Value: "a"}}}}},
	} {
		if err := json.Unmarshal([]byte(tc.in), tc.into); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(tc.into, tc.want) {
			t.Errorf("%s = %#v", tc.in, tc.into)
		}
		// And back: the same values.
		again := reflect.New(reflect.TypeOf(tc.into).Elem()).Interface()
		if err := json.Unmarshal(must(json.Marshal(tc.into)), again); err != nil || !reflect.DeepEqual(again, tc.want) {
			t.Errorf("%s round trip = %#v, %v", tc.in, again, err)
		}
	}
}

func must[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}

func TestIntegers(t *testing.T) {
	for _, tc := range []struct {
		c    templ.Component
		want string
	}{
		{Tabs(TabsProps{}), `<sb-tabs></sb-tabs>`},
		{Tabs(TabsProps{Selected: Int(0)}), `<sb-tabs selected="0"></sb-tabs>`}, // the server's word: back to the first tab
		{Tabs(TabsProps{Selected: Int(2)}), `<sb-tabs selected="2"></sb-tabs>`},
		// Server data: 0 is an answer like any other.
		{VirtualScroll(VirtualScrollProps{Offset: Int(0), Total: Int(0)}), `<sb-virtual-scroll total="0" offset="0"></sb-virtual-scroll>`},
		{VirtualScroll(VirtualScrollProps{Offset: Int(120), Total: Int(-1), Columns: Int(4)}), `<sb-virtual-scroll total="-1" offset="120" columns="4"></sb-virtual-scroll>`},
		{Select(SelectProps{MaxChips: Int(0), Delay: Int(300)}), `<sb-select delay="300" max-chips="0"></sb-select>`},
		{CopyButton(CopyButtonProps{ResetMS: Int(900)}), `<sb-copy-button reset-ms="900"></sb-copy-button>`},
		{Busy(BusyProps{Delay: 200}), `<sb-busy delay="200"></sb-busy>`},
	} {
		if got := render(t, tc.c); got != tc.want {
			t.Errorf("got  %s\nwant %s", got, tc.want)
		}
	}
	b := must(json.Marshal(ToastMessage{ID: "a", Text: "b", Duration: Int(4000)}))
	if string(b) != `{"id":"a","text":"b","duration":4000}` {
		t.Errorf("toast: %s", b)
	}
}

func TestStringLists(t *testing.T) {
	for _, tc := range []struct {
		c    templ.Component
		want string
	}{
		{Tree(TreeProps{}), `<sb-tree></sb-tree>`},
		{Tree(TreeProps{Value: []string{}, Expanded: []string{}}), `<sb-tree value="" expanded=""></sb-tree>`}, // the server clears both
		{Tree(TreeProps{Value: []string{"a"}, Expanded: []string{"a", "a/b", "ü"}}), `<sb-tree value="a" expanded="a a/b ü"></sb-tree>`},
		{TableOfContents(TableOfContentsProps{Levels: []string{"h2", "h3"}}), `<sb-toc levels="h2 h3"></sb-toc>`},
		{Select(SelectProps{}), `<sb-select></sb-select>`},
		{Select(SelectProps{Value: []string{}}), `<sb-select value=""></sb-select>`},
		{Select(SelectProps{Value: []string{"Mars, the red one"}}), `<sb-select value="Mars, the red one"></sb-select>`},
		{Select(SelectProps{Value: []string{" [x] "}}), `<sb-select value="[&#34; [x] &#34;]"></sb-select>`}, // read as JSON otherwise
		{Select(SelectProps{Value: []string{}, Multiple: true}), `<sb-select value="[]" multiple></sb-select>`},
		{Select(SelectProps{Value: []string{"a,b", "c"}, Multiple: true}), `<sb-select value="[&#34;a,b&#34;,&#34;c&#34;]" multiple></sb-select>`},
	} {
		if got := render(t, tc.c); got != tc.want {
			t.Errorf("got  %s\nwant %s", got, tc.want)
		}
	}
	for _, c := range []templ.Component{
		Tree(TreeProps{Expanded: []string{"a b"}}),
		Tree(TreeProps{Value: []string{""}}),
		TableOfContents(TableOfContentsProps{Levels: []string{"h2\th3"}}),
		Select(SelectProps{Value: []string{"a", "b"}}), // two without multiple
	} {
		var b strings.Builder
		if err := c.Render(context.Background(), &b); err == nil {
			t.Errorf("rendered %s", b.String())
		}
	}
}

// TestJSONDefaults: a JSON prop whose default isn't empty (sb-tabs' labels
// are ["Home","Docs","API"]) gets that default from nil, and an empty
// slice or map is written as [] or {}, never left out.
func TestJSONDefaults(t *testing.T) {
	n := 0
	for _, m := range manifests(t) {
		e := entry(t, m.Tag)
		for i, p := range m.Props {
			if p.Type != "json" && p.Type != "array" && p.Type != "custom" || len(p.Default) == 0 {
				continue
			}
			var def any
			json.Unmarshal(p.Default, &def)
			if def == nil || reflect.ValueOf(def).Kind() != reflect.Slice && reflect.ValueOf(def).Kind() != reflect.Map || reflect.ValueOf(def).Len() == 0 {
				continue
			}
			props := reflect.ValueOf(e.New())
			if got := host(t, render(t, e.Render(props.Interface())), m.Tag); len(got.attrs) != 0 {
				t.Errorf("%s %s: nil wrote %+v", m.Tag, p.Name, got.attrs)
			}
			f := props.Elem().Field(3 + i)
			want := "[]"
			switch f.Kind() {
			case reflect.Slice:
				f.Set(reflect.MakeSlice(f.Type(), 0, 0))
			case reflect.Map:
				f.Set(reflect.MakeMap(f.Type()))
				want = "{}"
			default:
				t.Errorf("%s %s: a %s for the default %s", m.Tag, p.Name, f.Type(), p.Default)
				continue
			}
			if err := present(p.Attribute, want)(host(t, render(t, e.Render(props.Interface())), m.Tag)); err != nil {
				t.Errorf("%s %s empty: %v", m.Tag, p.Name, err)
			}
			n++
		}
	}
	if n < 4 { // tabs' labels, theme-switch's themes, the playground's themes, the board's palette
		t.Errorf("only %d props with a default", n)
	}
	if got := render(t, Tabs(TabsProps{Labels: []string{}})); got != `<sb-tabs labels="[]"></sb-tabs>` {
		t.Errorf("tabs: %s", got)
	}
}

// TestEnumNames: every constant of a named string type is the type's name
// and then the value's (ToggleSizeLg, ToastVariantOK, SortDirDesc); a value
// that makes no name (sb-rating's "0.5") still has the type's prefix.
func TestEnumNames(t *testing.T) {
	fset := gotoken.NewFileSet()
	pkgs, err := parser.ParseDir(fset, ".", func(fi fs.FileInfo) bool { return !strings.HasSuffix(fi.Name(), "_test.go") }, 0)
	if err != nil {
		t.Fatal(err)
	}
	strTypes := map[string]bool{}
	type constant struct{ name, typ, value string }
	var consts []constant
	for _, f := range pkgs["sb"].Files {
		for _, d := range f.Decls {
			g, ok := d.(*ast.GenDecl)
			if !ok {
				continue
			}
			for _, s := range g.Specs {
				switch s := s.(type) {
				case *ast.TypeSpec:
					if id, ok := s.Type.(*ast.Ident); ok && id.Name == "string" {
						strTypes[s.Name.Name] = true
					}
				case *ast.ValueSpec:
					id, ok := s.Type.(*ast.Ident)
					if g.Tok != gotoken.CONST || !ok || len(s.Values) != len(s.Names) {
						continue
					}
					for i, n := range s.Names {
						if lit, ok := s.Values[i].(*ast.BasicLit); ok && lit.Kind == gotoken.STRING {
							v, _ := strconv.Unquote(lit.Value)
							consts = append(consts, constant{n.Name, id.Name, v})
						}
					}
				}
			}
		}
	}
	checked := 0
	for _, c := range consts {
		if !strTypes[c.typ] || c.typ == "Tag" { // Tag constants are named after the components
			continue
		}
		checked++
		rest, ok := strings.CutPrefix(c.name, c.typ)
		if !ok || rest == "" {
			t.Errorf("%s (a %s) doesn't start with %s", c.name, c.typ, c.typ)
			continue
		}
		var want strings.Builder
		for _, r := range c.value {
			if unicode.IsLetter(r) || unicode.IsDigit(r) {
				want.WriteRune(unicode.ToLower(r))
			}
		}
		if w := want.String(); w != "" && !unicode.IsDigit(rune(w[0])) && strings.ToLower(rest) != w {
			t.Errorf("%s = %q: want %s and the value's words", c.name, c.value, c.typ)
		}
	}
	if checked < 50 {
		t.Errorf("only %d constants checked", checked)
	}
}
