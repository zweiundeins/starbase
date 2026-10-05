package app_test

import (
	"strconv"
	"testing"

	"starbase/internal/tscheck"
)

// TestCodeEditorDiagnostics checks sb-code-editor's underlines and problem
// line, and the playground's type check from edit to underline.
func TestCodeEditorDiagnostics(t *testing.T) {
	compiler := tscheck.New(t.TempDir()) != nil
	_, body := probe(t, "/playground", "const compiler = "+strconv.FormatBool(compiler)+"\n"+diagnosticsJS)
	copyRows(t, body)
}

const diagnosticsJS = `
const rows = []
const check = (step, got, want) => rows.push({ step, got: JSON.stringify(got), want: JSON.stringify(want) })
const settle = (ms = 80) => new Promise((r) => setTimeout(r, ms))
const until = async (fn, ms = 8000) => {
	for (const end = performance.now() + ms; performance.now() < end; await settle(50)) if (fn()) break
	return fn()
}
// The box of word in the text under el (the highlighted code), by a Range.
const boxOf = (el, word) => {
	const walk = document.createTreeWalker(el, NodeFilter.SHOW_TEXT)
	for (let n; (n = walk.nextNode()); ) {
		const i = n.data.indexOf(word)
		if (i < 0) continue
		const r = document.createRange()
		r.setStart(n, i)
		r.setEnd(n, i + word.length)
		return r.getBoundingClientRect()
	}
}
const key = (area, k, shiftKey = false) => area.dispatchEvent(new KeyboardEvent('keydown', { key: k, shiftKey, bubbles: true, cancelable: true }))
try {
	await customElements.whenDefined('sb-code-editor')
	const el = document.createElement('sb-code-editor')
	el.setAttribute('value', 'const a = 1\nprops.lable.x\nfoo()')
	el.setAttribute('diagnostics', JSON.stringify([
		{ line: 2, col: 7, length: 5, text: 'lable', message: "Property 'lable' does not exist. Did you mean 'label'?" },
		{ line: 3, col: 1, length: 3, text: 'foo', message: "Cannot find name 'foo'." },
		{ line: 9, col: 1, length: 1, message: 'Not in the code.' },
	]))
	document.body.append(el)
	await settle()
	const root = el.shadowRoot
	const area = root.querySelector('textarea')
	const status = root.querySelector('[part="problem"]')
	const marks = () => [...root.querySelectorAll('.diag mark')].map((m) => m.textContent)
	const selected = () => area.value.slice(area.selectionStart, area.selectionEnd)

	check('marks', marks(), ['lable', 'foo'])
	check('the problem line shows the first', status.textContent, "1/2 · Line 2: Property 'lable' does not exist. Did you mean 'label'?")
	check('the problem line describes the textarea', area.getAttribute('aria-describedby') === status.id && status.checkVisibility(), true)
	const word = boxOf(root.querySelector('pre:not(.diag):not(.gutter)'), 'lable')
	const mark = root.querySelector('.diag mark').getBoundingClientRect()
	check('the underline sits under its word', [mark.left - word.left, mark.top - word.top, mark.width - word.width].map((d) => Math.abs(d) < 0.5), [true, true, true])

	const foo = area.value.indexOf('foo')
	area.focus()
	area.setSelectionRange(foo + 1, foo + 1)
	area.dispatchEvent(new KeyboardEvent('keyup', { key: 'ArrowRight', bubbles: true }))
	await settle()
	check('the problem line follows the caret', status.textContent, "2/2 · Line 3: Cannot find name 'foo'.")

	area.setSelectionRange(0, 0)
	key(area, 'F8')
	check('F8 selects the next problem', selected(), 'lable')
	key(area, 'F8')
	check('F8 again', selected(), 'foo')
	key(area, 'F8')
	check('F8 wraps around', selected(), 'lable')
	key(area, 'F8', true)
	check('Shift+F8 goes back, wrapping', selected(), 'foo')
	area.setSelectionRange(0, 0)
	status.click()
	check('a click on the problem line selects the next', selected(), 'lable')

	// Problems about an older version fade as the code changes.
	document.execCommand('insertText', false, 'label')
	await settle()
	check('a changed word loses its mark', marks(), ['foo'])
	check('the problem line', status.textContent, "Line 3: Cannot find name 'foo'.")
	area.setSelectionRange(0, 0)
	document.execCommand('insertText', false, '\n')
	await settle()
	check('moved code loses its marks', [marks(), status.checkVisibility()], [[], false])
	el.setAttribute('diagnostics', JSON.stringify([{ line: 4, col: 1, length: 3, text: 'foo', message: 'Again.' }]))
	await settle()
	check('a new list replaces the old', [marks(), status.textContent], [['foo'], 'Line 4: Again.'])
	el.remove()

	// The playground: an edit goes to the server's type check, and its answer to the editor.
	if (compiler) {
		const pg = document.querySelector('sb-code-playground')
		pg.files = { 'component.js': "// @ts-check\nimport { rocket } from 'datastar'\nrocket('sb-x', { props: ({ string }) => ({ label: string }), setup: ({ props }) => props.lable })\n" }
		const ed = pg.shadowRoot.querySelector('sb-code-editor[data-file="component.js"]')
		const got = await until(() => [...ed.shadowRoot.querySelectorAll('.diag mark')].map((m) => m.textContent).join())
		check('the playground underlines what the type check found', got, 'lable')
		check('with its message', ed.shadowRoot.querySelector('[part="problem"]').textContent.includes("Did you mean 'label'?"), true)
	}
} catch (e) {
	rows.push({ step: 'script', error: String(e?.stack || e) })
}
await fetch('/__probe/result', { method: 'POST', body: JSON.stringify(rows) })
`
