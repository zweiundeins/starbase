import { rocket } from 'datastar'

const styles = /* css */ `
:host {
	--_bg: var(--sb-surface-card, #141D32);
	--_inset: var(--sb-surface-inset, #0B1224);
	--_border: var(--sb-border, #283552);
	--_text: var(--sb-text-1, #F3F4FA);
	--_muted: var(--sb-text-2, #AEBBDD);
	--_brand: var(--sb-brand, #8C6BFF);
	--_radius: var(--sb-radius-lg, 10px);
	display: block;
	container-type: inline-size;
}
article {
	display: flex;
	flex-direction: column;
	block-size: 100%;
	overflow: hidden;
	border: 1px solid var(--_border);
	border-radius: var(--_radius);
	background: var(--_bg);
	color: var(--_muted);
	transition: border-color 180ms, box-shadow 180ms, translate 180ms;
}
.inset { background: var(--_inset); }
.glow { border-color: color-mix(in oklch, var(--_brand) 50%, var(--_border)); box-shadow: 0 8px 32px -12px color-mix(in oklch, var(--_brand) 60%, transparent); }
:host([href]) article:hover { border-color: color-mix(in oklch, var(--_brand) 60%, var(--_border)); translate: 0 -2px; }
.media { display: grid; }
.media[hidden], .footer[hidden], .heading[hidden] { display: none; }
::slotted([slot="media"]) { display: block; inline-size: 100%; block-size: auto; image-rendering: pixelated; }
.body { display: grid; gap: 0.375rem; padding: 1rem; }
.body:empty { display: none; }
.heading { margin: 0; color: var(--_text); font-size: 1rem; font-weight: 700; }
.heading a { color: inherit; text-decoration: none; }
.heading a::after { content: ""; position: absolute; inset: 0; }
:host([href]) article { position: relative; }
.footer { display: flex; align-items: center; gap: 0.5rem; padding: 0.75rem 1rem; border-block-start: 1px solid var(--_border); }
@container (width < 14rem) { .body { padding: 0.75rem; } }
`

rocket('sb-card', {
	props: ({ oneOf, string }) => ({
		heading: string.trim.docs({ description: 'Card title.' }),
		href: string.trim.docs({ description: 'Makes the whole card a link.' }),
		variant: oneOf('default', 'inset', 'glow').default('default').docs({ description: 'Surface style.' }),
	}),
	manifest: {
		slots: [
			{ name: 'media', description: 'Image or illustration on top, edge to edge.' },
			{ name: 'default', description: 'Body content.' },
			{ name: 'footer', description: 'Actions or metadata at the bottom.' },
		],
	},
	setup: ({ adoptStyles, host }) => adoptStyles(host, styles),
	// Hide slot wrappers that received no content, so empty areas take no space.
	onFirstRender: ({ cleanup, host }) => {
		const sync = () => {
			for (const slot of host.shadowRoot.querySelectorAll('slot[name]')) {
				slot.parentElement.hidden = slot.assignedNodes({ flatten: true }).length === 0
			}
		}
		sync()
		host.shadowRoot.addEventListener('slotchange', sync)
		cleanup(() => host.shadowRoot.removeEventListener('slotchange', sync))
	},
	render: ({ html, props: { heading, href, variant } }) => html`
		<article class="${variant}" part="card">
			<div class="media" part="media"><slot name="media"></slot></div>
			<div class="body" part="body">
				${heading ? html`<h3 class="heading">${href ? html`<a href="${href}">${heading}</a>` : heading}</h3>` : null}
				<slot></slot>
			</div>
			<div class="footer" part="footer"><slot name="footer"></slot></div>
		</article>
	`,
})
