package app_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/starfederation/datastar-go/datastar"
)

// TestKanbanBoardLanding checks sb-kanban-board's landing markers (patches
// 0130 to 0132) with the demo server's answers held until the script lets
// each one through or refuses it: the marker at the drop slot, through a
// morph of the target lane, released when the card arrives, by a refusal, by
// releaseLanding() and by the timeout; keyboard drops, lane moves, and no
// marker without data-kanban-landing.
func TestKanbanBoardLanding(t *testing.T) {
	handlers := func(app http.Handler) map[string]http.HandlerFunc {
		decide := make(chan string, 8)
		return map[string]http.HandlerFunc{
			"/demo/arrange/kanban-board": func(w http.ResponseWriter, r *http.Request) {
				select {
				case d := <-decide:
					if d == "refuse" {
						w.Header().Set("Access-Control-Allow-Origin", "*")
						http.Error(w, "refused", http.StatusUnprocessableEntity)
						return
					}
					q := r.URL.Query()
					q.Del("delay")
					r.URL.RawQuery = q.Encode()
					app.ServeHTTP(w, r)
				case <-r.Context().Done():
				case <-time.After(30 * time.Second):
				}
			},
			"/__land": func(w http.ResponseWriter, r *http.Request) {
				decide <- r.URL.Query().Get("do")
				w.WriteHeader(http.StatusNoContent)
			},
			// Another person's change: the lane the script sends, morphed in by id.
			"/__morph": func(w http.ResponseWriter, r *http.Request) {
				datastar.NewSSE(w, r).PatchElements(r.URL.Query().Get("html"))
			},
		}
	}
	_, body := probeWith(t, "/components/kanban-board", pdPrelude+kanbanLandingJS, handlers)
	copyRows(t, body)
}

