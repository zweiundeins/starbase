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
	check('open="false" is present: open, as in HTML', state(el), OPEN)
	await morph('<sb-dropdown id="' + el.id + '" open items=\'["Alpha"]\'></sb-dropdown>')
	check('a morph with open keeps it open', state(el), OPEN)
	await morph('<sb-dropdown id="' + el.id + '" items=\'["Alpha"]\'></sb-dropdown>')
	check('a morph that drops open closes it', state(el), SHUT)
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

// TestDropdownRichItems checks sb-dropdown's rich markup items in Chrome: a data-icon element and
// a lozenge are copied into the rows, the page colours each lozenge through ::part() and an inline
// custom property, the accessible names stay the labels, the radio check, sb-change, confirm and the
// type-ahead still work, a morph of the items updates the rows, and the trigger shows the chosen
// item and follows a server change. JSON items stay text. With STARBASE_DATASTAR_BUNDLE=<file>, on
// that Datastar build.
func TestDropdownRichItems(t *testing.T) {
	copyRows(t, probeCDP(t, "/", pdPrelude+morphJS+dropdownRichJS, map[string]http.HandlerFunc{"/__test/morph": morphHandler}))
}

const dropdownRichJS = `
const cdp = async (...cmds) => {
	const r = await fetch('/__probe/cdp', { method: 'POST', body: JSON.stringify(cmds) })
	if (!r.ok) throw new Error('cdp: ' + (await r.text()))
	return r.json()
}
const group = async (name, fn) => {
	try {
		await fn()
	} catch (e) {
		rows.push({ step: name, error: String(e?.stack || e) })
	}
}
const style = document.createElement('style')
style.textContent = '.rich-dd::part(lozenge) { background-color: var(--tone); color: rgb(1, 2, 3); } .rich-dd::part(lozenge-done) { color: rgb(0, 128, 0); }'
document.head.append(style)
const box = document.createElement('div')
document.body.prepend(box)
const changes = [], selects = []
document.addEventListener('sb-change', (e) => changes.push(e.detail))
document.addEventListener('sb-select', (e) => selects.push(e.detail.value))
await customElements.whenDefined('sb-dropdown')

const SVG = (d) => '<svg data-icon viewBox="0 0 16 16" width="16" height="16" fill="currentColor" aria-hidden="true"><path d="' + d + '"/></svg>'
const item = ([v, label, tone, d, part = '']) => '<button slot="item" value="' + v + '">' + SVG(d) + '<span part="lozenge' + part + '" style="--tone: ' + tone + '">' + label + '</span></button>'
const ITEMS = [
	['todo', 'To do', 'rgb(10, 20, 30)', 'M4 4h8v8H4z'],
	['doing', 'In progress', 'rgb(40, 50, 60)', 'M2 2h4v4H2z'],
	['done', 'Done', 'rgb(70, 80, 90)', 'M1 1h2v2H1z', ' lozenge-done'],
]
const status = (value, items = ITEMS) => '<sb-dropdown id="rd" class="rich-dd" type="radio" name="status" label="Status" confirm value="' + value + '">' + items.map(item).join('') + '</sb-dropdown>'
box.innerHTML = status('todo')
await settle()
const el = document.getElementById('rd')
const $ = (s, e = el) => e.shadowRoot.querySelector(s)
const row = (i, e = el) => $('.lvl0 [data-idx="' + i + '"]', e)
const look = (r) => {
	const loz = r.querySelector('.rich [part~="lozenge"]')
	return [r.querySelector('.art path')?.getAttribute('d') ?? '', loz?.textContent ?? '', loz ? getComputedStyle(loz).backgroundColor : '', loz ? getComputedStyle(loz).color : '']
}
const trig = (e = el) => {
	const t = $('.trigger', e)
	return { art: t.querySelector('.art path')?.getAttribute('d') ?? '', rich: t.querySelector('.rich')?.textContent ?? '', text: getComputedStyle(t.querySelector('.text')).display === 'none' ? '' : t.querySelector('.text').textContent, name: t.getAttribute('aria-label') }
}
// The menuitems and menuitemradios of the open menus, as [name, checked].
const ax = async () => {
	const [{ nodes }] = await cdp({ method: 'Accessibility.getFullAXTree', params: {} })
	return nodes.filter((n) => !n.ignored && /^menuitem/.test(n.role?.value ?? '')).map((n) => [n.name?.value ?? '', n.properties?.find((p) => p.name === 'checked')?.value?.value ?? null])
}

await group('rows', async () => {
	el.show()
	await settle()
	check('icon, lozenge and its colours per row', [0, 1, 2].map((i) => look(row(i))), [
		['M4 4h8v8H4z', 'To do', 'rgb(10, 20, 30)', 'rgb(1, 2, 3)'],
		['M2 2h4v4H2z', 'In progress', 'rgb(40, 50, 60)', 'rgb(1, 2, 3)'],
		['M1 1h2v2H1z', 'Done', 'rgb(70, 80, 90)', 'rgb(0, 128, 0)'],
	])
	const r = row(0)
	check('the icon has the icon size', ((w) => w > 12 && w < 24)(r.querySelector('.art svg').getBoundingClientRect().width), true)
	check('the label is read out, not shown; the copy is hidden from it', [getComputedStyle(r.querySelector('.label')).position, r.querySelector('.label').textContent, r.querySelector('.rich').getAttribute('aria-hidden'), r.querySelector('.art').getAttribute('aria-hidden')], ['absolute', 'To do', 'true', 'true'])
	check('radio rows, the server value checked', [0, 1, 2].map((i) => [row(i).getAttribute('role'), row(i).getAttribute('aria-checked')]), [['menuitemradio', 'true'], ['menuitemradio', 'false'], ['menuitemradio', 'false']])
	check('accessible names are the labels', await ax(), [['To do', 'true'], ['In progress', 'false'], ['Done', 'false']])
	const parsed = document.createElement('div')
	parsed.innerHTML = item(ITEMS[0])
	check('the page markup is untouched', el.querySelector('button').outerHTML, parsed.firstElementChild.outerHTML)
	el.hide()
	await settle()
})

await group('trigger', async () => {
	check('shows the chosen item, named by label and choice', trig(), { art: 'M4 4h8v8H4z', rich: 'To do', text: '', name: 'Status: To do' })
	check('the page colours its lozenge too', getComputedStyle($('.trigger [part~="lozenge"]')).backgroundColor, 'rgb(10, 20, 30)')
})

await group('a pick', async () => {
	el.show()
	await settle()
	row(1).click()
	await settle()
	check('sb-change, the check moves and waits for the server', [changes.at(-1), row(1).getAttribute('aria-checked'), el.matches(':state(pending)'), el.value], [{ name: 'status', value: 'doing' }, 'true', true, 'doing'])
	check('the trigger shows the pick', trig(), { art: 'M2 2h4v4H2z', rich: 'In progress', text: '', name: 'Status: In progress' })
	await morph(status('doing'))
	check('the server agrees: no longer pending', el.matches(':state(pending)'), false)
})

await group('keyboard', async () => {
	$('.trigger').focus()
	press($('.trigger'), 'ArrowDown')
	await settle()
	press(el.shadowRoot.activeElement, 'd')
	await settle()
	check('type-ahead finds the label', el.shadowRoot.activeElement === row(2), true)
	press(el.shadowRoot.activeElement, 'Enter')
	await settle()
	check('Enter picks it', [changes.at(-1)?.value, el.open], ['done', false])
	el.revert()
	await settle()
	check('revert() puts the server value back, in the trigger too', [el.value, trig().rich], ['doing', 'In progress'])
})

await group('a server change of the value', async () => {
	await morph(status('done'))
	check('the trigger follows it', trig(), { art: 'M1 1h2v2H1z', rich: 'Done', text: '', name: 'Status: Done' })
	check('in the colours of that item', getComputedStyle($('.trigger [part~="lozenge"]')).color, 'rgb(0, 128, 0)')
})

await group('a morph of the items', async () => {
	el.show()
	await settle()
	await morph(status('done', [ITEMS[0], ['review', 'In review', 'rgb(5, 5, 5)', 'M3 3h1v1H3z'], ['done', 'Shipped', 'rgb(9, 9, 9)', 'M6 6h2v2H6z', ' lozenge-done'], ITEMS[1]]))
	check('the rows follow, the menu stays open', [[0, 1, 2, 3].map((i) => look(row(i))), el.open], [[
		['M4 4h8v8H4z', 'To do', 'rgb(10, 20, 30)', 'rgb(1, 2, 3)'],
		['M3 3h1v1H3z', 'In review', 'rgb(5, 5, 5)', 'rgb(1, 2, 3)'],
		['M6 6h2v2H6z', 'Shipped', 'rgb(9, 9, 9)', 'rgb(0, 128, 0)'],
		['M2 2h4v4H2z', 'In progress', 'rgb(40, 50, 60)', 'rgb(1, 2, 3)'],
	], true])
	check('names and check follow', await ax(), [['To do', 'false'], ['In review', 'false'], ['Shipped', 'true'], ['In progress', 'false']])
	check('the trigger follows the new markup', trig(), { art: 'M6 6h2v2H6z', rich: 'Shipped', text: '', name: 'Status: Shipped' })
	el.querySelector(':scope > [value="done"] [part~="lozenge"]').style.setProperty('--tone', 'rgb(1, 1, 1)')
	await settle()
	check('an attribute inside an item reaches its copy', getComputedStyle($('.lvl0 [data-idx="2"] [part~="lozenge"]')).backgroundColor, 'rgb(1, 1, 1)')
	el.hide()
	await settle()
})

await group('icon and text, no lozenge', async () => {
	box.insertAdjacentHTML('beforeend', '<sb-dropdown id="pd" type="radio" name="priority" label="Priority" value="high"><button slot="item" value="high">' + SVG('M8 2h1v1H8z') + ' High</button><button slot="item" value="low">' + SVG('M8 12h1v1H8z') + ' Low</button></sb-dropdown>')
	await settle()
	const pd = document.getElementById('pd')
	pd.show()
	await settle()
	const r = row(0, pd)
	check('the label shows, beside the icon', [r.querySelector('.art path')?.getAttribute('d'), getComputedStyle(r.querySelector('.label')).position, r.querySelector('.label').textContent, getComputedStyle(r.querySelector('.rich')).display], ['M8 2h1v1H8z', 'static', 'High', 'none'])
	check('the trigger: icon and label, named by both', trig(pd), { art: 'M8 2h1v1H8z', rich: '', text: 'High', name: 'Priority: High' })
	pd.remove()
})

await group('an actions menu', async () => {
	box.insertAdjacentHTML('beforeend', '<sb-dropdown id="ad">' + item(['x', 'Ex', 'red', 'M1 1h1v1H1z']) + '</sb-dropdown>')
	await settle()
	const ad = document.getElementById('ad')
	ad.show()
	await settle()
	check('a rich menuitem', [row(0, ad).getAttribute('role'), row(0, ad).querySelector('.rich').textContent], ['menuitem', 'Ex'])
	check('the trigger keeps its label', trig(ad), { art: '', rich: '', text: 'Actions', name: 'Actions' })
	row(0, ad).click()
	await settle()
	check('sb-select', selects.at(-1), 'x')
	ad.remove()
})

await group('JSON items stay text', async () => {
	const js = document.createElement('sb-dropdown')
	js.setAttribute('items', JSON.stringify([{ value: 'a', label: '<b>A</b>', icon: '<img src=x>', artEl: {}, richEl: '<i>no</i>' }]))
	box.append(js)
	await settle()
	js.show()
	await settle()
	const r = row(0, js)
	check('label and icon as text, nothing copied', [r.querySelector('.label').textContent, r.querySelector('.icon').textContent, r.querySelectorAll('.art *, .rich *').length], ['<b>A</b>', '<img src=x>', 0])
	js.remove()
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

// TestDropdownGroups checks sb-dropdown's group headings in Chrome, as markup and as JSON: a heading
// is a label row nobody can focus, click or choose; the keys and the type-ahead skip it; the
// accessibility tree shows each group named by its heading with its rows inside; a radio menu keeps
// one value across groups; a submenu holds groups too; and a morph that changes the groups updates
// the rows. With STARBASE_DATASTAR_BUNDLE=<file>, on that Datastar build.
func TestDropdownGroups(t *testing.T) {
	copyRows(t, probeCDP(t, "/", pdPrelude+morphJS+dropdownGroupsJS, map[string]http.HandlerFunc{"/__test/morph": morphHandler}))
}

const dropdownGroupsJS = `
const cdp = async (...cmds) => {
	const r = await fetch('/__probe/cdp', { method: 'POST', body: JSON.stringify(cmds) })
	if (!r.ok) throw new Error('cdp: ' + (await r.text()))
	return r.json()
}
const group = async (name, fn) => {
	try {
		await fn()
	} catch (e) {
		rows.push({ step: name, error: String(e?.stack || e) })
	}
}
const box = document.createElement('div')
document.body.prepend(box)
const changes = [], selects = []
document.addEventListener('sb-change', (e) => changes.push(e.detail))
document.addEventListener('sb-select', (e) => selects.push(e.detail.value))
await customElements.whenDefined('sb-dropdown')

