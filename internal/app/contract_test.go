package app_test

import (
	"encoding/json"
	"net/http"
	"testing"
)

// TestServerWinsContract checks the value contract (CLAUDE.md) on every value
// component, in the browser, running the bundle as pages do:
//
//   - the property returns the user's local value, not the attribute;
//   - the server can clear or reset a value it never set: writing the
//     attribute onto an element that had none wins, even when the decoded
//     value equals the prop's default (observeProps alone misses that);
//   - a removed attribute is ignored (morphs strip reflected attributes), and
//     writing the same value again after a removal counts as a new word.
//
// A boolean (checked, open, busy) is a word by presence, sent through
// Datastar's morph as a server frame would: adding it wins, removing it wins,
// identical markup keeps a local edit, and any value reads as present, "false"
// too, as in HTML. With STARBASE_DATASTAR_BUNDLE=<file>, on that Datastar build.
func TestServerWinsContract(t *testing.T) {
	body := probeBundle(t, "/", morphJS+contractJS, map[string]http.HandlerFunc{"/__test/morph": morphHandler})
	var rows []struct {
		Tag, Step, Got, Want, Error string
	}
	if err := json.Unmarshal(body, &rows); err != nil {
		t.Fatalf("%v: %s", err, body)
	}
	if len(rows) == 0 {
		t.Fatal("no rows")
	}
	for _, r := range rows {
		switch {
		case r.Error != "":
			t.Errorf("%s: %s", r.Tag, r.Error)
		case r.Got != r.Want:
			t.Errorf("%s, %s: value %s, want %s", r.Tag, r.Step, r.Got, r.Want)
		}
	}
}

// TestPendingAttribute: the page's pending attribute sets :state(pending)
// and aria-busy (on the ElementInternals) while present, on sb-button and on
// every value component, and ORs with confirm's own pending: removing it
// leaves an unconfirmed edit pending, and revert() leaves the page's word.
// With STARBASE_DATASTAR_BUNDLE=<file>, on that Datastar build.
func TestPendingAttribute(t *testing.T) {
	copyRows(t, probeBundle(t, "/", pendingJS, nil))
}

