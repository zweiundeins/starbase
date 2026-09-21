import { rocket } from 'datastar'

// Tokens with fallbacks, so the component also works outside Starbase.
const styles = /* css */ `
:host {
	--_bg: var(--sb-surface-raised, #10182B);
	--_border: var(--sb-border, #283552);
	--_text: var(--sb-text-2, #AEBBDD);
	--_text-hover: var(--sb-text-1, #F3F4FA);
	--_ok: var(--sb-ok, #6EF59A);
	--_radius: var(--sb-radius-sm, 6px);
	--_focus: var(--sb-focus-ring, 0 0 0 2px #080D1D, 0 0 0 4px #B09AFF);
	display: inline-block;
	vertical-align: middle;
}
button {
	all: unset;
	position: relative;
	display: inline-grid;
	place-items: center;
	inline-size: 2rem;
	block-size: 2rem;
	border: 1px solid var(--_border);
	border-radius: var(--_radius);
	background: var(--_bg);
	color: var(--_text);
	cursor: pointer;
	transition: color 120ms, border-color 120ms, background 120ms;
}
button:hover { color: var(--_text-hover); border-color: color-mix(in oklch, var(--_border), white 15%); }
button:focus-visible { box-shadow: var(--_focus); }
button.copied { color: var(--_ok); border-color: color-mix(in oklch, var(--_ok) 50%, transparent); }
svg { inline-size: 1rem; block-size: 1rem; }
.tip {
	position: absolute;
	inset-block-end: calc(100% + 6px);
	padding: 2px 8px;
	border-radius: 4px;
	background: var(--_text-hover);
	color: #080D1D;
	font-size: 0.6875rem;
	font-weight: 600;
	white-space: nowrap;
	pointer-events: none;
	opacity: 0;
	translate: 0 4px;
	transition: opacity 120ms, translate 120ms;
}
button.copied .tip { opacity: 1; translate: 0 0; }
`

rocket('sb-copy-button', {
	props: ({ string, number }) => ({
		value: string.docs({ description: 'The text copied to the clipboard.' }),
		label: string.default('Copy to clipboard').docs({ description: 'Accessible label of the button.' }),
		copiedLabel: string.default('Copied!').docs({ description: 'Label shown after copying.' }),
		resetMs: number.min(300).default(1600).docs({ description: 'How long the copied state lasts, in ms.' }),
	}),
	manifest: {
		events: [
			{ name: 'sb-copy', kind: 'custom-event', bubbles: true, composed: true, description: 'After copying. detail: { value }.' },
		],
	},
	setup: ({ $$, action, adoptStyles, cleanup, emit, host, observeProps, props }) => {
		adoptStyles(host, styles)
		$$.copied = false
		$$.label = props.label
		$$.copiedLabel = props.copiedLabel
		observeProps(() => {
			$$.label = props.label
			$$.copiedLabel = props.copiedLabel
		}, 'label', 'copiedLabel')
		let timer = 0
		action('copy', async () => {
			try {
				await navigator.clipboard.writeText(props.value)
			} catch {
				return
			}
			$$.copied = true
			emit('sb-copy', { value: props.value })
			clearTimeout(timer)
			timer = setTimeout(() => ($$.copied = false), props.resetMs)
		})
		cleanup(() => clearTimeout(timer))
	},
	render: ({ html }) => html`
		<button
			type="button"
			part="button"
			data-on:click="@copy()"
			data-class:copied="$$copied"
			data-attr:aria-label="$$copied ? $$copiedLabel : $$label"
		>
			<svg data-show="!$$copied" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><rect width="14" height="14" x="8" y="8" rx="2"/><path d="M4 16c-1.1 0-2-.9-2-2V4c0-1.1.9-2 2-2h10c1.1 0 2 .9 2 2"/></svg>
			<svg data-show="$$copied" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M20 6 9 17l-5-5"/></svg>
			<span class="tip" aria-hidden="true" data-text="$$copiedLabel"></span>
		</button>
	`,
})