// The open menu named name as the accessibility tree has it: [role, name, checked] per row and
// [role, name, rows] per group; ignored and generic nodes are looked through.
const ax = async (name) => {
	const [{ nodes }] = await cdp({ method: 'Accessibility.getFullAXTree', params: {} })
	const by = new Map(nodes.map((n) => [n.nodeId, n]))
	const role = (n) => n.role?.value ?? ''
	const walk = (n) =>
		(n.childIds ?? []).flatMap((id) => {
			const c = by.get(id)
			if (!c) return []
			if (c.ignored || ['generic', 'none', 'presentation'].includes(role(c))) return walk(c)
			if (role(c) === 'group') return [[role(c), c.name?.value ?? '', walk(c)]]
			if (role(c).startsWith('menuitem')) return [[role(c), c.name?.value ?? '', c.properties?.find((p) => p.name === 'checked')?.value?.value ?? null]]
			if (role(c) === 'separator') return [['separator']]
			return []
		})
	const menu = nodes.find((n) => !n.ignored && role(n) === 'menu' && n.name?.value === name)
	return menu ? walk(menu) : null
}

const STATUS = [['group', 'Move to'], ['todo', 'To do'], ['doing', 'In progress'], ['review', 'In review'], ['group', 'Other'], ['blocked', 'Blocked'], ['wontdo', 'Won’t do'], ['-'], ['archive', 'Archive']]
const markup = (value, items = STATUS) =>
	'<sb-dropdown id="gd" type="radio" name="status" label="Status" value="' + value + '">' +
	items.map(([v, l]) => (v === '-' ? '<hr slot="item">' : v === 'group' ? '<div slot="item" data-group>' + l + '</div>' : '<button slot="item" value="' + v + '">' + l + '</button>')).join('') +
	'</sb-dropdown>'
