package web_test

import (
	"bufio"
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"starbase/components"
	"starbase/content"
	"starbase/internal/catalog"
	"starbase/internal/commands"
	"starbase/internal/config"
	"starbase/internal/cqrs"
	"starbase/internal/db"
	"starbase/internal/queries"
	"starbase/internal/web"
	"starbase/static"
)

func newServer(t *testing.T) (*httptest.Server, *http.Client) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	d, err := db.Open(ctx, filepath.Join(t.TempDir(), "web.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { d.Close() })
	cat, err := catalog.Load(components.FS)
	if err != nil {
		t.Fatal(err)
	}
	hub := cqrs.NewHub()
	bus := cqrs.NewBus(d.W, hub, log)
	go bus.Run(ctx)
	if err := bus.Exec(ctx, commands.SyncCatalog{Catalog: cat}); err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewUnstartedServer(nil)
	ts.Start()
	cfg := config.Config{BaseURL: ts.URL, RepoURL: "https://example.com/repo"}
	srv := web.New(ctx, web.Deps{Config: cfg, Log: log, Bus: bus, Hub: hub, Queries: queries.New(d.R),
		Catalog: cat, StaticFS: static.FS, Content: content.FS})
	ts.Config.Handler = srv.Handler()
	t.Cleanup(func() { cancel(); ts.Close() })
	jar, _ := cookiejar.New(nil)
	return ts, &http.Client{Jar: jar}
}

func get(t *testing.T, c *http.Client, url string) (*http.Response, string) {
	t.Helper()
	res, err := c.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	b, _ := io.ReadAll(res.Body)
	return res, string(b)
}

func post(t *testing.T, c *http.Client, url, body, site string) *http.Response {
	t.Helper()
	req, _ := http.NewRequest(http.MethodPost, url, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Datastar-Request", "true")
	if site != "" {
		req.Header.Set("Sec-Fetch-Site", site)
	}
	res, err := c.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	return res
}

func TestPages(t *testing.T) {
	ts, c := newServer(t)
	for path, want := range map[string]int{
		"/":                   200,
		"/?cat=forms&q=but":   200,
		"/components/button":  200,
		"/components/missing": 404,
		"/themes":             200,
		"/contribute":         200,
		"/about":              200,
		"/showcase":           200,
		"/does/not/exist":     404,
		"/c/index.js":         200,
		"/c/button/button.js": 200,
		"/c/button/README.md": 404, // only the component module is public
		"/art/hero.svg":       200,
	} {
		res, _ := get(t, c, ts.URL+path)
		if res.StatusCode != want {
			t.Errorf("GET %s = %d, want %d", path, res.StatusCode, want)
		}
	}
}

func TestDocumentHasNonceAndStream(t *testing.T) {
	ts, c := newServer(t)
	res, body := get(t, c, ts.URL+"/")
	csp := res.Header.Get("Content-Security-Policy")
	m := regexp.MustCompile(`'nonce-([^']+)'`).FindStringSubmatch(csp)
	if m == nil {
		t.Fatalf("CSP without nonce: %q", csp)
	}
	if !strings.Contains(body, `<script type="importmap" nonce="`+m[1]+`">`) {
		t.Error("import map does not carry the CSP nonce")
	}
	if !strings.Contains(body, `data-init="@post(location.pathname + location.search`) {
		t.Error("page does not open its render stream")
	}
	if strings.Contains(body, "/dev/reload") {
		t.Error("dev reload leaked into a non-dev build")
	}
}

func TestCommandsRequireSameOrigin(t *testing.T) {
	ts, c := newServer(t)
	get(t, c, ts.URL+"/") // obtain a session cookie
	body := `{"tabid":"tab12345","q":"modal"}`
	if res := post(t, c, ts.URL+"/cmd/browse", body, "cross-site"); res.StatusCode != http.StatusForbidden {
		t.Errorf("cross-site = %d, want 403", res.StatusCode)
	}
	if res := post(t, c, ts.URL+"/cmd/browse", body, "same-origin"); res.StatusCode != http.StatusNoContent {
		t.Errorf("same-origin = %d, want 204", res.StatusCode)
	}
	if res := post(t, c, ts.URL+"/cmd/browse", `{"tabid":"x"}`, "same-origin"); res.StatusCode != http.StatusBadRequest {
		t.Errorf("bad tab = %d, want 400", res.StatusCode)
	}
}

// TestRenderStream drives the full CQRS loop: open a tab's stream, send a
// command, and receive a new frame that reflects it.
func TestRenderStream(t *testing.T) {
	ts, c := newServer(t)
	get(t, c, ts.URL+"/")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, ts.URL+"/", strings.NewReader(`{"tabid":"tab12345"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.Header.Set("Accept-Encoding", "identity")
	res, err := c.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if ct := res.Header.Get("Content-Type"); !strings.HasPrefix(ct, "text/event-stream") {
		t.Fatalf("content type = %q", ct)
	}
	frames := make(chan string)
	go func() {
		sc := bufio.NewScanner(res.Body)
		sc.Buffer(make([]byte, 1<<20), 8<<20)
		var frame strings.Builder
		for sc.Scan() {
			if sc.Text() == "" && frame.Len() > 0 {
				frames <- frame.String()
				frame.Reset()
				continue
			}
			frame.WriteString(sc.Text() + "\n")
		}
		close(frames)
	}()
	next := func() string {
		select {
		case f, ok := <-frames:
			if !ok {
				t.Fatal("stream closed")
			}
			return f
		case <-ctx.Done():
			t.Fatal("timed out waiting for a frame")
		}
		return ""
	}

	cat, _ := catalog.Load(components.FS)
	first := next()
	if n := strings.Count(first, `class="card"`); !strings.Contains(first, "event: datastar-patch-elements") || n != len(cat.Components) {
		t.Fatalf("first frame should list all %d cards, got %d", len(cat.Components), n)
	}
	if r := post(t, c, ts.URL+"/cmd/browse", `{"tabid":"tab12345","cat":"feedback","sort":"name"}`, "same-origin"); r.StatusCode != 204 {
		t.Fatalf("command = %d", r.StatusCode)
	}
	second := next()
	if n := strings.Count(second, `class="card"`); n != 3 {
		t.Fatalf("filtered frame has %d cards, want 3", n)
	}
	if !regexp.MustCompile(`replaceState\(null, &#39;&#39;, &#34;/\?cat=feedback\\u0026sort=name&#34;\)`).MatchString(second) {
		i := strings.Index(second, "replaceState")
		t.Errorf("frame does not sync the URL: %s", second[i:min(len(second), i+80)])
	}
}
