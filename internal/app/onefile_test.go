package app_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"slices"
	"strings"
	"sync/atomic"
	"testing"

	"starbase/components"
	"starbase/internal/catalog"
)

// bundlePage is a page that loads components the way a site using the
// one-file bundles does: an import map for Datastar, a module script for
// each URL in scripts, and body.
func bundlePage(datastar string, scripts []string, body string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var b strings.Builder
		fmt.Fprintf(&b, `<!doctype html><html lang="en"><head><meta charset="utf-8"><title>One-file bundles</title>
<link rel="stylesheet" href="/bundle/site.css">
<script type="importmap">{"imports": {"datastar": %q}}</script>
`, datastar)
		for _, s := range scripts {
			fmt.Fprintf(&b, "<script type=\"module\" src=%q></script>\n", s)
		}
		fmt.Fprintf(&b, "</head><body><main class=\"prose\">%s</main></body></html>", body)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(b.String()))
	}
}

// TestComponentBundlesDefineEveryTag loads every component from its
// one-file bundle alone: each defines its tag with the props its
// manifest.json lists, and the page requests nothing from /c/ but the
// bundles (Datastar comes from /static/).
func TestComponentBundlesDefineEveryTag(t *testing.T) {
	cat, err := catalog.Load(components.FS)
	if err != nil {
		t.Fatal(err)
	}
	var tags, want []string
	for _, c := range cat.Components {
		tags = append(tags, c.Tag)
		want = append(want, "/c/"+c.VersionedBundle())
	}
	handlers := func(app http.Handler) map[string]http.HandlerFunc {
		return map[string]http.HandlerFunc{"/__bundles": bundlePage(datastarPath(t, app), want, "")}
	}
	tj, _ := json.Marshal(tags)
	_, body := probeWith(t, "/__bundles", strings.Replace(definedPropsJS, "%TAGS%", string(tj), 1), handlers)
	var got definedProps
	if err := json.Unmarshal(body, &got); err != nil || got.Error != "" {
		t.Fatalf("%v: %s", err, body)
	}
	checkProps(t, cat, got.Manifests, "the one-file bundles")
	slices.Sort(want)
	slices.Sort(got.Loaded)
	if !slices.Equal(got.Loaded, want) {
		t.Errorf("the page loaded %v\nwant only the bundles and Datastar", got.Loaded)
	}
}

// TestKanbanBoardBundle runs sb-kanban-board's keyboard and pointer test
// (TestKanbanBoardMove) on a page that loads it from its one-file bundle:
// one request, where its module files take one per file. With
// STARBASE_DATASTAR_BUNDLE=<file>, on that Datastar build (see
// TestVirtualScrollFocus).
func TestKanbanBoardBundle(t *testing.T) {
	var official []byte
	if p := os.Getenv("STARBASE_DATASTAR_BUNDLE"); p != "" {
		var err error
		if official, err = os.ReadFile(p); err != nil {
			t.Fatal(err)
		}
	}
	cat, err := catalog.Load(components.FS)
	if err != nil {
		t.Fatal(err)
	}
	kanban, _ := cat.Get("kanban-board")
	i := slices.IndexFunc(kanban.Examples, func(e string) bool { return strings.Contains(e, `id="mission-board"`) })
	if i < 0 {
		t.Fatal("the README has no #mission-board example")
	}
	// The requests to the component's files, as a row ("loaded") the test reads.
	loaded := `
await customElements.whenDefined('sb-kanban-board')
const loaded = performance.getEntriesByType('resource').map((e) => new URL(e.name).pathname).filter((p) => p.startsWith('/c/kanban-board'))
rows.push({ step: 'loaded', got: JSON.stringify(loaded.length), want: JSON.stringify(loaded.length) })
`
	for _, tc := range []struct {
		name, module, script string
		files                int
	}{
		{"modules", "/c/" + kanban.VersionedMinScript(), loaded + "await report()", len(kanban.Sizes.Files)},
		{"bundle", "/c/" + kanban.VersionedBundle(), loaded + "check('one file', loaded, ['/c/" + kanban.VersionedBundle() + "'])\n" + kanbanBoardJS, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var served atomic.Int32
			handlers := func(app http.Handler) map[string]http.HandlerFunc {
				m := map[string]http.HandlerFunc{"/__kanban": bundlePage(datastarPath(t, app), []string{tc.module}, kanban.Examples[i])}
				if official != nil {
					m[datastarPath(t, app)] = func(w http.ResponseWriter, r *http.Request) {
						served.Add(1)
						w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
						w.Write(official)
					}
				}
				return m
			}
			_, body := probeWith(t, "/__kanban", pdPrelude+tc.script, handlers)
			if official != nil && served.Load() == 0 {
				t.Error("the page did not load STARBASE_DATASTAR_BUNDLE")
			}
			copyRows(t, body)
			var rows []struct{ Step, Got string }
			json.Unmarshal(body, &rows)
			for _, r := range rows {
				if r.Step == "loaded" {
					if r.Got != fmt.Sprint(tc.files) {
						t.Errorf("%s: %s requests to /c/kanban-board, want %d", tc.name, r.Got, tc.files)
					}
					t.Logf("sb-kanban-board from its %s: %s requests to its files", tc.name, r.Got)
				}
			}
		})
	}
}

