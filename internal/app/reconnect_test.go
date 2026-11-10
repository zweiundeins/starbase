package app_test

import (
	"context"
	"io"
	"log/slog"
	"maps"
	"net"
	"net/http"
	"net/http/httptest"
	"slices"
	"sync"
	"testing"
	"time"

	"starbase/internal/app"
	"starbase/internal/config"
)

// TestStreamsReconnectAfterDeploy: a deploy restarts the app under an open
// Showcase. Its render stream and Mission Control's telemetry end with
// sse.Reconnect, the copies of their elements ask again with retry 'error'
// through the proxy's 502s, and the next process, on the same database,
// serves them: a command sent after the restart shows on the page, and the
// telemetry moves again.
func TestStreamsReconnectAfterDeploy(t *testing.T) {
	chrome := findChrome(t)
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	base := "http://" + ln.Addr().String()
	d := &deploy{cfg: appConfig(t, base), refused: map[string]int{}}
	copySeeded(t, d.cfg.DBPath) // a fresh database's seeding holds the first command for seconds under -race
	if err := d.start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(d.close)
	results := serveProbe(t, ln, "/showcase", reconnectJS, d, map[string]http.HandlerFunc{"/__probe/restart": d.restart})
	got := string(runChrome(t, chrome, base+"/showcase", results))

	d.mu.Lock()
	refused, err := maps.Clone(d.refused), d.err
	d.mu.Unlock()
	if err != nil {
		t.Fatal(err)
	}
	if want := "moving, 30%, restart, 70%, moving, new elements"; got != want {
		t.Fatalf("the page shows %q, want %q", got, want)
	}
	if refused["POST /showcase"] == 0 || refused["GET /demo/telemetry"] == 0 {
		t.Errorf("the streams met no 502 between the processes: %v", refused)
	}
}

const reconnectJS = `
const until = async (f, ms) => {
	for (const end = Date.now() + ms; Date.now() < end; await new Promise((r) => setTimeout(r, 50))) {
		const v = f()
		if (v) return v
	}
	return ''
}
// Mission Control's clock follows the telemetry, four times a second. Once
// it moves, Datastar has wired the page.
const clock = () => document.querySelector('.mission__clock')?.textContent ?? ''
const moving = async () => {
	const c = clock()
	return (await until(() => clock() !== c, 15000)) ? 'moving' : 'still'
}
// The command's outcome comes back on the render stream.
const thrust = () => [...document.querySelectorAll('.flight__state dd')].map((d) => d.textContent).find((t) => t.endsWith('%')) ?? ''
const command = async (value, ms) => {
	document.querySelector('sb-slider[name="thrust"]').dispatchEvent(new CustomEvent('sb-change', { detail: { name: 'thrust', value }, bubbles: true, composed: true }))
	return (await until(() => thrust() === value + '%' && thrust(), ms)) || thrust() || 'nothing'
}
const out = [await moving(), await command(30, 10000)]
const stream = document.getElementById('stream'), telemetry = document.getElementById('telemetry')
const restart = await fetch('/__probe/restart', { method: 'POST' })
out.push(restart.ok ? 'restart' : 'restart failed')
out.push(await command(70, 20000), await moving())
const copied = document.getElementById('stream') !== stream && document.getElementById('telemetry') !== telemetry
out.push(copied ? 'new elements' : 'same elements')
await fetch('/__probe/result', { method: 'POST', body: out.join(', ') })
`

// deploy runs the app as cmd/starbase does, one process after another on the
// same database, and serves it as the proxy in front of it does: the current
// process, or 502 while none runs.
type deploy struct {
	cfg config.Config

	mu      sync.Mutex
	app     *app.App
	stop    context.CancelFunc // the process's SIGTERM
	serving sync.WaitGroup     // the requests the process answers
	refused map[string]int     // the 502s, by method and path
	err     error
}

// start runs the next process.
func (d *deploy) start() error {
	ctx, stop := context.WithCancel(context.Background())
	a, err := app.New(ctx, d.cfg, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		stop()
		return err
	}
	d.mu.Lock()
	d.app, d.stop = a, stop
	d.mu.Unlock()
	return nil
}

// close ends the process as SIGTERM does: its context ends the streams, its
// HTTP server waits for the requests under way, and Close closes the database.
func (d *deploy) close() {
	d.mu.Lock()
	a, stop := d.app, d.stop
	d.app = nil
	d.mu.Unlock()
	if a == nil {
		return
	}
	stop()
	d.serving.Wait()
	a.Close()
}

func (d *deploy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	d.mu.Lock()
	a := d.app
	if a == nil {
		d.refused[r.Method+" "+r.URL.Path]++
		d.mu.Unlock()
		http.Error(w, "no process", http.StatusBadGateway)
		return
	}
	d.serving.Add(1)
	d.mu.Unlock()
	defer d.serving.Done()
	a.Handler.ServeHTTP(w, r)
}

// restart is the deploy. The next process starts once both streams have met
// a 502, so that the test sees retry 'error' carry them through one.
func (d *deploy) restart(w http.ResponseWriter, r *http.Request) {
	d.close()
	for end := time.Now().Add(10 * time.Second); time.Now().Before(end) && !d.bothRefused(); {
		time.Sleep(20 * time.Millisecond)
	}
	if err := d.start(); err != nil {
		d.mu.Lock()
		d.err = err
		d.mu.Unlock()
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (d *deploy) bothRefused() bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.refused["POST /showcase"] > 0 && d.refused["GET /demo/telemetry"] > 0
}

// TestRefusedStreamStaysClosed: a 404 page's stream asks once. The server
// refuses it for good with 204, and retry 'error', which repeats every 4xx,
// stops there.
func TestRefusedStreamStaysClosed(t *testing.T) {
	chrome := findChrome(t)
	a, base, ln := startApp(t, "")
	var mu sync.Mutex
	var codes []int // the answers to the page's stream
	count := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			a.Handler.ServeHTTP(w, r)
			return
		}
		rec := httptest.NewRecorder()
		a.Handler.ServeHTTP(rec, r)
		mu.Lock()
		codes = append(codes, rec.Code)
		mu.Unlock()
		maps.Copy(w.Header(), rec.Header())
		w.WriteHeader(rec.Code)
		w.Write(rec.Body.Bytes())
	})
	results := serveProbe(t, ln, "/no-such-page", refusedJS, count, nil)
	got := string(runChrome(t, chrome, base+"/no-such-page", results))
	mu.Lock()
	defer mu.Unlock()
	if got != "1" || !slices.Equal(codes, []int{http.StatusNoContent}) {
		t.Fatalf("the page asked %s times, the server answered %v; want once, 204", got, codes)
	}
}

// refusedJS waits for the stream's first answer, then for as long as retry
// 'error' would take to ask three more times (after 400, 800 and 1600 ms).
const refusedJS = `
const asked = () => performance.getEntriesByType('resource').filter((e) => e.initiatorType === 'fetch' && new URL(e.name).pathname === location.pathname).length
for (const end = Date.now() + 15000; Date.now() < end && asked() === 0; ) await new Promise((r) => setTimeout(r, 50))
await new Promise((r) => setTimeout(r, 3000))
await fetch('/__probe/result', { method: 'POST', body: String(asked()) })
`
