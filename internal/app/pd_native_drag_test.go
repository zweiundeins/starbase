package app_test

import "testing"

// TestPDNativeDragStart checks patches/pd-rockets 0004 in three surfaces: a
// press that can become the surface's drag cancels the browser's own drag of
// a link or an image under it (which would send pointercancel), and a link
// the surface doesn't drag by keeps its own.
func TestPDNativeDragStart(t *testing.T) {
	_, body := probe(t, "/components/kanban-board", pdPrelude+nativeDragJS)
	copyRows(t, body)
}

const nativeDragJS = `
try {
	for (const tag of ['sb-kanban-board', 'sb-sortable-list', 'sb-drag-group']) await customElements.whenDefined(tag)
	const img = '<img alt="" width="16" height="16" src="data:image/gif;base64,R0lGODlhAQABAAAAACw=">'
	document.body.insertAdjacentHTML('afterbegin', ` + "`" + `
		<sb-kanban-board id="nd-board"><section data-kanban-lane data-col="1"><div data-kanban-lane-cards>
			<article data-kanban-card="a" tabindex="0"><a href="#a" data-kanban-card-main>Handle</a> <a href="#b" class="other">Other</a></article>
		</div></section></sb-kanban-board>
		<sb-sortable-list id="nd-list"><div data-sortable-item="x" tabindex="0">${img} X</div><div data-sortable-item="y" tabindex="0">Y</div></sb-sortable-list>
		<sb-drag-group id="nd-group"><section data-drop-list="l"><div data-drag-item="p" tabindex="0">${img} P</div></section></sb-drag-group>` + "`" + `)
	await settle()
	// dragstart after a press on el; true when the surface cancelled it.
	const pressThenDrag = (el) => {
		const at = center(el)
		pointer(el, 'pointerdown', at)
		const ev = new DragEvent('dragstart', { bubbles: true, composed: true, cancelable: true })
		el.dispatchEvent(ev)
		pointer(el, 'pointerup', at)
		return ev.defaultPrevented
	}
	const q = (s) => document.querySelector(s)
	check('kanban: a card handle that is a link', pressThenDrag(q('#nd-board [data-kanban-card-main]')), true)
	check('sortable-list: an image in an item', pressThenDrag(q('#nd-list img')), true)
	check('drag-group: an image in an item', pressThenDrag(q('#nd-group img')), true)
	check('kanban: a link that is not the handle keeps its drag', pressThenDrag(q('#nd-board .other')), false)
	const unpressed = new DragEvent('dragstart', { bubbles: true, cancelable: true })
	q('#nd-list img').dispatchEvent(unpressed)
	check('without a press, nothing is cancelled', unpressed.defaultPrevented, false)
	check('no errors', errors, [])
} catch (e) {
	rows.push({ step: 'script', error: String(e?.stack || e) })
}
await report()
`
