import { rocket, startPeeking, stopPeeking } from 'datastar'

// data-bind may set a property before the element is upgraded; adopt it.
const early = (host, name) => {
	const d = Object.getOwnPropertyDescriptor(host, name)
	if (!d || !('value' in d)) return undefined
	delete host[name]
	return d.value
}

// Host getters must not subscribe callers (e.g. data-bind's sync effect).
const peek = (fn) => {
	startPeeking()
	try {
		return fn()
	} finally {
		stopPeeking()
	}
}

// Pixel sprites, 7×7: '#' is the shape, '+' its highlight.
const SPRITES = {
	heart: ['.##.##.', '#+#####', '#######', '#######', '.#####.', '..###..', '...#...'],
	star: ['...#...', '..###..', '#######', '.#+###.', '..###..', '.##.##.', '.#...#.'],
}
const path = (rows, ch) => {
	let d = ''
	rows.forEach((row, y) => [...row].forEach((c, x) => ch.includes(c) && (d += `M${x} ${y}h1v1h-1z`)))
	return d
}
const svg = (kind, cls) =>
	`<svg class="${cls}" viewBox="0 0 7 7" shape-rendering="crispEdges" aria-hidden="true"><path class="body" d="${path(SPRITES[kind], '#+')}"/><path class="shine" d="${path(SPRITES[kind], '+')}"/></svg>`

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
:host([icon="star"]) { --_full: var(--sb-rating-color, var(--sb-warn, #F5C451)); }
:host([disabled]) { opacity: 0.5; pointer-events: none; }
.label { color: var(--_label); font-size: 0.8125rem; font-weight: 600; }
.row { display: inline-flex; gap: calc(var(--_size) / 7); inline-size: max-content; border-radius: 4px; cursor: pointer; touch-action: none; }
.row.readonly { cursor: default; }
.row:focus-visible { outline: 2px solid var(--_focus); outline-offset: 4px; }
.sm { --_size: 1rem; }
.lg { --_size: 2.25rem; }
.unit { position: relative; display: block; inline-size: var(--_size); block-size: var(--_size); }
svg { position: absolute; inset: 0; inline-size: 100%; block-size: 100%; }
.empty .body { fill: var(--_empty); }
.empty .shine { fill: var(--_empty); }
.full .body { fill: var(--_full); }
.full .shine { fill: var(--_shine); }
/* The filled layer is cut to the unit's share of the value (--fill, 0…1). */
.full { clip-path: inset(0 calc((1 - var(--fill, 0)) * 100%) 0 0); }
.row:not(.readonly) .unit.hot .full { filter: brightness(1.15); }
`

rocket('sb-rating', {
	props: ({ bool, number, oneOf, string }) => ({
		value: number.min(0).docs({ description: 'Initial value. Read the live value from the value property.' }),
		max: number.clamp(1, 20).default(5).docs({ description: 'Number of hearts (or stars).' }),
		precision: oneOf('1', '0.5').default('1').docs({ description: 'Step: whole or half units.' }),
		icon: oneOf('heart', 'star').default('heart').docs({ description: 'Pixel sprite.' }),
		size: oneOf('sm', 'md', 'lg').default('md').docs({ description: 'Size.' }),
		label: string.trim.docs({ description: 'Visible label (also the accessible name).' }),
		readonly: bool.docs({ description: 'Show the value; no interaction.' }),
		clearable: bool.docs({ description: 'Picking the current value again clears it to 0.' }),
		disabled: bool.docs({ description: 'Disable interaction.' }),
	}),
	manifest: {
		events: [
			{ name: 'change', kind: 'event', bubbles: true, composed: true, description: 'When the value is committed.' },
			{ name: 'sb-change', kind: 'custom-event', bubbles: true, composed: true, description: 'Same moment. detail: { value }.' },
		],
	},
	setup: ({ $$, action, adoptStyles, emit, host, observeProps, overrideProp, props }) => {
		adoptStyles(host, styles)
		const step = () => Number(props.precision)
		const clamp = (v) => Math.min(props.max, Math.max(0, Math.round((Number(v) || 0) / step()) * step()))
		$$.value = clamp(props.value)
		const pre = early(host, 'value')
		$$.dirty = pre !== undefined
		if ($$.dirty) $$.value = clamp(pre)
		$$.hover = -1 // preview while pointing, -1 when not
		// Like a native <input>: the attribute is only the default (see sb-slider).
		observeProps(() => peek(() => ($$.value = clamp($$.dirty ? $$.value : props.value))), 'value', 'max', 'precision')
		overrideProp('value', () => peek(() => $$.value), (v) => peek(() => (($$.dirty = true), ($$.value = clamp(v)))))
		$$.shown = () => ($$.hover >= 0 ? $$.hover : $$.value)
		$$.sprite = () => svg(props.icon, 'empty') + svg(props.icon, 'full')

		const commit = (v) => {
			v = clamp(v)
			if (props.clearable && v === $$.value) v = 0
			if (v === $$.value) return
			$$.value = v
			$$.dirty = true
			emit('change')
			emit('sb-change', { value: v })
		}
		// Which value the pointer is on: the unit under it, and for half steps
		// which half.
		const at = (evt) => {
			const unit = evt.target.closest?.('.unit')
			if (!unit) return -1
			const i = Number(unit.dataset.i)
			const r = unit.getBoundingClientRect()
			const half = step() < 1 && evt.clientX - r.left < r.width / 2
			return i + (half ? 0.5 : 1)
		}
		const live = () => !props.readonly && !props.disabled
		action('point', ({ evt }) => live() && ($$.hover = at(evt)))
		action('leave', () => ($$.hover = -1))
		action('pick', ({ evt }) => {
			if (!live()) return
			const v = at(evt)
			if (v >= 0) commit(v)
		})
		action('key', ({ evt }) => {
			if (!live()) return
			const keys = { ArrowRight: step(), ArrowUp: step(), ArrowLeft: -step(), ArrowDown: -step() }
			if (evt.key in keys) commit($$.value + keys[evt.key])
			else if (evt.key === 'Home') commit(0)
			else if (evt.key === 'End') commit(props.max)
			else return
			evt.preventDefault()
		})
	},
	render: ({ html, props: { max, size, label, readonly, disabled } }) => {
		const units = Array.from({ length: max }, (_, i) => i)
		return html`
			${label ? html`<span class="label" part="label" id="label">${label}</span>` : null}
			<div
				class="row ${size} ${readonly ? 'readonly' : ''}"
				part="base"
				role="${readonly ? 'img' : 'slider'}"
				tabindex="${readonly || disabled ? null : '0'}"
				aria-labelledby="${label ? 'label' : null}"
				aria-label="${label ? null : 'Rating'}"
				aria-valuemin="${readonly ? null : '0'}"
				aria-valuemax="${readonly ? null : max}"
				aria-disabled="${disabled ? 'true' : null}"
				data-attr:aria-valuenow="${readonly ? 'null' : '$$value'}"
				data-attr:aria-valuetext="$$value + ' of ${max}'"
				data-on:pointermove="@point()"
				data-on:pointerleave="@leave()"
				data-on:click="@pick()"
				data-on:keydown="@key()"
			>
				${units.map(
					(i) => html`<span class="unit" part="unit" data-i="${i}"
						data-class:hot="$$hover > ${i}"
						data-style:--fill="Math.max(0, Math.min(1, $$shown - ${i}))"
						data-effect="el.innerHTML = $$sprite"></span>`,
				)}
			</div>
		`
	},
})
