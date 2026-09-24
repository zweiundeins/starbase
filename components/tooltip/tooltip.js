import { rocket } from 'datastar'

const styles = /* css */ `
:host {
	--_bg: var(--sb-surface-raised, #10182B);
	--_border: var(--sb-border-strong, #3A4868);
	--_text: var(--sb-text-1, #F3F4FA);
	--_gap: 10px;
	/* Hug the trigger exactly: the tip is positioned against this box. */
	position: relative;
	display: inline-flex;
	vertical-align: middle;
}
:host([hidden]) { display: none; }
.anchor { display: inline-flex; }
.tip {
	position: absolute;
	z-index: var(--sb-z-tooltip, 70);
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
	left: calc(50% - 4px);
	top: calc(50% - 2px);
	inline-size: 8px;
	block-size: 4px;
	background: var(--_border);
	clip-path: polygon(0 0, 100% 0, 75% 50%, 50% 100%, 25% 50%);
}
/* Physical sides, in RTL too. --_n is the slide-in. */
.top, .bottom { left: 50%; translate: -50% var(--_n); }
.left, .right { top: 50%; translate: var(--_n) -50%; }
.top { bottom: calc(100% + var(--_gap)); --_n: 4px; }
.bottom { top: calc(100% + var(--_gap)); --_n: -4px; }
.left { right: calc(100% + var(--_gap)); --_n: 4px; }
.right { left: calc(100% + var(--_gap)); --_n: -4px; }
.top::after { top: 100%; }
.bottom::after { top: auto; bottom: 100%; rotate: 180deg; }
.left::after { left: calc(100% - 2px); rotate: -90deg; }
.right::after { left: auto; right: calc(100% - 2px); rotate: 90deg; }
.show { opacity: 1; --_n: 0px; }
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
	setup: ({ $$, adoptStyles, host }) => {
		adoptStyles(host, styles)
		$$.hover = false
	},
	// Events from the slotted trigger bubble through the anchor. `open` is
	// interpolated: a prop change re-renders.
	render: ({ html, props: { content, open, placement } }) => html`
		<span class="anchor"
			data-on:pointerenter="$$hover = true"
			data-on:pointerleave="$$hover = false"
			data-on:focusin="$$hover = true"
			data-on:focusout="$$hover = false"
			data-on:keydown="evt.key === 'Escape' && ($$hover = false)"
		><slot></slot></span>
		<span class="tip ${placement}" part="tip" role="tooltip" data-class:show="${open} || $$hover">${content}</span>
	`,
})
