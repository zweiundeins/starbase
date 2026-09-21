import { rocket } from 'datastar'

const slug = (s) => String(s).toLowerCase().trim().replace(/[^a-z0-9]+/g, '-').replace(/^-|-$/g, '')

const styles = /* css */ `
:host {
	--_bg: var(--sb-surface-inset, #0B1224);
	--_border: var(--sb-border, #283552);
	--_text: var(--sb-text-2, #AEBBDD);
	--_text-strong: var(--sb-text-1, #F3F4FA);
	--_active-bg: var(--sb-brand-light, #B09AFF);
	--_active-text: var(--sb-bg, #080D1D);
	--_radius: var(--sb-control-radius, 6px);
	--_focus: var(--sb-focus-ring, 0 0 0 2px #080D1D, 0 0 0 4px #B09AFF);
	display: block;
}
[role="tablist"] {
	display: inline-flex;
	gap: 0.25rem;
	padding: 0.25rem;
	border: 1px solid var(--_border);
	border-radius: calc(var(--_radius) + 4px);
	background: var(--_bg);
	max-inline-size: 100%;
	overflow-x: auto;
	scrollbar-width: none;
}
[role="tab"] {
	all: unset;
	padding: 0.5rem 1rem;
	border: 1px solid var(--_border);
	border-radius: var(--_radius);
	color: var(--_text);
	font-size: 0.875rem;
	font-weight: 600;
	white-space: nowrap;
	cursor: pointer;
	transition: background 120ms, color 120ms, border-color 120ms;
}
[role="tab"]:hover { color: var(--_text-strong); }
[role="tab"]:focus-visible { box-shadow: var(--_focus); }
[role="tab"][aria-selected="true"] {
	border-color: var(--_active-bg);
	background: var(--_active-bg);
	color: var(--_active-text);
	box-shadow: 0 2px 0 color-mix(in oklch, var(--_active-bg) 50%, black);
}
[role="tabpanel"] { padding-block-start: 1rem; }
[role="tabpanel"]:focus-visible { outline: 2px solid var(--_active-bg); outline-offset: 4px; border-radius: 4px; }
`

rocket('sb-tabs', {
	props: ({ array, number, string }) => ({
		labels: array(string.trim).default(() => ['Home', 'Docs', 'API']).docs({ description: 'Tab labels, as a JSON array.' }),
		selected: number.min(0).docs({ description: 'Index of the initially selected tab.' }),
	}),
	manifest: {
		slots: [{ name: '<label-slug>', description: 'Panel content per tab, e.g. slot="docs" for a tab labelled "Docs".' }],
		events: [{ name: 'sb-tab-change', kind: 'custom-event', bubbles: true, composed: true, description: 'detail: { index, label }.' }],
	},
	setup: ({ $$, action, adoptStyles, emit, host, observeProps, props }) => {
		adoptStyles(host, styles)
		$$.selected = props.selected
		observeProps(() => ($$.selected = props.selected), 'selected')
		const select = (i, focus) => {
			const n = props.labels.length
			const next = ((i % n) + n) % n
			if (next !== $$.selected) {
				$$.selected = next
				emit('sb-tab-change', { index: next, label: props.labels[next] })
			}
			if (focus) host.shadowRoot?.querySelectorAll('[role="tab"]')[next]?.focus()
		}
		action('select', (_, i) => select(i, false))
		action('key', ({ evt }, i) => {
			const moves = { ArrowRight: i + 1, ArrowLeft: i - 1, Home: 0, End: props.labels.length - 1 }
			if (evt.key in moves) {
				evt.preventDefault()
				select(moves[evt.key], true)
			}
		})
	},
	render: ({ html, props: { labels } }) => html`
		<div role="tablist" part="tablist">
			${labels.map((label, i) => html`
				<button
					type="button"
					role="tab"
					part="tab"
					id="tab-${i}"
					aria-controls="panel-${i}"
					data-attr:aria-selected="String($$selected === ${i})"
					data-attr:tabindex="$$selected === ${i} ? 0 : -1"
					data-on:click="@select(${i})"
					data-on:keydown="@key(${i})"
				>${label}</button>
			`)}
		</div>
		${labels.map((label, i) => html`
			<div role="tabpanel" part="panel" id="panel-${i}" aria-labelledby="tab-${i}" tabindex="0" data-show="$$selected === ${i}">
				<slot name="${slug(label)}"></slot>
			</div>
		`)}
	`,
})
