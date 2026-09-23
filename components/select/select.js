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

// Options come as strings or {value, label?, description?, disabled?}.
const normalize = (list) =>
	(Array.isArray(list) ? list : []).map((o) =>
		typeof o === 'object' && o !== null
			? { value: String(o.value ?? o.label ?? ''), label: String(o.label ?? o.value ?? ''), description: o.description ? String(o.description) : '', disabled: !!o.disabled }
			: { value: String(o), label: String(o), description: '', disabled: false },
	)

// Case- and accent-insensitive matching.
const fold = (s) => s.normalize('NFD').replace(/\p{Diacritic}/gu, '').toLowerCase()

const anchors = typeof CSS !== 'undefined' && CSS.supports?.('anchor-name: --a')

const styles = /* css */ `
:host {
	--_bg: var(--sb-control-bg, #0B1224);
	--_border: var(--sb-control-border, #283552);
	--_border-hover: var(--sb-control-border-hover, #3A4868);
	--_text: var(--sb-control-text, #F3F4FA);
	--_placeholder: var(--sb-control-placeholder, #7785A8);
	--_label: var(--sb-text-2, #AEBBDD);
	--_muted: var(--sb-text-muted, #7785A8);
	--_panel: var(--sb-surface-raised, #141D32);
	--_hover: var(--sb-surface-hover, #1A2540);
	--_brand: var(--sb-brand, #8C6BFF);
	--_brand-light: var(--sb-brand-light, #B09AFF);
	--_brand-subtle: var(--sb-brand-subtle, rgb(140 107 255 / 0.14));
	--_radius: var(--sb-control-radius, 6px);
	display: block;
	inline-size: 100%;
	max-inline-size: 26rem;
}
:host([disabled]) { opacity: 0.5; pointer-events: none; }
.field { display: grid; gap: 0.4rem; }
.label { color: var(--_label); font-size: 0.8125rem; font-weight: 600; }
.control {
	display: flex;
	flex-wrap: wrap;
	align-items: center;
	gap: 0.3rem;
	min-block-size: 2.75rem;
	padding: 0.3rem 2.25rem 0.3rem 0.5rem;
	box-sizing: border-box;
	border: 1px solid var(--_border);
	border-radius: var(--_radius);
	background: var(--_bg);
	cursor: text;
	position: relative;
	anchor-name: --sb-select;
	transition: border-color 120ms, box-shadow 120ms;
}
.control:hover { border-color: var(--_border-hover); }
.control:focus-within { border-color: var(--_brand-light); box-shadow: 0 0 0 3px var(--_brand-subtle); }
.control::after {
	content: "";
	position: absolute;
	inset-inline-end: 0.85rem;
	inset-block-start: 50%;
	inline-size: 8px;
	block-size: 6px;
	translate: 0 -50%;
	background: var(--_placeholder);
	clip-path: polygon(0 0, 8px 0, 8px 2px, 6px 2px, 6px 4px, 4px 4px, 4px 6px, 4px 6px, 4px 4px, 2px 4px, 2px 2px, 0 2px);
}
.open .control::after { rotate: 180deg; }
.chip {
	display: inline-flex;
	align-items: center;
	gap: 0.25rem;
	padding: 0.15rem 0.2rem 0.15rem 0.5rem;
	border-radius: calc(var(--_radius) - 2px);
	background: var(--_brand-subtle);
	color: var(--_text);
	font-size: 0.8125rem;
}
.chip button { all: unset; display: grid; place-items: center; inline-size: 1.1rem; block-size: 1.1rem; border-radius: 3px; color: var(--_muted); cursor: pointer; }
.chip button:hover { color: var(--_text); background: var(--_hover); }
input {
	all: unset;
	flex: 1;
	min-inline-size: 5ch;
	block-size: 2rem;
	padding-inline: 0.35rem;
	color: var(--_text);
	font: inherit;
}
input::placeholder { color: var(--_placeholder); }
input[readonly] { cursor: pointer; }
.clear { all: unset; position: absolute; inset-inline-end: 2rem; display: grid; place-items: center; inline-size: 1.25rem; block-size: 1.25rem; border-radius: 3px; color: var(--_muted); cursor: pointer; }
.clear:hover { color: var(--_text); background: var(--_hover); }
.spin { position: absolute; inset-inline-end: 2rem; inline-size: 12px; block-size: 12px; background: var(--_brand-light); animation: spin 0.6s steps(4) infinite; clip-path: polygon(0 0, 4px 0, 4px 4px, 0 4px, 0 0, 8px 8px, 12px 8px, 12px 12px, 8px 12px, 8px 8px); }
@keyframes spin { to { rotate: 360deg; } }
[popover] {
	margin: 0;
	padding: 4px;
	border: 1px solid var(--_border);
	border-radius: var(--_radius);
	background: var(--_panel);
	color: var(--_text);
	box-shadow: 0 16px 40px -16px rgb(0 0 0 / 0.6);
	max-block-size: min(18rem, 50dvh);
	overflow: auto;
	box-sizing: border-box;
}
/* Under the control, as wide as it (CSS anchor positioning; a JS fallback sets top/left elsewhere). */
@supports (anchor-name: --a) {
	[popover] {
		position-anchor: --sb-select;
		inset: auto;
		position-area: bottom span-right;
		inline-size: anchor-size(width);
		margin-block-start: 4px;
		position-try-fallbacks: flip-block;
	}
}
[role="option"] { display: grid; gap: 0.1rem; padding: 0.45rem 0.6rem; border-radius: calc(var(--_radius) - 2px); cursor: pointer; }
[role="option"][aria-disabled="true"] { opacity: 0.45; cursor: default; }
[role="option"].active { background: var(--_hover); box-shadow: inset 2px 0 0 var(--_brand); }
[role="option"][aria-selected="true"] { color: var(--_brand-light); font-weight: 600; }
.desc { color: var(--_muted); font-size: 0.75rem; font-weight: 400; }
.note { padding: 0.6rem; color: var(--_muted); font-size: 0.8125rem; }
@media (prefers-reduced-motion: reduce) { .spin { animation: none; } }
`

