import { rocket, startPeeking, stopPeeking } from 'datastar'
import Prism from './vendor/prism.js'

// Host getters must not subscribe callers (e.g. data-bind's sync effect) to
// the internal signal, or that effect writes the stale bound value back.
const peek = (fn) => {
	startPeeking()
	try {
		return fn()
	} finally {
		stopPeeking()
	}
}

// Initial code can come from a child <script type="text/plain">: its text
// is never parsed as HTML and never executed. Common indentation is removed.
const dedent = (text) => {
	const lines = text.replace(/^\s*\n|\n\s*$/g, '').split('\n')
	const indent = Math.min(...lines.filter((l) => l.trim()).map((l) => l.match(/^[ \t]*/)[0].length))
	return lines.map((l) => l.slice(Number.isFinite(indent) ? indent : 0)).join('\n')
}

// Highlighting by Prism (vendored with the js-templates plugin), so the
// HTML and CSS inside Rocket's html`…` and /* css */ `…` templates light up.
const GRAMMARS = { js: 'javascript', html: 'markup', css: 'css' }
const highlight = (src, lang) => {
	const name = GRAMMARS[lang] || 'javascript'
	// A trailing newline keeps the last (empty) line's height in the <pre>.
	return Prism.highlight(src, Prism.languages[name], name) + '\n'
}

const styles = /* css */ `
:host {
	--_bg: var(--sb-surface-inset, #0B1224);
	--_border: var(--sb-border, #283552);
	--_text: var(--sb-text-1, #F3F4FA);
	--_muted: var(--sb-text-muted, #7785A8);
	--_brand: var(--sb-brand-light, #B09AFF);
	--_sel: var(--sb-selection, rgb(140 107 255 / 0.4));
	--_radius: var(--sb-radius, 8px);
	--_font: var(--sb-font-ui, "JetBrains Mono", ui-monospace, monospace);
	display: block;
	inline-size: 100%;
}
.label { display: block; margin-block-end: 0.4rem; color: var(--sb-text-2, #AEBBDD); font-size: 0.8125rem; font-weight: 600; }
.scroller {
	overflow: auto;
	min-block-size: var(--sb-code-editor-min-height, 0);
	max-block-size: var(--sb-code-editor-height, 28rem);
	border: 1px solid var(--_border);
	border-radius: var(--_radius);
	background: var(--_bg);
	scrollbar-width: thin;
}
.scroller:focus-within { border-color: var(--_brand); }
/* Gutter | code. The code column is at least the viewport and grows with
   the longest line, so the textarea and <pre> always line up. */
.grid { display: grid; grid-template-columns: auto minmax(calc(100% - 3rem), max-content); min-block-size: 100%; }
.no-gutter .grid { grid-template-columns: minmax(100%, max-content); }
.no-gutter .gutter { display: none; }
.gutter {
	position: sticky;
	inset-inline-start: 0;
	z-index: 1;
	padding: 0.75rem 0.5rem 0.75rem 0.75rem;
	border-inline-end: 1px solid color-mix(in oklch, var(--_border) 60%, transparent);
	background: var(--_bg);
	color: var(--_muted);
	text-align: end;
	user-select: none;
}
.code { display: grid; }
.code > * { grid-area: 1 / 1; }
pre, textarea, .gutter {
	margin: 0;
	font-family: var(--_font);
	font-size: 0.8125rem;
	line-height: 1.6;
	white-space: pre;
	tab-size: inherit;
	font-variant-ligatures: none;
}
pre, textarea { padding: 0.75rem 1rem; border: 0; }
pre { color: var(--_text); pointer-events: none; }
textarea {
	box-sizing: border-box;
	inline-size: 100%;
	block-size: 100%;
	resize: none;
	overflow: hidden;
	outline: none;
	background: transparent;
	color: transparent;
	caret-color: var(--_text);
	-webkit-text-fill-color: transparent;
}
textarea::selection { background: var(--_sel); -webkit-text-fill-color: transparent; }
:host([readonly]) textarea { caret-color: transparent; }
/* Prism tokens, coloured from theme tokens. */
.token.comment, .token.prolog, .token.doctype, .token.cdata { color: var(--_muted); font-style: italic; }
.token.string, .token.char, .token.attr-value, .token.template-punctuation, .token.url { color: var(--sb-code-string, #6EF59A); }
.token.number, .token.boolean, .token.constant, .token.unit, .token.hexcode { color: var(--sb-code-number, #F5C451); }
.token.keyword, .token.atrule, .token.important, .token.rule { color: var(--sb-code-keyword, #B09AFF); }
.token.function, .token.class-name, .token.attr-name, .token.property { color: var(--sb-code-function, #CBBEFF); }
.token.tag, .token.selector, .token.builtin { color: var(--sb-code-tag, #65BFFF); }
.token.punctuation, .token.operator, .token.interpolation-punctuation { color: var(--_muted); }
/* Code inside attributes and templates keeps the base text colour. */
.token.attr-value .token.punctuation.attr-equals, .token.attr-value > .token.punctuation:first-child { color: var(--_muted); }
.token.embedded-code, .token.script, .token.style, .token.interpolation, .token.value.javascript { color: var(--_text); }
`

