package app_test

import "testing"

// TestBentoWorkspaceMoves moves the focus, and moves, resizes and carries
// tiles of sb-bento-workspace's demo with the keyboard, and checks the layout
// the server sends back, the tiles that glide to it, and what a refused move
// leaves.
func TestBentoWorkspaceMoves(t *testing.T) {
	_, body := probe(t, "/components/bento-workspace", pdPrelude+bentoPrelude+bentoKeyboardJS)
	copyRows(t, body)
}

// TestBentoWorkspacePointer drags a tile and a resize handle of the demo and
// checks the events, the server's answer, the tiles that glide and the focus.
func TestBentoWorkspacePointer(t *testing.T) {
	_, body := probe(t, "/components/bento-workspace", pdPrelude+bentoPrelude+bentoPointerJS)
	copyRows(t, body)
}

const bentoPrelude = `
const deck = document.getElementById('mission-deck')
const tile = (id) => deck.querySelector('[data-bento-item="' + id + '"]')
const focused = () => document.activeElement?.dataset?.bentoItem
const shown = () => [...deck.querySelectorAll('[data-bento-grid]')].map((g) => g.dataset.bentoGrid + ' ' +
	[...g.querySelectorAll('[data-bento-item]')].map((t) => [t.dataset.bentoItem, t.dataset.bentoCol, t.dataset.bentoRow, t.dataset.bentoWidth, t.dataset.bentoHeight].join('.')).join(' ')).join(' ')
const start = 'deck thrust.1.1.2.2 fuel.3.1.2.1 speed.3.2.2.1 shelf shields.1.1.2.1 crew.1.2.1.1'
const answered = async (name, before, want) => {
	await until(() => deck.dataset.state !== before)
	check(name + ': the server applied it', [deck.dataset.state, shown()], [want, want])
}
// The tiles the FLIP animation moved since the last call, once it has ended. A
// move within a grid changes only the tiles' attributes (patch 0005).
const glides = []
const animate = Element.prototype.animate
Element.prototype.animate = function (...args) {
	if (this.dataset?.bentoItem) glides.push(this.dataset.bentoItem)
	return animate.apply(this, args)
}
const glided = async () => {
	await settle(300)
	return glides.splice(0)
}
`

const bentoKeyboardJS = `
try {
	await customElements.whenDefined('sb-bento-workspace')
	check('the server rendered the layout', [deck.dataset.state, shown()], [start, start])

	const focusKey = (from, key, init) => {
		tile(from).focus()
		press(tile(from), key, init)
		return focused()
	}
	check('ArrowDown focuses the tile below', focusKey('fuel', 'ArrowDown'), 'speed')
	check('k focuses the tile above', focusKey('speed', 'k'), 'fuel')
	check('ArrowLeft focuses the tile to the left', focusKey('fuel', 'ArrowLeft'), 'thrust')
	check('Home focuses the first tile', focusKey('crew', 'Home'), 'thrust')
	check('End focuses the last tile', focusKey('thrust', 'End'), 'crew')
	focusKey('speed', 'ArrowRight')
	check('ArrowRight at the edge crosses to the next grid', document.activeElement?.closest('[data-bento-grid]')?.dataset.bentoGrid, 'shelf')
	focusKey('shields', 'h')
	check('h at the edge crosses to the previous grid', document.activeElement?.closest('[data-bento-grid]')?.dataset.bentoGrid, 'deck')

	// While the key is held, the tiles in the way move to their new places: only
	// the tile moved glides there, and a resize moves none.
	const step = async (name, id, key, init, release_, want, glide) => {
		const before = deck.dataset.state
		tile(id).focus()
		press(tile(id), key, init)
		check(name + ': staged', deck.hasAttribute('data-key-staging'), true)
		// Release once the tiles in the way have reached their new places, as a person would.
		await settle(50)
		await until(() => deck.getAnimations({ subtree: true }).length === 0, 5000)
		release(release_)
		await answered(name, before, want)
		check(name + ': the tile keeps the focus', focused(), id)
		check(name + ': the tiles that glide', await glided(), glide)
	}
	await step('a move down', 'fuel', 'ArrowDown', { altKey: true }, 'Alt',
		'deck thrust.1.1.2.2 fuel.3.2.2.1 speed.3.3.2.1 shelf shields.1.1.2.1 crew.1.2.1.1', ['fuel'])
	await step('a resize', 'shields', 'ArrowDown', { shiftKey: true }, 'Shift',
		'deck thrust.1.1.2.2 fuel.3.2.2.1 speed.3.3.2.1 shelf shields.1.1.2.2 crew.1.3.1.1', [])
	await step('to the next grid', 'speed', 'PageDown', { altKey: true }, 'Alt',
		'deck thrust.1.1.2.2 fuel.3.2.2.1 shelf speed.1.1.2.1 shields.1.2.2.2 crew.1.4.1.1', ['speed'])
	await step('past the right edge', 'fuel', 'ArrowRight', { altKey: true }, 'Alt',
		'deck thrust.1.1.2.2 shelf speed.1.1.2.1 fuel.1.2.2.1 shields.1.3.2.2 crew.1.5.1.1', ['fuel'])
	await step('past the left edge', 'fuel', 'ArrowLeft', { altKey: true }, 'Alt',
		'deck thrust.1.1.2.2 fuel.3.2.2.1 shelf speed.1.1.2.1 shields.1.3.2.2 crew.1.5.1.1', ['fuel'])

	// Escape cancels a staged move: nothing is sent.
	let before = deck.dataset.state
	tile('thrust').focus()
	press(tile('thrust'), 'ArrowRight', { altKey: true })
	press(tile('thrust'), 'Escape')
	check('a cancelled move leaves no staging', deck.hasAttribute('data-key-staging'), false)
	release('Alt')
	await settle(400)
	check('a cancelled move', deck.dataset.state, before)

	// The demo's server takes 30 rows: a move to row 31 is refused, and the
	// shown layout goes after 2 seconds.
	before = deck.dataset.state
	tile('crew').focus()
	for (let k = 0; k < 26; k++) press(tile('crew'), 'ArrowDown', { altKey: true })
	check('row 31 is staged', deck.querySelector('[data-bento-target]')?.style.gridRow, '31 / span 1')
	release('Alt')
	await settle(1000)
	check('the refused move is still shown', [deck.dataset.state, !!deck.querySelector('[data-bento-projecting]')], [before, true])
	await settle(1300)
	const projection = () => ({
		target: !!deck.querySelector('[data-bento-target]'),
		projecting: !!deck.querySelector('[data-bento-projecting]'),
		transforms: [...deck.querySelectorAll('[data-bento-item]')].filter((t) => t.style.transform).length,
		stretched: [...deck.querySelectorAll('[data-bento-grid]')].filter((g) => g.style.minHeight).length,
	})
	check('a refusal leaves no projection', [deck.dataset.state, shown(), projection()], [before, before, { target: false, projecting: false, transforms: 0, stretched: 0 }])
	check('nothing glides for a cancelled or refused move', await glided(), [])
	check('no errors', errors, [])
} catch (e) {
	rows.push({ step: 'script', error: String(e?.stack || e) })
}
await report()
`

