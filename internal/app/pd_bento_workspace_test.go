package app_test

import "testing"

// TestBentoWorkspaceMoves moves, resizes and carries tiles of
// sb-bento-workspace's demo with the keyboard and checks the layout the server
// sends back.
func TestBentoWorkspaceMoves(t *testing.T) {
	_, body := probe(t, "/components/bento-workspace", pdPrelude+bentoWorkspaceJS)
	copyRows(t, body)
}

const bentoWorkspaceJS = `
try {
	await customElements.whenDefined('sb-bento-workspace')
	const deck = document.getElementById('mission-deck')
	const tile = (id) => deck.querySelector('[data-bento-item="' + id + '"]')
	const shown = () => [...deck.querySelectorAll('[data-bento-grid]')].map((g) => g.dataset.bentoGrid + ' ' +
		[...g.querySelectorAll('[data-bento-item]')].map((t) => [t.dataset.bentoItem, t.dataset.bentoCol, t.dataset.bentoRow, t.dataset.bentoWidth, t.dataset.bentoHeight].join('.')).join(' ')).join(' ')
	const step = async (name, id, key, init, release_, want) => {
		const before = deck.dataset.state
		tile(id).focus()
		press(tile(id), key, init)
		check(name + ': staged', deck.hasAttribute('data-key-staging'), true)
		release(release_)
		await until(() => deck.dataset.state !== before)
		check(name + ': the server applied it', [deck.dataset.state, shown()], [want, want])
		check(name + ': the tile keeps the focus', document.activeElement?.dataset?.bentoItem, id)
	}
	const start = 'deck thrust.1.1.2.2 fuel.3.1.2.1 speed.3.2.2.1 shelf shields.1.1.2.1 crew.1.2.1.1'
	check('the server rendered the layout', [deck.dataset.state, shown()], [start, start])

	await step('a move down', 'fuel', 'ArrowDown', { altKey: true }, 'Alt',
		'deck thrust.1.1.2.2 fuel.3.2.2.1 speed.3.3.2.1 shelf shields.1.1.2.1 crew.1.2.1.1')
	await step('a resize', 'shields', 'ArrowDown', { shiftKey: true }, 'Shift',
		'deck thrust.1.1.2.2 fuel.3.2.2.1 speed.3.3.2.1 shelf shields.1.1.2.2 crew.1.3.1.1')
	await step('to the next grid', 'speed', 'PageDown', { altKey: true }, 'Alt',
		'deck thrust.1.1.2.2 fuel.3.2.2.1 shelf speed.1.1.2.1 shields.1.2.2.2 crew.1.4.1.1')

	// Escape cancels a staged move: nothing is sent.
	const before = deck.dataset.state
	tile('thrust').focus()
	press(tile('thrust'), 'ArrowRight', { altKey: true })
	press(tile('thrust'), 'Escape')
	check('a cancelled move leaves no staging', deck.hasAttribute('data-key-staging'), false)
	release('Alt')
	await settle(400)
	check('a cancelled move', deck.dataset.state, before)
	check('no errors', errors, [])
} catch (e) {
	rows.push({ step: 'script', error: String(e?.stack || e) })
}
await report()
`
