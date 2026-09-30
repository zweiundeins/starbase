package app_test

import (
	"encoding/json"
	"testing"
)

// TestCopyButtonFallback checks sb-copy-button's copies without the Clipboard API.
// javaScriptCanAccessClipboard lets execCommand('copy') work without a user activation.
func TestCopyButtonFallback(t *testing.T) {
	t.Run("plain http", func(t *testing.T) {
		_, body := probeAt(t, "sb-plain.test", "/", copyPrelude+copyPlainJS, "--blink-settings=javaScriptCanAccessClipboard=true")
		copyRows(t, body)
	})
	t.Run("secure origin", func(t *testing.T) {
		_, body := probe(t, "/", copyPrelude+copySecureJS)
		copyRows(t, body)
	})
}

func copyRows(t *testing.T, body []byte) {
	t.Helper()
	var rows []struct {
		Step, Got, Want, Error string
	}
	if err := json.Unmarshal(body, &rows); err != nil {
		t.Fatalf("%v: %s", err, body)
	}
	if len(rows) == 0 {
		t.Fatal("no rows")
	}
	for _, r := range rows {
		switch {
		case r.Error != "":
			t.Errorf("%s: %s", r.Step, r.Error)
		case r.Got != r.Want:
			t.Errorf("%s: got %s, want %s", r.Step, r.Got, r.Want)
		}
	}
}

// copyPrelude adds a button and the helpers both runs use; each run ends by
// posting rows.
const copyPrelude = `
const rows = []
const check = (step, got, want) => rows.push({ step, got: JSON.stringify(got), want: JSON.stringify(want) })
const settle = (ms = 60) => new Promise((r) => setTimeout(r, ms))
const made = async (value) => {
	await customElements.whenDefined('sb-copy-button')
	const el = document.createElement('sb-copy-button')
	el.setAttribute('value', value)
	el.setAttribute('reset-ms', '300')
	document.body.append(el)
	await settle()
	const root = el.shadowRoot
	const part = (p) => root.querySelector('[part~="' + p + '"]')
	const events = []
	for (const name of ['sb-copy', 'sb-copy-error']) el.addEventListener(name, (e) => events.push({ name, detail: e.detail, cancelable: e.cancelable }))
	const states = () => ['copied', 'failed', 'manual'].filter((s) => el.matches(':state(' + s + ')'))
	return { el, root, button: part('button'), tip: part('tip'), panel: part('manual'), text: part('manual-text'), events, states }
}
const exec = document.execCommand
const key = /mac|iphone|ipad|ipod/i.test(navigator.userAgentData?.platform ?? navigator.platform) ? '⌘C' : 'Ctrl+C'
`

