package app_test

import (
	"encoding/json"
	"testing"
)

// TestDataTableCells checks sb-data-table in the browser, one row per check:
// cell objects (checks 1 to 8), hidden columns and their menu (9), sb-export (10).
func TestDataTableCells(t *testing.T) {
	_, body := probe(t, "/", dataTableJS)
	var rows []struct {
		Check                  int
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
			t.Errorf("check %d: %s", r.Check, r.Error)
		case r.Got != r.Want:
			t.Errorf("check %d, %s: got %s, want %s", r.Check, r.Step, r.Got, r.Want)
		}
	}
}

const dataTableJS = `
await customElements.whenDefined('sb-data-table')
const settle = (ms = 80) => new Promise((r) => setTimeout(r, ms))
const out = []
let n = 0
const row = (step, got, want) => out.push({ check: n, step, got: JSON.stringify(got), want: JSON.stringify(want) })
const make = async (attrs) => {
	const el = document.createElement('sb-data-table')
	for (const [k, v] of Object.entries(attrs)) el.setAttribute(k, typeof v === 'string' ? v : JSON.stringify(v))
	el.style.blockSize = '20rem'
	document.body.append(el)
	await settle()
	return el
}
const $ = (el, s) => el.shadowRoot.querySelector(s)
const $$ = (el, s) => [...el.shadowRoot.querySelectorAll(s)]
const texts = (el, j = 0) => $$(el, '.row').map((r) => r.querySelectorAll('.cell')[j]?.textContent)
const resolved = (u) => new URL(u, document.baseURI).href
const checks = []
const check = (f) => checks.push(f)
// An error anywhere on the page (a component's effect, an action) fails the check it happened in.
addEventListener('error', (e) => out.push({ check: n, error: 'page: ' + (e.error?.stack || e.message) }))
addEventListener('unhandledrejection', (e) => out.push({ check: n, error: 'page: ' + (e.reason?.stack || e.reason) }))

check(async () => {
	const el = await make({ columns: [{ key: 'name' }], rows: [
		{ id: 1, name: { value: 'Io', href: '/moons/io' } },
		{ id: 2, name: { value: 'Mail', href: 'mailto:io@example.com' } },
		{ id: 3, name: { value: 'Ext', href: 'https://example.com/a b' } },
		{ id: 4, name: { value: 'Frag', href: '#io' } },
	] })
	const as = $$(el, '.row a')
	row('hrefs are resolved', as.map((a) => a.getAttribute('href')), ['/moons/io', 'mailto:io@example.com', 'https://example.com/a b', '#io'].map(resolved))
	row('part has link', as.map((a) => a.getAttribute('part').split(' ').includes('link')), [true, true, true, true])
	row('tabindex', as.map((a) => a.getAttribute('tabindex')), ['-1', '-1', '-1', '-1'])
	el.remove()
})

check(async () => {
	const bad = ['javascript:alert(1)', ' JavaScript:alert(1)', 'data:text/html,x', 'vbscript:x', 'blob:x', 'file:///etc/passwd', 'http://[::1', 'java\tscript:alert(1)']
	const el = await make({ columns: [{ key: 'name' }], rows: bad.map((h, i) => ({ id: i, name: { value: 'v' + i, href: h } })) })
	row('no href', $$(el, '.row a').map((a) => a.hasAttribute('href') || a.hasAttribute('tabindex')), bad.map(() => false))
	row('the text stays', texts(el), bad.map((_, i) => 'v' + i))
	el.remove()
})

check(async () => {
	window.__sbPwned = 0
	const img = '<img src=x onerror="window.__sbPwned = 1">'
	const expr = "$$sel = ['x']", post = "@post('/__sb-never')"
	const el = await make({ selection: 'multiple', columns: [{ key: 'a' }, { key: 'b' }, { key: 'c' }], rows: [
		{ id: 'r1', a: { value: img }, b: { text: expr, suffix: post }, c: { value: 1, text: post, suffix: img, href: '#x', tone: 'info' } },
		{ id: 'r2', a: img, b: expr, c: post },
	] })
	await settle(200)
	row('no img in the shadow root', el.shadowRoot.querySelectorAll('img').length, 0)
	row('onerror never ran', window.__sbPwned, 0)
	row('the texts are the strings', $$(el, '.row').map((r) => [...r.querySelectorAll('.cell')].map((c) => c.textContent)), [[img, expr + post, post + img], [img, expr, post]])
	row('the selection is untouched', el.selected, [])
	row('nothing was posted', performance.getEntriesByType('resource').some((e) => e.name.includes('__sb-never')), false)
	el.remove()
})

check(async () => {
	const el = await make({ columns: [{ key: 'a' }], rows: [
		{ id: 1, a: { value: 5, suffix: 'km' } },
		{ id: 2, a: { value: 'ok', tone: 'success' } },
		{ id: 3, a: { value: 'odd', tone: 'purple' } },
		{ id: 4, a: { value: 'plain' } },
	] })
	const cells = $$(el, '.row .cell')
	const suffix = (c) => c.querySelector('.suffix')
	row('suffix part and text', [suffix(cells[0]).getAttribute('part'), suffix(cells[0]).textContent], ['suffix', 'km'])
	row('the suffix shows, empty ones do not', cells.map((c) => getComputedStyle(suffix(c)).display !== 'none'), [true, false, false, false])
	row('tone', cells.map((c) => c.querySelector('a').getAttribute('data-tone')), [null, 'success', null, null])
	row('parts', cells.map((c) => c.querySelector('a').getAttribute('part')), ['text', 'text badge', 'text', 'text'])
	el.remove()
})

check(async () => {
	const el = await make({ columns: [{ key: 'name' }, { key: 'radius', sortable: true }], rows: [
		{ id: 'io', name: 'Io', radius: { value: 1821.6, suffix: 'km' } },
		{ id: 'x', name: 'X', radius: { text: '' } },
		{ id: 'amalthea', name: 'Amalthea', radius: { value: 83.5, suffix: 'km' } },
		{ id: 'ganymede', name: 'Ganymede', radius: { value: 2634.1, suffix: 'km' } },
		{ id: 'europa', name: 'Europa', radius: 1560.8 },
	] })
	const button = $$(el, '.th button')[1]
	button.click()
	await settle()
	row('ascending by value, missing last', texts(el), ['Amalthea', 'Europa', 'Io', 'Ganymede', 'X'])
	button.click()
	await settle()
	row('descending by value, missing last', texts(el), ['Ganymede', 'Io', 'Europa', 'Amalthea', 'X'])
	el.remove()
})

check(async () => {
	const el = await make({ 'row-height': '36', columns: [{ key: 'a' }, { key: 'b' }, { key: 'c', width: '5rem' }], rows: [...Array(8).keys()].map((i) => ({
		id: i,
		a: { value: 'Star ' + i, href: '#s' + i, suffix: 'a long muted suffix that overflows the cell' },
		b: { value: i, tone: ['info', 'success', 'warning', 'danger', 'neutral'][i % 5] },
		c: { value: i * 1000, suffix: 'ly', tone: 'info' },
	})) })
	const hs = $$(el, '.row').map((r) => r.getBoundingClientRect().height)
	row('rows rendered', hs.length, 8)
	row('every row is row-height tall', [...new Set(hs)], [36])
	el.remove()
})

check(async () => {
	const el = await make({ selection: 'single', columns: [{ key: 'name' }, { key: 'n' }], rows: [{ id: 'io', name: { value: 'Io', href: '#moon-io' }, n: 1 }] })
	const got = [], acts = []
	let cancel = true
	el.addEventListener('sb-cell-activate', (e) => (got.push({ ...e.detail, cancelable: e.cancelable }), cancel && e.preventDefault()))
	el.addEventListener('sb-row-activate', (e) => acts.push(e.detail))
	const before = location.href
	const a = $(el, '.row a[href]')
	const click = (type, detail) => a.dispatchEvent(new MouseEvent(type, { bubbles: true, cancelable: true, composed: true, detail }))
	click('click', 1)
	click('click', 2)
	click('dblclick', 2)
	await settle()
	const detail = { key: 'io', column: 'name', value: 'Io', href: resolved('#moon-io'), cancelable: true }
	row('a click emits sb-cell-activate once per click', got, [detail, detail])
	row('a link click does not select', el.selected, [])
	row('a double click on a link is no row activation', acts, [])
	row('preventDefault keeps the page', location.href, before)
	got.length = 0
	const aux = (button) => { const e = new MouseEvent('auxclick', { bubbles: true, cancelable: true, composed: true, button }); a.dispatchEvent(e); return e.defaultPrevented }
	row('a middle click emits it and preventDefault keeps the new tab closed', [aux(1), aux(2), got], [true, false, [detail]])
	got.length = 0
	const cells = $$(el, '.row .cell')
	cells[0].focus()
	cells[0].dispatchEvent(new KeyboardEvent('keydown', { key: 'Enter', bubbles: true, composed: true, cancelable: true }))
	await settle()
	row('Enter on a link cell emits it', got, [detail])
	cells[1].focus()
	cells[1].dispatchEvent(new KeyboardEvent('keydown', { key: 'Enter', bubbles: true, composed: true, cancelable: true }))
	await settle()
	row('Enter on a text cell emits sb-row-activate', acts, [{ key: 'io' }])
	cells[0].focus()
	cells[0].dispatchEvent(new KeyboardEvent('keydown', { key: ' ', bubbles: true, composed: true, cancelable: true }))
	await settle()
	row('Space on a link cell selects the row', el.selected, ['io'])
	cancel = false
	a.click()
	await settle()
	row('without preventDefault the link is followed', location.hash, '#moon-io')
	history.replaceState(null, '', before)
	el.remove()
})

check(async () => {
	const loops = (el) =>
		$$(el, '.grid > template').filter((t) => t.getAttribute('data-for').startsWith('r in')).map((t) => {
			let rows = 0
			for (let x = t.nextSibling; x && !(x.nodeType === 8 && x.data === 'rocket-for:end'); x = x.nextSibling) rows += x.nodeType === 1 && x.classList.contains('row')
			return rows
		})
	const plain = [{ id: 1, a: 'x', b: 2 }, { id: 2, a: 'y', b: 3 }]
	const el = await make({ columns: [{ key: 'a' }, { key: 'b' }], rows: plain })
	const bare = () => $$(el, '.row .cell').every((c) => c.childElementCount === 0 && c.hasAttribute('data-text'))
	row('plain rows: the plain loop', loops(el), [2, 0])
	row('plain cells: text only', bare(), true)
	el.setAttribute('rows', JSON.stringify([plain[0], { id: 2, a: 'y', b: { value: 3, suffix: 'x' } }]))
	await settle()
	row('one cell object: the rich loop', loops(el), [0, 2])
	row('rich cells: text and suffix', $$(el, '.row .cell').map((c) => c.querySelector('.text').textContent + '|' + c.querySelector('.suffix').textContent), ['x|', '⁨2⁩|', 'y|', '⁨3⁩|x'])
	el.setAttribute('rows', JSON.stringify(plain))
	await settle()
	row('back to plain', loops(el), [2, 0])
	row('plain cells again', bare(), true)
	el.remove()
})

check(async () => {
	const cols = [{ key: 'a', label: 'A' }, { key: 'b', label: 'B' }, { key: 'c', label: 'C' }]
	const el = await make({ 'column-picker': '', 'hidden-columns': ['b'], selection: 'multiple', confirm: '', selected: [], columns: cols,
		rows: [{ id: 1, a: 'a1', b: 'b1', c: 'c1' }, { id: 2, a: 'a2', b: 'b2', c: 'c2' }] })
	const heads = () => $$(el, '.th').map((t) => t.lastElementChild.textContent)
	const shows = (x) => getComputedStyle(x).display !== 'none'
	const toolbar = $(el, '.toolbar'), button = $(el, '.columns'), menu = $(el, '.menu')
	const boxes = () => $$(el, '.menu input')
	const focused = () => boxes().indexOf(el.shadowRoot.activeElement)
	const key = (k, shiftKey = false) => el.shadowRoot.activeElement.dispatchEvent(new KeyboardEvent('keydown', { key: k, shiftKey, bubbles: true, composed: true, cancelable: true }))
	const got = []
	el.addEventListener('sb-columns', (e) => got.push(e.detail))
	row('hidden columns leave the header', heads(), ['A', 'C'])
	row('and the rows', $$(el, '.row').map((r) => [...r.querySelectorAll('.cell')].map((c) => c.textContent)), [['a1', 'c1'], ['a2', 'c2']])
	row('the toolbar shows its Columns button', [shows(toolbar), shows(button), button.textContent], [true, true, 'Columns'])
	row('the button', ['aria-haspopup', 'aria-controls', 'aria-expanded', 'popovertarget'].map((a) => button.getAttribute(a)), ['dialog', 'columns', 'false', 'columns'])
	row('the menu', ['popover', 'role', 'aria-modal', 'aria-label'].map((a) => menu.getAttribute(a)), ['auto', 'dialog', 'true', 'Columns'])
	const forced = [...el.shadowRoot.adoptedStyleSheets].flatMap((s) => [...s.cssRules]).filter((r) => r.conditionText === '(forced-colors: active)').flatMap((r) => [...r.cssRules])
	row('forced colours mark the open button and a hovered option', ['.columns:is(:hover, [aria-expanded="true"])', '.option:not(:has(:disabled)):hover'].map((sel) => forced.some((r) => r.selectorText === sel)), [true, true])

	button.click()
	await settle()
	row('the menu opens', [menu.matches(':popover-open'), button.getAttribute('aria-expanded')], [true, 'true'])
	row('the focus lands on the first checkbox', focused(), 0)
	row('the options', $$(el, '.option').map((o) => [o.getAttribute('part'), o.textContent]), [['column-option', 'A'], ['column-option', 'B'], ['column-option', 'C']])
	row('checked follows the list', boxes().map((b) => b.checked), [true, false, true])
	key('Tab')
	row('Tab moves on', focused(), 1)
	key('Tab'), key('Tab')
	row('Tab cycles', focused(), 0)
	key('Tab', true)
	row('Shift+Tab cycles back', focused(), 2)
	menu.focus()
	key('Tab')
	row('Tab from the menu itself: the first', focused(), 0)
	menu.focus()
	key('Tab', true)
	row('Shift+Tab from the menu itself: the last', focused(), 2)
	boxes()[2].click()
	await settle()
	row('sb-columns with the whole list', got, [{ name: '', hidden: ['b', 'c'] }])
	row('the column is gone', heads(), ['A'])
	row('the focus stays on the checkbox', focused(), 2)
	row('the last shown column cannot be hidden', boxes().map((b) => b.disabled), [true, false, false])
	row('the table never writes the attribute', el.getAttribute('hidden-columns'), '["b"]')
	row('the property is the local list', el.hiddenColumns, ['b', 'c'])
	row('hidden columns are not pending', el.matches(':state(pending)'), false)
	$$(el, '.row')[0].click()
	await settle()
	row('a selection is', [el.selected, el.matches(':state(pending)')], [['1'], true])
	boxes()[1].click()
	await settle()
	row('a column change leaves it pending', [heads(), el.matches(':state(pending)')], [['A', 'B'], true])
	el.revert()
	await settle()
	row('revert() keeps the hidden columns', [el.selected, el.matches(':state(pending)'), el.hiddenColumns, heads()], [[], false, ['c'], ['A', 'B']])

	el.setAttribute('hidden-columns', '["a"]')
	await settle()
	row('the server\'s list while open: checked', boxes().map((b) => b.checked), [false, true, true])
	row('and the grid', heads(), ['B', 'C'])
	row('no sb-columns for it', got.length, 2)
	el.hiddenColumns = ['a', 'c']
	await settle()
	row('a property write: the grid, no event, no attribute', [heads(), got.length, el.getAttribute('hidden-columns')], [['B'], 2, '["a"]'])
	el.setAttribute('hidden-columns', '["a","b","c","gone"]')
	await settle()
	row('every column hidden: the first shows', [heads(), boxes().map((b) => [b.checked, b.disabled])], [['A'], [[true, true], [false, false], [false, false]]])
	boxes()[2].click()
	await settle()
	row('picking another keeps the one shown', [heads(), got.at(-1).hidden], [['A', 'C'], ['b', 'gone']])

	const options = () => $$(el, '.option').map((o) => o.textContent)
	el.setAttribute('hidden-columns', '[]')
	await settle()
	boxes()[2].focus()
	el.setAttribute('columns', JSON.stringify([cols[1], cols[2]]))
	await settle()
	row('a column dropped before the focused one: the focus stays with its column', [options(), focused()], [['B', 'C'], 1])
	el.setAttribute('columns', JSON.stringify(cols))
	await settle()
	row('and when it comes back', [options(), focused()], [['A', 'B', 'C'], 2])
	el.setAttribute('columns', JSON.stringify(cols.slice(0, 2)))
	await settle()
	row('the focused column dropped: its neighbour', [options(), focused(), menu.matches(':popover-open')], [['A', 'B'], 1, true])
	el.setAttribute('columns', JSON.stringify(cols.slice(0, 1)))
	await settle()
	row('no checkbox left to focus: the menu', [boxes().map((b) => b.disabled), el.shadowRoot.activeElement === menu], [[true], true])
	el.setAttribute('columns', JSON.stringify(cols))
	await settle()

	const seen = []
	const onKey = (e) => seen.push(e.key)
	window.addEventListener('keydown', onKey)
	key('Escape')
	window.removeEventListener('keydown', onKey)
	row('Escape in the menu goes no further', seen, [])
	menu.hidePopover()
	await settle()
	row('closing returns the focus to the button', [el.shadowRoot.activeElement === button, button.getAttribute('aria-expanded')], [true, 'false'])
	button.click()
	await settle()
	el.removeAttribute('column-picker')
	await settle()
	const stop = $(el, '.grid [tabindex="0"]')
	row('the picker taken away while open: the focus goes to the grid', [menu.matches(':popover-open'), shows(button), !!stop && el.shadowRoot.activeElement === stop], [false, false, true])
	el.remove()
	await settle()
	const next = await make({ columns: cols, rows: [{ id: 1, a: 'a1' }] })
	row('a removed table with confirm leaves the next one rendering', $$(next, '.row').length, 1)
	next.remove()

	const two = [{ key: 'a' }, { key: 'b' }]
	const bare = await make({ columns: two, rows: [{ id: 1, a: 1, b: 2 }] })
	row('no picker, nothing slotted: no toolbar', getComputedStyle($(bare, '.toolbar')).display, 'none')
	bare.remove()
	// In a shadow root of its own, away from the page's [hidden] rule.
	const wrap = document.createElement('div')
	wrap.attachShadow({ mode: 'open' })
	document.body.append(wrap)
	const tools = document.createElement('sb-data-table')
	tools.setAttribute('columns', JSON.stringify(two))
	tools.innerHTML = '<button slot="toolbar">CSV</button>'
	wrap.shadowRoot.append(tools)
	await settle()
	const bar = () => getComputedStyle($(tools, '.toolbar')).display !== 'none'
	row('a slotted control: the toolbar, without the Columns button', [bar(), getComputedStyle($(tools, '.columns')).display], [true, 'none'])
	const csv = tools.querySelector('button')
	csv.hidden = true
	await settle()
	row('its only control hidden: no toolbar', bar(), false)
	row('and the control stays hidden', getComputedStyle(csv).display, 'none')
	csv.hidden = false
	await settle()
	row('shown again: the toolbar', bar(), true)
	csv.style.display = 'none'
	await settle()
	row('hidden by data-show: no toolbar', bar(), false)
	wrap.remove()

	await customElements.whenDefined('sb-drawer')
	const drawer = document.createElement('sb-drawer')
	drawer.setAttribute('modal', 'false')
	drawer.innerHTML = '<button id="dt-other">x</button>'
	const table = document.createElement('sb-data-table')
	for (const [k, v] of Object.entries({ 'column-picker': '', columns: JSON.stringify(two) })) table.setAttribute(k, v)
	drawer.append(table)
	document.body.append(drawer)
	await settle()
	drawer.show()
	await settle(300)
	$(table, '.columns').click()
	await settle()
	table.shadowRoot.activeElement.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape', bubbles: true, composed: true, cancelable: true }))
	await settle()
	row('Escape in the menu leaves a drawer open', drawer.isOpen, true)
	document.getElementById('dt-other').dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape', bubbles: true, composed: true, cancelable: true }))
	await settle()
	row('Escape elsewhere in it closes the drawer', drawer.isOpen, false)
	drawer.remove()
})

check(async () => {
	const el = await make({ name: 'stars', selection: 'multiple', 'hidden-columns': ['b'],
		columns: [{ key: 'a', sortable: true }, { key: 'b' }, { key: 'c', sortable: true }], rows: [{ id: 1, a: 'x', b: 1, c: 3 }, { id: 2, a: 'y', b: 2, c: 1 }] })
	const got = []
	el.addEventListener('sb-export', (e) => got.push({ ...e.detail, bubbles: e.bubbles, composed: e.composed }))
	const button = $$(el, '.th button')[1]
	button.click()
	await settle()
	button.click()
	await settle()
	$$(el, '.row')[0].click()
	await settle()
	el.requestExport('json')
	el.requestExport()
	const want = { name: 'stars', format: 'json', sort: { key: 'c', dir: 'desc' }, columns: ['a', 'c'], selected: ['1'], bubbles: true, composed: true }
	row('sb-export: the local order, the shown columns, the selection', got, [want, { ...want, format: 'csv' }])
	el.remove()

	const server = await make({ columns: [{ key: 'a', sortable: true }], rows: [{ id: 1, a: 'x' }, { id: 2, a: 'y' }], total: 100, sort: { key: 'a', dir: 'asc' } })
	const asked = []
	server.addEventListener('sb-sort', (e) => asked.push(e.detail.dir))
	server.addEventListener('sb-export', (e) => asked.push(e.detail.sort))
	$(server, '.th button').click()
	await settle()
	server.requestExport()
	row('an unanswered sb-sort is not the order on screen', asked, ['desc', { key: 'a', dir: 'asc' }])
	server.remove()
})

for (const [i, f] of checks.entries()) {
	n = i + 1
	try {
		await f()
	} catch (e) {
		out.push({ check: n, error: String(e.stack || e) })
	}
}
await fetch('/__probe/result', { method: 'POST', body: JSON.stringify(out) })
`
