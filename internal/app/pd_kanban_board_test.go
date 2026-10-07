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

// TestKanbanBoardLanes moves the demo's lanes: a drag by a grip, Alt and the
// arrows on a grip (committed, and cancelled with Escape), a step button, a
// card drag next to the grips, and the server's refusal of stale lane ids.
func TestKanbanBoardLanes(t *testing.T) {
	_, body := probe(t, "/components/kanban-board", pdPrelude+kanbanLanesJS)
	copyRows(t, body)
}

const kanbanLanesJS = `
try {
	await customElements.whenDefined('sb-kanban-board')
	const board = document.getElementById('mission-board')
	board.scrollIntoView({ block: 'center' })
	const lane = (col) => board.querySelector('[data-kanban-lane][data-col="' + col + '"]')
	const grip = (col) => lane(col).querySelector('[data-kanban-lane-grip]')
	const card = (id) => board.querySelector('[data-kanban-card="' + id + '"]')
	const order = () => [...board.querySelectorAll('[data-kanban-lane]')].map((l) => l.dataset.col).join(' ')
	const marks = () => [...board.querySelectorAll('[data-lane-dragging], [data-lane-drop-target]')].map((l) => l.dataset.col + (l.hasAttribute('data-lane-dragging') ? ' dragging' : ' target'))
	const laneMoves = []
	const cardMoves = []
	board.addEventListener('sb-kanban-lane-move', (e) => laneMoves.push(e.detail))
	board.addEventListener('sb-kanban-move', (e) => cardMoves.push(e.detail))
	const animated = []
	const animate = Element.prototype.animate
	Element.prototype.animate = function (...args) {
		if (this.matches?.('[data-kanban-lane]')) animated.push(this.dataset.col)
		return animate.apply(this, args)
	}
	let state = '4: mars jupiter neptune | 7: europa titan | 9: moon'
	check('the server rendered the board', [board.dataset.state, order()], [state, '4 7 9'])
	const answered = async (step, want, cols) => {
		await until(() => board.dataset.state === want)
		check(step, [board.dataset.state, order()], [want, cols])
		state = want
	}
	// Alt with keys on a lane's grip, then Alt released.
	const keyMove = async (col, keys, want, cols, step) => {
		grip(col).focus()
		for (const key of keys) press(grip(col), key, { altKey: true })
		release('Alt')
		await answered(step, want, cols)
	}

	// A drag by the grip marks the lane and the one whose place it takes, and
	// Escape ends it without a move.
	let from = center(grip(4)), to = center(lane(9))
	pointer(grip(4), 'pointerdown', from)
	for (let k = 1; k <= 8; k++) {
		const p = { x: from.x + ((to.x - from.x) * k) / 8, y: from.y + ((to.y - from.y) * k) / 8 }
		pointer(document.elementFromPoint(p.x, p.y) ?? document.body, 'pointermove', p)
		await new Promise(requestAnimationFrame)
	}
	const preview = document.querySelector('[data-drag-preview][data-kanban-lane]')
	check('marks while a lane is dragged', [marks(), preview?.dataset.col, preview?.querySelector('[id]') ?? null], [['4 dragging', '9 target'], '4', null])
	press(grip(4), 'Escape')
	pointer(document.elementFromPoint(to.x, to.y), 'pointerup', to)
	await settle(300)
	check('Escape ends a lane drag', [laneMoves.length, marks(), !!document.querySelector('[data-drag-preview]'), board.dataset.state], [0, [], false, state])

	// A drag by the grip onto the last lane: the lane goes to the end, keeps
	// its element, and animates into its place.
	const first = lane(4)
	await drag(grip(4), center(lane(9)))
	check('a lane drag sends the move', laneMoves.at(-1), { col: 4, before: '' })
	await answered('a lane drag', '7: europa titan | 9: moon | 4: mars jupiter neptune', '7 9 4')
	await settle(50)
	check('the lane keeps its element and animates', [lane(4) === first, animated.includes('4')], [true, true])

	// Alt and the arrows on a grip: staged, then sent when Alt is released.
	grip(4).focus()
	press(grip(4), 'ArrowLeft', { altKey: true })
	check('a staged lane move', [marks(), board.hasAttribute('data-key-staging')], [['9 target', '4 dragging'], true])
	release('Alt')
	await answered('Alt+Left on a grip', '7: europa titan | 4: mars jupiter neptune | 9: moon', '7 4 9')
	check('the grip keeps the focus', document.activeElement === grip(4), true)
	await keyMove(9, ['ArrowLeft', 'ArrowLeft'], '9: moon | 7: europa titan | 4: mars jupiter neptune', '9 7 4', 'two places left')
	await keyMove(4, ['Home'], '4: mars jupiter neptune | 9: moon | 7: europa titan', '4 9 7', 'Alt+Home on a grip')
	await keyMove(9, ['End'], '4: mars jupiter neptune | 7: europa titan | 9: moon', '4 7 9', 'Alt+End on a grip')

	// Escape cancels a staged lane move; the edges claim their keys.
	const sent = laneMoves.length
	grip(7).focus()
	press(grip(7), 'ArrowRight', { altKey: true })
	press(grip(7), 'Escape')
	release('Alt')
	const left = press(grip(4), 'ArrowLeft', { altKey: true })
	const right = press(grip(9), 'ArrowRight', { altKey: true })
	release('Alt')
	await settle(400)
	check('a cancelled lane move', [laneMoves.length, marks(), board.hasAttribute('data-key-staging'), board.dataset.state], [sent, [], false, state])
	check('Alt+arrows at the edges are claimed', [left, right], [false, false])

	// A step button moves its lane one place.
	lane(7).querySelector('.demo-kanban__head').insertAdjacentHTML('beforeend', '<button type="button" data-kanban-lane-step="-1">◀</button>')
	lane(7).querySelector('[data-kanban-lane-step]').click()
	check('a step button sends the move', laneMoves.at(-1), { col: 7, before: '4' })
	await answered('a step button', '7: europa titan | 4: mars jupiter neptune | 9: moon', '7 4 9')

	// Cards still drag next to the grips (once the lanes' animation is over).
	await settle(300)
	const end = lane(4).getBoundingClientRect()
	await drag(card('moon'), { x: end.left + end.width / 2, y: end.bottom - 4 })
	check('a card drag next to grips', cardMoves, [{ cardId: 'moon', col: 4, before: '' }])
	await answered('a card drag', '7: europa titan | 4: mars jupiter neptune moon | 9:', '7 4 9')

	// The server refuses lane ids its board doesn't have: the page is stale.
	const ask = (move) => fetch('/demo/arrange/kanban-board?datastar=' + encodeURIComponent(JSON.stringify({ id: board.id, state, move }))).then((r) => r.status)
	check('stale ids are refused', [await ask({ col: 4, before: '5' }), await ask({ col: 3, before: '' }), await ask({ col: 4, before: '9' })], [422, 422, 200])
	board.dispatchEvent(new CustomEvent('sb-kanban-lane-move', { bubbles: true, detail: { col: 4, before: '5' } }))
	await settle(600)
	check('a refused move changes nothing', [board.dataset.state, order()], [state, '7 4 9'])
	check('no errors', errors, [])
} catch (e) {
	rows.push({ step: 'script', error: String(e?.stack || e) })
}
await report()
`

