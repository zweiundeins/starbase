package web_test

import (
	"encoding/json"
	"html"
	"net/http"
	"net/http/cookiejar"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"

	"starbase/internal/catalog"
	"starbase/internal/commands"
	"starbase/static"
)

// installPanels returns the Installation section's snippets by slot, in
// order, as their copy buttons copy them.
func installPanels(t *testing.T, page string) map[string][]string {
	t.Helper()
	i := strings.Index(page, `<sb-tabs class="install-tabs"`)
	j := strings.Index(page[max(i, 0):], "</sb-tabs>")
	if i < 0 || j < 0 {
		t.Fatal("no installation tabs")
	}
	out := map[string][]string{}
	for _, panel := range strings.Split(page[i:i+j], `<div slot="`)[1:] {
		slot, rest, _ := strings.Cut(panel, `"`)
		for _, m := range copyRe.FindAllStringSubmatch(rest, -1) {
			out[slot] = append(out[slot], html.UnescapeString(m[1]))
		}
	}
	return out
}

var (
	copyRe      = regexp.MustCompile(`<sb-copy-button class="code-copy" value="([^"]*)"`)
	scriptRe    = regexp.MustCompile(`<script type="module" src="([^"]+)"(?: integrity="([^"]+)")?></script>`)
	selectedRe  = regexp.MustCompile(`<sb-tabs class="install-tabs" labels="[^"]*" selected="(\d)"`)
	integrityRe = regexp.MustCompile(`"integrity": (\{[^}]*\})`)
)

// servedSRI fetches u and returns the SRI hash of what is served there.
func servedSRI(t *testing.T, c *http.Client, u string) string {
	t.Helper()
	res, body := get(t, c, u)
	if res.StatusCode != 200 {
		t.Fatalf("GET %s = %d", u, res.StatusCode)
	}
	return catalog.SRI([]byte(body))
}

func TestInstallSnippets(t *testing.T) {
	ts, c, _, cat := newServerBus(t)
	slider, _ := cat.Get("slider")
	_, page := get(t, c, ts.URL+"/components/slider")
	p := installPanels(t, page)
	if len(p) != 4 {
		t.Fatalf("want 4 panels with a copy button, got %v", len(p))
	}
	// The snippets end with the usage, not the gallery card's preview.
	usage := strings.TrimSpace(slider.Usage)
	if usage == "" || usage == strings.TrimSpace(slider.Preview) {
		t.Fatalf("the slider needs a usage: that differs from its preview for this test, got %q", usage)
	}
	for slot, snips := range p {
		for _, snip := range snips {
			if strings.Contains(snip, "<!--") {
				t.Errorf("%s: snippets carry no comments:\n%s", slot, snip)
			}
			if !strings.HasSuffix(snip, usage) {
				t.Errorf("%s: should end with the component's markup:\n%s", slot, snip)
			}
		}
	}
	if n := []int{len(p["autoloader"]), len(p["this-component"]), len(p["pinned"]), len(p["self-host"])}; !slices.Equal(n, []int{1, 2, 1, 2}) {
		t.Fatalf("snippets per panel: %v", n)
	}

	// Autoloader: the latest autoloader and the pinned Datastar build.
	datastarURL := ts.URL + "/c/datastar@" + catalog.VersionHash(static.Datastar()) + "/datastar-rocket.js"
	if a := p["autoloader"][0]; !strings.Contains(a, `"datastar": "`+datastarURL+`"`) || !strings.Contains(a, `<script type="module" src="`+ts.URL+`/c/autoloader.js"></script>`) {
		t.Errorf("autoloader panel:\n%s", a)
	}

	// This component: its one-file bundle, then its minified module, pinned
	// with the frozen integrity that matches what the URL serves.
	for i, want := range []string{slider.VersionedBundle(), slider.VersionedMinScript()} {
		m := scriptRe.FindAllStringSubmatch(p["this-component"][i], -1)
		if len(m) != 1 || m[0][1] != ts.URL+"/c/"+want {
			t.Fatalf("this-component panel:\n%s", p["this-component"][i])
		}
		if got := servedSRI(t, c, m[0][1]); got != m[0][2] {
			t.Errorf("%s: integrity %s, served %s", m[0][1], m[0][2], got)
		}
	}

	// Pinned: the snapshot autoloader, and integrity entries that match both
	// what is served and the snapshot's importmap.json.
	pinned := p["pinned"][0]
	m := scriptRe.FindAllStringSubmatch(pinned, -1)
	if len(m) != 1 || m[0][1] != ts.URL+"/c/@"+cat.Hash+"/autoloader.js" || servedSRI(t, c, m[0][1]) != m[0][2] {
		t.Fatalf("pinned autoloader:\n%s", pinned)
	}
	var entries map[string]string
	if im := integrityRe.FindStringSubmatch(pinned); im == nil || json.Unmarshal([]byte(im[1]), &entries) != nil {
		t.Fatalf("pinned import map:\n%s", pinned)
	}
	_, mapJSON := get(t, c, ts.URL+"/c/@"+cat.Hash+"/importmap.json")
	var full struct{ Integrity map[string]string }
	json.Unmarshal([]byte(mapJSON), &full)
	if entries[datastarURL] != catalog.SRI(static.Datastar()) {
		t.Error("pinned import map lacks Datastar's integrity")
	}
	if entries[ts.URL+"/c/"+slider.VersionedBundle()] == "" {
		t.Error("pinned import map lacks the component's bundle, which the snapshot's autoloader loads")
	}
	for u, sri := range entries {
		if full.Integrity[u] != sri || servedSRI(t, c, u) != sri {
			t.Errorf("%s: integrity %s doesn't match importmap.json or the served file", u, sri)
		}
	}
	if !strings.Contains(page, `href="`+ts.URL+`/c/@`+cat.Hash+`/importmap.json"`) {
		t.Error("pinned panel should link the full import map")
	}

	// Self-host: an import map at your own Datastar, your copy of the
	// bundle or of the module, and links to every file.
	for i, file := range []string{"slider.bundle.min.js", "slider.min.js"} {
		if s := p["self-host"][i]; !strings.Contains(s, `"datastar": "/js/datastar-rocket.js"`) || !strings.Contains(s, `src="/js/slider/`+file+`"`) {
			t.Errorf("self-host panel:\n%s", s)
		}
	}
	for _, u := range []string{ts.URL + "/c/" + slider.VersionedBundle(), ts.URL + "/c/" + slider.VersionedMinScript(), ts.URL + "/c/" + slider.VersionedScript()} {
		if !strings.Contains(page, `href="`+u+`"`) {
			t.Errorf("self-host panel lacks a link to %s", u)
		}
	}
}