box.innerHTML = '<style>#gd::part(group-label) { text-transform: uppercase }</style>' + markup('doing')
await settle()
const el = document.getElementById('gd')
const $ = (s, e = el) => e.shadowRoot.querySelector(s)
const $$ = (s, e = el) => [...e.shadowRoot.querySelectorAll(s)]
// The rows of a level as the user sees them: "# " and a heading's text, or an item's label.
const seen = (k = 0, e = el) => $$('.lvl' + k + ' .menu > a', e).filter((a) => a.checkVisibility() && a.getAttribute('role') !== 'separator').map((a) => (a.getAttribute('part') === 'group-label' ? '# ' + a.textContent.trim() : a.querySelector('.label').textContent))
const focused = (e = el) => e.shadowRoot.activeElement?.querySelector('.label')?.textContent ?? null
const key = async (k, e = el) => (press(e.shadowRoot.activeElement ?? $('.trigger', e), k), await settle())

await group('headings', async () => {
	el.show()
	await settle()
	check('rows in order, headings among them', seen(), ['# Move to', 'To do', 'In progress', 'In review', '# Other', 'Blocked', 'Won’t do', 'Archive'])
	const heads = $$('.lvl0 [part="group-label"]')
	check('a heading is a group, with no tabindex, row index or href', heads.map((h) => [h.getAttribute('role'), h.hasAttribute('tabindex'), h.hasAttribute('data-idx'), h.hasAttribute('href')]), [['group', false, false, false], ['group', false, false, false]])
	check('named by its own text, owning the rows up to the next heading or divider', heads.map((h) => [h.getAttribute('aria-label'), h.getAttribute('aria-owns').split(' ').map((id) => el.shadowRoot.getElementById(id).querySelector('.label').textContent)]), [['Move to', ['To do', 'In progress', 'In review']], ['Other', ['Blocked', 'Won’t do']]])
	const cs = getComputedStyle(heads[0]), item = getComputedStyle($('.lvl0 [data-idx="1"]'))
	check('muted small text', [cs.color !== item.color, parseFloat(cs.fontSize) < parseFloat(item.fontSize), cs.cursor], [true, true, 'default'])
	heads[0].focus()
	check('a heading takes no focus', el.shadowRoot.activeElement === heads[0], false)
	check('the accessibility tree: menu, groups named by their headings, rows inside', await ax('Status'), [
		['group', 'Move to', [['menuitemradio', 'To do', 'false'], ['menuitemradio', 'In progress', 'true'], ['menuitemradio', 'In review', 'false']]],
		['group', 'Other', [['menuitemradio', 'Blocked', 'false'], ['menuitemradio', 'Won’t do', 'false']]],
		['separator'],
		['menuitemradio', 'Archive', 'false'],
	])
	heads[0].click()
	await settle()
	check('a click on a heading chooses nothing and keeps the menu open', [changes.length, el.open, el.value], [0, true, 'doing'])
	el.hide()
	await settle()
})

