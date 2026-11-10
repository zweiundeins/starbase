package app_test

import (
	"context"
	"errors"
	"io"
	"io/fs"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"starbase/internal/app"
	"starbase/internal/config"
)

// probe runs the app in-process, loads path in headless Chrome with script
// (a module; it must end by posting its result to /__probe/result) added to
// the page, and returns what the script posted. The page's CSP (rightly)
// forbids an inline script, so this test server drops it. An image holds
// the page's load event, and with it Chrome (which quits once the page has
// loaded), until the result arrives.
func probe(t *testing.T, path, script string) (*app.App, []byte) {
	t.Helper()
	return probeAt(t, "", path, script)
}

// probeAt is probe from http://<host>:<port>, which Chrome maps to the app, plus chromeArgs.
// A host other than localhost makes a non-secure context; "" is 127.0.0.1, as in probe.
func probeAt(t *testing.T, host, path, script string, chromeArgs ...string) (*app.App, []byte) {
	t.Helper()
	return probeFull(t, host, path, script, nil, chromeArgs...)
}

// probeWith is probe with test-only handlers by path, answered before the app's; they are made
// before Chrome starts, from the app's handler, so they can wait for it or pass requests on.
func probeWith(t *testing.T, path, script string, handlers func(app http.Handler) map[string]http.HandlerFunc) (*app.App, []byte) {
	t.Helper()
	return probeFull(t, "", path, script, handlers)
}

// probeFull is probeAt with probeWith's handlers.
func probeFull(t *testing.T, host, path, script string, handlers func(app http.Handler) map[string]http.HandlerFunc, chromeArgs ...string) (*app.App, []byte) {
	t.Helper()
	return probeRun(t, host, path, script, handlers, nil, chromeArgs...)
}

// probeRun is probeFull with run in place of runChrome (nil: runChrome), given the page's URL
// and the channel of what the script posts.
func probeRun(t *testing.T, host, path, script string, handlers func(app http.Handler) map[string]http.HandlerFunc, run func(url string, results <-chan []byte) []byte, chromeArgs ...string) (*app.App, []byte) {
	t.Helper()
	chrome := ""
	if run == nil {
		chrome = findChrome(t)
	}
	a, base, ln := startApp(t, host)
	var extra map[string]http.HandlerFunc
	if handlers != nil {
		extra = handlers(a.Handler)
	}
	results := serveProbe(t, ln, path, script, a.Handler, extra)
	if host != "" {
		// HttpsUpgrades would try https://<host> first.
		chromeArgs = append([]string{"--host-resolver-rules=MAP " + host + " 127.0.0.1", "--disable-features=HttpsUpgrades"}, chromeArgs...)
	}
	if run != nil {
		return a, run(base+path, results)
	}
	return a, runChrome(t, chrome, base+path, results, chromeArgs...)
}

// serveProbe serves h on ln for Chrome: the page at path with script added,
// the probe's own endpoints, and extra handlers by path before h's. It
// returns the channel that gets what the script posts.
func serveProbe(t *testing.T, ln net.Listener, path, script string, h http.Handler, extra map[string]http.HandlerFunc) <-chan []byte {
	t.Helper()
	var once sync.Once
	results := make(chan []byte, 1)
	release := make(chan struct{})
	inject := `<img src="/__probe/wait" alt="" hidden><script type="module">` + "\n" + script + "\n</script>"
	page, _, _ := strings.Cut(path, "?")
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
		case page:
			if r.Method != http.MethodGet {
				if x := extra[r.Method+" "+page]; x != nil { // a test's own stream handler
					x(w, r)
					return
				}
				h.ServeHTTP(w, r)
				return
			}
			ph := h
			if x := extra[page]; x != nil { // a test's own page, which gets the script too
				ph = x
			}
			r.Header.Del("Accept-Encoding")
			rec := httptest.NewRecorder()
			ph.ServeHTTP(rec, r)
			for k, v := range rec.Header() {
				if k != "Content-Security-Policy" && k != "Content-Length" {
					w.Header()[k] = v
				}
			}
			w.WriteHeader(rec.Code)
			doc := strings.Replace(rec.Body.String(), "</body>", inject+"</body>", 1)
			w.Write([]byte(doc))
		default:
			if f := extra[r.URL.Path]; f != nil {
				f(w, r)
				return
			}
			if strings.HasPrefix(r.URL.Path, "/demo/arrange/") { // a demo's delay is for readers; tests hold answers themselves
				q := r.URL.Query()
				q.Del("delay")
				r.URL.RawQuery = q.Encode()
			}
			h.ServeHTTP(w, r)
		}
	})}
	go srv.Serve(ln)
	t.Cleanup(func() { srv.Close() })
	return results
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

// startApp runs the app in-process on a free port of 127.0.0.1, with its
// base URL on host when one is given; the caller serves ln.
func startApp(t *testing.T, host string) (*app.App, string, net.Listener) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	base := "http://" + ln.Addr().String()
	if host != "" {
		_, port, _ := net.SplitHostPort(ln.Addr().String())
		base = "http://" + net.JoinHostPort(host, port)
	}
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	cfg := appConfig(t, base)
	copySeeded(t, cfg.DBPath)
	a, err := app.New(ctx, cfg, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { a.Close() })
	return a, base, ln
}

// seeded is a database seeded once per test binary, which every test app
// starts from: 100,000 demo stars seeded per app made the package slow
// under -race, and everything running beside it.
var seeded = sync.OnceValues(func() (string, error) {
	dir, err := os.MkdirTemp("", "starbase-seeded-")
	if err != nil {
		return "", err
	}
	seededDir = dir
	cfg := config.Load()
	cfg.DBPath = filepath.Join(dir, "db.sqlite")
	cfg.GitHubClientID, cfg.GitHubClientSecret = "", ""
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	a, err := app.New(ctx, cfg, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		return "", err
	}
	defer a.Close()
	return cfg.DBPath, a.Settle(ctx)
})

var seededDir string

func TestMain(m *testing.M) {
	code := m.Run()
	if seededDir != "" {
		os.RemoveAll(seededDir)
	}
	os.Exit(code)
}

// copySeeded puts a copy of the seeded database at path.
func copySeeded(t *testing.T, path string) {
	t.Helper()
	from, err := seeded()
	if err != nil {
		t.Fatal(err)
	}
	for _, suffix := range []string{"", "-wal"} {
		b, err := os.ReadFile(from + suffix)
		if suffix != "" && errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err == nil {
			err = os.WriteFile(path+suffix, b, 0o600)
		}
		if err != nil {
			t.Fatal(err)
		}
	}
}

// appConfig is the app's configuration for base, with a database of its own.
func appConfig(t *testing.T, base string) config.Config {
	cfg := config.Load()
	cfg.BaseURL = base
	cfg.DBPath = filepath.Join(t.TempDir(), "db.sqlite")
	cfg.GitHubClientID, cfg.GitHubClientSecret = "", ""
	return cfg
}

// runChrome loads url in headless Chrome, with extra flags, and returns what
// the page posted.
func runChrome(t *testing.T, chrome, url string, results <-chan []byte, extra ...string) []byte {
	t.Helper()
	args := append([]string{"--headless=new", "--disable-gpu", "--window-size=1440,1000"}, extra...)
	args = append(args, "--dump-dom", url)
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
