package app_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/a-h/templ"
	"github.com/zweiundeins/starbase/templ/sb"

	"starbase/components"
	"starbase/internal/catalog"
	"starbase/internal/templgen/registry"
	"starbase/static"
)

// templValues are fixture values for string props whose text has a format
// (a selector, a date, an attribute name), by "<tag>.<prop>"; every other
// string prop gets templText.
var templValues = map[string]any{
	"sb-autoloader.pattern":          "/c/{tag}/{tag}.js",
	"sb-autoloader.match":            "^x-",
	"sb-autoloader.base":             "/c/",
	"sb-autoloader.cloak":            "x-cloak",
	"sb-busy.for":                    "#nothing",
	"sb-code-playground.runner":      "/playground/run",
	"sb-code-playground.base":        "/c/button/",
	"sb-code-playground.theme":       "nebula",
	"sb-count-up.lang":               "de-CH",
	"sb-date-picker.value":           "2026-10-14",
	"sb-date-picker.min":             "2026-01-01",
	"sb-date-picker.max":             "2026-12-31",
	"sb-date-picker.month":           "2026-10",
	"sb-date-picker.timeZone":        "Europe/Zurich",
	"sb-date-picker.lang":            "en-GB",
	"sb-echarts.lang":                "en",
	"sb-odometer.lang":               "de-CH",
	"sb-pixel-board.cells":           strings.Repeat("0123", 4),
	"sb-relative-time.datetime":      "2026-09-22T08:00:00Z",
	"sb-relative-time.lang":          "en",
	"sb-relative-time.timeZone":      "UTC",
	"sb-relative-time.titleLang":     "en-GB",
	"sb-theme-switch.attribute":      "data-templ-theme",
	"sb-theme-switch.cookie":         "sb-templ-theme",
	"sb-theme-switch.domain":         "127.0.0.1",
	"sb-toc.content":                 "main",
	"sb-toc.levels":                  []string{"h2", "h3"},
	"sb-toc.compact":                 "(max-width: 10px)",
	"sb-tree.value":                  []string{"b"},
	"sb-tree.expanded":               []string{"a", "l"},
	"sb-virtual-scroll.itemSize":     "24",
	"sb-select.value":                []string{"b"},
	"sb-radio-group.value":           "b",
	"sb-dropdown.value":              "b",
	"sb-code-editor.value":           "const answer = 42",
	"sb-input.value":                 "Ünï <&> \"q\"",
	"sb-qr-code.value":               "https://data-star.dev",
	"sb-sparkline.values":            []float64{1, 5, 3},
	"sb-pixel-board.palette":         []string{"#000000", "#111111", "#222222", "#333333", "#444444", "#555555", "#666666", "#777777", "#888888", "#999999", "#AAAAAA", "#BBBBBB", "#CCCCCC", "#DDDDDD", "#EEEEEE", "#FFFFFF"},
	"sb-code-playground.themes":      []string{"deep-space", "nebula"},
	"sb-theme-switch.themes":         []string{"auto", "dark", "light"},
	"sb-theme-switch.labels":         []string{"Auto", "Dunkel", "Hell"},
	"sb-date-picker.disabledDates":   []string{"2026-10-15"},
	"sb-checkbox-group.value":        []string{"b"},
	"sb-data-table.selected":         []string{"b"},
	"sb-data-table.hiddenColumns":    []string{"moons"},
	"sb-data-table.widths":           map[string]float64{"name": 180},
	"sb-data-table.sort":             &sb.TableSort{Key: "name", Dir: sb.SortDirDesc},
	"sb-data-table.columns":          []sb.Column{{Key: "name", Label: "Planet", Width: "12rem"}, {Key: "moons", Label: "Moons", Align: sb.ColumnAlignEnd, Sortable: true, Resizable: sb.Off, Min: 60}},
	"sb-data-table.rows":             []map[string]any{{"id": "a", "name": "Ünï <&>", "moons": 1.0}, {"id": "b", "name": sb.Cell{Value: "Mars", Text: "Mars!", Href: "https://example.com/", Tone: sb.CellToneDanger}, "moons": 2.0}},
	"sb-dropdown.items":              []sb.DropdownItem{{Value: "a", Label: "A <&>", Icon: "⛽"}, {Divider: true}, {Group: "More"}, {Value: "b", Label: "B", Danger: true}, {Label: "Sub", Children: []sb.DropdownItem{{Value: "c", Label: "C"}}}},
	"sb-select.options":              []sb.Option{{Value: "a"}, {Value: "b", Label: "B", Description: "Bee", Disabled: true}},
	"sb-select.results":              []sb.Option{{Value: "r", Label: "R"}},
	"sb-radio-group.options":         []sb.Option{{Value: "a"}, {Value: "b", Label: "B"}},
	"sb-checkbox-group.options":      []sb.Option{{Value: "a"}, {Value: "b", Label: "B", Description: "Bee"}},
	"sb-range.value":                 &sb.RangeValue{Start: 20, End: 60},
	"sb-toast.toasts":                []sb.ToastMessage{{ID: "t", Title: "Saved", Text: "Live <&>", Variant: sb.ToastVariantOK, Duration: sb.Int(0)}},
	"sb-tree.items":                  []sb.TreeItem{{ID: "a", Label: "A", Icon: "🪐", Children: []sb.TreeItem{{ID: "b", Label: "B"}}}, {ID: "l", Label: "Lazy", Lazy: true}},
	"sb-tree.loaded":                 map[string][]sb.TreeItem{"l": {{ID: "l1", Label: "L1"}}},
	"sb-tabs.labels":                 []string{"One", "Two <&>"},
	"sb-typewriter.phrases":          []string{"one", "two <&>"},
	"sb-code-editor.diagnostics":     []sb.Diagnostic{{Line: 1, Col: 7, Length: 6, Message: "unused <&>", Text: "answer"}},
	"sb-code-playground.diagnostics": map[string][]sb.Diagnostic{"component.js": {{Line: 1, Col: 1, Length: 1, Message: "x"}}},
	"sb-code-playground.deps":        map[string]string{"sb-button": "/c/button/button.js"},
	"sb-code-playground.initial":     map[string]string{"index.html": "<p>Hi</p>"},
	"sb-autoloader.modules":          map[string]string{"x-none": "/c/none.js"},
	"sb-autoloader.requires":         map[string][]string{"x-none": {"x-other"}},
	"sb-echarts.option":              map[string]any{"xAxis": map[string]any{"type": "category", "data": []any{"a", "b"}}, "yAxis": map[string]any{"type": "value"}, "series": []any{map[string]any{"type": "bar", "data": []any{1.0, 2.0}}}},
	// Props that rule out others' values: a single choice, a closed range.
	"sb-select.multiple":      false,
	"sb-date-picker.mode":     "single",
	"sb-date-picker.inline":   false,
	"sb-date-picker.disabled": false,
	"sb-date-picker.time":     false, // a date without a time
	// The calendar is a popover="auto": another open layer on the page (a
	// modal, a dropdown) closes it, and open reads that live state.
	"sb-date-picker.open": false,
	// Numbers inside each prop's clamp (and step), and ranges that hold them.
	"sb-busy.delay":             200.0,
	"sb-busy.min":               400.0,
	"sb-data-table.rowHeight":   40.0,
	"sb-starfield.density":      50.0,
	"sb-code-playground.delay":  300.0,
	"sb-slider.min":             0.0,
	"sb-slider.max":             100.0,
	"sb-range.min":              0.0,
	"sb-range.max":              100.0,
	"sb-busy.lines":             2.0,
	"sb-copy-button.resetMs":    900.0,
	"sb-meter.segments":         10.0,
	"sb-meter.decimals":         1.0,
	"sb-gauge.decimals":         1.0,
	"sb-count-up.decimals":      2.0,
	"sb-odometer.decimals":      1.0,
	"sb-nebula.density":         0.25,
	"sb-nebula.levels":          4.0,
	"sb-nebula.pixel":           3.0,
	"sb-pixel-board.size":       8.0,
	"sb-pixel-board.color":      3.0,
	"sb-qr-code.border":         3.0,
	"sb-rating.value":           2.0,
	"sb-rating.max":             4.0,
	"sb-sparkline.decimals":     1.0,
	"sb-sparkline.length":       20.0,
	"sb-tabs.selected":          1.0,
	"sb-date-picker.step":       900.0,
	"sb-image-compare.position": 30.0,
	"sb-slider.value":           40.0,
	"sb-slider.step":            5.0,
	"sb-range.step":             5.0,
	"sb-voxel.zoom":             1.5,
	"sb-select.maxChips":        2.0,
	"sb-select.minChars":        2.0,
	"sb-toast.max":              2.0,
	"sb-code-editor.tabSize":    4.0,
	// Server data: 0 is the first window, written like any other offset.
	"sb-virtual-scroll.offset": 0,
	"sb-virtual-scroll.total":  40,
	"sb-data-table.offset":     0,
	"sb-data-table.total":      7,
	"sb-select.total":          12,
	"sb-select.delay":          275,
	"sb-toast.duration":        4500,
	"sb-typewriter.hold":       2500,
}

