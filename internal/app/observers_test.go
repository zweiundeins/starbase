package app_test

import "testing"

// TestBoundPropsSettle inserts components after load with a prop bound to a
// signal: their observeProps callbacks run inside that binding's effect, and
// one that doesn't peek subscribes it to the component's own signals, which
// loops when it writes a new array (pixel-board's palette did).
func TestBoundPropsSettle(t *testing.T) {
	_, body := probe(t, "/", boundPropsJS)
	copyRows(t, body)
}

const boundPropsJS = `
const rows = []
const check = (step, got, want) => rows.push({ step, got: JSON.stringify(got), want: JSON.stringify(want) })
const settle = (ms) => new Promise((r) => setTimeout(r, ms))
const errors = []
addEventListener('error', (e) => errors.push(String(e.message)))
const log = console.error
console.error = (...a) => (errors.push(a.map(String).join(' ')), log(...a))
const palette = JSON.stringify(['#000000', '#1D2B53', '#7E2553', '#008751', '#AB5236', '#5F574F', '#C2C3C7', '#FFF1E8', '#FF004D', '#FFA300', '#FFEC27', '#00E436', '#29ADFF', '#83769C', '#FF77A8', '#FFCCAA'])
const cases = [
	['sb-pixel-board', 'palette', palette, 'JSON.stringify($_v)', ' size="8" local'],
	['sb-count-up', 'value', '4200', '$_v', ' duration="1500"'],
	['sb-gauge', 'value', '42', '$_v', ''],
	['sb-qr-code', 'value', JSON.stringify('https://example.com'), '$_v', ''],
	['sb-relative-time', 'datetime', JSON.stringify(new Date(Date.now() - 5000).toISOString()), '$_v', ' sync'],
	['sb-code-editor', 'language', JSON.stringify('css'), '$_v', ' value="a { color: red }"'],
	['sb-typewriter', 'phrases', JSON.stringify(['alpha bravo', 'charlie']), 'JSON.stringify($_v)', ' loop interval="5" delay="0" hold="50"'],
]
try {
	for (const [tag, attr, value, expr, extra] of cases) {
		await customElements.whenDefined(tag)
		const before = errors.length
		const box = document.createElement('div')
		box.setAttribute('data-signals:_v', value)
		box.innerHTML = '<' + tag + ' data-attr:' + attr + '="' + expr + '"' + extra + '></' + tag + '>'
		document.body.append(box)
		// count-up animates and relative-time ticks meanwhile.
		await settle(1600)
		check(tag + ': no errors', errors.slice(before), [])
		check(tag + ': bound', box.firstElementChild.hasAttribute(attr), true)
		box.remove()
	}
} catch (e) {
	rows.push({ step: 'script', error: String(e?.stack || e) })
}
await fetch('/__probe/result', { method: 'POST', body: JSON.stringify(rows) })
`
