package catalog_test

import (
	"encoding/json"
	"strings"
	"testing"
	"testing/fstest"

	"starbase/internal/catalog"
)

const pgManifest = `{"tag":"sb-widget","props":[
 {"name":"yaw","attribute":"yaw","type":"number","default":0},
 {"name":"zoom","attribute":"zoom","type":"number","default":1},
 {"name":"spin","attribute":"spin","type":"boolean","default":false},
 {"name":"showValue","attribute":"show-value","type":"boolean","default":true},
 {"name":"model","attribute":"model","type":"oneOf","default":"rocket","values":["rocket","planet"]},
 {"name":"label","attribute":"label","type":"string","default":""},
 {"name":"items","attribute":"items","type":"array","default":[]},
 {"name":"href","attribute":"href","type":"string","default":""}
]}`

func pgComponent(t *testing.T, extraFrontMatter string) *catalog.Component {
	t.Helper()
	readme := strings.Replace(validReadme, "preview: <sb-widget></sb-widget>\n",
		"preview: <sb-widget></sb-widget>\n"+extraFrontMatter, 1)
	cat, err := catalog.Load(fstest.MapFS{
		"widget/README.md":     {Data: []byte(readme)},
		"widget/widget.js":     {Data: []byte("rocket('sb-widget', {})")},
		"widget/manifest.json": {Data: []byte(strings.ReplaceAll(pgManifest, `"sb-widget"`, `"sb-widget"`))},
	})
	if err != nil {
		t.Fatal(err)
	}
	c, _ := cat.Get("widget")
	return c
}

func TestPlaygroundControls(t *testing.T) {
	c := pgComponent(t, `playground:
  props: {yaw: {min: -180, max: 180}, zoom: {min: 0.5, max: 3, step: 0.1}}
  values: {yaw: 30, model: planet}
  content: Hello
  attrs: {items: "[1,2]"}
  exclude: [href]
`)
	pg := c.Playground()
	if pg == nil {
		t.Fatal("no playground")
	}
	kinds := map[string]catalog.Control{}
	for _, ctl := range pg.Controls {
		kinds[ctl.Prop] = ctl
	}
	if _, ok := kinds["items"]; ok {
		t.Error("array props must be skipped")
	}
	if _, ok := kinds["href"]; ok {
		t.Error("excluded props must be skipped")
	}
	if y := kinds["yaw"]; y.Kind != catalog.ControlNumber || y.Min != -180 || y.Max != 180 || y.Initial != 30.0 {
		t.Errorf("yaw = %+v", y)
	}
	if z := kinds["zoom"]; z.Step != 0.1 || z.Min != 0.5 {
		t.Errorf("zoom = %+v", z)
	}
	if m := kinds["model"]; m.Kind != catalog.ControlSelect || len(m.Options) != 2 || m.Initial != "planet" {
		t.Errorf("model = %+v", m)
	}

	var sig map[string]any
	if err := json.Unmarshal([]byte(pg.Signals()), &sig); err != nil || sig["yaw"] != 30.0 || sig["showValue"] != true {
		t.Errorf("signals = %s (%v)", pg.Signals(), err)
	}

	el := pg.Element()
	for _, want := range []string{
		` yaw="30" data-attr:yaw="$_pg.yaw"`,
		`data-attr:spin="$_pg.spin || (el.spin = false)"`,
		`data-attr:show-value="$_pg.showValue ? null : 'false'"`,
		`data-attr:label="$_pg.label || null"`,
		`>Hello</sb-widget>`,
		`<sb-widget items="[1,2]"`,
	} {
		if !strings.Contains(el, want) {
			t.Errorf("element lacks %q:\n%s", want, el)
		}
	}
}

// An event named in sync moves the controls of the props its detail
// reports, snapped to their steps, on the live element only.
func TestPlaygroundSync(t *testing.T) {
	pg := pgComponent(t, `playground:
  props: {zoom: {min: 0.5, max: 3, step: 0.05}}
  sync: {sb-orbit: [yaw, zoom, spin, nope]}
`).Playground()
	want := ` data-on:sb-orbit="$_pg.yaw = Math.round(evt.detail.yaw); $_pg.zoom = +(Math.round(evt.detail.zoom / 0.05) * 0.05).toFixed(2)"`
	if !strings.Contains(pg.Element(), want) {
		t.Errorf("element lacks %q:\n%s", want, pg.Element())
	}
	if strings.Contains(pg.Markup(), "sb-orbit") {
		t.Errorf("the shown markup has the sync listener: %s", pg.Markup())
	}
}

func TestPlaygroundMarkupKeepsStaticAttrs(t *testing.T) {
	pg := pgComponent(t, "playground:\n  attrs: {items: \"[1,2]\"}\n").Playground()
	if !strings.Contains(pg.Markup(), `sb-widget items='[1,2]'`) {
		t.Errorf("markup = %s", pg.Markup())
	}
}

func TestPlaygroundNeedsManifest(t *testing.T) {
	c := &catalog.Component{}
	if c.Playground() != nil {
		t.Fatal("no manifest, no playground")
	}
}
