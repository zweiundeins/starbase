package app_test

import (
	"encoding/json"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"starbase/components"
	"starbase/internal/catalog"
)

// TestShrunkModulesKeepProps loads the homepage, which runs every component
// from the bundle (built from shrunk sources: catalog/shrink.go leaves out
// .docs() and the manifest: block), and checks that each component still
// defines exactly the props its manifest.json lists — names, attributes,
// types, defaults and values — only without the docs. A shrink that cut into
// a prop chain would show up here.
func TestShrunkModulesKeepProps(t *testing.T) {
	cat, err := catalog.Load(components.FS)
	if err != nil {
		t.Fatal(err)
	}
	var tags []string
	for _, c := range cat.Components {
		tags = append(tags, c.Tag)
	}
	tj, _ := json.Marshal(tags)
	_, body := probe(t, "/", strings.Replace(`let out
try {
	const all = %TAGS%
	await Promise.race([Promise.all(all.map((t) => customElements.whenDefined(t))), new Promise((_, no) => setTimeout(() => no(new Error('not all tags were defined: ' + all.filter((t) => !customElements.get(t)))), 20000))])
	out = JSON.stringify(Object.fromEntries(all.map((t) => [t, customElements.get(t).manifest()])))
} catch (e) { out = JSON.stringify({ error: String(e) }) }
await fetch('/__probe/result', { method: 'POST', body: out })`, "%TAGS%", string(tj), 1))
	var got map[string]struct {
		Props []map[string]any `json:"props"`
	}
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("%v: %s", err, body)
	}
	if len(got) != len(cat.Components) {
		t.Fatalf("the page reported %d components, want %d: %s", len(got), len(cat.Components), body)
	}
	n := 0
	for _, c := range cat.Components {
		raw, err := components.FS.ReadFile(filepath.Join(c.Slug, "manifest.json"))
		if err != nil {
			t.Fatal(err)
		}
		var want struct {
			Props []map[string]any `json:"props"`
		}
		json.Unmarshal(raw, &want)
		for _, p := range want.Props {
			delete(p, "docs")
		}
		for _, p := range got[c.Tag].Props {
			if _, ok := p["docs"]; ok {
				t.Errorf("%s: prop %v still has docs in the bundle", c.Tag, p["name"])
				delete(p, "docs")
			}
		}
		n += len(want.Props)
		if !reflect.DeepEqual(got[c.Tag].Props, want.Props) {
			gj, _ := json.Marshal(got[c.Tag].Props)
			wj, _ := json.Marshal(want.Props)
			t.Errorf("%s: the bundle defines other props than manifest.json:\n got %s\nwant %s", c.Tag, gj, wj)
		}
	}
	t.Logf("%d props of %d components match their manifests", n, len(cat.Components))
}