const copyPlainJS = `
try {
	check('location.protocol', location.protocol, 'http:')
	check('isSecureContext', isSecureContext, false)
	check('navigator.clipboard is undefined', navigator.clipboard === undefined, true)
	if (location.protocol !== 'http:' || isSecureContext || navigator.clipboard !== undefined) throw new Error('the page is not a plain-HTTP page without the Clipboard API')

	const value = 'first line\n\tindented, café ✓\nlast'
	const { el, root, button, tip, panel, text, events, states } = await made(value)

	// Without the API: the textarea, in the click's own task.
	let source = null
	const onCopy = (e) => {
		const t = e.composedPath()[0]
		source = { t, tag: t.localName, inRoot: t.getRootNode() === root, selected: t.value?.slice(t.selectionStart, t.selectionEnd) }
	}
	document.addEventListener('copy', onCopy, true)
	button.click()
	document.removeEventListener('copy', onCopy, true)
	check('without the API, sb-copy has fired when click() returns', events, [{ name: 'sb-copy', detail: { value, method: 'execCommand' }, cancelable: false }])
	check('the copy came from a textarea in the shadow root', [source?.tag, source?.inRoot], ['textarea', true])
	check('the textarea had the value selected', source?.selected, value)
	check('the textarea is gone', source?.t.isConnected, false)
	check('the focus is back on the button', [document.activeElement === el, root.activeElement === button], [true, true])
	check('the button had no focus ring, and has none now', button.matches(':focus-visible'), false)
	check('copied', states(), ['copied'])
	check('the tip says copied-label', tip.textContent, 'Copied!')
	await settle(450)
	check('copied clears after reset-ms', states(), [])
	button.blur()
	button.focus({ focusVisible: true })
	const before = button.matches(':focus-visible')
	button.click()
	check('a button that had the ring when clicked keeps it', [before, root.activeElement === button, button.matches(':focus-visible')], [true, true, true])

	// The textarea fails: sb-copy-error, then the panel.
	button.blur()
	events.length = 0
	document.execCommand = () => false
	button.click()
	check('execCommand false: one cancelable sb-copy-error', events, [{ name: 'sb-copy-error', detail: { value, error: 'TypeError' }, cancelable: true }])
	check('the panel is open', panel.matches(':popover-open'), true)
	check('manual', states(), ['manual'])
	check('the panel\'s textarea has the focus', root.activeElement === text, true)
	check('with the whole value selected', [text.value, text.selectionStart, text.selectionEnd], [value, 0, value.length])
	const hint = root.getElementById(text.getAttribute('aria-labelledby'))?.textContent
	check('the message names the textarea and says the key', hint, 'Press ' + key + ' to copy')

	// Copies from the panel, through the browser's own copy command (the real one).
	let copies = 0
	text.addEventListener('copy', () => copies++)
	events.length = 0
	text.setSelectionRange(3, 3)
	exec.call(document, 'copy')
	await settle()
	check('a copy with nothing selected: no event, the panel stays', [copies, events, panel.matches(':popover-open'), states()], [1, [], true, ['manual']])
	text.setSelectionRange(0, 5)
	exec.call(document, 'copy')
	await settle()
	check('a copy of a part: no event, the panel stays', [copies, events, panel.matches(':popover-open'), states()], [2, [], true, ['manual']])
	text.select()
	text.dispatchEvent(new KeyboardEvent('keydown', { key: 'c', ctrlKey: true, bubbles: true, composed: true }))
	exec.call(document, 'copy')
	check('a copy of the whole text: sb-copy, method manual', [copies, events], [3, [{ name: 'sb-copy', detail: { value, method: 'manual' }, cancelable: false }]])
	check('copied after the panel copy', states(), ['copied'])
	await settle()
	check('the panel closed', panel.matches(':popover-open'), false)
	check('the focus is back on the button after a panel copy made with the keys, with a ring', [root.activeElement === button, button.matches(':focus-visible')], [true, true])
	check('the tip says copied-label after the panel copy', tip.textContent, 'Copied!')
	await settle(450)

	// A throwing execCommand behaves like a false one.
	events.length = 0
	document.execCommand = () => {
		throw new Error('refused')
	}
	button.blur(), button.focus({ focusVisible: false }) // as a mouse click leaves it
	button.click()
	check('execCommand throwing: one cancelable sb-copy-error', events, [{ name: 'sb-copy-error', detail: { value, error: 'TypeError' }, cancelable: true }])
	check('execCommand throwing: the panel is open', panel.matches(':popover-open'), true)
	check('execCommand throwing: manual', states(), ['manual'])
	check('execCommand throwing: the panel\'s textarea has the focus', root.activeElement === text, true)
	check('execCommand throwing: with the whole value selected', [text.value, text.selectionStart, text.selectionEnd], [value, 0, value.length])

	// hidePopover() (like Escape or light dismiss) closes it quietly.
	events.length = 0
	panel.hidePopover()
	await settle()
	check('hidePopover(): idle, no event, the focus on the button', [states(), events.length, root.activeElement === button, button.matches(':focus-visible')], [[], 0, true, false])

	// The page cancels sb-copy-error: v0.5.0's failed state.
	events.length = 0
	const cancel = (e) => e.preventDefault()
	el.addEventListener('sb-copy-error', cancel)
	button.click()
	el.removeEventListener('sb-copy-error', cancel)
	check('cancelled: failed', states(), ['failed'])
	check('cancelled: the tip says failed-label', tip.textContent, 'Copy failed')
	check('cancelled: no panel', panel.matches(':popover-open'), false)
	await settle(450)
	check('failed clears after reset-ms', states(), [])

	// A new label re-renders the button and closes an open panel.
	button.click()
	el.setAttribute('label', 'Copy the lines')
	await settle()
	check('a new label closes the panel', [root.querySelector('[part~="manual"]').matches(':popover-open'), states()], [false, []])
	check('the new label', root.querySelector('[part~="button"]').getAttribute('aria-label'), 'Copy the lines')
	el.remove()
} catch (e) {
	rows.push({ step: 'script', error: String(e?.stack ?? e) })
}
document.execCommand = exec
await fetch('/__probe/result', { method: 'POST', body: JSON.stringify(rows) })
`

