import { rocket, startPeeking, stopPeeking } from 'datastar'

// Host getters must not subscribe callers (e.g. data-bind's sync effect) to
// the internal signal, or that effect writes the stale bound value back.
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

// Pixel corners: notches every corner by p (2px times --sb-notch; at 0 the
// border-radius takes over).
const notch = (p) => `polygon(${p} 0, calc(100% - ${p}) 0, calc(100% - ${p}) ${p}, 100% ${p}, 100% calc(100% - ${p}), calc(100% - ${p}) calc(100% - ${p}), calc(100% - ${p}) 100%, ${p} 100%, ${p} calc(100% - ${p}), 0 calc(100% - ${p}), 0 ${p}, ${p} ${p})`

// Choices come as strings or {value, label?, description?, disabled?}, from the
// options prop or from <sb-radio> children.
const normalize = (list) =>
	(Array.isArray(list) ? list : []).map((o) =>
		typeof o === 'object' && o !== null
			? { value: String(o.value ?? o.label ?? ''), label: String(o.label ?? o.value ?? ''), description: o.description ? String(o.description) : '', disabled: !!o.disabled }
			: { value: String(o), label: String(o), description: '', disabled: false },
	)

const styles = /* css */ `
:host {
	--_bg: var(--sb-control-bg, #0B1224);
	--_border: var(--sb-control-border, #283552);
	--_border-hover: var(--sb-control-border-hover, #3A4868);
	--_text: var(--sb-control-text, #F3F4FA);
	--_label: var(--sb-text-2, #AEBBDD);
	--_muted: var(--sb-text-muted, #7785A8);
	--_brand: var(--sb-brand, #8C6BFF);
	--_brand-light: var(--sb-brand-light, #B09AFF);
	--_brand-subtle: var(--sb-brand-subtle, rgb(140 107 255 / 0.14));
	--_hover: var(--sb-surface-hover, #1A2540);
	--_radius: var(--sb-control-radius, 6px);
	--_notch: var(--sb-notch, 1);
	--_n: calc(2px * var(--_notch));
	display: block;
}
:host([disabled]) { opacity: 0.5; pointer-events: none; }
.group { display: grid; gap: 0.5rem; }
.label { color: var(--_label); font-size: 0.8125rem; font-weight: 600; }
/* flex-start: an item is as wide as its own text, so the checked tint and the
   hover don't stretch a whole column. */
.items { display: flex; flex-direction: column; align-items: flex-start; gap: 0.25rem; }
.items.row { flex-direction: row; flex-wrap: wrap; column-gap: 0.75rem; }
.item {
	display: flex;
	align-items: flex-start;
	gap: 0.6rem;
	padding: 0.4rem 0.6rem;
	color: var(--_text);
	font-size: 0.875rem;
	cursor: pointer;
	clip-path: ${notch('var(--_n)')};
	border-radius: calc(var(--_radius) * (1 - var(--_notch)));
	transition: background 120ms;
}
.item.hot { background: var(--_hover); }
.item.checked { background: var(--_brand-subtle); }
.item.off { opacity: 0.45; cursor: default; }
/* The outline would be clipped away by the notch, so the notch steps aside. */
.item:focus-visible { outline: 2px solid var(--_brand-light); outline-offset: 2px; clip-path: none; }
.dot {
	position: relative;
	flex: none;
	box-sizing: border-box;
	inline-size: 16px;
	block-size: 16px;
	margin-block-start: 0.1rem;
	border: 2px solid var(--_border);
	background: var(--_bg);
	clip-path: ${notch('var(--_n)')};
	border-radius: calc(8px * (1 - var(--_notch)));
	transition: border-color 120ms;
}
.item.hot .dot { border-color: var(--_border-hover); }
.item.checked .dot { border-color: var(--_brand-light); }
.item.checked .dot::after {
	content: "";
	position: absolute;
	inset: 2px;
	background: var(--_brand);
	clip-path: ${notch('calc(1px * var(--_notch))')};
	border-radius: calc(4px * (1 - var(--_notch)));
}
.text { display: grid; gap: 0.1rem; }
.desc { color: var(--_muted); font-size: 0.75rem; }
@media (prefers-reduced-motion: reduce) { .item, .dot { transition: none; } }
`

