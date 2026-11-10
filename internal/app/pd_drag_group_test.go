package app_test

import "testing"

// TestDragGroupMove moves items of sb-drag-group's demo with the keyboard and
// the pointer, within a list, to the other one and into an empty one, and
// checks the events and what the server sends back.
func TestDragGroupMove(t *testing.T) {
	_, body := probe(t, "/components/drag-group", pdPrelude+dragGroupJS)
	copyRows(t, body)
}

// TestDragGroupLists checks the arrow keys around empty lists, and nesting:
// a group in another group's item, and a host that a separate copy of the
// PD rockets core registered (as a surface bundled on its own does).
func TestDragGroupLists(t *testing.T) {
	_, body := probe(t, "/components/drag-group", pdPrelude+dragGroupListsJS)
	copyRows(t, body)
}

const dragGroupJS = `
try {
	await customElements.whenDefined('sb-drag-group')
	const group = document.getElementById('sort-bodies')
	group.scrollIntoView({ block: 'center' })
	const item = (id) => group.querySelector('[data-drag-item="' + id + '"]')
	const list = (name) => group.querySelector('[data-drop-list="' + name + '"]')
	const lists = () => [...group.querySelectorAll('[data-drop-list]')].map((l) => l.dataset.dropList + '=' + [...l.querySelectorAll('[data-drag-item]')].map((i) => i.dataset.dragItem).join(',')).join(' ')
	const moves = []
	group.addEventListener('sb-drag-group-move', (e) => moves.push(e.detail))
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

	// A move key with nowhere to go still keeps the key from the browser (Alt+Left goes back).
	item('earth').focus()
	check('Alt+Left in the leftmost list is the group\'s', press(item('earth'), 'ArrowLeft', { altKey: true }), false)
	check('Alt+Up on the first item is the group\'s', press(item('earth'), 'ArrowUp', { altKey: true }), false)
	check('neither stages a move', group.hasAttribute('data-key-staging'), false)
	release('Alt')

	// Across lists an item keeps its index, clamped to the other list's length
	// (patch 0023): Mars, second of the planets, lands above Ceres, and
	// Alt+Down takes it one further before Alt is released.
	item('mars').focus()
	press(item('mars'), 'ArrowRight', { altKey: true })
	check('Alt+Right keeps the index', group.querySelector('[data-drop-before]')?.dataset.dragItem, 'ceres')
	press(item('mars'), 'ArrowDown', { altKey: true })
	check('Alt+Down moves it on from there', group.querySelector('[data-drop-before]')?.dataset.dragItem, 'venus')
	release('Alt')
	await until(() => group.dataset.state !== across)
	const kept = 'planets=earth,jupiter dwarfs=pluto,ceres,mars,venus'
	check('a move to the same index in the other list', [group.dataset.state, lists()], [kept, kept])
	check('it keeps the focus there too', document.activeElement?.dataset?.dragItem, 'mars')

	// Third of the dwarfs, Mars goes back to the end of the two planets.
	press(item('mars'), 'ArrowLeft', { altKey: true })
	check('Alt+Left clamps the index to the shorter list', [list('planets').hasAttribute('data-drop-end'), group.querySelector('[data-drop-before]')], [true, null])
	release('Alt')
	await until(() => group.dataset.state !== kept)
	const clamped = 'planets=earth,jupiter,mars dwarfs=pluto,ceres,venus'
	check('a move clamped to the end', [group.dataset.state, lists()], [clamped, clamped])

	// The pointer: Earth to the end of the dwarfs. Its id lets the morph move it with the focus.
	moves.length = 0
	await settle(250)
	item('earth').focus()
	await drag(item('earth'), { x: center(item('venus')).x, y: item('venus').getBoundingClientRect().bottom - 2 })
	await until(() => group.dataset.state !== clamped)
	const dropped = 'planets=jupiter,mars dwarfs=pluto,ceres,venus,earth'
	check('a drop sends the move', moves, [{ itemId: 'earth', fromList: 'planets', toList: 'dwarfs', before: '' }])
	check('the server applies the drop', [group.dataset.state, lists()], [dropped, dropped])
	check('the dropped item keeps the focus', document.activeElement?.dataset?.dragItem, 'earth')

	// Empty the planets: Mars and Jupiter go right, each to its index there.
	for (const id of ['mars', 'jupiter']) {
		const before = group.dataset.state
		item(id).focus()
		press(item(id), 'ArrowRight', { altKey: true })
		release('Alt')
		await until(() => group.dataset.state !== before)
	}
	const emptied = 'planets= dwarfs=jupiter,pluto,mars,ceres,venus,earth'
	check('the planets are empty', [group.dataset.state, lists()], [emptied, emptied])

	// The arrows reach the empty list (it has tabindex="-1"), and go on from it.
	item('pluto').focus()
	press(item('pluto'), 'ArrowLeft')
	check('Arrow Left focuses the empty list', document.activeElement?.dataset?.dropList, 'planets')
	press(list('planets'), 'ArrowRight')
	check('Arrow Right from it focuses the first item there', document.activeElement?.dataset?.dragItem, 'jupiter')

	// A drop into the empty list.
	moves.length = 0
	await settle(250)
	await drag(item('venus'), list('planets'))
	await until(() => group.dataset.state !== emptied)
	const into = 'planets=venus dwarfs=jupiter,pluto,mars,ceres,earth'
	check('a drop into the empty list', moves, [{ itemId: 'venus', fromList: 'dwarfs', toList: 'planets', before: '' }])
	check('the server fills it', [group.dataset.state, lists()], [into, into])
	check('no errors', errors, [])
} catch (e) {
	rows.push({ step: 'script', error: String(e?.stack || e) })
}
await report()
`

