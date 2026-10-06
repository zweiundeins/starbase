package app_test

import "testing"

// TestInlineEditRename renames the nicknames of sb-inline-edit's demo: the
// server opens the field, saves a valid name, refuses an empty or a long one
// with the text kept, and closes the field on Escape.
func TestInlineEditRename(t *testing.T) {
	_, body := probe(t, "/components/inline-edit", pdPrelude+inlineEditJS)
	copyRows(t, body)
}

const inlineEditJS = `
// A real press on the name also moves the focus to it.
const doublePress = (el) => {
	el.focus()
	for (let i = 0; i < 2; i++) el.dispatchEvent(new PointerEvent('pointerdown', { button: 0, bubbles: true, composed: true }))
}
try {
	await customElements.whenDefined('sb-inline-edit')
	const earth = document.getElementById('nick-earth')
	const field = (host) => host.querySelector('[data-inline-edit-input]')
	const name = (host) => host.querySelector('[data-inline-edit-trigger]')?.textContent
	check('the server rendered the nickname', [earth.dataset.state, name(earth)], ['body=earth name=Blue+Marble', 'Blue Marble'])

	// A double press asks for the field; the server renders it, and it takes the focus.
	doublePress(earth.querySelector('[data-inline-edit-trigger]'))
	await until(() => field(earth))
	check('the field', [field(earth)?.value, document.activeElement === field(earth)], ['Blue Marble', true])

	// Enter commits; the server saves and the name takes the focus back.
	field(earth).value = 'Pale Blue Dot'
	press(field(earth), 'Enter')
	await until(() => name(earth) === 'Pale Blue Dot')
	check('a saved nickname', [earth.dataset.state, name(earth), document.activeElement === earth.querySelector('[data-inline-edit-trigger]')], ['body=earth name=Pale+Blue+Dot back=1', 'Pale Blue Dot', true])

	// The keyboard's way in: Enter on the name.
	press(earth.querySelector('[data-inline-edit-trigger]'), 'Enter')
	await until(() => field(earth))
	check('Enter opens the field', document.activeElement === field(earth), true)

	// An empty nickname: refused, with the reason, the text kept and the focus in the field.
	field(earth).value = '   '
	press(field(earth), 'Enter')
	await until(() => earth.querySelector('[role="alert"]'))
	check('an empty nickname is refused', [earth.querySelector('[role="alert"]')?.textContent, field(earth)?.value, field(earth)?.getAttribute('aria-invalid'), document.activeElement === field(earth)], ['A nickname needs at least one character.', '   ', 'true', true])
	const long = 'Third rock from the Sun, give or take a few'
	field(earth).value = long
	press(field(earth), 'Enter')
	await until(() => earth.querySelector('[role="alert"]')?.textContent.includes('40'))
	check('a long one too', [earth.querySelector('[role="alert"]')?.textContent, field(earth)?.value], ['A nickname has at most 40 characters.', long])

	// Escape closes the field; the saved nickname is back.
	press(field(earth), 'Escape')
	await until(() => !field(earth))
	check('Escape cancels', [name(earth), earth.dataset.state], ['Pale Blue Dot', 'body=earth name=Pale+Blue+Dot back=1'])

	// A commit on one title and a request on another cross: neither is lost.
	const mars = document.getElementById('nick-mars'), jupiter = document.getElementById('nick-jupiter')
	doublePress(mars.querySelector('[data-inline-edit-trigger]'))
	await until(() => field(mars))
	field(mars).value = 'Rust Bucket'
	doublePress(jupiter.querySelector('[data-inline-edit-trigger]')) // Mars's field loses the focus: a commit
	await until(() => name(mars) === 'Rust Bucket' && field(jupiter))
	check('both answers arrive', [name(mars), !!field(jupiter)], ['Rust Bucket', true])
	check('no errors', errors, [])
} catch (e) {
	rows.push({ step: 'script', error: String(e?.stack || e) })
}
await report()
`
