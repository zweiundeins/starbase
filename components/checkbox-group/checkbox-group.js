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

// The check mark in 2px pixels, cut out of the box's 12px inside.
const check = 'path("M10 1h2v2h-2zM8 3h4v2H8zM0 5h2v2H0zM6 5h4v2H6zM0 7h8v2H0zM2 9h4v2H2z")'

// Choices come as strings or {value, label?, description?, disabled?}, from the
// options prop or from <sb-check> children. A choice needs a value.
const normalize = (list) =>
	(Array.isArray(list) ? list : [])
		.map((o) => (typeof o === 'object' && o !== null ? o : { value: String(o) })) // a string is its value and label
		.map((o) => ({ value: String(o.value ?? o.label ?? ''), label: String(o.label ?? o.value ?? ''), description: o.description ? String(o.description) : '', disabled: !!o.disabled }))
		.filter((o) => o.value)

// The value: a JSON array (or an array), each value once.
const list = (v) => {
	try {
		v = typeof v === 'string' ? JSON.parse(v) : v
	} catch {}
	return Array.isArray(v) ? [...new Set(v.map(String))] : []
}
// Order doesn't matter when comparing values.
const same = (a, b) => JSON.stringify([...a].sort()) === JSON.stringify([...b].sort())

// A choice that can be checked (and take the focus).
const live = (o) => !!o && !o.disabled

