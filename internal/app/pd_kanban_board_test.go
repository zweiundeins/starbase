package app_test

import "testing"

// TestKanbanBoardMove moves cards of sb-kanban-board's demo with the
// keyboard, within a lane and across lanes, and checks the board the server
// sends back.
func TestKanbanBoardMove(t *testing.T) {
	_, body := probe(t, "/components/kanban-board", pdPrelude+kanbanBoardJS)
	copyRows(t, body)
}

const kanbanBoardJS = `
try {
	await customElements.whenDefined('sb-kanban-board')
	const board = document.getElementById('mission-board')
	const card = (id) => board.querySelector('[data-kanban-card="' + id + '"]')
	const lanes = () => [...board.querySelectorAll('[data-kanban-lane]')].map((l) => [...l.querySelectorAll('[data-kanban-card]')].map((c) => c.dataset.kanbanCard).join(' ')).join(' | ')
	const start = 'mars jupiter neptune | europa titan | moon'
	check('the server rendered the board', [board.dataset.state, lanes()], [start, start])

	// The keyboard's focus moves report the card.
	const selected = []
	board.addEventListener('sb-kanban-select', (e) => selected.push(e.detail.cardId))
	card('mars').focus()
	press(card('mars'), 'ArrowDown')
	press(document.activeElement, 'ArrowRight')
	check('focus moves emit sb-kanban-select', [selected, document.activeElement?.dataset?.kanbanCard], [['jupiter', 'titan'], 'titan'])

	// Within a lane: Jupiter up, before Mars.
	card('jupiter').focus()
	press(card('jupiter'), 'ArrowUp', { altKey: true })
	check('a staged move', board.hasAttribute('data-key-staging'), true)
	release('Alt')
	let want = 'jupiter mars neptune | europa titan | moon'
	await until(() => board.dataset.state === want)
	check('a move within a lane', [board.dataset.state, lanes()], [want, want])
	check('the moved card keeps the focus', document.activeElement?.dataset?.kanbanCard, 'jupiter')

	// Across lanes: Titan to the end of Visited.
	card('titan').focus()
	press(card('titan'), 'ArrowRight', { altKey: true })
	release('Alt')
	want = 'jupiter mars neptune | europa | moon titan'
	await until(() => board.dataset.state === want)
	check('a move across lanes', [board.dataset.state, lanes()], [want, want])

	// Escape cancels a staged move: nothing is sent.
	card('europa').focus()
	press(card('europa'), 'ArrowLeft', { altKey: true })
	press(card('europa'), 'Escape')
	release('Alt')
	await settle(400)
	check('a cancelled move', [board.dataset.state, board.hasAttribute('data-key-staging')], [want, false])
	check('no errors', errors, [])
} catch (e) {
	rows.push({ step: 'script', error: String(e?.stack || e) })
}
await report()
`
