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
let out
try {
	// The first test of a run starts the app cold: give the page time.
	const slider = await until(() => document.querySelector('sb-slider[name="thrust"]'), 60000)
	if (!slider) throw new Error('no slider')
	let conflicts = 0
	document.addEventListener('datastar-fetch', (e) => e.detail.el === slider && e.detail.type === 'error' && e.detail.argsRaw.status === '409' && conflicts++)
	// Until Datastar has wired the page, a commit goes nowhere: commit until one meets the 409.
	for (const end = Date.now() + 60000; !conflicts && Date.now() < end; ) {
		slider.value = 70
		slider.shadowRoot.querySelector('input').dispatchEvent(new Event('change'))
		await until(() => conflicts, 250)
	}
	await fetch('/__probe/open', { method: 'POST' })
	const shown = await until(() => thrust() === '70%' && thrust(), 30000)
	out = [conflicts ? '409' : 'no 409', shown || thrust() || 'nothing', slider.value, slider.matches(':state(pending)') ? 'pending' : 'not pending'].join(', ')
} catch (e) {
	out = String(e?.stack || e)
}
await fetch('/__probe/result', { method: 'POST', body: out })
`

// TestCommandLostOnReconnect: a command answered 204 that never reached the
// tab's row (here the test swallows it) is decided by the stream: when it
// connects again without the op's ack, the page puts the control back and
// says so.
func TestCommandLostOnReconnect(t *testing.T) {
	handlers := func(http.Handler) map[string]http.HandlerFunc {
		return map[string]http.HandlerFunc{
			"/cmd/flight": func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNoContent) },
		}
	}
	_, body := probeWith(t, "/showcase", lostOnReconnectJS, handlers)
	if got, want := string(body), "pending, 40, not pending, said so"; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

const lostOnReconnectJS = `
const until = async (f, ms) => {
	for (const end = Date.now() + ms; Date.now() < end; await new Promise((r) => setTimeout(r, 50))) {
		const v = f()
		if (v) return v
	}
	return ''
}
let framed = false
document.addEventListener('datastar-fetch', (e) => e.detail.el?.id === 'stream' && e.detail.type === 'datastar-patch-elements' && (framed = true))
const slider = await until(() => document.querySelector('sb-slider[name="thrust"]'), 10000)
// A commit before the stream's first frame would end at that frame, without its ack.
await until(() => framed, 5000)
let answered = false
document.addEventListener('datastar-fetch', (e) => e.detail.el === slider && e.detail.type === 'finished' && (answered = true))
// Until Datastar has wired the page and the stream is open, a commit goes nowhere: commit again.
for (let i = 0; i < 20 && !answered; i++) {
	slider.value = 70
	slider.shadowRoot.querySelector('input').dispatchEvent(new Event('change'))
	await until(() => answered, 500)
}
await new Promise((r) => setTimeout(r, 500))
const out = [slider.matches(':state(pending)') ? 'pending' : 'not pending']
const s = document.getElementById('stream')
s.replaceWith(s.cloneNode(true))
await until(() => slider.value === 40 && !slider.matches(':state(pending)'), 10000)
out.push(slider.value, slider.matches(':state(pending)') ? 'pending' : 'not pending')
out.push([...document.querySelectorAll('.flight__error')].some((p) => p.textContent.includes('did not go through')) ? 'said so' : 'silent')
await fetch('/__probe/result', { method: 'POST', body: out.join(', ') })
`

// TestShareRefusalInChrome: code over 64 KB is refused in the tab's frame,
// which acks the save's op: the button stops waiting and the page says why.
func TestShareRefusalInChrome(t *testing.T) {
	_, body := probe(t, "/playground", shareRefusalJS)
	if got, want := string(body), "refused, not pending, no failure"; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

const shareRefusalJS = `
const until = async (f, ms) => {
	for (const end = Date.now() + ms; Date.now() < end; await new Promise((r) => setTimeout(r, 50))) {
		const v = f()
		if (v) return v
	}
	return ''
}
const pg = await until(() => document.querySelector('sb-code-playground'), 10000)
Object.defineProperty(pg, 'files', { get: () => ({ 'component.js': 'x'.repeat(65 << 10) }) })
const button = () => document.querySelector('.code-page__actions .btn--primary')
const refused = () => [...document.querySelectorAll('.code-page__refusal')].some((p) => p.textContent.includes('over 64 KB'))
let failed = false
document.addEventListener('sb-op-failed', () => (failed = true), true)
// Until Datastar has wired the page, a click goes nowhere: click again.
for (let i = 0; i < 10 && !refused(); i++) {
	button().click()
	await until(refused, 2000)
}
await until(() => !button().classList.contains('is-pending'), 3000)
const out = [refused() ? 'refused' : 'not refused', button().classList.contains('is-pending') ? 'pending' : 'not pending', failed ? 'failed' : 'no failure']
await fetch('/__probe/result', { method: 'POST', body: out.join(', ') })
`
