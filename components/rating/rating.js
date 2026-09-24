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

// Pixel sprites, 7×7: '#' is the shape, '+' its highlight.
const SPRITES = {
	heart: ['.##.##.', '#+#####', '#######', '#######', '.#####.', '..###..', '...#...'],
	star: ['...#...', '..###..', '#######', '.#+###.', '..###..', '.##.##.', '.#...#.'],
}
// One square per pixel whose character is in ch.
const path = (rows, ch) => rows.map((row, y) => [...row].map((c, x) => (ch.includes(c) ? `M${x} ${y}h1v1h-1z` : '')).join('')).join('')

// - Disabled is styled from the rendered row (aria-disabled), not
//   :host([disabled]), which disabled="false" matches too.
// - touch-action pan-y: a vertical swipe still scrolls the page, a sideways
//   drag rates.
// - The filled layer is cut to the unit's share of the value (--fill, 0…1)
//   from the start side. Clip and filter stay on an <svg>: Safari applies CSS
//   filters to root <svg> elements only.
const styles = /* css */ `
:host {
	--_full: var(--sb-rating-color, var(--sb-danger, #F2777A));
	--_empty: var(--sb-border-strong, #3A4868);
	--_shine: var(--sb-rating-shine, rgb(255 255 255 / 0.7));
	--_focus: var(--sb-brand-light, #B09AFF);
	--_label: var(--sb-text-2, #AEBBDD);
	--_size: 1.5rem;
	display: inline-flex;
	flex-direction: column;
	gap: 0.35rem;
}
:host([hidden]) { display: none; }
:host([icon="star"]) { --_full: var(--sb-rating-color, var(--sb-warn, #F5C451)); }
.label:has(+ [aria-disabled]), [aria-disabled] { opacity: 0.5; pointer-events: none; }
.label { color: var(--_label); font-size: 0.8125rem; font-weight: 600; }
.row { display: inline-flex; gap: calc(var(--_size) / 7); inline-size: max-content; border-radius: 4px; cursor: pointer; touch-action: pan-y; }
[aria-readonly] { cursor: default; }
.row:focus-visible { outline: 2px solid var(--_focus); outline-offset: 4px; }
.sm { --_size: 1rem; }
.lg { --_size: 2.25rem; }
.unit { position: relative; display: block; inline-size: var(--_size); block-size: var(--_size); }
svg { position: absolute; inset: 0; inline-size: 100%; block-size: 100%; fill: var(--_empty); }
.full { fill: var(--_full); clip-path: inset(0 calc((1 - var(--fill, 0)) * 100%) 0 0); }
.shine { fill: var(--_shine); }
:host(:dir(rtl)) .full { clip-path: inset(0 0 0 calc((1 - var(--fill, 0)) * 100%)); }
.hot .full { filter: brightness(1.15); }
`