const copySecureJS = `
try {
	check('isSecureContext', isSecureContext, true)
	const value = 'go run .'
	const { el, root, button, tip, panel, text, events, states } = await made(value)
	const added = []
	new MutationObserver((ms) => ms.forEach((m) => m.addedNodes.forEach((n) => added.push(n.localName)))).observe(root, { childList: true, subtree: true })
	let execs = 0
	document.execCommand = () => (execs++, false)
	const clipboard = (writeText) => Object.defineProperty(navigator, 'clipboard', { value: { writeText }, configurable: true })

	// The API refuses and the textarea fails: sb-copy-error with the API's error, and the panel.
	clipboard(() => Promise.reject(new DOMException('refused', 'NotAllowedError')))
	button.click()
	await settle()
	check('refused: sb-copy-error with the API\'s error', events, [{ name: 'sb-copy-error', detail: { value, error: 'NotAllowedError' }, cancelable: true }])
	check('refused: the textarea was tried', execs, 1)
	check('refused: the panel, focused', [panel.matches(':popover-open'), states(), root.activeElement === text], [true, ['manual'], true])
	check('refused: the message says the key', root.getElementById(text.getAttribute('aria-labelledby'))?.textContent, 'Press ' + key + ' to copy')
	panel.hidePopover()

	// A working API: v0.5.0 plus the method, and no textarea.
	events.length = 0
	execs = 0
	added.length = 0
	let wrote = null
	clipboard((v) => ((wrote = v), Promise.resolve()))
	button.click()
	await settle()
	check('a working API: sb-copy, method clipboard', events, [{ name: 'sb-copy', detail: { value, method: 'clipboard' }, cancelable: false }])
	check('a working API: it wrote the value', wrote, value)
	check('a working API: no textarea', [added.filter((n) => n === 'textarea'), execs], [[], 0])
	check('a working API: copied, the tip says copied-label', [states(), tip.textContent], [['copied'], 'Copied!'])
	await settle(450)
	check('a working API: copied clears after reset-ms', states(), [])

	// Without field-sizing the panel's field gets the rows its wrapped text fills: as tall as with it.
	clipboard(() => Promise.reject(new DOMException('refused', 'NotAllowedError')))
	const style = document.createElement('style')
	style.textContent = '.fixed::part(manual-text) { field-sizing: fixed }'
	document.head.append(style)
	const line = 'nfdump -M /var/nfsen/profiles-data/live/upstream1 -T -r 2026/09/30/nfcapd.202609301200 -n 10 -s srcip/bytes -o extended '
	for (const [name, v, want] of [['one line', value, 1], ['a wrapped line', line, 'from 2 to 5'], ['a long wrapped line', line.repeat(5), 6]]) {
		const seen = []
		for (const fixed of [false, true]) {
			el.classList.toggle('fixed', fixed)
			el.setAttribute('value', v)
			button.click()
			await settle()
			seen.push([getComputedStyle(text).fieldSizing, text.offsetHeight])
			if (fixed) check(name + ' without field-sizing: rows', text.rows > 1 && text.rows < 6 ? 'from 2 to 5' : text.rows, want)
			panel.hidePopover()
			await settle()
		}
		check(name + ': field-sizing, then none', seen.map(([f]) => f), ['content', 'fixed'])
		check(name + ': as tall without field-sizing', seen[1][1], seen[0][1])
	}
	el.remove()
} catch (e) {
	rows.push({ step: 'script', error: String(e?.stack ?? e) })
}
document.execCommand = exec
delete navigator.clipboard
await fetch('/__probe/result', { method: 'POST', body: JSON.stringify(rows) })
`
