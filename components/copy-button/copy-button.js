import { rocket } from 'datastar'

// Tokens with fallbacks, so the component also works outside Starbase.
// Forced colors drop the focus ring (a box-shadow): the transparent outline
// shows there instead. The tip is centred with a physical left, like its
// translate: a logical inset would put it off-centre in RTL.
const styles = /* css */ `
:host {
	--_bg: var(--sb-surface-raised, #10182B);
	--_border: var(--sb-border, #283552);
	--_border-strong: var(--sb-border-strong, #3A4868);
	--_text: var(--sb-text-2, #AEBBDD);
	--_text-hover: var(--sb-text-1, #F3F4FA);
	--_ok: var(--sb-ok, #6EF59A);
	--_danger: var(--sb-danger, #F2777A);
	--_on-danger: var(--sb-text-on-danger, #1B0A0C);
	--_tip-bg: var(--sb-surface-raised, #10182B);
	--_radius: var(--sb-radius-sm, 6px);
	--_focus: var(--sb-focus-ring, 0 0 0 2px #080D1D, 0 0 0 4px #B09AFF);
	display: inline-block;
	position: relative;
	vertical-align: middle;
}
:host([hidden]) { display: none; }
button {
	all: unset;
	position: relative;
	display: inline-grid;
	place-items: center;
	inline-size: 2rem;
	block-size: 2rem;
	/* The icon is half the button, whatever size a page gives it (::part(button)). */
	container-type: size;
	border: 1px solid var(--_border);
	border-radius: var(--_radius);
	background: var(--_bg);
	color: var(--_text);
	cursor: pointer;
	transition: color 120ms, border-color 120ms, background 120ms;
}
button:hover { color: var(--_text-hover); border-color: var(--_border-strong); }
button:focus-visible { box-shadow: var(--_focus); outline: 2px solid transparent; outline-offset: 2px; }
button.copied { color: var(--_ok); border-color: color-mix(in oklch, var(--_ok) 50%, transparent); }
button.failed { color: var(--_danger); border-color: color-mix(in oklch, var(--_danger) 50%, transparent); }
svg { inline-size: 50cqi; block-size: 50cqi; } /* half the button (the 2rem box inside its border) */
.tip {
	position: absolute;
	inset-block-end: calc(100% + 6px);
	left: 50%;
	padding: 2px 8px;
	border: 1px solid var(--_border-strong);
	border-radius: 4px;
	background: var(--_tip-bg);
	color: var(--_text-hover);
	font-size: 0.6875rem;
	font-weight: 600;
	white-space: nowrap;
	pointer-events: none;
	opacity: 0;
	translate: -50% 4px;
	transition: opacity 120ms, translate 120ms;
}
.tip.failed { border-color: var(--_danger); background: var(--_danger); color: var(--_on-danger); }
.tip.shown { opacity: 1; translate: -50% 0; }
`

rocket('sb-copy-button', {
	props: ({ string, number }) => ({
		value: string.docs({ description: 'The text copied to the clipboard.' }),
		label: string.default('Copy to clipboard').docs({ description: 'Accessible label of the button.' }),
		copiedLabel: string.default('Copied!').docs({ description: 'Shown and announced after copying.' }),
		failedLabel: string.default('Copy failed').docs({ description: 'Shown and announced when the browser refuses the clipboard (no secure context, permissions policy…).' }),
		resetMs: number.min(300).default(1600).docs({ description: 'How long the copied or failed state lasts, in ms (at least 300).' }),
	}),
	manifest: {
		events: [
			{ name: 'sb-copy', kind: 'custom-event', bubbles: true, composed: true, description: 'After copying. detail: { value }.' },
			{ name: 'sb-copy-error', kind: 'custom-event', bubbles: true, composed: true, description: 'When the browser refuses the clipboard write. detail: { value, error } (the error\'s name: "NotAllowedError" when refused, "TypeError" where there is no Clipboard API, e.g. on plain http://).' },
		],
	},
	setup: ({ $$, action, adoptStyles, cleanup, emit, host, props }) => {
		adoptStyles(host, styles)
		$$.state = '' // '', 'copied' or 'failed'
		// The message is read out by the status region (and shown in the tip).
		$$.message = () => ($$.state === 'copied' ? props.copiedLabel : $$.state === 'failed' ? props.failedLabel : '')
		let timer = 0
		action('copy', async () => {
			const value = props.value
			// Removed while the browser was writing: its signals are gone, don't bring them back.
			try {
				await navigator.clipboard.writeText(value)
				if (!host.isConnected) return
				$$.state = 'copied'
				emit('sb-copy', { value })
			} catch (err) {
				if (!host.isConnected) return
				$$.state = 'failed'
				emit('sb-copy-error', { value, error: err?.name || String(err) })
			}
			clearTimeout(timer)
			timer = setTimeout(() => ($$.state = ''), props.resetMs)
		})
		cleanup(() => clearTimeout(timer))
	},
	// The template reads only label: a new value (e.g. data-attr'd on every
	// keystroke) needn't re-render and morph the shadow root.
	renderOnPropChange: ({ changes }) => 'label' in changes,
	// The status region sits outside the button: a button's children are
	// presentational, so a live region inside it may never be announced.
	render: ({ html, props: { label } }) => html`
		<button
			type="button"
			part="button"
			aria-label="${label}"
			data-on:click="@copy()"
			data-class:copied="$$state === 'copied'"
			data-class:failed="$$state === 'failed'"
		>
			<svg data-show="!$$state" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><rect width="14" height="14" x="8" y="8" rx="2"/><path d="M4 16c-1.1 0-2-.9-2-2V4c0-1.1.9-2 2-2h10c1.1 0 2 .9 2 2"/></svg>
			<svg data-show="$$state === 'copied'" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M20 6 9 17l-5-5"/></svg>
			<svg data-show="$$state === 'failed'" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M18 6 6 18M6 6l12 12"/></svg>
		</button>
		<span class="tip" part="tip" role="status" data-class:shown="$$state" data-class:failed="$$state === 'failed'" data-text="$$message"></span>
	`,
})
