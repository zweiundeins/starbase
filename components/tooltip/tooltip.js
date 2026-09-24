import { rocket } from 'datastar'

const styles = /* css */ `
:host {
	--_bg: var(--sb-surface-raised, #10182B);
	--_border: var(--sb-border-strong, #3A4868);
	--_text: var(--sb-text-1, #F3F4FA);
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
	/* Hidden also leaves the accessibility tree; the fade-out still plays. */
	opacity: 0;
	visibility: hidden;
	transition: opacity 120ms, translate 120ms, visibility 120ms;
	box-shadow: 0 8px 24px -12px rgb(0 0 0 / 0.6);
}
.tip:empty { display: none; }
.tip::before, .tip::after { content: ""; position: absolute; }
/* The pointer can move onto a shown tip: this bridges the 10px gap (and
   the border) on the trigger's side (--_b), once the tip has slid into
   place, so it never covers the trigger. */
.tip::before { inset: var(--_b); visibility: hidden; }
.show::before { visibility: visible; transition: 0s 120ms; }
/* A pixel arrow. */
.tip::after {
	left: calc(50% - 4px);
	top: calc(50% - 2px);
	inline-size: 8px;
	block-size: 4px;
	background: var(--_border);
	clip-path: polygon(0 0, 100% 0, 50% 100%);
}
/* Physical sides, in RTL too. --_n is the slide-in. */
.top, .bottom { left: 50%; translate: -50% var(--_n); }
.left, .right { top: 50%; translate: var(--_n) -50%; }
.top { bottom: calc(100% + 10px); --_n: 4px; --_b: 100% 0 -11px; }
.bottom { top: calc(100% + 10px); --_n: -4px; --_b: -11px 0 100%; }
.left { right: calc(100% + 10px); --_n: 4px; --_b: 0 -11px 0 100%; }
.right { left: calc(100% + 10px); --_n: -4px; --_b: 0 100% 0 -11px; }
.top::after { top: 100%; }
.bottom::after { top: auto; bottom: 100%; rotate: 180deg; }
.left::after { left: calc(100% - 2px); rotate: -90deg; }
.right::after { left: auto; right: calc(100% - 2px); rotate: 90deg; }
.show { opacity: 1; visibility: visible; --_n: 0px; }
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
	setup: ({ $$, adoptStyles, cleanup, host }) => {
		adoptStyles(host, styles)
		$$.hover = $$.focus = false
		// Escape dismisses the tip wherever focus is.
		const esc = (e) => e.key === 'Escape' && ($$.hover = $$.focus = false)
		addEventListener('keydown', esc)
		cleanup(() => removeEventListener('keydown', esc))
	},
	// The anchor holds the trigger and the tip, so the pointer can move onto the
	// tip. Focus pins the tip, except a mouse click's (the pointer is on the
	// trigger and it isn't :focus-visible); a tap's focus comes after its
	// pointerleave, so it pins. The live region announces the tip: a shadow-DOM
	// tip can't be the trigger's aria-describedby. `open` is interpolated: a
	// prop change re-renders.
	render: ({ html, props: { content, open, placement } }) => html`
		<span class="anchor"
			data-on:pointerenter="$$hover = true"
			data-on:pointerleave="$$hover = false"
			data-on:focusin="$$focus = !$$hover || evt.composedPath()[0].matches(':focus-visible')"
			data-on:focusout="$$focus = false"
		><slot></slot><span aria-live="polite"><span class="tip ${placement}" part="tip" role="tooltip" data-class:show="${open} || $$hover || $$focus">${content}</span></span></span>
	`,
})
