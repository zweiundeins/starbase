package app_test

import (
	"strconv"
	"testing"

	"starbase/internal/tscheck"
)

// TestPlaygroundTypeScript switches the playground to TypeScript: the module
// is renamed and highlighted as TypeScript, it runs, its errors come at the
// lines written, and the type check reaches its editor.
func TestPlaygroundTypeScript(t *testing.T) {
	compiler := tscheck.New(t.TempDir()) != nil
	_, body := probe(t, "/playground", "const compiler = "+strconv.FormatBool(compiler)+"\n"+playgroundTSJS)
	copyRows(t, body)
}

const playgroundTSJS = `
const rows = []
const check = (step, got, want) => rows.push({ step, got: JSON.stringify(got), want: JSON.stringify(want) })
const settle = (ms = 80) => new Promise((r) => setTimeout(r, ms))
const until = async (fn, ms = 8000) => {
	for (const end = performance.now() + ms; performance.now() < end; await settle(50)) if (fn()) break
	return fn()
}
try {
	await customElements.whenDefined('sb-code-playground')
	const pg = document.querySelector('sb-code-playground')
	const root = await until(() => pg.shadowRoot?.querySelector('iframe') && pg.shadowRoot)
	const output = () => [...root.querySelectorAll('.console div div')].map((d) => d.textContent).join(' || ')
	const editor = () => root.querySelector('sb-code-editor')
	const box = () => [...root.querySelectorAll('.check')].find((l) => l.textContent.includes('TypeScript'))?.querySelector('input')
	check('the size line has its first numbers', /^[\d.]+ (B|kB)$/.test(document.querySelector('.pg-size strong').textContent), true)

	pg.files = { 'component.js': "const a = 1\n\nthrow new Error('js on line 3')\n" }
	check('JavaScript: a thrown error comes with its line', await until(() => output().includes('js on line 3') && output()), 'js on line 3 (line 3)')

	box().click()
	await until(() => root.querySelector('[role="tab"]').textContent.trim() === 'component.ts')
	check('the switch renames the module', [Object.keys(pg.files), editor().getAttribute('language'), box().checked], [['component.ts', 'index.html'], 'ts', true])

	pg.files = { 'component.ts': "interface Crew {\n\tname: string\n}\nconst crew: Crew = { name: 'Ada' }\nthrow new Error(crew.name + ' on line 5')\n" }
	check('TypeScript runs, its errors at the lines written', await until(() => output().includes('Ada on line 5') && output()), 'Ada on line 5 (line 5)')
	const keyword = await until(() => [...editor().shadowRoot.querySelectorAll('pre:not(.diag) .token.keyword')].find((t) => t.textContent === 'interface'))
	check('highlighted as TypeScript', !!keyword, true)

	if (compiler) {
		pg.files = { 'component.ts': "export const n: number = 'four'\n" }
		// The first check unpacks the compiler: seconds under -race on a slow runner.
		check('the type check reaches the editor', await until(() => editor().shadowRoot.querySelector('.diag mark')?.textContent, 30000), 'n')
	}

	box().click()
	await until(() => root.querySelector('[role="tab"]').textContent.trim() === 'component.js')
	check('and back to JavaScript', [Object.keys(pg.files), editor().getAttribute('language')], [['component.js', 'index.html'], 'js'])
} catch (e) {
	rows.push({ step: 'script', error: String(e?.stack || e) })
}
await fetch('/__probe/result', { method: 'POST', body: JSON.stringify(rows) })
`
