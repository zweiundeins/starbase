package app_test

import "testing"

// TestSortableTreeMove moves rows of sb-sortable-tree's demo with the
// keyboard, among siblings and between folders, and checks the tree the
// server sends back.
func TestSortableTreeMove(t *testing.T) {
	_, body := probe(t, "/components/sortable-tree", pdPrelude+sortableTreePrelude+sortableTreeJS)
	copyRows(t, body)
}

// TestSortableTreeDrag drags rows of the demo with the pointer: into a
// folder, before a row, into a closed folder, a folder onto its own child,
// and a move the server refuses.
func TestSortableTreeDrag(t *testing.T) {
	_, body := probe(t, "/components/sortable-tree", pdPrelude+sortableTreePrelude+sortableTreeDragJS)
	copyRows(t, body)
}

// TestSortableTreeClick opens and closes folders with a click on their row
// (patches/pd-rockets/0041).
func TestSortableTreeClick(t *testing.T) {
	_, body := probe(t, "/components/sortable-tree", pdPrelude+sortableTreePrelude+sortableTreeClickJS)
	copyRows(t, body)
}

const sortableTreePrelude = `
await customElements.whenDefined('sb-sortable-tree')
const tree = document.getElementById('solar-moons')
const row = (id) => tree.querySelector('[data-tree-node="' + id + '"] > [data-tree-row]')
const list = (id) => tree.querySelector('[data-tree-node="' + id + '"] > [data-tree-children]')
// The tree as the DOM holds it, in data-state's words.
const shape = (l) => [...l.children].filter((n) => n.matches('[data-tree-node]')).map((n) => {
	const kids = n.querySelector(':scope > [data-tree-children]')
	return n.dataset.treeNode + (kids ? '(' + shape(kids) + ')' : '')
}).join(' ')
const dom = () => shape(tree.querySelector(':scope > [data-tree-children]'))
const moves = []
tree.addEventListener('sb-tree-move', (e) => moves.push(e.detail))
`

const sortableTreeJS = `
try {
	// Stage the keys on the focused row with Alt held, then release Alt; the server answers.
	const move = async (id, ...keys) => {
		const before = tree.dataset.state
		row(id).focus()
		for (const k of keys) press(row(id), k, { altKey: true })
		release('Alt')
		await until(() => tree.dataset.state !== before)
	}
	check('the server rendered the tree', [tree.dataset.state, dom()], ['earth(moon phobos) mars(deimos) jupiter(europa io) callisto', 'earth(moon phobos) mars(deimos) jupiter(europa io) callisto'])

	// Focus moves without Alt.
	row('earth').focus()
	press(row('earth'), 'ArrowRight')
	check('Arrow Right on an open folder focuses its first row', document.activeElement === row('moon'), true)
	press(row('moon'), 'End')
	check('End focuses the last row', document.activeElement === row('callisto'), true)
	press(row('callisto'), 'Home')
	check('Home focuses the first row', document.activeElement === row('earth'), true)

	await move('io', 'ArrowUp')
	check('among siblings', [tree.dataset.state, dom()], ['earth(moon phobos) mars(deimos) jupiter(io europa) callisto', 'earth(moon phobos) mars(deimos) jupiter(io europa) callisto'])
	check('the moved row keeps the focus', document.activeElement === row('io'), true)

	row('callisto').focus()
	press(row('callisto'), 'ArrowRight', { altKey: true })
	check('staged into a folder', row('jupiter').hasAttribute('data-tree-into'), true)
	release('Alt')
	await until(() => tree.dataset.state.endsWith('callisto)'))
	check('into the folder above', dom(), 'earth(moon phobos) mars(deimos) jupiter(io europa callisto)')

	// Out of Earth (before Mars), one further down (before Jupiter), then into Mars.
	await move('phobos', 'ArrowDown', 'ArrowDown', 'ArrowRight')
	check('out of one folder and into another', dom(), 'earth(moon) mars(deimos phobos) jupiter(io europa callisto)')
	check('still focused', document.activeElement === row('phobos'), true)

	// A closed folder stays closed when the server's answer morphs the tree.
	row('earth').focus()
	press(row('earth'), 'ArrowLeft')
	check('closed', [list('earth').hidden, row('earth').getAttribute('aria-expanded')], [true, 'false'])
	await move('deimos', 'ArrowDown')
	check('moved', dom(), 'earth(moon) mars(phobos deimos) jupiter(io europa callisto)')
	await settle(100)
	check('still closed after the morph', [list('earth').hidden, row('earth').getAttribute('aria-expanded')], [true, 'false'])
	row('earth').focus()
	press(row('earth'), 'ArrowRight')
	check('Arrow Right opens a closed folder', [list('earth').hidden, document.activeElement === row('earth')], [false, true])

	// Escape cancels a staged move: nothing is sent.
	row('europa').focus()
	press(row('europa'), 'ArrowUp', { altKey: true })
	press(row('europa'), 'Escape')
	release('Alt')
	await settle(400)
	check('a cancelled move', tree.dataset.state, 'earth(moon) mars(phobos deimos) jupiter(io europa callisto)')
	check('no errors', errors, [])
} catch (e) {
	rows.push({ step: 'script', error: String(e?.stack || e) })
}
await report()
`

