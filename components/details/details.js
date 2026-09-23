import { rocket, startPeeking, stopPeeking } from 'datastar'

// Host getters must not subscribe their caller (e.g. data-bind's sync effect)
// to the internal signal, and attribute changes arrive inside the effect of
// whoever set them: reading signals there must not subscribe that effect.
const peek = (fn) => {
	startPeeking()
	try {
		return fn()
	} finally {
		stopPeeking()
	}
}

// Pixel corners: notches every corner by p (3px times --sb-notch; at 0 the
// border-radius takes over).
const notch = (p) => `polygon(${p} 0, calc(100% - ${p}) 0, calc(100% - ${p}) ${p}, 100% ${p}, 100% calc(100% - ${p}), calc(100% - ${p}) calc(100% - ${p}), calc(100% - ${p}) 100%, ${p} 100%, ${p} calc(100% - ${p}), 0 calc(100% - ${p}), 0 ${p}, ${p} ${p})`

// Accordion groups: group name -> the records in it. A plain Map of plain Sets
// on purpose: assigning an object to a signal *merges* into it, so keys would
// never go away, and none of this is rendered anyway.
const groups = new Map()

// Two frames: the panel has to be laid out (hidden removed) and seen at 0fr
// before the row grows, or the browser skips the transition.
const next = (fn) => requestAnimationFrame(() => requestAnimationFrame(fn))

// The longest duration in a transition-duration value, in milliseconds.
const ms = (v) => Math.max(0, ...String(v).split(',').map((s) => (parseFloat(s) || 0) * (s.includes('ms') ? 1 : 1000)))

const styles = /* css */ `
:host {
	--_bg: var(--sb-surface-card, #141D32);
	--_bg-hover: var(--sb-surface-hover, #1A2540);
	--_border: var(--sb-border, #283552);
	--_text: var(--sb-text-1, #F3F4FA);
	--_muted: var(--sb-text-2, #AEBBDD);
	--_marker: var(--sb-text-muted, #7785A8);
	--_focus: var(--sb-brand-light, #B09AFF);
	--_radius: var(--sb-radius, 8px);
	--_notch: var(--sb-notch, 1);
	--_n: calc(3px * var(--_notch));
	--_dur: var(--sb-details-duration, 220ms);
	display: block;
}
:host([disabled]) { opacity: 0.55; }
.details {
	border: 1px solid var(--_border);
	background: var(--_bg);
	color: var(--_muted);
	clip-path: ${notch('var(--_n)')};
	border-radius: calc(var(--_radius) * (1 - var(--_notch)));
}
.summary {
	all: unset;
	box-sizing: border-box;
	display: flex;
	align-items: center;
	gap: 0.625rem;
	inline-size: 100%;
	padding: 0.75rem 0.875rem;
	color: var(--_text);
	font: inherit;
	font-weight: 600;
	text-align: start;
	cursor: pointer;
}
.summary:hover { background: var(--_bg-hover); }
/* An inset ring: the notched clip-path on .details would cut an outer one off. */
.summary:focus-visible { outline: 2px solid var(--_focus); outline-offset: -3px; }
.summary[disabled] { cursor: not-allowed; }
.summary[disabled]:hover { background: none; }
.icon { flex: none; }
.label { flex: 1; min-inline-size: 0; }
/* A pixel triangle that turns a quarter when the panel opens. */
.marker {
	flex: none;
	display: grid;
	place-items: center;
	inline-size: 1rem;
	block-size: 1rem;
	color: var(--_marker);
}
.marker::before {
	content: "";
	inline-size: 6px;
	block-size: 8px;
	background: currentColor;
	clip-path: polygon(0 0, 2px 0, 2px 1px, 4px 1px, 4px 3px, 6px 3px, 6px 5px, 4px 5px, 4px 7px, 2px 7px, 2px 8px, 0 8px);
	transition: rotate var(--_dur) steps(3, end);
}
.summary:hover .marker { color: var(--_text); }
.open > .summary .marker::before { rotate: 90deg; }
/*
 * The animation. The panel is a one-row grid that grows from 0fr to 1fr, so
 * the open height is the content's own height: nothing is measured, and the
 * transition always ends exactly right. hidden="until-found" keeps the closed
 * content out of the layout and out of the tab order, and still lets the
 * browser's find-in-page reach it (it fires beforematch, and we open).
 *
 * display comes from :not([hidden]) so the UA's [hidden] rule still wins where
 * until-found is unsupported; an author "display: grid" would beat it.
 */
.panel:not([hidden]) { display: grid; }
.panel {
	grid-template-rows: 0fr;
	transition: grid-template-rows var(--_dur) cubic-bezier(0.2, 0, 0, 1);
}
.panel.open { grid-template-rows: 1fr; }
.panel.instant { transition: none; }
/* The row is the only thing with a height: the content is clipped while it grows. */
.clip { overflow: hidden; }
.body {
	padding: 0.25rem 0.875rem 0.875rem;
	border-block-start: 1px solid transparent;
	font-size: 0.875rem;
}
.open > .panel .body { border-block-start-color: var(--_border); }
@media (prefers-reduced-motion: reduce) {
	.panel, .marker::before { transition: none; }
}
`