// :state(disabled) comes from the decoded prop, so disabled="false" is not
// disabled. Forced colours paint every background Canvas: checked boxes and
// their marks take the system's Highlight colours there.
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
	--_mark: var(--sb-text-on-brand, #F3F4FA);
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
/* flex-start: an item is as wide as its own text, so the hover doesn't
   stretch a whole column. */
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
@media (hover: hover) { .item:not(.off):hover { background: var(--_hover); } }
.item.off { opacity: 0.45; cursor: default; }
/* The outline would be clipped away by the notch, so the notch steps aside. */
.item:focus-visible { outline: 2px solid var(--_brand-light); outline-offset: 2px; clip-path: none; }
.box {
	position: relative;
	flex: none;
	box-sizing: border-box;
	width: 16px;
	height: 16px;
	margin-block-start: 0.1rem;
	border: 2px solid var(--_border);
	background: var(--_bg);
	clip-path: ${notch('var(--_n)')};
	border-radius: calc(4px * (1 - var(--_notch)));
	transition: border-color 120ms, background 120ms;
}
.item:not(.off):hover .box { border-color: var(--_border-hover); }
.on .box, .mixed .box { border-color: var(--_brand-light); background: var(--_brand); }
.on .box::after, .mixed .box::after { content: ""; position: absolute; inset: 0; background: var(--_mark); clip-path: ${check}; }
.mixed .box::after { clip-path: inset(5px 2px); }
.text { display: grid; gap: 0.1rem; }
.desc { color: var(--_muted); font-size: 0.75rem; }
@media (prefers-reduced-motion: reduce) { .item, .box { transition: none; } }
@media (forced-colors: active) {
	.box { forced-color-adjust: none; border-color: CanvasText; background: Canvas; }
	.on .box, .mixed .box { border-color: Highlight; background: Highlight; }
	.box::after { forced-color-adjust: none; background: HighlightText; }
}
`

rocket('sb-checkbox-group', {
	props: ({ bool, json, oneOf, string }) => ({
		value: json.default(() => []).docs({ description: 'The checked values, as a JSON array. A new value from the server replaces it; the live value is the value property.' }),
		options: json.default(() => []).docs({ description: 'Choices from the server: ["A", "B"] or [{value, label, description?, disabled?}]. <sb-check> children win over it.' }),
		label: string.trim.docs({ description: 'Visible label, and the accessible name of the group.' }),
		selectAll: string.trim.docs({ description: 'Show a leading checkbox that checks or clears every enabled choice, labelled with this text ("All" when empty).' }),
		orientation: oneOf('vertical', 'horizontal').default('vertical').docs({ description: 'Stack the choices or lay them out in a row.' }),
		disabled: bool.docs({ description: 'Disable the whole group.' }),
		confirm: bool.docs({ description: 'Server-confirmed value: :state(pending) while the local value differs from the server\'s value attribute (see revert()).' }),
		name: string.trim.docs({ description: 'Name reported in sb-change (e.g. the field of a command), and the field each checked choice submits in a form.' }),
	}),
	manifest: {
		slots: [{ name: '', description: '<sb-check value="…" [description] [disabled]>Label</sb-check> items: markup the group reads as its choices, like <option> in a native <select>. Nothing is slotted; the group renders the items itself.' }],
		events: [
			{ name: 'change', kind: 'event', bubbles: true, composed: true, description: 'When the user checks or clears a choice (or all of them).' },
			{ name: 'sb-change', kind: 'custom-event', bubbles: true, composed: true, description: 'Same moment. detail: { name, value } (value is the array of checked values): ready for a command.' },
		],
	},
	// Rendered once: choices and checks flow through signals, so a new prop
	// never rebuilds the items and takes the keyboard focus with it.
	renderOnPropChange: false,
	setup: ({ $$, action, adoptStyles, cleanup, defineHostProp, effect, emit, host, observeProps, overrideProp, props }) => {
		adoptStyles(host, styles)
		// <sb-check> children are declarative markup, like <option> in a native
		// <select>: inert elements the group reads, not components, so the
		// repeated rows contain no custom elements (Datastar's morph is not
		// re-entrant). normalize() does the rest.
		const fromChildren = () =>
			[...host.querySelectorAll(':scope > sb-check')].map((el) => ({
				value: el.getAttribute('value'),
				label: el.getAttribute('label') ?? el.textContent.trim(),
				description: el.getAttribute('description'),
				disabled: el.hasAttribute('disabled'),
			}))

		// Kept across moves (Rocket drops $$ on disconnect, setup runs again on
		// re-attach) on the element's own ElementInternals: { value: the live
		// value, served: the server's last value attribute }.
		const keep = internalsOf(host)
		const states = keep.states
		$$.value = keep.value ?? []
		$$.items = []
		const take = (p) => {
			$$.label = p.label
			$$.row = p.orientation === 'horizontal'
			$$.disabled = p.disabled
			states[p.disabled ? 'add' : 'delete']('disabled')
		}
		take(props)
		// The select-all box: checked when every enabled choice is, mixed when some are.
		$$.sel = () => {
			const l = $$.items?.filter(live) ?? [] // signals are gone while the element is detached
			const n = l.filter((o) => $$.value?.includes(o.value)).length
			return n && n === l.length ? 'true' : n ? 'mixed' : 'false'
		}

		// The focus: every enabled choice is a tab stop. When the focused one is
		// dropped or disabled, its neighbour takes the focus, so the keyboard
		// stays in the group instead of falling out to the document.
		let hasFocus = false
		let fv = '' // the focused choice's value
		let fi = 0 // and its index
		const refocus = () =>
			requestAnimationFrame(() => {
				// Only pick the focus back up when it fell on the floor, or when the
				// row it sits on shows another choice now (rows are reused by index).
				const a = document.activeElement
				const cur = host.shadowRoot?.activeElement
				if (!hasFocus || props.disabled || (a && a !== document.body && a !== host) || (cur && (!cur.dataset.value || cur.dataset.value === fv))) return
				const l = $$.items
				const at = Math.min(fi, l.length - 1)
				let o = l.find((x) => x.value === fv && live(x))
				// Nearest first, both ways, so it reaches every choice.
				for (let i = 0; !o && i < l.length; i++) o = [l[at + i], l[at - i]].find(live)
				if (o) host.shadowRoot.querySelector(`[data-value="${CSS.escape(o.value)}"]`)?.focus()
			})
		// The items are only re-assigned when they really changed: any other
		// attribute would otherwise rebuild the rows on every morph.
		let shown = '[]'
		const rebuild = () => {
			const kids = fromChildren()
			const items = normalize(kids.length ? kids : props.options)
			const json = JSON.stringify(items)
			if (json !== shown) (shown = json), ($$.items = items), refocus()
		}

		// Children can arrive after the upgrade (the parser is still in the
		// element, or a morph brings other choices): no attribute form for that.
		// The same observer watches the value attribute: the server's value wins
		// when it changes (a morph with a new value); re-sending the same markup
		// changes nothing, so edits survive re-renders. A *removed* attribute
		// changes nothing either (morphs also remove attributes that were only
		// reflected); to clear, the server sends value="[]".
		const attrs = () =>
			peek(() => {
				const a = host.getAttribute('value')
				if (a !== null && a !== keep.served) $$.value = list(a)
				keep.served = a
				$$.aria = host.getAttribute('aria-label')
				$$.all = host.hasAttribute('select-all') ? props.selectAll || 'All' : ''
				rebuild()
			})
		attrs() // also catches what the server said while the element was detached
		const mo = new MutationObserver(attrs)
		mo.observe(host, { childList: true, subtree: true, attributes: true, characterData: true })
		cleanup(() => mo.disconnect())

		overrideProp('value', () => peek(() => [...$$.value]), (v) => peek(() => ($$.value = list(v))))
		// Commands: the attribute is the server's value, $$.value the local one.
		// With confirm, :state(pending) marks an edit the server hasn't confirmed
		// yet; revert() returns to the server's value (e.g. a rejected command).
		const sync = () => peek(() => states[props.confirm && !same($$.value, list(props.value)) ? 'add' : 'delete']('pending'))
		// Rocket's disconnect wipes $$ (re-running this) before a move re-attaches.
		effect(() => ((keep.value = $$.value ? [...$$.value] : keep.value), sync()))
		// peek: attribute changes arrive inside the effect of whoever set them
		// (e.g. data-attr:options); reading signals here must not subscribe it.
		observeProps((p) => peek(() => (take(p), rebuild(), sync())))
		const revert = () => peek(() => (($$.value = list(props.value)), sync()))
		defineHostProp('revert', { value: revert })

		// Forms: until Rocket can make this element form-associated, join the
		// submissions and resets of the form it sits in. `formdata` also fires for
		// new FormData(form), so Datastar's contentType: 'form' posts include it.
		// Both are heard on the root (document or shadow root), once every
		// listener on the form has run: a reset the page cancelled (whenever its
		// listener was added) leaves the value, as it leaves native fields, and a
		// form nested in this one by a script, whose events bubble through it,
		// isn't taken for it. setup reruns on a re-attach, so a move follows.
		// Like checkboxes sharing a name: one entry per checked, enabled choice,
		// in their order. A reset is revert(): the server's value, no events.
		const form = host.closest('form')
		const root = host.getRootNode()
		const onData = (evt) => evt.target === form && peek(() => props.name && !props.disabled && $$.items.forEach((o) => live(o) && $$.value.includes(o.value) && evt.formData.append(props.name, o.value)))
		const onReset = (evt) => evt.target === form && !evt.defaultPrevented && revert()
		root.addEventListener('formdata', onData)
		root.addEventListener('reset', onReset)
		cleanup(() => (root.removeEventListener('formdata', onData), root.removeEventListener('reset', onReset)))

		// The new value keeps the choices' order; values the choices don't
		// (or no longer) offer stay, at the end.
		const commit = (next) => {
			const known = $$.items.map((o) => o.value)
			$$.value = [...known.filter((v) => next.includes(v)), ...next.filter((v) => !known.includes(v))]
			emit('change')
			emit('sb-change', { name: props.name, value: [...$$.value] })
		}
		action('pick', (_, v) => {
			if (props.disabled || !live($$.items.find((o) => o.value === v))) return
			const on = $$.value.includes(v)
			commit(on ? $$.value.filter((x) => x !== v) : [...$$.value, v])
		})
		// Select-all checks every enabled choice, or clears them all when they
		// are all checked. Disabled choices keep their state.
		action('all', () => {
			const l = $$.items.filter(live).map((o) => o.value)
			if (props.disabled || !l.length) return
			commit($$.sel === 'true' ? $$.value.filter((v) => !l.includes(v)) : [...new Set([...$$.value, ...l])])
		})
		// Space toggles the focused box; each choice is its own tab stop, so no arrows.
		action('key', ({ evt }) => evt.key === ' ' && (evt.preventDefault(), evt.repeat || evt.target.click()))

		// A morph parks a row before removing it, so an empty relatedTarget is
		// decided a microtask later: a row that is still there and focusable, in
		// a document that has the focus, means the user left (a click on text).
		action('focusin', ({ evt }) => {
			hasFocus = true
			const v = evt.target.dataset?.value
			if (v) (fv = v), (fi = $$.items.findIndex((o) => o.value === v))
		})
		action('focusout', ({ el, evt }) => {
			const t = evt.target
			if (evt.relatedTarget === null) queueMicrotask(() => t.isConnected && t.hasAttribute('tabindex') && document.hasFocus() && (hasFocus = false))
			else if (!el.contains(evt.relatedTarget)) hasFocus = false
		})
	},
	render: ({ html }) => html`
		<div class="group" part="base" role="group"
			data-attr:aria-labelledby="$$label ? 'label' : null"
			data-attr:aria-label="$$label ? null : $$aria"
			data-attr:aria-disabled="$$disabled ? 'true' : null">
			<span id="label" class="label" part="label" data-show="$$label" data-text="$$label"></span>
			<div class="items" part="items"
				data-class:row="$$row"
				data-on:keydown="@key()" data-on:focusin="@focusin()" data-on:focusout="@focusout()">
				<div class="item" part="item all" role="checkbox"
					data-show="$$all"
					data-attr:aria-checked="$$sel"
					data-attr:aria-disabled="$$disabled ? 'true' : null"
					data-attr:tabindex="$$disabled ? null : 0"
					data-class:on="$$sel === 'true'"
					data-class:mixed="$$sel === 'mixed'"
					data-on:click="@all()">
					<span class="box" part="box" aria-hidden="true"></span>
					<span data-text="$$all"></span>
				</div>
				<!-- o?.: when the list shrinks, data-for can re-evaluate a removed row once with o undefined.
				     No ids on these repeated elements: the morph would park and move them. -->
				<template data-for="o in $$items">
					<div class="item" part="item" role="checkbox"
						data-attr:data-value="o?.value"
						data-attr:aria-checked="String(!!$$value?.includes(o?.value))"
						data-attr:aria-disabled="o?.disabled || $$disabled ? 'true' : null"
						data-attr:tabindex="o?.disabled || $$disabled ? null : 0"
						data-class:on="$$value?.includes(o?.value)"
						data-class:off="o?.disabled"
						data-on:click="@pick(o?.value)">
						<span class="box" part="box" aria-hidden="true"></span>
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
