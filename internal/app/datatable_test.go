package app_test

import (
	"encoding/json"
	"testing"
)

// TestDataTableCells checks sb-data-table's cell objects in the browser, one
// row per check: links only for http, https and mailto, text that is never
// markup or an expression, suffixes and badges, the local sort by value, a
// fixed row height, sb-cell-activate, and the plain loop kept for plain rows.
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
