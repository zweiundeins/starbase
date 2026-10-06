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
	check('no errors', errors, [])
} catch (e) {
	rows.push({ step: 'script', error: String(e?.stack || e) })
}
await report()
`
