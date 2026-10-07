package app_test

import "testing"

// TestSortableListMove moves an item of sb-sortable-list's demo with the
// keyboard and checks the order the server sends back.
func TestSortableListMove(t *testing.T) {
	_, body := probe(t, "/components/sortable-list", pdPrelude+sortableListJS)
	copyRows(t, body)
}

const sortableListJS = `
try {
	await customElements.whenDefined('sb-sortable-list')
	const list = document.getElementById('inner-planets')
	const order = () => [...list.querySelectorAll('[data-sortable-item]')].map((i) => i.dataset.sortableItem).join(' ')
	check('the server rendered the order', [list.dataset.state, order()], ['earth mercury mars venus', 'earth mercury mars venus'])

	const earth = list.querySelector('[data-sortable-item="earth"]')
	earth.focus()
	press(earth, 'ArrowDown', { altKey: true })
	check('a staged move', list.hasAttribute('data-key-staging'), true)
	release('Alt')
	await until(() => list.dataset.state !== 'earth mercury mars venus')
	check('the server applied it', [list.dataset.state, order()], ['mercury earth mars venus', 'mercury earth mars venus'])
	check('the moved item keeps the focus', document.activeElement?.dataset?.sortableItem, 'earth')

	// Escape cancels a staged move: nothing is sent.
	const mars = list.querySelector('[data-sortable-item="mars"]')
	mars.focus()
	press(mars, 'ArrowUp', { altKey: true })
	press(mars, 'Escape')
	release('Alt')
	await settle(400)
	check('a cancelled move', list.dataset.state, 'mercury earth mars venus')

	// Another frame of the page (here: the next install tab) leaves the demo as the reader arranged it.
	const tab = () => document.querySelector('sb-tabs.install-tabs')?.getAttribute('selected')
	const was = tab()
	await fetch('/cmd/install-tab', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ tab: was === '1' ? 'autoloader' : 'component' }) })
	await until(() => tab() !== was, 60000) // the command queues behind the startup seeding, slow under -race
	check('a page frame keeps the arrangement', [tab() !== was, list.dataset.state, order()], [true, 'mercury earth mars venus', 'mercury earth mars venus'])
	check('no errors', errors, [])
} catch (e) {
	rows.push({ step: 'script', error: String(e?.stack || e) })
}
await report()
`

// TestSortableListDrag drags items of the demo with the pointer: onto an
// item, and just outside the list's ends (patches/pd-rockets 0010), which
// lands there; farther away a drop does nothing.
func TestSortableListDrag(t *testing.T) {
	_, body := probe(t, "/components/sortable-list", pdPrelude+sortableListDragJS)
	copyRows(t, body)
}

const sortableListDragJS = `
try {
	await customElements.whenDefined('sb-sortable-list')
	const list = () => document.getElementById('inner-planets')
	const item = (id) => list().querySelector('[data-sortable-item="' + id + '"]')
	const order = () => [...list().querySelectorAll('[data-sortable-item]')].map((i) => i.dataset.sortableItem).join(' ')
	const moves = []
	document.addEventListener('sb-sortable-move', (e) => moves.push(e.detail))
	const move = async (id, to, want) => {
		const before = list().dataset.state
		moves.length = 0
		await drag(item(id), to)
		await until(() => list().dataset.state !== before, want === before ? 600 : 15000)
		return [moves, list().dataset.state, order()]
	}
	const point = (id, y) => {
		const r = item(id).getBoundingClientRect()
		return { x: r.left + r.width / 2, y: y(r) }
	}
	list().scrollIntoView({ block: 'center' })

	let want = 'mercury earth mars venus'
	check('onto the upper half of an item', await move('earth', point('mars', (r) => r.top + r.height / 4), want), [[{ itemId: 'earth', before: 'mars' }], want, want])
	check('the drag cleaned up', [document.querySelector('[data-drag-preview]'), list().hasAttribute('data-drag-active'), list().querySelector('[data-dragging], [data-drop-before]')], [null, false, null])

	want = 'earth mars venus mercury'
	check('just below the last item', await move('mercury', point('venus', (r) => r.bottom + 10), want), [[{ itemId: 'mercury', before: '' }], want, want])

	want = 'venus earth mars mercury'
	check('just above the first item', await move('venus', point('earth', (r) => r.top - 10), want), [[{ itemId: 'venus', before: 'earth' }], want, want])

	check('far below the list', await move('earth', point('mercury', (r) => r.bottom + r.height * 3), want), [[], want, want])
	check('no errors', errors, [])
} catch (e) {
	rows.push({ step: 'script', error: String(e?.stack || e) })
}
await report()
`

// TestSortableListCard drags in the gallery card, which the server answers
// like the demo.
func TestSortableListCard(t *testing.T) {
	_, body := probe(t, "/?q=sortable+list", pdPrelude+sortableListCardJS)
	copyRows(t, body)
}

const sortableListCardJS = `
try {
	await customElements.whenDefined('sb-sortable-list')
	const list = () => document.getElementById('sortable-list-card')
	list().scrollIntoView({ block: 'center' })
	const earth = list().querySelector('[data-sortable-item="earth"]')
	const venus = list().querySelector('[data-sortable-item="venus"]').getBoundingClientRect()
	await drag(earth, { x: venus.left + venus.width / 2, y: venus.bottom + 4 })
	await until(() => list().dataset.state !== 'earth mars venus')
	check('the card moved earth to the end', [list().dataset.state, [...list().querySelectorAll('[data-sortable-item]')].map((i) => i.textContent)], ['mars venus earth', ['🪐 Mars', '🪐 Venus', '🪐 Earth']])
	check('no errors', errors, [])
} catch (e) {
	rows.push({ step: 'script', error: String(e?.stack || e) })
}
await report()
`
