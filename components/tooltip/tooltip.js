import { rocket } from 'datastar'

const styles = /* css */ `
:host {
	--_bg: var(--sb-surface-raised, #10182B);
	--_border: var(--sb-border-strong, #3A4868);
	--_text: var(--sb-text-1, #F3F4FA);
	--_gap: 10px;
	position: relative;
	display: inline-block;
	vertical-align: middle;
}
.anchor { display: inline-block; }
.tip {
	position: absolute;
	z-index: 70;
	inline-size: max-content;
	max-inline-size: 16rem;
	padding: 0.4rem 0.75rem;
	border: 1px solid var(--_border);
	border-radius: 6px;
	background: var(--_bg);
	color: var(--_text);
	font-size: 0.8125rem;
	font-weight: 600;
	line-height: 1.4;
	pointer-events: none;
	opacity: 0;
	transition: opacity 120ms, translate 120ms;
	box-shadow: 0 8px 24px -12px rgb(0 0 0 / 0.6);
}
/* A pixel arrow. */
.tip::after {
	content: "";
	position: absolute;
	inline-size: 8px;
	block-size: 4px;
	background: var(--_border);
	clip-path: polygon(0 0, 100% 0, 75% 50%, 50% 100%, 25% 50%);
}
.top { inset-block-end: calc(100% + var(--_gap)); inset-inline-start: 50%; translate: -50% 4px; }
.top::after { inset-block-start: 100%; inset-inline-start: calc(50% - 4px); }
.bottom { inset-block-start: calc(100% + var(--_gap)); inset-inline-start: 50%; translate: -50% -4px; }
.bottom::after { inset-block-end: 100%; inset-inline-start: calc(50% - 4px); rotate: 180deg; }
.left { inset-inline-end: calc(100% + var(--_gap)); inset-block-start: 50%; translate: 4px -50%; }
.left::after { inset-inline-start: 100%; inset-block-start: calc(50% - 2px); rotate: -90deg; translate: -2px 0; }
.right { inset-inline-start: calc(100% + var(--_gap)); inset-block-start: 50%; translate: -4px -50%; }
.right::after { inset-inline-end: 100%; inset-block-start: calc(50% - 2px); rotate: 90deg; translate: 2px 0; }
.show.top, .show.bottom { opacity: 1; translate: -50% 0; }
.show.left, .show.right { opacity: 1; translate: 0 -50%; }
@media (prefers-reduced-motion: reduce) { .tip { transition: none; } }
`

rocket('sb-tooltip', {
	props: ({ bool, oneOf, string }) => ({
		content: string.docs({ description: 'Tooltip text.' }),
		placement: oneOf('top', 'bottom', 'left', 'right').default('top').docs({ description: 'Side of the trigger.' }),
		open: bool.docs({ description: 'Keep the tooltip visible.' }),
	}),
	manifest: {
		slots: [{ name: 'default', description: 'The trigger element.' }],
	},
	setup: ({ $$, adoptStyles, host, observeProps, props }) => {
		adoptStyles(host, styles)
		$$.forced = props.open
		$$.hover = false
		observeProps(() => ($$.forced = props.open), 'open')
	},
	// Events from the slotted trigger bubble through the anchor.
	render: ({ html, props: { content, placement } }) => html`
		<span class="anchor"
			data-on:pointerenter="$$hover = true"
			data-on:pointerleave="$$hover = false"
			data-on:focusin="$$hover = true"
			data-on:focusout="$$hover = false"
			data-on:keydown="evt.key === 'Escape' && ($$hover = false)"
		><slot></slot></span>
		<span class="tip ${placement}" part="tip" role="tooltip" data-class:show="$$forced || $$hover">${content}</span>
	`,
})
