package sb

import (
	"context"
	"encoding/json"
	"io"
	"reflect"
	"strings"
	"testing"

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
		{MeterProps{Max: Num(1e21), Segments: Num(12)}, `<sb-meter max="1000000000000000000000" segments="12"></sb-meter>`},
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
	if got := render(t, Select(SelectProps{Value: Str("")})); got != `<sb-select value=""></sb-select>` {
		t.Errorf("Str(\"\"): got %s", got)
	}
	if got := render(t, Select(SelectProps{})); got != `<sb-select></sb-select>` {
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
		{`[{"id":"a","text":"Hi","duration":0}]`, &[]ToastMessage{}, &[]ToastMessage{{ID: "a", Text: "Hi", Duration: Num(0)}}},
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
