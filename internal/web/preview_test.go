package web

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

func TestPreviewCache(t *testing.T) {
	const commit = "bc0016f744fbcb99b0e2d36c4c5b5d58688ac6f2"
	var hits atomic.Int32
	raw := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		switch r.URL.Path {
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
	c := newPreviewCache("https://github.com/o/r")
	if c.raw != "https://raw.githubusercontent.com/o/r" {
		t.Fatalf("raw = %s", c.raw)
	}
	c.raw = raw.URL + "/o/r"

	for range 2 {
		p, err := c.get(context.Background(), commit+"/nebula")
		if err != nil {
			t.Fatal(err)
		}
		if p.Name != "Nebula" || p.Files["component.js"] != "rocket('sb-nebula', {})" || strings.TrimSpace(p.Files["index.html"]) != "<sb-nebula></sb-nebula>" {
			t.Fatalf("preview = %+v", p)
		}
	}
	if hits.Load() != 2 { // before the file requests below
		t.Errorf("fetched %d files, want 2 (the second get is cached)", hits.Load())
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

	for _, ref := range []string{commit + "/missing", "main/nebula", commit + "/../x", commit} {
		if _, err := c.get(context.Background(), ref); !errors.Is(err, errNoPreview) {
			t.Errorf("%s: err = %v, want errNoPreview", ref, err)
		}
	}
}