const pendingJS = `
const settle = () => new Promise((r) => setTimeout(r, 60))
const rows = []
const check = (step, got, want) => rows.push({ step, got: JSON.stringify(got), want: JSON.stringify(want) })
const ints = new WeakMap()
const attach = HTMLElement.prototype.attachInternals
HTMLElement.prototype.attachInternals = function () {
	const i = attach.call(this)
	ints.set(this, i)
	return i
}
const box = document.createElement('div')
document.body.append(box)
const said = (el) => [el.matches(':state(pending)'), ints.get(el)?.ariaBusy ?? null]
const cases = [
	{ tag: 'sb-button' },
	{ tag: 'sb-input', server: 'a', local: 'b' },
	{ tag: 'sb-select', attrs: { options: '["a","b"]' }, server: 'a', local: 'b' },
	{ tag: 'sb-slider', server: '40', local: 70 },
	{ tag: 'sb-toggle', attr: 'checked', prop: 'checked', server: '', local: false },
	{ tag: 'sb-checkbox', attr: 'checked', prop: 'checked', server: '', local: false },
	{ tag: 'sb-code-editor', server: 'a', local: 'b' },
	{ tag: 'sb-tabs', attrs: { labels: '["A","B","C"]' }, attr: 'selected', prop: 'selected', server: '0', local: 2 },
	{ tag: 'sb-rating', server: '2', local: 4 },
	{ tag: 'sb-tree', attrs: { items: '[{"id":"a","label":"A"},{"id":"b","label":"B"}]' }, server: 'a', local: 'b' },
	{ tag: 'sb-checkbox-group', attrs: { options: '["a","b"]' }, server: '["a"]', local: ['b'] },
	{ tag: 'sb-date-picker', server: '2026-09-01', local: '2026-09-29' },
	{ tag: 'sb-range', server: '{"start":0,"end":100}', local: { start: 10, end: 20 } },
	{ tag: 'sb-radio-group', attrs: { options: '["a","b"]' }, server: 'a', local: 'b' },
	{ tag: 'sb-dropdown', attrs: { type: 'radio', items: '[{"value":"a","label":"A"},{"value":"b","label":"B"}]' }, server: 'a', local: 'b' },
	{ tag: 'sb-data-table', attrs: { selection: 'multiple', columns: '[{"key":"name"}]', rows: '[{"id":"a","name":"A"},{"id":"b","name":"B"}]' }, attr: 'selected', prop: 'selected', server: '["a"]', local: ['b'] },
]
for (const c of cases) {
	let el
	const step = (s) => c.tag + ': ' + s
	try {
		await customElements.whenDefined(c.tag)
		el = document.createElement(c.tag)
		for (const [k, v] of Object.entries(c.attrs || {})) el.setAttribute(k, v)
		if ('server' in c) el.setAttribute(c.attr || 'value', c.server)
		box.append(el)
		await settle()
		check(step('idle'), said(el), [false, null])
		el.setAttribute('pending', '')
		await settle()
		check(step('pending: the state and aria-busy'), said(el), [true, 'true'])
		el.removeAttribute('pending')
		await settle()
		check(step('pending removed: both gone'), said(el), [false, null])
		if (!('server' in c)) continue
		el.setAttribute('confirm', '')
		el[c.prop || 'value'] = c.local
		await settle()
		check(step('confirm: an edit is pending, not busy'), said(el), [true, null])
		el.setAttribute('pending', '')
		await settle()
		check(step('confirm and pending'), said(el), [true, 'true'])
		el.removeAttribute('pending')
		await settle()
		check(step('pending removed: the edit is still pending'), said(el), [true, null])
		el.setAttribute('pending', '')
		await settle()
		el.revert()
		await settle()
		check(step('revert() ends the edit, not the page\'s pending'), said(el), [true, 'true'])
		el.removeAttribute('pending')
		await settle()
		check(step('then pending removed: idle'), said(el), [false, null])
	} catch (e) {
		rows.push({ step: c.tag, error: String(e?.stack || e) })
	}
	el?.remove()
}
// Driven by a signal, as a page does: data-attr:pending="$_saving".
try {
	box.innerHTML = '<div data-signals:_saving="false"><sb-button id="p-btn" data-attr:pending="$_saving">Save</sb-button><sb-toggle id="p-tog" checked data-attr:pending="$_saving"></sb-toggle><button id="p-flip" data-on:click="$_saving = !$_saving"></button></div>'
	await settle()
	const btn = document.getElementById('p-btn'), tog = document.getElementById('p-tog'), flip = document.getElementById('p-flip')
	tog.shadowRoot.querySelector('button').focus()
	flip.click()
	await settle()
	check('data-attr:pending: on', [said(btn), said(tog)], [[true, 'true'], [true, 'true']])
	flip.click()
	await settle()
	check('data-attr:pending: off', [said(btn), said(tog)], [[false, null], [false, null]])
	tog.shadowRoot.querySelector('button').focus()
	flip.click()
	await settle()
	check('pending re-renders nothing: the switch keeps the focus', tog.shadowRoot.activeElement === tog.shadowRoot.querySelector('button'), true)
} catch (e) {
	rows.push({ step: 'data-attr:pending', error: String(e?.stack || e) })
}
await fetch('/__probe/result', { method: 'POST', body: JSON.stringify(rows) })
`

// TestServerClosesDialogs: a server frame that drops open from an open sb-modal
// or sb-drawer (modal and not) closes its <dialog> for real: not open, no
// backdrop over the page, and the focus back on the element that had it
// before the server opened it. With STARBASE_DATASTAR_BUNDLE=<file>, on that
// Datastar build.
func TestServerClosesDialogs(t *testing.T) {
	copyRows(t, probeBundle(t, "/", pdPrelude+morphJS+closesDialogsJS, map[string]http.HandlerFunc{"/__test/morph": morphHandler}))
}

