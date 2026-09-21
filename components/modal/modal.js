import { rocket } from 'datastar'

const styles = /* css */ `
:host {
	--_bg: var(--sb-surface-raised, #10182B);
	--_border: var(--sb-border, #283552);
	--_text: var(--sb-text-1, #F3F4FA);
	--_muted: var(--sb-text-2, #AEBBDD);
	--_overlay: var(--sb-surface-overlay, rgb(5 8 20 / 0.72));
	--_brand: var(--sb-brand, #8C6BFF);
	--_radius: var(--sb-radius-lg, 10px);
	display: contents;
}
.panel {
	box-sizing: border-box;
	inline-size: min(28rem, 100vw - 2rem);
	padding: 0;
	border: 1px solid var(--_border);
	border-radius: var(--_radius);
	background: var(--_bg);
	color: var(--_muted);
	box-shadow: 0 24px 48px -16px rgb(0 0 0 / 0.7);
}
.inline { display: block; inline-size: 100%; max-inline-size: 28rem; }
dialog[open] { animation: pop 180ms cubic-bezier(0.2, 0, 0, 1); }
dialog::backdrop { background: var(--_overlay); backdrop-filter: blur(2px); }
header { display: flex; align-items: center; justify-content: space-between; gap: 1rem; padding: 1rem 1.25rem 0; }
h2 { margin: 0; color: var(--_text); font-size: 1rem; font-weight: 700; }
.body { padding: 0.5rem 1.25rem 1.25rem; font-size: 0.875rem; }
footer { display: flex; justify-content: flex-end; gap: 0.5rem; padding: 0.875rem 1.25rem; border-block-start: 1px solid var(--_border); }
footer[hidden] { display: none; }
.close {
	all: unset;
	display: grid;
	place-items: center;
	inline-size: 1.75rem;
	block-size: 1.75rem;
	border-radius: 4px;
	color: var(--_muted);
	cursor: pointer;
}
.close:hover { color: var(--_text); background: color-mix(in oklch, var(--_text) 8%, transparent); }
.close:focus-visible { outline: 2px solid var(--_brand); }
.close svg { inline-size: 1rem; block-size: 1rem; }
@keyframes pop { from { opacity: 0; scale: 0.96; } }
@media (prefers-reduced-motion: reduce) { dialog[open] { animation: none; } }
`

rocket('sb-modal', {
	props: ({ bool, string }) => ({
		heading: string.trim.default('Dialog').docs({ description: 'Title of the dialog.' }),
		open: bool.docs({ description: 'Open on first render. Use show() and close() afterwards.' }),
		inline: bool.docs({ description: 'Render in place, without an overlay (previews, docs).' }),
		closable: bool.default(true).docs({ description: 'Show the close button.' }),
	}),
	manifest: {
		slots: [
			{ name: 'default', description: 'Dialog body.' },
			{ name: 'footer', description: 'Action buttons. Elements with data-sb-close close the dialog.' },
		],
		events: [
			{ name: 'sb-open', kind: 'custom-event', bubbles: true, composed: true, description: 'After opening.' },
			{ name: 'sb-close', kind: 'custom-event', bubbles: true, composed: true, description: 'After closing. detail: { reason, value }.' },
		],
	},
	setup: ({ $$, action, adoptStyles, cleanup, defineHostProp, emit, host, observeProps, props }) => {
		adoptStyles(host, styles)
		$$.open = props.open
		observeProps(() => ($$.open = props.open), 'open')
		const show = () => {
			if ($$.open) return
			$$.open = true
			emit('sb-open')
		}
		const close = (reason = 'api', value) => {
			if (!$$.open) return
			$$.open = false
			emit('sb-close', { reason, value })
		}
		defineHostProp('show', { value: show })
		defineHostProp('close', { value: close })
		defineHostProp('isOpen', { get: () => $$.open })
		action('close', () => close('button'))

		// Footer buttons marked data-sb-close close the dialog and report
		// their value, like <form method="dialog">.
		const onClick = (e) => {
			const btn = e.target.closest?.('[data-sb-close]')
			if (btn && host.contains(btn)) close('action', btn.getAttribute('data-sb-close') || btn.textContent.trim())
		}
		host.addEventListener('click', onClick)
		cleanup(() => host.removeEventListener('click', onClick))
	},
	onFirstRender: ({ $$, effect, host, props }) => {
		// Rendered DOM exists now: hide an empty footer and wire the
		// native dialog to the open signal.
		const footer = host.shadowRoot.querySelector('footer')
		const slot = footer.querySelector('slot')
		const sync = () => (footer.hidden = slot.assignedNodes().length === 0)
		slot.addEventListener('slotchange', sync)
		sync()
		if (props.inline) return
		const dialog = host.shadowRoot.querySelector('dialog')
		effect(() => {
			if ($$.open && !dialog.open) dialog.showModal()
			else if (!$$.open && dialog.open) dialog.close()
		})
		dialog.addEventListener('cancel', (e) => {
			e.preventDefault()
			host.close('escape')
		})
		dialog.addEventListener('click', (e) => {
			if (e.target === dialog) host.close('backdrop')
		})
	},
	render: ({ html, props: { heading, inline, closable } }) => {
		const inner = html`
			<header>
				<h2 id="title" part="heading">${heading}</h2>
				${closable ? html`<button class="close" type="button" aria-label="Close" data-on:click="@close()"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" aria-hidden="true"><path d="M18 6 6 18"/><path d="m6 6 12 12"/></svg></button>` : null}
			</header>
			<div class="body" part="body"><slot></slot></div>
			<footer part="footer"><slot name="footer"></slot></footer>
		`
		return inline
			? html`<section class="panel inline" part="panel" role="group" aria-labelledby="title">${inner}</section>`
			: html`<dialog class="panel" part="panel" aria-labelledby="title">${inner}</dialog>`
	},
})
