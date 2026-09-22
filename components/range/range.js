import { rocket, startPeeking, stopPeeking } from 'datastar'

// Host getters must not subscribe callers (e.g. data-bind's sync effect) to
// the internal signals, or that effect writes stale values back.
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

const thumb = /* css */ `
	pointer-events: auto;
	inline-size: 14px;
	block-size: 20px;
	border: 0;
	background: var(--_thumb);
	box-shadow: inset 0 -3px 0 var(--_thumb-edge);
	clip-path: ${notch('var(--_n)')};
	border-radius: calc(7px * (1 - var(--_notch)));
	cursor: grab;
`

const styles = /* css */ `
:host {
	--_track: var(--sb-surface-inset, #0B1224);
	--_border: var(--sb-border, #283552);
	--_fill: var(--sb-brand, #8C6BFF);
	--_thumb: var(--sb-text-1, #F3F4FA);
	--_thumb-edge: var(--sb-brand-light, #B09AFF);
	--_label: var(--sb-text-2, #AEBBDD);
	--_muted: var(--sb-text-muted, #7785A8);
	--_value: var(--sb-text-1, #F3F4FA);
	--_notch: var(--sb-notch, 1);
	--_n: calc(2px * var(--_notch));
	display: block;
	inline-size: 100%;
	min-inline-size: 8rem;
}
:host([disabled]) { opacity: 0.5; pointer-events: none; }
.field { display: grid; gap: 0.5rem; }
.head { display: flex; justify-content: space-between; align-items: baseline; gap: 1rem; font-size: 0.8125rem; }
.label { color: var(--_label); font-weight: 600; }
output { color: var(--_value); font-variant-numeric: tabular-nums; font-weight: 700; }
/* Two native range inputs on one rail: only their thumbs take the pointer.
   The fill runs between the thumbs' centres (--_a, --_b are 0..1), which
   sit 7px (half a thumb) inside the ends. */
.rail { position: relative; block-size: 20px; }
.rail::before {
	--_x: calc(7px + (100% - 14px) * var(--_a));
	--_y: calc(7px + (100% - 14px) * var(--_b));
	content: "";
	position: absolute;
	inset-inline: 0;
	inset-block-start: 6px;
	block-size: 8px;
	background: linear-gradient(to right, var(--_track) var(--_x), var(--_fill) var(--_x), var(--_fill) var(--_y), var(--_track) var(--_y));
	box-shadow: 0 0 0 2px var(--_border);
	clip-path: ${notch('var(--_n)')};
	border-radius: calc(4px * (1 - var(--_notch)));
}
input {
	position: absolute;
	inset: 0;
	inline-size: 100%;
	block-size: 20px;
	margin: 0;
	background: transparent;
	appearance: none;
	pointer-events: none;
}
/* When the thumbs meet in the upper half, the start thumb goes on top, so it
   can still be dragged down. */
input.top { z-index: 1; }
input::-webkit-slider-runnable-track { background: transparent; block-size: 20px; }
input::-moz-range-track { background: transparent; }
input::-webkit-slider-thumb { appearance: none; ${thumb} transition: translate 80ms; }
input::-moz-range-thumb { ${thumb} }
input:active::-webkit-slider-thumb { translate: 0 1px; cursor: grabbing; }
input:focus-visible { outline: none; }
input:focus-visible::-webkit-slider-thumb { outline: 2px solid var(--_thumb-edge); outline-offset: 2px; }
input:focus-visible::-moz-range-thumb { outline: 2px solid var(--_thumb-edge); outline-offset: 2px; }
.ticks { display: flex; justify-content: space-between; color: var(--_muted); font-size: 0.6875rem; font-variant-numeric: tabular-nums; }
`

