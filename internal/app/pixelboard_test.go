package app_test

import "testing"

// TestPixelBoardPending: a painted pixel stays pending until cells answers
// it, with its colour or another painter's, or until the page calls
// revert(); never on a timer.
func TestPixelBoardPending(t *testing.T) {
	_, body := probe(t, "/", pixelBoardJS)
	if got, want := string(body), "pending,pending after 3.5 s,server's after revert(),echoed,another painter's"; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

const pixelBoardJS = `
const frame = () => new Promise((r) => requestAnimationFrame(() => requestAnimationFrame(r)))
const sleep = (ms) => new Promise((r) => setTimeout(r, ms))
await customElements.whenDefined('sb-pixel-board')
const el = document.createElement('sb-pixel-board')
el.setAttribute('size', '8')
el.setAttribute('cells', '0'.repeat(64))
document.body.append(el)
await frame()
const canvas = el.shadowRoot.querySelector('canvas')
const rgb = (i) => [...canvas.getContext('2d').getImageData(i % 8, (i / 8) | 0, 1, 1).data.slice(0, 3)].join(' ')
const board = (c) => ({ 0: '8 13 29', 7: '176 154 255', 15: '229 72 77' })[c]
const paint = async () => {
	canvas.focus()
	canvas.dispatchEvent(new KeyboardEvent('keydown', { key: 'Enter', bubbles: true }))
	canvas.blur()
	await frame()
}
const cell = 36 // the cursor starts in the middle
const out = []
await paint()
out.push(rgb(cell) !== board(0) && rgb(cell) !== board(7) ? 'pending' : 'not pending: ' + rgb(cell))
await sleep(3500)
await frame()
out.push(rgb(cell) !== board(0) ? 'pending after 3.5 s' : 'expired')
el.revert()
await frame()
out.push(rgb(cell) === board(0) ? "server's after revert()" : 'still ' + rgb(cell))
await paint()
el.setAttribute('cells', '0'.repeat(cell) + '7' + '0'.repeat(63 - cell))
await frame()
out.push(rgb(cell) === board(7) ? 'echoed' : 'not echoed: ' + rgb(cell))
el.setAttribute('cells', '0'.repeat(64))
await frame()
await paint()
el.setAttribute('cells', '0'.repeat(cell) + 'f' + '0'.repeat(63 - cell))
await frame()
out.push(rgb(cell) === board(15) ? "another painter's" : 'still pending: ' + rgb(cell))
await fetch('/__probe/result', { method: 'POST', body: out.join(',') })
`