// A component that renders others pins them too (code-playground renders
// code-editor, which imports its vendored Prism).
func TestInstallDependencies(t *testing.T) {
	ts, c, _, cat := newServerBus(t)
	pg, _ := cat.Get("code-playground")
	editor, _ := cat.Get("code-editor")
	deps := cat.Deps(pg)
	if len(deps) == 0 || deps[0] != editor {
		t.Fatalf("code-playground's dependencies: %v", deps)
	}
	_, page := get(t, c, ts.URL+"/components/code-playground")
	p := installPanels(t, page)
	for i, scripts := range [][]string{{pg.VersionedBundle(), editor.VersionedBundle()}, {pg.VersionedMinScript(), editor.VersionedMinScript()}} {
		m := scriptRe.FindAllStringSubmatch(p["this-component"][i], -1)
		if len(m) != 1+len(deps) || m[0][1] != ts.URL+"/c/"+scripts[0] || m[1][1] != ts.URL+"/c/"+scripts[1] {
			t.Fatalf("this-component panel:\n%s", p["this-component"][i])
		}
		for _, s := range m {
			if servedSRI(t, c, s[1]) != s[2] {
				t.Errorf("%s: integrity doesn't match the served file", s[1])
			}
		}
	}
	prism := ts.URL + "/c/" + editor.Slug + "@" + editor.Hash + "/vendor/prism.min.js"
	if !strings.Contains(p["this-component"][0], `"`+prism+`"`) || !strings.Contains(p["pinned"][0], `"`+prism+`"`) || !strings.Contains(page, `href="`+prism+`"`) {
		t.Error("the pinned snippets and the self-host panel should cover code-editor's vendored Prism")
	}
	for i, file := range []string{"code-editor.bundle.min.js", "code-editor.min.js"} {
		if !strings.Contains(p["self-host"][i], `src="/js/code-editor/`+file+`"`) {
			t.Errorf("self-host panel:\n%s", p["self-host"][i])
		}
	}
}

