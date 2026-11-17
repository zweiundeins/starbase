package app_test

import "testing"

// TestToastActions checks sb-toast's link and action in Chrome: a toast with href or action and a
// label renders a real link or button, in the tab order after its message and before its close
// button, named by the label and described by the message in the accessibility tree, and announced
// with it; without a label, or with a javascript: URL, there is none. The action emits
// sb-toast-action and closes the toast on Enter, and the keyboard focus moves on to the next
// toast; a plain click on the link closes it, a Ctrl click keeps it. The countdown stops while the
// page is hidden and resumes once it is visible again. With STARBASE_DATASTAR_BUNDLE=<file>, on
// that Datastar build.
func TestToastActions(t *testing.T) {
	copyRows(t, probeCDP(t, "/", pdPrelude+toastActionsJS))
}

const toastActionsJS = `
const group = async (name, fn) => {
	try {
		await fn()
	} catch (e) {
		rows.push({ step: name, error: String(e?.stack || e) })
	}
}
const cdp = (...cmds) => fetch('/__probe/cdp', { method: 'POST', body: JSON.stringify(cmds) }).then((r) => r.json())
const KEYS = { Enter: ['Enter', 13, '\r'], Tab: ['Tab', 9, ''] }
const key = async (k) => {
	const [code, vk, text] = KEYS[k]
	await cdp(
		{ method: 'Input.dispatchKeyEvent', params: { type: 'keyDown', key: k, code, windowsVirtualKeyCode: vk, text } },
		{ method: 'Input.dispatchKeyEvent', params: { type: 'keyUp', key: k, code, windowsVirtualKeyCode: vk } },
	)
	await settle()
}
const mouse = async (el, modifiers = 0) => {
	const r = el.getBoundingClientRect()
	const x = r.left + r.width / 2, y = r.top + r.height / 2
	await cdp(
		{ method: 'Input.dispatchMouseEvent', params: { type: 'mouseMoved', x, y } },
		{ method: 'Input.dispatchMouseEvent', params: { type: 'mousePressed', x, y, button: 'left', clickCount: 1, modifiers } },
		{ method: 'Input.dispatchMouseEvent', params: { type: 'mouseReleased', x, y, button: 'left', clickCount: 1, modifiers } },
		{ method: 'Input.dispatchMouseEvent', params: { type: 'mouseMoved', x: 1400, y: 980 } },
	)
	await settle()
}
const events = []
for (const type of ['sb-toast-action', 'sb-dismiss']) document.addEventListener(type, (e) => events.push([type, e.detail]))
// Link clicks as the page sees them; the test keeps them from navigating.
const clicks = []
addEventListener('click', (e) => {
	const a = e.composedPath().find((n) => n.localName === 'a')
	if (!a || !e.composedPath().some((n) => n.localName === 'sb-toast')) return
	clicks.push([e.ctrlKey, a.getAttribute('href')])
	e.preventDefault()
})

const box = document.createElement('div')
box.lang = 'de'
box.style.cssText = 'position: fixed; inset-block-start: 0; inset-inline-start: 0; z-index: 99999; inline-size: 24rem; background: #000'
document.body.append(box)
await customElements.whenDefined('sb-toast')
const TOASTS = [
	{ id: 'news', title: 'Das Treffen steht', text: 'Samstag, 14 Uhr.', href: '/match/7', label: 'Match öffnen' },
	{ id: 'msg', title: 'Neue Nachricht', text: 'Anna: Hallo', action: 'open-chat', label: 'Antworten' },
	{ id: 'bare', text: 'Ohne Label', href: '/x', action: 'y' },
	{ id: 'js', text: 'Skript', href: 'javascript:alert(1)', label: 'Nein' },
]
const make = async (toasts) => {
	box.innerHTML = '<button id="before">before</button><sb-toast placement="inline" max="4" duration="0" label="Mitteilungen" dismiss-label="Schliessen: {message}"></sb-toast>'
	const el = box.querySelector('sb-toast')
	el.setAttribute('toasts', JSON.stringify(toasts))
	await settle(250)
	return el
}
const rowsOf = (el) => [...el.shadowRoot.querySelectorAll('.toast')]
const look = (el) => rowsOf(el).map((r) => {
	const a = r.querySelector('a.act'), b = r.querySelector('button.act')
	return [a.checkVisibility() ? [a.getAttribute('href'), a.textContent] : null, b.checkVisibility() ? b.textContent : null]
})
// Where the keyboard focus is: the toast's message and the control's kind.
const focused = (el) => {
	const f = el.shadowRoot.activeElement
	if (!f) return null
	return [f.closest('.toast').querySelector('.text').textContent, f.matches('a') ? 'link' : f.matches('.act') ? 'action' : 'close', f.matches(':focus-visible')]
}
const ax = async () => {
	const [{ nodes }] = await cdp({ method: 'Accessibility.getFullAXTree', params: {} })
	return nodes.filter((n) => !n.ignored && ['link', 'button'].includes(n.role?.value) && /Match|Antworten|Nein/.test(n.name?.value ?? '')).map((n) => [n.role.value, n.name.value, n.description?.value ?? ''])
}

await group('rendered', async () => {
	const el = await make(TOASTS)
	check('a link, a button, and neither without a label or with a javascript: URL', look(el), [[['/match/7', 'Match öffnen'], null], [null, 'Antworten'], [null, null], [null, null]])
	check('the controls are parts', [...el.shadowRoot.querySelectorAll('[part=action]')].length, 8)
	check('named by the label and described by the message', await ax(), [['link', 'Match öffnen', 'Das Treffen steht Samstag, 14 Uhr.'], ['button', 'Antworten', 'Neue Nachricht Anna: Hallo']])
	const polite = el.shadowRoot.querySelector('[role=status]').textContent
	check('the live region announces the control with its toast', polite.trim(), 'Das Treffen steht. Samstag, 14 Uhr. Match öffnen. Neue Nachricht. Anna: Hallo. Antworten. Ohne Label. Skript')
	box.querySelector('#before').focus()
	const order = []
	for (let k = 0; k < 4; k++) {
		await key('Tab')
		order.push(focused(el))
	}
	check('in the tab order: each control after its message, before its close button', order, [
		['Samstag, 14 Uhr.', 'link', true],
		['Samstag, 14 Uhr.', 'close', true],
		['Anna: Hallo', 'action', true],
		['Anna: Hallo', 'close', true],
	])
})

await group('the action, with the keyboard', async () => {
	const el = await make(TOASTS)
	box.querySelector('#before').focus()
	await key('Tab')
	await key('Tab')
	await key('Tab')
	check('on the action', focused(el), ['Anna: Hallo', 'action', true])
	events.length = 0
	await key('Enter')
	await settle(300)
	check('sb-toast-action, then the toast closes', events, [['sb-toast-action', { id: 'msg', action: 'open-chat' }], ['sb-dismiss', { id: 'msg', reason: 'user' }]])
	check('gone', rowsOf(el).map((r) => r.querySelector('.text').textContent), ['Samstag, 14 Uhr.', 'Ohne Label', 'Skript'])
	check('the focus moves on to the next toast', focused(el), ['Ohne Label', 'close', true])
})

await group('the link', async () => {
	const el = await make(TOASTS)
	const link = () => el.shadowRoot.querySelector('a.act')
	events.length = 0
	clicks.length = 0
	await mouse(link(), 2)
	await settle(300)
	check('a Ctrl click opens it elsewhere and keeps the toast', [clicks, events, rowsOf(el).length], [[[true, '/match/7']], [], 4])
	await mouse(link())
	await settle(300)
	check('a plain click follows it and closes the toast', [clicks.at(-1), events], [[false, '/match/7'], [['sb-dismiss', { id: 'news', reason: 'user' }]]])
	check('gone', rowsOf(el).length, 3)
	const again = await make(TOASTS)
	box.querySelector('#before').focus()
	await key('Tab')
	events.length = 0
	await key('Enter')
	await settle(300)
	check('Enter follows it and closes the toast', [clicks.at(-1), events], [[false, '/match/7'], [['sb-dismiss', { id: 'news', reason: 'user' }]]])
	check('the focus moves on to the next toast, to its control', focused(again), ['Anna: Hallo', 'action', true])
})

await group('hidden page', async () => {
	let vis = 'visible'
	Object.defineProperty(document, 'visibilityState', { configurable: true, get: () => vis })
	Object.defineProperty(document, 'hidden', { configurable: true, get: () => vis === 'hidden' })
	const show = (v) => ((vis = v), document.dispatchEvent(new Event('visibilitychange')))
	const el = await make([{ id: 'v', text: 'Bald weg', duration: 800 }])
	const region = el.shadowRoot.querySelector('.region')
	events.length = 0
	show('hidden')
	await settle(1300)
	check('while hidden: no countdown, the bar stands', [events, rowsOf(el).length, region.classList.contains('paused')], [[], 1, true])
	show('visible')
	await settle(250)
	check('visible again: it resumes where it stopped', [events, rowsOf(el).length, region.classList.contains('paused')], [[], 1, false])
	await settle(900)
	check('and then expires', events, [['sb-dismiss', { id: 'v', reason: 'timeout' }]])
	el.remove()
	show('hidden')
	show('visible')
	delete document.visibilityState
	delete document.hidden
})

check('no errors', errors, [])
await report()
`
