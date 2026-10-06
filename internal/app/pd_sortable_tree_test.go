package app_test

import "testing"

// TestSortableTreeMove moves rows of sb-sortable-tree's demo with the
// keyboard, among siblings and between folders, and checks the tree the
// server sends back.
func TestSortableTreeMove(t *testing.T) {
	_, body := probe(t, "/components/sortable-tree", pdPrelude+sortableTreeJS)
	copyRows(t, body)
}

const sortableTreeJS = `
try {
	await customElements.whenDefined('sb-sortable-tree')
	const tree = document.getElementById('solar-moons')
	const row = (id) => tree.querySelector('[data-tree-node="' + id + '"] > [data-tree-row]')
	// The tree as the DOM holds it, in data-state's words.
	const shape = (list) => [...list.children].filter((n) => n.matches('[data-tree-node]')).map((n) => {
		const kids = n.querySelector(':scope > [data-tree-children]')
		return n.dataset.treeNode + (kids ? '(' + shape(kids) + ')' : '')
	}).join(' ')
	const dom = () => shape(tree.querySelector(':scope > [data-tree-children]'))
	// Stage the keys on the focused row with Alt held, then release Alt; the server answers.
	const move = async (id, ...keys) => {
		const before = tree.dataset.state
		row(id).focus()
		for (const k of keys) press(row(id), k, { altKey: true })
		release('Alt')
		await until(() => tree.dataset.state !== before)
	}
	check('the server rendered the tree', [tree.dataset.state, dom()], ['earth(moon phobos) mars(deimos) jupiter(europa io) callisto', 'earth(moon phobos) mars(deimos) jupiter(europa io) callisto'])

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
	const earthList = tree.querySelector('[data-tree-node="earth"] > [data-tree-children]')
	check('closed', [earthList.hidden, row('earth').getAttribute('aria-expanded')], [true, 'false'])
	await move('deimos', 'ArrowDown')
	check('moved', dom(), 'earth(moon) mars(phobos deimos) jupiter(io europa callisto)')
	await settle(100)
	check('still closed after the morph', [tree.querySelector('[data-tree-node="earth"] > [data-tree-children]').hidden, row('earth').getAttribute('aria-expanded')], [true, 'false'])

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