func TestSetInstallTabValidates(t *testing.T) {
	for _, tc := range []struct {
		cmd commands.SetInstallTab
		ok  bool
	}{
		{commands.SetInstallTab{SID: "s", Tab: "autoloader"}, true},
		{commands.SetInstallTab{SID: "s", Tab: "component"}, true},
		{commands.SetInstallTab{SID: "s", Tab: "pinned"}, true},
		{commands.SetInstallTab{SID: "s", Tab: "self-host"}, true},
		{commands.SetInstallTab{SID: "s", Tab: "1"}, false},
		{commands.SetInstallTab{SID: "s", Tab: ""}, false},
		{commands.SetInstallTab{SID: "", Tab: "pinned"}, false},
	} {
		if err := tc.cmd.Validate(); (err == nil) != tc.ok {
			t.Errorf("%+v: %v", tc.cmd, err)
		}
	}
}

// The chosen tab is a session preference: the first GET of any component
// page renders it, and another session doesn't see it.
func TestInstallTabRemembered(t *testing.T) {
	ts, c := newServer(t)
	selected := func(c *http.Client, path string) string {
		_, page := get(t, c, ts.URL+path)
		m := selectedRe.FindStringSubmatch(page)
		if m == nil {
			t.Fatalf("%s: no selected tab", path)
		}
		return m[1]
	}
	if s := selected(c, "/components/button"); s != "0" {
		t.Fatalf("default tab = %s, want 0 (autoloader)", s)
	}
	for body, want := range map[string]int{`{"tab":"nope"}`: 400, `{"tab":2}`: 400, `{"tab":"component"}`: 204} {
		if r := post(t, c, ts.URL+"/cmd/install-tab", body, "same-origin"); r.StatusCode != want {
			t.Errorf("%s = %d, want %d", body, r.StatusCode, want)
		}
	}
	deadline := time.Now().Add(3 * time.Second)
	for selected(c, "/components/button") != "1" {
		if time.Now().After(deadline) {
			t.Fatal("the stored tab is not rendered as selected")
		}
		time.Sleep(20 * time.Millisecond)
	}
	if s := selected(c, "/components/slider"); s != "1" {
		t.Errorf("another component page: selected %s, want 1", s)
	}
	jar, _ := cookiejar.New(nil)
	if s := selected(&http.Client{Jar: jar}, "/components/slider"); s != "0" {
		t.Errorf("another session sees selected %s, want 0", s)
	}
}

// Every file a pinned tab loads is covered by integrity: a script tag's
// integrity covers only that file, so what a module imports itself (Prism,
// ECharts) must be in the import map's integrity, and every hash must match
// the bytes served. Already-minified vendored files are their own .min.
func TestInstallIntegrityCoversImports(t *testing.T) {
	ts, c, bus, _ := newServerBus(t)
	_ = bus
	for slug, imported := range map[string]string{"code-editor": "vendor/prism.min.js", "echarts": "vendor/echarts.esm.min.js"} {
		_, page := get(t, c, ts.URL+"/components/"+slug)
		p := installPanels(t, page)
		for _, snippet := range [][2]string{{"this-component", p["this-component"][0]}, {"this-component modules", p["this-component"][1]}, {"pinned", p["pinned"][0]}} {
			tab := snippet[0]
			m := integrityRe.FindStringSubmatch(snippet[1])
			if m == nil {
				t.Errorf("%s %s: no integrity map:\n%s", slug, tab, snippet[1])
				continue
			}
			var entries map[string]string
			if err := json.Unmarshal([]byte(m[1]), &entries); err != nil {
				t.Fatalf("%s %s: %v", slug, tab, err)
			}
			found := false
			for u, sri := range entries {
				if strings.HasSuffix(u, "/"+imported) {
					found = true
				}
				if got := servedSRI(t, c, strings.Replace(u, "http://localhost:7331", ts.URL, 1)); got != sri {
					t.Errorf("%s %s: %s integrity %s, served %s", slug, tab, u, sri, got)
				}
			}
			if !found {
				t.Errorf("%s %s: %s is not pinned", slug, tab, imported)
			}
		}
		for tab, snippets := range p {
			for _, snippet := range snippets {
				if strings.Contains(snippet, ".min.min.") {
					t.Errorf("%s %s mentions a file that does not exist (.min.min):\n%s", slug, tab, snippet)
				}
			}
		}
		if strings.Contains(page, ".min.min.") {
			t.Errorf("%s: the page links a .min.min file", slug)
		}
	}
}