const dragGroupListsJS = `
try {
	await customElements.whenDefined('sb-drag-group')
	const stage = document.createElement('div')
	stage.style.cssText = 'position: fixed; inset: 0; z-index: 10000; padding: 20px; background: Canvas'
	stage.innerHTML = '<style>#stage-lists, #outer { display: grid; grid-template-columns: repeat(3, 10rem); gap: 16px } #inner { display: grid; grid-template-columns: 1fr 1fr; gap: 8px } [data-drop-list] { min-block-size: 40px; padding: 4px; border: 1px dashed } [data-drag-item] { padding: 6px; border: 1px solid }</style>' +
		'<sb-drag-group id="stage-lists">' +
		'<section data-drop-list="a"><div data-drag-item="x" tabindex="0">x</div></section>' +
		'<section data-drop-list="b"></section>' +
		'<section data-drop-list="c"><div data-drag-item="y" tabindex="0">y</div></section>' +
		'</sb-drag-group>' +
		'<sb-drag-group id="outer" style="margin-block-start: 24px">' +
		'<section data-drop-list="a">' +
		'<div data-drag-item="g" tabindex="0"><span id="g-label">g</span><sb-drag-group id="inner">' +
		'<div data-drop-list="p"><div data-drag-item="p1" tabindex="0">p1</div></div>' +
		'<div data-drop-list="q"><div data-drag-item="q1" tabindex="0">q1</div></div>' +
		'</sb-drag-group></div>' +
		'<div data-drag-item="h" tabindex="0">h<div id="foreign"><div data-drag-item="f1" tabindex="0">f1</div></div></div>' +
		'</section>' +
		'<section data-drop-list="b"><div data-drag-item="z" tabindex="0">z</div></section>' +
		'</sb-drag-group>'
	document.body.append(stage)
	await settle()
	const $ = (s) => stage.querySelector(s)
	const moves = []
	stage.addEventListener('sb-drag-group-move', (e) => moves.push(e.target.id + ' ' + e.detail.itemId + ' ' + e.detail.fromList + '>' + e.detail.toList))
	const focused = () => document.activeElement?.dataset?.dragItem ?? document.activeElement?.dataset?.dropList

	// An empty list without a tabindex: the arrows pass over it.
	$('[data-drag-item="x"]').focus()
	press($('[data-drag-item="x"]'), 'ArrowRight')
	check('Arrow Right passes over an empty list', focused(), 'y')
	press($('[data-drag-item="y"]'), 'ArrowLeft')
	check('and so does Arrow Left', focused(), 'x')

	// With one, they stop there.
	$('#stage-lists [data-drop-list="b"]').tabIndex = -1
	press($('[data-drag-item="x"]'), 'ArrowRight')
	check('Arrow Right focuses an empty list with a tabindex', focused(), 'b')
	press($('#stage-lists [data-drop-list="b"]'), 'ArrowRight')
	check('Arrow Right goes on from it', focused(), 'y')
	press($('[data-drag-item="y"]'), 'ArrowLeft')
	press($('#stage-lists [data-drop-list="b"]'), 'ArrowLeft')
	check('Arrow Left goes back through it', focused(), 'x')

	// A keyboard move into the empty list.
	press($('[data-drag-item="x"]'), 'ArrowRight', { altKey: true })
	check('the empty list is the target', $('#stage-lists [data-drop-list="b"]').hasAttribute('data-drop-end'), true)
	release('Alt')
	check('a keyboard move into an empty list', moves, ['stage-lists x a>b'])

	// Nested groups: the nearest one owns each gesture.
	moves.length = 0
	await drag($('[data-drag-item="p1"]'), $('#inner [data-drop-list="q"]'))
	check('a drag in the inner group is the inner group\'s', moves, ['inner p1 p>q'])
	moves.length = 0
	await drag($('#g-label'), $('[data-drag-item="z"]'))
	check('a drag of the item holding it is the outer group\'s', moves, ['outer g a>b'])
	moves.length = 0
	$('[data-drag-item="p1"]').focus()
	press($('[data-drag-item="p1"]'), 'ArrowRight', { altKey: true })
	check('only the inner group stages a keyboard move', [$('#inner').hasAttribute('data-key-staging'), $('#outer').hasAttribute('data-key-staging')], [true, false])
	release('Alt')
	check('and sends it', moves, ['inner p1 p>q'])

	// A host registered by another copy of the core (a surface bundled with its own) owns its insides too.
	const core = await import('/c/drag-group/core/ownership.js')
	core.markRocketHost($('#foreign'))
	moves.length = 0
	await drag($('[data-drag-item="f1"]'), $('[data-drag-item="z"]'))
	check('the outer group leaves a drag inside another surface alone', [moves, document.querySelectorAll('[data-drag-preview]').length], [[], 0])
	$('[data-drag-item="f1"]').focus()
	check('and its keys', press($('[data-drag-item="f1"]'), 'ArrowRight', { altKey: true }), true)
	release('Alt')
	check('no errors', errors, [])
	stage.remove()
} catch (e) {
	rows.push({ step: 'script', error: String(e?.stack || e) })
}
await report()
`
