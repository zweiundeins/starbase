package app_test

import (
	"fmt"
	"maps"
	"net/http"
	"os"
	"sync/atomic"
	"testing"
)

// TestSelectMultiple checks sb-select's compact closed state (summary, max-chips, each alone and
// together) and its list actions in Chrome: one sb-change with the whole value, the list stays
// open, the keyboard, pending, revert(), server-wins and forms. With
// STARBASE_DATASTAR_BUNDLE=<file>, on that Datastar build (see TestVirtualScrollFocus).
func TestSelectMultiple(t *testing.T) {
	copyRows(t, probeBundle(t, "/", selectMultipleJS, nil))
}

// probeBundle is probeWith with these test handlers, on the Datastar build at
// STARBASE_DATASTAR_BUNDLE when that is set (see TestVirtualScrollFocus).
func probeBundle(t *testing.T, path, script string, extra map[string]http.HandlerFunc) []byte {
	t.Helper()
	var bundle []byte
	if p := os.Getenv("STARBASE_DATASTAR_BUNDLE"); p != "" {
		var err error
		if bundle, err = os.ReadFile(p); err != nil {
			t.Fatal(err)
		}
	}
	var served atomic.Int32
	handlers := func(app http.Handler) map[string]http.HandlerFunc {
		m := maps.Clone(extra)
		if m == nil {
			m = map[string]http.HandlerFunc{}
		}
		if bundle != nil {
			m[datastarPath(t, app)] = func(w http.ResponseWriter, r *http.Request) {
				served.Add(1)
				w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
				w.Write(bundle)
			}
		}
		return m
	}
	_, body := probeWith(t, path, script, handlers)
	if bundle != nil && served.Load() == 0 {
		t.Error("the page did not load STARBASE_DATASTAR_BUNDLE")
	}
	return body
}

// morphHandler answers a Datastar request with ?html= as an element patch: a server morph.
func morphHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	fmt.Fprintf(w, "event: datastar-patch-elements\ndata: elements %s\n\n", r.URL.Query().Get("html"))
}

// morphJS: morph(html) patches html through Datastar, as a server frame would.
const morphJS = `
const morpher = document.createElement('button')
morpher.hidden = true
morpher.setAttribute('data-on:click', "@get(el.dataset.url)")
document.body.append(morpher)
const morph = async (html) => {
	const done = new Promise((r) => document.addEventListener('datastar-fetch', function f(e) {
		if (e.detail.el === morpher && e.detail.type === 'finished') document.removeEventListener('datastar-fetch', f), r()
	}))
	await settle()
	morpher.dataset.url = '/__test/morph?html=' + encodeURIComponent(html)
	morpher.click()
	await done
	await settle()
}
`

