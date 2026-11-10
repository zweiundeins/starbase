package app_test

import (
	"context"
	"io"
	"log/slog"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"starbase/internal/app"
)

// TestWatchdogReopensASilentStream: a render stream whose connection drops
// without closing (no FIN or RST) goes quiet, its heartbeats ($_ping) stop,
// and the page's watchdog re-creates #stream after three missed ones. The
// new stream carries the next command's outcome. A stream that keeps pinging
// is left alone.
func TestWatchdogReopensASilentStream(t *testing.T) {
	got, n := runWatchdog(t, 300*time.Millisecond, 0, watchdogJS)
	if want := "steady, reopened, 30%"; got != want || n != 2 {
		t.Fatalf("the page shows %q after %d streams, want %q after 2", got, n, want)
	}
}

// TestWatchdogKeepsAReconnectAfterAnOutage: the stream the watchdog reopens
// answers only after two and a half heartbeats, as after an outage, so the
// counter is one tick short of firing when it opens. Its first frame resets
// the counter, and the stream lives on until its first ping and after.
func TestWatchdogKeepsAReconnectAfterAnOutage(t *testing.T) {
	const beat = 600 * time.Millisecond
	got, n := runWatchdog(t, beat, beat*5/2, outageJS)
	if want := "steady, reopened, kept, 30%"; got != want || n != 2 {
		t.Fatalf("the page shows %q after %d streams, want %q after 2", got, n, want)
	}
}

// runWatchdog serves /showcase with heartbeat, runs script in Chrome and
// returns what it reports and how many render streams opened. POST
// /__probe/stall silences the newest stream; the streams after the first
// answer after delay.
func runWatchdog(t *testing.T, heartbeat, delay time.Duration, script string) (string, int) {
	t.Helper()
	chrome := findChrome(t)
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	base := "http://" + ln.Addr().String()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	cfg := appConfig(t, base)
	cfg.Heartbeat = heartbeat
	copySeeded(t, cfg.DBPath)
	a, err := app.New(ctx, cfg, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { a.Close() })

	var mu sync.Mutex
	var streams []*silencer // the page's render streams, in the order they opened
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.Header.Get("Accept") != "text/event-stream" {
			a.Handler.ServeHTTP(w, r)
			return
		}
		s := &silencer{ResponseWriter: w}
		mu.Lock()
		streams = append(streams, s)
		later := len(streams) > 1
		mu.Unlock()
		if later {
			time.Sleep(delay)
		}
		a.Handler.ServeHTTP(s, r)
	})
	stall := func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		streams[len(streams)-1].silent.Store(true)
		mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
	}
	results := serveProbe(t, ln, "/showcase", script, h, map[string]http.HandlerFunc{"/__probe/stall": stall})
	got := string(runChrome(t, chrome, base+"/showcase", results))
	mu.Lock()
	defer mu.Unlock()
	return got, len(streams)
}

// silencer passes a stream on until it goes silent, and then swallows it, as
// a connection that dropped without closing would.
type silencer struct {
	http.ResponseWriter
	silent atomic.Bool
}

func (s *silencer) Write(b []byte) (int, error) {
	if s.silent.Load() {
		return len(b), nil
	}
	return s.ResponseWriter.Write(b)
}

func (s *silencer) Unwrap() http.ResponseWriter { return s.ResponseWriter }

// watchdogPrelude waits for the page's first frame and defines the probe's helpers.
const watchdogPrelude = `
const until = async (f, ms) => {
	for (const end = Date.now() + ms; Date.now() < end; await new Promise((r) => setTimeout(r, 50))) {
		const v = f()
		if (v) return v
	}
	return ''
}
const clock = () => document.querySelector('.mission__clock')?.textContent ?? ''
const c = clock()
await until(() => clock() !== c, 15000)
const thrust = () => [...document.querySelectorAll('.flight__state dd')].map((d) => d.textContent).find((t) => t.endsWith('%')) ?? ''
const command = async (value, ms) => {
	document.querySelector('sb-slider[name="thrust"]').dispatchEvent(new CustomEvent('sb-change', { detail: { name: 'thrust', value }, bubbles: true, composed: true }))
	return (await until(() => thrust() === value + '%' && thrust(), ms)) || thrust() || 'nothing'
}
const out = []
const first = document.getElementById('stream')
`

const watchdogJS = watchdogPrelude + `
// Six heartbeats with pings: the watchdog keeps still.
await new Promise((r) => setTimeout(r, 1800))
out.push(document.getElementById('stream') === first ? 'steady' : 'reopened too early')
await fetch('/__probe/stall', { method: 'POST' })
out.push((await until(() => document.getElementById('stream') !== first, 10000)) ? 'reopened' : 'still silent')
out.push(await command(30, 10000))
await fetch('/__probe/result', { method: 'POST', body: out.join(', ') })
`

const outageJS = watchdogPrelude + `
await new Promise((r) => setTimeout(r, 1800))
out.push(document.getElementById('stream') === first ? 'steady' : 'reopened too early')
await fetch('/__probe/stall', { method: 'POST' })
const second = await until(() => document.getElementById('stream') !== first && document.getElementById('stream'), 10000)
out.push(second ? 'reopened' : 'still silent')
// The reconnect answers 1.5 s later; the first ping 0.6 s after that.
await new Promise((r) => setTimeout(r, 4200))
out.push(document.getElementById('stream') === second ? 'kept' : 'torn down')
out.push(await command(30, 10000))
await fetch('/__probe/result', { method: 'POST', body: out.join(', ') })
`
