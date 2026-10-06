package app_test

import "testing"

// TestKanbanBoardMove drives sb-kanban-board's demo with the keyboard and the
// pointer: focus moves and selection, moves within and across lanes, into an
// empty lane, the keys at the board's edges, and the board the server sends
// back after each move.
func TestKanbanBoardMove(t *testing.T) {
	_, body := probe(t, "/components/kanban-board", pdPrelude+kanbanBoardJS)
	copyRows(t, body)
}

const kanbanBoardJS = `
try {
	await customElements.whenDefined('sb-kanban-board')
	const board = document.getElementById('mission-board')
	board.scrollIntoView({ block: 'center' })
	const card = (id) => board.querySelector('[data-kanban-card="' + id + '"]')
	const lane = (col) => board.querySelector('[data-kanban-lane][data-col="' + col + '"]')
	const lanes = () => [...board.querySelectorAll('[data-kanban-lane]')].map((l) => [...l.querySelectorAll('[data-kanban-card]')].map((c) => c.dataset.kanbanCard).join(' ')).join(' | ').replace(/  +/g, ' ').trim()
	const focused = () => document.activeElement?.dataset?.kanbanCard ?? (document.activeElement?.matches('[data-kanban-lane]') ? 'lane ' + document.activeElement.dataset.col : document.activeElement?.tagName)
	const moves = []
	const selected = []
	board.addEventListener('sb-kanban-move', (e) => moves.push(e.detail))
	board.addEventListener('sb-kanban-select', (e) => selected.push(e.detail.cardId))
	const animated = []
	const animate = Element.prototype.animate
	Element.prototype.animate = function (...args) {
		if (this.dataset?.kanbanCard) animated.push(this.dataset.kanbanCard)
		return animate.apply(this, args)
	}
	let state = 'mars jupiter neptune | europa titan | moon'
	check('the server rendered the board', [board.dataset.state, lanes()], [state, state])
	const answered = async (step, want) => {
		await until(() => board.dataset.state === want)
		check(step, [board.dataset.state, lanes()], [want, want])
		state = want
	}
	// Alt with a key on a card, then Alt released: the staged move commits.
	const keyMove = async (id, keys, want, step) => {
		card(id).focus()
		for (const key of keys) press(card(id), key, { altKey: true })
		release('Alt')
		await answered(step, want)
	}

	// Focus moves report the card; so does a press of the pointer.
	card('mars').focus()
	press(card('mars'), 'ArrowDown')
	press(document.activeElement, 'ArrowRight')
	check('arrow focus emits sb-kanban-select', [selected, focused()], [['jupiter', 'titan'], 'titan'])
	pointer(card('neptune'), 'pointerdown', center(card('neptune')))
	pointer(card('neptune'), 'pointerup', center(card('neptune')))
	check('a press of the pointer emits sb-kanban-select', selected.at(-1), 'neptune')

	// Alt+Left in the first lane and Alt+Right in the last stage nothing, and
	// keep the browser from going back or forward.
	card('mars').focus()
	const left = press(card('mars'), 'ArrowLeft', { altKey: true })
	card('moon').focus()
	const right = press(card('moon'), 'ArrowRight', { altKey: true })
	release('Alt')
	check('Alt+arrows at the edges are claimed', [left, right, board.hasAttribute('data-key-staging')], [false, false, false])

	// Within a lane: Jupiter up, before Mars; it keeps the focus and animates.
	card('jupiter').focus()
	press(card('jupiter'), 'ArrowUp', { altKey: true })
	check('a staged move', board.hasAttribute('data-key-staging'), true)
	release('Alt')
	await answered('a move within a lane', 'jupiter mars neptune | europa titan | moon')
	check('the moved card keeps the focus', focused(), 'jupiter')
	await settle(50)
	check('the move animates', animated.includes('jupiter'), true)

	// Across lanes a card keeps its row: Mars (second) goes before Titan (second).
	card('mars').focus()
	press(card('mars'), 'ArrowRight', { altKey: true })
	check('the staged target keeps the row', card('titan').hasAttribute('data-drop-before'), true)
	release('Alt')
	await answered('a move across lanes keeps the row', 'jupiter neptune | europa mars titan | moon')

	// Alt+Down adjusts a staged move across lanes before Alt is released.
	await keyMove('europa', ['ArrowRight', 'ArrowDown'], 'jupiter neptune | mars titan | moon europa', 'a staged move adjusted')

	// Alt+Home and Alt+End: to the top and the bottom of the lane.
	await keyMove('europa', ['Home'], 'jupiter neptune | mars titan | europa moon', 'Alt+Home')
	await keyMove('jupiter', ['End'], 'neptune jupiter | mars titan | europa moon', 'Alt+End')

	// Escape cancels a staged move: nothing is sent.
	const sent = moves.length
	card('europa').focus()
	press(card('europa'), 'ArrowLeft', { altKey: true })
	press(card('europa'), 'Escape')
	release('Alt')
	await settle(400)
	check('a cancelled move', [board.dataset.state, board.hasAttribute('data-key-staging'), moves.length], [state, false, sent])

	// A drag within a lane: Titan before Mars. The card keeps its element, so
	// it keeps the focus.
	const titan = card('titan')
	titan.focus()
	await drag(titan, { x: center(card('mars')).x, y: card('mars').getBoundingClientRect().top + 3 })
	check('a drag sends the move', moves.at(-1), { cardId: 'titan', col: 7, before: 'mars' })
	await answered('a drag within a lane', 'neptune jupiter | titan mars | europa moon')
	check('the dragged card keeps the focus', [document.activeElement === titan, titan.isConnected], [true, true])

	// Drags to the end of another lane, until En route is empty.
	const end = (col) => {
		const r = lane(col).getBoundingClientRect()
		return { x: r.left + r.width / 2, y: r.bottom - 4 }
	}
	await drag(card('titan'), end(9))
	await answered('a drag to the end of another lane', 'neptune jupiter | mars | europa moon titan')
	await drag(card('mars'), end(9))
	await answered('a drag that empties a lane', 'neptune jupiter | | europa moon titan mars')

	// The empty lane (tabindex="-1") is a focus stop; focusing it selects no card.
	const picked = selected.length
	card('jupiter').focus()
	press(card('jupiter'), 'ArrowRight')
	check('Right reaches the empty lane', focused(), 'lane 7')
	press(lane(7), 'ArrowRight')
	check('Right goes on from the empty lane', focused(), 'europa')
	press(card('europa'), 'ArrowLeft')
	press(lane(7), 'ArrowLeft')
	check('Left crosses it the other way', focused(), 'neptune')
	press(card('jupiter'), 'ArrowDown')
	check('Down stops at the empty lane in board order', focused(), 'lane 7')
	press(lane(7), 'Home')
	check('Home from the lane', focused(), 'neptune')
	check('focusing a lane selects no card', selected.slice(picked), ['europa', 'neptune', 'neptune'])
	check('Alt+arrows on a lane are claimed', [press(lane(7), 'ArrowLeft', { altKey: true }), board.hasAttribute('data-key-staging')], [false, false])
	release('Alt')

	// Without a tabindex, the arrows pass over an empty lane.
	lane(7).removeAttribute('tabindex')
	card('jupiter').focus()
	press(card('jupiter'), 'ArrowRight')
	check('Right passes over an empty lane without tabindex', focused(), 'moon')
	lane(7).setAttribute('tabindex', '-1')

	// Into the empty lane with the keyboard, and back out keeping the row.
	await keyMove('europa', ['ArrowLeft'], 'neptune jupiter | europa | moon titan mars', 'a keyboard move into an empty lane')
	await keyMove('europa', ['ArrowRight'], 'neptune jupiter | | europa moon titan mars', 'and back to its row')

	// Into the empty lane with the pointer: the lane and its end are marked.
	const from = center(card('mars')), to = center(lane(7))
	pointer(card('mars'), 'pointerdown', from)
	for (let k = 1; k <= 8; k++) {
		const p = { x: from.x + ((to.x - from.x) * k) / 8, y: from.y + ((to.y - from.y) * k) / 8 }
		pointer(document.elementFromPoint(p.x, p.y) ?? document.body, 'pointermove', p)
		await new Promise(requestAnimationFrame)
	}
	check('an empty lane is a drop target', [lane(7).hasAttribute('data-drop-active'), lane(7).querySelector('[data-kanban-lane-cards]').hasAttribute('data-drop-end')], [true, true])
	pointer(document.elementFromPoint(to.x, to.y), 'pointerup', to)
	check('a drop into an empty lane', moves.at(-1), { cardId: 'mars', col: 7, before: '' })
	await answered('a drag into an empty lane', 'neptune jupiter | mars | europa moon titan')
	check('no errors', errors, [])
} catch (e) {
	rows.push({ step: 'script', error: String(e?.stack || e) })
}
await report()
`