// templText is the text every other string prop gets: markup to escape,
// and letters beyond ASCII.
const templText = "Ünï <&> \"q\" 'a'"

// fixture sets every field of props (a pointer) for the component with
// manifest m, and returns the values the element's props should decode to,
// as JSON, by prop name.
func fixture(t *testing.T, tag string, m *catalog.Manifest, props any) map[string]json.RawMessage {
	t.Helper()
	v := reflect.ValueOf(props).Elem()
	want := map[string]json.RawMessage{}
	for i, p := range m.Props {
		f := v.Field(3 + i)
		key := tag + "." + p.Name
		var val any
		switch {
		case templValues[key] != nil:
			val = templValues[key]
		case f.Kind() == reflect.Bool:
			val = true
		case f.Type() == reflect.TypeFor[sb.Flag]():
			val = sb.Off
		case f.Kind() == reflect.String && f.Type().Name() != "string": // an enum: its last value, rarely the default
			val = p.Values[len(p.Values)-1].(string)
		case f.Kind() == reflect.String, f.Type() == reflect.TypeFor[*string]():
			val = templText
		case f.Kind() == reflect.Float64, f.Type() == reflect.TypeFor[*float64]():
			val = 3.5 // a fraction, which an int prop would lose
		case f.Kind() == reflect.Int, f.Type() == reflect.TypeFor[*int]():
			val = 3
		default:
			t.Fatalf("%s: no fixture for %s", key, f.Type())
		}
		rv := reflect.ValueOf(val)
		switch {
		case f.Kind() == reflect.Pointer && rv.Kind() != reflect.Pointer:
			p := reflect.New(f.Type().Elem())
			p.Elem().Set(rv.Convert(f.Type().Elem()))
			f.Set(p)
		default:
			f.Set(rv.Convert(f.Type()))
		}
		// What the prop decodes to: the value itself, as JSON (a Flag is a
		// bool, a list in a string prop the string the component reads).
		var b []byte
		if fl, ok := val.(sb.Flag); ok {
			b, _ = json.Marshal(fl == sb.On)
		} else if list, ok := val.([]string); ok && p.Type == "string" {
			b, _ = json.Marshal(strings.Join(list, " ")) // sb-select's one value, without multiple
		} else {
			b, _ = json.Marshal(f.Interface())
		}
		want[p.Name] = b
	}
	return want
}