rocket('sb-code-editor', {
	props: ({ bool, number, oneOf, string }) => ({
		language: oneOf('js', 'html', 'css').default('js').docs({ description: 'Syntax to highlight.' }),
		value: string.docs({ description: 'The code (or a child <script type="text/plain">). A new value from the server replaces it; the live code is the value property.' }),
		lineNumbers: bool.default(true).docs({ description: 'Show a line-number gutter.' }),
		tabSize: number.clamp(1, 8).default(2).docs({ description: 'Visual width of a tab.' }),
		readonly: bool.docs({ description: 'Make the code read-only.' }),
		label: string.trim.docs({ description: 'Visible label; also the accessible name.' }),
	}),
	manifest: {
		events: [
			{ name: 'input', kind: 'event', bubbles: true, composed: true, description: 'On every edit (native, re-targeted to the host).' },
			{ name: 'change', kind: 'event', bubbles: true, composed: true, description: 'When the field loses focus after edits.' },
			{ name: 'sb-run', kind: 'custom-event', bubbles: true, composed: true, description: 'Ctrl/Cmd+Enter. detail: { value }.' },
		],
	},
	renderOnPropChange: ({ changes }) => 'label' in changes,
	setup: ({ $$, action, adoptStyles, emit, host, observeProps, overrideProp, props }) => {
		adoptStyles(host, styles)
		const child = host.querySelector(':scope > script[type="text/plain"]')
		const initial = host.hasAttribute('value') || !child ? props.value : dedent(child.textContent)

		// Local signals the markup renders from.
		$$.code = initial
		$$.lang = props.language
		$$.gutter = props.lineNumbers
		$$.tab = props.tabSize
		$$.readonly = props.readonly
		// Rocket clears local signals when the element is removed, and computeds
		// may run once more: treat missing code as empty.
		$$.html = () => highlight($$.code ?? '', $$.lang)
		$$.numbers = () => Array.from({ length: ($$.code ?? '').split('\n').length }, (_, i) => i + 1).join('\n')

		observeProps(() => {
			$$.lang = props.language
			$$.gutter = props.lineNumbers
			$$.tab = props.tabSize
			$$.readonly = props.readonly
		}, 'language', 'lineNumbers', 'tabSize', 'readonly')

		// Typing updates $$code; the textarea's data-effect only writes back
		// external changes (the values differ), so the caret never jumps.
		// A value attribute sent by the server wins when it changes (a morph
		// with a new value); re-sending the same markup changes nothing, so edits
		// survive re-renders. A *removed* attribute changes nothing either: morphs
		// also remove attributes that were only reflected (e.g. from a data-bind
		// write before the upgrade). To clear it, the server sends value="".
		observeProps(() => peek(() => host.hasAttribute('value') && ($$.code = props.value)), 'value')
		overrideProp('value', () => peek(() => $$.code), (v) => peek(() => ($$.code = String(v ?? ''))))

		const insert = (area, text) => {
			// execCommand keeps the browser's undo stack; setRangeText is the fallback.
			if (!document.execCommand?.('insertText', false, text)) {
				area.setRangeText(text, area.selectionStart, area.selectionEnd, 'end')
				area.dispatchEvent(new Event('input', { bubbles: true, composed: true }))
			}
		}
		let escaped = false
		action('change', () => emit('change'))
		action('key', ({ el: area, evt: e }) => {
			if (e.key === 'Escape') {
				escaped = true // the next Tab moves focus instead of indenting
				return
			}
			if ((e.metaKey || e.ctrlKey) && e.key === 'Enter') {
				e.preventDefault()
				emit('sb-run', { value: area.value })
				return
			}
			if (area.readOnly) return
			if (e.key === 'Tab' && !escaped) {
				e.preventDefault()
				const { selectionStart: s, selectionEnd: t, value: v } = area
				const lineStart = v.lastIndexOf('\n', s - 1) + 1
				if (!e.shiftKey && s === t) return insert(area, '\t')
				// Block (de)indent of every selected line.
				const block = v.slice(lineStart, t)
				const next = e.shiftKey ? block.replace(/^(\t| {1,2})/gm, '') : block.replace(/^/gm, '\t')
				area.setSelectionRange(lineStart, t)
				insert(area, next)
				area.setSelectionRange(lineStart, lineStart + next.length)
				return
			}
			escaped = false
			if (e.key === 'Enter' && !e.shiftKey && !e.altKey) {
				e.preventDefault()
				const v = area.value
				const lineStart = v.lastIndexOf('\n', area.selectionStart - 1) + 1
				const line = v.slice(lineStart, area.selectionStart)
				const indent = line.match(/^[ \t]*/)[0]
				insert(area, '\n' + indent + (/[{([]\s*$/.test(line) ? '\t' : ''))
			}
		})
	},
	render: ({ html, props: { label } }) => html`
		${label ? html`<span class="label" part="label">${label}</span>` : null}
		<div class="scroller" part="editor" data-class:no-gutter="!$$gutter" data-style:tab-size="$$tab">
			<div class="grid">
				<pre class="gutter" aria-hidden="true" data-text="$$numbers"></pre>
				<div class="code">
					<pre aria-hidden="true"><code data-effect="el.innerHTML = $$html"></code></pre>
					<textarea
						part="textarea"
						spellcheck="false"
						autocapitalize="off"
						autocomplete="off"
						autocorrect="off"
						wrap="off"
						rows="1"
						aria-label="${label || 'Code'}"
						data-effect="el.value !== $$code && (el.value = $$code)"
						data-attr:readonly="$$readonly"
						data-on:input="$$code = el.value"
						data-on:change="@change()"
						data-on:keydown="@key()"
					></textarea>
				</div>
			</div>
		</div>
	`,
})