const selectMultipleJS = `
const settle = () => new Promise((r) => setTimeout(r, 60))
const rows = []
const check = (step, got, want) => rows.push({ step, got: JSON.stringify(got), want: JSON.stringify(want) })
const group = async (name, fn) => {
	try {
		await fn()
	} catch (e) {
		rows.push({ step: name, error: String(e?.stack || e) })
	}
}
const box = document.createElement('div')
box.style.inlineSize = '20rem'
document.body.prepend(box)
await customElements.whenDefined('sb-select')
const OPTS = '["Alpha","Beta","Gamma","Delta","Epsilon"]'
const make = async (attrs, into = box) => {
	const el = document.createElement('sb-select')
	for (const [k, v] of Object.entries({ multiple: '', options: OPTS, ...attrs })) v === null || el.setAttribute(k, v)
	into.append(el)
	await settle()
	return el
}
const $ = (el, s) => el.shadowRoot.querySelector(s)
const $$ = (el, s) => [...el.shadowRoot.querySelectorAll(s)]
const seen = (n) => !!n && n.getClientRects().length > 0
// What the closed control shows: the chips' labels, the "+K" chip, the summary.
const closed = (el) => ({
	chips: $$(el, '.chip:not(.more)').filter(seen).map((c) => c.firstElementChild.textContent),
	more: seen($(el, '.more')) ? $(el, '.more').textContent : '',
	summary: seen($(el, '.sum')) ? $(el, '.sum').textContent : '',
})
const input = (el) => $(el, 'input')
const key = (el, key) => input(el).dispatchEvent(new KeyboardEvent('keydown', { key, bubbles: true, composed: true, cancelable: true }))
const press = async (el, ...keys) => {
	for (const k of keys) {
		key(el, k)
		await settle()
	}
}
const typeIn = async (el, s) => {
	input(el).value = s
	input(el).dispatchEvent(new InputEvent('input', { bubbles: true, composed: true }))
	await settle()
}
const events = (el) => {
	const log = []
	el.addEventListener('change', () => log.push('change'))
	el.addEventListener('sb-change', (e) => log.push(e.detail))
	return log
}
const open = (el) => input(el).getAttribute('aria-expanded') === 'true' && $(el, '[popover]').matches(':popover-open')
const rowsOf = (el) => $$(el, '[role=option]').map((r) => r.textContent.trim())
const actions = (el) => $$(el, '.act').map((r) => r.textContent + (r.getAttribute('aria-disabled') === 'true' ? ' (disabled)' : ''))
const activeId = (el) => input(el).getAttribute('aria-activedescendant')
const height = (el) => Math.round($(el, '.control').getBoundingClientRect().height)

await group('summary alone', async () => {
	const el = await make({ summary: '{count} of {total} ({more} more)', placeholder: 'All', value: '["Alpha","Gamma"]' })
	check('summary alone: the text replaces the chips', closed(el), { chips: [], more: '', summary: '2 of 5 (2 more)' })
	el.setAttribute('value', '[]')
	await settle()
	check('summary alone, nothing picked: the placeholder', [closed(el), input(el).placeholder], [{ chips: [], more: '', summary: '' }, 'All'])
	el.setAttribute('total', '84')
	el.setAttribute('value', '["Beta"]')
	await settle()
	check('summary: total sets {total}', closed(el).summary, '1 of 84 (1 more)')
	const log = events(el)
	await press(el, 'Backspace')
	check('summary alone: Backspace removes nothing it does not show', [el.value, log], [['Beta'], []])
	el.remove()
})

await group('max-chips alone', async () => {
	const el = await make({ 'max-chips': '2', value: '["Alpha","Beta","Gamma","Delta"]' })
	check('max-chips alone: N chips, then +K', closed(el), { chips: ['Alpha', 'Beta'], more: '+2', summary: '' })
	check('the +K chip names what it hides, and is no button', [$(el, '.more').title, $(el, '.more button')], ['Gamma, Delta', null])
	el.setAttribute('value', '["Alpha","Beta"]')
	await settle()
	check('max-chips alone, within N: the chips only', closed(el), { chips: ['Alpha', 'Beta'], more: '', summary: '' })
	el.setAttribute('max-chips', '0')
	await settle()
	check('max-chips="0" alone: only a count', closed(el), { chips: [], more: '+2', summary: '' })
	el.remove()
})

await group('summary and max-chips together', async () => {
	const el = await make({ 'max-chips': '2', summary: '+{more} more', value: '["Alpha","Beta","Gamma"]' })
	check('both, beyond N: N chips, then the summary in place of +K', closed(el), { chips: ['Alpha', 'Beta'], more: '', summary: '+1 more' })
	el.setAttribute('value', '["Alpha","Beta"]')
	await settle()
	check('both, within N: the chips only', closed(el), { chips: ['Alpha', 'Beta'], more: '', summary: '' })
	el.setAttribute('max-chips', '0')
	el.setAttribute('summary', '{count} of {total}')
	await settle()
	check('both, max-chips="0": the summary only', closed(el), { chips: [], more: '', summary: '2 of 5' })
	el.removeAttribute('summary')
	el.removeAttribute('max-chips')
	await settle()
	check('neither: every chip', closed(el), { chips: ['Alpha', 'Beta'], more: '', summary: '' })
	el.remove()
})

await group('one line, the full selection for screen readers, keyboard removal', async () => {
	const many = '["Alpha","Beta","Gamma","Delta","Epsilon"]'
	const one = await make({ 'max-chips': '2', value: '["Alpha"]' })
	const lots = await make({ 'max-chips': '2', value: many })
	const sum = await make({ summary: '{count} picked', value: many })
	const wraps = await make({ value: many })
	check('max-chips and summary keep the control one line high', [height(lots), height(sum)], [height(one), height(one)])
	check('without them, chips wrap', height(wraps) > height(one), true)
	const desc = (el) => el.shadowRoot.getElementById(input(el).getAttribute('aria-describedby'))?.textContent
	check('aria-describedby holds the whole selection', [desc(lots), desc(sum)], ['Alpha, Beta, Gamma, Delta, Epsilon', 'Alpha, Beta, Gamma, Delta, Epsilon'])
	const log = events(lots)
	await press(lots, 'Backspace')
	check('Backspace removes the last chip shown, not a hidden pick', [lots.value, closed(lots), log.length], [['Alpha', 'Gamma', 'Delta', 'Epsilon'], { chips: ['Alpha', 'Gamma'], more: '+2', summary: '' }, 2])
	$$(lots, '.chip:not(.more) button')[0].click()
	await settle()
	check('a shown chip\'s remove button', [lots.value, closed(lots).chips], [['Gamma', 'Delta', 'Epsilon'], ['Gamma', 'Delta']])
	$(lots, '.more').click()
	await settle()
	check('a click on +K opens the list and removes nothing', [open(lots), lots.value.length], [true, 3])
	await press(lots, 'Escape')
	await press(wraps, 'Backspace')
	check('without max-chips, Backspace removes the last pick', wraps.value, ['Alpha', 'Beta', 'Gamma', 'Delta'])
	for (const el of [one, lots, sum, wraps]) el.remove()
})

await group('actions with the keyboard', async () => {
	const el = await make({ actions: '', searchable: '', name: 'src', options: '["Alpha","Beta","Gamma","Delta",{"value":"Eta","disabled":true}]', value: '["Beta"]' })
	const log = events(el)
	input(el).focus()
	await press(el, 'ArrowDown')
	check('the actions come first, then the options', [actions(el), rowsOf(el).slice(0, 3)], [['Select all', 'Clear'], ['Select all', 'Clear', 'Alpha']])
	check('opening highlights the picked option, not an action', activeId(el), 'o1')
	await press(el, 'Home')
	check('Home reaches Select all', activeId(el), 'a-all')
	await press(el, 'Enter')
	check('Select all: one change, one sb-change with the whole value', log, ['change', { name: 'src', value: ['Beta', 'Alpha', 'Gamma', 'Delta'] }])
	check('Select all skips disabled options, and the list stays open', [el.value, open(el), actions(el)], [['Beta', 'Alpha', 'Gamma', 'Delta'], true, ['Select all (disabled)', 'Clear']])
	log.length = 0
	await press(el, 'Enter')
	check('a disabled action does nothing', log, [])
	await press(el, 'ArrowDown', 'Enter')
	check('Clear: one sb-change with [], the list stays open', [log, open(el), activeId(el)], [['change', { name: 'src', value: [] }], true, 'a-clear'])
	log.length = 0
	await typeIn(el, 'ta')
	check('while a query filters: Select the N matches', [actions(el), activeId(el)], [['Select the 2 matches', 'Clear (disabled)'], 'o0'])
	await press(el, 'Home', 'Enter')
	check('Select the matches adds what the query shows', [log, el.value, open(el), input(el).value], [['change', { name: 'src', value: ['Beta', 'Delta'] }], ['Beta', 'Delta'], true, 'ta'])
	await typeIn(el, 'gam')
	check('one match', actions(el)[0], 'Select the match')
	await press(el, 'Enter')
	check('options still pick after the actions', el.value, ['Beta', 'Delta', 'Gamma'])
	await typeIn(el, 'zzz')
	check('no matches: only Clear', [actions(el), activeId(el)], [['Clear'], null])
	el.remove()

	const plain = await make({ actions: '', value: '[]' })
	input(plain).focus()
	await press(plain, 'ArrowDown')
	check('nothing picked: the first option, Clear disabled', [activeId(plain), actions(plain)], ['o0', ['Select all', 'Clear (disabled)']])
	await press(plain, 'd')
	check('type-ahead lands on options, past the actions', activeId(plain), 'o3')
	await press(plain, 'Enter')
	check('type-ahead then Enter picks', plain.value, ['Delta'])
	plain.remove()

	const single = await make({ actions: '', multiple: null })
	input(single).focus()
	await press(single, 'ArrowDown')
	check('no actions without multiple', [actions(single), activeId(single)], [[], 'o0'])
	single.remove()
})

await group('actions with the pointer, and labels', async () => {
	const el = await make({ actions: '', searchable: '', 'select-all-label': 'Alle', 'clear-label': 'Leeren', 'matches-label': 'Den Treffer|Die {count} Treffer', clearable: '', value: '["Alpha"]' })
	const log = events(el)
	input(el).focus()
	await press(el, 'ArrowDown')
	check('labels can be replaced', [actions(el), $(el, '.clear').getAttribute('aria-label')], [['Alle', 'Leeren'], 'Leeren'])
	await typeIn(el, 'a')
	check('the matches label, many', actions(el)[0], 'Die 4 Treffer')
	await typeIn(el, 'gam')
	check('the matches label, one', actions(el)[0], 'Den Treffer')
	$$(el, '.act')[0].dispatchEvent(new MouseEvent('mousedown', { bubbles: true, cancelable: true, button: 0 }))
	await settle()
	check('a click on an action: one sb-change, the list stays open', [log, open(el)], [['change', { name: '', value: ['Alpha', 'Gamma'] }], true])
	el.remove()
})

await group('actions: pending, revert(), server-wins, forms', async () => {
	const form = document.createElement('form')
	box.append(form)
	const el = await make({ actions: '', confirm: '', name: 'src', value: '["Beta"]' }, form)
	const pending = () => el.matches(':state(pending)')
	input(el).focus()
	await press(el, 'ArrowDown', 'Home', 'Enter')
	check('Select all is pending until the server confirms', [el.value, pending()], [['Beta', 'Alpha', 'Gamma', 'Delta', 'Epsilon'], true])
	check('a form gets every picked value', [...new FormData(form)].map(([, v]) => v), ['Beta', 'Alpha', 'Gamma', 'Delta', 'Epsilon'])
	el.revert()
	await settle()
	check('revert() returns to the server\'s value', [el.value, pending(), open(el)], [['Beta'], false, true])
	await press(el, 'ArrowDown', 'Enter')
	check('Clear is pending', [el.value, pending()], [[], true])
	el.setAttribute('value', '["Gamma","Delta"]')
	await settle()
	check('a new value from the server wins after an action', [el.value, pending(), actions(el)], [['Gamma', 'Delta'], false, ['Select all', 'Clear']])
	await press(el, 'Home', 'Enter')
	el.setAttribute('value', '["Alpha","Beta","Gamma","Delta","Epsilon"]')
	await settle()
	check('the server confirms Select all', pending(), false)
	form.reset()
	await settle()
	check('a reset restores the server\'s value', el.value, ['Alpha', 'Beta', 'Gamma', 'Delta', 'Epsilon'])
	form.remove()
})

await group('remote total', async () => {
	const el = await make({ remote: '', 'min-chars': '0', summary: '{count} of {total}', options: '[]', results: '["Alpha","Beta","Gamma"]', value: '["Alpha"]' })
	check('remote: {total} counts the options it was offered', closed(el).summary, '1 of 3')
	el.setAttribute('results', '["Delta"]')
	await settle()
	check('remote: offers add up', closed(el).summary, '1 of 4')
	el.remove()
})

await fetch('/__probe/result', { method: 'POST', body: JSON.stringify(rows) })
`

