package app_test

// The tests of the components from PD rockets (pd_*_test.go) share this.

// pdPrelude: rows, check, until, errors, and press(el, key, init) / release(key).
const pdPrelude = `
const rows = []
const check = (step, got, want) => rows.push({ step, got: JSON.stringify(got), want: JSON.stringify(want) })
const settle = (ms = 80) => new Promise((r) => setTimeout(r, ms))
const until = async (fn, ms = 5000) => {
	for (const end = performance.now() + ms; performance.now() < end; await settle(50)) if (fn()) break
	return fn()
}
const errors = []
addEventListener('error', (e) => errors.push(String(e.message)))
const press = (el, key, init = {}) => el.dispatchEvent(new KeyboardEvent('keydown', { key, bubbles: true, composed: true, cancelable: true, ...init }))
const release = (key) => dispatchEvent(new KeyboardEvent('keyup', { key, bubbles: true }))
const report = () => fetch('/__probe/result', { method: 'POST', body: JSON.stringify(rows) })
`
