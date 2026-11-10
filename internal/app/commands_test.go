package app_test

import (
	"net/http"
	"sync"
	"testing"
)

// TestCommandWaitsForTheStream: a command sent before the tab's stream is
// open meets 409 and stays pending; once the stream opens, the page sends
// it again (static/js/commands.js) and the frame shows it.
func TestCommandWaitsForTheStream(t *testing.T) {
	gate := make(chan struct{})
	var once sync.Once
	handlers := func(a http.Handler) map[string]http.HandlerFunc {
		return map[string]http.HandlerFunc{
			"POST /showcase": func(w http.ResponseWriter, r *http.Request) {
				select {
				case <-gate:
				case <-r.Context().Done():
					return
				}
				a.ServeHTTP(w, r)
			},
			"/__probe/open": func(w http.ResponseWriter, r *http.Request) {
				once.Do(func() { close(gate) })
				w.WriteHeader(http.StatusNoContent)
			},
		}
	}
	_, body := probeWith(t, "/showcase", waitsForStreamJS, handlers)
	if got, want := string(body), "409, 70%, 70, not pending"; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

const waitsForStreamJS = `
const until = async (f, ms) => {
	for (const end = Date.now() + ms; Date.now() < end; await new Promise((r) => setTimeout(r, 50))) {
		const v = f()
		if (v) return v
	}
	return ''
}
const thrust = () => [...document.querySelectorAll('.flight__state dd')].map((d) => d.textContent).find((t) => t.endsWith('%')) ?? ''
const slider = await until(() => document.querySelector('sb-slider[name="thrust"]'), 10000)
let conflicts = 0
document.addEventListener('datastar-fetch', (e) => e.detail.el === slider && e.detail.type === 'error' && e.detail.argsRaw.status === '409' && conflicts++)
// Until Datastar has wired the page, a commit goes nowhere: commit until one meets the 409.
for (let i = 0; i < 40 && !conflicts; i++) {
	slider.value = 70
	slider.shadowRoot.querySelector('input').dispatchEvent(new Event('change'))
	await until(() => conflicts, 250)
}
await fetch('/__probe/open', { method: 'POST' })
const shown = await until(() => thrust() === '70%' && thrust(), 10000)
const out = [conflicts ? '409' : 'no 409', shown || thrust() || 'nothing', slider.value, slider.matches(':state(pending)') ? 'pending' : 'not pending']
await fetch('/__probe/result', { method: 'POST', body: out.join(', ') })
`
