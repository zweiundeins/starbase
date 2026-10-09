package app_test

import "testing"

// TestPixelDetailsFollowNotch checks, per component, that its stepped shapes, stepped motion and
// pixelated canvases follow --sb-notch: at 1 the computed values are exactly those from before they
// did, at 0 plain shapes, smooth motion and canvases at the screen's resolution.
func TestPixelDetailsFollowNotch(t *testing.T) {
	for slug, script := range map[string]string{
		"data-table": dataTableNotchJS,
		"tree":       treeNotchJS,
	} {
		t.Run(slug, func(t *testing.T) {
			t.Parallel()
			_, body := probe(t, "/components/"+slug, pdPrelude+notchPrelude+"try {\n"+script+`
	check('no errors', errors, [])
} catch (e) {
	rows.push({ step: 'script', error: String(e?.stack || e) })
}
await report()
`)
			copyRows(t, body)
		})
	}
}

const notchPrelude = `
const make = async (html) => {
	const box = document.createElement('div')
	box.innerHTML = html
	document.body.prepend(box)
	const el = box.firstElementChild
	await customElements.whenDefined(el.localName)
	await until(() => el.shadowRoot?.firstElementChild)
	await settle()
	return el
}
const cs = (el, sel, pseudo) => getComputedStyle(el.shadowRoot.querySelector(sel), pseudo)
const notch = async (n) => {
	document.documentElement.style.setProperty('--sb-notch', n)
	await settle(150)
}
// The stepped arrows from before, 8×6 pointing down and 6×8 pointing right.
const DOWN = 'polygon(0px 0px, 8px 0px, 8px 2px, 6px 2px, 6px 4px, 5px 4px, 5px 6px, 3px 6px, 3px 4px, 2px 4px, 2px 2px, 0px 2px)'
const RIGHT = 'polygon(0px 0px, 2px 0px, 2px 1px, 4px 1px, 4px 3px, 6px 3px, 6px 5px, 4px 5px, 4px 7px, 2px 7px, 2px 8px, 0px 8px)'
const DOWN0 = 'polygon(0px 0px, 8px 0px, 6.667px 2px, 6.667px 2px, 5.333px 4px, 5.333px 4px, 4px 6px, 4px 6px, 2.667px 4px, 2.667px 4px, 1.333px 2px, 1.333px 2px)'
const RIGHT0 = 'polygon(0px 0px, 0px 0px, 1.5px 1px, 1.5px 1px, 4.5px 3px, 6px 4px, 6px 4px, 4.5px 5px, 1.5px 7px, 1.5px 7px, 0px 8px, 0px 8px)'
`

const dataTableNotchJS = `
	const el = await make('<sb-data-table columns=\'[{"key":"a","label":"A","sortable":true}]\' rows=\'[{"a":1}]\'></sb-data-table>')
	await notch('1')
	check('notch 1: the stepped sort arrow', cs(el, '.th button', '::after').clipPath, DOWN)
	await notch('0')
	check('notch 0: a triangle', cs(el, '.th button', '::after').clipPath, DOWN0)
`

const treeNotchJS = `
	const el = await make('<sb-tree items=\'[{"id":"a","label":"A","children":[{"id":"b","label":"B"}]}]\'></sb-tree>')
	const look = () => [cs(el, '.caret', '::before').clipPath, cs(el, '.caret', '::before').transition]
	await notch('1')
	check('notch 1: the stepped caret, turning in two steps', look(), [RIGHT, 'rotate 0.12s steps(2)'])
	await notch('0')
	check('notch 0: a triangle, turning smoothly', look(), [RIGHT0, 'rotate 0.12s steps(1000)'])
`
