package app_test

import "testing"

// TestDragGroupMove moves items of sb-drag-group's demo with the keyboard,
// within a list and to the other one, and checks what the server sends back.
func TestDragGroupMove(t *testing.T) {
	_, body := probe(t, "/components/drag-group", pdPrelude+dragGroupJS)
	copyRows(t, body)
}

const dragGroupJS = `
try {
	await customElements.whenDefined('sb-drag-group')
	const group = document.getElementById('sort-bodies')
	const item = (id) => group.querySelector('[data-drag-item="' + id + '"]')
	const lists = () => [...group.querySelectorAll('[data-drop-list]')].map((l) => l.dataset.dropList + '=' + [...l.querySelectorAll('[data-drag-item]')].map((i) => i.dataset.dragItem).join(',')).join(' ')
	const start = 'planets=earth,ceres,mars,jupiter dwarfs=pluto,venus'
	check('the server rendered the lists', [group.dataset.state, lists()], [start, start])

	// Within a list: Mars up, above Ceres.
	item('mars').focus()
	press(item('mars'), 'ArrowUp', { altKey: true })
	check('a staged move', group.hasAttribute('data-key-staging'), true)
	release('Alt')
	await until(() => group.dataset.state !== start)
	const within = 'planets=earth,mars,ceres,jupiter dwarfs=pluto,venus'
	check('a move within a list', [group.dataset.state, lists()], [within, within])
	check('the moved item keeps the focus', document.activeElement?.dataset?.dragItem, 'mars')

	// To the other list: Ceres right, then up, above Venus.
	item('ceres').focus()
	press(item('ceres'), 'ArrowRight', { altKey: true })
	press(item('ceres'), 'ArrowUp', { altKey: true })
	release('Alt')
	await until(() => group.dataset.state !== within)
	const across = 'planets=earth,mars,jupiter dwarfs=pluto,ceres,venus'
	check('a move to the other list', [group.dataset.state, lists()], [across, across])
	check('it keeps the focus there', document.activeElement?.dataset?.dragItem, 'ceres')

	// Escape cancels a staged move: nothing is sent.
	item('venus').focus()
	press(item('venus'), 'ArrowLeft', { altKey: true })
	press(item('venus'), 'Escape')
	release('Alt')
	await settle(400)
	check('a cancelled move', group.dataset.state, across)
	check('no errors', errors, [])
} catch (e) {
	rows.push({ step: 'script', error: String(e?.stack || e) })
}
await report()
`
