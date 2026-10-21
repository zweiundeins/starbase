package app_test

import (
	"net/http"
	"testing"
)

// TestDropdownFollowsNotch checks that sb-dropdown's pixel details follow --sb-notch: at 1 the
// stepped caret, turning in two steps, and the stepped submenu arrow; at 0 triangles and a smooth turn.
func TestDropdownFollowsNotch(t *testing.T) {
	_, body := probe(t, "/components/dropdown", pdPrelude+dropdownNotchJS)
	copyRows(t, body)
}

const dropdownNotchJS = `
try {
	await customElements.whenDefined('sb-dropdown')
	const el = document.createElement('sb-dropdown')
	el.setAttribute('items', '[{"label":"Sort","children":[{"value":"a","label":"A"}]}]')
	document.body.prepend(el)
	await settle()
	const root = el.shadowRoot
	root.querySelector('.trigger').click()
	await until(() => root.querySelector('.more'))
	const look = () => {
		const cs = (q) => getComputedStyle(root.querySelector(q))
		return [cs('[part=caret]').clipPath.includes('6.667px'), cs('[part=caret]').transitionTimingFunction, cs('.more').clipPath.includes('4.5px')]
	}
	document.documentElement.style.setProperty('--sb-notch', '1')
	check('notch 1: the stepped caret, turning in two steps, and the stepped arrow', look(), [false, 'steps(2)', false])
	document.documentElement.style.setProperty('--sb-notch', '0')
	check('notch 0: triangles, and a smooth turn', look(), [true, 'steps(1000)', true])
	check('no errors', errors, [])
} catch (e) {
	rows.push({ step: 'script', error: String(e?.stack || e) })
}
await report()
`

// TestDropdownOpenState checks sb-dropdown's :state(open) in Chrome: it follows the menu whichever
// way it opens (a click, the keys, show()) or closes (Escape, a pick, an outside click, Tab,
// disabled, a server morph, removal), survives a morph that keeps it open, and starts closed after
// a re-attach. And the caret part: the default turn, a page's ::part(caret) rules over it, and
// aria-hidden. With STARBASE_DATASTAR_BUNDLE=<file>, on that Datastar build.
func TestDropdownOpenState(t *testing.T) {
	copyRows(t, probeBundle(t, "/", pdPrelude+morphJS+dropdownOpenJS, map[string]http.HandlerFunc{"/__test/morph": morphHandler}))
}

