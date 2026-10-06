package app_test

import "testing"

// TestContextMenu opens sb-context-menu's demo from a button and with a
// right click, walks it with the keyboard, chooses actions the server
// applies, and closes it with Escape, also inside a non-modal drawer.
func TestContextMenu(t *testing.T) {
	_, body := probe(t, "/components/context-menu", pdPrelude+contextMenuJS)
	copyRows(t, body)
}

const contextMenuJS = `
try {
	await customElements.whenDefined('sb-context-menu')
	const demo = document.getElementById('planet-menus')
	const menu = document.getElementById('planet-menus-menu')
	const button = (id) => document.querySelector('#planet-menus-' + id + ' button')
	const focused = () => document.activeElement?.textContent.trim()
	const open = () => menu.matches(':popover-open')

	// From the button, as Enter or Space would.
	button('earth').focus()
	button('earth').click()
	check('a button opens it', [open(), document.activeElement === menu, button('earth').getAttribute('aria-expanded')], [true, true, 'true'])
	press(menu, 'ArrowDown')
	check('Arrow Down: the first item', focused(), 'Move to the top')
	press(document.activeElement, 'ArrowUp')
	check('Arrow Up wraps around', focused(), 'Remove')
	press(document.activeElement, 'k')
	check('k: the previous item', focused(), 'Move one place')
	press(document.activeElement, 'ArrowRight')
	const step = document.getElementById('planet-menus-step')
	check('Arrow Right opens the submenu', step?.matches(':popover-open'), true)
	press(document.activeElement, 'ArrowDown')
	check('its first item', focused(), 'Up')
	press(document.activeElement, 'Escape')
	check('Escape closes the submenu only', [step.isConnected && step.matches(':popover-open'), open(), focused()], [false, true, 'Move one place'])
	press(document.activeElement, 'Escape')
	check('Escape closes the menu and returns the focus', [open(), document.activeElement === button('earth'), button('earth').getAttribute('aria-expanded')], [false, true, 'false'])

	// Choose an action; the server moves Earth, and the focus stays with it.
	button('earth').click()
	press(menu, 'ArrowDown')
	press(document.activeElement, 'ArrowDown')
	check('the second item', focused(), 'Move to the bottom')
	document.activeElement.click()
	await until(() => demo.dataset.state !== 'mercury venus earth mars')
	check('the server moved it', demo.dataset.state, 'mercury venus mars earth')
	check('in the list too', [...demo.querySelectorAll('li')].map((li) => li.dataset.contextId).join(' '), 'mercury venus mars earth')
	check('the focus stays with Earth', document.activeElement === button('earth'), true)

	// A right click opens it at the pointer, for that row.
	const mars = document.querySelector('#planet-menus-mars span')
	const r = mars.getBoundingClientRect()
	mars.dispatchEvent(new MouseEvent('contextmenu', { bubbles: true, cancelable: true, clientX: r.left + 5, clientY: r.top + 5 }))
	check('a right click opens it', await until(open), true)
	menu.querySelector('[data-action="remove"]').click()
	await until(() => demo.dataset.state.includes('-mars'))
	check('the server removed Mars', demo.dataset.state, 'mercury venus earth -mars')
	const restore = demo.querySelector('.demo-menu__restore')
	check('and offers it back', restore?.textContent, 'Bring back Mars')
	restore.click()
	await until(() => !demo.dataset.state.includes('-'))
	check('back again', demo.dataset.state, 'mercury venus earth mars')

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
	press(inner, 'Escape')
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
