package app_test

import (
	"net/http"
	"os"
	"testing"
)

// TestTreeMarkup checks sb-tree's server-markup form in Chrome: the list works
// before JavaScript (a sandboxed frame without scripts) and renders as rows
// after; link rows are <a href> whose clicks, modifier clicks, middle clicks,
// Enter and Space are the browser's; aria-current, icons and meta; the keys;
// morphs of the list keep the user's open branches and focus (a dropped row
// hands it to its neighbour), while a changed data-open or a new current page
// win; lazy branches load through sb-load and a patched-in list, and keep
// their children when a morph sends them again without one; long labels are
// cut in a narrow tree; value and
// expanded are server-wins; the JSON form still works, and a list wins over it.
func TestTreeMarkup(t *testing.T) {
	copyRows(t, probeCDP(t, "/", pdPrelude+morphJS+treeMarkupJS, map[string]http.HandlerFunc{"/__test/morph": morphHandler}))
}

const treeMarkupJS = `
let at_ = ''
const group = async (name, fn) => {
	at_ = name
	try {
		await fn()
	} catch (e) {
		rows.push({ step: name, error: String(e?.stack || e) })
	}
}
setTimeout(() => (rows.push({ step: 'the script', error: 'stuck in ' + at_ }), report()), 100000)
const cdp = (...cmds) => fetch('/__probe/cdp', { method: 'POST', body: JSON.stringify(cmds) }).then((r) => r.json())
const at = (el) => {
	const r = el.getBoundingClientRect()
	return { x: r.left + r.width / 2, y: r.top + r.height / 2 }
}
const mouse = async (el, { button = 'left', modifiers = 0 } = {}) => {
	const { x, y } = at(el)
	await cdp(
		{ method: 'Input.dispatchMouseEvent', params: { type: 'mouseMoved', x, y } },
		{ method: 'Input.dispatchMouseEvent', params: { type: 'mousePressed', x, y, button, clickCount: 1, modifiers } },
		{ method: 'Input.dispatchMouseEvent', params: { type: 'mouseReleased', x, y, button, clickCount: 1, modifiers } },
	)
	await settle()
}
const KEYS = { Enter: ['Enter', 13, '\r'], ' ': ['Space', 32, ' '] }
const key = async (k) => {
	const [code, vk, text] = KEYS[k]
	await cdp(
		{ method: 'Input.dispatchKeyEvent', params: { type: 'keyDown', key: k, code, windowsVirtualKeyCode: vk, text } },
		{ method: 'Input.dispatchKeyEvent', params: { type: 'keyUp', key: k, code, windowsVirtualKeyCode: vk } },
	)
	await settle()
}
// Clicks as the page sees them last; the test stops them from navigating.
const clicks = []
for (const type of ['click', 'auxclick'])
	addEventListener(type, (e) => {
		const a = e.composedPath().find((n) => n.localName === 'a')
		if (!a || !e.composedPath().some((n) => n.localName === 'sb-tree')) return
		clicks.push([type, e.button, e.ctrlKey, e.defaultPrevented, a.getAttribute('href')])
		e.preventDefault()
	})

const box = document.createElement('div')
box.style.cssText = 'position: fixed; inset-block-start: 0; inset-inline-start: 0; z-index: 99999; inline-size: 20rem; background: #000'
document.body.append(box)
const make = async (html) => {
	box.innerHTML = html
	await settle()
	return box.firstElementChild
}
const rowsOf = (el) => [...el.shadowRoot.querySelectorAll('.row')]
const labels = (el) => rowsOf(el).map((r) => r.querySelector('.label').textContent)
const rowOf = (el, label) => rowsOf(el).find((r) => r.querySelector('.label').textContent === label)
const focused = (el) => (document.activeElement === el ? el.shadowRoot.activeElement?.querySelector('.label')?.textContent : null) ?? null
const keydown = (el, k, init) => press(el.shadowRoot.activeElement, k, init)
const frames = () => new Promise((r) => requestAnimationFrame(() => requestAnimationFrame(r)))

const SVG = '<svg data-icon viewBox="0 0 16 16" width="16" height="16" fill="currentColor" aria-hidden="true"><path d="M3 1h7v1h1v1h1v1h1v11H3z"/></svg>'
const nav = ({ top = '', api = true, refOpen = false, current = 'install' } = {}) =>
	'<sb-tree id="nav" label="Docs"><ul>' + top +
	'<li data-id="guide"><a href="/__nav/guide">' + SVG + ' Guide <span data-meta>3</span></a><ul>' +
	'<li><a href="/__nav/start"' + (current === 'start' ? ' aria-current="page"' : '') + '>Getting started</a></li>' +
	'<li><a href="/__nav/install"' + (current === 'install' ? ' aria-current="page"' : '') + '>Install</a></li>' +
	'<li data-id="ideas"><span>Ideas</span> <span data-meta>draft</span></li></ul></li>' +
	'<li data-id="ref"' + (refOpen ? ' data-open' : '') + '><a href="/__nav/ref">Reference</a>' +
	(api ? '<ul><li><a href="/__nav/api"' + (current === 'api' ? ' aria-current="page"' : '') + '>API</a></li></ul>' : '') + '</li>' +
	'<li><a href="/__nav/faq">FAQ</a></li>' +
	'</ul></sb-tree>'

await customElements.whenDefined('sb-tree')

await group('before JavaScript', async () => {
	const f = document.createElement('iframe')
	f.setAttribute('sandbox', 'allow-same-origin')
	f.style.cssText = 'inline-size: 400px; block-size: 400px'
	f.srcdoc = '<!doctype html><script type="module" src="/c/bundle.js"><' + '/script>' + nav()
	const loaded = new Promise((r) => (f.onload = r))
	document.body.append(f)
	await loaded
	const d = f.contentDocument
	check('pre-JS: not upgraded', !!f.contentWindow.customElements.get('sb-tree'), false)
	check('pre-JS: every link shows, with its href', [...d.querySelectorAll('sb-tree a')].filter((a) => a.getClientRects().length).map((a) => a.getAttribute('href')),
		['/__nav/guide', '/__nav/start', '/__nav/install', '/__nav/ref', '/__nav/api', '/__nav/faq'])
	check('pre-JS: the icon is the SVG at its own size', d.querySelector('svg[data-icon]').getBoundingClientRect().width, 16)
	f.remove()
})

let tree
await group('after the upgrade', async () => {
	tree = await make(nav())
	check('the source list is not shown', tree.querySelector('ul').getClientRects().length, 0)
	check('rows, the current page\'s branch open', labels(tree), ['Guide', 'Getting started', 'Install', 'Ideas', 'Reference', 'FAQ'])
	check('every row an <a>, links with their href', rowsOf(tree).map((r) => [r.localName, r.getAttribute('href')]),
		[['a', '/__nav/guide'], ['a', '/__nav/start'], ['a', '/__nav/install'], ['a', null], ['a', '/__nav/ref'], ['a', '/__nav/faq']])
	const cur = rowOf(tree, 'Install')
	check('aria-current, the current part, the only tab stop', [cur.getAttribute('aria-current'), cur.getAttribute('part'), rowsOf(tree).filter((r) => r.tabIndex === 0).map((r) => r.querySelector('.label').textContent)], ['page', 'item current', ['Install']])
	check('aria-selected only on plain items', rowsOf(tree).map((r) => r.getAttribute('aria-selected')), [null, null, null, 'false', null, null])
	check('levels and expanded', rowsOf(tree).map((r) => [r.getAttribute('aria-level'), r.getAttribute('aria-expanded')]),
		[['1', 'true'], ['2', null], ['2', null], ['2', null], ['1', 'false'], ['1', null]])
	check('meta, outside the label', rowsOf(tree).map((r) => [r.querySelector('.label').textContent, r.querySelector('.meta').textContent]),
		[['Guide', '3'], ['Getting started', ''], ['Install', ''], ['Ideas', 'draft'], ['Reference', ''], ['FAQ', '']])
	const art = rowOf(tree, 'Guide').querySelector('.art svg')
	const w = art?.getBoundingClientRect().width
	check('the icon is a copy of the SVG, at the icon size', [!!art?.querySelector('path'), w > 12 && w < 24, rowOf(tree, 'FAQ').querySelector('.art').children.length], [true, true, 0])
	check('a page stylesheet is not needed: the rows are styled', getComputedStyle(cur).display + ' ' + getComputedStyle(cur).textDecorationLine, 'flex none')
})

await group('link rows are the browser\'s', async () => {
	const faq = rowOf(tree, 'FAQ').querySelector('.label')
	await mouse(faq)
	await mouse(faq, { modifiers: 2 })
	await mouse(faq, { button: 'middle' })
	check('a click, a Ctrl+click and a middle click go through', clicks.splice(0), [
		['click', 0, false, false, '/__nav/faq'],
		['click', 0, true, false, '/__nav/faq'],
		['auxclick', 1, false, false, '/__nav/faq'],
	])
	rowOf(tree, 'Install').focus()
	await key('Enter')
	rowOf(tree, 'Getting started').focus()
	await key(' ')
	check('Enter and Space follow the link', clicks.splice(0), [['click', 0, false, false, '/__nav/install'], ['click', 0, false, false, '/__nav/start']])
	const changes = []
	tree.addEventListener('sb-change', (e) => changes.push(e.detail.value))
	rowOf(tree, 'Ideas').focus()
	await key('Enter')
	check('Enter selects a plain item', [tree.value, changes, rowOf(tree, 'Ideas').getAttribute('aria-selected'), clicks.splice(0)], ['ideas', ['ideas'], 'true', []])
	await mouse(rowOf(tree, 'Reference').querySelector('.caret'))
	check('the caret opens a link row without following it', [clicks.splice(0), labels(tree).includes('API')], [[['click', 0, false, true, '/__nav/ref']], true])
})

await group('keys', async () => {
	const toggles = []
	tree.addEventListener('sb-toggle', (e) => toggles.push(e.detail.id + ':' + e.detail.open))
	rowOf(tree, 'Install').focus()
	await settle()
	const steps = []
	for (const k of ['ArrowDown', 'ArrowUp', 'ArrowLeft', 'ArrowLeft', 'ArrowRight', 'ArrowRight', 'End', 'Home']) {
		keydown(tree, k)
		await settle()
		steps.push(focused(tree))
	}
	check('Down, Up, Left to the parent, Left closes, Right opens, Right to the child, End, Home', steps, ['Ideas', 'Install', 'Guide', 'Guide', 'Guide', 'Getting started', 'FAQ', 'Guide'])
	check('sb-toggle', toggles, ['guide:false', 'guide:true'])
	check('Alt+Left is left to the browser', keydown(tree, 'ArrowLeft', { altKey: true }), true)
})

await group('a morph keeps the user\'s branches and focus', async () => {
	// The user: Guide closed, Reference open, the focus on API.
	rowOf(tree, 'Guide').focus()
	keydown(tree, 'ArrowLeft')
	await settle()
	rowOf(tree, 'API').focus()
	await settle()
	await morph(nav({ top: '<li><a href="/__nav/news">News</a></li>' }))
	await frames()
	check('an item arrives above: same branches, same focus', [labels(tree), focused(tree)], [['News', 'Guide', 'Reference', 'API', 'FAQ'], 'API'])
	await morph(nav({ top: '<li><a href="/__nav/news">News</a></li>', api: false }))
	await frames()
	check('the focused row is dropped: the focus goes to its neighbour', [labels(tree), focused(tree)], [['News', 'Guide', 'Reference', 'FAQ'], 'FAQ'])
})

await group('server-wins: data-open and the current page', async () => {
	await morph(nav())
	check('API returns under the Reference the user opened', labels(tree), ['Guide', 'Reference', 'API', 'FAQ'])
	rowOf(tree, 'Reference').focus()
	keydown(tree, 'ArrowLeft')
	await settle()
	check('the user closed Reference', labels(tree), ['Guide', 'Reference', 'FAQ'])
	await morph(nav({ refOpen: true }))
	check('a new data-open opens it', labels(tree), ['Guide', 'Reference', 'API', 'FAQ'])
	rowOf(tree, 'Reference').focus()
	keydown(tree, 'ArrowLeft')
	await settle()
	await morph(nav({ refOpen: true }))
	check('the same data-open again keeps the user\'s choice', labels(tree), ['Guide', 'Reference', 'FAQ'])
	await morph(nav({ refOpen: true, current: 'start' }))
	check('a new current page opens the branches above it', [labels(tree), rowOf(tree, 'Getting started').getAttribute('aria-current'), rowOf(tree, 'Install').hasAttribute('aria-current')],
		[['Guide', 'Getting started', 'Install', 'Ideas', 'Reference', 'FAQ'], 'page', false])
})

await group('lazy branches', async () => {
	const sky = await make('<sb-tree id="sky" label="Sky" data-on:sb-load="@get(\'/demo/data/children?parent=\' + evt.detail.id + \'&tree=sky&delay=400\')"><ul><li data-id="solarsystem" data-lazy>Solar System</li><li data-id="plain">Plain</li></ul></sb-tree>')
	const loads = []
	sky.addEventListener('sb-load', (e) => loads.push(e.detail.id))
	check('a lazy item is a closed branch', rowOf(sky, 'Solar System').getAttribute('aria-expanded'), 'false')
	rowOf(sky, 'Solar System').querySelector('.label').click()
	await settle(100)
	const r = rowOf(sky, 'Solar System')
	check('opening it asks with sb-load, busy meanwhile', [loads, r.getAttribute('aria-busy'), r.querySelector('.busy').getClientRects().length > 0], [['solarsystem'], 'true', true])
	await until(() => labels(sky).includes('Jupiter'))
	const j = rowOf(sky, 'Jupiter')
	check('the patched-in list becomes its children', [!!sky.querySelector('li[data-id="solarsystem"] > ul'), rowOf(sky, 'Solar System').hasAttribute('aria-busy'), j?.getAttribute('aria-level'), j?.getAttribute('aria-expanded'), Number(j?.querySelector('.meta').textContent) > 0, !!j?.querySelector('.art svg')],
		[true, false, '2', 'false', true, true])
	check('one request', loads, ['solarsystem'])
})

await group('lazy branches in a shared view', async () => {
	const src = (list = '') => '<sb-tree id="shared" label="Shared"><ul><li data-id="docs" data-lazy>Docs' + list + '</li><li data-id="faq">FAQ</li></ul></sb-tree>'
	const kids = {
		docs: '<ul><li data-id="intro"><a href="/__nav/intro" aria-current="page">Intro</a></li><li data-id="more" data-lazy>More</li></ul>',
		more: '<ul><li data-id="deep">Deep</li></ul>',
	}
	const t = await make(src())
	const loads = []
	t.addEventListener('sb-load', (e) => {
		loads.push(e.detail.id)
		setTimeout(() => t.querySelector('li[data-id="' + e.detail.id + '"]').insertAdjacentHTML('beforeend', kids[e.detail.id]))
	})
	rowOf(t, 'Docs').querySelector('.label').click()
	await until(() => labels(t).includes('More'))
	rowOf(t, 'More').querySelector('.label').click()
	await until(() => labels(t).includes('Deep'))
	check('two branches loaded', [labels(t), loads], [['Docs', 'Intro', 'More', 'Deep', 'FAQ'], ['docs', 'more']])
	await morph(src())
	check('the same items again without their lists: the children stay, open, with no new request',
		[!!t.querySelector('li > ul'), labels(t), rowOf(t, 'Docs').getAttribute('aria-expanded'), rowOf(t, 'More').getAttribute('aria-expanded'), loads],
		[false, ['Docs', 'Intro', 'More', 'Deep', 'FAQ'], 'true', 'true', ['docs', 'more']])
	check('a kept item is no longer the current page', rowOf(t, 'Intro').hasAttribute('aria-current'), false)
	await morph(src('<ul><li data-id="new">New</li></ul>'))
	check('a list from the server wins', labels(t), ['Docs', 'New', 'FAQ'])
	await morph(src())
	check('and is kept in turn', labels(t), ['Docs', 'New', 'FAQ'])
	await morph(src('<ul></ul>'))
	await morph(src())
	check('an empty list: no children, also when kept', [labels(t), rowOf(t, 'Docs').hasAttribute('aria-expanded'), loads], [['Docs', 'FAQ'], false, ['docs', 'more']])
	await morph('<sb-tree id="shared" label="Shared"><ul><li data-id="faq">FAQ</li></ul></sb-tree>')
	await morph(src())
	await until(() => labels(t).includes('Deep'))
	check('an item the server dropped loads again, into the branches the user opened', [labels(t), loads], [['Docs', 'Intro', 'More', 'Deep', 'FAQ'], ['docs', 'more', 'docs', 'more']])
})

await group('a narrow tree cuts long labels', async () => {
	const long = 'A label far too long for the sidebar it sits in'
	for (const [form, html] of [
		['markup', '<sb-tree label="Narrow"><ul><li data-id="x">' + SVG + ' ' + long + ' <span data-meta>42</span></li></ul></sb-tree>'],
		['JSON', '<sb-tree label="Narrow" items=\'[{"id":"x","label":"' + long + '","icon":"★"}]\'></sb-tree>'],
	]) {
		const side = await make('<div style="display: grid; grid-template-columns: 9rem 1fr">' + html + '</div>')
		const t = side.firstElementChild
		const row = rowsOf(t)[0], label = row.querySelector('.label'), meta = row.querySelector('.meta')
		const host = t.getBoundingClientRect(), r = row.getBoundingClientRect()
		check(form + ': the row fits the element, the label ends in an ellipsis', [Math.round(host.width), r.right <= host.right + 0.5, label.scrollWidth > label.clientWidth, getComputedStyle(label).textOverflow], [144, true, true, 'ellipsis'])
		if (form === 'markup') check('the meta stays in view', meta.getBoundingClientRect().width > 0 && meta.getBoundingClientRect().right <= host.right + 0.5, true)
	}
})

await group('server-wins: value and expanded', async () => {
	const pick = (attrs) => '<sb-tree id="vt" label="Pick"' + attrs + '><ul><li data-id="a">A<ul><li data-id="a1">A1</li></ul></li><li data-id="b">B</li></ul></sb-tree>'
	const vt = await make(pick(' value="b"'))
	check('the value', [vt.value, rowOf(vt, 'B').getAttribute('aria-selected')], ['b', 'true'])
	rowOf(vt, 'A').querySelector('.label').click()
	await settle()
	check('a click selects and opens', [vt.value, labels(vt)], ['a', ['A', 'A1', 'B']])
	await morph(pick(' value="b"'))
	check('the same value again keeps the user\'s', vt.value, 'a')
	await morph(pick(' value="a1"'))
	check('a new value wins', vt.value, 'a1')
	await morph(pick(' value=""'))
	check('value="" clears it', vt.value, '')
	await morph(pick(' value="" expanded=""'))
	check('expanded="" closes every branch', labels(vt), ['A', 'B'])
	rowOf(vt, 'A').focus()
	keydown(vt, 'ArrowRight')
	await settle()
	await morph(pick(' value="" expanded=""'))
	check('the same expanded again keeps the user\'s', labels(vt), ['A', 'A1', 'B'])
})

await group('the JSON form', async () => {
	const js = await make('<sb-tree id="js" expanded="p" value="c" items=\'[{"id":"p","label":"Parent","icon":"★","children":[{"id":"c","label":"Child"}]},{"id":"l","label":"Lazy","lazy":true}]\'></sb-tree>')
	check('rows, icon text, selection', [labels(js), rowOf(js, 'Parent').querySelector('.icon').textContent, rowOf(js, 'Child').getAttribute('aria-selected'), rowsOf(js).some((r) => r.hasAttribute('href'))],
		[['Parent', 'Child', 'Lazy'], '★', 'true', false])
	const loads = []
	js.addEventListener('sb-load', (e) => loads.push(e.detail.id))
	rowOf(js, 'Lazy').querySelector('.label').click()
	await settle()
	js.setAttribute('loaded', '{"l":[{"id":"x","label":"X"}]}')
	await settle()
	check('lazy through loaded', [loads, labels(js)], [['l'], ['Parent', 'Child', 'Lazy', 'X']])
	js.insertAdjacentHTML('beforeend', '<ul><li data-id="m">Markup</li></ul>')
	await settle()
	check('a list inside wins over items', labels(js), ['Markup'])
	js.querySelector('ul').remove()
	await settle()
	check('without it, items again', labels(js), ['Parent', 'Child', 'Lazy', 'X'])
})

check('no errors', errors, [])
await report()
`

