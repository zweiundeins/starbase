import { rocket } from 'datastar'

// Requires <sb-code-editor> to be loaded on the page as well.

const dedent = (text) => {
	const lines = text.replace(/^\s*\n|\n\s*$/g, '').split('\n')
	const indent = Math.min(...lines.filter((l) => l.trim()).map((l) => l.match(/^[ \t]*/)[0].length))
	return lines.map((l) => l.slice(Number.isFinite(indent) ? indent : 0)).join('\n')
}
const langOf = (name) => (name.endsWith('.html') ? 'html' : name.endsWith('.css') ? 'css' : 'js')
const TAGS_RE = /rocket\(\s*['"](sb-[a-z0-9-]+)['"]/g

// Per-instance state shared by setup, render and onFirstRender.
const state = new WeakMap()

const styles = /* css */ `
:host {
	--_bg: var(--sb-surface-card, #141D32);
	--_inset: var(--sb-surface-inset, #0B1224);
	--_border: var(--sb-border, #283552);
	--_text: var(--sb-text-1, #F3F4FA);
	--_text-2: var(--sb-text-2, #AEBBDD);
	--_muted: var(--sb-text-muted, #7785A8);
	--_brand: var(--sb-brand, #8C6BFF);
	--_brand-light: var(--sb-brand-light, #B09AFF);
	--_danger: var(--sb-danger, #F2777A);
	--_warn: var(--sb-warn, #F5C451);
	--_radius: var(--sb-radius-lg, 10px);
	--_h: var(--sb-code-playground-height, 34rem);
	display: block;
	container-type: inline-size;
}
.pg { display: grid; grid-template-rows: auto 1fr; block-size: var(--_h); border: 1px solid var(--_border); border-radius: var(--_radius); background: var(--_bg); overflow: hidden; }
.bar { display: flex; flex-wrap: wrap; align-items: center; gap: 0.5rem; padding: 0.5rem 0.75rem; border-block-end: 1px solid var(--_border); font-size: 0.8125rem; }
[role="tablist"] { display: flex; gap: 0.25rem; margin-inline-end: auto; overflow-x: auto; }
[role="tab"], .btn, select {
	all: unset;
	box-sizing: border-box;
	padding: 0.35rem 0.75rem;
	border: 1px solid transparent;
	border-radius: 6px;
	color: var(--_text-2);
	font: inherit;
	cursor: pointer;
	white-space: nowrap;
}
[role="tab"]:hover, .btn:hover { color: var(--_text); background: color-mix(in oklch, var(--_text) 6%, transparent); }
[role="tab"][aria-selected="true"] { color: var(--_text); border-color: var(--_border); background: var(--_inset); }
[role="tab"]:focus-visible, .btn:focus-visible, select:focus-visible { outline: 2px solid var(--_brand-light); outline-offset: 2px; }
select { border-color: var(--_border); background: var(--_inset); color: var(--_text); }
.btn.run { border-color: var(--_brand); background: var(--_brand); color: var(--sb-text-on-brand, #fff); font-weight: 700; }
.btn.run:hover { background: color-mix(in oklch, var(--_brand), white 12%); }
label.auto { display: inline-flex; align-items: center; gap: 0.35rem; color: var(--_text-2); cursor: pointer; }
label.auto input { accent-color: var(--_brand); }
.panes { display: grid; grid-template-columns: minmax(0, 1fr) minmax(0, 1fr); min-block-size: 0; }
@container (width < 48rem) { .panes { grid-template-columns: minmax(0, 1fr); grid-template-rows: minmax(0, 1fr) minmax(0, 1fr); } }
.editors { min-block-size: 0; border-inline-end: 1px solid var(--_border); overflow: hidden; }
@container (width < 48rem) { .editors { border-inline-end: 0; border-block-end: 1px solid var(--_border); } }
.editors sb-code-editor { block-size: 100%; --sb-code-editor-height: 100%; --sb-code-editor-min-height: 100%; }
.editors sb-code-editor::part(editor) { border: 0; border-radius: 0; block-size: 100%; }
/* The console owns a fixed share of the pane and scrolls inside it, so the
   preview never moves as output arrives. */
.preview { display: grid; grid-template-rows: minmax(0, 1fr) minmax(4.5rem, 30%); min-block-size: 0; }
iframe { inline-size: 100%; block-size: 100%; border: 0; background: var(--sb-bg, #080D1D); }
.console { min-block-size: 0; overflow: auto; margin: 0; padding: 0.5rem 0.75rem; border-block-start: 1px solid var(--_border); background: var(--_inset); color: var(--_text-2); font: 0.75rem/1.5 var(--sb-font-ui, ui-monospace, monospace); }
.console:not(:has(div))::before { content: "Console"; color: var(--_muted); }
.console .error { color: var(--_danger); }
.console .warn { color: var(--_warn); }
.status { color: var(--_muted); font-size: 0.75rem; }
`

rocket('sb-code-playground', {
	props: ({ array, bool, json, number, string }) => ({
		runner: string.trim.default('/playground/run').docs({ description: 'URL of the sandbox runner page (see the protocol in the docs).' }),
		deps: json.default(() => ({})).docs({ description: 'Other components to load in the preview, as JSON {"sb-tag": "module URL"}. Tags defined by the edited code are skipped.' }),
		autoRun: bool.default(true).docs({ description: 'Re-run shortly after every edit.' }),
		theme: string.trim.default('deep-space').docs({ description: 'Initial preview theme (data-sb-theme).' }),
		themes: array(string.trim).default(() => ['deep-space', 'nebula', 'terminal', 'daylight']).docs({ description: 'Themes offered in the picker.' }),
		delay: number.clamp(100, 5000).default(600).docs({ description: 'Auto-run debounce, in ms.' }),
		base: string.trim.docs({ description: 'URL that relative imports in component.js resolve against (the folder its vendored files are served from).' }),
		initial: json.default(() => ({})).docs({ description: 'Initial files as JSON {"component.js": "…"}; handy for server-rendered pages. Child scripts are used when empty.' }),
	}),
	manifest: {
		slots: [{ name: '(files)', description: 'Child <script type="text/plain" data-file="component.js|index.html|style.css"> elements with the initial files.' }],
		events: [
			{ name: 'sb-change', kind: 'custom-event', bubbles: true, composed: true, description: 'After an edit. detail: { files }.' },
			{ name: 'sb-run', kind: 'custom-event', bubbles: true, composed: true, description: 'When the preview is (re)started. detail: { files }.' },
		],
	},
	renderOnPropChange: false,
	setup: ({ $$, action, adoptStyles, cleanup, defineHostProp, emit, host, props }) => {
		adoptStyles(host, styles)
		const files = new Map()
		for (const el of host.querySelectorAll(':scope > script[type="text/plain"][data-file]')) {
			files.set(el.dataset.file, dedent(el.textContent))
		}
		for (const [name, code] of Object.entries(props.initial || {})) files.set(name, String(code))
		if (!files.size) files.set('component.js', '')
		const refs = {} // filled from data-ref:* in onFirstRender
		state.set(host, { files, refs })

		// Local signals: everything the markup binds to.
		$$.active = 0
		$$.auto = props.autoRun
		$$.theme = props.theme
		$$.status = ''
		$$.src = ''
		$$.lines = [] // console output

		const all = () => Object.fromEntries(files)
		let pending = null, runs = 0, started = 0, timer = 0

		const run = () => {
			clearTimeout(timer)
			$$.lines = []
			$$.status = 'Running…'
			started = performance.now()
			// Skip stock modules for tags the edited code defines.
			const own = new Set()
			for (const code of files.values()) for (const m of code.matchAll(TAGS_RE)) own.add(m[1])
			const deps = Object.entries(props.deps || {})
				.filter(([tag]) => !own.has(tag))
				.map(([, url]) => new URL(url, location.href).href)
			const base = props.base ? new URL(props.base, location.href).href : ''
			pending = { type: 'run', files: all(), deps, theme: $$.theme, base }
			const url = new URL(props.runner, location.href)
			url.searchParams.set('r', String(++runs))
			$$.src = url.href // a fresh document: custom elements can't be redefined
			emit('sb-run', { files: all() })
		}
		const log = (level, text) => {
			$$.lines = [...$$.lines.slice(-199), { level, text }]
		}

		action('run', run)
		action('edit', ({ el }) => {
			files.set(el.dataset.file, el.value)
			emit('sb-change', { files: all() })
			clearTimeout(timer)
			if ($$.auto) timer = setTimeout(run, props.delay)
		})
		action('message', ({ evt }) => {
			const frame = refs.frame
			if (!frame || evt.source !== frame.contentWindow || evt.data?.source !== 'sb-runner') return
			const m = evt.data
			if (m.type === 'ready' && pending) frame.contentWindow.postMessage(pending, '*')
			else if (m.type === 'console') log(m.level, m.args.join(' '))
			else if (m.type === 'error') log('error', m.message + (m.line ? ` (line ${m.line})` : ''))
			else if (m.type === 'done') $$.status = `Ran in ${Math.round(performance.now() - started)} ms`
		})

		defineHostProp('files', {
			get: all,
			set: (obj) => {
				for (const [k, v] of Object.entries(obj || {})) files.set(k, String(v))
				for (const ed of host.shadowRoot?.querySelectorAll('sb-code-editor') ?? []) ed.value = files.get(ed.dataset.file) ?? ''
				run()
			},
		})
		defineHostProp('run', { value: run })
		cleanup(() => clearTimeout(timer))
		run()
	},
	render: ({ html, host, props: { themes } }) => {
		const { files } = state.get(host)
		const names = [...files.keys()]
		return html`
			<div class="pg" part="playground" data-on:message__window="@message()">
				<div class="bar">
					<div role="tablist" aria-label="Files">
						${names.map((n, i) => html`
							<button type="button" role="tab"
								data-attr:aria-selected="String($$active === ${i})"
								data-on:click="$$active = ${i}">${n}</button>`)}
					</div>
					<span class="status" role="status" data-text="$$status"></span>
					<label class="auto"><input type="checkbox" data-bind:auto> Auto</label>
					<select aria-label="Preview theme" data-bind:theme data-on:change="@run()">
						${themes.map((t) => html`<option value="${t}">${t}</option>`)}
					</select>
					<button type="button" class="btn run" part="run" title="Run (Ctrl+Enter)" data-on:click="@run()">▶ Run</button>
				</div>
				<div class="panes">
					<div class="editors">
						${names.map((n, i) => html`
							<sb-code-editor data-file="${n}" language="${langOf(n)}" value="${files.get(n)}"
								data-show="$$active === ${i}"
								data-on:input="@edit()"
								data-on:sb-run__stop="@run()"></sb-code-editor>`)}
					</div>
					<div class="preview">
						<iframe part="preview" title="Preview" loading="lazy" sandbox="allow-scripts allow-modals" data-ref:frame data-attr:src="$$src"></iframe>
						<div class="console" part="console" role="log" aria-label="Console"
							data-effect="$$lines.length, requestAnimationFrame(() => (el.scrollTop = el.scrollHeight))">
							<template data-for="line in $$lines">
								<div data-attr:class="line.level" data-text="line.text"></div>
							</template>
						</div>
					</div>
				</div>
			</div>
		`
	},
	onFirstRender: ({ host, refs }) => Object.assign(state.get(host).refs, refs),
})