const kanbanLandingJS = `
try {
	await customElements.whenDefined('sb-kanban-board')
	const board = document.getElementById('mission-board')
	board.scrollIntoView({ block: 'center' })
	const lane = (col) => board.querySelector('[data-kanban-lane][data-col="' + col + '"]')
	const card = (id) => board.querySelector('[data-kanban-card="' + id + '"]')
	const grip = (col) => lane(col).querySelector('[data-kanban-lane-grip]')
	const laneOf = (id) => card(id)?.closest('[data-kanban-lane]')?.dataset.col
	const marker = () => document.querySelector('[data-landing-marker]')
	const answer = (what = 'go') => fetch('/__land?do=' + what)
	const frames = (n = 2) => new Promise((r) => { const step = () => (n-- > 0 ? requestAnimationFrame(step) : r()); step() })
	const rect = (el) => { const r = el.getBoundingClientRect(); return { left: r.left, top: r.top, width: r.width, height: r.height } }
	// Where the marker is placed: its own rect also moves while it flies in from the drop.
	const slot = (m) => ({ left: parseFloat(m?.style.left), top: parseFloat(m?.style.top), width: parseFloat(m?.style.width), height: parseFloat(m?.style.height) })
	const near = (a, b) => Math.abs(a - b) <= 1
	const end = (col) => { const r = lane(col).getBoundingClientRect(); return { x: r.left + r.width / 2, y: r.bottom - 4 } }
	const ends = []
	board.addEventListener('sb-kanban-landing-end', (e) => ends.push(e.detail))
	// The gap a landing opens: an animation of a card's margin.
	const gaps = (el) => el.getAnimations().filter((a) => a.effect.getKeyframes().some((k) => 'marginTop' in k || 'marginBottom' in k)).length
	const animated = []
	const animate = Element.prototype.animate
	Element.prototype.animate = function (...args) {
		if (this.dataset?.kanbanCard && args[0]?.[0]?.transform) animated.push(this.dataset.kanbanCard)
		return animate.apply(this, args)
	}
	const gap = parseFloat(getComputedStyle(lane(9).querySelector('[data-kanban-lane-cards]')).rowGap)

	// A drop into Visited: a marker the size of Mars below Moon, in a gap the lane
	// opens there; Mars stays in its lane until the server answers.
	const mars = rect(card('mars'))
	await drag(card('mars'), end(9))
	await frames(4)
	let m = marker()
	const moon = rect(card('moon'))
	check('a marker at the drop slot', [m?.dataset.landingMarker, near(slot(m).top, moon.top + moon.height + gap), near(slot(m).height, mars.height), near(slot(m).left, moon.left), near(slot(m).width, moon.width)], ['card', true, true, true, true])
	check('the marker is out of reach', [m.getAttribute('aria-hidden'), m.inert, getComputedStyle(m).pointerEvents, m.parentElement === document.body, m.hasAttribute('id')], ['true', true, 'none', true, false])
	check('the card stays where the server put it', laneOf('mars'), '4')
	check('the lane opens a gap', gaps(card('moon')), 1)

	// Once the gap is open and nothing moves, a pending landing reads no layout.
	await settle(300)
	let reads = 0
	const measure = Element.prototype.getBoundingClientRect
	Element.prototype.getBoundingClientRect = function () { reads++; return measure.call(this) }
	await settle(400)
	Element.prototype.getBoundingClientRect = measure
	check('an idle landing reads nothing', reads, 0)

	// Another person's card lands at the top of Visited: the lane is morphed in
	// full, and the marker stays, following its slot. (The demo's board ignores
	// morphs until its own answer; an app's board doesn't.)
	board.removeAttribute('data-ignore-morph')
	const lane9 = lane(9).cloneNode(true)
	lane9.querySelector('[data-kanban-lane-cards]').insertAdjacentHTML('afterbegin', '<article id="mission-board-ceres" data-kanban-card="ceres" tabindex="0">🪨 Ceres</article>')
	document.body.insertAdjacentHTML('beforeend', '<button id="morph" hidden data-on:click="@get(\'/__morph?html=\' + encodeURIComponent(el.dataset.html))"></button>')
	const trigger = document.getElementById('morph')
	trigger.dataset.html = lane9.outerHTML
	await settle(100)
	trigger.click()
	await until(() => card('ceres'))
	board.setAttribute('data-ignore-morph', '')
	await frames(4)
	m = marker()
	const moonAfter = rect(card('moon'))
	check('the marker survives the morph and follows', [!!m, moonAfter.top > moon.top, near(slot(m).top, moonAfter.top + moonAfter.height + gap)], [true, true, true])

	// The answer moves Mars into Visited: the marker goes, and Mars comes from it.
	await answer()
	await until(() => !marker())
	check('released when the card arrives', [laneOf('mars'), ends.at(-1), gaps(card('moon'))], ['9', { cardId: 'mars', col: 9, reason: 'arrived' }, 0])
	check('the card animates from the marker', animated.includes('mars'), true)

	// A refused move releases its marker (the demo's datastar-fetch handler).
	const before = board.dataset.state
	const europa = card('europa').getBoundingClientRect()
	await drag(card('jupiter'), { x: europa.left + europa.width / 2, y: europa.top + 3 })
	await frames(4)
	check('a marker before Europa', [!!marker(), near(slot(marker()).top, europa.top)], [true, true])
	await answer('refuse')
	await until(() => !marker())
	check('a refusal releases it', [ends.at(-1), laneOf('jupiter'), board.dataset.state], [{ cardId: 'jupiter', col: 7, reason: 'released' }, '4', before])

	// releaseLanding() by the page, once the refused gap has closed.
	await settle(300)
	await drag(card('neptune'), end(7))
	await frames(4)
	const released = board.releaseLanding({ cardId: 'neptune' })
	check('releaseLanding({ cardId })', [released, !!marker(), ends.at(-1)], [true, false, { cardId: 'neptune', col: 7, reason: 'released' }])
	await answer('refuse')
	await settle(300)

	// The timeout.
	board.setAttribute('data-kanban-landing-timeout', '300')
	await drag(card('neptune'), end(7))
	await frames(4)
	check('a marker until the timeout', !!marker(), true)
	await settle(500)
	check('the timeout releases it', [!!marker(), ends.at(-1)], [false, { cardId: 'neptune', col: 7, reason: 'timeout' }])
	await answer('refuse')
	await settle(300)
	board.setAttribute('data-kanban-landing-timeout', '3000')

	// A keyboard move: Titan, second in En route, into Visited before its second card.
	const second = [...lane(9).querySelectorAll('[data-kanban-card]')][1]
	card('titan').focus()
	press(card('titan'), 'ArrowRight', { altKey: true })
	release('Alt')
	await settle(250)
	check('a keyboard move gets a marker', [!!marker(), near(slot(marker()).top + slot(marker()).height + gap, rect(second).top), gaps(second)], [true, true, 1])
	await answer()
	await until(() => !marker())
	check('and arrives', [laneOf('titan'), ends.at(-1)], ['9', { cardId: 'titan', col: 9, reason: 'arrived' }])

	// A lane move: an outline over the place the lane takes.
	await settle(300)
	await drag(grip(9), lane(4))
	await frames(4)
	m = marker()
	check('a lane move gets a marker', [m?.dataset.landingMarker, near(slot(m).left, rect(lane(4)).left), near(slot(m).width, rect(lane(4)).width)], ['lane', true, true])
	await answer()
	await until(() => !marker())
	check('and arrives', [[...board.querySelectorAll('[data-kanban-lane]')].map((l) => l.dataset.col).join(' '), ends.at(-1)], ['9 4 7', { col: 9, reason: 'arrived' }])

	// A slot in a lane that scrolls: the marker follows the scrolling and is clipped by it.
	await settle(300)
	const list4 = lane(4).querySelector('[data-kanban-lane-cards]')
	list4.style.cssText = 'max-block-size: 60px; overflow: auto'
	const into = list4.querySelectorAll('[data-kanban-card]')[1].getBoundingClientRect()
	await drag(card('moon'), { x: into.left + into.width / 2, y: into.top + 3 })
	await settle(250)
	const top = slot(marker()).top
	list4.scrollTop = 20
	await frames(3)
	check('the marker follows a scrolling lane', [near(slot(marker()).top, top - 20), marker().style.clipPath.startsWith('inset(')], [true, true])
	await answer('refuse')
	await until(() => !marker())
	list4.style.cssText = ''

	// Without data-kanban-landing nothing changes: no marker, no gap.
	await settle(300)
	board.removeAttribute('data-kanban-landing')
	const count = ends.length
	await drag(card('europa'), end(4))
	await frames(4)
	check('no opt-in, no marker', [!!marker(), [...board.querySelectorAll('[data-kanban-card]')].some((c) => gaps(c) > 0)], [false, false])
	await answer()
	await until(() => laneOf('europa') === '4')
	check('and no landing events', ends.length, count)
	check('no errors', errors, [])
} catch (e) {
	rows.push({ step: 'script', error: String(e?.stack || e) })
}
await report()
`