rocket('sb-select', {
	props: ({ bool, json, number, string }) => ({
		options: json.default(() => []).docs({ description: 'Choices: ["A", "B"] or [{value, label, description?, disabled?}].' }),
		results: json.default(() => []).docs({ description: 'Remote: the results of the current search, in the same shape. The server sets it (a signal patch through data-attr, or a morph).' }),
		value: string.docs({ description: 'The value; for multiple, a JSON array or values separated by commas. A new value from the server replaces it; the live value is the value property.' }),
		label: string.trim.docs({ description: 'Visible label.' }),
		placeholder: string.docs({ description: 'Placeholder text.' }),
		multiple: bool.docs({ description: 'Pick several; they show as chips.' }),
		searchable: bool.docs({ description: 'Type to filter the options (in the browser).' }),
		remote: bool.docs({ description: 'Type to search on the server: emits sb-search; the server answers with results.' }),
		delay: number.clamp(0, 2000).default(250).docs({ description: 'Remote: debounce before sb-search, in ms.' }),
		minChars: number.clamp(0, 10).default(1).docs({ description: 'Remote: characters needed before searching.' }),
		loading: bool.docs({ description: 'Show that results are on their way (bind it to data-indicator).' }),
		clearable: bool.docs({ description: 'Show a button that clears the value.' }),
		disabled: bool.docs({ description: 'Disable the control.' }),
		name: string.trim.docs({ description: 'Name reported in sb-change (e.g. the field of a command).' }),
		confirm: bool.docs({ description: 'Server-confirmed value: :state(pending) while the local value differs from the server\'s value attribute (see revert()).' }),
	}),
	manifest: {
		events: [
			{ name: 'sb-search', kind: 'custom-event', bubbles: true, composed: true, description: 'Remote: the query changed (debounced). detail: { query }. Answer by setting results.' },
			{ name: 'change', kind: 'event', bubbles: true, composed: true, description: 'The value changed.' },
			{ name: 'sb-change', kind: 'custom-event', bubbles: true, composed: true, description: 'The value changed. detail: { name, value } (a string, or an array for multiple): ready for a command.' },
		],
	},
	// Rendered once: options, query and selection all flow through signals,
	// so updates (e.g. server results while typing) never rebuild the input.
	renderOnPropChange: false,
	setup: ({ $$, action, adoptStyles, cleanup, defineHostProp, effect, emit, host, observeProps, overrideProp, props }) => {
		adoptStyles(host, styles)
		const parseValue = (v) => {
			if (Array.isArray(v)) return v.map(String)
			const s = String(v ?? '').trim()
			if (!s) return []
			if (s.startsWith('[')) {
				try {
					return JSON.parse(s).map(String)
				} catch {}
			}
			return props.multiple ? s.split(',').map((x) => x.trim()).filter(Boolean) : [s]
		}
		// Labels of everything ever offered, so a selection keeps its label
		// when remote results move on.
		const labels = new Map()
		let options = [] // what the list offers: options, or results when remote
		const learn = () => {
			options = normalize(props.remote ? props.results : props.options)
			for (const o of options) labels.set(o.value, o.label)
		}
		learn()

		$$.selected = parseValue(props.value)
		$$.query = ''
		$$.open = false
		$$.active = -1 // index into $$.view
		$$.view = []
		$$.note = ''
		$$.label = props.label
		$$.placeholder = props.placeholder
		$$.multiple = props.multiple
		$$.typing = props.searchable || props.remote
		$$.loading = props.loading
		$$.clearable = props.clearable
		$$.pending = false // remote: typed, waiting for the debounce or the answer

		const chipsOf = () => $$.selected.map((v) => ({ value: v, label: labels.get(v) ?? v }))
		$$.chips = chipsOf()
		// What the input shows: the query while typing, else (single) the label.
		$$.text = () => ($$.typing && ($$.open || $$.multiple) ? $$.query : $$.multiple ? '' : ($$.chips?.[0]?.label ?? ''))

		const refresh = () => {
			const q = fold($$.query.trim())
			let view = options
			if (props.searchable && !props.remote && q) view = options.filter((o) => fold(o.label).includes(q) || fold(o.description).includes(q))
			$$.view = view.map((o, i) => ({ ...o, id: 'o' + i, selected: $$.selected.includes(o.value) }))
			if ($$.active >= view.length) $$.active = view.length ? 0 : -1
			$$.note = props.remote && $$.query.trim().length < props.minChars ? (props.minChars > 0 ? 'Type to search' : '') : (props.loading || $$.pending) && !view.length ? 'Searching' : !view.length ? 'No results' : ''
			$$.chips = chipsOf()
		}
		refresh()

		// peek: attribute changes arrive inside the effect of whoever set them.
		observeProps((_, changes) =>
			peek(() => {
				// The search is answered: results arrived, or the request ended.
				if ('results' in changes || !props.loading) $$.pending = false
				learn()
				// A value attribute sent by the server wins when it changes; removed
				// attributes are ignored (morphs also strip reflected ones; see sb-slider).
				if ('value' in changes && host.hasAttribute('value')) $$.selected = parseValue(props.value)
				$$.label = props.label
				$$.placeholder = props.placeholder
				$$.multiple = props.multiple
				$$.typing = props.searchable || props.remote
				$$.loading = props.loading
				$$.clearable = props.clearable
				if (props.remote && $$.open && $$.active < 0 && options.length) $$.active = 0
				refresh()
			}),
		)

		const value = () => (props.multiple ? [...$$.selected] : ($$.selected[0] ?? ''))
		overrideProp('value', () => peek(value), (v) => peek(() => (($$.selected = parseValue(v)), refresh())))
		// Commands: the attribute is the server's value, JSON.stringify($$.selected) the local one.
		// With confirm, :state(pending) marks an edit the server hasn't confirmed
		// yet; revert() returns to the server's value (e.g. a rejected command).
		const states = internalsOf(host).states
		const sync = () => peek(() => (props.confirm && JSON.stringify($$.selected) !== JSON.stringify(parseValue(props.value)) ? states.add('pending') : states.delete('pending')))
		effect(() => (JSON.stringify($$.selected), sync()))
		observeProps(sync)
		defineHostProp('revert', { value: () => peek(() => (($$.selected = parseValue(props.value), refresh()), sync())) })

		const popover = () => host.shadowRoot?.querySelector('[popover]')
		const input = () => host.shadowRoot?.querySelector('input')
		const place = () => {
			// Without CSS anchor positioning: put the list under the control.
			if (anchors) return
			const r = host.shadowRoot.querySelector('.control').getBoundingClientRect()
			const p = popover()
			Object.assign(p.style, { position: 'fixed', inset: 'auto', left: r.left + 'px', top: r.bottom + 4 + 'px', width: r.width + 'px' })
		}
		const setOpen = (open) => {
			if (open === $$.open || props.disabled) return
			$$.open = open
			const p = popover()
			try {
				if (open) (p.showPopover(), place())
				else p.hidePopover()
			} catch {}
			if (!open && !props.multiple) $$.query = ''
			if (open) $$.active = Math.max(0, $$.view.findIndex((o) => o.selected))
			refresh()
		}
		const change = () => {
			refresh()
			emit('change')
			emit('sb-change', { name: props.name, value: value() })
		}
		const pick = (v) => {
			const o = options.find((x) => x.value === v)
			if (!o || o.disabled) return
			if (props.multiple) {
				$$.selected = $$.selected.includes(v) ? $$.selected.filter((x) => x !== v) : [...$$.selected, v]
				$$.query = ''
				change()
				search()
			} else {
				$$.selected = [v]
				change()
				setOpen(false)
			}
		}

		let timer = 0
		const search = () => {
			if (!props.remote) return
			clearTimeout(timer)
			const q = $$.query.trim()
			if (q.length < props.minChars) return ($$.pending = false)
			$$.pending = true
			timer = setTimeout(() => emit('sb-search', { query: q }), props.delay)
		}
		cleanup(() => clearTimeout(timer))

		action('type', ({ el }) => {
			$$.query = el.value
			$$.active = 0
			setOpen(true)
			refresh()
			search()
		})
		action('toggle', ({ evt }) => {
			if (evt.target.closest('button')) return
			input()?.focus()
			setOpen(!$$.open)
		})
		action('pick', ({ evt }, v) => {
			evt.preventDefault() // keep focus in the input
			pick(v)
		})
		action('remove', ({ evt }, v) => {
			evt.stopPropagation()
			$$.selected = $$.selected.filter((x) => x !== v)
			change()
		})
		action('clear', ({ evt }) => {
			evt.stopPropagation()
			$$.selected = []
			$$.query = ''
			change()
			input()?.focus()
		})
		action('blur', () => setTimeout(() => host.shadowRoot?.activeElement || setOpen(false), 0))
		action('key', ({ evt }) => {
			const n = $$.view.length
			switch (evt.key) {
				case 'ArrowDown':
					if (!$$.open) setOpen(true)
					else if (n) $$.active = ($$.active + 1) % n
					break
				case 'ArrowUp':
					if (!$$.open) setOpen(true)
					else if (n) $$.active = ($$.active - 1 + n) % n
					break
				case 'Home':
				case 'End':
					if (!$$.open || !n) return
					$$.active = evt.key === 'Home' ? 0 : n - 1
					break
				case 'Enter':
					if ($$.open && $$.view[$$.active]) pick($$.view[$$.active].value)
					else if (!$$.open) setOpen(true)
					break
				case 'Escape':
					if (!$$.open) return
					setOpen(false)
					break
				case 'Backspace':
					if (!props.multiple || $$.query || !$$.selected.length) return
					$$.selected = $$.selected.slice(0, -1)
					change()
					break
				case 'Tab':
					setOpen(false)
					return
				default:
					return
			}
			evt.preventDefault()
			// Keep the highlighted option in view.
			requestAnimationFrame(() => host.shadowRoot?.getElementById($$.view[$$.active]?.id)?.scrollIntoView({ block: 'nearest' }))
		})
	},
	render: ({ html }) => html`
		<div class="field" data-class:open="$$open">
			<label class="label" part="label" for="input" data-show="$$label" data-text="$$label"></label>
			<div class="control" part="control" data-on:click="@toggle()">
				<template data-for="c in $$chips">
					<span class="chip" part="chip" data-show="$$multiple">
						<span data-text="c?.label"></span>
						<button type="button" tabindex="-1" data-attr:aria-label="'Remove ' + c?.label" data-on:click="@remove(c?.value)">×</button>
					</span>
				</template>
				<input id="input" part="input" role="combobox" autocomplete="off" spellcheck="false"
					aria-controls="list" aria-autocomplete="list"
					data-attr:aria-label="$$label ? null : ($$placeholder || 'Select')"
					data-attr:aria-expanded="String($$open)"
					data-attr:aria-activedescendant="$$open && $$view[$$active] ? $$view[$$active].id : null"
					data-attr:aria-busy="$$loading ? 'true' : null"
					data-attr:readonly="!$$typing"
					data-attr:placeholder="$$multiple && $$chips?.length ? null : $$placeholder"
					data-effect="el.value !== $$text && (el.value = $$text)"
					data-on:input="@type()"
					data-on:keydown="@key()"
					data-on:blur="@blur()"/>
				<span class="spin" aria-hidden="true" data-show="$$loading || $$pending"></span>
				<button type="button" class="clear" part="clear" aria-label="Clear" tabindex="-1"
					data-show="$$clearable && $$chips?.length && !$$loading && !$$pending" data-on:click="@clear()">×</button>
			</div>
			<div id="list" part="listbox" popover="manual" role="listbox"
				data-attr:aria-multiselectable="$$multiple ? 'true' : null"
				data-attr:aria-label="$$label || $$placeholder || 'Options'">
				<template data-for="o, i in $$view">
					<div role="option"
						data-attr:id="o?.id"
						data-attr:aria-selected="String(!!o?.selected)"
						data-attr:aria-disabled="o?.disabled ? 'true' : null"
						data-class:active="i === $$active"
						data-on:mousedown="@pick(o?.value)"
						data-on:mousemove="$$active = i">
						<span data-text="o?.label"></span>
						<span class="desc" data-show="o?.description" data-text="o?.description"></span>
					</div>
				</template>
				<div class="note" data-show="$$note" data-text="$$note === 'Searching' ? 'Searching…' : $$note"></div>
			</div>
		</div>
	`,
})
