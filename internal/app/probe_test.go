package app_test

import (
	"context"
	"io"
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
	chrome := findChrome(t)
	a, base, ln := startApp(t, host)
	var extra map[string]http.HandlerFunc
	if handlers != nil {
		extra = handlers(a.Handler)
	}

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
				a.Handler.ServeHTTP(w, r)
				return
			}
			r.Header.Del("Accept-Encoding")
			rec := httptest.NewRecorder()
			a.Handler.ServeHTTP(rec, r)
			for k, v := range rec.Header() {
				if k != "Content-Security-Policy" && k != "Content-Length" {
					w.Header()[k] = v
				}
			}
			w.WriteHeader(rec.Code)
			doc := strings.Replace(rec.Body.String(), "</body>", inject+"</body>", 1)
			w.Write([]byte(doc))
		default:
			if h := extra[r.URL.Path]; h != nil {
				h(w, r)
				return
			}
			a.Handler.ServeHTTP(w, r)
		}
	})}
	go srv.Serve(ln)
	t.Cleanup(func() { srv.Close() })
	if host != "" {
		// HttpsUpgrades would try https://<host> first.
		chromeArgs = append([]string{"--host-resolver-rules=MAP " + host + " 127.0.0.1", "--disable-features=HttpsUpgrades"}, chromeArgs...)
	}
	return a, runChrome(t, chrome, base+path, results, chromeArgs...)
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