// TestKanbanBoardGalleryCard drags a card in the gallery's live preview: the
// server answers it, and the card is one Tab stop that nothing overflows.
func TestKanbanBoardGalleryCard(t *testing.T) {
	_, body := probe(t, "/", pdPrelude+kanbanCardJS)
	copyRows(t, body)
}

const kanbanCardJS = `
try {
	await customElements.whenDefined('sb-kanban-board')
	const board = () => document.getElementById('kanban-board-card')
	board().scrollIntoView({ block: 'center' })
	const preview = board().closest('.card__preview')
	check('one Tab stop', [...preview.querySelectorAll('*')].filter((e) => e.tabIndex >= 0).map((e) => e.dataset.kanbanCard), ['mars'])
	const over = [board(), ...board().querySelectorAll('[data-kanban-lane], [data-kanban-card]')].filter((e) => e.scrollWidth > e.clientWidth + 1)
	check('nothing overflows', over.map((e) => e.dataset.kanbanCard ?? e.dataset.col ?? 'board'), [])
	const moves = []
	board().addEventListener('sb-kanban-move', (e) => moves.push(e.detail))
	await drag(board().querySelector('[data-kanban-card="mars"]'), board().querySelector('[data-col="9"]'))
	check('a drop into the empty lane', moves, [{ cardId: 'mars', col: 9, before: '' }])
	await until(() => board().dataset.state === '4: io | 7: moon | 9: mars')
	check('the server moved it', [board().dataset.state, board().querySelector('[data-col="9"] [data-kanban-card]')?.id], ['4: io | 7: moon | 9: mars', 'kanban-board-card-mars'])
	check('no errors', errors, [])
} catch (e) {
	rows.push({ step: 'script', error: String(e?.stack || e) })
}
await report()
`

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
	// The steps below name each lane's cards; data-state adds the lane ids.
	const withIds = (cards) => cards.split('|').map((p, i) => [4, 7, 9][i] + ':' + (p.trim() ? ' ' + p.trim() : '')).join(' | ')
	let state = 'mars jupiter neptune | europa titan | moon'
	check('the server rendered the board', [board.dataset.state, lanes()], [withIds(state), state])
	const answered = async (step, want) => {
		await until(() => board.dataset.state === withIds(want))
		check(step, [board.dataset.state, lanes()], [withIds(want), want])
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
	check('a cancelled move', [board.dataset.state, board.hasAttribute('data-key-staging'), moves.length], [withIds(state), false, sent])

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