const closesDialogsJS = `
const box = document.createElement('div')
document.body.prepend(box)
const opener = document.createElement('button')
opener.textContent = 'Open card'
document.body.prepend(opener)
const at = (el) => {
	const r = el.getBoundingClientRect()
	return document.elementFromPoint(r.x + r.width / 2, r.y + r.height / 2)
}
for (const c of [
	{ tag: 'sb-modal', name: 'sb-modal', modal: true },
	{ tag: 'sb-drawer', name: 'sb-drawer', modal: true },
	{ tag: 'sb-drawer', name: 'sb-drawer modal="false"', attrs: ' modal="false"', modal: false },
]) {
	try {
		await customElements.whenDefined(c.tag)
		const html = (open) => '<div id="card-region"><' + c.tag + ' id="card"' + (c.attrs || '') + (open ? ' open' : '') + ' heading="Card"><button id="inside">Edit</button></' + c.tag + '></div>'
		box.innerHTML = html(false)
		await settle()
		opener.focus()
		await morph(html(true))
		let dlg = document.getElementById('card').shadowRoot.querySelector('dialog')
		await until(() => dlg.open)
		check(c.name + ': the server opens it', [dlg.open, dlg.matches(':modal')], [true, c.modal])
		if (c.modal) check(c.name + ': open, it covers the page', at(opener) === opener, false)
		document.getElementById('inside').focus()
		await morph(html(false))
		dlg = document.getElementById('card').shadowRoot.querySelector('dialog')
		await until(() => !dlg.open)
		check(c.name + ': the server removes open: the dialog closes', [dlg.open, dlg.matches(':modal'), document.getElementById('card').isOpen], [false, false, false])
		check(c.name + ': no backdrop over the page', at(opener) === opener, true)
		check(c.name + ': the focus is back on the opener', document.activeElement === opener, true)
	} catch (e) {
		rows.push({ step: c.name, error: String(e?.stack || e) })
	}
	box.replaceChildren()
}
check('no errors', errors, [])
await fetch('/__probe/result', { method: 'POST', body: JSON.stringify(rows) })
`