const bentoPointerJS = `
try {
	await customElements.whenDefined('sb-bento-workspace')
	deck.scrollIntoView({ block: 'center' })
	await settle()
	const events = []
	for (const name of ['sb-bento-move', 'sb-bento-resize']) deck.addEventListener(name, (e) => events.push([name, e.detail]))

	// A drop on fuel's cell, off its middle: speed takes it and glides there
	// from the pointer, fuel moves down. The morph reuses tiles by position,
	// and the focus follows speed all the same.
	let before = deck.dataset.state
	tile('speed').focus()
	const cell = center(tile('fuel'))
	await drag(tile('speed'), { x: cell.x + 20, y: cell.y + 10 })
	check('the drop emits the move', events.shift(), ['sb-bento-move', { itemId: 'speed', fromGrid: 'deck', toGrid: 'deck', updates: [
		{ itemId: 'speed', grid: 'deck', col: 3, row: 1, width: 2, height: 1 },
		{ itemId: 'fuel', grid: 'deck', col: 3, row: 2, width: 2, height: 1 },
	] }])
	await answered('a pointer move', before, 'deck thrust.1.1.2.2 speed.3.1.2.1 fuel.3.2.2.1 shelf shields.1.1.2.1 crew.1.2.1.1')
	check('a pointer move: the dropped tile glides from the pointer', (await glided()).includes('speed'), true)
	await until(() => focused() === 'speed', 1000)
	check('a pointer move: the tile keeps the focus', focused(), 'speed')
	check('a pointer move: nothing is left shown', [!!deck.querySelector('[data-bento-target]'), deck.hasAttribute('data-drag-active'), !!document.querySelector('[data-drag-preview]')], [false, false, false])

	// The resize handle, dragged one cell to the right.
	before = deck.dataset.state
	const handle = tile('crew').querySelector('[data-bento-resize]')
	const grid = tile('crew').closest('[data-bento-grid]')
	const step = tile('shields').getBoundingClientRect().width / 2 + parseFloat(getComputedStyle(grid).columnGap) / 2
	const from = center(handle)
	await drag(handle, { x: from.x + step, y: from.y })
	check('the handle emits the resize', events.shift(), ['sb-bento-resize', { itemId: 'crew', grid: 'shelf', updates: [
		{ itemId: 'crew', grid: 'shelf', col: 1, row: 2, width: 2, height: 1 },
	] }])
	await answered('a pointer resize', before, 'deck thrust.1.1.2.2 speed.3.1.2.1 fuel.3.2.2.1 shelf shields.1.1.2.1 crew.1.2.2.1')
	check('a pointer resize: nothing glides', await glided(), [])
	check('no other events', events, [])
	check('no errors', errors, [])
} catch (e) {
	rows.push({ step: 'script', error: String(e?.stack || e) })
}
await report()
`