const dropdownOpenJS = `
const box = document.createElement('div')
document.body.prepend(box)
const outside = document.createElement('button')
outside.textContent = 'outside'
document.body.prepend(outside)
const style = document.createElement('style')
style.textContent = '.chev::part(caret) { rotate: 45deg; clip-path: none; background: none; border: solid currentColor; border-width: 0 2px 2px 0; } .chev:state(open)::part(caret) { rotate: 225deg; } .tint::part(caret) { color: rgb(255, 0, 0); }'
document.head.append(style)
await customElements.whenDefined('sb-dropdown')
const ITEMS = '["Alpha","Beta","Gamma"]'
let n = 0
const make = async (attrs = {}) => {
	const el = document.createElement('sb-dropdown')
	el.id = 'dd' + ++n
	for (const [k, v] of Object.entries({ items: ITEMS, ...attrs })) el.setAttribute(k, v)
	box.append(el)
	await settle()
	return el
}
const $ = (el, s) => el.shadowRoot.querySelector(s)
const trigger = (el) => $(el, '[part=trigger]')
// Keys go where the focus is: the focused row while the menu has it, else the trigger.
const key = async (el, k) => (press(el.shadowRoot.activeElement ?? trigger(el), k), await settle())
const state = (el) => [el.matches(':state(open)'), $(el, '.lvl0').matches(':popover-open')]
const group = async (name, fn) => {
	try {
		await fn()
	} catch (e) {
		rows.push({ step: name, error: String(e?.stack || e) })
	}
}
const OPEN = [true, true], SHUT = [false, false]

await group('open and close', async () => {
	const el = await make()
	check('closed at first', state(el), SHUT)
	trigger(el).click()
	await settle()
	check('a click opens', state(el), OPEN)
	trigger(el).click()
	await settle()
	check('a second click closes', state(el), SHUT)
	trigger(el).focus()
	await key(el, 'ArrowDown')
	check('ArrowDown opens', state(el), OPEN)
	await key(el, 'Escape')
	check('Escape closes', state(el), SHUT)
	await key(el, 'ArrowUp')
	check('ArrowUp opens', state(el), OPEN)
	await key(el, 'Home')
	await key(el, 'Enter')
	check('Enter picks and closes', state(el), SHUT)
	trigger(el).click()
	await settle()
	$(el, '[data-idx="1"]').click()
	await settle()
	check('a click on an item picks and closes', state(el), SHUT)
	trigger(el).click()
	await settle()
	outside.dispatchEvent(new PointerEvent('pointerdown', { bubbles: true, composed: true, pointerType: 'mouse' }))
	outside.focus()
	await settle()
	check('a click outside closes', state(el), SHUT)
	trigger(el).focus()
	await key(el, 'ArrowDown')
	await key(el, 'Tab')
	check('Tab closes', state(el), SHUT)
	el.show()
	await settle()
	check('show() opens', state(el), OPEN)
	el.hide()
	await settle()
	check('hide() closes', state(el), SHUT)
	el.show()
	await settle()
	el.setAttribute('disabled', '')
	await settle()
	check('disabled closes', state(el), SHUT)
	el.remove()
})

await group('a submenu', async () => {
	const el = await make({ items: '[{"label":"Sort","children":["A","B"]},"Rename"]' })
	trigger(el).focus()
	await key(el, 'ArrowDown')
	await key(el, 'ArrowRight')
	check('a submenu leaves it open', [state(el), $(el, '.lvl1').matches(':popover-open')], [OPEN, true])
	await key(el, 'Escape')
	check('Escape in the submenu keeps the root open', state(el), OPEN)
	await key(el, 'Escape')
	check('Escape at the root closes', state(el), SHUT)
	el.remove()
})

await group('morphs', async () => {
	const el = await make()
	el.show()
	await settle()
	await morph('<sb-dropdown id="' + el.id + '" items=\'["Alpha","Beta","Gamma","Delta"]\'></sb-dropdown>')
	check('a morph with new items leaves it open', [state(el), el.shadowRoot.querySelectorAll('.lvl0 [data-idx]').length], [OPEN, 4])
	await morph('<sb-dropdown id="' + el.id + '" open="false" items=\'["Alpha"]\'></sb-dropdown>')
	check('a morph with open="false" closes it', state(el), SHUT)
	await morph('<sb-dropdown id="' + el.id + '" open items=\'["Alpha"]\'></sb-dropdown>')
	check('a morph with open opens it', state(el), OPEN)
	await morph('<sb-dropdown id="' + el.id + '" open disabled items=\'["Alpha"]\'></sb-dropdown>')
	check('a morph that disables it closes it', state(el), SHUT)
	el.remove()
})

await group('removal and re-attach', async () => {
	const el = await make()
	el.show()
	await settle()
	el.remove()
	await settle()
	check('removed while open: no longer open', el.matches(':state(open)'), false)
	box.append(el)
	await settle()
	check('re-attached: closed', state(el), SHUT)
	trigger(el).click()
	await settle()
	check('re-attached: a click opens', state(el), OPEN)
	await key(el, 'Escape')
	check('re-attached: Escape closes', state(el), SHUT)
	el.remove()
})

await group('the caret part', async () => {
	const plain = await make()
	const chev = await make({ class: 'chev' })
	const tint = await make({ class: 'tint' })
	const caret = (el) => $(el, '[part=caret]')
	const look = (el) => ((cs) => ({ rotate: cs.rotate, clip: cs.clipPath === 'none' ? 'none' : 'polygon', border: cs.borderRightWidth, bg: cs.backgroundColor }))(getComputedStyle(caret(el)))
	check('aria-hidden', [plain, chev].map((el) => caret(el).getAttribute('aria-hidden')), ['true', 'true'])
	check('default, closed: the stepped caret, no rotation, no opacity of its own', [look(plain).rotate, look(plain).clip, look(plain).border, getComputedStyle(caret(plain)).opacity], ['none', 'polygon', '0px', '1'])
	check('default: filled with its color', look(plain).bg, getComputedStyle(caret(plain)).color)
	check('default: its color is the trigger text at 70%', getComputedStyle(caret(plain)).color.endsWith(' / 0.7)'), true)
	for (const el of [plain, chev, tint]) el.show()
	await settle(400)
	check('default, open: turned over', look(plain).rotate, '180deg')
	check('a page rule replaces the look and the rotation, open', look(chev), { rotate: '225deg', clip: 'none', border: '2px', bg: 'rgba(0, 0, 0, 0)' })
	check('a colour-only page rule keeps the turn', [look(tint).rotate, look(tint).bg], ['180deg', 'rgb(255, 0, 0)'])
	for (const el of [plain, chev, tint]) el.hide()
	await settle(400)
	check('closed again', [look(plain).rotate, look(chev).rotate, look(tint).rotate], ['none', '45deg', 'none'])
	for (const el of [plain, chev, tint]) el.remove()
})

check('no errors', errors, [])
await report()
`