rocket('sb-radio-group', {
	props: ({ bool, json, oneOf, string }) => ({
		value: string.docs({ description: 'The selected value. A new value from the server replaces it; the live value is the value property.' }),
		options: json.default(() => []).docs({ description: 'Choices from the server: ["A", "B"] or [{value, label, description?, disabled?}]. <sb-radio> children win over it.' }),
		label: string.trim.docs({ description: 'Visible label, and the accessible name of the group.' }),
		orientation: oneOf('vertical', 'horizontal').default('vertical').docs({ description: 'Stack the choices or lay them out in a row.' }),
		disabled: bool.docs({ description: 'Disable the whole group.' }),
		confirm: bool.docs({ description: 'Server-confirmed value: :state(pending) while the local value differs from the server\'s value attribute (see revert()).' }),
		name: string.trim.docs({ description: 'Name reported in sb-change (e.g. the field of a command).' }),
	}),
	manifest: {
		slots: [{ name: '', description: '<sb-radio value="…" [description] [disabled]>Label</sb-radio> items: markup the group reads as its choices, like <option> in a native <select>. Nothing is slotted; the group renders the items itself.' }],
		events: [
			{ name: 'change', kind: 'event', bubbles: true, composed: true, description: 'When the user picks a choice.' },
			{ name: 'sb-change', kind: 'custom-event', bubbles: true, composed: true, description: 'Same moment. detail: { name, value }: ready for a command.' },
		],
	},
	// Rendered once: choices, selection and roving focus all flow through
	// signals, so a new prop never rebuilds the items and takes the keyboard
	// focus with it.
	renderOnPropChange: false,
	setup: ({ $$, action, adoptStyles, cleanup, defineHostProp, effect, emit, host, observeProps, overrideProp, props }) => {
		adoptStyles(host, styles)
		const str = (v) => String(v ?? '')
		// <sb-radio> children are declarative markup, like <option> in a native
		// <select>: inert elements the group reads. They are not their own Rocket
		// component, so the repeated items contain no custom elements (Datastar's
		// morph is not re-entrant) and the roving focus stays in one shadow root.
		const fromChildren = () =>
			[...host.querySelectorAll(':scope > sb-radio')].map((el) => {
				const text = el.textContent.trim()
				return {
					value: el.getAttribute('value') ?? el.getAttribute('label') ?? text,
					label: el.getAttribute('label') ?? text,
					description: el.getAttribute('description') ?? '',
					disabled: el.hasAttribute('disabled'),
				}
			})

		$$.value = str(props.value)
		$$.focus = '' // the item in the tab order (roving tabindex)
		$$.hover = '' // the item under the pointer
		$$.hasFocus = false // whether the keyboard focus is inside the group
		$$.items = []
		$$.label = props.label
		$$.row = props.orientation === 'horizontal'
		$$.disabled = props.disabled

		// Only the checked item is tabbable; with nothing checked, the first
		// enabled one is, so the group is always reachable with one Tab.
		// The items are only re-assigned when they really changed: any other
		// attribute would otherwise rebuild the rows on every morph.
		let shown = '[]'
		const rebuild = () => {
			const kids = fromChildren()
			const items = normalize(kids.length ? kids : props.options).filter((o) => o.value !== '')
			const json = JSON.stringify(items)
			// Where the focus sat before the new list: a choice the server drops
			// hands the focus to its neighbour, so the keyboard stays in the group
			// instead of jumping back to the top or falling out to the document.
			const was = $$.items.findIndex((o) => o.value === $$.focus)
			if (json !== shown) (shown = json), ($$.items = items)
			const list = $$.items
			const live = (o) => !!o && !o.disabled
			const neighbour = () => {
				const at = Math.min(Math.max(was, 0), list.length - 1)
				for (let i = 0; i < list.length; i++) {
					if (live(list[at + i])) return list[at + i]
					if (live(list[at - i])) return list[at - i]
				}
			}
			const checked = list.find((o) => o.value === $$.value && live(o))
			const stays = list.find((o) => o.value === $$.focus && live(o))
			const next = (checked ?? stays ?? neighbour() ?? list.find(live))?.value ?? ''
			if (next !== $$.focus) $$.focus = next
			refocus()
		}
		// Move the DOM focus to the focus item, also after the items re-rendered
		// (they are new elements then).
		const refocus = () =>
			requestAnimationFrame(() => {
				if (!$$.hasFocus) return
				// Only pick the focus back up when it fell on the floor (the item was
				// replaced or dropped): if the user moved on to something else, leave
				// it alone.
				const active = document.activeElement
				if (active && active !== document.body && active !== host) return
				const el = host.shadowRoot?.querySelector(`[data-value="${CSS.escape($$.focus)}"]`)
				if (el && host.shadowRoot.activeElement !== el) el.focus()
			})
		rebuild()

		// Children can arrive after the upgrade (the parser is still in the
		// element, or a morph brings other choices): no attribute form for that.
		const mo = new MutationObserver(() => peek(rebuild))
		mo.observe(host, { childList: true, subtree: true, attributes: true, characterData: true })
		cleanup(() => mo.disconnect())

		// peek: attribute changes arrive inside the effect of whoever set them
		// (e.g. data-attr:options); reading signals here must not subscribe it.
		observeProps((p, changes) =>
			peek(() => {
				// A value attribute sent by the server wins when it changes (a morph
				// with a new value); re-sending the same markup changes nothing, so
				// edits survive re-renders. A *removed* attribute changes nothing
				// either: morphs also remove attributes that were only reflected
				// (e.g. from a data-bind write before the upgrade). To clear it, the
				// server sends value="".
				if ('value' in changes && host.hasAttribute('value')) $$.value = str(p.value)
				$$.label = p.label
				$$.row = p.orientation === 'horizontal'
				$$.disabled = p.disabled
				rebuild()
			}),
		)

		overrideProp('value', () => peek(() => $$.value), (v) => peek(() => (($$.value = str(v)), rebuild())))
		// Commands: the attribute is the server's value, $$.value the local one.
		// With confirm, :state(pending) marks an edit the server hasn't confirmed
		// yet; revert() returns to the server's value (e.g. a rejected command).
		const states = internalsOf(host).states
		const sync = () => peek(() => (props.confirm && $$.value !== str(props.value) ? states.add('pending') : states.delete('pending')))
		effect(() => ($$.value, sync()))
		observeProps(sync)
		defineHostProp('revert', { value: () => peek(() => (($$.value = str(props.value)), rebuild(), sync())) })

		const item = (v) => $$.items.find((o) => o.value === v)
		// Arrow keys move and select, like native radios.
		const pick = (v) => {
			const o = item(v)
			if (!o || o.disabled || props.disabled) return
			$$.focus = v
			refocus()
			if (v === $$.value) return
			$$.value = v
			emit('change')
			emit('sb-change', { name: props.name, value: v })
		}
		// The next enabled choice, wrapping around; dir -1 goes back.
		const step = (from, dir) => {
			const list = $$.items
			const n = list.length
			const at = Math.max(0, list.findIndex((o) => o.value === from))
			for (let i = 1; i <= n; i++) {
				const o = list[(((at + dir * i) % n) + n) % n]
				if (o && !o.disabled) return o.value
			}
			return from
		}
		const edge = (dir) => {
			const list = dir > 0 ? $$.items : [...$$.items].reverse()
			return list.find((o) => !o.disabled)?.value ?? ''
		}

		// Focus can also arrive by Tab or a click: keep the roving focus in step.
		action('focusin', ({ evt }) => {
			$$.hasFocus = true
			const v = evt.target.closest?.('[role="radio"]')?.dataset.value
			if (v) $$.focus = v
		})
		action('focusout', ({ el, evt }) => {
			// An item re-rendered away also "loses" focus: that's not leaving. The
			// morph can park an item before removing it, so the item is still
			// connected and only the empty relatedTarget gives it away; refocus()
			// decides whether the focus really went somewhere else.
			if (!evt.target.isConnected || evt.relatedTarget === null) return
			if (!el.contains(evt.relatedTarget)) $$.hasFocus = false
		})
		action('pick', (_, v) => pick(v))
		action('key', ({ evt }) => {
			if (props.disabled || !$$.items.length) return
			switch (evt.key) {
				case 'ArrowDown':
				case 'ArrowRight':
					pick(step($$.focus, 1))
					break
				case 'ArrowUp':
				case 'ArrowLeft':
					pick(step($$.focus, -1))
					break
				case 'Home':
					pick(edge(1))
					break
				case 'End':
					pick(edge(-1))
					break
				case ' ':
				case 'Enter':
					pick($$.focus)
					break
				default:
					return
			}
			evt.preventDefault()
		})
	},
	render: ({ html }) => html`
		<div class="group" part="base">
			<span class="label" part="label" id="group-label" data-show="$$label" data-text="$$label"></span>
			<div class="items" part="items" role="radiogroup"
				data-class:row="$$row"
				data-attr:aria-orientation="$$row ? 'horizontal' : 'vertical'"
				data-attr:aria-labelledby="$$label ? 'group-label' : null"
				data-attr:aria-label="$$label ? null : 'Choice'"
				data-attr:aria-disabled="$$disabled ? 'true' : null"
				data-on:keydown="@key()" data-on:focusin="@focusin()" data-on:focusout="@focusout()">
				<!-- o?.: when the list shrinks, data-for can re-evaluate a removed row once with o undefined.
				     No ids on these repeated elements: the morph would park and move them. -->
				<template data-for="o in $$items">
					<div class="item" part="item" role="radio"
						data-attr:data-value="o?.value"
						data-attr:aria-checked="String(o?.value === $$value)"
						data-attr:aria-disabled="o?.disabled || $$disabled ? 'true' : null"
						data-attr:tabindex="!$$disabled && o?.value === $$focus ? 0 : -1"
						data-class:checked="o?.value === $$value"
						data-class:off="o?.disabled"
						data-class:hot="o?.value === $$hover && !o?.disabled && !$$disabled"
						data-on:click="@pick(o?.value)"
						data-on:pointerenter="$$hover = o?.value"
						data-on:pointerleave="$$hover = ''">
						<span class="dot" part="dot" aria-hidden="true"></span>
						<span class="text">
							<span class="label-text" data-text="o?.label"></span>
							<span class="desc" part="description" data-show="o?.description" data-text="o?.description"></span>
						</span>
					</div>
				</template>
			</div>
		</div>
	`,
})
