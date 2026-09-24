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
	--_inner: calc(var(--_radius) - 1px); /* inside the 1px border */
	display: block;
}
:host([hidden]) { display: none; }
article {
	display: flex;
	flex-direction: column;
	block-size: 100%;
	border: 1px solid var(--_border);
	border-radius: var(--_radius);
	background: var(--_bg);
	color: var(--_muted);
	transition: border-color 180ms, box-shadow 180ms, translate 180ms;
}
.inset { background: var(--_inset); }
.glow { border-color: color-mix(in oklch, var(--_brand) 50%, var(--_border)); box-shadow: 0 8px 32px -12px color-mix(in oklch, var(--_brand) 60%, transparent); }
/* A linked card: hover and keyboard focus on its one link light up the whole card. */
article:has(a:is(:hover, :focus-visible)) { border-color: color-mix(in oklch, var(--_brand) 60%, var(--_border)); }
article:has(a:focus-visible) { box-shadow: var(--sb-focus-ring, 0 0 0 2px #080D1D, 0 0 0 4px #B09AFF); outline: 2px solid transparent; outline-offset: 2px; }
@media (prefers-reduced-motion: no-preference) { article:has(a:is(:hover, :focus-visible)) { translate: 0 -2px; } }
/* Only the media is clipped to the corners (all four when nothing follows it),
   so tooltips and badges can leave the card. */
.media { display: grid; overflow: hidden; border-radius: var(--_inner) var(--_inner) 0 0; }
.media.end { border-radius: var(--_inner); }
::slotted([slot="media"]) { display: block; inline-size: 100%; block-size: auto; }
/* 0.75rem under 14rem, without a size container, which would leave the card
   no width of its own in flex rows and fit-content layouts. 100% is the
   card's inner width (14rem - 2px at the switch). While the browser measures
   the content it counts as 0, which gives 1rem, so text never wraps early. */
.body { display: grid; gap: 0.375rem; padding: clamp(0.75rem, 1rem + 99 * max(-100%, 100% + 2px - 14rem), 1rem); }
/* A block, so inline markup in the body stays in one flow. */
.body > slot { display: block; }
.heading { margin: 0; color: var(--_text); font-size: 1rem; font-weight: 700; }
.heading a { color: inherit; text-decoration: none; outline: none; }
.heading a::after { content: ""; position: absolute; inset: 0; }
/* Links and buttons in the body and everything in the footer rise above the stretched link. */
article:has(a), article:has(a) ::slotted(:is(a, button, [slot="footer"])) { position: relative; }
.footer { display: flex; align-items: center; gap: 0.5rem; margin-block-start: auto; padding: 0.75rem 1rem; border-block-start: 1px solid var(--_border); }
`

rocket('sb-card', {
	props: ({ oneOf, string }) => ({
		heading: string.trim.docs({ description: 'Card title.' }),
		href: string.trim.docs({ description: 'Makes the whole card a link: the heading is the link, so it needs `heading`.' }),
		variant: oneOf('default', 'inset', 'glow').default('default').docs({ description: 'Surface style.' }),
	}),
	manifest: {
		slots: [
			{ name: 'media', description: 'Image or illustration on top, edge to edge.' },
			{ name: 'default', description: 'Body content.' },
			{ name: 'footer', description: 'Actions or metadata at the bottom.' },
		],
	},
	setup: ({ $$, action, adoptStyles, cleanup, host }) => {
		adoptStyles(host, styles)
		// Which slots received content (whitespace doesn't count); empty sections collapse.
		$$.media = $$.body = $$.footer = false
		const slots = () => {
			for (const slot of host.shadowRoot.querySelectorAll('slot')) {
				$$[slot.name || 'body'] = slot.assignedNodes({ flatten: true }).some((n) => n.nodeType != 3 || n.data.trim())
			}
		}
		action('slots', slots)
		// A morph rewrites a text node in place, without a slotchange.
		const watch = new MutationObserver(slots)
		watch.observe(host, { characterData: true, subtree: true })
		cleanup(() => watch.disconnect())
	},
	render: ({ html, props: { heading, href, variant } }) => html`
		<article class="${variant}" part="card" data-init="@slots()" data-on:slotchange="@slots()">
			<div class="media" part="media" data-show="$$media" data-class:end="${!heading} && !$$body && !$$footer"><slot name="media"></slot></div>
			<div class="body" part="body" data-show="${heading ? 'true' : '$$body'}">
				${heading ? html`<h3 class="heading" part="heading">${href ? html`<a href="${href}">${heading}</a>` : heading}</h3>` : null}
				<slot data-show="$$body"></slot>
			</div>
			<div class="footer" part="footer" data-show="$$footer"><slot name="footer"></slot></div>
		</article>
	`,
})
