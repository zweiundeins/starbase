package app_test

import (
	"encoding/json"
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
func TestServerWinsContract(t *testing.T) {
	_, body := probe(t, "/", contractJS)
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

const contractJS = `
const cases = [
	{ tag: 'sb-input', attr: 'value', local: 'typed', server: '', want: '' },
	{ tag: 'sb-toggle', prop: 'checked', attr: 'checked', local: true, server: 'false', want: false },
	{ tag: 'sb-slider', attr: 'value', local: 70, server: '0', want: 0 },
	{ tag: 'sb-rating', attr: 'value', local: 3, server: '0', want: 0 },
	{ tag: 'sb-range', attr: 'value', local: { start: 10, end: 20 }, server: '{"start":0,"end":100}', want: { start: 0, end: 100 } },
	{ tag: 'sb-code-editor', attr: 'value', local: 'x = 1', server: '', want: '' },
	{ tag: 'sb-select', attrs: { options: '["A","B"]' }, attr: 'value', local: 'B', server: '', want: '' },
	{ tag: 'sb-radio-group', attrs: { options: '[{"value":"a","label":"A"},{"value":"b","label":"B"}]' }, attr: 'value', local: 'b', server: '', want: '' },
	{ tag: 'sb-tree', attrs: { items: '[{"id":"a","label":"A"},{"id":"b","label":"B"}]' }, attr: 'value', local: 'b', server: '', want: '' },
	{ tag: 'sb-dropdown', attrs: { type: 'radio', items: '[{"value":"a","label":"A"},{"value":"b","label":"B"}]' }, attr: 'value', local: 'b', server: '', want: '' },
	{ tag: 'sb-tabs', prop: 'selected', attr: 'selected', local: 2, server: '0', want: 0 },
	{ tag: 'sb-details', prop: 'open', attr: 'open', local: true, server: 'false', want: false },
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
await fetch('/__probe/result', { method: 'POST', body: JSON.stringify(rows) })
`
