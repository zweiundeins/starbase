import { rocket } from 'datastar'

const styles = /* css */ `
:host {
	--_bg: var(--sb-surface-inset, #0B1224);
	--_border: var(--sb-border, #283552);
	--_text: var(--sb-text-1, #F3F4FA);
	--_muted: var(--sb-text-2, #AEBBDD);
	--_radius: var(--sb-radius, 8px);
	display: block;
}
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
		open: bool.default(true).docs({ description: 'Shown. The server can hide or re-show it by changing the attribute (open="false").' }),
	}),
	manifest: {
		slots: [{ name: 'default', description: 'The message.' }],
		events: [{ name: 'sb-close', kind: 'custom-event', bubbles: true, composed: true, description: 'After the alert was dismissed.' }],
	},
	setup: ({ $$, action, adoptStyles, defineHostProp, emit, host, observeProps, props }) => {
		adoptStyles(host, styles)
		$$.open = props.open
		observeProps(() => ($$.open = props.open), 'open')
		action('close', () => {
			$$.open = false
			emit('sb-close')
		})
		defineHostProp('show', { value: () => ($$.open = true) })
		defineHostProp('hide', { value: () => ($$.open = false) })
	},
	render: ({ html, props: { variant, heading, closable } }) => html`
		<div
			class="alert ${variant}"
			part="alert"
			role="${variant === 'danger' || variant === 'warning' ? 'alert' : 'status'}"
			data-show="$$open"
		>
			<span class="light" aria-hidden="true"></span>
			<div>
				${heading ? html`<strong class="heading" part="heading">${heading}</strong>` : null}
				<div class="msg" part="message"><slot></slot></div>
			</div>
			${closable ? html`<button class="close" type="button" aria-label="Dismiss" data-on:click="@close()"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" aria-hidden="true"><path d="M18 6 6 18"/><path d="m6 6 12 12"/></svg></button>` : null}
		</div>
	`,
})
