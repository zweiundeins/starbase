import { rocket } from 'datastar'

const styles = /* css */ `
:host {
	--_brand: var(--sb-brand, #8C6BFF);
	--_brand-hover: var(--sb-brand-hover, #A58BFF);
	--_brand-light: var(--sb-brand-light, #B09AFF);
	--_brand-subtle: var(--sb-brand-subtle, rgb(140 107 255 / 0.14));
	--_text: var(--sb-text-1, #F3F4FA);
	--_on-brand: var(--sb-text-on-brand, #F3F4FA);
	--_border: var(--sb-border, #283552);
	--_hover: var(--sb-surface-hover, #1A2440);
	--_bg: var(--sb-bg, #080D1D);
	--_radius: var(--sb-control-radius, 6px);
	--_frame: var(--sb-frame-color, #B09AFF);
	--_step: var(--sb-frame-step, 3px);
	--_display: var(--sb-font-display, "Pixelify Sans", ui-monospace, monospace);
	--_focus: var(--sb-focus-ring, 0 0 0 2px #080D1D, 0 0 0 4px #B09AFF);
	display: inline-block;
	vertical-align: middle;
}
:host([disabled]) { pointer-events: none; opacity: 0.5; }
:host([loading]) .btn { cursor: progress; }
/* The same pixel spinner as sb-busy, sized to the button's own text so it
   follows the label at every size. It only exists while loading, so an idle
   button is exactly as wide as it would be without it. */
.spin { position: relative; flex: none; inline-size: 1em; block-size: 1em; }
.spin i {
	position: absolute;
	inset: 0;
	margin: auto;
	inline-size: 0.22em;
	block-size: 0.22em;
	background: currentColor;
	opacity: 0.22;
	transform: rotate(calc(var(--i) * 45deg)) translateY(-0.39em);
	animation: sb-button-blink 720ms steps(1, end) infinite;
	animation-delay: calc(var(--i) * 90ms);
}
@keyframes sb-button-blink { 0% { opacity: 1; } 12.5%, 100% { opacity: 0.22; } }
@media (prefers-reduced-motion: reduce) {
	.spin i { animation: none; opacity: 0.3; }
	.spin i:first-child { opacity: 1; }
}
.btn {
	all: unset;
	box-sizing: border-box;
	display: inline-flex;
	align-items: center;
	justify-content: center;
	gap: 0.5em;
	block-size: var(--_h);
	padding-inline: var(--_px);
	border: 1px solid transparent;
	border-radius: var(--_radius);
	color: var(--_text);
	font: inherit;
	font-size: var(--_fs);
	font-weight: 600;
	line-height: 1;
	white-space: nowrap;
	cursor: pointer;
	user-select: none;
	transition: background 120ms, border-color 120ms, translate 120ms, box-shadow 120ms;
}
.btn:focus-visible { box-shadow: var(--_focus); }
.btn:active { translate: 0 1px; }
.sm { --_h: 2rem; --_px: 0.75rem; --_fs: 0.8125rem; }
.md { --_h: 2.5rem; --_px: 1.125rem; --_fs: 0.875rem; }
.lg { --_h: 3rem; --_px: 1.5rem; --_fs: 1rem; }

.primary { background: var(--_brand); border-color: var(--_brand); color: var(--_on-brand); }
.primary:hover { background: var(--_brand-hover); border-color: var(--_brand-hover); translate: 0 -1px; }
.outline { border-color: var(--_brand-light); }
.outline:hover { background: var(--_brand-subtle); }
.ghost:hover { background: var(--_hover); }
.danger { background: var(--sb-danger, #F2777A); border-color: var(--sb-danger, #F2777A); color: var(--sb-text-on-danger, #1B0A0C); }

/* 8-bit: light plate, notched frame, hard drop shadow. */
.pixel {
	margin: var(--_step);
	border: 0;
	border-radius: 0;
	background: var(--_text);
	color: var(--_bg);
	font-family: var(--_display);
	font-weight: 700;
	letter-spacing: 0.1em;
	text-transform: uppercase;
	--_shadow: calc(var(--_step) * 2);
	box-shadow:
		0 calc(-1 * var(--_step)) 0 0 var(--_frame),
		0 var(--_step) 0 0 var(--_frame),
		calc(-1 * var(--_step)) 0 0 0 var(--_frame),
		var(--_step) 0 0 0 var(--_frame),
		var(--_shadow) var(--_shadow) 0 0 color-mix(in oklch, var(--_frame) 45%, black);
}
.pixel:hover { translate: -1px -1px; --_shadow: calc(var(--_step) * 3); }
.pixel:active { translate: var(--_step) var(--_step); --_shadow: var(--_step); }
.pixel:focus-visible { outline: 2px solid var(--_frame); outline-offset: calc(var(--_step) * 3); }

svg { inline-size: 1.1em; block-size: 1.1em; flex: none; }
::slotted(svg) { inline-size: 1.1em; block-size: 1.1em; }
`

rocket('sb-button', {
	props: ({ bool, oneOf, string }) => ({
		variant: oneOf('primary', 'outline', 'ghost', 'pixel', 'danger')
			.default('primary')
			.docs({ description: 'Visual style.' }),
		size: oneOf('sm', 'md', 'lg').default('md').docs({ description: 'Height and padding.' }),
		href: string.trim.docs({ description: 'Render as a link to this URL.' }),
		caret: bool.docs({ description: 'Show a trailing chevron.' }),
		disabled: bool.docs({ description: 'Disable interaction.' }),
		loading: bool.docs({ description: 'The button\'s action is running: an inline spinner, clicks and Enter blocked, aria-busy. Bind it to data-indicator (and add data-preserve-attr="loading").' }),
	}),
	manifest: {
		slots: [
			{ name: 'default', description: 'The label.' },
			{ name: 'prefix', description: 'An icon before the label.' },
			{ name: 'suffix', description: 'An icon after the label.' },
		],
	},
	setup: ({ adoptStyles, cleanup, host, props }) => {
		adoptStyles(host, styles)
		// While loading the button stays focusable (a disabled button would drop
		// the focus mid-action) but does nothing: the capture phase runs before
		// the page's own data-on:click on this same element.
		const block = (evt) => {
			if (!props.loading) return
			evt.preventDefault()
			evt.stopImmediatePropagation()
		}
		for (const type of ['click', 'keydown']) host.addEventListener(type, block, true)
		cleanup(() => {
			for (const type of ['click', 'keydown']) host.removeEventListener(type, block, true)
		})
	},
	render: ({ html, props: { variant, size, href, caret, disabled, loading } }) => {
		const inner = html`
			${loading ? html`<span class="spin" part="spinner" aria-hidden="true">${Array.from({ length: 8 }, (_, i) => html`<i style="--i: ${i}"></i>`)}</span>` : null}
			<slot name="prefix"></slot>
			<slot></slot>
			<slot name="suffix"></slot>
			${caret ? html`<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3" stroke-linecap="square" aria-hidden="true"><path d="m9 18 6-6-6-6"/></svg>` : null}
		`
		const busy = loading ? 'true' : null
		return href && !disabled
			? html`<a class="btn ${variant} ${size}" part="button" href="${href}" aria-busy="${busy}" aria-disabled="${busy}">${inner}</a>`
			: html`<button class="btn ${variant} ${size}" part="button" type="button" disabled="${disabled}" aria-busy="${busy}" aria-disabled="${busy}">${inner}</button>`
	},
})