// TestAutoloaderLoadsBundles opens the gallery with the autoloader
// (?load=auto): every component on it is defined from its one-file bundle,
// and nothing under /c/ is requested but the autoloader, bundles, and files
// a bundle loads on first use.
func TestAutoloaderLoadsBundles(t *testing.T) {
	cat, err := catalog.Load(components.FS)
	if err != nil {
		t.Fatal(err)
	}
	var tags []string
	for _, c := range cat.Components {
		tags = append(tags, c.Tag)
	}
	tj, _ := json.Marshal(tags)
	_, body := probe(t, "/?load=auto", `let out
try {
	const catalog = new Set(`+string(tj)+`)
	const tags = [...new Set([...document.querySelectorAll('*')].map((e) => e.localName).filter((t) => catalog.has(t)))]
	await Promise.race([Promise.all(tags.map((t) => customElements.whenDefined(t))), new Promise((_, no) => setTimeout(() => no(new Error('not defined: ' + tags.filter((t) => !customElements.get(t)))), 20000))])
	const loaded = performance.getEntriesByType('resource').map((e) => new URL(e.name).pathname).filter((p) => p.startsWith('/c/'))
	out = JSON.stringify({ tags, loaded })
} catch (e) { out = JSON.stringify({ error: String(e) }) }
await fetch('/__probe/result', { method: 'POST', body: out })`)
	var got struct {
		Tags, Loaded []string
		Error        string
	}
	if err := json.Unmarshal(body, &got); err != nil || got.Error != "" {
		t.Fatalf("%v: %s", err, body)
	}
	// sb-autoloader's gallery card loads its demo's own element.
	allowed := map[string]bool{"/c/autoloader.js": true, "/c/autoloader/demo-badge.js": true}
	for _, c := range cat.Components {
		allowed["/c/"+c.VersionedBundle()] = true
		for _, f := range c.Sizes.Lazy {
			allowed["/c/"+c.Slug+"@"+c.Hash+"/"+f.Name] = true
		}
	}
	for _, p := range got.Loaded {
		if !allowed[p] {
			t.Errorf("the page loaded %s, which is no bundle", p)
		}
	}
	n := 0
	for _, tag := range got.Tags {
		for _, c := range cat.Components {
			if c.Tag == tag {
				n++
				if !slices.Contains(got.Loaded, "/c/"+c.VersionedBundle()) {
					t.Errorf("<%s> is defined, but its bundle was not loaded", tag)
				}
			}
		}
	}
	if n < 20 {
		t.Errorf("only %d components on the gallery: %v", n, got.Tags)
	}
	t.Logf("%d components defined from %d requests under /c/", n, len(got.Loaded))
}
