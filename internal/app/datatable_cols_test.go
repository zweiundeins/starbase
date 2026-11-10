package app_test

import "testing"

// TestDataTableColumns checks sb-data-table's columns in Chrome, driven
// through the DevTools protocol where a page can't fake it (real pointer
// input, forced colours, the accessibility tree): the header lines up with
// the rows, clipped cells peek, and columns resize by pointer and keyboard.
// With STARBASE_DATASTAR_BUNDLE=<file>, on that Datastar build (see
// TestVirtualScrollFocus); STARBASE_SHOTS=<dir> saves screenshots there.
func TestDataTableColumns(t *testing.T) {
	copyRows(t, probeCDP(t, "/", dataTableColumnsJS))
}

const dataTableColumnsJS = `
await customElements.whenDefined('sb-data-table')
await Promise.all(['400 14px "Pixelify Sans"', '600 14px "Pixelify Sans"', '400 14px "JetBrains Mono"', '600 14px "JetBrains Mono"'].map((f) => document.fonts.load(f)))
const settle = (ms = 80) => new Promise((r) => setTimeout(r, ms))
const frame = () => new Promise((r) => requestAnimationFrame(() => requestAnimationFrame(r)))
const out = []
let group = ''
const check = (step, got, want) => out.push({ step: group + ': ' + step, got: JSON.stringify(got), want: JSON.stringify(want) })
addEventListener('error', (e) => out.push({ step: group, error: 'page: ' + (e.error?.stack || e.message) }))
addEventListener('unhandledrejection', (e) => out.push({ step: group, error: 'page: ' + (e.reason?.stack || e.reason) }))
const cdp = async (...cmds) => {
	const r = await fetch('/__probe/cdp', { method: 'POST', body: JSON.stringify(cmds) })
	if (!r.ok) throw new Error('cdp: ' + (await r.text()))
	return r.json()
}
const css = document.createElement('style')
document.head.append(css)
const make = async (attrs, style = '') => {
	const el = document.createElement('sb-data-table')
	for (const [k, v] of Object.entries(attrs)) el.setAttribute(k, typeof v === 'string' ? v : JSON.stringify(v))
	el.style.cssText = 'block-size: 20rem; inline-size: 900px; ' + style
	document.body.prepend(el)
	await settle()
	await frame()
	return el
}
const $ = (el, s) => el.shadowRoot.querySelector(s)
const $$ = (el, s) => [...el.shadowRoot.querySelectorAll(s)]
const rect = (e) => e.getBoundingClientRect()
const round = (x) => Math.round(x * 10) / 10
const groups = []
const test = (name, f) => groups.push([name, f])
const mouse = (type, x, y, extra = {}) => cdp({ method: 'Input.dispatchMouseEvent', params: { type, x, y, button: type === 'mouseMoved' ? 'none' : 'left', ...extra } })
const center = (e) => { const r = rect(e); return [r.left + r.width / 2, r.top + r.height / 2] }
const keyCodes = { ArrowLeft: 37, ArrowRight: 39, ArrowUp: 38, ArrowDown: 40, Home: 36, End: 35, Enter: 13, Escape: 27, Tab: 9 }
// A real key press; modifiers: 1 Alt, 8 Shift.
const press = async (key, modifiers = 0) => {
	const p = { key, code: key, windowsVirtualKeyCode: keyCodes[key], modifiers }
	await cdp({ method: 'Input.dispatchKeyEvent', params: { type: 'rawKeyDown', ...p } }, { method: 'Input.dispatchKeyEvent', params: { type: 'keyUp', ...p } })
	await frame()
}
const shot = (name, el, pad = 8) => { const r = rect(el); return cdp({ method: 'shot', params: { name, clip: { x: r.left + scrollX - pad, y: r.top + scrollY - pad, width: r.width + 2 * pad, height: r.height + 2 * pad } } }) }
const scrollerOf = (el) => $(el, '.grid').shadowRoot.querySelector('.scroller')
const long = 'a value far too long for its column to show'

test('alignment', async () => {
	css.textContent = '.dt-align { font-family: "Pixelify Sans"; } .dt-align::part(column) { font-family: "JetBrains Mono"; font-size: 1.15em; }'
	const widths = ['14ch', '9ch', '6em', '11ch', '8em', '12ch', '7ch']
	const columns = widths.map((width, i) => ({ key: 'c' + i, label: 'Column ' + i, width }))
	const el = await make({ class: 'dt-align', columns, rows: [Object.fromEntries([['id', 1], ...widths.map((_, i) => ['c' + i, 'value ' + i])])] })
	el.className = 'dt-align'
	await frame()
	const heads = $$(el, '.th').map((e) => rect(e).left)
	const cells = $$(el, '.row .cell').map((e) => rect(e).left)
	check('header cells start where the cells do (within 1 px)', heads.map((x, i) => Math.abs(x - cells[i]) <= 1), widths.map(() => true))
	check('the header font changed', getComputedStyle($(el, '.th')).fontFamily.includes('JetBrains'), true)
	el.remove()
	css.textContent = ''
})

test('peek', async () => {
	const rows = [...Array(12).keys()].map((i) => ({ id: i, a: i === 1 ? 'short' : 'A ' + long, b: i === 1 ? 'short' : 'B ' + long, c: i === 1 ? 'short' : 'C ' + long }))
	const el = await make({ 'column-picker': '', columns: [{ key: 'a', label: 'A', width: '8rem' }, { key: 'b', label: 'B', width: '8rem' }, { key: 'c', label: 'C', width: '8rem' }], rows }, 'inline-size: 420px')
	const scroller = scrollerOf(el)
	const before = [scroller.scrollWidth, scroller.clientHeight]
	const cellAt = (r, j) => $$(el, '.row')[r].querySelectorAll('.cell')[j]
	const tracks = [0, 1, 2].map((j) => rect(cellAt(0, j)))
	const height = rect($$(el, '.row')[2]).height
	const sizes = () => [scroller.scrollWidth, scroller.clientHeight]

	const mid = cellAt(2, 1)
	await mouse('mouseMoved', ...center(mid))
	await frame()
	check('a hovered clipped cell peeks', [mid.hasAttribute('data-peek'), getComputedStyle(mid).overflow, getComputedStyle(mid).zIndex], [true, 'visible', '1'])
	check('it grows from its start', [round(rect(mid).left) === round(tracks[1].left), rect(mid).width > tracks[1].width + 50], [true, true])
	check('the row keeps its height', rect($$(el, '.row')[2]).height, height)
	check('the scroller keeps its scroll width and client height', sizes(), before)
	await shot('data-table-peek-middle', $(el, '.grid'))

	const short = cellAt(1, 1)
	await mouse('mouseMoved', ...center(short))
	await frame()
	check('an unclipped cell does not peek', [short.hasAttribute('data-peek'), round(rect(short).width)], [false, round(tracks[1].width)])
	check('the cell left behind stops peeking', [mid.hasAttribute('data-peek'), round(rect(mid).width)], [false, round(tracks[1].width)])

	const last = cellAt(2, 2)
	await mouse('mouseMoved', ...center(last))
	await frame()
	check('the last column peeks toward the start', [last.hasAttribute('data-peek'), Math.abs(rect(last).right - tracks[2].right) <= 1, rect(last).left < tracks[2].left - 50], [true, true, true])
	check('and the scroller still keeps its sizes', sizes(), before)
	await shot('data-table-peek-last', $(el, '.grid'))

	await mouse('mouseMoved', 1400, 990)
	await frame()
	check('the pointer gone: no peek', $$(el, '.cell[data-peek]').length, 0)

	// The keyboard: the focus moves from the first cell of row 3 to the next.
	cellAt(3, 0).focus()
	await frame()
	check('a focused clipped cell peeks', cellAt(3, 0).hasAttribute('data-peek'), true)
	await press('ArrowRight')
	const next = cellAt(3, 1)
	check('the arrow key moves the peek with the focus', [el.shadowRoot.activeElement === next, next.hasAttribute('data-peek'), cellAt(3, 0).hasAttribute('data-peek')], [true, true, false])
	check('the focused peek grows', rect(next).width > tracks[1].width + 50, true)
	check('focus keeps the sizes too', sizes(), before)

	// The sticky header paints over a peeking cell scrolled under it.
	scroller.scrollTop = 20
	await frame()
	const under = cellAt(0, 1)
	under.focus({ preventScroll: true })
	await frame()
	const head = rect($(el, '.head'))
	const hit = el.shadowRoot.elementFromPoint(Math.min(rect(under).right, head.right) - 20, head.bottom - 4)
	check('the header paints over it', [under.hasAttribute('data-peek'), rect(under).top < head.bottom, !!hit?.closest('.head')], [true, true, true])
	scroller.scrollTop = 0
	await frame()

	// The column menu (top layer) paints over a peek.
	const peeking = cellAt(2, 1)
	el.shadowRoot.activeElement?.blur()
	await mouse('mouseMoved', rect(peeking).left + 10, rect(peeking).top + 10)
	await frame()
	$(el, '.columns').click()
	await settle()
	const menu = rect($(el, '.menu')), p = rect(peeking)
	const x = (Math.max(menu.left, p.left) + Math.min(menu.right, p.right)) / 2, y = (Math.max(menu.top, p.top) + Math.min(menu.bottom, p.bottom)) / 2
	const overlap = Math.min(menu.right, p.right) > Math.max(menu.left, p.left) && Math.min(menu.bottom, p.bottom) > Math.max(menu.top, p.top)
	check('the column menu paints over it', [peeking.hasAttribute('data-peek'), overlap, !!el.shadowRoot.elementFromPoint(x, y)?.closest('.menu')], [true, true, true])
	$(el, '.menu').hidePopover()
	await settle()
	el.remove()
})

// Resizing.
const gripOf = (el, j) => $$(el, '.th')[j].querySelector('.grip')
const widthAt = (el, j) => Math.round(rect($$(el, '.th')[j]).width)
const cellWidths = (el, j) => $$(el, '.row').map((r) => Math.round(rect(r.querySelectorAll('.cell')[j]).width))
const heard = (el) => {
	const got = []
	el.addEventListener('sb-column-widths', (e) => got.push(e.detail))
	return got
}
// A real drag of the grip by dx px, in steps.
const dragBy = async (g, dx, steps = 5, during) => {
	const [x, y] = center(g)
	await mouse('mousePressed', x, y, { clickCount: 1 })
	for (let i = 1; i <= steps; i++) await mouse('mouseMoved', x + (dx * i) / steps, y, { button: 'left', buttons: 1 })
	await frame()
	const mid = during?.()
	await mouse('mouseReleased', x + dx, y, { clickCount: 1 })
	await frame()
	return mid
}
const dblclick = async (g) => {
	g.scrollIntoView({ block: 'nearest', inline: 'nearest' })
	await frame()
	const [x, y] = center(g)
	for (const clickCount of [1, 2]) await mouse('mousePressed', x, y, { clickCount }), await mouse('mouseReleased', x, y, { clickCount })
	await frame()
}
const cols3 = [{ key: 'a', label: 'Alpha', width: '10rem', sortable: true }, { key: 'b', label: 'Beta', width: '10rem', min: 60 }, { key: 'c', label: 'Gamma' }]
const rows3 = [...Array(6).keys()].map((i) => ({ id: i, a: 'alpha ' + i, b: 'beta ' + i, c: 'gamma ' + i }))

test('resize by pointer', async () => {
	const el = await make({ resizable: '', name: 'stars', columns: cols3, rows: rows3 })
	const got = heard(el), sorts = []
	el.addEventListener('sb-sort', (e) => sorts.push(e.detail.key))
	const g = gripOf(el, 0)
	check('a grip in every resizable header', $$(el, '.grip').map((x) => getComputedStyle(x).display !== 'none'), [true, true, true])
	check('the grip', ['part', 'role', 'aria-orientation', 'tabindex', 'title', 'aria-label'].map((a) => g.getAttribute(a)), ['column-grip', 'separator', 'vertical', '-1', 'Drag to resize, double-click to fit', 'Alpha'])
	check('it sits at the header\'s end', Math.abs(rect(g).right - rect($$(el, '.th')[0]).right) <= 1, true)
	check('headers are named by their title', $$(el, '.th').map((t) => t.getAttribute('aria-label')), ['Alpha', 'Beta', 'Gamma'])
	const w0 = widthAt(el, 0)
	const mid = await dragBy(g, 60, 6, () => [widthAt(el, 0), got.length, g.hasAttribute('data-drag')])
	check('the column follows the pointer, and nothing is emitted while it moves', mid, [w0 + 60, 0, true])
	check('one sb-column-widths on release, with the whole map', got, [{ name: 'stars', widths: { a: w0 + 60 } }])
	check('the header and the cells take the width', [widthAt(el, 0), [...new Set(cellWidths(el, 0))]], [w0 + 60, [w0 + 60]])
	check('the grip is not a sort', [sorts, g.hasAttribute('data-drag')], [[], false])
	check('the property is the local map; the attribute stays', [el.widths, el.getAttribute('widths')], [{ a: w0 + 60 }, null])

	await dragBy(gripOf(el, 1), -500)
	check('no narrower than min', [widthAt(el, 1), got.at(-1)], [60, { name: 'stars', widths: { a: w0 + 60, b: 60 } }])
	await dragBy(gripOf(el, 2), -1000)
	check('the default min is 40 px', widthAt(el, 2), 40)

	const n = got.length
	const [x, y] = center(g)
	await mouse('mousePressed', x, y, { clickCount: 1 })
	await mouse('mouseReleased', x, y, { clickCount: 1 })
	await frame()
	check('a click without a move commits nothing', got.length, n)

	// pointercancel: the width goes back, nothing is emitted. A synthetic PointerEvent (pointerId 1, the mouse).
	await mouse('mousePressed', x, y, { clickCount: 1 })
	await mouse('mouseMoved', x + 40, y, { button: 'left', buttons: 1 })
	await frame()
	g.dispatchEvent(new PointerEvent('pointercancel', { pointerId: 1, bubbles: true, composed: true }))
	await frame()
	await mouse('mouseReleased', x + 40, y, { clickCount: 1 })
	await frame()
	check('a cancelled drag goes back', [widthAt(el, 0), got.length], [w0 + 60, n])

	// A synthetic drag (pointerId 1) works too.
	const p = (type, dx) => g.dispatchEvent(new PointerEvent(type, { pointerId: 1, button: 0, buttons: 1, clientX: x + dx, clientY: y, bubbles: true, composed: true, cancelable: true, isPrimary: true }))
	p('pointerdown', 0), p('pointermove', -20)
	await frame()
	p('pointerup', -20)
	await frame()
	check('a synthetic drag', [widthAt(el, 0), got.at(-1).widths.a], [w0 + 40, w0 + 40])

	el.setAttribute('resizable', 'false')
	await frame()
	check('resizable="false": no grips', $$(el, '.grip').map((x) => getComputedStyle(x).display), ['none', 'none', 'none'])
	el.remove()

	const fixed = await make({ resizable: '', columns: [{ key: 'a', resizable: false }, { key: 'b' }], rows: rows3 })
	check('resizable: false on a column: no grip there', $$(fixed, '.grip').map((x) => getComputedStyle(x).display !== 'none'), [false, true])
	fixed.remove()
})

test('fit', async () => {
	const rows = [{ id: 1, a: 'x', b: 'a fairly long value in this column', c: { value: 1234567.5, suffix: 'light years' }, d: { value: 'Danger zone', tone: 'danger' } }, { id: 2, a: 'y', b: 'sh', c: 3, d: 'z' }]
	const columns = [{ key: 'a', label: 'A header label longer than its values', width: '4rem' }, { key: 'b', label: 'B', width: '4rem' }, { key: 'c', label: 'C', width: '4rem', align: 'end' }, { key: 'd', label: 'D', width: '4rem' }, { key: 'e', label: 'E', width: '4rem', fit: 250 }]
	const el = await make({ resizable: '', columns, rows })
	const got = heard(el)
	const clipped = (j) => $$(el, '.row').map((r) => r.querySelectorAll('.cell')[j]).filter((c) => c.scrollWidth > c.clientWidth).length
	check('clipped before', [1, 2, 3].map(clipped), [1, 1, 1])
	for (const j of [1, 2, 3]) await dblclick(gripOf(el, j))
	check('a double click fits every value (the table has every row)', [1, 2, 3].map(clipped), [0, 0, 0])
	check('snug: within a few px of the widest value', [1, 2, 3].map((j) => Math.max(...$$(el, '.row').map((r) => r.querySelectorAll('.cell')[j].scrollWidth)) >= widthAt(el, j) - 4), [true, true, true])
	check('one event per fit', got.length, 3)
	await dblclick(gripOf(el, 0))
	const label = $$(el, '.th')[0].querySelector('span[data-text]') ?? $$(el, '.th')[0].lastElementChild
	check('never narrower than the header', $$(el, '.th')[0].scrollWidth <= $$(el, '.th')[0].clientWidth, true)
	await dblclick(gripOf(el, 4))
	check('the server\'s fit', widthAt(el, 4), 250)
	el.remove()

	// A window of a larger result: the rows at hand.
	const win = await make({ resizable: '', columns: [{ key: 'a', width: '4rem' }], rows: [...Array(20).keys()].map((i) => ({ id: i, a: 'v'.repeat(i + 1) })), total: 1000 })
	await dblclick(gripOf(win, 0))
	const cells = $$(win, '.row .cell')
	check('a window: fit to the rows at hand', [cells.filter((c) => c.scrollWidth > c.clientWidth).length, widthAt(win, 0) - cells.at(-1).scrollWidth <= 4], [0, true])
	win.remove()
})

test('resize by keyboard', async () => {
	const el = await make({ resizable: '', name: 'k', columns: [...cols3.slice(0, 2), { ...cols3[2], fit: 300 }], rows: rows3 })
	const got = heard(el)
	const g = gripOf(el, 0), button = $$(el, '.th')[0].querySelector('button')
	check('one tab stop in the grid; the grips are not', [$$(el, '[tabindex="0"]').length, $$(el, '.grip').map((x) => x.tabIndex)], [1, [-1, -1, -1]])
	button.focus()
	await press('ArrowRight', 1)
	check('Alt+ArrowRight on a header: its separator', el.shadowRoot.activeElement === g, true)
	const w0 = widthAt(el, 0)
	check('its values', ['aria-valuenow', 'aria-valuemin', 'aria-valuemax', 'aria-valuetext'].map((a) => g.getAttribute(a)), [String(w0), '40', '2000', w0 + ' px'])
	check('still one tab stop', $$(el, '[tabindex="0"]').length, 1)
	await press('ArrowRight')
	check('ArrowRight: 8 px wider, emitted', [widthAt(el, 0), got], [w0 + 8, [{ name: 'k', widths: { a: w0 + 8 } }]])
	await press('ArrowRight', 8)
	check('Shift: 32 px', widthAt(el, 0), w0 + 40)
	await press('ArrowLeft')
	await frame()
	check('ArrowLeft: narrower, and aria-valuenow follows', [widthAt(el, 0), g.getAttribute('aria-valuenow'), g.getAttribute('aria-valuetext')], [w0 + 32, String(w0 + 32), w0 + 32 + ' px'])
	await press('Home')
	check('Home: the minimum', widthAt(el, 0), 40)
	await press('End')
	check('End: the fit', widthAt(el, 0) > 40, true)
	check('one event per key', got.length, 5)
	const seen = []
	const onKey = (e) => seen.push(e.key)
	window.addEventListener('keydown', onKey)
	await press('Escape')
	window.removeEventListener('keydown', onKey)
	check('Escape restores the width from before, and returns to the header', [widthAt(el, 0), got.at(-1).widths, el.shadowRoot.activeElement === button], [w0, {}, true])
	check('Escape goes no further', seen, [])

	$$(el, '.th')[2].focus()
	await press('ArrowLeft', 1)
	check('Alt+ArrowLeft too, on a header without a sort', el.shadowRoot.activeElement === gripOf(el, 2), true)
	await press('End')
	check('End: the server\'s fit', widthAt(el, 2), 300)
	await press('Enter')
	check('Enter returns to the header', el.shadowRoot.activeElement === $$(el, '.th')[2], true)
	await press('ArrowLeft', 1)
	await press('Tab')
	check('so does Tab', el.shadowRoot.activeElement === $$(el, '.th')[2], true)
	const before = got.length
	await press('ArrowRight', 1)
	await press('ArrowDown')
	check('ArrowDown on a separator does nothing', [el.shadowRoot.activeElement === gripOf(el, 2), got.length], [true, before])
	await press('Escape')

	// The accessibility tree: headers named by their titles, not "Alpha 50".
	const [tree] = await cdp({ method: 'Accessibility.getFullAXTree', params: {} })
	const names = (role) => tree.nodes.filter((x) => x.role?.value === role).map((x) => x.name?.value)
	check('column headers in the accessibility tree', names('columnheader').filter((x) => ['Alpha', 'Beta', 'Gamma'].includes(x)), ['Alpha', 'Beta', 'Gamma'])
	check('no header name with a number in it', names('columnheader').filter((x) => /\d/.test(x ?? '')), [])
	check('the separators are there', names('separator').filter((x) => ['Alpha', 'Beta', 'Gamma'].includes(x)), ['Alpha', 'Beta', 'Gamma'])
	el.remove()
})

test('widths: server wins', async () => {
	const el = await make({ resizable: '', confirm: '', selection: 'single', columns: cols3, rows: rows3, widths: { a: 150 } })
	const got = heard(el)
	check('widths sets the tracks', widthAt(el, 0), 150)
	gripOf(el, 0).focus()
	await press('ArrowRight')
	check('the user\'s', [widthAt(el, 0), el.widths], [158, { a: 158 }])
	el.setAttribute('widths', '{"a":150}')
	await frame()
	check('the same attribute again keeps the user\'s widths', widthAt(el, 0), 158)
	el.setAttribute('widths', '{"a":200,"b":90}')
	await frame()
	check('a new one wins', [widthAt(el, 0), widthAt(el, 1), el.widths], [200, 90, { a: 200, b: 90 }])
	el.removeAttribute('widths')
	await frame()
	check('a removed one is ignored', [widthAt(el, 0), el.widths], [200, { a: 200, b: 90 }])
	el.widths = { c: 120 }
	await frame()
	check('a property write: no event, no attribute', [widthAt(el, 2), widthAt(el, 0) === 200, got.length, el.hasAttribute('widths')], [120, false, 1, false])
	el.setAttribute('widths', '{"a":"170px","b":-3,"c":"x"}')
	await frame()
	check('only positive px', el.widths, { a: 170 })
	$$(el, '.row')[0].click()
	await frame()
	gripOf(el, 0).focus()
	await press('ArrowRight')
	check('widths are not pending', [el.matches(':state(pending)'), el.selected], [true, ['0']])
	el.revert()
	await frame()
	check('revert() leaves the widths', [el.matches(':state(pending)'), el.widths], [false, { a: 178 }])
	el.remove()
})

test('reset and hidden columns', async () => {
	const el = await make({ resizable: '', 'column-picker': '', columns: cols3, rows: rows3, widths: { b: 222 } })
	const got = heard(el)
	const reset = $(el, '.reset')
	check('the menu has a reset item', [reset.getAttribute('part'), reset.textContent, getComputedStyle(reset).display !== 'none'], ['reset-widths', 'Reset column widths', true])
	const boxes = () => $$(el, '.menu input')
	$(el, '.columns').click()
	await settle()
	boxes()[1].click()
	await settle()
	check('b hidden', $$(el, '.th').map((t) => t.getAttribute('aria-label')), ['Alpha', 'Gamma'])
	boxes()[1].click()
	await settle()
	check('shown again, it keeps its width', widthAt(el, 1), 222)
	boxes()[2].focus()
	el.shadowRoot.activeElement.dispatchEvent(new KeyboardEvent('keydown', { key: 'Tab', bubbles: true, composed: true, cancelable: true }))
	check('Tab reaches the reset item', el.shadowRoot.activeElement === reset, true)
	el.shadowRoot.activeElement.dispatchEvent(new KeyboardEvent('keydown', { key: 'Tab', bubbles: true, composed: true, cancelable: true }))
	check('and cycles on', el.shadowRoot.activeElement === boxes()[0], true)
	reset.click()
	await frame()
	check('reset: {} and the columns\' own widths', [got, el.widths, widthAt(el, 1)], [[{ name: '', widths: {} }], {}, 160])
	$(el, '.menu').hidePopover()
	el.removeAttribute('resizable')
	await frame()
	check('not resizable: no reset item', getComputedStyle(reset).display, 'none')
	el.remove()
})

test('forced colours', async () => {
	const el = await make({ resizable: '', columns: cols3, rows: rows3.map((r) => ({ ...r, a: 'A ' + long })) })
	const g = gripOf(el, 0)
	const after = (x) => getComputedStyle(x, '::after')
	check('the grip: a dashed line', [after(g).backgroundImage.startsWith('repeating-linear-gradient'), after(g).width], [true, '1px'])
	await shot('data-table-grip-default', $(el, '.head'), 4)
	$$(el, '.th')[0].querySelector('button').focus()
	await press('ArrowRight', 1)
	check('focused: wider, in the focus colour', [after(g).width, after(g).backgroundColor !== 'rgba(0, 0, 0, 0)'], ['3px', true])
	await shot('data-table-grip-focused', $(el, '.head'), 4)
	await press('Escape')

	await cdp({ method: 'Emulation.setEmulatedMedia', params: { features: [{ name: 'forced-colors', value: 'active' }] } })
	await frame()
	const sys = (c) => { const d = document.createElement('div'); d.style.color = c; document.body.append(d); const v = getComputedStyle(d).color; d.remove(); return v }
	check('forced colours are on', matchMedia('(forced-colors: active)').matches, true)
	check('the grip is CanvasText', after(g).backgroundColor, sys('CanvasText'))
	await mouse('mouseMoved', ...center(g))
	await frame()
	check('hovered: Highlight', after(g).backgroundColor, sys('Highlight'))
	await mouse('mouseMoved', 1400, 990)
	$$(el, '.th')[0].querySelector('button').focus()
	await press('ArrowRight', 1)
	check('focused: Highlight, with an outline the system colours', [after(g).backgroundColor, getComputedStyle(g).outlineStyle, getComputedStyle(g).outlineWidth], [sys('Highlight'), 'solid', '2px'])
	await shot('data-table-grip-forced', $(el, '.head'), 4)
	await press('Escape')
	const cell = $$(el, '.row')[1].querySelector('.cell')
	await mouse('mouseMoved', ...center(cell))
	await frame()
	check('a peek has an outline', [cell.hasAttribute('data-peek'), getComputedStyle(cell).outlineStyle], [true, 'solid'])
	await shot('data-table-peek-forced', $(el, '.grid'))
	await mouse('mouseMoved', 1400, 990)
	await cdp({ method: 'Emulation.setEmulatedMedia', params: { features: [] } })
	el.remove()
})

test('rtl', async () => {
	const el = await make({ dir: 'rtl', resizable: '', columns: cols3, rows: rows3.map((r) => ({ ...r, c: 'C ' + long })) }, 'inline-size: 700px')
	const got = heard(el)
	const g = gripOf(el, 0), th = $$(el, '.th')[0]
	check('the grip is at the header\'s end, its left', Math.abs(rect(g).left - rect(th).left) <= 1, true)
	const w0 = widthAt(el, 0)
	await dragBy(g, -50)
	check('dragging left widens it', [widthAt(el, 0), got.at(-1).widths.a], [w0 + 50, w0 + 50])
	th.querySelector('button').focus()
	await press('ArrowLeft', 1)
	check('Alt+ArrowLeft reaches the separator', el.shadowRoot.activeElement === g, true)
	await press('ArrowLeft')
	check('ArrowLeft widens it', widthAt(el, 0), w0 + 58)
	await press('ArrowRight', 8)
	check('Shift+ArrowRight narrows it by 32', widthAt(el, 0), w0 + 26)
	await press('Escape')
	const cell = $$(el, '.row')[1].querySelectorAll('.cell')[2], track = rect(cell)
	await mouse('mouseMoved', ...center(cell))
	await frame()
	check('the last column grows toward the start, to the right', [cell.hasAttribute('data-peek'), Math.abs(rect(cell).left - track.left) <= 1, rect(cell).right > track.right + 50], [true, true, true])
	await mouse('mouseMoved', 1400, 990)
	el.remove()
})

for (const [name, f] of groups) {
	group = name
	try {
		await f()
	} catch (e) {
		out.push({ step: name, error: String(e.stack || e) })
	}
}
await fetch('/__probe/result', { method: 'POST', body: JSON.stringify(out) })
`
