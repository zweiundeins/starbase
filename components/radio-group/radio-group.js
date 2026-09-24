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
// options prop or from <sb-radio> children. A choice needs a value: "" is the
// group's "nothing chosen".
const normalize = (list) =>
	(Array.isArray(list) ? list : [])
		.map((o) => (typeof o === 'object' && o !== null ? o : { value: String(o) })) // a string is its value and label
		.map((o) => ({ value: String(o.value ?? o.label ?? ''), label: String(o.label ?? o.value ?? ''), description: o.description ? String(o.description) : '', disabled: !!o.disabled }))
		.filter((o) => o.value)

// A choice that can be picked (and take the focus).
const live = (o) => !!o && !o.disabled

// :state(disabled) comes from the decoded prop, so disabled="false" is not
// disabled. Forced colours paint every background Canvas: the checked dot is
// filled with Highlight there.
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
:host([hidden]) { display: none; }
:host(:state(disabled)) { opacity: 0.5; pointer-events: none; }
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
	width: 16px;
	height: 16px;
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
@media (forced-colors: active) { .item.checked .dot::after { forced-color-adjust: none; background: Highlight; } }
`

rocket('sb-radio-group', {
	props: ({ bool, json, oneOf, string }) => ({
		value: string.docs({ description: 'The selected value. A new value from the server replaces it; the live value is the value property.' }),
		options: json.default(() => []).docs({ description: 'Choices from the server: ["A", "B"] or [{value, label, description?, disabled?}]. <sb-radio> children win over it.' }),
		label: string.trim.docs({ description: 'Visible label, and the accessible name of the group.' }),
		orientation: oneOf('vertical', 'horizontal').default('vertical').docs({ description: 'Stack the choices or lay them out in a row.' }),
		disabled: bool.docs({ description: 'Disable the whole group.' }),
		confirm: bool.docs({ description: 'Server-confirmed value: :state(pending) while the local value differs from the server\'s value attribute (see revert()).' }),
		name: string.trim.docs({ description: 'Name reported in sb-change (e.g. the field of a command), and the field it submits in a form.' }),
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
		// normalize() does the rest: the value falls back to the label, then to
		// the text.
		const fromChildren = () =>
			[...host.querySelectorAll(':scope > sb-radio')].map((el) => ({
				value: el.getAttribute('value'),
				label: el.getAttribute('label') ?? el.textContent.trim(),
				description: el.getAttribute('description'),
				disabled: el.hasAttribute('disabled'),
			}))

		$$.value = str(props.value)
		$$.focus = '' // the item in the tab order (roving tabindex)
		$$.hover = '' // the item under the pointer
		$$.hasFocus = false // whether the keyboard focus is inside the group
		$$.items = []
		// The props the template and the host's ARIA follow. The host is the
		// radiogroup, so a page's own aria-label or aria-labelledby on it names
		// the group (they win over the internals). Internals, like :state(),
		// survive morphs.
		const int = internalsOf(host)
		const states = int.states
		int.role = 'radiogroup'
		const take = (p) => {
			$$.label = p.label
			$$.row = p.orientation === 'horizontal'
			$$.disabled = p.disabled
			int.ariaLabel = p.label || null
			int.ariaOrientation = p.orientation
			int.ariaDisabled = p.disabled
			states[p.disabled ? 'add' : 'delete']('disabled')
		}
		take(props)

		// Only the checked item is tabbable; with nothing checked, the first
		// enabled one is, so the group is always reachable with one Tab.
		// The items are only re-assigned when they really changed: any other
		// attribute would otherwise rebuild the rows on every morph.
		let shown = '[]'
		const rebuild = () => {
			const kids = fromChildren()
			const items = normalize(kids.length ? kids : props.options)
			const json = JSON.stringify(items)
			// Where the focus sat before the new list: a choice the server drops
			// hands the focus to its neighbour, so the keyboard stays in the group
			// instead of jumping back to the top or falling out to the document.
			const was = $$.items.findIndex((o) => o.value === $$.focus)
			if (json !== shown) (shown = json), ($$.items = items)
			const list = $$.items
			// Nearest first, both ways, so it reaches every choice.
			const neighbour = () => {
				const at = Math.min(Math.max(was, 0), list.length - 1)
				for (let i = 0; i < list.length; i++) {
					if (live(list[at + i])) return list[at + i]
					if (live(list[at - i])) return list[at - i]
				}
			}
			const pickable = (v) => list.find((o) => o.value === v && live(o))
			const next = (pickable($$.value) ?? pickable($$.focus) ?? neighbour())?.value ?? ''
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
		// The same observer watches the value attribute: the server's value wins
		// when it changes (a morph with a new value); re-sending the same markup
		// changes nothing, so edits survive re-renders. A *removed* attribute
		// changes nothing either: morphs also remove attributes that were only
		// reflected (e.g. from a data-bind write before the upgrade). To clear
		// it, the server sends value="", which observeProps misses when there was
		// no attribute before (both decode to "").
		let served = host.getAttribute('value')
		const mo = new MutationObserver(() =>
			peek(() => {
				const a = host.getAttribute('value')
				if (a !== null && a !== served) $$.value = a
				served = a
				rebuild()
			}),
		)
		mo.observe(host, { childList: true, subtree: true, attributes: true, characterData: true })
		cleanup(() => mo.disconnect())

		overrideProp('value', () => peek(() => $$.value), (v) => peek(() => (($$.value = str(v)), rebuild())))
		// Commands: the attribute is the server's value, $$.value the local one.
		// With confirm, :state(pending) marks an edit the server hasn't confirmed
		// yet; revert() returns to the server's value (e.g. a rejected command).
		const sync = () => peek(() => states[props.confirm && $$.value !== str(props.value) ? 'add' : 'delete']('pending'))
		effect(() => ($$.value, sync()))
		// peek: attribute changes arrive inside the effect of whoever set them
		// (e.g. data-attr:options); reading signals here must not subscribe it.
		observeProps((p) => peek(() => (take(p), rebuild(), sync())))
		const revert = () => peek(() => (($$.value = str(props.value)), rebuild(), sync()))
		defineHostProp('revert', { value: revert })

		// Forms: until Rocket can make this element form-associated, join the
		// submissions and resets of the form it sits in. `formdata` also fires for
		// new FormData(form), so Datastar's contentType: 'form' posts include it.
		// Both are heard on the root (document or shadow root), once every
		// listener on the form has run: a reset the page cancelled (whenever its
		// listener was added) leaves the value, as it leaves native fields, and a
		// form nested in this one by a script, whose events bubble through it,
		// isn't taken for it. setup reruns on a re-attach, so a move follows.
		// Like radios, only a checked choice that is enabled is submitted. A reset
		// is revert(): the server's value, no change events.
		const form = host.closest('form')
		const root = host.getRootNode()
		const onData = (evt) => evt.target === form && peek(() => props.name && !props.disabled && live($$.items.find((o) => o.value === $$.value)) && evt.formData.append(props.name, $$.value))
		const onReset = (evt) => evt.target === form && !evt.defaultPrevented && revert()
		root.addEventListener('formdata', onData)
		root.addEventListener('reset', onReset)
		cleanup(() => (root.removeEventListener('formdata', onData), root.removeEventListener('reset', onReset)))

		// Arrow keys move and select, like native radios.
		const pick = (v) => {
			if (props.disabled || !live($$.items.find((o) => o.value === v))) return
			$$.focus = v
			refocus()
			if (v === $$.value) return
			$$.value = v
			emit('change')
			emit('sb-change', { name: props.name, value: v })
		}
		// The next enabled choice after index at, wrapping around; dir -1 goes
		// back. From -1 onwards it is the first, from 0 backwards the last.
		const step = (at, dir) => {
			const list = $$.items
			const n = list.length
			for (let i = 1; i <= n; i++) {
				const o = list[(((at + dir * i) % n) + n) % n]
				if (live(o)) return o.value
			}
		}

		// Focus can also arrive by Tab or a click: keep the roving focus in step.
		action('focusin', ({ evt }) => {
			$$.hasFocus = true
			const v = evt.target.closest?.('[role="radio"]')?.dataset.value
			if (v) $$.focus = v
		})
		action('focusout', ({ el, evt }) => {
			// An item re-rendered away (or disabled, so it lost its tabindex) also
			// "loses" focus: that's not leaving, and refocus() hands the focus to
			// a neighbour. data-for fires focusout while it removes the row, which
			// is still connected then, so an empty relatedTarget is decided a
			// microtask later: a row that is still there and focusable, in a
			// document that has the focus, means the focus went to the page (a
			// click on text), and the group must not pull it back. A switch to
			// another window or tab is not leaving: the focus comes back to the
			// row, or to its neighbour if the row was dropped meanwhile.
			const t = evt.target
			if (evt.relatedTarget === null) queueMicrotask(() => t.isConnected && t.hasAttribute('tabindex') && document.hasFocus() && ($$.hasFocus = false))
			else if (!el.contains(evt.relatedTarget)) $$.hasFocus = false
		})
		action('pick', (_, v) => pick(v))
		action('key', ({ evt }) => {
			// Modified arrows belong to the browser (Alt+Left is Back).
			if (props.disabled || !$$.items.length || evt.altKey || evt.ctrlKey || evt.metaKey) return
			const at = Math.max(0, $$.items.findIndex((o) => o.value === $$.focus))
			switch (evt.key) {
				case 'ArrowDown':
					pick(step(at, 1))
					break
				case 'ArrowUp':
					pick(step(at, -1))
					break
				// Left and Right follow the reading direction, like native radios:
				// Left goes on in right-to-left text.
				case 'ArrowRight':
				case 'ArrowLeft':
					pick(step(at, (evt.key === 'ArrowLeft') === (getComputedStyle(host).direction === 'rtl') ? 1 : -1))
					break
				case 'Home':
					pick(step(-1, 1))
					break
				case 'End':
					pick(step(0, -1))
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
			<span class="label" part="label" aria-hidden="true" data-show="$$label" data-text="$$label"></span>
			<div class="items" part="items"
				data-class:row="$$row"
				data-on:keydown="@key()" data-on:focusin="@focusin()" data-on:focusout="@focusout()">
				<!-- o?.: when the list shrinks, data-for can re-evaluate a removed row once with o undefined.
				     No ids on these repeated elements: the morph would park and move them. -->
				<template data-for="o in $$items">
					<div class="item" part="item" role="radio"
						data-attr:data-value="o?.value"
						data-attr:aria-checked="String(o?.value === $$value)"
						data-attr:aria-disabled="o?.disabled || $$disabled ? 'true' : null"
						data-attr:tabindex="o?.disabled ? null : !$$disabled && o?.value === $$focus ? 0 : -1"
						data-class:checked="o?.value === $$value"
						data-class:off="o?.disabled"
						data-class:hot="o?.value === $$hover && !o?.disabled && !$$disabled"
						data-on:click="@pick(o?.value)"
						data-on:pointerenter="$$hover = o?.value"
						data-on:pointerleave="$$hover = ''">
						<span class="dot" part="dot" aria-hidden="true"></span>
						<span class="text">
							<span data-text="o?.label"></span>
							<span class="desc" part="description" data-show="o?.description" data-text="o?.description"></span>
						</span>
					</div>
				</template>
			</div>
		</div>
	`,
})
