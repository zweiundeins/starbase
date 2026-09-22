// A plain custom element (no Datastar, no Rocket) for the docs: it exists to
// be loaded on demand by <sb-autoloader>, and shows when it arrived. ?tag=
// names the element, so a second example can load a tag of its own.
const css = /* css */ `
:host {
	display: inline-flex;
	align-items: center;
	gap: 0.5rem;
	padding: 0.35rem 0.6rem;
	border: 1px solid var(--sb-ok, #6EF59A);
	border-radius: var(--sb-radius-sm, 6px);
	background: var(--sb-surface-raised, #10182B);
	color: var(--sb-text-1, #F3F4FA);
	font-size: 0.8125rem;
}
b { color: var(--sb-ok, #6EF59A) }
`

const tag = new URL(import.meta.url).searchParams.get('tag') || 'demo-badge'

customElements.define(
	tag,
	class extends HTMLElement {
		connectedCallback() {
			if (this.shadowRoot) return
			const root = this.attachShadow({ mode: 'open' })
			const sheet = new CSSStyleSheet()
			sheet.replaceSync(css)
			root.adoptedStyleSheets = [sheet]
			root.innerHTML = '<b>✔</b><span><slot>loaded on demand</slot></span>'
		}
	},
)
