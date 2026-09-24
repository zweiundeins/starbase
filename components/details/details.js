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
//
// <details name="..."> would do this natively, but only within one tree scope:
// every host has its own shadow root, so no two of ours are ever in the same
// one. The registry is what makes the group work across hosts.
const groups = new Map()

// Each host's open state and the server's last word on it: { open, served }.
// Rocket drops a component's $$ when it is disconnected and reruns setup when
// it comes back, so a panel that app code moves (append, moveBefore, a portal)
// would otherwise fall back to its attribute, silently. A native <details>
// keeps its state when moved, and so does this.
const states = new WeakMap()

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
	/* Lets block-size animate to and from auto, so the panel ends at exactly
	 * its own height without anything being measured. */
	interpolate-size: allow-keywords;
}
details {
	border: 1px solid var(--_border);
	background: var(--_bg);
	color: var(--_muted);
	clip-path: ${notch('var(--_n)')};
	border-radius: calc(var(--_radius) * (1 - var(--_notch)));
}
details.disabled { opacity: 0.55; }
summary {
	display: flex;
	align-items: center;
	gap: 0.625rem;
	padding: 0.875rem;
	color: var(--_text);
	font-weight: 600;
	cursor: pointer;
	/* The UA's disclosure triangle, in all three spellings. */
	list-style: none;
	-webkit-tap-highlight-color: transparent;
}
summary::-webkit-details-marker { display: none; }
summary::marker { content: ""; }
summary:hover { background: var(--_bg-hover); }
/* An inset ring: the notched clip-path on details would cut an outer one off. */
summary:focus-visible { outline: 2px solid var(--_focus); outline-offset: -3px; }
details.disabled summary { cursor: not-allowed; }
details.disabled summary:hover { background: none; }
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
summary:hover .marker { color: var(--_text); }
details[open] .marker::before { rotate: 90deg; }
:host(:dir(rtl)) details:not([open]) .marker::before { rotate: 180deg; }
/*
 * The animation. ::details-content is the box the browser already wraps the
 * revealed content in, so the height it grows to is the content's own height.
 * content-visibility flips discretely at the ends of the transition, which is
 * what keeps the closed content out of the layout and out of the tab order
 * while still letting find-in-page reach it (and open the panel).
 *
 * Where ::details-content is unsupported the whole rule is dropped and the
 * panel simply opens at once.
 */