rocket('sb-rating', {
	// oneOf() defaults to its first value.
	props: ({ bool, number, oneOf, string }) => ({
		value: number.min(0).docs({ description: 'The value. A new value from the server replaces it; the live value is the value property.' }),
		max: number.clamp(1, 20).default(5).docs({ description: 'Number of hearts (or stars).' }),
		precision: oneOf('1', '0.5').docs({ description: 'Step: whole or half units.' }),
		icon: oneOf('heart', 'star').docs({ description: 'Pixel sprite.' }),
		size: oneOf('sm', 'md', 'lg').default('md').docs({ description: 'Size.' }),
		label: string.trim.docs({ description: 'Visible label (also the accessible name).' }),
		readonly: bool.docs({ description: 'Show the value; no interaction.' }),
		clearable: bool.docs({ description: 'Picking the current value again clears it to 0.' }),
		disabled: bool.docs({ description: 'Disable interaction.' }),
		confirm: bool.docs({ description: 'Server-confirmed value: :state(pending) while the local value differs from the server\'s value attribute (see revert()).' }),
		name: string.trim.docs({ description: 'Name reported in sb-change (e.g. the field of a command).' }),
	}),
	manifest: {
		events: [
			{ name: 'change', kind: 'event', bubbles: true, composed: true, description: 'When the value is committed.' },
			{ name: 'sb-change', kind: 'custom-event', bubbles: true, composed: true, description: 'Same moment. detail: { name, value }: ready for a command.' },
		],
	},
	setup: ({ $$, action, adoptStyles, cleanup, defineHostProp, effect, emit, host, observeProps, overrideProp, props }) => {
		adoptStyles(host, styles)
		// Number(), not +: property writes aren't decoded, and +1n throws.
		const step = () => Number(props.precision)
		const clamp = (v) => Math.min(props.max, Math.max(0, Math.round((Number(v) || 0) / step()) * step()))
		$$.value = clamp(props.value)
		$$.hover = -1 // preview while pointing, -1 when not
		// A value attribute sent by the server wins when it changes; a removed
		// attribute is ignored (see sb-details). Watched as an attribute, not
		// with observeProps: that only fires when the decoded value changes, so
		// value="0" on an element rendered without one (a server clear) would
		// go unnoticed.
		let served = host.hasAttribute('value') ? props.value : null
		const watch = new MutationObserver(() =>
			peek(() => {
				if (!host.hasAttribute('value')) served = null
				else if (props.value !== served) $$.value = clamp((served = props.value))
			}),
		)
		watch.observe(host, { attributeFilter: ['value'] })
		cleanup(() => watch.disconnect())
		// A new max or precision re-clamps the current value. Only those: clamp()
		// is not idempotent with a max between steps (max 3.2 clamps 9 to 3.2,
		// but 3.2 to 3), so a re-clamp on any prop would change the value.
		observeProps(() => peek(() => ($$.value = clamp($$.value))), 'max', 'precision')
		overrideProp('value', () => peek(() => $$.value), (v) => peek(() => ($$.value = clamp(v))))
		// Commands: the attribute is the server's value, $$.value the local one.
		// With confirm, :state(pending) marks an edit the server hasn't confirmed
		// yet; revert() returns to the server's value (e.g. a rejected command).
		const states = internalsOf(host).states
		// $$.value is read first, so the effect always tracks it (props are not
		// signals). Outside the effect, sync() is called inside peek().
		const sync = () => ($$.value !== clamp(props.value) && props.confirm ? states.add('pending') : states.delete('pending'))
		effect(sync)
		observeProps(() => peek(sync))
		defineHostProp('revert', { value: () => peek(() => (($$.value = clamp(props.value)), sync())) })

		const commit = (v) => {
			v = clamp(v)
			if (v === $$.value) return
			$$.value = v
			emit('change')
			emit('sb-change', { name: props.name, value: v })
		}
		const rtl = () => getComputedStyle(host).direction === 'rtl'
		// Which value the pointer is on, from its position (a touch drag keeps
		// its first target): the last unit whose start it has passed, so a gap
		// belongs to the unit before it, and for half steps which half. -1
		// before the first unit.
		const at = ({ clientX, currentTarget }, back = rtl()) =>
			[...currentTarget.children].reduce((v, unit, i) => {
				const r = unit.getBoundingClientRect()
				const f = (back ? r.right - clientX : clientX - r.left) / r.width
				return f < 0 ? v : i + (f < 0.5 ? step() : 1)
			}, -1)
		const live = () => !props.readonly && !props.disabled
		// A preview left from before readonly or disabled goes on the next move.
		action('point', ({ evt }) => ($$.hover = live() ? at(evt) : -1))
		// On pointerup, not click: a touch drag ends without a click. Only after
		// a press on the row ($$down): a mouse drag that started elsewhere (e.g.
		// selecting text) and ends over it picks nothing.
		action('pick', ({ evt }) => {
			const v = at(evt)
			if (!$$.down || !live() || evt.button || v < 0) return
			$$.hover = -1 // show the result, not the preview
			// clearable is for picking: a key at the maximum stays there.
			commit(props.clearable && v === $$.value ? 0 : v)
		})
		action('key', ({ evt }) => {
			if (!live()) return
			const s = rtl() ? -step() : step() // Left and Right follow the row
			// The value each key goes to (End exactly max: a step past it would
			// clamp differently with a max between steps).
			const keys = { ArrowRight: $$.value + s, ArrowUp: $$.value + step(), ArrowLeft: $$.value - s, ArrowDown: $$.value - step(), Home: 0, End: props.max }
			if (!(evt.key in keys)) return
			commit(keys[evt.key])
			$$.hover = -1
			evt.preventDefault()
		})
	},
	render: ({ html, svg, props: { max, icon, size, label, readonly, disabled } }) => {
		const body = path(SPRITES[icon], '#+'), shine = path(SPRITES[icon], '+')
		// aria-readonly and aria-disabled use a ternary, not `readonly && 'true'`:
		// property writes aren't decoded, and el.disabled = 0 would render
		// aria-disabled="0" (dimmed, while the keyboard still works).
		return html`
			${label ? html`<span class="label" part="label" id="label">${label}</span>` : null}
			<div
				class="row ${size}"
				part="base"
				role="slider"
				tabindex="${readonly || disabled ? null : '0'}"
				aria-labelledby="${label ? 'label' : null}"
				aria-label="${label ? null : 'Rating'}"
				aria-valuemin="0"
				aria-valuemax="${max}"
				aria-readonly="${readonly ? 'true' : null}"
				aria-disabled="${disabled ? 'true' : null}"
				data-attr:aria-valuenow="$$value"
				data-attr:aria-valuetext="$$value + ' of ${max}'"
				data-on:pointermove="@point()"
				data-on:pointerleave="$$hover = -1; $$down = 0"
				data-on:pointerdown="$$down = 1"
				data-on:pointerup="@pick()"
				data-on:keydown="@key()"
			>
				${Array.from(
					{ length: max },
					// Two layers: the empty body, and the full body with its highlight.
					// d via data-attr: a template placeholder in a raw d="" is an
					// invalid path for a moment, which browsers log. The paths are
					// letters and digits.
					(_, i) => html`<span class="unit" part="unit"
						data-class:hot="$$hover > ${i}"
						data-style:--fill="Math.max(0, Math.min(1, ($$hover >= 0 ? $$hover : $$value) - ${i}))">${svg`<svg viewBox="0 0 7 7" shape-rendering="crispEdges" aria-hidden="true"><path data-attr:d="'${body}'"></path></svg><svg class="full" viewBox="0 0 7 7" shape-rendering="crispEdges" aria-hidden="true"><path data-attr:d="'${body}'"></path><path class="shine" data-attr:d="'${shine}'"></path></svg>`}</span>`,
				)}
			</div>
		`
	},
})
