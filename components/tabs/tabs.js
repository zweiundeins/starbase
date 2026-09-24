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

// One ElementInternals per element: attachInternals() works once, and setup
// runs again when the element is re-attached. Its custom states
// (:state(pending)) are styleable from the page and morph-proof.
const internals = new WeakMap()
const internalsOf = (host) => internals.get(host) ?? internals.set(host, host.attachInternals()).get(host)

// A panel's slot: the label lowercased, every run of anything but letters and
// digits (in any script) a hyphen. ui.tabSlot (internal/ui/install.go) is its
// Go twin and must match.
const slug = (s) => String(s).toLowerCase().replace(/[^\p{L}\p{N}]+/gu, '-').replace(/^-|-$/g, '')

const styles = /* css */ `
:host {
	--_bg: var(--sb-surface-inset, #0B1224);
	--_border: var(--sb-border, #283552);
	--_text: var(--sb-text-2, #AEBBDD);
	--_text-strong: var(--sb-text-1, #F3F4FA);
	--_active-bg: var(--sb-brand-light, #B09AFF);
	--_active-text: var(--sb-bg, #080D1D);
	--_radius: var(--sb-control-radius, 6px);
	--_focus: var(--sb-focus-ring, 0 0 0 2px #080D1D, 0 0 0 4px #B09AFF);
	display: block;
}
:host([hidden]) { display: none; }
/* A thin scrollbar only when the tabs overflow: the hint that there are more. */
[role="tablist"] {
	display: inline-flex;
	box-sizing: border-box;
	gap: 0.25rem;
	padding: 0.25rem;
	border: 1px solid var(--_border);
	border-radius: calc(var(--_radius) + 4px);
	background: var(--_bg);
	max-inline-size: 100%;
	overflow-x: auto;
	scrollbar-width: thin;
	scrollbar-color: var(--_border) transparent;
}
[role="tab"] {
	all: unset;
	padding: 0.5rem 1rem;
	border: 1px solid var(--_border);
	border-radius: var(--_radius);
	color: var(--_text);
	font-size: 0.875rem;
	font-weight: 600;
	white-space: nowrap;
	cursor: pointer;
	transition: background 120ms, color 120ms, border-color 120ms;
}
[role="tab"]:hover { color: var(--_text-strong); }
[role="tab"][aria-selected="true"] {
	border-color: var(--_active-bg);
	background: var(--_active-bg);
	color: var(--_active-text);
	box-shadow: 0 2px 0 color-mix(in oklch, var(--_active-bg) 50%, black);
}
/* After the selected rule: the focus is on the selected tab most of the time.
 * The transparent outline is what forced colors shows (it drops box-shadows). */
[role="tab"]:focus-visible { box-shadow: var(--_focus); outline: 2px solid transparent; outline-offset: 2px; }
[role="tabpanel"] { padding-block-start: 1rem; }
[role="tabpanel"]:focus-visible { outline: 2px solid var(--_active-bg); outline-offset: 4px; border-radius: 4px; }
@media (forced-colors: active) {
	[role="tab"][aria-selected="true"] { forced-color-adjust: none; background: Highlight; color: HighlightText; outline-color: CanvasText; }
}
`

