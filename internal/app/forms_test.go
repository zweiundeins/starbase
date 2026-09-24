package app_test

import (
	"encoding/json"
	"testing"
)

// TestFormParticipation checks that the form-like components take part in
// the <form> they sit in, like their native counterparts, although Rocket
// can't make them form-associated yet: new FormData(form) (which native
// submits and Datastar's contentType: 'form' use) holds their value under
// their name, the user's local value once it changed, nothing while
// disabled, and form.reset() brings back the server's value without firing
// change events — unless the page cancelled the reset — and a form nested in
// theirs by a script neither collects them nor resets them.
func TestFormParticipation(t *testing.T) {
	_, body := probe(t, "/", formsJS)
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
			t.Errorf("%s, %s: FormData %s, want %s", r.Tag, r.Step, r.Got, r.Want)
		}
	}
}

const formsJS = `
const cases = [
	{ tag: 'sb-input', attrs: { value: 'server' }, local: 'mine', server: [['f', 'server']], mine: [['f', 'mine']] },
	{ tag: 'sb-code-editor', attrs: { value: 'a = 1' }, local: 'b = 2', server: [['f', 'a = 1']], mine: [['f', 'b = 2']] },
	{ tag: 'sb-select', attrs: { options: '["A","B","C"]', value: 'A' }, local: 'B', server: [['f', 'A']], mine: [['f', 'B']] },
	{ tag: 'sb-select', attrs: { multiple: '', options: '["A","B","C"]', value: '["A","C"]' }, local: [], server: [['f', 'A'], ['f', 'C']], mine: [] },
	{ tag: 'sb-radio-group', attrs: { options: '[{"value":"a","label":"A"},{"value":"b","label":"B"}]', value: 'a' }, local: 'b', server: [['f', 'a']], mine: [['f', 'b']] },
	{ tag: 'sb-radio-group', attrs: { options: '[{"value":"a","label":"A"},{"value":"b","label":"B"}]' }, local: 'b', server: [], mine: [['f', 'b']] },
	{ tag: 'sb-toggle', prop: 'checked', attrs: { checked: '' }, local: false, server: [['f', 'on']], mine: [] },
	{ tag: 'sb-slider', attrs: { value: '30' }, local: 70, server: [['f', '30']], mine: [['f', '70']] },
	{ tag: 'sb-rating', attrs: { value: '2' }, local: 4, server: [['f', '2']], mine: [['f', '4']] },
	{ tag: 'sb-range', attrs: { value: '{"start":20,"end":60}' }, local: { start: 10, end: 30 }, server: [['f', '{"start":20,"end":60}']], mine: [['f', '{"start":10,"end":30}']] },
]
const settle = () => new Promise((r) => setTimeout(r, 60))
const rows = []
for (const c of cases) {
	const prop = c.prop || 'value'
	const form = document.createElement('form')
	document.body.append(form)
	const entries = () => JSON.stringify([...new FormData(form)])
	const row = (step, want) => rows.push({ tag: c.tag, step, got: entries(), want: JSON.stringify(want) })
	try {
		await customElements.whenDefined(c.tag)
		const el = document.createElement(c.tag)
		el.setAttribute('name', 'f')
		for (const [k, v] of Object.entries(c.attrs)) el.setAttribute(k, v)
		form.append(el)
		await settle()
		row('the server value is submitted', c.server)
		el[prop] = c.local
		await settle()
		row('the local value is submitted', c.mine)
		let events = 0
		for (const e of ['input', 'change', 'sb-change']) el.addEventListener(e, () => events++)
		form.reset()
		await settle()
		row('a reset brings back the server value', c.server)
		rows.push({ tag: c.tag, step: 'a reset fires no change events', got: String(events), want: '0' })
		// A reset the page cancels, with a listener added after the component.
		el[prop] = c.local
		await settle()
		const cancel = (e) => e.preventDefault()
		form.addEventListener('reset', cancel)
		form.reset()
		await settle()
		row('a cancelled reset keeps the local value', c.mine)
		form.removeEventListener('reset', cancel)
		// A form nested in this one by a script (the parser never nests forms).
		const inner = document.createElement('form')
		form.append(inner)
		rows.push({ tag: c.tag, step: 'a nested form does not collect it', got: JSON.stringify([...new FormData(inner)]), want: '[]' })
		inner.reset()
		await settle()
		row('a reset of the nested form leaves it', c.mine)
		inner.remove()
		form.reset()
		await settle()
		el.setAttribute('disabled', '')
		await settle()
		row('nothing is submitted while disabled', [])
		el.removeAttribute('name')
		el.removeAttribute('disabled')
		await settle()
		row('nothing is submitted without a name', [])
	} catch (e) {
		rows.push({ tag: c.tag, error: String(e) })
	}
	form.remove()
}
await fetch('/__probe/result', { method: 'POST', body: JSON.stringify(rows) })
`
