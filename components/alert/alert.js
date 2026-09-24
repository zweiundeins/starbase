import { rocket } from 'datastar'

// The custom states of each element's ElementInternals: attachInternals() works
// once, and setup runs again when the element is re-attached. :state(closed) is
// styleable from the page, morph-proof, and survives a move.
const statesOf = new WeakMap()

const styles = /* css */ `
:host {
	--_bg: var(--sb-surface-inset, #0B1224);
	--_border: var(--sb-border, #283552);
	--_text: var(--sb-text-1, #F3F4FA);
	--_muted: var(--sb-text-2, #AEBBDD);
	--_radius: var(--sb-radius, 8px);
	display: block;
}
:host([hidden]) { display: none; }
/* No gap left in a stack; !important beats a page's display rule. */
:host(:state(closed)) { display: none !important; }
.alert {
	--_tone: var(--sb-info, #65BFFF);
	display: grid;
	grid-template-columns: auto 1fr auto;
	align-items: start;
	gap: 0.625rem;
	padding: 0.75rem 0.875rem;
	border: 1px solid color-mix(in oklch, var(--_tone) 30%, var(--_border));
	border-radius: var(--_radius);
	background: linear-gradient(color-mix(in oklch, var(--_tone) 7%, transparent), transparent), var(--_bg);
	color: var(--_muted);
	animation: in 180ms cubic-bezier(0.2, 0, 0, 1);
}
.success { --_tone: var(--sb-ok, #6EF59A); }
.info { --_tone: var(--sb-info, #65BFFF); }
.warning { --_tone: var(--sb-warn, #F5C451); }
.danger { --_tone: var(--sb-danger, #F2777A); }
/* A pixel "status light". */
.light {
	inline-size: 12px;
	block-size: 12px;
	margin-block-start: 0.3em;
	background: var(--_tone);
	clip-path: polygon(3px 0, 9px 0, 9px 3px, 12px 3px, 12px 9px, 9px 9px, 9px 12px, 3px 12px, 3px 9px, 0 9px, 0 3px, 3px 3px);
	box-shadow: 0 0 12px var(--_tone);
}
.heading { display: block; color: var(--_tone); font-weight: 700; }
.heading + .msg { margin-block-start: 0.15rem; }
.msg { font-size: 0.875rem; }
.close {
	all: unset;
	display: grid;
	place-items: center;
	inline-size: 1.75rem;
	block-size: 1.75rem;
	margin: -0.25rem -0.375rem 0 0;
	border-radius: 4px;
	color: var(--_muted);
	cursor: pointer;
}
.close:hover { color: var(--_text); background: color-mix(in oklch, var(--_text) 8%, transparent); }
.close:focus-visible { outline: 2px solid var(--_tone); }
.close svg { inline-size: 1rem; block-size: 1rem; }
@keyframes in { from { opacity: 0; translate: 0 -4px; } }
@media (prefers-reduced-motion: reduce) { .alert { animation: none; } }
`

rocket('sb-alert', {
	props: ({ bool, oneOf, string }) => ({
		variant: oneOf('info', 'success', 'warning', 'danger').default('info').docs({ description: 'Tone of the message.' }),
		heading: string.trim.docs({ description: 'Bold first line.' }),
		closable: bool.docs({ description: 'Show a dismiss button.' }),
		open: bool.default(true).docs({
			description: 'Shown. Any change the server makes to the attribute wins, over a dismissal too: open="false" hides it, open or no attribute shows it. Unchanged markup keeps a dismissal. From script: host.open, show(), hide().',
		}),
	}),
	manifest: {
		slots: [{ name: 'default', description: 'The message.' }],
		events: [{ name: 'sb-close', kind: 'event', bubbles: true, composed: true, description: 'The user pressed the close button (not sent for hide(), host.open or the server). No detail.' }],
	},
	setup: ({ action, adoptStyles, defineHostProp, emit, host, overrideProp, props }) => {
		adoptStyles(host, styles)
		let states = statesOf.get(host)
		const set = (open) => void states[open ? 'delete' : 'add']('closed')
		if (!states) {
			statesOf.set(host, (states = host.attachInternals().states))
			set(props.open)
			// The attribute is the server's word, so every change to it wins, over
			// a dismissal too (observeProps only fires when the decoded value
			// changes, and absent, "" and "true" all decode to true). A morph
			// writes only what differs, so markup sent again unchanged keeps a
			// dismissal. Unlike sb-details, a removed attribute counts: it means
			// the default, open (the Playground's switch shows it that way).
			// Observes for the element's lifetime, while moved or detached too.
			let served = host.getAttribute('open')
			new MutationObserver(() => served !== (served = host.getAttribute('open')) && set(props.open)).observe(host, { attributeFilter: ['open'] })
		}
		// Local: never reflected, so a morph can't undo it.
		overrideProp('open', () => !states.has('closed'), set)
		defineHostProp('show', { value: () => set(true) })
		defineHostProp('hide', { value: () => set(false) })
		action('close', () => {
			set(false)
			emit('sb-close')
		})
	},
	render: ({ html, props: { variant, heading, closable } }) => html`
		<div class="alert ${variant}" part="alert" role="${variant === 'danger' || variant === 'warning' ? 'alert' : 'status'}">
			<span class="light" aria-hidden="true"></span>
			<div>
				${heading ? html`<strong class="heading" part="heading">${heading}</strong>` : null}
				<div class="msg" part="message"><slot></slot></div>
			</div>
			${closable ? html`<button class="close" type="button" aria-label="Dismiss" data-on:click="@close()"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" aria-hidden="true"><path d="M18 6 6 18"/><path d="m6 6 12 12"/></svg></button>` : null}
		</div>
	`,
})
