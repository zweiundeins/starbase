package web

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"sync/atomic"
	"testing"
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
