package app_test

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"starbase/components"
	"starbase/internal/app"
	"starbase/internal/config"
)

// TestShrunkModulesKeepProps loads the homepage, which runs every component
// from the bundle (built from shrunk sources: catalog/shrink.go leaves out
// .docs() and the manifest: block), and checks that each component still
// defines exactly the props its manifest.json lists — names, attributes,
// types, defaults and values — only without the docs. A shrink that cut into
// a prop chain would show up here.
func TestShrunkModulesKeepProps(t *testing.T) {
	chrome := findChrome(t)
	a, base, ln := startApp(t)

	var once sync.Once
	results := make(chan []byte, 1)
	release := make(chan struct{})
	probe := `<img src="/__probe/wait" alt="" hidden><script type="module">
let out
try {
	const all = %TAGS%
	await Promise.race([Promise.all(all.map((t) => customElements.whenDefined(t))), new Promise((_, no) => setTimeout(() => no(new Error('not all tags were defined: ' + all.filter((t) => !customElements.get(t)))), 20000))])
	out = JSON.stringify(Object.fromEntries(all.map((t) => [t, customElements.get(t).manifest()])))
} catch (e) { out = JSON.stringify({ error: String(e) }) }
await fetch('/__probe/result', { method: 'POST', body: out })
</script>`
	var tags []string
	cat := a.Catalog
	for _, c := range cat.Components {
		tags = append(tags, c.Tag)
	}
	tj, _ := json.Marshal(tags)
	probe = strings.Replace(probe, "%TAGS%", string(tj), 1)
	srv := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/__probe/wait":
			select {
			case <-release:
			case <-time.After(60 * time.Second):
			}
			w.WriteHeader(http.StatusNoContent)
		case "/__probe/result":
			body, _ := io.ReadAll(r.Body)
			once.Do(func() { results <- body; close(release) })
		case "/":
			r.Header.Del("Accept-Encoding")
			rec := httptest.NewRecorder()
			a.Handler.ServeHTTP(rec, r)
			for k, v := range rec.Header() {
				if k != "Content-Security-Policy" && k != "Content-Length" {
					w.Header()[k] = v
				}
			}
			w.WriteHeader(rec.Code)
			w.Write([]byte(strings.Replace(rec.Body.String(), "</body>", probe+"</body>", 1)))
		default:
			a.Handler.ServeHTTP(w, r)
		}
	})}
	go srv.Serve(ln)
	defer srv.Close()

	body := runChrome(t, chrome, base+"/", results)
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

func findChrome(t *testing.T) string {
	t.Helper()
	chrome := os.Getenv("CHROME")
	for _, name := range []string{"google-chrome", "google-chrome-stable", "chromium", "chromium-browser", "chrome"} {
		if chrome != "" {
			break
		}
		chrome, _ = exec.LookPath(name)
	}
	if chrome == "" {
		t.Skip("no Chrome/Chromium found (set CHROME)")
	}
	return chrome
}

// startApp runs the app in-process on a free port; the caller serves ln.
func startApp(t *testing.T) (*app.App, string, net.Listener) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	base := "http://" + ln.Addr().String()
	cfg := config.Load()
	cfg.BaseURL = base
	cfg.DBPath = filepath.Join(t.TempDir(), "db.sqlite")
	cfg.GitHubClientID, cfg.GitHubClientSecret = "", ""
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	a, err := app.New(ctx, cfg, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { a.Close() })
	return a, base, ln
}

// runChrome loads url in headless Chrome and returns what the page posted.
func runChrome(t *testing.T, chrome, url string, results <-chan []byte) []byte {
	t.Helper()
	args := []string{"--headless=new", "--disable-gpu", "--window-size=1440,1000", "--dump-dom", url}
	if os.Geteuid() == 0 || os.Getenv("STARBASE_CHROME_NO_SANDBOX") == "1" {
		args = append([]string{"--no-sandbox"}, args...)
	}
	cmd := exec.Command(chrome, args...)
	var stderr strings.Builder
	cmd.Stdout, cmd.Stderr = io.Discard, &stderr
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case err := <-done:
		if err != nil && strings.Contains(stderr.String(), "No usable sandbox") {
			t.Skip("Chrome has no usable sandbox here; set STARBASE_CHROME_NO_SANDBOX=1 to run this test (as for go tool task manifests)")
		}
		if err != nil {
			t.Fatalf("chrome: %v\n%s", err, stderr.String())
		}
	case <-time.After(90 * time.Second):
		cmd.Process.Kill()
		t.Fatal("chrome timed out")
	}
	select {
	case b := <-results:
		return b
	default:
		t.Fatalf("the page posted nothing\n%s", stderr.String())
	}
	return nil
}