// TestTreeDocsShots saves screenshots of sb-tree's docs examples in both
// themes and the smooth style to $STARBASE_SHOTS; it does nothing without it.
func TestTreeDocsShots(t *testing.T) {
	if os.Getenv("STARBASE_SHOTS") == "" {
		t.Skip("set STARBASE_SHOTS=<dir> to save the screenshots")
	}
	copyRows(t, probeCDP(t, "/components/tree", pdPrelude+treeShotsJS))
}

const treeShotsJS = `
const cdp = (...cmds) => fetch('/__probe/cdp', { method: 'POST', body: JSON.stringify(cmds) }).then((r) => r.json())
const shot = (name, el, pad = 16) => {
	const r = el.getBoundingClientRect()
	return cdp({ method: 'shot', params: { name, clip: { x: r.left + scrollX - pad, y: r.top + scrollY - pad, width: r.width + 2 * pad, height: r.height + 2 * pad } } })
}
try {
	await customElements.whenDefined('sb-tree')
	const nav = document.querySelector('sb-tree[label="Components"]')
	const sky = document.getElementById('sky')
	const files = document.querySelector('sb-tree.my-tree')
	await until(() => [...sky.shadowRoot.querySelectorAll('.label')].some((l) => l.textContent === 'Moon'))
	check('the lazy example loaded down to the Moon', true, true)
	const html = document.documentElement
	for (const [name, theme, style] of [['deep-space', 'deep-space', null], ['daylight', 'daylight', null], ['smooth', 'deep-space', 'smooth'], ['daylight-smooth', 'daylight', 'smooth']]) {
		html.setAttribute('data-sb-theme', theme)
		style ? html.setAttribute('data-sb-style', style) : html.removeAttribute('data-sb-style')
		await settle(300)
		for (const [what, el] of [['nav', nav], ['lazy', sky.parentElement], ['styled', files]]) await shot('tree-' + what + '-' + name, el)
	}
	check('no errors', errors, [])
} catch (e) {
	rows.push({ step: 'script', error: String(e?.stack || e) })
}
await report()
`