rocket('sb-range', {
	props: ({ bool, number, object, string }) => ({
		// A missing start or end means the whole range (the codec defaults are
		// clamped to min and max).
		value: object({ start: number.default(-Infinity), end: number.default(Infinity) }).docs({
			description: 'The range, as JSON: {"start": 20, "end": 60}; a missing part is min or max. A new value from the server replaces it; the live value is the value property.',
		}),
		min: number.docs({ description: 'Minimum.' }),
		max: number.default(100).docs({ description: 'Maximum.' }),
		step: number.min(0).default(1).docs({ description: 'Step size (0 for continuous).' }),
		label: string.trim.docs({ description: 'Visible label; the thumbs are announced as "<label> start" and "<label> end".' }),
		unit: string.docs({ description: 'Suffix after the shown values, e.g. "°" or " kg".' }),
		showValue: bool.default(true).docs({ description: 'Show the current range next to the label.' }),
		ticks: bool.docs({ description: 'Show min and max below the track.' }),
		disabled: bool.docs({ description: 'Disable interaction.' }),
		confirm: bool.docs({ description: 'Server-confirmed value: :state(pending) while the local range differs from the server\'s value attribute (see revert()).' }),
		name: string.trim.docs({ description: 'Name reported in sb-change (e.g. the field of a command).' }),
	}),
	manifest: {
		events: [
			{ name: 'input', kind: 'event', bubbles: true, composed: true, description: 'While dragging either thumb (native, re-targeted to the host).' },
			{ name: 'change', kind: 'event', bubbles: true, composed: true, description: 'When the range is committed.' },
			{ name: 'sb-change', kind: 'custom-event', bubbles: true, composed: true, description: 'When the range is committed. detail: { name, value: { start, end } }: the whole range, one command.' },
		],
	},
	setup: ({ $$, action, adoptStyles, defineHostProp, effect, emit, host, observeProps, overrideProp, props }) => {
		adoptStyles(host, styles)
		const clamp = (v) => Math.min(props.max, Math.max(props.min, Number.isNaN(v) ? props.min : v))
		// A range in order and in bounds; accepts {start, end} or [start, end].
		const norm = (v) => {
			const [a, b] = Array.isArray(v) ? v : [v?.start, v?.end]
			const s = clamp(Number(a ?? -Infinity))
			const e = clamp(Number(b ?? Infinity))
			return s <= e ? { start: s, end: e } : { start: e, end: s }
		}
		const set = (v) => peek(() => ({ start: $$.start, end: $$.end } = norm(v)))
		set(props.value)
		// The server's value attribute wins when it changes; a removed attribute
		// changes nothing (morphs also strip reflected ones). New bounds re-clamp.
		observeProps((p, changes) => set('value' in changes && host.hasAttribute('value') ? p.value : peek(() => ({ start: $$.start, end: $$.end }))), 'value', 'min', 'max')
		overrideProp('value', () => peek(() => ({ start: $$.start, end: $$.end })), set)
		// Commands: the attribute is the server's range, $$ the local one. Both
		// ends are one value: pending while either differs, revert() restores both.
		const states = internalsOf(host).states
		const sync = () =>
			peek(() => {
				const s = norm(props.value)
				props.confirm && (s.start !== $$.start || s.end !== $$.end) ? states.add('pending') : states.delete('pending')
			})
		effect(() => ($$.start, $$.end, sync()))
		observeProps(sync)
		defineHostProp('revert', { value: () => (set(props.value), sync()) })
		const decimals = () => (String(props.step).split('.')[1] || '').length
		const fmt = (n) => Number(n).toFixed(decimals()) + props.unit
		$$.startText = () => fmt($$.start)
		$$.endText = () => fmt($$.end)
		$$.shown = () => ($$.start === $$.end ? $$.startText : `${$$.startText} – ${$$.endText}`)
		action('commit', () => {
			emit('change')
			emit('sb-change', { name: props.name, value: { start: $$.start, end: $$.end } })
		})
	},
	// The fill between the thumbs follows the range through custom properties.
	onFirstRender: ({ $$, effect, host, props }) => {
		const rail = host.shadowRoot.querySelector('.rail')
		const at = (v) => String((v - props.min) / (props.max - props.min || 1))
		effect(() => {
			rail.style.setProperty('--_a', at($$.start))
			rail.style.setProperty('--_b', at($$.end))
		})
	},
	render: ({ html, props: { min, max, step, label, showValue, ticks, disabled, unit } }) => html`
		<div class="field" role="group" aria-label="${label || null}">
			${label || showValue ? html`
				<span class="head">
					<span class="label" part="label">${label}</span>
					${showValue ? html`<output part="value" data-text="$$shown"></output>` : null}
				</span>` : null}
			<span class="rail" part="rail">
				<input
					type="range"
					part="input start"
					min="${min}"
					max="${max}"
					step="${step || 'any'}"
					aria-label="${(label || 'Range') + ' start'}"
					disabled="${disabled}"
					data-class:top="$$start >= ${(min + max) / 2}"
					data-attr:aria-valuetext="$$startText"
					data-effect="el.value != $$start && (el.value = $$start)"
					data-on:input="$$start = Math.min(+el.value, $$end); el.value = $$start"
					data-on:change="@commit()"
				/>
				<input
					type="range"
					part="input end"
					min="${min}"
					max="${max}"
					step="${step || 'any'}"
					aria-label="${(label || 'Range') + ' end'}"
					disabled="${disabled}"
					data-attr:aria-valuetext="$$endText"
					data-effect="el.value != $$end && (el.value = $$end)"
					data-on:input="$$end = Math.max(+el.value, $$start); el.value = $$end"
					data-on:change="@commit()"
				/>
			</span>
			${ticks ? html`<span class="ticks" aria-hidden="true"><span>${min}${unit}</span><span>${max}${unit}</span></span>` : null}
		</div>
	`,
})
