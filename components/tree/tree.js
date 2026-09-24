import { rocket, startPeeking, stopPeeking } from 'datastar'

// Host getters must not subscribe callers (e.g. data-bind's sync effect).
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

const styles = /* css */ `
:host {
	--_text: var(--sb-text-1, #F3F4FA);
	--_muted: var(--sb-text-muted, #7785A8);
	--_hover: var(--sb-surface-hover, #1A2540);
	--_sel: var(--sb-brand-subtle, rgb(140 107 255 / 0.14));
	--_brand: var(--sb-brand, #8C6BFF);
	--_focus: var(--sb-brand-light, #B09AFF);
	--_indent: 1.25rem;
	display: block;
	color: var(--_text);
	font-size: 0.875rem;
}
:host([hidden]) { display: none; }
[role="tree"] { display: grid; gap: 1px; outline: none; }
[role="treeitem"] {
	display: flex;
	align-items: center;
	gap: 0.4rem;
	min-block-size: 2rem;
	padding-inline: calc(var(--depth) * var(--_indent) + 0.25rem) 0.5rem;
	border-radius: var(--sb-radius-sm, 6px);
	cursor: pointer;
	user-select: none;
}
[role="treeitem"]:hover { background: var(--_hover); }
[role="treeitem"][aria-selected="true"] { background: var(--_sel); box-shadow: inset 2px 0 0 var(--_brand); }
[role="treeitem"]:focus-visible { outline: 2px solid var(--_focus); outline-offset: -2px; }
.caret { display: grid; place-items: center; inline-size: 1rem; block-size: 1rem; flex: none; color: var(--_muted); }
/* A pixel triangle that turns when open. */
.caret::before {
	content: "";
	inline-size: 6px;
	block-size: 8px;
	background: currentColor;
	clip-path: polygon(0 0, 2px 0, 2px 1px, 4px 1px, 4px 3px, 6px 3px, 6px 5px, 4px 5px, 4px 7px, 2px 7px, 2px 8px, 0 8px);
	transition: rotate 120ms steps(2);
}
[aria-expanded="true"] > .caret::before { rotate: 90deg; }
:not([aria-expanded]) > .caret::before { display: none; }
.icon { flex: none; inline-size: 1.1rem; text-align: center; }
.label { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.busy { color: var(--_muted); font-size: 0.75rem; }
.busy::after { content: "…"; animation: dots 1s steps(3) infinite; display: inline-block; inline-size: 1.2em; overflow: hidden; vertical-align: bottom; }
@keyframes dots { from { inline-size: 0 } }
@media (prefers-reduced-motion: reduce) { .caret::before, .busy::after { transition: none; animation: none; } }
`