// TestDropdownLinks checks sb-dropdown's link items in Chrome with real input: an item with an
// href renders as a link the browser follows on a click, Enter or Space (a Ctrl+click is left to
// the browser too), with no sb-select, and the menu closes; keyboard moves, type-ahead and submenus
// reach links like any item; a disabled link has no href, a radio group's items never get one, and
// <a slot="item" href> works like the JSON.
func TestDropdownLinks(t *testing.T) {
	copyRows(t, probeCDP(t, "/", dropdownLinksJS))
}

const dropdownLinksJS = `
const rows = []
const check = (step, got, want) => rows.push({ step, got: JSON.stringify(got), want: JSON.stringify(want) })
const settle = (ms = 80) => new Promise((r) => setTimeout(r, ms))
const errors = []
addEventListener('error', (e) => errors.push(String(e.message)))
const cdp = async (...cmds) => {
	const r = await fetch('/__probe/cdp', { method: 'POST', body: JSON.stringify(cmds) })
	if (!r.ok) throw new Error('cdp: ' + (await r.text()))
	return r.json()
}
const keyCodes = { ArrowDown: 40, ArrowRight: 39, Enter: 13, Escape: 27, ' ': 32, m: 77 }
const press = async (key) => {
	const p = { key, code: key === ' ' ? 'Space' : key, windowsVirtualKeyCode: keyCodes[key] }
	await cdp({ method: 'Input.dispatchKeyEvent', params: { type: 'rawKeyDown', ...p } }, { method: 'Input.dispatchKeyEvent', params: { type: 'keyUp', ...p } })
	await settle()
}
const click = async (el, modifiers = 0) => {
	const r = el.getBoundingClientRect()
	const p = { x: r.left + r.width / 2, y: r.top + r.height / 2, button: 'left', clickCount: 1, modifiers }
	await cdp({ method: 'Input.dispatchMouseEvent', params: { type: 'mousePressed', ...p } }, { method: 'Input.dispatchMouseEvent', params: { type: 'mouseReleased', ...p } })
	await settle()
}
const group = async (name, fn) => {
	try {
		await fn()
	} catch (e) {
		rows.push({ step: name, error: String(e?.stack || e) })
	}
}
const selects = []
document.addEventListener('sb-select', (e) => selects.push(e.detail.value))
const pages = async () => (await cdp({ method: 'Target.getTargets', params: {} }))[0].targetInfos.filter((t) => t.type === 'page').length
await customElements.whenDefined('sb-dropdown')
const ITEMS = JSON.stringify([
	{ value: 'alpha', label: 'Alpha', href: '#alpha' },
	{ value: 'act', label: 'Act' },
	{ value: 'off', label: 'Off', href: '#off', disabled: true },
	{ value: 'mars', label: 'Mars', href: '#mars' },
	{ label: 'More', href: '#parent', children: [{ value: 'deep', label: 'Deep', href: '#deep' }] },
	{ label: 'Sort', type: 'radio', children: [{ value: 'name', label: 'Name', href: '#name' }] },
])
const el = document.createElement('sb-dropdown')
el.setAttribute('items', ITEMS)
document.body.prepend(el)
await settle()
const $ = (s) => el.shadowRoot.querySelector(s)
const row = (i, k = 0) => $('.lvl' + k + ' [data-idx="' + i + '"]')
const open = () => el.matches(':state(open)')
const reset = () => {
	history.replaceState(null, '', location.pathname)
	selects.length = 0
}

await group('markup', async () => {
	el.show()
	await settle()
	check('a link is an a with role menuitem and its href', [row(0).localName, row(0).getAttribute('role'), row(0).getAttribute('href')], ['a', 'menuitem', '#alpha'])
	check('an action has no href', row(1).hasAttribute('href'), false)
	check('a disabled link has no href', [row(2).hasAttribute('href'), row(2).getAttribute('aria-disabled')], [false, 'true'])
	check('a submenu parent drops its href', [row(4).hasAttribute('href'), row(4).getAttribute('aria-haspopup')], [false, 'menu'])
	el.hide()
	await settle()
})

await group('a click follows the link', async () => {
	reset()
	el.show()
	await settle()
	await click(row(0))
	check('followed, menu closed, no sb-select', [location.hash, open(), selects], ['#alpha', false, []])
})

await group('Ctrl+click is left to the browser', async () => {
	reset()
	el.show()
	await settle()
	const before = await pages()
	await click(row(3), 2)
	await settle(300)
	check('a new tab, no navigation here, closed, no sb-select', [(await pages()) - before, location.hash, open(), selects], [1, '', false, []])
})

await group('Enter follows the link', async () => {
	reset()
	$('.trigger').focus()
	await press('ArrowDown')
	check('the first row has the focus', el.shadowRoot.activeElement === row(0), true)
	await press('Enter')
	check('followed, closed, focus back on the trigger, no sb-select', [location.hash, open(), el.shadowRoot.activeElement === $('.trigger'), selects], ['#alpha', false, true, []])
})

await group('Space follows the link', async () => {
	reset()
	$('.trigger').focus()
	await press('ArrowDown')
	await press('ArrowDown')
	await press('ArrowDown')
	check('ArrowDown skips the disabled link', el.shadowRoot.activeElement === row(3), true)
	await press(' ')
	check('followed and closed', [location.hash, open(), selects], ['#mars', false, []])
})

await group('type-ahead and an action', async () => {
	reset()
	$('.trigger').focus()
	await press('ArrowDown')
	await press('m')
	check('type-ahead reaches a link', el.shadowRoot.activeElement === row(3), true)
	await press('ArrowDown')
	await press('ArrowDown')
	await press('ArrowDown')
	await press('ArrowDown')
	check('back round to the action', el.shadowRoot.activeElement === row(1), true)
	await press('Enter')
	check('an action still reports sb-select and navigates nowhere', [selects, location.hash], [['act'], ''])
})

await group('a link in a submenu', async () => {
	reset()
	el.show()
	await settle()
	row(4).focus()
	await press('ArrowRight')
	check('the submenu has the link focused', [el.shadowRoot.activeElement === row(0, 1), row(0, 1).getAttribute('href')], [true, '#deep'])
	await press('Enter')
	check('followed, every level closed', [location.hash, open(), $('.lvl1').matches(':popover-open')], ['#deep', false, false])
})

await group('a radio group has no links', async () => {
	reset()
	el.show()
	await settle()
	row(5).focus()
	await press('ArrowRight')
	check('menuitemradio without href', [row(0, 1).getAttribute('role'), row(0, 1).hasAttribute('href')], ['menuitemradio', false])
	await press('Escape')
	await press('Escape')
})

await group('slotted links', async () => {
	const dd = document.createElement('sb-dropdown')
	dd.innerHTML = '<a slot="item" href="#slot-a">Slot A</a><button slot="item" value="b">B</button><a slot="item" href="#slot-c" aria-disabled="true">C</a>'
	document.body.prepend(dd)
	await settle()
	dd.show()
	await settle()
	const r = (i) => dd.shadowRoot.querySelector('.lvl0 [data-idx="' + i + '"]')
	check('read as link, action and disabled link', [r(0).getAttribute('href'), r(1).hasAttribute('href'), r(2).hasAttribute('href')], ['#slot-a', false, false])
	dd.querySelector('a').setAttribute('href', '#slot-z')
	await settle()
	check('a new href in the page reaches the menu', r(0).getAttribute('href'), '#slot-z')
	reset()
	await click(r(0))
	check('followed and closed', [location.hash, dd.matches(':state(open)'), selects], ['#slot-z', false, []])
	dd.remove()
})

check('no errors', errors, [])
await fetch('/__probe/result', { method: 'POST', body: JSON.stringify(rows) })
`