details::details-content {
	block-size: 0;
	overflow: hidden;
	transition: block-size var(--_dur) cubic-bezier(0.2, 0, 0, 1), content-visibility var(--_dur) allow-discrete;
}
details[open]::details-content { block-size: auto; }
.body {
	padding: 0.875rem;
	border-block-start: 1px solid var(--_border);
	font-size: 0.875rem;
}
@media (prefers-reduced-motion: reduce) {
	details::details-content, .marker::before { transition: none; }
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
		// The open state lives in `st` (see `states`) and in $$.open, which
		// data-effect writes to the inner <details> (the template renders it
		// closed, so that effect is its one writer). It is never written to the
		// *host* attribute: that one belongs to the server. The <details>
		// reflects its own state into its own open attribute, but that one is
		// inside the shadow root, where no morph can see it. The first paint
		// never animates: Rocket renders and applies in connectedCallback,
		// before any style is computed, and CSS transitions don't run on an
		// element's first style.
		let st = states.get(host)
		if (!st) states.set(host, (st = { open: props.open, served: host.hasAttribute('open') ? props.open : null }))

		// The one place the open state changes, whoever asked: the native
		// element's own toggle, host.open, show()/hide(), a sibling, the server.
		// A server change arrives in a MutationObserver microtask, inside
		// peek(): no effect is running, so a sibling's sb-toggle handler (and a
		// @post it starts) can't be tracked by one.
		const set = (want, announce = true) => {
			if (want === st.open) return
			$$.open = st.open = want
			if (want) for (const other of others()) other.close()
			if (announce) emit('sb-toggle', { name: props.name, open: want })
		}

		// --- accordion groups ----------------------------------------------
		const rec = { host, close: () => set(false) }
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
		// The rest of the group. Same document only: a host moved into another
		// document (a same-origin iframe) stays out of this one's accordions.
		const others = () => [...(groups.get(group) ?? [])].filter((other) => other !== rec && other.host.isConnected && other.host.ownerDocument === host.ownerDocument)
		join(props.group)
		cleanup(leave)
		// Exclusive from the first paint, like <details name>: a panel that
		// arrives open while another one in its group is open starts closed.
		// It asks again in a microtask, once the mutation observers (queued by
		// then: connecting a host sets attributes) have seen the rest of the
		// morph that brought it, so a morph that closes the open panel and adds
		// a new open one leaves the new one open. Both come before the first
		// style: nothing animates. (`group` is empty once it has left.)
		const clash = () => others().some((other) => other.host.open)
		if (st.open && clash()) {
			st.open = false
			queueMicrotask(() => group && !clash() && set(true, false))
		}
		$$.open = st.open

		// --- the server owns open when it says so ---------------------------
		observeProps((p) =>
			peek(() => {
				$$.summary = p.summary
				$$.icon = p.icon
				$$.disabled = p.disabled
				join(p.group)
			}),
		)

		// st.served is the server's last word on open, or null while it has no
		// opinion. Only a *different* one wins, so re-rendering the same markup
		// can never re-open a panel the user just closed, however often the
		// attribute is written.
		const serverSays = () =>
			peek(() => {
				// A *removed* attribute is ignored: morphs also strip attributes that
				// were only reflected (see sb-slider). To close, the server sends
				// open="false". A removal does clear the server's last word, so
				// sending the attribute again later counts as a change.
				if (!host.hasAttribute('open')) return void (st.served = null)
				if (props.open === st.served) return
				set((st.served = props.open), false)
			})
		// Why not observeProps: it only fires when the decoded value changes, so
		// adding open="false" to an element that had no open attribute at all
		// would go unnoticed — and that *is* the server changing its mind. The
		// instruction is the attribute, so watch the attribute. (The callback is
		// a microtask, after Rocket has decoded it and outside the morph's effect.)
		const watch = new MutationObserver(serverSays)
		watch.observe(host, { attributes: true, attributeFilter: ['open'] })
		cleanup(() => watch.disconnect())
		// Back from a move: an attribute changed while the panel was detached
		// (nothing was watching then) still wins. A no-op on the first setup.
		serverSays()

		overrideProp('open', () => st.open, (v) => peek(() => set(!!v)))
		defineHostProp('show', { value: () => peek(() => set(true)) })
		defineHostProp('hide', { value: () => peek(() => set(false)) })

		// Everything the browser opens or closes by itself lands here: the click
		// and the keyboard on <summary>, and find-in-page revealing the content.
		action('sync', ({ el }) => set(el.open))
		// <summary> has no disabled state, so the default action is what we stop.
		// It is the default action of Enter and Space on it as well.
		action('guard', ({ evt }) => $$.disabled && evt.preventDefault())
	},
	render: ({ html }) => html`
		<details
			part="details"
			data-class:disabled="$$disabled"
			data-effect="el.open !== $$open && (el.open = $$open)"
			data-on:toggle="@sync()"
		>
			<summary
				part="summary"
				id="summary"
				data-attr:aria-expanded="String($$open)"
				data-attr:aria-disabled="$$disabled ? 'true' : null"
				data-attr:tabindex="$$disabled ? '-1' : null"
				data-on:click="@guard()"
			>
				<span class="icon" part="icon" aria-hidden="true" data-show="$$icon" data-text="$$icon"></span>
				<span class="label" part="label"><slot name="summary" data-text="$$summary"></slot></span>
				<span class="marker" part="marker" aria-hidden="true"></span>
			</summary>
			<div class="body" part="content" role="region" aria-labelledby="summary"><slot></slot></div>
		</details>
	`,
})
