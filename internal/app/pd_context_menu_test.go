package app_test

import "testing"

// TestContextMenu opens sb-context-menu's demo from a button and with a
// right click, walks it with the keyboard, chooses actions the server
// applies, checks where the focus goes after each, and closes it with
// Escape, also inside a non-modal drawer. The keys are synthetic; the
// browser's own close requests (Escape in a modal) need real ones.
func TestContextMenu(t *testing.T) {
	_, body := probe(t, "/components/context-menu", pdPrelude+contextMenuJS)
	copyRows(t, body)
}

// TestContextMenuOnAPhone opens the demo's submenu on a narrow screen, where
// the menu's data-sb-mobile-sheet (patch 0059) lets the example's CSS make it
// a sheet with the submenu on top.
func TestContextMenuOnAPhone(t *testing.T) {
	_, body := probeAt(t, "", "/components/context-menu", pdPrelude+contextMenuPhoneJS, "--window-size=375,740")
	copyRows(t, body)
}

const contextMenuJS = `
try {
	await customElements.whenDefined('sb-context-menu')
	const demo = document.getElementById('planet-menus')
	const menu = document.getElementById('planet-menus-menu')
	const row = (id) => document.getElementById('planet-menus-' + id)
	const button = (id) => row(id).querySelector('button')
	const item = (action) => menu.querySelector('[data-action="' + action + '"]')
	const focused = () => document.activeElement?.textContent.trim()
	const open = () => menu.matches(':popover-open')
	const actions = []
	menu.addEventListener('sb-menu-action', (e) => actions.push(e.detail))

	// From the button, as Enter or Space would: the first item takes the focus.
	button('earth').focus()
	button('earth').click()
	check('a button opens it on its first item', [open(), focused(), button('earth').getAttribute('aria-expanded')], [true, 'Move to the top', 'true'])
	const step = document.getElementById('planet-menus-step')
	check('the submenu stays hidden', step.checkVisibility(), false)
	press(document.activeElement, 'ArrowUp')
	check('Arrow Up wraps around, to a label from the row', focused(), 'Remove Earth')
	press(document.activeElement, 'k')
	check('k: the previous item', focused(), 'Move one place')
	press(document.activeElement, 'ArrowRight')
	check('Arrow Right opens the submenu on its first item', [step.matches(':popover-open'), focused()], [true, 'Up'])
	press(document.activeElement, 'Escape')
	check('Escape closes the submenu only', [step.isConnected && step.matches(':popover-open'), open(), focused()], [false, true, 'Move one place'])
	press(document.activeElement, 'Escape')
	check('Escape closes the menu and returns the focus', [open(), document.activeElement === button('earth'), button('earth').getAttribute('aria-expanded')], [false, true, 'false'])

	// Enter on the menu element itself chooses nothing.
	button('earth').click()
	menu.focus()
	press(menu, 'Enter')
	press(menu, ' ')
	check('Enter on the menu chooses nothing', [open(), actions], [true, []])

	// Tab closes it and hands the focus to the trigger, for the browser to move on.
	press(document.activeElement, 'Tab')
	check('Tab closes it', [open(), document.activeElement === button('earth')], [false, true])

	// A second click on the button closes it: light dismiss runs between pointerdown and click.
	button('earth').click()
	button('earth').dispatchEvent(new PointerEvent('pointerdown', { bubbles: true, composed: true }))
	button('earth').dispatchEvent(new MouseEvent('click', { bubbles: true, cancelable: true, detail: 1 }))
	check('a second click closes it', open(), false)

	// The first planet can't move up: those items are off.
	button('mercury').click()
	check('it opens on the first item that applies', [focused(), item('top').getAttribute('aria-disabled'), item('up').getAttribute('aria-disabled')], ['Move to the bottom', 'true', 'true'])
	item('top').click()
	check('a disabled item does nothing', [open(), actions], [true, []])
	press(document.activeElement, 'Escape')

	// Choose an action; the server moves Earth, and the focus stays with it.
	button('earth').click()
	press(document.activeElement, 'ArrowDown')
	check('the second item', focused(), 'Move to the bottom')
	document.activeElement.click()
	await until(() => demo.dataset.state !== 'mercury venus earth mars')
	check('the server moved it', demo.dataset.state, 'mercury venus mars earth')
	check('in the list too', [...demo.querySelectorAll('li')].map((li) => li.dataset.contextId).join(' '), 'mercury venus mars earth')
	check('the focus stays with Earth', document.activeElement === button('earth'), true)
	check('the rows know their new ends', [row('mars').dataset.menuDisabled, row('earth').dataset.menuDisabled], [undefined, 'bottom down'])

	// A right click opens it at the pointer, for that row, which takes the focus back.
	const span = row('venus').querySelector('span')
	const r = span.getBoundingClientRect()
	span.dispatchEvent(new MouseEvent('contextmenu', { bubbles: true, cancelable: true, clientX: r.left + 5, clientY: r.top + 5 }))
	check('a right click opens it', await until(open), true)
	check('the row takes no aria-expanded', row('venus').hasAttribute('aria-expanded'), false)
	item('top').click()
	check('the row has the focus', document.activeElement === row('venus'), true)
	await until(() => demo.dataset.state.startsWith('venus'))
	check('the server moved Venus, with the focus', [demo.dataset.state, document.activeElement === row('venus')], ['venus mercury mars earth', true])

	// Removing a planet hands the focus to the next one, also twice in a row.
	button('mercury').click()
	item('remove').click()
	await until(() => demo.dataset.state.includes('-mercury'))
	check('the server removed Mercury', demo.dataset.state, 'venus *mars earth -mercury')
	check('the focus moves to Mars', await until(() => document.activeElement === button('mars')), true)
	button('venus').click()
	item('remove').click()
	await until(() => demo.dataset.state.includes('-venus'))
	check('and Venus', demo.dataset.state, '*mars earth -mercury -venus')
	check('the focus is on Mars again', await until(() => document.activeElement === button('mars')), true)
	const restore = demo.querySelector('.demo-menu__restore')
	check('the server offers them back', restore?.textContent, 'Bring back Mercury, Venus')
	restore.focus()
	restore.click()
	await until(() => !demo.dataset.state.includes('-'))
	check('back again', demo.dataset.state, 'mars earth *mercury venus')
	check('with the focus', await until(() => document.activeElement === button('mercury')), true)

	// An answer that arrives while the menu is open leaves it open, by its button.
	button('earth').click()
	const before = menu.getBoundingClientRect().top - button('earth').getBoundingClientRect().bottom
	demo.dispatchEvent(new CustomEvent('sb-menu-action', { detail: { action: 'bottom', contextId: 'mars' } }))
	await until(() => demo.dataset.state.startsWith('earth'))
	await settle(100)
	const after = menu.getBoundingClientRect().top - button('earth').getBoundingClientRect().bottom
	check('a morph leaves the open menu by its button', [demo.dataset.state, open(), focused(), Math.round(after - before), button('earth').getAttribute('aria-expanded')], ['earth mercury venus mars', true, 'Move to the top', 0, 'true'])
	press(document.activeElement, 'Escape')

	// A morph that strips popover from the open menu doesn't break the next open.
	button('earth').click()
	menu.removeAttribute('popover')
	check('the stripped menu is closed', [open(), menu.isOpen()], [false, false])
	button('earth').click()
	check('and opens again', [open(), focused()], [true, 'Move to the bottom'])
	press(document.activeElement, 'Escape')

	// In a non-modal drawer, Escape closes the menu first, then the drawer.
	await customElements.whenDefined('sb-drawer')
	const box = document.createElement('div')
	box.innerHTML = '<sb-drawer id="pd-drawer" modal="false" heading="Drawer"><button type="button" id="pd-trigger" data-menu-for="pd-menu">⋯</button>' +
		'<sb-context-menu id="pd-menu" aria-label="Actions"><template data-sb-menu><button type="button" role="menuitem" data-action="x">X</button></template></sb-context-menu></sb-drawer>'
	document.body.append(box)
	const drawer = document.getElementById('pd-drawer')
	await settle(100)
	drawer.show()
	await until(() => drawer.isOpen)
	const inner = document.getElementById('pd-menu')
	document.getElementById('pd-trigger').click()
	check('the menu opens in the drawer', inner.matches(':popover-open'), true)
	press(document.activeElement, 'Escape')
	await settle(100)
	check('Escape closes the menu, not the drawer', [inner.matches(':popover-open'), drawer.isOpen], [false, true])
	press(document.activeElement, 'Escape')
	await settle(100)
	check('the next Escape closes the drawer', drawer.isOpen, false)
	box.remove()
	check('no errors', errors, [])
} catch (e) {
	rows.push({ step: 'script', error: String(e?.stack || e) })
}
await report()
`

const contextMenuPhoneJS = `
try {
	await customElements.whenDefined('sb-context-menu')
	const menu = document.getElementById('planet-menus-menu')
	const box = (el) => {
		const r = el.getBoundingClientRect()
		return { left: Math.round(r.left), right: Math.round(r.right), top: Math.round(r.top), bottom: Math.round(r.bottom) }
	}
	check('a phone', innerWidth <= 650, true)
	document.querySelector('#planet-menus-earth button').click()
	menu.querySelector('[data-submenu]').click()
	const step = document.getElementById('planet-menus-step')
	const m = box(menu), s = box(step), width = document.documentElement.clientWidth
	check('both are sheets', [menu.hasAttribute('data-mobile-sheet'), step.hasAttribute('data-mobile-sheet'), step.matches(':popover-open')], [true, true, true])
	check('the menu sits at the bottom, full width', [m.left, m.right, m.bottom], [0, width, innerHeight])
	check('the submenu sits on top of it', [s.left, s.right, s.bottom <= m.top, s.top >= 0], [0, width, true, true])
	check('no errors', errors, [])
} catch (e) {
	rows.push({ step: 'script', error: String(e?.stack || e) })
}
await report()
`