const contractJS = `
const cases = [
	{ tag: 'sb-input', attr: 'value', local: 'typed', server: '', want: '' },
	{ tag: 'sb-checkbox-group', attrs: { options: '["a","b"]' }, attr: 'value', local: ['b'], server: '[]', want: [] },
	{ tag: 'sb-slider', attr: 'value', local: 70, server: '0', want: 0 },
	{ tag: 'sb-rating', attr: 'value', local: 3, server: '0', want: 0 },
	{ tag: 'sb-range', attr: 'value', local: { start: 10, end: 20 }, server: '{"start":0,"end":100}', want: { start: 0, end: 100 } },
	{ tag: 'sb-date-picker', attr: 'value', local: '2026-09-29', server: '', want: '' },
	{ tag: 'sb-date-picker', attrs: { time: '' }, attr: 'value', local: '2026-09-29T14:30', server: '', want: '' },
	{ tag: 'sb-date-picker', attrs: { time: '', 'time-zone': 'Europe/Zurich' }, attr: 'value', local: '2026-09-29T14:30+02:00', server: '2026-09-29T12:00:00Z', want: '2026-09-29T14:00+02:00' },
	{ tag: 'sb-code-editor', attr: 'value', local: 'x = 1', server: '', want: '' },
	{ tag: 'sb-select', attrs: { options: '["A","B"]' }, attr: 'value', local: 'B', server: '', want: '' },
	{ tag: 'sb-radio-group', attrs: { options: '[{"value":"a","label":"A"},{"value":"b","label":"B"}]' }, attr: 'value', local: 'b', server: '', want: '' },
	{ tag: 'sb-tree', attrs: { items: '[{"id":"a","label":"A"},{"id":"b","label":"B"}]' }, attr: 'value', local: 'b', server: '', want: '' },
	{ tag: 'sb-data-table', attrs: { selection: 'multiple', columns: '[{"key":"name"}]', rows: '[{"id":"a","name":"A"},{"id":"b","name":"B"}]' }, prop: 'selected', attr: 'selected', local: ['b'], server: '[]', want: [] },
	{ tag: 'sb-data-table', attrs: { columns: '[{"key":"a"},{"key":"b"},{"key":"c"}]' }, prop: 'hiddenColumns', attr: 'hidden-columns', local: ['b'], server: '[]', want: [] },
	{ tag: 'sb-data-table', attrs: { resizable: '', columns: '[{"key":"a"},{"key":"b"}]' }, prop: 'widths', attr: 'widths', local: { a: 120 }, server: '{}', want: {} },
	{ tag: 'sb-dropdown', attrs: { type: 'radio', items: '[{"value":"a","label":"A"},{"value":"b","label":"B"}]' }, attr: 'value', local: 'b', server: '', want: '' },
	{ tag: 'sb-tabs', prop: 'selected', attr: 'selected', local: 2, server: '0', want: 0 },
]
const settle = () => new Promise((r) => setTimeout(r, 60))
const rows = []
const box = document.createElement('div')
document.body.append(box)
for (const c of cases) {
	const prop = c.prop || 'value'
	const row = (step, want) => rows.push({ tag: c.tag, step, got: JSON.stringify(el[prop]), want: JSON.stringify(want) })
	let el
	try {
		await customElements.whenDefined(c.tag)
		el = document.createElement(c.tag)
		for (const [k, v] of Object.entries(c.attrs || {})) el.setAttribute(k, v)
		box.append(el)
		await settle()
		el[prop] = c.local
		await settle()
		row('the property returns the local value', c.local)
		el.setAttribute(c.attr, c.server)
		await settle()
		row('the server writes the attribute onto an element without it', c.want)
		el[prop] = c.local
		await settle()
		el.removeAttribute(c.attr)
		await settle()
		row('a removed attribute is ignored', c.local)
		el.setAttribute(c.attr, c.server)
		await settle()
		row('the same word again after a removal counts', c.want)
	} catch (e) {
		rows.push({ tag: c.tag, error: String(e) })
	}
	el?.remove()
}
// rev: a new one makes the server's value win even when it is unchanged (a
// refusal), and ends pending; the same rev again changes nothing.
const revs = [
	{ tag: 'sb-input', server: 'STARBASE-1', local: 'no spaces', want: 'STARBASE-1' },
	{ tag: 'sb-select', attrs: { options: '["A","B"]' }, server: 'A', local: 'B', want: 'A' },
	{ tag: 'sb-select', name: 'sb-select multiple', attrs: { multiple: '', options: '["A","B"]' }, server: '["A"]', local: ['A', 'B'], want: ['A'] },
	{ tag: 'sb-slider', server: '40', local: 70, want: 40 },
	{ tag: 'sb-toggle', attr: 'checked', prop: 'checked', server: '', local: false, want: true },
	{ tag: 'sb-checkbox', attr: 'checked', prop: 'checked', server: '', local: false, want: true },
	{ tag: 'sb-code-editor', server: 'x = 1', local: 'x = 2', want: 'x = 1' },
	{ tag: 'sb-tabs', attrs: { labels: '["A","B","C"]' }, attr: 'selected', prop: 'selected', server: '0', local: 2, want: 0 },
	{ tag: 'sb-rating', server: '2', local: 4, want: 2 },
	{ tag: 'sb-tree', attrs: { items: '[{"id":"a","label":"A"},{"id":"b","label":"B"}]' }, server: 'a', local: 'b', want: 'a' },
	{ tag: 'sb-checkbox-group', attrs: { options: '["a","b"]' }, server: '["a"]', local: ['b'], want: ['a'] },
	{ tag: 'sb-date-picker', server: '2026-09-01', local: '2026-09-29', want: '2026-09-01' },
	{ tag: 'sb-range', server: '{"start":0,"end":100}', local: { start: 10, end: 20 }, want: { start: 0, end: 100 } },
	{ tag: 'sb-radio-group', attrs: { options: '["a","b"]' }, server: 'a', local: 'b', want: 'a' },
	{ tag: 'sb-radio-group', name: 'sb-radio-group segmented', attrs: { variant: 'segmented', options: '["a","b"]' }, server: 'a', local: 'b', want: 'a' },
	{ tag: 'sb-dropdown', attrs: { type: 'radio', items: '[{"value":"a","label":"A"},{"value":"b","label":"B"}]' }, server: 'a', local: 'b', want: 'a' },
	{ tag: 'sb-data-table', attrs: { selection: 'multiple', columns: '[{"key":"name"}]', rows: '[{"id":"a","name":"A"},{"id":"b","name":"B"}]' }, attr: 'selected', prop: 'selected', server: '["a"]', local: ['b'], want: ['a'] },
]
for (const c of revs) {
	const name = (c.name || c.tag) + ' rev'
	const prop = c.prop || 'value'
	const row = (step, want, pending) => rows.push({ tag: name, step, got: JSON.stringify([el[prop], el.matches(':state(pending)')]), want: JSON.stringify([want, pending]) })
	let el
	try {
		await customElements.whenDefined(c.tag)
		el = document.createElement(c.tag)
		for (const [k, v] of Object.entries({ ...c.attrs, confirm: '', [c.attr || 'value']: c.server, rev: '1' })) el.setAttribute(k, v)
		box.append(el)
		await settle()
		el[prop] = c.local
		await settle()
		row('a local edit is pending', c.local, true)
		el.setAttribute('rev', '2')
		await settle()
		row('a new rev with the value unchanged wins and ends pending', c.want, false)
		el[prop] = c.local
		await settle()
		el.setAttribute('rev', '2')
		await settle()
		row('the same rev again changes nothing', c.local, true)
		el.setAttribute('rev', '3')
		await settle()
		row('the next rev wins again', c.want, false)
	} catch (e) {
		rows.push({ tag: name, error: String(e?.stack || e) })
	}
	el?.remove()
}
// The case rev is for: the server refuses a pick (204, then a frame that
// morphs the same value with the next rev), and the chip goes again.
{
	const c = { tag: 'sb-select' }
	const html = (rev) => '<div id="rev-region"><sb-select id="rev-case" name="crew" multiple confirm options="[&quot;A&quot;,&quot;B&quot;]" value="[&quot;A&quot;]" rev="' + rev + '"></sb-select></div>'
	let el
	const row = (step, want, pending) => rows.push({ tag: 'sb-select rev, morphed', step, got: JSON.stringify([el.value, el.matches(':state(pending)')]), want: JSON.stringify([want, pending]) })
	try {
		await customElements.whenDefined(c.tag)
		box.innerHTML = html(7)
		el = document.getElementById('rev-case')
		await settle()
		el.value = ['A', 'B']
		await settle()
		row('a picked chip is pending', ['A', 'B'], true)
		await morph(html(7))
		row('identical markup keeps it', ['A', 'B'], true)
		await morph(html(8))
		row('the refusal: the same value with the next rev drops the chip', ['A'], false)
	} catch (e) {
		rows.push({ tag: 'sb-select rev, morphed', error: String(e?.stack || e) })
	}
	box.replaceChildren()
}
// Booleans: the attribute's presence is the server's word.
const dialogOf = (el) => el.shadowRoot.querySelector('dialog')
const shown = (el) => el.isOpen
const show = (el, v) => (v ? el.show() : el.close())
const bools = [
	{ tag: 'sb-toggle', attr: 'checked' },
	{ tag: 'sb-checkbox', attr: 'checked' },
	{ tag: 'sb-details', attr: 'open' },
	{ tag: 'sb-popover', attr: 'open', inner: '<button slot="trigger">Brief</button><p>Hi</p>' },
	{ tag: 'sb-dropdown', attr: 'open', attrs: { items: '["a","b"]' } },
	{ tag: 'sb-date-picker', attr: 'open' },
	{ tag: 'sb-modal', attr: 'open', get: shown, set: show, dialog: true },
	{ tag: 'sb-drawer', attr: 'open', get: shown, set: show, dialog: true },
	{ tag: 'sb-drawer', name: 'sb-drawer modal="false"', attr: 'open', attrs: { modal: 'false' }, get: shown, set: show, dialog: true },
]
const markup = (c, word) => {
	const a = { id: 'bool-case', ...c.attrs }
	if (word != null) a[c.attr] = word
	return '<' + c.tag + Object.entries(a).map(([k, v]) => ' ' + k + '="' + v.replaceAll('"', '&quot;') + '"').join('') + '>' + (c.inner || '') + '</' + c.tag + '>'
}
for (const c of bools) {
	const name = c.name || c.tag
	const get = c.get || ((el) => el[c.attr])
	const set = c.set || ((el, v) => (el[c.attr] = v))
	let el
	const row = (step, want) => rows.push({ tag: name, step, got: JSON.stringify(get(el)), want: JSON.stringify(want) })
	const local = async (v) => (set(el, v), await settle())
	const server = async (word) => (await morph(markup(c, word)), (el = document.getElementById('bool-case')))
	try {
		await customElements.whenDefined(c.tag)
		box.innerHTML = markup(c, null)
		el = box.firstElementChild
		await settle()
		await local(true)
		row('the property returns the local value', true)
		await server(null)
		row('identical markup (no attribute) keeps a local edit', true)
		await local(false)
		await server('')
		row('the server adds the attribute: it wins', true)
		await local(false)
		await server('')
		row('identical markup (the attribute) keeps a local edit', false)
		await local(true)
		await server(null)
		row('the server removes the attribute: it wins', false)
		if (c.dialog) {
			rows.push({ tag: name, step: 'removed: the dialog is closed', got: JSON.stringify(dialogOf(el).open), want: 'false' })
			rows.push({ tag: name, step: 'removed: no modal dialog (no backdrop)', got: JSON.stringify(dialogOf(el).matches(':modal')), want: 'false' })
		}
		await server('false')
		row('the server adds the attribute as "false": present, so true, as in HTML', true)
		await local(false)
		await server('')
		row('"false" to "" is no new word: a local edit stays', false)
	} catch (e) {
		rows.push({ tag: name, error: String(e?.stack || e) })
	}
	el?.remove()
	box.replaceChildren()
}
// sb-busy: no local edit, the indicator follows the server's busy word.
{
	const c = { tag: 'sb-busy', attr: 'busy' }
	let el
	const row = (step, want) => rows.push({ tag: c.tag, step, got: JSON.stringify(el.matches(':state(busy)')), want: JSON.stringify(want) })
	const server = async (word) => (await morph(markup(c, word)), (el = document.getElementById('bool-case')))
	try {
		await customElements.whenDefined(c.tag)
		box.innerHTML = markup(c, null)
		el = box.firstElementChild
		await settle()
		row('no attribute: idle', false)
		await server('')
		row('the server adds busy: busy', true)
		await server(null)
		row('the server removes busy: idle', false)
		await server('false')
		row('busy="false" is present: busy, as in HTML', true)
	} catch (e) {
		rows.push({ tag: c.tag, error: String(e?.stack || e) })
	}
	box.replaceChildren()
}
await fetch('/__probe/result', { method: 'POST', body: JSON.stringify(rows) })
`
