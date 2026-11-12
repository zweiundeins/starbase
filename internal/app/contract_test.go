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
// rev: a new one makes the server's value win even when it is unchanged.
for (const c of [{ tag: 'sb-input', server: 'STARBASE-1', local: 'no spaces', want: 'STARBASE-1' }]) {
	const row = (step, want) => rows.push({ tag: c.tag + ' rev', step, got: JSON.stringify(el.value), want: JSON.stringify(want) })
	let el
	try {
		await customElements.whenDefined(c.tag)
		el = document.createElement(c.tag)
		for (const [k, v] of Object.entries({ confirm: '', value: c.server, rev: '1' })) el.setAttribute(k, v)
		box.append(el)
		await settle()
		el.value = c.local
		await settle()
		row('a local edit', c.local)
		el.setAttribute('rev', '2')
		await settle()
		row('a new rev with the value unchanged wins', c.want)
	} catch (e) {
		rows.push({ tag: c.tag + ' rev', error: String(e) })
	}
	el?.remove()
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
