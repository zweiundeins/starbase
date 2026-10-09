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
