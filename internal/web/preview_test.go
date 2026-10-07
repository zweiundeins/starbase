package web

import (
	"context"
	"errors"
	"log/slog"
	"maps"
	"net/http"
	"net/http/httptest"
	"regexp"
	"slices"
	"strings"
	"sync/atomic"
	"testing"

	"starbase/internal/catalog"
	"starbase/internal/tscheck"
	"starbase/internal/ui"
)

func TestPreviewCache(t *testing.T) {
	const commit = "bc0016f744fbcb99b0e2d36c4c5b5d58688ac6f2"
	const forkCommit = "0000000000000000000000000000000000000001"
	var hits atomic.Int32
	raw := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		// Files exist at every commit: only the submission check stops forks.
		path := regexp.MustCompile(`^/o/r/[0-9a-f]{40}/`).ReplaceAllString(r.URL.Path, "/o/r/"+commit+"/")
		switch path {
		case "/api/o/r/commits/" + commit + "/pulls":
			hits.Add(-1) // not a file fetch
			w.Write([]byte(`[{"head": {"ref": "component/nebula", "repo": {"full_name": "o/r"}}}]`))
		case "/api/o/r/commits/" + forkCommit + "/pulls":
			hits.Add(-1)
			w.Write([]byte(`[{"head": {"ref": "component/nebula", "repo": {"full_name": "evil/r"}}}]`))
		case "/o/r/" + commit + "/components/nebula/nebula.js":
			w.Write([]byte("rocket('sb-nebula', {})"))
		case "/o/r/" + commit + "/components/nebula/vendor/lib.js":
			w.Write([]byte("export const x = 1"))
		case "/o/r/" + commit + "/components/nebula/README.md":
			w.Write([]byte("---\nname: Nebula\n---\n\n```html preview\n<sb-nebula></sb-nebula>\n```\n"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer raw.Close()
	c := newPreviewCache("https://github.com/o/r", "")
	if c.raw != "https://raw.githubusercontent.com/o/r" || c.api != "https://api.github.com/repos/o/r" {
		t.Fatalf("raw = %s, api = %s", c.raw, c.api)
	}
	c.raw, c.api = raw.URL+"/o/r", raw.URL+"/api/o/r"

	for range 2 {
		p, err := c.get(context.Background(), commit+"/nebula")
		if err != nil {
			t.Fatal(err)
		}
		if p.Name != "Nebula" || p.Files["component.js"] != "rocket('sb-nebula', {})" || strings.TrimSpace(p.Files["index.html"]) != "<sb-nebula></sb-nebula>" {
			t.Fatalf("preview = %+v", p)
		}
	}
	if hits.Load() != 3 { // before the file requests below
		t.Errorf("fetched %d files, want 3: nebula.ts (missing), nebula.js and README.md (the second get is cached)", hits.Load())
	}
	srv := &Server{previews: c}
	for _, tc := range []struct {
		file string
		code int
	}{{"vendor/lib.js", 200}, {"vendor/missing.js", 404}, {"README.md", 404}, {"../x.js", 404}} {
		req := httptest.NewRequest("GET", "/", nil)
		req.SetPathValue("commit", commit)
		req.SetPathValue("slug", "nebula")
		req.SetPathValue("file", tc.file)
		rec := httptest.NewRecorder()
		srv.servePreviewFile(rec, req)
		if rec.Code != tc.code {
			t.Errorf("%s: status %d, want %d", tc.file, rec.Code, tc.code)
		}
		if tc.code == 200 && (rec.Body.String() != "export const x = 1" || rec.Header().Get("Cache-Control") != immutable) {
			t.Errorf("%s: body %q, headers %v", tc.file, rec.Body, rec.Header())
		}
	}

	// A commit from a fork's pull request is not a submission.
	if _, err := c.get(context.Background(), forkCommit+"/nebula"); !errors.Is(err, errNoPreview) {
		t.Errorf("fork commit: err = %v, want errNoPreview", err)
	}
	req := httptest.NewRequest("GET", "/", nil)
	req.SetPathValue("commit", forkCommit)
	req.SetPathValue("slug", "nebula")
	req.SetPathValue("file", "vendor/lib.js")
	rec := httptest.NewRecorder()
	srv.servePreviewFile(rec, req)
	if rec.Code != 404 {
		t.Errorf("fork commit file: status %d, want 404", rec.Code)
	}

	for _, ref := range []string{commit + "/missing", "main/nebula", commit + "/../x", commit} {
		if _, err := c.get(context.Background(), ref); !errors.Is(err, errNoPreview) {
			t.Errorf("%s: err = %v, want errNoPreview", ref, err)
		}
	}
}

// A component written in TypeScript opens as its source, like ?component=:
// the runner transpiles it, and its imports of ./x.ts load the committed x.js
// through the proxy. The type check reads the modules it imports at that commit.
func TestPreviewTypeScript(t *testing.T) {
	const commit = "22691c39452cf9cc73b872f9a7b4005097c036f7"
	const ref = commit + "/comet"
	const main = "import { orbit } from './core/orbit.ts'\nimport type { Tail } from './contracts/comet.ts'\n\n" +
		"export const tail: Tail = { length: orbit(2) }\nexport const dust = () => import('./vendor/dust.js')\n"
	files := map[string]string{
		"comet.ts":           main,
		"comet.js":           "// Generated from comet.ts by `go tool task ts`: edit the TypeScript, not this file.\nimport { orbit } from './core/orbit.js'\n",
		"core/orbit.ts":      "import type { Tail } from '../contracts/comet.ts'\n\nexport const orbit = (n: number): number => n * 2\nexport const none: Tail = { length: 0 }\nexport const later = () => import('./gone.ts')\n",
		"core/orbit.js":      "export const orbit = (n) => n * 2\n",
		"contracts/comet.ts": "export interface Tail {\n\tlength: number\n}\n",
		"vendor/dust.js":     "export default 1\n",
		"README.md":          "---\nname: Comet\n---\n\n```html preview\n<sb-comet></sb-comet>\n```\n",
	}
	var hits atomic.Int32
	raw := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/o/r/commits/"+commit+"/pulls" {
			w.Write([]byte(`[{"head": {"ref": "component/comet", "repo": {"full_name": "o/r"}}}]`))
			return
		}
		hits.Add(1)
		if b, ok := files[strings.TrimPrefix(r.URL.Path, "/o/r/"+commit+"/components/comet/")]; ok {
			w.Write([]byte(b))
			return
		}
		http.NotFound(w, r)
	}))
	defer raw.Close()
	c := newPreviewCache("https://github.com/o/r", "")
	c.raw, c.api = raw.URL+"/o/r", raw.URL+"/api/o/r"

	p, err := c.get(context.Background(), ref)
	if err != nil {
		t.Fatal(err)
	}
	if _, js := p.Files["component.js"]; js || p.Files["component.ts"] != main || p.Name != "Comet" {
		t.Fatalf("files = %v", p.Files)
	}
	srv := &Server{previews: c, catalog: &catalog.Catalog{}, sizes: map[[32]byte]map[string]any{},
		checks: map[[32]byte][]tscheck.Diagnostic{}, log: slog.New(slog.DiscardHandler)}
	req := httptest.NewRequest("GET", "/", nil)
	req.SetPathValue("commit", commit)
	req.SetPathValue("slug", "comet")
	req.SetPathValue("file", "core/orbit.js")
	rec := httptest.NewRecorder()
	srv.servePreviewFile(rec, req)
	if rec.Code != 200 || rec.Body.String() != files["core/orbit.js"] {
		t.Errorf("core/orbit.js: %d %q", rec.Code, rec.Body)
	}

	// The page opens component.ts, and its check names the preview.
	v, err := srv.codePlaygroundPage(&renderCtx{ctx: context.Background(), req: httptest.NewRequest("GET", "/playground?preview="+ref, nil)})
	if err != nil {
		t.Fatal(err)
	}
	var page strings.Builder
	v.Body(ui.Shell{}).Render(context.Background(), &page)
	if !strings.Contains(page.String(), `{&#34;component.ts&#34;:`) || !strings.Contains(page.String(), `preview: &#34;`+ref+`&#34;`) {
		t.Errorf("page:\n%s", page.String())
	}

	// The modules its TypeScript reaches, once: ./gone.ts is missing, and the module itself is component.ts.
	before := hits.Load()
	for range 2 {
		mods, err := c.modules(context.Background(), ref)
		if err != nil {
			t.Fatal(err)
		}
		if got := slices.Sorted(maps.Keys(mods)); !slices.Equal(got, []string{"contracts/comet.ts", "core/orbit.ts", "vendor/dust.js"}) {
			t.Errorf("modules = %v", got)
		}
	}
	if n := hits.Load() - before; n != 4 {
		t.Errorf("fetched %d modules, want 4 (with the missing core/gone.ts; the second call is cached)", n)
	}
	if mods, err := c.modules(context.Background(), "0000000000000000000000000000000000000000/comet"); err != nil || mods != nil {
		t.Errorf("no preview: %v, %v", mods, err)
	}

	if srv.checker = tscheck.New(t.TempDir()); srv.checker == nil {
		t.Skip("no compiler embedded: go run ./cmd/fetchtsc")
	}
	if ds := srv.typecheck(context.Background(), "", ref, "component.ts", main); len(ds) != 0 {
		t.Errorf("the preview's TypeScript: %+v", ds)
	}
	if ds := srv.typecheck(context.Background(), "", ref, "component.ts", strings.Replace(main, "orbit(2)", "orbit('2')", 1)); len(ds) != 1 || ds[0].Code != 2345 {
		t.Errorf("a helper's types: %+v", ds)
	}
	if ds := srv.typecheck(context.Background(), "", "", "component.ts", main); len(ds) == 0 || ds[0].Code != 2307 {
		t.Errorf("without the preview, its imports are not found: %+v", ds)
	}
}