// TestTemplWrappersRoundTrip renders every component through its templ
// wrapper (package sb), every field set, on a page that loads them with
// sb.Imports from this server, and reads each element's props in the
// browser: what Rocket decoded from the attributes equals the Go values
// (strings, booleans by presence, flags, enums, numbers, JSON). With
// STARBASE_DATASTAR_BUNDLE=<file>, on that Datastar build (TestTemplWrappersOnOfficialBundle).
func TestTemplWrappersRoundTrip(t *testing.T) {
	templRoundTrip(t)
}

// TestTemplWrappersOnOfficialBundle is TestTemplWrappersRoundTrip on the
// official Datastar + Rocket release.
func TestTemplWrappersOnOfficialBundle(t *testing.T) {
	if os.Getenv("STARBASE_DATASTAR_BUNDLE") == "" {
		t.Setenv("STARBASE_DATASTAR_BUNDLE", officialBundle)
	}
	templRoundTrip(t)
}

func templRoundTrip(t *testing.T) {
	cat, err := catalog.Load(components.FS)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]map[string]json.RawMessage{}
	flagsOn := map[string][]string{}
	extras := map[string]map[string]any{} // element id → prop → what it decodes to
	var body bytes.Buffer
	var tags []string
	for _, e := range registry.Entries {
		c := find(cat, e.Tag)
		props := e.New()
		want[e.Tag] = fixture(t, e.Tag, c.Manifest, props)
		reflect.ValueOf(props).Elem().Field(0).SetString("templ-" + e.Tag)
		if err := e.Render(props).Render(context.Background(), &body); err != nil {
			t.Fatalf("%s: %v", e.Tag, err)
		}
		tags = append(tags, e.Tag)
		// And with every flag On, which must decode to true as well.
		on := e.New()
		pv := reflect.ValueOf(on).Elem()
		for i := range pv.NumField() {
			if pv.Field(i).Type() == reflect.TypeFor[sb.Flag]() {
				pv.Field(i).Set(reflect.ValueOf(sb.On))
				flagsOn[e.Tag] = append(flagsOn[e.Tag], c.Manifest.Props[i-3].Name)
			}
		}
		if len(flagsOn[e.Tag]) > 0 {
			pv.Field(0).SetString("templ-on-" + e.Tag)
			if err := e.Render(on).Render(context.Background(), &body); err != nil {
				t.Fatalf("%s: %v", e.Tag, err)
			}
		}
		// A JSON prop whose default isn't empty, given an empty slice or
		// map: [] or {}, not the default.
		empty := e.New()
		ev := reflect.ValueOf(empty).Elem()
		for i, p := range c.Manifest.Props {
			var def any
			json.Unmarshal(p.Default, &def)
			if dv := reflect.ValueOf(def); def == nil || (dv.Kind() != reflect.Slice && dv.Kind() != reflect.Map) || dv.Len() == 0 {
				continue
			}
			f := ev.Field(3 + i)
			if f.Kind() == reflect.Map {
				f.Set(reflect.MakeMap(f.Type()))
				extras["templ-empty-"+e.Tag] = map[string]any{p.Name: map[string]any{}}
			} else {
				f.Set(reflect.MakeSlice(f.Type(), 0, 0))
				extras["templ-empty-"+e.Tag] = map[string]any{p.Name: []any{}}
			}
		}
		if extras["templ-empty-"+e.Tag] != nil {
			ev.Field(0).SetString("templ-empty-" + e.Tag)
			if err := e.Render(empty).Render(context.Background(), &body); err != nil {
				t.Fatalf("%s: %v", e.Tag, err)
			}
		}
	}
	if len(extras) < 4 {
		t.Errorf("%d JSON props with a default that isn't empty", len(extras))
	}
	// Lists: several values, and a value that would read as JSON.
	for id, c := range map[string]templ.Component{
		"templ-multi":      sb.Select(sb.SelectProps{ID: "templ-multi", Multiple: true, Value: []string{"a,b", "c"}, Options: []sb.Option{{Value: "a,b"}, {Value: "c"}}}),
		"templ-brackets":   sb.Select(sb.SelectProps{ID: "templ-brackets", Value: []string{"[x]"}}),
		"templ-tree-multi": sb.Tree(sb.TreeProps{ID: "templ-tree-multi", Selection: sb.TreeSelectionMultiple, Value: []string{"a", "b"}, Expanded: []string{}, Items: []sb.TreeItem{{ID: "a", Label: "A"}, {ID: "b", Label: "B"}}}),
	} {
		if err := c.Render(context.Background(), &body); err != nil {
			t.Fatalf("%s: %v", id, err)
		}
	}
	extras["templ-multi"] = map[string]any{"value": []any{"a,b", "c"}}
	extras["templ-brackets"] = map[string]any{"value": "[x]"}
	extras["templ-tree-multi"] = map[string]any{"value": []any{"a", "b"}, "expanded": ""}
	// The panels' slots, which sb.TabSlot names.
	slotLabels := []string{"Home", "C++", "C#", "c", "  Hello, World!  ", "", "--", "Übersicht", "İstanbul", "ΟΔΟΣ", "ΣΑ", "ΟΔΟΣ.", "Σ", "日本語 2", "x²", "a", "a", "a-1", "Ǆemal", "ﬀ"}
	if err := sb.Tabs(sb.TabsProps{ID: "templ-slots", Labels: slotLabels}).Render(context.Background(), &body); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(body.String(), `<sb-virtual-scroll id="templ-sb-virtual-scroll" total="40" offset="0"`) {
		t.Errorf("no offset=\"0\" on the virtual scroll")
	}
	official := os.Getenv("STARBASE_DATASTAR_BUNDLE") != ""
	handlers := map[string]http.HandlerFunc{"/__templ": func(w http.ResponseWriter, r *http.Request) {
		base := "http://" + r.Host
		p := sb.ImportsProps{Base: base}
		for _, tag := range tags {
			p.Components = append(p.Components, sb.Tag(tag))
		}
		if official { // served in place of the pinned build: without its integrity
			p.Datastar = base + "/c/" + catalog.DatastarSlug + "@" + catalog.VersionHash(static.Datastar()) + "/" + catalog.DatastarFile
		}
		var head bytes.Buffer
		if err := sb.Imports(p).Render(templ.WithNonce(r.Context(), "n"), &head); err != nil {
			t.Error(err)
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprintf(w, "<!doctype html><html lang=\"en\"><head><meta charset=\"utf-8\"><title>templ</title>%s</head><body><main><h2 id=\"h\">Heading</h2>%s</main></body></html>", head.String(), body.String())
	}}
	names := map[string][]string{}
	for _, c := range cat.Components {
		for _, p := range c.Manifest.Props {
			names[c.Tag] = append(names[c.Tag], p.Name)
		}
	}
	nj, _ := json.Marshal(names)
	oj, _ := json.Marshal(flagsOn)
	ej, _ := json.Marshal(extras)
	script := `let out
try {
	const names = ` + string(nj) + `
	const all = Object.keys(names)
	await Promise.race([Promise.all(all.map((t) => customElements.whenDefined(t))), new Promise((_, no) => setTimeout(() => no(new Error('not all tags were defined: ' + all.filter((t) => !customElements.get(t)))), 20000))])
	await new Promise((r) => requestAnimationFrame(() => requestAnimationFrame(r)))
	const got = {}
	for (const t of all) {
		const el = document.getElementById('templ-' + t)
		got[t] = {}
		for (const n of names[t]) got[t][n] = el[n] === undefined ? '(undefined)' : el[n]
	}
	const on = {}
	for (const [t, flags] of Object.entries(` + string(oj) + `)) {
		const el = document.getElementById('templ-on-' + t)
		on[t] = Object.fromEntries(flags.map((n) => [n, el[n]]))
	}
	const extras = {}
	for (const [id, props] of Object.entries(` + string(ej) + `)) {
		const el = document.getElementById(id)
		extras[id] = Object.fromEntries(Object.keys(props).map((n) => [n, el[n]]))
	}
	const slots = [...document.getElementById('templ-slots').shadowRoot.querySelectorAll('[role=tabpanel] > slot')].map((s) => s.name)
	const vs = document.getElementById('templ-sb-virtual-scroll')
	out = JSON.stringify({ got, on, extras, slots, offset: [vs.getAttribute('offset'), vs.offset] })
} catch (e) { out = JSON.stringify({ error: String(e) }) }
await fetch('/__probe/result', { method: 'POST', body: out })`
	res := probeBundle(t, "/__templ", script, handlers)
	var got struct {
		Got    map[string]map[string]json.RawMessage
		On     map[string]map[string]any
		Extras map[string]map[string]any
		Slots  []string
		Offset []any
		Error  string
	}
	if err := json.Unmarshal(res, &got); err != nil || got.Error != "" {
		t.Fatalf("%v: %s", err, res)
	}
	n := 0
	for tag, props := range want {
		for name, w := range props {
			var wv, gv any
			json.Unmarshal(w, &wv)
			json.Unmarshal(got.Got[tag][name], &gv)
			if !reflect.DeepEqual(wv, gv) {
				t.Errorf("%s %s: decoded %s, want %s", tag, name, got.Got[tag][name], w)
			}
			n++
		}
	}
	for tag, flags := range flagsOn {
		for _, name := range flags {
			if got.On[tag][name] != true {
				t.Errorf("%s %s = On: decoded %v, want true", tag, name, got.On[tag][name])
			}
			n++
		}
	}
	for id, props := range extras {
		for name, w := range props {
			if !reflect.DeepEqual(got.Extras[id][name], w) {
				t.Errorf("%s %s: decoded %#v, want %#v", id, name, got.Extras[id][name], w)
			}
			n++
		}
	}
	var slots []string
	for i := range slotLabels {
		slots = append(slots, sb.TabSlot(slotLabels, i))
	}
	if !slices.Equal(got.Slots, slots) {
		t.Errorf("sb-tabs' slots %q, sb.TabSlot %q", got.Slots, slots)
	}
	if !reflect.DeepEqual(got.Offset, []any{"0", 0.0}) {
		t.Errorf("sb-virtual-scroll offset: attribute and prop %v", got.Offset)
	}
	t.Logf("%d props of %d components compared, %d slots", n, len(want), len(slots))
}

func find(cat *catalog.Catalog, tag string) *catalog.Component {
	for _, c := range cat.Components {
		if c.Tag == tag {
			return c
		}
	}
	panic("no component " + tag)
}