// TestSelectFollowsNotch checks that sb-select's pixel details follow --sb-notch: at 1 the stepped
// arrow and the stepped pixel spinner, at 0 a plain triangle and a ring that turns smoothly.
func TestSelectFollowsNotch(t *testing.T) {
	_, body := probe(t, "/components/select", pdPrelude+selectNotchJS)
	copyRows(t, body)
}

const selectNotchJS = `
try {
	await customElements.whenDefined('sb-select')
	const root = document.querySelector('sb-select').shadowRoot
	const look = () => {
		const cs = (q, p) => getComputedStyle(root.querySelector(q))[p]
		const arrow = getComputedStyle(root.querySelector('[part=arrow]')).clipPath
		return [cs('.spin', 'opacity'), cs('.spin', 'animationTimingFunction'), cs('.ring', 'opacity'), arrow.includes('6.667px')]
	}
	document.documentElement.style.setProperty('--sb-notch', '1')
	check('notch 1: the pixel spinner, stepped, and the stepped arrow', look(), ['1', 'steps(4)', '0', false])
	document.documentElement.style.setProperty('--sb-notch', '0')
	check('notch 0: the ring, and the arrow a triangle', look(), ['0', 'steps(1000)', '1', true])
	check('no errors', errors, [])
} catch (e) {
	rows.push({ step: 'script', error: String(e?.stack || e) })
}
await report()
`

