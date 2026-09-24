import { rocket, startPeeking, stopPeeking } from 'datastar'

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

// One ElementInternals per element: attachInternals() works once, and setup
// runs again when the element is re-attached. Its custom states
// (:state(pending)) are styleable from the page and morph-proof.
const internals = new WeakMap()
const internalsOf = (host) => internals.get(host) ?? internals.set(host, host.attachInternals()).get(host)
// The live code outlives a re-attach (setup runs again), like a moved <textarea>'s.
const live = new WeakMap()

// Initial code can come from a child <script type="text/plain">: its text
// is never parsed as HTML and never executed. Common indentation is removed.
const dedent = (text) => {
	const lines = text.replace(/^\s*\n|\n\s*$/g, '').split('\n')
	const indent = Math.min(...lines.filter((l) => l.trim()).map((l) => l.match(/^[ \t]*/)[0].length))
	return lines.map((l) => l.slice(Number.isFinite(indent) ? indent : 0)).join('\n')
}

// Highlighting by Prism (vendored with the js-templates plugin), so the
// HTML and CSS inside Rocket's html`…` and /* css */ `…` templates light up.
// It loads with the first editor, so pages without one never download it;
// until it arrives, the code shows uncoloured. js, html and css are Prism's own
// names (aliases). A trailing newline keeps the last (empty) line's height.
let Prism
const highlight = (src, lang) => (Prism ? Prism.highlight(src, Prism.languages[lang], lang) : src.replace(/&/g, '&amp;').replace(/</g, '&lt;')) + '\n'

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
	--_code-size: var(--sb-code-editor-font-size, 0.8125rem);
	display: block;
	inline-size: 100%;
}
:host([hidden]) { display: none; }
.label { display: block; margin-block-end: 0.4rem; color: var(--sb-text-2, #AEBBDD); font-size: 0.8125rem; font-weight: 600; }
/* A grid, so its one child fills --sb-code-editor-min-height. */
.scroller {
	display: grid;
	overflow: auto;
	min-block-size: var(--sb-code-editor-min-height, 0);
	max-block-size: var(--sb-code-editor-height, 28rem);
	border: 1px solid var(--_border);
	border-radius: var(--_radius);
	background: var(--_bg);
	scrollbar-width: thin;
}
.scroller:focus-within { border-color: var(--_brand); }
/* Disabled: dimmed like the other controls, but it still scrolls. */
.label:has(+ * :disabled), .scroller:has(:disabled) { opacity: 0.5; }
/* Gutter | code. The grid is at least the viewport and grows with the
   longest line, so the textarea and <pre> always line up. The gutter is at
   least 3rem whatever the lines' length, so the code never shifts sideways
   (and the textarea has cols="1": its default 20 columns set a minimum width). */
.grid { display: grid; grid-template-columns: minmax(3rem, auto) 1fr; inline-size: max-content; min-inline-size: 100%; }
.no-gutter .grid { grid-template-columns: 1fr; }
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
/* The gutter is a <pre> too. */
pre, textarea {
	margin: 0;
	padding: 0.75rem 1rem;
	border: 0;
	font-family: var(--_font);
	/* One size for the textarea, the highlighted <pre> and the gutter: the caret and the colours must line up. */
	font-size: var(--_code-size);
	line-height: 1.6;
	white-space: pre;
	font-variant-ligatures: none;
}
/* The browser gives <code> its own monospace font: a second font on every
   line makes each line box taller, and the highlighting drifts away from the
   caret and the line numbers. It takes the <pre>'s font, size and line height. */
pre code { font: inherit; }
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
/* Prism tokens, coloured from theme tokens. */
.token.comment, .token.prolog, .token.doctype, .token.cdata { color: var(--_muted); font-style: italic; }
.token.string, .token.attr-value, .token.url { color: var(--sb-code-string, #6EF59A); }
.token.number, .token.boolean, .token.constant { color: var(--sb-code-number, #F5C451); }
.token.keyword, .token.atrule, .token.important, .token.rule { color: var(--sb-code-keyword, #B09AFF); }
.token.function, .token.class-name, .token.attr-name, .token.property { color: var(--sb-code-function, #CBBEFF); }
.token.tag, .token.selector { color: var(--sb-code-tag, #65BFFF); }
.token.punctuation, .token.operator { color: var(--_muted); }
/* Code inside attributes and templates keeps the base text colour. */
.token.embedded-code, .token.script, .token.style, .token.interpolation, .token.value.javascript { color: var(--_text); }
`

rocket('sb-code-editor', {
	props: ({ bool, number, oneOf, string }) => ({
		language: oneOf('js', 'html', 'css').default('js').docs({ description: 'Syntax to highlight.' }),
		value: string.docs({ description: 'The code (or a child <script type="text/plain">). A new value from the server replaces it; the live code is the value property.' }),
		lineNumbers: bool.default(true).docs({ description: 'Show a line-number gutter.' }),
		tabSize: number.clamp(1, 8).default(2).docs({ description: 'Visual width of a tab.' }),
		readonly: bool.docs({ description: 'Make the code read-only (a form still submits it).' }),
		disabled: bool.docs({ description: 'Disable editing; a form doesn\'t submit it, like a disabled <textarea>.' }),
		label: string.trim.docs({ description: 'Visible label; also the accessible name.' }),
		confirm: bool.docs({ description: 'Server-confirmed value: :state(pending) while the local value differs from the server\'s value attribute (see revert()).' }),
		name: string.trim.docs({ description: 'Name reported in sb-change and submitted with the form it sits in (e.g. the field of a command).' }),
	}),
	manifest: {
		events: [
			{ name: 'input', kind: 'event', bubbles: true, composed: true, description: 'On every edit (native, re-targeted to the host).' },
			{ name: 'change', kind: 'event', bubbles: true, composed: true, description: 'When the field loses focus after edits.' },
			{ name: 'sb-change', kind: 'custom-event', bubbles: true, composed: true, description: 'Same moment. detail: { name, value }: ready for a command.' },
			{ name: 'sb-run', kind: 'custom-event', bubbles: true, composed: true, description: 'Ctrl/Cmd+Enter. detail: { value }.' },
		],
	},
	renderOnPropChange: ({ changes }) => 'label' in changes,
	setup: ({ $$, action, adoptStyles, cleanup, defineHostProp, effect, emit, host, observeProps, overrideProp, props, render }) => {
		adoptStyles(host, styles)
		const child = host.querySelector(':scope > script[type="text/plain"]')
		const script = child && dedent(child.textContent)
		// The server's value: the value attribute, else the child script's code.
		const server = () => (host.hasAttribute('value') || !child ? props.value : script)

		// Local signals the markup renders from.
		$$.code = live.get(host) ?? server()
		const mirror = () => Object.assign($$, { lang: props.language, gutter: props.lineNumbers, tab: props.tabSize, readonly: props.readonly, disabled: props.disabled })
		mirror()
		observeProps(mirror, 'language', 'lineNumbers', 'tabSize', 'readonly', 'disabled')
		Prism || import('./vendor/prism.js').then((m) => ((Prism = m.default), host.isConnected && ($$.ready = 1)))
		// Rocket clears local signals when the element is removed, and computeds
		// may run once more: treat missing code as empty.
		$$.html = () => ($$.ready, highlight($$.code ?? '', $$.lang || 'js'))
		$$.numbers = () => ($$.code ?? '').split('\n').map((_, i) => i + 1).join('\n')

		// Typing updates $$code; the textarea's data-effect only writes back
		// external changes (the values differ), so the caret never jumps.
		// A value attribute sent by the server wins when it differs from the last
		// one (a morph with a new value); re-sending the same markup changes
		// nothing, so edits survive re-renders. A *removed* attribute is ignored:
		// morphs also remove attributes that were only reflected (e.g. from a
		// data-bind write before the upgrade). To clear it, the server sends
		// value="". Not observeProps: it only fires when the decoded value
		// changes, and value="" on an editor without the attribute is "" again.
		// The host's aria-label names the textarea (without a label), so a new
		// one renders again.
		let served = host.hasAttribute('value') ? props.value : null
		const watch = new MutationObserver((records) => peek(() => {
			if (records.some((r) => r.attributeName !== 'value')) render({})
			if (!host.hasAttribute('value')) return void (served = null)
			if (props.value !== served) $$.code = served = props.value
			sync()
		}))
		watch.observe(host, { attributeFilter: ['value', 'aria-label'] })
		cleanup(() => watch.disconnect())
		overrideProp('value', () => peek(() => $$.code), (v) => peek(() => ($$.code = String(v ?? ''))))
		// Commands: server() is the server's value, $$.code the local one.
		// With confirm, :state(pending) marks an edit the server hasn't confirmed
		// yet; revert() returns to the server's value (e.g. a rejected command).
		const states = internalsOf(host).states
		const sync = () => peek(() => states[props.confirm && $$.code !== server() ? 'add' : 'delete']('pending'))
		// Every change of the code: keep it for a re-attach, and sync.
		effect(() => ($$.code != null && live.set(host, $$.code), sync()))
		observeProps(sync)
		defineHostProp('revert', { value: () => peek(() => (($$.code = server()), sync())) })
		// Forms: until Rocket can make this element form-associated, join the
		// submissions and resets of the form it sits in, like a <textarea>.
		// `formdata` also fires for new FormData(form), so Datastar's
		// contentType: 'form' posts include it. A reset brings back the server's
		// value without events (revert()). setup reruns on a re-attach, so a
		// move into another form follows.
		const form = host.closest('form')
		const onData = (evt) => peek(() => props.name && !props.disabled && evt.formData.append(props.name, $$.code))
		const onReset = () => host.revert()
		form?.addEventListener('formdata', onData)
		form?.addEventListener('reset', onReset)
		cleanup(() => (form?.removeEventListener('formdata', onData), form?.removeEventListener('reset', onReset)))
		// The host isn't focusable: focus() goes to the textarea.
		defineHostProp('focus', { value: (o) => host.shadowRoot.querySelector('textarea').focus(o) })

		const insert = (area, text) => {
			// execCommand keeps the browser's undo stack; setRangeText is the fallback.
			if (!document.execCommand?.('insertText', false, text)) {
				area.setRangeText(text, area.selectionStart, area.selectionEnd, 'end')
				area.dispatchEvent(new Event('input', { bubbles: true, composed: true }))
			}
		}
		let escaped = false
		action('change', () => (emit('change'), emit('sb-change', { name: props.name, value: $$.code })))
		action('blur', () => (escaped = false))
		action('key', ({ el: area, evt: e }) => {
			// Enter and Tab also confirm IME input (Safari: keyCode 229 after it).
			if (e.isComposing || e.keyCode === 229) return
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
			const { selectionStart: s, value: v } = area
			// Where the caret's line starts. (At 0 the line starts at 0:
			// lastIndexOf('\n', -1) would still find a newline at v[0].)
			const a = s && v.lastIndexOf('\n', s - 1) + 1
			if (e.key === 'Tab' && !escaped) {
				e.preventDefault()
				let t = area.selectionEnd
				if (!e.shiftKey && s === t) return insert(area, '\t')
				// (De)indent the selected lines, whole, but not a line the selection
				// only reaches at its column 0. Shift+Tab alone outdents the caret's
				// line and keeps the caret where it was in the text.
				if (t > s && v[t - 1] === '\n') t--
				const b = (v.indexOf('\n', t) + 1 || v.length + 1) - 1
				const block = v.slice(a, b)
				const next = e.shiftKey ? block.replace(/^(\t| {1,2})/gm, '') : block.replace(/^/gm, '\t')
				area.setSelectionRange(a, b)
				insert(area, next)
				const c = Math.max(a, s + next.length - block.length)
				s === t ? area.setSelectionRange(c, c) : area.setSelectionRange(a, a + next.length)
				return
			}
			escaped = false
			if (e.key === 'Enter' && !e.shiftKey && !e.altKey) {
				e.preventDefault()
				const line = v.slice(a, s)
				insert(area, '\n' + line.match(/^[ \t]*/)[0] + (/[{([]\s*$/.test(line) ? '\t' : ''))
			}
		})
	},
	render: ({ html, host, props: { label } }) => html`
		${label ? html`<label class="label" part="label" for="ta">${label}</label>` : null}
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
						id="ta"
						cols="1"
						aria-label="${label || host.getAttribute('aria-label') || 'Code'}"
						data-effect="el.value !== $$code && (el.value = $$code)"
						data-attr:readonly="$$readonly"
						data-attr:disabled="$$disabled"
						data-on:input="$$code = el.value"
						data-on:change="@change()"
						data-on:blur="@blur()"
						data-on:keydown="@key()"
					></textarea>
				</div>
			</div>
		</div>
	`,
})
