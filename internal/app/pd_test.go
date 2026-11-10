package app_test

// The tests of the components from PD rockets (pd_*_test.go) share this.

// pdPrelude: rows, check, until, errors, press(el, key, init) / release(key),
// and drag(from, to, steps): a mouse drag from the middle of an element to
// another element's middle or a point {x, y}.
const pdPrelude = `
const rows = []
const check = (step, got, want) => rows.push({ step, got: JSON.stringify(got), want: JSON.stringify(want) })
const settle = (ms = 80) => new Promise((r) => setTimeout(r, ms))
const until = async (fn, ms = 15000) => {
	for (const end = performance.now() + ms; performance.now() < end; await settle(50)) if (fn()) break
	return fn()
}
const errors = []
addEventListener('error', (e) => errors.push(String(e.message)))
const press = (el, key, init = {}) => el.dispatchEvent(new KeyboardEvent('keydown', { key, bubbles: true, composed: true, cancelable: true, ...init }))
const release = (key) => dispatchEvent(new KeyboardEvent('keyup', { key, bubbles: true }))
const center = (el) => {
	const r = el.getBoundingClientRect()
	return { x: r.left + r.width / 2, y: r.top + r.height / 2 }
}
const pointer = (el, type, { x, y }) =>
	el.dispatchEvent(new PointerEvent(type, { pointerId: 1, pointerType: 'mouse', isPrimary: true, button: 0, buttons: type === 'pointerup' ? 0 : 1, clientX: x, clientY: y, bubbles: true, composed: true, cancelable: true }))
const drag = async (from, to, steps = 8) => {
	const a = center(from), b = to instanceof Element ? center(to) : to
	pointer(from, 'pointerdown', a)
	for (let k = 1; k <= steps; k++) {
		const p = { x: a.x + ((b.x - a.x) * k) / steps, y: a.y + ((b.y - a.y) * k) / steps }
		pointer(document.elementFromPoint(p.x, p.y) ?? document.body, 'pointermove', p)
		await new Promise(requestAnimationFrame)
	}
	pointer(document.elementFromPoint(b.x, b.y) ?? document.body, 'pointerup', b)
}
const report = () => fetch('/__probe/result', { method: 'POST', body: JSON.stringify(rows) })
`
