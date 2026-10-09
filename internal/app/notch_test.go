package app_test

import "testing"

// TestPixelDetailsFollowNotch checks, per component, that its stepped shapes, stepped motion and
// pixelated canvases follow --sb-notch: at 1 the computed values are exactly those from before they
// did, at 0 plain shapes, smooth motion and canvases at the screen's resolution.
func TestPixelDetailsFollowNotch(t *testing.T) {
	for slug, script := range map[string]string{
		"data-table": dataTableNotchJS,
		"tree":       treeNotchJS,
		"details":    detailsNotchJS,
		"popover":    popoverNotchJS,
		"toggle":     toggleNotchJS,
		"meter":      meterNotchJS,
		"busy":       busyNotchJS,
		"button":     buttonNotchJS,
		"toast":      toastNotchJS,
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

const detailsNotchJS = `
	const el = await make('<sb-details summary="Hull">Intact.</sb-details>')
	const look = () => [cs(el, '.marker', '::before').clipPath, cs(el, '.marker', '::before').transition]
	await notch('1')
	check('notch 1: the stepped marker, turning in three steps', look(), [RIGHT, 'rotate 0.22s steps(3)'])
	await notch('0')
	check('notch 0: a triangle, turning smoothly', look(), [RIGHT0, 'rotate 0.22s steps(1000)'])
`

// The arrow per side, and flipped to the other side (--_c2), from the stepped tip of before.
const popoverNotchJS = `
	const el = await make('<sb-popover arrow>Fuel: 82%</sb-popover>')
	const pop = el.shadowRoot.querySelector('.pop'), arrow = el.shadowRoot.querySelector('.arrow')
	const tip = (pts) => 'polygon(' + pts.map(([x, y]) => x + 'px ' + y + 'px').join(', ') + ')'
	const turn = { top: (x, y) => [x, 8 - y], bottom: (x, y) => [x, y], start: (x, y) => [8 - y, x], end: (x, y) => [y, x] }
	const flip = { top: 'bottom', bottom: 'top', start: 'end', end: 'start' }
	const shapes = (pts) => Object.keys(turn).flatMap((side) => [tip(pts.map(([x, y]) => turn[side](x, y))), tip(pts.map(([x, y]) => turn[flip[side]](x, y)))])
	const look = () => Object.keys(turn).flatMap((side) => {
		pop.setAttribute('data-side', side)
		arrow.style.removeProperty('--_c')
		const own = getComputedStyle(arrow).clipPath
		arrow.style.setProperty('--_c', 'var(--_c2)')
		return [own, getComputedStyle(arrow).clipPath]
	})
	await notch('1')
	check('notch 1: the stepped arrow on every side', look(), shapes([[4, 2], [8, 2], [8, 4], [10, 4], [10, 6], [12, 6], [12, 8], [0, 8], [0, 6], [2, 6], [2, 4], [4, 4]]))
	await notch('0')
	check('notch 0: a triangle on every side', look(), shapes([[6, 2], [6, 2], [8, 4], [8, 4], [10, 6], [10, 6], [12, 8], [0, 8], [2, 6], [2, 6], [4, 4], [4, 4]]))
`

const toggleNotchJS = `
	const el = await make('<sb-toggle label="Shields"></sb-toggle>')
	const look = () => [cs(el, 'button').transition, cs(el, '.knob').transition]
	await notch('1')
	check('notch 1: stepped', look(), ['background 0.18s steps(3)', 'inset-inline-start 0.18s steps(5)'])
	await notch('0')
	check('notch 0: smooth', look(), ['background 0.18s steps(1000)', 'inset-inline-start 0.18s steps(1000)'])
`

const meterNotchJS = `
	const el = await make('<sb-meter value="50"></sb-meter>')
	await notch('1')
	check('notch 1: blocks light in two steps', cs(el, '.bar > *').transition, '0.12s steps(2) 0.09s')
	await notch('0')
	check('notch 0: smoothly', cs(el, '.bar > *').transition, '0.12s steps(1000) 0.09s')
`

// The spinner's dots also get a trail at notch 0: the keyframe at 12.5% fades to 0.6, not 0.22.
const busyNotchJS = `
	const spinner = await make('<sb-busy busy></sb-busy>')
	const bar = await make('<sb-busy busy variant="bar" value="40"></sb-busy>')
	const sweep = await make('<sb-busy busy variant="bar"></sb-busy>')
	const skel = await make('<sb-busy busy variant="skeleton"></sb-busy>')
	const dot = spinner.shadowRoot.querySelector('.spinner span')
	// The first dot's opacity just after its lit eighth of the turn.
	const trail = () => {
		const a = dot.getAnimations()[0]
		a.pause()
		a.currentTime = 91
		const at = +getComputedStyle(dot).opacity
		a.play()
		return Math.round(at * 100) / 100
	}
	const look = () => [cs(spinner, '.spinner span').animation, cs(bar, '.bar .fill').transition, cs(sweep, '.bar.sweeping .fill').animation, cs(skel, '.skel span').animation, cs(spinner, '.spinner span').borderRadius]
	await notch('1')
	check('notch 1: stepped, square blocks', look(), ['0.72s steps(1) infinite sb-busy-blink', 'inline-size 0.16s steps(6)', '1.1s steps(9) infinite sb-busy-sweep', '1.4s steps(12) infinite sb-busy-shimmer', '0px'])
	check('notch 1: a lit dot goes dark at once', trail(), 0.22)
	await notch('0')
	check('notch 0: smooth, round dots', look(), ['0.72s steps(1000) infinite sb-busy-blink', 'inline-size 0.16s steps(1000)', '1.1s steps(1000) infinite sb-busy-sweep', '1.4s steps(1000) infinite sb-busy-shimmer', '2.5px'])
	check('notch 0: a lit dot fades, leaving a trail', trail(), 0.6)
`

const buttonNotchJS = `
	const el = await make('<sb-button variant="pixel" loading>Launch</sb-button>')
	// The radius as a share of the block's size: 0, or half of it (a dot).
	const look = () => [cs(el, '.spin i').animation, Math.round((parseFloat(cs(el, '.spin i').borderRadius) / parseFloat(cs(el, '.spin i').blockSize)) * 10) / 10]
	await notch('1')
	check('notch 1: square blocks that blink', look(), ['0.72s steps(1) infinite sb-button-blink', 0])
	await notch('0')
	check('notch 0: round dots that fade', look(), ['0.72s steps(1000) infinite sb-button-blink', 0.5])
`

const toastNotchJS = `
	const el = await make('<sb-toast placement="inline" duration="0" toasts=\'[{"id":"a","variant":"ok","title":"Docked"}]\'></sb-toast>')
	await until(() => el.shadowRoot.querySelector('.light'))
	const look = () => [cs(el, '.light').clipPath, cs(el, '.light').borderRadius]
	await notch('1')
	check('notch 1: the notched status light', look(), ['polygon(3px 0px, calc(100% - 3px) 0px, calc(100% - 3px) 3px, 100% 3px, 100% calc(100% - 3px), calc(100% - 3px) calc(100% - 3px), calc(100% - 3px) 100%, 3px 100%, 3px calc(100% - 3px), 0px calc(100% - 3px), 0px 3px, 3px 3px)', '0px'])
	await notch('0')
	check('notch 0: a dot', look(), ['polygon(0px 0px, 100% 0px, 100% 0px, 100% 0px, 100% 100%, 100% 100%, 100% 100%, 0px 100%, 0px 100%, 0px 100%, 0px 0px, 0px 0px)', '6px'])
`
