import { rocket } from 'datastar'

// Loads by tag (a promise, or 0 once it failed), shared by every instance:
// two autoloaders on one page (or the same one after a morph) never fetch a
// module twice, and each one waits for the loads it finds, whoever started
// them.
const loads = new Map()
// The browser's module map remembers a failed import by URL, so a retry
// imports it under a new #fragment (which the HTTP cache ignores). The
// fragment counts failures, not retries: tags that share a module and
// come back together still share one fetch and one evaluation.
let failures = 0
// Per host, what outlives a re-attach (Rocket reruns setup on every
// connect): sb-ready fires once, and el.ready stays the same promise.
const hosts = new WeakMap()

// The element adds nothing to the page: no box of its own, and any children
// render exactly where they are written (so it can wrap markup, or stand
// alone as a single tag).
const styles = /* css */ `
:host { display: contents }
`

rocket('sb-autoloader', {
	props: ({ json, number, string }) => ({
		modules: json.default(() => ({})).docs({ description: 'Explicit map of tag to module URL: {"x-chart": "/js/x-chart.js"}. Wins over pattern.' }),
		pattern: string.trim.docs({ description: 'URL template for tags not in modules, with {tag}, e.g. "/components/{tag}/{tag}.js".' }),
		match: string.trim.docs({ description: 'Regular expression the tag must match before pattern applies, e.g. "^x-". Empty: every unknown custom element.' }),
		requires: json.default(() => ({})).docs({ description: 'Tags a component renders itself, loaded with it: {"x-table": ["x-cell"]}.' }),
		base: string.trim.docs({ description: 'Base for relative URLs. Default: the document\'s base URL.' }),
		cloak: string.trim.docs({ description: 'Class removed from <html> once the components present at startup have loaded or failed, or after timeout (against the flash of undefined elements).' }),
		timeout: number.min(0).default(3000).docs({ description: 'Remove the cloak class after this many ms, whatever happens (sb-ready still waits for the loads).' }),
	}),
	manifest: {
		slots: [{ name: 'default', description: 'Optional markup to wrap; it renders exactly as written.' }],
		events: [
			{ name: 'sb-load', kind: 'custom-event', bubbles: true, composed: true, description: 'After a component is defined. detail: { tag, url }.' },
			{ name: 'sb-load-error', kind: 'custom-event', bubbles: true, composed: true, description: 'When a module fails to load, or loads without defining its tag. detail: { tag, url, error }.' },
			{ name: 'sb-ready', kind: 'custom-event', bubbles: true, composed: true, description: 'Once, when every component present at startup has loaded or failed (right away when there were none). detail: { loaded }.' },
		],
	},
	setup: ({ adoptStyles, cleanup, defineHostProp, emit, host, props }) => {
		adoptStyles(host, styles)
		const uncloak = () => props.cloak && document.documentElement.classList.remove(props.cloak)
		let s = hosts.get(host)
		if (!s) {
			hosts.set(host, (s = { loaded: 0, pending: 0 }))
			s.ready = new Promise((resolve) => (s.done = resolve))
			// Never leave a page cloaked, even when the loader is gone by then.
			setTimeout(uncloak, props.timeout)
		}

		// match, compiled once per value. A broken one is reported and matches
		// nothing, so the pattern is skipped.
		let src, re
		const matches = (tag) => {
			if (src !== props.match) {
				src = props.match
				try {
					re = new RegExp(src) // empty: every tag
				} catch (cause) {
					re = null
					reportError(new Error(`<sb-autoloader> match="${src}" is not a regular expression`, { cause }))
				}
			}
			return re?.test(tag)
		}

		// Where a tag's module lives: the explicit map first, then the pattern
		// (only for tags that match), resolved against base.
		const urlFor = (tag) => {
			const from = props.modules?.[tag] || (props.pattern && matches(tag) ? props.pattern.replaceAll('{tag}', tag) : '')
			return from && new URL(from, new URL(props.base, document.baseURI)).href
		}

		// After the microtasks, so tags that appear right after a module runs
		// (a component rendering its own children) still count.
		const settle = () =>
			setTimeout(() => {
				if (s.pending || !s.done) return
				uncloak()
				s.done(s.loaded)
				s.done = 0
				emit('sb-ready', { loaded: s.loaded })
			})
		const done = () => (s.pending--, settle())

		const load = (tag) => {
			if (customElements.get(tag)) return
			let p = loads.get(tag)
			if (!p) {
				const url = urlFor(tag)
				if (!url) return
				loads.set(
					tag,
					(p = import(loads.has(tag) ? `${url}#${failures}` : url)
						.then(() => {
							// sb-autoloader runs once Datastar is ready, so a Rocket module, like
							// any other, defines its tag while it runs.
							if (!customElements.get(tag)) throw new Error(`${url} does not define <${tag}>`)
							s.loaded++
							emit('sb-load', { tag, url })
						})
						.catch((cause) => {
							loads.set(tag, 0) // failed: a later attempt may succeed
							failures++
							emit('sb-load-error', { tag, url, error: String(cause?.message ?? cause) })
							reportError(new Error(`<sb-autoloader> could not load <${tag}> from ${url}`, { cause }))
							throw cause
						})),
				)
				for (const dep of props.requires?.[tag] ?? []) load(dep) // after set: a cycle ends here
			}
			s.pending++
			p.then(done, done)
			return p
		}

		// Undefined custom elements in a tree (an element, the document, or a
		// shadow root), including the root itself.
		const discover = (root) =>
			Promise.allSettled([root, ...(root.querySelectorAll?.(':not(:defined)') ?? [])].map((el) => el.localName?.includes('-') && load(el.localName)))

		discover(document.documentElement)
		settle() // nothing to load: ready right away
		// Tags that appear later (a Datastar morph, a template, any script).
		const observer = new MutationObserver((records) => {
			for (const r of records) for (const n of r.addedNodes) if (n.nodeType === 1) discover(n)
		})
		observer.observe(document.documentElement, { subtree: true, childList: true })
		cleanup(() => observer.disconnect())

		defineHostProp('ready', { value: s.ready })
		defineHostProp('load', { value: load })
		defineHostProp('discover', { value: discover })
	},
	render: ({ html }) => html`<slot></slot>`,
})
