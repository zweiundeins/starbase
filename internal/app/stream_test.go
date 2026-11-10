package app_test

import "testing"

// TestRenderStreamInChrome: a command's outcome reaches the page through the
// tab's render stream, compressed as Chrome asks for it (zstd), and morphs in.
func TestRenderStreamInChrome(t *testing.T) {
	_, body := probe(t, "/showcase", renderStreamJS)
	if got := string(body); got != "70% 1 watching" {
		t.Fatalf("after the command the page shows %q, want the thrust and the viewer count", got)
	}
}

const renderStreamJS = `
const until = async (f, ms) => {
	for (const end = Date.now() + ms; Date.now() < end; await new Promise((r) => setTimeout(r, 50))) {
		const v = f()
		if (v) return v
	}
	return ''
}
const thrust = () => [...document.querySelectorAll('.flight__state dd')].map((d) => d.textContent).find((t) => t.endsWith('%'))
const slider = await until(() => { const s = document.querySelector('sb-slider[name="thrust"]'); return s?.shadowRoot?.querySelector('input') && s }, 30000) // upgraded, with its shadow root
// Until Datastar has wired the page, the event goes nowhere: send it again.
let shown = ''
for (let i = 0; i < 10 && !shown; i++) {
	slider.dispatchEvent(new CustomEvent('sb-change', { detail: { name: 'thrust', value: 70 }, bubbles: true, composed: true }))
	shown = await until(() => thrust() === '70%' && thrust(), 1000)
}
const viewers = document.querySelector('.board-stats')?.textContent.trim().split(' · ')[0] ?? ''
await fetch('/__probe/result', { method: 'POST', body: (shown || thrust() || 'nothing') + ' ' + viewers })
`

// TestBoardSurvivesPageFrames: a paint reaches the board through the board's
// own view, and a frame of the tab's view (the whole #app, after a command
// of the tab) carries the board as it is, so the morph keeps the element
// and its cells.
func TestBoardSurvivesPageFrames(t *testing.T) {
	_, body := probe(t, "/showcase", boardFramesJS)
	if got := string(body); got != "same board 7 70% 1 watching" {
		t.Fatalf("after a paint and a command the page shows %q, want the same board with the paint", got)
	}
}

const boardFramesJS = `
const until = async (f, ms) => {
	for (const end = Date.now() + ms; Date.now() < end; await new Promise((r) => setTimeout(r, 50))) {
		const v = f()
		if (v) return v
	}
	return ''
}
const cell = () => document.getElementById('board')?.getAttribute('cells')?.[0] ?? ''
const board = await until(() => document.getElementById('board'), 10000)
// Until Datastar has wired the page, the event goes nowhere: send it again.
for (let i = 0; i < 10 && cell() !== '7'; i++) {
	board.dispatchEvent(new CustomEvent('sb-paint', { detail: { color: 7, cells: [0] }, bubbles: true, composed: true }))
	await until(() => cell() === '7', 1000)
}
const thrust = () => [...document.querySelectorAll('.flight__state dd')].map((d) => d.textContent).find((t) => t.endsWith('%'))
const slider = document.querySelector('sb-slider[name="thrust"]')
slider.dispatchEvent(new CustomEvent('sb-change', { detail: { name: 'thrust', value: 70 }, bubbles: true, composed: true }))
await until(() => thrust() === '70%', 5000)
const same = document.getElementById('board') === board ? 'same board' : 'another board'
const viewers = document.querySelector('.board-stats')?.textContent.trim().split(' · ')[0] ?? ''
await fetch('/__probe/result', { method: 'POST', body: [same, cell(), thrust(), viewers].join(' ') })
`

// TestFlightRefusalInChrome: a thrust the rules forbid comes back as a
// refusal in the tab's frame with the op's ack, on which the page puts the
// slider back; an accepted thrust ends pending with its echo; a call sign
// the server normalizes back to the stored one goes back on its ack too.
func TestFlightRefusalInChrome(t *testing.T) {
	_, body := probe(t, "/showcase", flightRefusalJS)
	if got, want := string(body), "refused 40 not pending 40%, accepted 70 not pending, normalized STARBASE-1 not pending"; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

const flightRefusalJS = `
const until = async (f, ms) => {
	for (const end = Date.now() + ms; Date.now() < end; await new Promise((r) => setTimeout(r, 50))) {
		const v = f()
		if (v) return v
	}
	return ''
}
const refusal = () => [...document.querySelectorAll('.flight__error')].some((p) => p.textContent.includes('flight director'))
const pending = (el) => (el.matches(':state(pending)') ? 'pending' : 'not pending')
const thrust = () => [...document.querySelectorAll('.flight__state dd')].map((d) => d.textContent).find((t) => t.endsWith('%'))
const slider = await until(() => { const s = document.querySelector('sb-slider[name="thrust"]'); return s?.shadowRoot?.querySelector('input') && s }, 30000) // upgraded, with its shadow root
const commit = (v) => ((slider.value = v), slider.shadowRoot.querySelector('input').dispatchEvent(new Event('change')))
// Until Datastar has wired the page, a commit goes nowhere: commit again.
for (let i = 0; i < 10 && !refusal(); i++) {
	commit(95)
	await until(refusal, 1500)
}
await until(() => slider.value === 40 && !slider.matches(':state(pending)'), 3000)
const out = [['refused', slider.value, pending(slider), thrust()].join(' ')]
commit(70)
await until(() => thrust() === '70%' && !slider.matches(':state(pending)'), 5000)
out.push(['accepted', slider.value, pending(slider)].join(' '))
const sign = document.querySelector('sb-input[name="callsign"]')
sign.value = 'starbase-1'
sign.dispatchEvent(new CustomEvent('sb-change', { detail: { name: 'callsign', value: 'starbase-1' }, bubbles: true, composed: true }))
await until(() => sign.value === 'STARBASE-1' && !sign.matches(':state(pending)'), 5000)
out.push(['normalized', sign.value, pending(sign)].join(' '))
await fetch('/__probe/result', { method: 'POST', body: out.join(', ') })
`