// TestSelectOpenState checks sb-select's :state(open) in Chrome: it follows the list whichever way
// it opens (a click, the keys, typing) or closes (Escape, a pick, Tab, blur, disabled, a server
// morph, removal), survives a morph that keeps it open, and starts closed after a re-attach. And
// the arrow part: the default flip, a page's ::part(arrow) rules over it, and aria-hidden.
func TestSelectOpenState(t *testing.T) {
	copyRows(t, probeBundle(t, "/", pdPrelude+morphJS+selectOpenJS, map[string]http.HandlerFunc{"/__test/morph": morphHandler}))
}

const selectOpenJS = `
const box = document.createElement('div')
box.style.inlineSize = '20rem'
document.body.prepend(box)
const outside = document.createElement('button')
outside.textContent = 'outside'
document.body.prepend(outside)
const style = document.createElement('style')
style.textContent = '.chev::part(arrow) { rotate: 45deg; clip-path: none; background: none; border: solid currentColor; border-width: 0 2px 2px 0; } .chev:state(open)::part(arrow) { rotate: 225deg; } .tint::part(arrow) { color: rgb(255, 0, 0); }'
document.head.append(style)
await customElements.whenDefined('sb-select')
const OPTS = '["Alpha","Beta","Gamma"]'
let n = 0
const make = async (attrs = {}) => {
	const el = document.createElement('sb-select')
	el.id = 'so' + ++n
	for (const [k, v] of Object.entries({ options: OPTS, ...attrs })) el.setAttribute(k, v)
	box.append(el)
	await settle()
	return el
}
const $ = (el, s) => el.shadowRoot.querySelector(s)
const input = (el) => $(el, 'input')
const key = async (el, k) => (press(input(el), k), await settle())
const state = (el) => [el.matches(':state(open)'), $(el, '[popover]').matches(':popover-open')]
const group = async (name, fn) => {
	try {
		await fn()
	} catch (e) {
		rows.push({ step: name, error: String(e?.stack || e) })
	}
}
const OPEN = [true, true], SHUT = [false, false]

await group('single', async () => {
	const el = await make()
	check('closed at first', state(el), SHUT)
	$(el, '.control').click()
	await settle()
	check('a click opens', state(el), OPEN)
	await key(el, 'Escape')
	check('Escape closes', state(el), SHUT)
	await key(el, 'ArrowDown')
	check('ArrowDown opens', state(el), OPEN)
	await key(el, 'Enter')
	check('Enter picks and closes', [state(el), el.value], [SHUT, 'Alpha'])
	await key(el, 'g')
	check('type-ahead opens', state(el), OPEN)
	$(el, '[role=option]').dispatchEvent(new MouseEvent('mousedown', { bubbles: true, cancelable: true, button: 0 }))
	await settle()
	check('a click on an option picks and closes', [state(el), el.value], [SHUT, 'Alpha'])
	input(el).focus()
	await key(el, 'ArrowDown')
	input(el).blur()
	await settle()
	check('blur closes', state(el), SHUT)
	input(el).focus()
	await key(el, 'ArrowDown')
	outside.focus()
	await settle()
	check('the focus moving out closes', state(el), SHUT)
	await key(el, 'ArrowDown')
	await key(el, 'Tab')
	check('Tab closes', state(el), SHUT)
	await key(el, 'ArrowDown')
	el.setAttribute('disabled', '')
	await settle()
	check('disabled closes', state(el), SHUT)
	el.remove()
})

await group('searchable and multiple', async () => {
	const el = await make({ searchable: '', multiple: '' })
	input(el).focus()
	input(el).value = 'be'
	input(el).dispatchEvent(new InputEvent('input', { bubbles: true, composed: true }))
	await settle()
	check('typing opens', state(el), OPEN)
	await key(el, 'Enter')
	check('a pick with multiple keeps it open', [state(el), el.value], [OPEN, ['Beta']])
	await key(el, 'Escape')
	check('Escape closes', state(el), SHUT)
	el.remove()
})

await group('morphs', async () => {
	const el = await make({ multiple: '' })
	await key(el, 'ArrowDown')
	await morph('<sb-select id="' + el.id + '" multiple options=\'["Alpha","Beta","Gamma","Delta"]\'></sb-select>')
	check('a morph that keeps it enabled leaves it open', [state(el), [...el.shadowRoot.querySelectorAll('[role=option]')].length], [OPEN, 4])
	await morph('<sb-select id="' + el.id + '" multiple disabled options=\'["Alpha"]\'></sb-select>')
	check('a morph that disables it closes it', state(el), SHUT)
	el.remove()
})

await group('removal and re-attach', async () => {
	const el = await make()
	await key(el, 'ArrowDown')
	el.remove()
	await settle()
	check('removed while open: no longer open', el.matches(':state(open)'), false)
	box.append(el)
	await settle()
	check('re-attached: closed', state(el), SHUT)
	$(el, '.control').click()
	await settle()
	check('re-attached: a click opens', state(el), OPEN)
	await key(el, 'Escape')
	check('re-attached: Escape closes', state(el), SHUT)
	el.remove()
})

await group('the arrow part', async () => {
	const plain = await make()
	const chev = await make({ class: 'chev' })
	const tint = await make({ class: 'tint' })
	const arrow = (el) => $(el, '[part=arrow]')
	const look = (el) => ((cs) => ({ rotate: cs.rotate, clip: cs.clipPath === 'none' ? 'none' : 'polygon', border: cs.borderRightWidth, bg: cs.backgroundColor }))(getComputedStyle(arrow(el)))
	check('aria-hidden', [plain, chev].map((el) => arrow(el).getAttribute('aria-hidden')), ['true', 'true'])
	check('default, closed: the stepped arrow, no rotation', [look(plain).rotate, look(plain).clip, look(plain).border], ['none', 'polygon', '0px'])
	check('default: filled with its color', look(plain).bg, getComputedStyle(arrow(plain)).color)
	for (const el of [plain, chev, tint]) await key(el, 'ArrowDown')
	check('default, open: turned over', look(plain).rotate, '180deg')
	check('a page rule replaces the look and the rotation, closed and open', look(chev), { rotate: '225deg', clip: 'none', border: '2px', bg: 'rgba(0, 0, 0, 0)' })
	check('a page rule without rotate keeps the flip', [look(tint).rotate, look(tint).bg], ['180deg', 'rgb(255, 0, 0)'])
	for (const el of [plain, chev, tint]) await key(el, 'Escape')
	check('closed again', [look(plain).rotate, look(chev).rotate, look(tint).rotate], ['none', '45deg', 'none'])
	for (const el of [plain, chev, tint]) el.remove()
})

check('no errors', errors, [])
await report()
`