rocket('sb-details', {
	props: ({ bool, string }) => ({
		summary: string.trim.default('Details').docs({ description: 'The heading on the summary row. The server can morph it; the summary slot overrides it.' }),
		open: bool.docs({ description: 'Open. View state, not a value: a changed attribute from the server wins, a removed one is ignored (send open="false" to close). Local toggling never reflects it.' }),
		icon: string.trim.docs({ description: 'Short text or emoji before the summary (decorative).' }),
		disabled: bool.docs({ description: 'Block toggling. The server can still open or close it.' }),
		group: string.trim.docs({ description: 'Accordion group: opening one sb-details with this group name closes the others, and each of them emits its own sb-toggle.' }),
		name: string.trim.docs({ description: 'Name reported in sb-toggle (e.g. the field of a command).' }),
	}),
	manifest: {
		slots: [
			{ name: 'default', description: 'The content the panel reveals.' },
			{ name: 'summary', description: 'Rich content for the summary row; replaces the summary prop.' },
		],
		events: [
			{ name: 'sb-toggle', kind: 'custom-event', bubbles: true, composed: true, description: 'The panel opened or closed. detail: { name, open }. View state: never pending, and not emitted for a change the server itself made.' },
		],
	},
	// Rendered once: the summary row holds the keyboard focus and the panel
	// holds the running animation, so every prop drives a signal instead of a
	// re-render.
	renderOnPropChange: false,
	setup: ({ $$, action, adoptStyles, cleanup, defineHostProp, emit, host, observeProps, overrideProp, props }) => {
		adoptStyles(host, styles)
		$$.summary = props.summary
		$$.icon = props.icon
		$$.disabled = props.disabled
		// open: what the user (or the server) asked for.
		// rendered: whether the panel takes part in the layout at all; it stays
		// on until a closing animation has finished.
		$$.open = props.open
		$$.rendered = props.open
		// The first paint never animates: an initially open panel is simply open.
		$$.instant = true
		next(() => ($$.instant = false))

		const panel = () => host.shadowRoot?.querySelector('.panel')
		const duration = () => {
			const el = panel()
			return el ? ms(getComputedStyle(el).transitionDuration) : 0
		}
		let timer = 0
		const stop = () => {
			clearTimeout(timer)
			timer = 0
		}
		cleanup(stop)

		const toggled = (open, defer) => {
			const fire = () => emit('sb-toggle', { name: props.name, open })
			// A change that came from the server arrives during a morph, and a
			// @post the page starts in the handler would be tracked by whatever
			// effect is running: let it out in a later task instead.
			if (defer) setTimeout(fire)
			else fire()
		}

		// The one place the open state changes, whoever asked: the click, the
		// keyboard, host.open, show()/hide(), a group sibling, the server.
		const set = (want, { announce = true, defer = false, instant = false } = {}) => {
			if (want === $$.open) return
			stop()
			const d = instant ? 0 : duration()
			if (want) {
				$$.rendered = true
				if (d <= 0) {
					// Nothing to animate (reduced motion, or a find-in-page reveal):
					// suppress the transition for this frame instead of waiting for it.
					$$.instant = true
					$$.open = true
					next(() => ($$.instant = false))
				} else {
					next(() => $$.rendered && ($$.open = true))
				}
				closeSiblings(defer)
			} else {
				$$.open = false
				// Keep the panel in the layout until the row has shrunk. transitionend
				// normally gets there first; this is the fallback when it never fires.
				if (d <= 0) $$.rendered = false
				else timer = setTimeout(() => ($$.rendered = false), d + 60)
			}
			if (announce) toggled(want, defer)
		}

		// --- accordion groups ----------------------------------------------
		const rec = { host, close: (defer) => set(false, { defer }) }
		let group = ''
		const leave = () => {
			const members = groups.get(group)
			members?.delete(rec)
			if (members && !members.size) groups.delete(group)
			group = ''
		}
		const join = (name) => {
			if ((name || '') === group) return
			if (group) leave()
			group = name || ''
			if (!group) return
			if (!groups.has(group)) groups.set(group, new Set())
			groups.get(group).add(rec)
		}
		const closeSiblings = (defer) => {
			if (!group) return
			for (const other of [...(groups.get(group) ?? [])]) {
				// Same document only: two previews on one page shouldn't fight over
				// a group name, and a runner iframe is its own world.
				if (other !== rec && other.host.isConnected && other.host.ownerDocument === host.ownerDocument) other.close(defer)
			}
		}
		join(props.group)
		cleanup(leave)

		// --- the server owns open when it says so ---------------------------
		observeProps((p) =>
			peek(() => {
				$$.summary = p.summary
				$$.icon = p.icon
				$$.disabled = p.disabled
				join(p.group)
			}),
		)

		// The server's last word on open, or null while it has no opinion. Only a
		// *different* one wins, so re-rendering the same markup can never re-open
		// a panel the user just closed, however often the attribute is written.
		let served = host.hasAttribute('open') ? props.open : null
		const serverSays = () =>
			peek(() => {
				// A *removed* attribute is ignored: morphs also strip attributes that
				// were only reflected (see sb-slider). To close, the server sends
				// open="false". A removal does clear the server's last word, so
				// sending the attribute again later counts as a change.
				if (!host.hasAttribute('open')) return void (served = null)
				if (props.open === served) return
				served = props.open
				set(props.open, { announce: false, defer: true })
			})
		// Why not observeProps: it only fires when the decoded value changes, so
		// adding open="false" to an element that had no open attribute at all
		// would go unnoticed — and that *is* the server changing its mind. The
		// instruction is the attribute, so watch the attribute. (The callback is
		// a microtask, after Rocket has decoded it and outside the morph's effect.)
		const watch = new MutationObserver(serverSays)
		watch.observe(host, { attributes: true, attributeFilter: ['open'] })
		cleanup(() => watch.disconnect())

		overrideProp('open', () => peek(() => $$.open), (v) => peek(() => set(!!v)))
		defineHostProp('show', { value: () => peek(() => set(true)) })
		defineHostProp('hide', { value: () => peek(() => set(false)) })

		// A real <button> brings Enter and Space with it.
		action('toggle', () => !$$.disabled && set(!$$.open))
		action('settled', ({ el, evt }) => {
			if (evt.target !== el || evt.propertyName !== 'grid-template-rows') return
			stop()
			if (!$$.open) $$.rendered = false
		})
		// Find-in-page hit the closed content: the browser is about to reveal it
		// and scroll there, so open at once instead of scrolling to a growing box.
		action('found', () => set(true, { instant: true }))
	},
	render: ({ html }) => html`
		<div class="details" part="details" data-class:open="$$open">
			<button
				type="button"
				class="summary"
				part="summary"
				id="summary"
				aria-controls="panel"
				data-attr:aria-expanded="String($$open)"
				data-attr:disabled="$$disabled ? 'disabled' : null"
				data-on:click="@toggle()"
			>
				<span class="icon" part="icon" aria-hidden="true" data-show="$$icon" data-text="$$icon"></span>
				<span class="label" part="label"><slot name="summary" data-text="$$summary"></slot></span>
				<span class="marker" part="marker" aria-hidden="true"></span>
			</button>
			<div
				class="panel"
				part="panel"
				id="panel"
				role="region"
				aria-labelledby="summary"
				data-attr:hidden="$$rendered ? null : 'until-found'"
				data-class:open="$$open"
				data-class:instant="$$instant"
				data-on:transitionend="@settled()"
				data-on:beforematch="@found()"
			>
				<div class="clip">
					<div class="body" part="content"><slot></slot></div>
				</div>
			</div>
		</div>
	`,
})
