package app_test

import (
	"encoding/json"
	"net/http"
	"os"
	"sync/atomic"
	"testing"
	"time"
)

// TestInlineEditRename renames the nicknames of sb-inline-edit's demo: a
// double press, Enter or F2 asks for the field, the server saves a valid
// name and refuses an empty or a long one with the text kept, and Escape
// closes the field. A refusal that crosses the user's typing keeps the field
// and the new text, a field the answer removes sends nothing, a failed
// request says so, and the focus moves only after a key. STARBASE_DATASTAR_BUNDLE=<file> runs it on another Datastar build,
// such as the official release (see TestVirtualScrollFocus).
func TestInlineEditRename(t *testing.T) {
	var bundle []byte
	if p := os.Getenv("STARBASE_DATASTAR_BUNDLE"); p != "" {
		var err error
		if bundle, err = os.ReadFile(p); err != nil {
			t.Fatal(err)
		}
	}
	var served atomic.Int32
	handlers := func(app http.Handler) map[string]http.HandlerFunc {
		m := map[string]http.HandlerFunc{"/demo/arrange/inline-edit": func(w http.ResponseWriter, r *http.Request) {
			var p struct{ Move struct{ Type, Value string } }
			json.Unmarshal([]byte(r.URL.Query().Get("datastar")), &p)
			switch {
			case p.Move.Type == "commit" && p.Move.Value == "":
				time.Sleep(800 * time.Millisecond) // the user types on while the refusal is on its way
			case p.Move.Type == "cancel":
				time.Sleep(500 * time.Millisecond) // time to click back into a field that is about to close
			case p.Move.Value == "Fail":
				http.Error(w, "down", http.StatusInternalServerError)
				return
			}
			app.ServeHTTP(w, r)
		}}
		if bundle != nil {
			m[datastarPath(t, app)] = func(w http.ResponseWriter, r *http.Request) {
				served.Add(1)
				w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
				w.Write(bundle)
			}
		}
		return m
	}
	_, body := probeWith(t, "/components/inline-edit", pdPrelude+inlineEditJS, handlers)
	if bundle != nil && served.Load() == 0 {
		t.Error("the page did not load STARBASE_DATASTAR_BUNDLE")
	}
	copyRows(t, body)
}