const sortableTreeDragJS = `
try {
	// The drag helper hits the rows where the viewport shows them.
	tree.scrollIntoView({ block: 'center' })
	await settle()
	const point = (el, ratio) => {
		const r = el.getBoundingClientRect()
		return { x: r.left + r.width / 2, y: r.top + r.height * ratio }
	}
	const dropped = async (from, to) => {
		const before = tree.dataset.state
		moves.length = 0
		await drag(row(from), to)
		await until(() => tree.dataset.state !== before)
		// Rows measured while the move animates are off.
		await until(() => tree.getAnimations({ subtree: true }).length === 0, 2000)
		return moves.slice()
	}

	let sent = await dropped('phobos', row('mars'))
	check('onto the middle of a folder: the event', sent, [{ itemId: 'phobos', fromParent: 'earth', toParent: 'mars', before: '' }])
	check('onto the middle of a folder: the answer', [tree.dataset.state, dom()], ['earth(moon) mars(deimos phobos) jupiter(europa io) callisto', 'earth(moon) mars(deimos phobos) jupiter(europa io) callisto'])

	sent = await dropped('callisto', point(row('europa'), 0.15))
	check('onto the upper part of a row: the event', sent, [{ itemId: 'callisto', fromParent: '', toParent: 'jupiter', before: 'europa' }])
	check('onto the upper part of a row: the answer', dom(), 'earth(moon) mars(deimos phobos) jupiter(callisto europa io)')

	sent = await dropped('moon', point(row('io'), 0.85))
	check('onto the lower part of the last row', sent, [{ itemId: 'moon', fromParent: 'earth', toParent: 'jupiter', before: '' }])
	check('after it', dom(), 'earth() mars(deimos phobos) jupiter(callisto europa io moon)')

	// A folder never goes into itself: the drop sends nothing.
	moves.length = 0
	await drag(row('jupiter'), row('europa'))
	await settle(300)
	check('a folder onto its own child', [moves, dom()], [[], 'earth() mars(deimos phobos) jupiter(callisto europa io moon)'])

	// A move into a closed folder opens it once the server has answered.
	row('mars').focus()
	press(row('mars'), 'ArrowLeft')
	check('Mars is closed', list('mars').hidden, true)
	sent = await dropped('io', row('mars'))
	check('into a closed folder', [sent, dom()], [[{ itemId: 'io', fromParent: 'jupiter', toParent: 'mars', before: '' }], 'earth() mars(deimos phobos io) jupiter(callisto europa moon)'])
	await until(() => !list('mars').hidden, 2000)
	check('it opens', [list('mars').hidden, row('mars').getAttribute('aria-expanded')], [false, 'true'])

	// The server refuses a move its state doesn't allow (422): nothing changes, and the tree still works.
	const state = tree.dataset.state
	tree.dataset.state = 'earth() mars(deimos phobos io moon) jupiter(callisto europa)'
	let refused = false
	document.addEventListener('datastar-fetch', (e) => { if (e.detail.el === tree && e.detail.type === 'error') refused = true })
	moves.length = 0
	await drag(row('moon'), row('earth'))
	await until(() => refused, 5000)
	await settle(200)
	check('a refused move', [refused, moves.length, dom()], [true, 1, 'earth() mars(deimos phobos io) jupiter(callisto europa moon)'])
	tree.dataset.state = state
	sent = await dropped('moon', row('earth'))
	check('a move after the refusal', dom(), 'earth(moon) mars(deimos phobos io) jupiter(callisto europa)')
	check('no errors', errors, [])
} catch (e) {
	rows.push({ step: 'script', error: String(e?.stack || e) })
}
await report()
`

const sortableTreeClickJS = `
try {
	row('earth').click()
	check('a click closes a folder', [list('earth').hidden, row('earth').getAttribute('aria-expanded')], [true, 'false'])
	row('earth').click()
	check('a second opens it', [list('earth').hidden, row('earth').getAttribute('aria-expanded')], [false, 'true'])
	row('moon').click()
	check('a click on a file changes nothing', [list('earth').hidden, row('moon').hasAttribute('aria-expanded')], [false, false])

	const button = document.createElement('button')
	button.textContent = 'Rename'
	row('mars').append(button)
	button.click()
	check('a click on a button in the row is the button\'s', list('mars').hidden, false)
	button.remove()

	row('jupiter').addEventListener('click', (e) => e.preventDefault(), { once: true })
	row('jupiter').click()
	check('a cancelled click changes nothing', list('jupiter').hidden, false)

	// A closed folder stays closed through the server's answer.
	row('jupiter').click()
	moves.length = 0
	const before = tree.dataset.state
	row('callisto').focus()
	press(row('callisto'), 'ArrowUp', { altKey: true })
	release('Alt')
	await until(() => tree.dataset.state !== before)
	await settle(100)
	check('moved', dom(), 'earth(moon phobos) mars(deimos) callisto jupiter(europa io)')
	check('still closed', [list('jupiter').hidden, row('jupiter').getAttribute('aria-expanded')], [true, 'false'])
	check('no errors', errors, [])
} catch (e) {
	rows.push({ step: 'script', error: String(e?.stack || e) })
}
await report()
`