rocket('sb-tree', {
	props: ({ bool, json, oneOf, string }) => ({
		items: json.default(() => []).docs({ description: 'The tree: [{id, label, icon?, children?: [...], lazy?: true}]. A lazy item without children asks for them with sb-load when opened.' }),
		loaded: json.default(() => ({})).docs({ description: 'Loaded children by parent id: {"<id>": [items]}. Bind it to a signal the server patches (see the docs).' }),
		selection: oneOf('single', 'multiple', 'none').default('single').docs({ description: 'How many items can be selected.' }),
		value: string.docs({ description: 'Selection: an id, or ids separated by spaces (multiple). A new value from the server replaces it; the live value is the value property.' }),
		expanded: string.docs({ description: 'Open items: ids separated by spaces. A new list from the server replaces it; lazy items in it load their children.' }),
		label: string.trim.default('Tree').docs({ description: 'Accessible name.' }),
		confirm: bool.docs({ description: 'Server-confirmed value: :state(pending) while the local value differs from the server\'s value attribute (see revert()).' }),
		name: string.trim.docs({ description: 'Name reported in sb-change (e.g. the field of a command).' }),
	}),
	manifest: {
		events: [
			{ name: 'sb-load', kind: 'custom-event', bubbles: true, composed: true, description: 'A lazy item was opened and has no children yet. detail: { id }. Answer by adding its children to the loaded prop.' },
			{ name: 'change', kind: 'event', bubbles: true, composed: true, description: 'The selection changed.' },
			{ name: 'sb-change', kind: 'custom-event', bubbles: true, composed: true, description: 'The selection changed. detail: { name, value } (an id, or an array of ids for multiple): ready for a command.' },
			{ name: 'sb-toggle', kind: 'custom-event', bubbles: true, composed: true, description: 'An item opened or closed. detail: { id, open }.' },
		],
	},
	// Rendered once: everything that changes goes through signals, so a new
	// `loaded` doesn't rebuild the DOM (and take the keyboard focus with it).
	renderOnPropChange: false,
	setup: ({ $$, action, adoptStyles, cleanup, defineHostProp, effect, emit, host, observeProps, overrideProp, props }) => {
		adoptStyles(host, styles)
		$$.label = props.label
		$$.mode = props.selection
		const ids = (v) => (Array.isArray(v) ? v.map(String) : String(v ?? '').split(/\s+/).filter(Boolean))
		$$.selected = ids(props.value)
		// Plain sets, not signals: assigning an object to a signal merges into
		// it (keys never go away), and only $$.rows is rendered anyway.
		const open = new Set(ids(props.expanded))
		const loading = new Set() // ids whose children are on their way
		$$.focus = '' // the row in the tab order (roving tabindex)
		$$.hasFocus = false // whether keyboard focus is inside the tree
		$$.rows = []

		// Move DOM focus to the focus row, also after the rows re-render (they
		// are new elements then, e.g. when a branch opens).
		const refocus = () =>
			requestAnimationFrame(() => {
				if (!$$.hasFocus) return
				// Only pick the focus back up when it fell on the floor (the row was
				// replaced or dropped): if the user moved on to something else, leave
				// it alone.
				const active = document.activeElement
				if (active && active !== document.body && active !== host) return
				const el = host.shadowRoot.querySelector(`[data-id="${CSS.escape($$.focus)}"]`)
				if (el && host.shadowRoot.activeElement !== el) el.focus()
			})

		// The visible rows: a depth-first walk through the open items.
		const rebuild = () => {
			const rows = []
			const walk = (items, depth, parent) => {
				items.forEach((it, i) => {
					const id = String(it.id)
					const kids = it.children ?? props.loaded[id]
					const branch = !!(kids?.length || (it.lazy && !kids))
					const isOpen = open.has(id) && branch
					// Children arrived (in items or loaded): done loading. An open
					// lazy item without children asks for them (also when the server
					// opened it through expanded).
					if (kids) loading.delete(id)
					else if (isOpen && it.lazy && !loading.has(id)) {
						loading.add(id)
						// The request goes out in a later task: during setup the page's
						// data-on:sb-load isn't attached yet, and inside another effect
						// (a morph) the @get it starts would be tracked by that effect.
						setTimeout(() => emit('sb-load', { id }))
					}
					rows.push({ id, label: String(it.label ?? id), icon: it.icon ?? '', depth, parent, branch, open: isOpen, loading: loading.has(id), pos: i + 1, size: items.length })
					if (isOpen && kids?.length) walk(kids, depth + 1, id)
				})
			}
			walk(props.items || [], 0, '')
			// A row the server dropped hands the focus to its neighbour, so the
			// keyboard stays in the tree instead of falling out to the document.
			const was = $$.rows.findIndex((r) => r.id === $$.focus)
			$$.rows = rows
			if (!rows.some((r) => r.id === $$.focus)) $$.focus = rows[Math.min(Math.max(was, 0), rows.length - 1)]?.id ?? ''
			refocus()
		}
		rebuild()
		// peek: attribute changes arrive inside the effect of whoever set them
		// (e.g. data-attr:loaded); reading signals here must not subscribe it.
		observeProps(() =>
			peek(() => {
				$$.label = props.label
				$$.mode = props.selection
				rebuild()
			}),
		)
		// The server's value and expanded win whenever it sends new ones, also
		// ones the props already decode to (value="" on an element that never
		// had one: observeProps stays silent). The same attribute again keeps
		// the user's edits, and a removed one is ignored (morphs also strip
		// reflected attributes).
		const served = { value: host.getAttribute('value'), expanded: host.getAttribute('expanded') }
		const watch = new MutationObserver(() =>
			peek(() => {
				for (const a in served) {
					const v = host.getAttribute(a)
					if (v === served[a]) continue
					served[a] = v
					if (v === null) continue
					if (a === 'value') $$.selected = ids(v)
					else (open.clear(), ids(v).forEach((id) => open.add(id)))
				}
				rebuild()
			}),
		)
		watch.observe(host, { attributeFilter: ['value', 'expanded'] })
		cleanup(() => watch.disconnect())

		const value = () => (props.selection === 'multiple' ? [...$$.selected] : $$.selected[0] ?? '')
		overrideProp('value', () => peek(value), (v) => peek(() => ($$.selected = ids(v))))
		// Commands: the attribute is the server's value, $$.selected the local one
		// (compared as sets: in multiple mode the click order doesn't matter).
		// With confirm, :state(pending) marks an edit the server hasn't confirmed
		// yet; revert() returns to the server's value (e.g. a rejected command).
		const states = internalsOf(host).states
		const key = (a) => a?.toSorted().join(' ')
		const sync = () => peek(() => (props.confirm && key($$.selected) !== key(ids(props.value)) ? states.add('pending') : states.delete('pending')))
		effect(() => (JSON.stringify($$.selected), sync()))
		observeProps(sync)
		defineHostProp('revert', { value: () => peek(() => (($$.selected = ids(props.value)), sync())) })
		const row = (id) => $$.rows.find((r) => r.id === id)
		// Tabbing in lands on the selected row (the first visible one). (rows?.:
		// effects run once more with the signals gone when the element is removed.)
		effect(() => {
			const r = !$$.hasFocus && $$.rows?.find((r) => $$.selected.includes(r.id))
			if (r) $$.focus = r.id
		})

		const setOpen = (id, wantOpen) => {
			const r = row(id)
			if (!r?.branch || r.open === wantOpen) return
			if (wantOpen) open.add(id)
			else open.delete(id)
			emit('sb-toggle', { id, open: wantOpen })
			rebuild()
		}
		const select = (id) => {
			if (props.selection === 'none') return
			// Choosing the selected item again is no change.
			if (props.selection === 'single' && key($$.selected) === id) return
			const has = $$.selected.includes(id)
			$$.selected = props.selection === 'multiple' ? (has ? $$.selected.filter((x) => x !== id) : [...$$.selected, id]) : [id]
			emit('change')
			emit('sb-change', { name: props.name, value: value() })
		}
		const focus = (id) => id && (($$.focus = id), refocus())

		// Focus can also arrive by Tab or a click: keep the roving focus in step
		// (only rows are focusable, so the target is a row).
		action('focusin', ({ evt }) => {
			$$.hasFocus = true
			const id = evt.target.dataset.id
			if (id) $$.focus = id
		})
		action('focusout', ({ evt }) => {
			// A row the server dropped also "loses" focus, but that's not leaving:
			// its focusout comes while it is still connected and without a
			// relatedTarget, just like a click on the page. So decide a frame
			// later: a removed row is gone by then (refocus() hands the focus to
			// its neighbour); otherwise the focus left, unless it is on another
			// row. (Keep evt.target: it is cleared after dispatch.)
			const t = evt.target
			requestAnimationFrame(() => t.isConnected && !host.shadowRoot.activeElement && ($$.hasFocus = false))
		})
		action('click', ({ evt }, id) => {
			$$.focus = id
			if (row(id)?.branch && evt.target.closest('.caret')) return setOpen(id, !row(id).open)
			select(id)
			// A click also opens or closes a branch, except in multiple mode (setOpen skips leaves).
			if (props.selection !== 'multiple') setOpen(id, !row(id)?.open)
		})
		action('key', ({ evt }) => {
			const rows = $$.rows
			const i = rows.findIndex((r) => r.id === $$.focus)
			const r = rows[i]
			if (!r) return
			switch (evt.key) {
				case 'ArrowDown': focus(rows[i + 1]?.id); break
				case 'ArrowUp': focus(rows[i - 1]?.id); break
				case 'Home': focus(rows[0].id); break
				case 'End': focus(rows.at(-1).id); break
				case 'ArrowRight':
					if (r.branch && !r.open) setOpen(r.id, true)
					else if (rows[i + 1]?.parent === r.id) focus(rows[i + 1].id)
					break
				case 'ArrowLeft':
					if (r.open) setOpen(r.id, false)
					else focus(r.parent)
					break
				case 'Enter':
				case ' ':
					select(r.id)
					break
				default:
					return
			}
			evt.preventDefault()
		})
	},
	render: ({ html }) => html`
		<div role="tree" part="tree"
			data-attr:aria-label="$$label"
			data-attr:aria-multiselectable="$$mode === 'multiple' && 'true'"
			data-on:keydown="@key()" data-on:focusin="@focusin()" data-on:focusout="@focusout()">
			<!-- r?.: when the list shrinks, data-for can re-evaluate a removed row once with r undefined. -->
			<template data-for="r in $$rows">
				<div role="treeitem"
					data-attr:part="$$mode !== 'none' && $$selected.includes(r?.id) ? 'item selected' : 'item'"
					data-attr:data-id="r?.id"
					data-attr:aria-level="r?.depth + 1"
					data-attr:aria-posinset="r?.pos"
					data-attr:aria-setsize="r?.size"
					data-attr:aria-expanded="r?.branch && String(r?.open)"
					data-attr:aria-selected="$$mode !== 'none' && String($$selected.includes(r?.id))"
					data-attr:aria-busy="r?.loading && 'true'"
					data-attr:tabindex="r?.id === $$focus ? 0 : -1"
					data-style:--depth="r?.depth"
					data-on:click="@click(r?.id)">
					<span class="caret" aria-hidden="true"></span>
					<span class="icon" aria-hidden="true" data-show="r?.icon" data-text="r?.icon"></span>
					<span class="label" data-text="r?.label"></span>
					<span class="busy" data-show="r?.loading">Loading</span>
				</div>
			</template>
		</div>
	`,
})