const inlineEditJS = `
const events = []
for (const type of ['sb-inline-edit-request', 'sb-inline-edit-commit', 'sb-inline-edit-cancel'])
	document.addEventListener(type, (e) => events.push({ type: type.slice(15), ...e.detail }))
// A real press on the name also moves the focus to it.
const doublePress = (el) => {
	el.focus()
	for (let i = 0; i < 2; i++) pointer(el, 'pointerdown', center(el))
}
try {
	await customElements.whenDefined('sb-inline-edit')
	const earth = document.getElementById('nick-earth')
	const field = (host) => host.querySelector('[data-inline-edit-input]')
	const title = (host) => host.querySelector('[data-inline-edit-trigger]')
	const name = (host) => title(host)?.textContent
	const problem = (host) => host.querySelector('.demo-edit__problem')
	const failed = () => earth.querySelector('.demo-edit__failed').textContent
	const focused = (el) => !!el && document.activeElement === el
	check('the server rendered the nickname', [earth.dataset.state, name(earth)], ['body=earth name=Blue+Marble', 'Blue Marble'])

	// Two presses of the main button: the request, and the server's answer, a field that takes the focus.
	doublePress(title(earth))
	await until(() => field(earth))
	check('a double press asks for the field', [events.splice(0), field(earth)?.value, focused(field(earth)), earth.getAttribute('aria-busy')], [[{ type: 'request', contextId: 'earth' }], 'Blue Marble', true, 'false'])

	// Enter commits; the server saves, and since a key ended the edit, the name takes the focus back.
	field(earth).value = 'Pale Blue Dot'
	press(field(earth), 'Enter')
	await until(() => name(earth) === 'Pale Blue Dot')
	check('Enter saves', [events.splice(0), earth.dataset.state, focused(title(earth))], [[{ type: 'commit', contextId: 'earth', value: 'Pale Blue Dot' }], 'body=earth name=Pale+Blue+Dot focus=1', true])

	// The keyboard's way in: F2 and Enter on the name, but not a held Enter's repeats.
	// Escape closes the field and the name takes the focus.
	press(title(earth), 'Enter', { repeat: true })
	await settle(200)
	check('a repeated Enter on the name asks nothing', [events.splice(0), !!field(earth)], [[], false])
	press(title(earth), 'F2')
	await until(() => field(earth))
	check('F2 opens the field', [events.splice(0), focused(field(earth))], [[{ type: 'request', contextId: 'earth' }], true])
	press(field(earth), 'Escape')
	await until(() => !field(earth))
	check('Escape cancels', [events.splice(0), earth.dataset.state, focused(title(earth))], [[{ type: 'cancel', contextId: 'earth' }], 'body=earth name=Pale+Blue+Dot focus=1', true])
	press(title(earth), 'Enter')
	await until(() => field(earth))
	check('Enter opens the field', [events.splice(0), focused(field(earth))], [[{ type: 'request', contextId: 'earth' }], true])
	// A held Enter: its repeats don't close the field it opened.
	press(field(earth), 'Enter', { repeat: true })
	await settle(200)
	check('a repeated Enter does nothing', [events.splice(0), focused(field(earth))], [[], true])

	// An empty nickname by Enter: refused, with the reason, the text kept and the focus back in the field.
	const input = field(earth)
	input.value = '   '
	press(input, 'Enter')
	await until(() => problem(earth))
	const first = problem(earth)
	check('an empty nickname is refused', [first.id, first.textContent, field(earth) === input, input.value, input.getAttribute('aria-describedby'), focused(input)], ['nick-earth-problem-1', 'A nickname needs at least one character.', true, '   ', 'nick-earth-problem-1', true])
	// The same reason again comes in a new alert, so it is announced again.
	press(input, 'Enter')
	await until(() => problem(earth)?.id === 'nick-earth-problem-2')
	check('a repeated refusal', [problem(earth) !== first, first.isConnected, problem(earth).textContent, focused(input)], [true, false, 'A nickname needs at least one character.', true])

	// A refusal after a blur (a click on the page) leaves the focus where it went.
	input.value = 'x'.repeat(41)
	input.blur()
	await until(() => problem(earth)?.id === 'nick-earth-problem-3')
	await settle(150)
	check('a refused blur keeps the focus away', [problem(earth).textContent, document.activeElement === document.body, input.value], ['A nickname has at most 40 characters.', true, 'x'.repeat(41)])

	// Enter on an empty field, then the user clicks back in and types before the refusal is back:
	// it is pending meanwhile, and the refusal keeps the field and the new text without sending anything.
	input.focus()
	events.splice(0)
	input.value = ''
	press(input, 'Enter')
	await until(() => earth.getAttribute('aria-busy') === 'true')
	input.focus()
	input.value = 'Pale Blue'
	check('a commit is pending', [events.splice(0), earth.getAttribute('aria-busy')], [[{ type: 'commit', contextId: 'earth', value: '' }], 'true'])
	await until(() => problem(earth)?.id === 'nick-earth-problem-4')
	await settle(150)
	check('the refusal keeps what was typed since', [field(earth) === input, input.value, focused(input), events.splice(0), earth.getAttribute('aria-busy')], [true, 'Pale Blue', true, [], 'false'])
	press(input, 'Enter')
	await until(() => name(earth) === 'Pale Blue')
	await settle(150)
	check('then Enter saves it', [events.splice(0), earth.dataset.state, focused(title(earth))], [[{ type: 'commit', contextId: 'earth', value: 'Pale Blue' }], 'body=earth name=Pale+Blue focus=1', true])

	// A request that fails: the field stays as it was, and a message says so until the next request.
	press(title(earth), 'Enter')
	await until(() => field(earth))
	field(earth).value = 'Fail'
	press(field(earth), 'Enter')
	await until(() => failed())
	await settle(100)
	check('a failed request says so', [failed(), field(earth)?.value, earth.getAttribute('aria-busy'), title(earth)], ['Nothing changed: the request failed. Try again.', 'Fail', 'false', null])
	field(earth).focus()
	field(earth).value = 'Pale Blue'
	press(field(earth), 'Enter')
	await until(() => title(earth))
	check('the next answer clears it', [events.splice(0).map((e) => e.type), failed(), name(earth)], [['request', 'commit', 'cancel'], '', 'Pale Blue'])

	// Escape, then a click back into the field and some typing before the answer: the answer removes
	// the focused field, which sends nothing.
	press(title(earth), 'Enter')
	await until(() => field(earth))
	const closing = field(earth)
	press(closing, 'Escape')
	closing.focus()
	closing.value = 'Stray'
	await until(() => !field(earth))
	await settle(200)
	check('a field the answer removes sends nothing', [events.splice(0).map((e) => e.type), name(earth), failed()], [['request', 'cancel'], 'Pale Blue', ''])

	// A commit on one title and a request on another cross: neither is lost.
	const mars = document.getElementById('nick-mars'), jupiter = document.getElementById('nick-jupiter')
	doublePress(title(mars))
	await until(() => field(mars))
	field(mars).value = 'Rust Bucket'
	doublePress(title(jupiter)) // Mars's field loses the focus: a commit
	await until(() => name(mars) === 'Rust Bucket' && field(jupiter))
	check('both answers arrive', [name(mars), !!field(jupiter), focused(field(jupiter))], ['Rust Bucket', true, true])
	check('no errors', errors, [])
} catch (e) {
	rows.push({ step: 'script', error: String(e?.stack || e) })
}
await report()
`