await group('keys', async () => {
	$('.trigger').focus()
	await key('ArrowDown')
	check('ArrowDown opens at the first item, not the heading', focused(), 'To do')
	await key('ArrowUp')
	check('ArrowUp wraps past the heading to the last item', focused(), 'Archive')
	await key('Home')
	check('Home skips the heading', focused(), 'To do')
	await key('ArrowDown')
	await key('ArrowDown')
	await key('ArrowDown')
	check('ArrowDown steps over the next heading', focused(), 'Blocked')
	await key('ArrowUp')
	check('ArrowUp steps back over it', focused(), 'In review')
	await key('End')
	check('End', focused(), 'Archive')
	await key('o')
	check('the type-ahead never lands on a heading ("Other")', focused(), 'Archive')
	await settle(800)
	await key('m')
	check('nor on "Move to"', focused(), 'Archive')
	await settle(800)
	await key('b')
	check('it finds an item in another group', focused(), 'Blocked')
	const head = $('.lvl0 [part="group-label"]')
	head.dispatchEvent(new PointerEvent('pointermove', { bubbles: true, composed: true, pointerType: 'mouse' }))
	await settle()
	check('the pointer over a heading leaves the focus where it is', focused(), 'Blocked')
	await key('Enter')
	check('Enter picks it: one value across groups', [changes.at(-1), el.value, el.open], [{ name: 'status', value: 'blocked' }, 'blocked', false])
	el.show()
	await settle()
	check('one check, in the other group', $$('.lvl0 [role="menuitemradio"]').filter((r) => r.getAttribute('aria-checked') === 'true').map((r) => r.querySelector('.label').textContent), ['Blocked'])
	check('the trigger names the pick', $('.trigger').getAttribute('aria-label'), 'Status: Blocked')
	el.hide()
	await settle()
})

