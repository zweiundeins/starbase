package app_test

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"errors"
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

//go:embed testdata/editor_drift.js
var editorDriftJS string

// driftRow is one editor at one font size, as editor_drift.js reports it.
type driftRow struct {
	Editor      string            `json:"editor"`
	Size        string            `json:"size"`
	Lines       int               `json:"lines"`
	LineHeight  float64           `json:"lineHeight"`
	Fonts       map[string]string `json:"fonts"`
	CaretDrift  float64           `json:"caretDrift"`
	GutterFirst *float64          `json:"gutterFirst"`
	GutterLast  *float64          `json:"gutterLast"`
	OK          bool              `json:"ok"`
	Error       string            `json:"error"`
}

// TestCodeEditorLinesAlign loads the playground in headless Chrome (like
// cmd/manifests: the app in-process, Chrome quits once the page has loaded)
// and measures that sb-code-editor's highlighted text, its textarea (the
// caret) and its line numbers stay on the same lines from the first line to
// the last, at the default size and at a larger --sb-code-editor-font-size.
// A second font in the highlighted <pre> once made every line 1 px taller
// than the caret's, two rows off after 40 lines.
func TestCodeEditorLinesAlign(t *testing.T) {
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
	defer cancel()
	a, err := app.New(ctx, cfg, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()

	// The playground as the site serves it, plus the measurement. The page's
	// CSP (rightly) forbids an inline script, so this test server drops it.
	// The script posts its rows back; until then an image holds the page's
	// load event, and with it Chrome, which quits once the page has loaded.
	probe := `<img src="/__drift/wait" alt="" hidden><script type="module">` + "\n" + editorDriftJS + `
let out
try { out = JSON.stringify(await sbEditorDrift()) } catch (e) { out = JSON.stringify([{ error: String(e) }]) }
await fetch('/__drift/result', { method: 'POST', body: out })
</script>`
	// One run per page: the result it posted, and the release of its image.
	type run struct {
		results chan []byte
		release chan struct{}
		once    sync.Once
	}
	var (
		mu  sync.Mutex
		cur *run
	)
	current := func() *run { mu.Lock(); defer mu.Unlock(); return cur }
	srv := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/__drift/wait":
			select {
			case <-current().release:
			case <-time.After(60 * time.Second):
			}
			w.WriteHeader(http.StatusNoContent)
		case "/__drift/result":
			body, _ := io.ReadAll(r.Body)
			run := current()
			run.once.Do(func() {
				run.results <- body
				close(run.release)
			})
		case "/playground":
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

	// The starter, and the longest component the playground opens.
	for _, path := range []string{"/playground", "/playground?component=tree"} {
		run := &run{results: make(chan []byte, 1), release: make(chan struct{})}
		mu.Lock()
		cur = run
		mu.Unlock()
		rows, err := measureDrift(chrome, base+path, run.results)
		if errors.Is(err, errNoSandbox) {
			t.Skip("Chrome has no usable sandbox here; set STARBASE_CHROME_NO_SANDBOX=1 to run this test (as for go tool task manifests)")
		}
		if err != nil {
			t.Fatalf("%s: %v", path, err)
		}
		sizes := map[string]bool{}
		for _, r := range rows {
			if r.Error != "" {
				t.Fatalf("%s: %s", path, r.Error)
			}
			sizes[r.Size] = true
			gutter := "no gutter"
			if r.GutterFirst != nil {
				gutter = "gutter " + ftoa(*r.GutterFirst) + "/" + ftoa(*r.GutterLast) + " px"
			}
			t.Logf("%s %s [%s]: %d lines at %s px, caret drift %s px, %s (code font %s)", path, r.Editor, r.Size, r.Lines, ftoa(r.LineHeight), ftoa(r.CaretDrift), gutter, r.Fonts["code"])
			if !r.OK {
				t.Errorf("%s %s [%s]: the highlighted text drifts from the caret or the line numbers over %d lines (caret %s px, %s; code font %q, textarea font %q)",
					path, r.Editor, r.Size, r.Lines, ftoa(r.CaretDrift), gutter, r.Fonts["code"], r.Fonts["textarea"])
			}
		}
		if !sizes["default"] || !sizes["1.25rem"] {
			t.Errorf("%s: no editor measured at both sizes (got %v)", path, sizes)
		}
	}
}

var errNoSandbox = errors.New("chrome: no usable sandbox")

// measureDrift opens url in headless Chrome and returns the rows the page posted.
func measureDrift(chrome, url string, results <-chan []byte) ([]driftRow, error) {
	args := []string{"--headless=new", "--disable-gpu", "--window-size=1440,1000", "--dump-dom", url}
	// As in cmd/manifests: Chrome's sandbox can't start as root or without user namespaces.
	if os.Geteuid() == 0 || os.Getenv("STARBASE_CHROME_NO_SANDBOX") == "1" {
		args = append([]string{"--no-sandbox"}, args...)
	}
	cmd := exec.Command(chrome, args...)
	var stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = io.Discard, &stderr
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case err := <-done:
		if err != nil && strings.Contains(stderr.String(), "No usable sandbox") {
			return nil, errNoSandbox
		}
		if err != nil {
			return nil, errors.New("chrome: " + err.Error() + "\n" + stderr.String())
		}
	case <-time.After(90 * time.Second):
		cmd.Process.Kill()
		return nil, errors.New("chrome timed out")
	}
	var body []byte
	select {
	case body = <-results:
	default:
		return nil, errors.New("the page posted no measurement (did the playground load?)\n" + stderr.String())
	}
	var rows []driftRow
	if err := json.Unmarshal(body, &rows); err != nil {
		return nil, err
	}
	return rows, nil
}

func ftoa(f float64) string {
	b, _ := json.Marshal(f)
	return string(b)
}
