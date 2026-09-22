import { rocket } from 'datastar'

// Tags already being loaded, shared by every instance: two autoloaders on one
// page (or the same one after a morph) never fetch a module twice.
const started = new Set()

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
		cloak: string.trim.docs({ description: 'Class removed from <html> once the first components are defined (against the flash of undefined elements).' }),
		timeout: number.min(0).default(3000).docs({ description: 'Remove the cloak class after this many ms, whatever happens.' }),
	}),
	manifest: {
		events: [
			{ name: 'sb-load', kind: 'custom-event', bubbles: true, composed: true, description: 'After a component is defined. detail: { tag, url }.' },
			{ name: 'sb-load-error', kind: 'custom-event', bubbles: true, composed: true, description: 'When a module fails to load. detail: { tag, url, error }.' },
			{ name: 'sb-ready', kind: 'custom-event', bubbles: true, composed: true, description: 'Once, when the components present at startup are defined (right away when there were none). detail: { loaded }.' },
		],
	},
	setup: ({ adoptStyles, cleanup, defineHostProp, emit, host, props }) => {
		adoptStyles(host, styles)
		let pending = 0
		let loaded = 0
		let settled = false
		let markReady
		const ready = new Promise((resolve) => (markReady = resolve))

		const report = (err) => (typeof reportError === 'function' ? reportError(err) : console.error(err))

		// match, compiled once per value. A broken pattern disables the fallback
		// instead of throwing on every tag.
		let re, reSrc, reBad
		const matches = (tag) => {
			if (!props.match) return true
			if (reSrc !== props.match) {
				reSrc = props.match
				try {
					;(re = new RegExp(props.match)), (reBad = false)
				} catch (cause) {
					re = null
					if (!reBad) report(new Error(`<sb-autoloader> match="${props.match}" is not a regular expression`, { cause }))
					reBad = true
				}
			}
			return !!re?.test(tag)
		}

		// Where a tag's module lives: the explicit map first, then the pattern
		// (only for tags that match), resolved against base.
		const urlFor = (tag) => {
			const from = props.modules?.[tag] || (props.pattern && matches(tag) ? props.pattern.replaceAll('{tag}', tag) : '')
			if (!from) return ''
			const base = props.base ? new URL(props.base, document.baseURI) : document.baseURI
			return new URL(from, base).href
		}

		const uncloak = () => {
			if (settled) return
			settled = true
			if (props.cloak) document.documentElement.classList.remove(props.cloak)
			markReady(loaded)
			emit('sb-ready', { loaded })
		}
		// A microtask and a frame, so tags that appear right after a module
		// runs (a component rendering its own children) still count.
		const settle = () => queueMicrotask(() => pending === 0 && requestAnimationFrame(() => pending === 0 && uncloak()))
		const timer = setTimeout(uncloak, props.timeout) // never leave a page cloaked

		const load = (tag) => {
			if (started.has(tag) || customElements.get(tag)) return
			const url = urlFor(tag)
			if (!url) return
			started.add(tag)
			for (const dep of props.requires?.[tag] ?? []) load(dep)
			pending++
			import(url)
				// Rocket defines its elements once Datastar is ready: wait for that too.
				.then(() => customElements.whenDefined(tag))
				.then(() => {
					loaded++
					emit('sb-load', { tag, url })
				})
				.catch((cause) => {
					started.delete(tag) // a later attempt may succeed
					emit('sb-load-error', { tag, url, error: String(cause?.message ?? cause) })
					report(new Error(`<sb-autoloader> could not load <${tag}> from ${url}`, { cause }))
				})
				.finally(() => {
					pending--
					settle()
				})
		}

		// Undefined custom elements in a tree (an element, the document, or a
		// shadow root), including the root itself.
		const discover = (root) => {
			if (root.localName?.includes('-')) load(root.localName)
			for (const el of root.querySelectorAll?.(':not(:defined)') ?? []) load(el.localName)
		}

		discover(document.documentElement)
		settle() // nothing to load: ready right away
		// Tags that appear later (a Datastar morph, a template, any script).
		const observer = new MutationObserver((records) => {
			for (const r of records) for (const n of r.addedNodes) if (n.nodeType === 1) discover(n)
		})
		observer.observe(document.documentElement, { subtree: true, childList: true })
		cleanup(() => {
			observer.disconnect()
			clearTimeout(timer)
		})

		defineHostProp('ready', { get: () => ready })
		defineHostProp('load', { value: load })
		defineHostProp('discover', { value: discover })
	},
	render: ({ html }) => html`<slot></slot>`,
})
