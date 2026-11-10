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
	_, body := probe(t, "/", strings.Replace(definedPropsJS, "%TAGS%", string(tj), 1))
	var got definedProps
	if err := json.Unmarshal(body, &got); err != nil || got.Error != "" {
		t.Fatalf("%v: %s", err, body)
	}
	checkProps(t, cat, got.Manifests, "the bundle")
}

// definedProps is what definedPropsJS posts.
type definedProps struct {
	Manifests map[string]struct {
		Props []map[string]any `json:"props"`
	}
	Loaded []string
	Error  string
}

// definedPropsJS waits for every tag in %TAGS% to be defined, then posts
// {manifests: each tag's Rocket manifest, loaded: every /c/ path the page
// requested}.
const definedPropsJS = `let out
try {
	const all = %TAGS%
	await Promise.race([Promise.all(all.map((t) => customElements.whenDefined(t))), new Promise((_, no) => setTimeout(() => no(new Error('not all tags were defined: ' + all.filter((t) => !customElements.get(t)))), 20000))])
	const loaded = performance.getEntriesByType('resource').map((e) => new URL(e.name).pathname).filter((p) => p.startsWith('/c/'))
	out = JSON.stringify({ manifests: Object.fromEntries(all.map((t) => [t, customElements.get(t).manifest()])), loaded })
} catch (e) { out = JSON.stringify({ error: String(e) }) }
await fetch('/__probe/result', { method: 'POST', body: out })`

// checkProps compares the props the page defined, by tag, with every
// manifest.json, docs aside: what has them (the shrunk modules) has none.
func checkProps(t *testing.T, cat *catalog.Catalog, got map[string]struct {
	Props []map[string]any `json:"props"`
}, what string) {
	t.Helper()
	if len(got) != len(cat.Components) {
		t.Fatalf("the page reported %d components, want %d", len(got), len(cat.Components))
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
				t.Errorf("%s: prop %v still has docs in %s", c.Tag, p["name"], what)
				delete(p, "docs")
			}
		}
		n += len(want.Props)
		if !reflect.DeepEqual(got[c.Tag].Props, want.Props) {
			gj, _ := json.Marshal(got[c.Tag].Props)
			wj, _ := json.Marshal(want.Props)
			t.Errorf("%s: %s defines other props than manifest.json:\n got %s\nwant %s", c.Tag, what, gj, wj)
		}
	}
	t.Logf("%s: %d props of %d components match their manifests", what, n, len(cat.Components))
}