rocket('sb-tabs', {
	props: ({ array, bool, number, string }) => ({
		labels: array(string.trim).default(() => ['Home', 'Docs', 'API']).docs({ description: 'Tab labels, as a JSON array.' }),
		selected: number.min(0).docs({ description: 'Index of the selected tab (past the last tab means the last one). A new index from the server wins; a removed attribute is ignored.' }),
		confirm: bool.docs({ description: 'Server-confirmed value: :state(pending) while the local selection differs from the server\'s selected attribute (see revert()).' }),
		name: string.trim.docs({ description: 'Name reported in sb-change (e.g. the field of a command).' }),
	}),
	manifest: {
		slots: [{ name: '<label-slug>', description: 'Panel content per tab: slot="docs" for a tab labelled "Docs", slot="übersicht" for "Übersicht". A slot name an earlier tab already has gets the tab\'s index appended ("c-1" for a second tab whose label gives "c").' }],
		events: [
			{ name: 'input', kind: 'event', bubbles: true, composed: true, description: 'The user moved the selection (a click or an arrow key); host.selected is the new index.' },
			{ name: 'change', kind: 'event', bubbles: true, composed: true, description: 'The selection is committed: at once for a click, after a short pause for the keyboard.' },
			{ name: 'sb-change', kind: 'custom-event', bubbles: true, composed: true, description: 'Same moment. detail: { name, value, label } (value is the index): ready for a command.' },
			{ name: 'sb-tab-change', kind: 'custom-event', bubbles: true, composed: true, description: 'Same moment. detail: { index, label }.' },
		],
	},
	setup: ({ $$, action, adoptStyles, cleanup, defineHostProp, effect, emit, host, observeProps, overrideProp, props }) => {
		adoptStyles(host, styles)
		// A selection always names a tab, so the tablist stays in the tab order.
		const clamp = (i) => Math.max(0, Math.min(i | 0, props.labels.length - 1))
		// last: the index the server has (its attribute, or the last one sent).
		// want: the index asked for (by the server, the user or a property
		// write), even past the last tab: new labels re-clamp it, so labels
		// that shrink and grow again return to it.
		let served, last, timer, want
		last = $$.selected = clamp((want = props.selected))
		// The server's last word on selected, or null while it has no opinion.
		// Only a *different* one wins, so re-sent markup keeps the user's choice.
		// A *removed* attribute is ignored: morphs also strip attributes that
		// were only reflected. The same observer copies the host's aria-label
		// onto the tablist: a label on the host can't name an element inside.
		const serverSays = () =>
			peek(() => {
				// false, not null: a null would delete the signal.
				$$.label = host.getAttribute('aria-label') || false
				if (!host.hasAttribute('selected')) return void (served = null)
				if (props.selected === served) return
				last = $$.selected = clamp((want = served = props.selected))
			})
		serverSays()
		const watch = new MutationObserver(serverSays)
		watch.observe(host, { attributeFilter: ['selected', 'aria-label'] })
		// A keyboard commit still waiting (see select) dies with the element.
		cleanup(() => (watch.disconnect(), clearTimeout(timer)))
		overrideProp('selected', () => peek(() => $$.selected), (v) => peek(() => ($$.selected = clamp((want = v)))))
		// Commands: the attribute is the server's value, $$.selected the local one.
		// With confirm, :state(pending) marks an edit the server hasn't confirmed
		// yet; revert() returns to the server's value (e.g. a rejected command).
		// New labels re-clamp the selection. sync is also the effect on
		// $$.selected: it reads $$.selected first, so the effect tracks it with
		// or without confirm (props aren't signals; its other callers peek).
		const states = internalsOf(host).states
		const sync = () => ($$.selected !== clamp(props.selected) && props.confirm ? states.add('pending') : states.delete('pending'))
		effect(sync)
		observeProps(() => peek(() => (($$.selected = clamp(want)), sync())))
		defineHostProp('revert', { value: () => peek(() => ((last = $$.selected = clamp((want = props.selected))), sync())) })
		// From a click or a timer, never inside an effect: no peek needed.
		const commit = () => {
			clearTimeout(timer)
			if ($$.selected === last) return
			last = $$.selected
			emit('change')
			emit('sb-change', { name: props.name, value: last, label: props.labels[last] })
			emit('sb-tab-change', { index: last, label: props.labels[last] })
		}
		// Selection follows focus, so a click commits at once, but arrow keys
		// commit only after a pause: every press would be a command, and the
		// server's echo of an earlier one would pull the selection back.
		// i is a tab's index, or one past either end (an arrow key), which wraps.
		const select = (i, key) => {
			const n = props.labels.length
			const next = (want = (i + n) % n)
			if (next !== $$.selected) ($$.selected = next), emit('input')
			if (!key) return commit()
			host.shadowRoot.getElementById('tab-' + next)?.focus()
			clearTimeout(timer)
			timer = setTimeout(commit, 250)
		}
		action('select', (_, i) => select(i))
		action('key', ({ evt }, i) => {
			// The strip is mirrored right to left, and so are the arrows. (The
			// computed direction: CSS can set it without a dir attribute.)
			const d = getComputedStyle(host).direction === 'rtl' ? -1 : 1
			const moves = { ArrowRight: i + d, ArrowLeft: i - d, Home: 0, End: props.labels.length - 1 }
			if (evt.key in moves) {
				evt.preventDefault()
				select(moves[evt.key], true)
			}
		})
	},
	// The selected tab scrolls into view in a strip that overflows, whoever
	// selected it. Only the strip scrolls: scrollIntoView() would move the page.
	onFirstRender: ({ $$, effect, host }) =>
		effect(() => {
			const tab = host.shadowRoot.getElementById('tab-' + $$.selected)
			const list = tab?.parentNode
			if (!list) return
			const a = tab.getBoundingClientRect()
			const b = list.getBoundingClientRect()
			// 8px: the strip's padding and border, so the focus ring shows too.
			list.scrollLeft += Math.min(a.left - b.left - 8, 0) || Math.max(a.right - b.right + 8, 0)
		}),
	render: ({ html, props: { labels } }) => {
		// Two labels can give the same slot (C++ and C#): the later one gets its
		// index (again, while another label has that slot too).
		const slots = new Set()
		return html`
			<div role="tablist" part="tablist" data-attr:aria-label="$$label">
				${labels.map((label, i) => html`
					<button
						type="button"
						role="tab"
						data-attr:part="$$selected === ${i} ? 'tab selected' : 'tab'"
						id="tab-${i}"
						aria-controls="panel-${i}"
						data-attr:aria-selected="String($$selected === ${i})"
						data-attr:tabindex="$$selected === ${i} ? 0 : -1"
						data-on:click="@select(${i})"
						data-on:keydown="@key(${i})"
					>${label}</button>
				`)}
			</div>
			${labels.map((label, i) => {
				let slot = slug(label)
				while (slots.has(slot)) slot += '-' + i
				slots.add(slot)
				return html`
					<div role="tabpanel" part="panel" id="panel-${i}" aria-labelledby="tab-${i}" tabindex="0" data-show="$$selected === ${i}">
						<slot name="${slot}"></slot>
					</div>
				`
			})}
		`
	},
})