await group('a morph that changes the groups', async () => {
	el.show()
	await settle()
	await morph(markup('review', [['group', 'Next'], ['review', 'In review'], ['done', 'Done'], ['group', 'Back'], ['todo', 'To do'], ['-'], ['archive', 'Archive'], ['group', 'Danger zone'], ['delete', 'Delete']]))
	check('the rows follow, the menu stays open', [seen(), el.open], [['# Next', 'In review', 'Done', '# Back', 'To do', 'Archive', '# Danger zone', 'Delete'], true])
	check('and the tree, with the server value checked', await ax('Status'), [
		['group', 'Next', [['menuitemradio', 'In review', 'true'], ['menuitemradio', 'Done', 'false']]],
		['group', 'Back', [['menuitemradio', 'To do', 'false']]],
		['separator'],
		['menuitemradio', 'Archive', 'false'],
		['group', 'Danger zone', [['menuitemradio', 'Delete', 'false']]],
	])
	await morph(markup('review', [['review', 'In review'], ['done', 'Done']]))
	check('headings removed', [seen(), $$('.lvl0 [aria-owns]').length], [['In review', 'Done'], 0])
	el.hide()
	await settle()
})

await group('JSON, and groups in a submenu', async () => {
	const js = document.createElement('sb-dropdown')
	js.setAttribute('label', 'Issue')
	js.setAttribute('items', JSON.stringify([
		{ label: 'Move', children: [{ group: 'Now' }, 'Start', 'Review', { group: '<b>Later</b>' }, 'Park'] },
		{ group: 'Edit' },
		'Rename',
		'Copy',
	]))
	box.append(js)
	await settle()
	$('.trigger', js).focus()
	await key('ArrowDown', js)
	check('JSON: the root, a heading after the submenu parent', seen(0, js), ['Move', '# Edit', 'Rename', 'Copy'])
	check('JSON tree', await ax('Issue'), [['menuitem', 'Move', null], ['group', 'Edit', [['menuitem', 'Rename', null], ['menuitem', 'Copy', null]]]])
	await key('ArrowDown', js)
	check('ArrowDown from the parent skips the heading', focused(js), 'Rename')
	await key('Home', js)
	await key('ArrowRight', js)
	check('the submenu opens at its first item', [seen(1, js), focused(js)], [['# Now', 'Start', 'Review', '# <b>Later</b>', 'Park'], 'Start'])
	check('the submenu tree; a heading is text', await ax('Move'), [['group', 'Now', [['menuitem', 'Start', null], ['menuitem', 'Review', null]]], ['group', '<b>Later</b>', [['menuitem', 'Park', null]]]])
	await key('ArrowDown', js)
	await key('ArrowDown', js)
	check('ArrowDown across the submenu heading', focused(js), 'Park')
	await key('Enter', js)
	check('sb-select from a grouped item', [selects.at(-1), js.open], ['Park', false])
	js.remove()
})

check('no errors', errors, [])
await report()
`
