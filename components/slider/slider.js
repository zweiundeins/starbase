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

const styles = /* css */ `
:host {
	--_track: var(--sb-surface-inset, #0B1224);
	--_border: var(--sb-border, #283552);
	--_fill: var(--sb-brand, #8C6BFF);
	--_thumb: var(--sb-slider-thumb, var(--sb-text-1, #F3F4FA));
	--_thumb-edge: var(--sb-slider-thumb-edge, var(--sb-brand-light, #B09AFF));
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
.rail { position: relative; block-size: 20px; display: grid; align-items: center; }
/* Pixel track: notched ends, filled part drawn with a gradient stop at --_p. */
.rail::before {
	content: "";
	position: absolute;
	inset-inline: 0;
	block-size: 8px;
	background: linear-gradient(to right, var(--_fill) var(--_p), var(--_track) var(--_p));
	box-shadow: 0 0 0 2px var(--_border);
	clip-path: ${notch('var(--_n)')};
	border-radius: calc(4px * (1 - var(--_notch)));
}
input {
	position: relative;
	inline-size: 100%;
	block-size: 20px;
	margin: 0;
	background: transparent;
	appearance: none;
	cursor: pointer;
}
input::-webkit-slider-runnable-track { background: transparent; block-size: 20px; }
input::-moz-range-track { background: transparent; }
input::-webkit-slider-thumb {
	appearance: none;
	inline-size: 14px;
	block-size: 20px;
	border: 0;
	background: var(--_thumb);
	box-shadow: inset 0 -3px 0 var(--_thumb-edge);
	clip-path: ${notch('var(--_n)')};
	border-radius: calc(7px * (1 - var(--_notch)));
	transition: translate 80ms;
}
input::-moz-range-thumb {
	inline-size: 14px;
	block-size: 20px;
	border: 0;
	border-radius: calc(7px * (1 - var(--_notch)));
	background: var(--_thumb);
	box-shadow: inset 0 -3px 0 var(--_thumb-edge);
}
input:active::-webkit-slider-thumb { translate: 0 1px; }
input:focus-visible { outline: 2px solid var(--_thumb-edge); outline-offset: 4px; }
.ticks { display: flex; justify-content: space-between; color: var(--_muted); font-size: 0.6875rem; font-variant-numeric: tabular-nums; }
`

rocket('sb-slider', {
	props: ({ bool, number, string }) => ({
		value: number.docs({ description: 'The value. A new value from the server replaces it; the live value is the value property.' }),
		min: number.docs({ description: 'Minimum.' }),
		max: number.default(100).docs({ description: 'Maximum.' }),
		step: number.min(0).default(1).docs({ description: 'Step size (0 for continuous).' }),
		label: string.trim.docs({ description: 'Visible label.' }),
		unit: string.docs({ description: 'Suffix after the shown value, e.g. "°" or "%".' }),
		showValue: bool.default(true).docs({ description: 'Show the current value next to the label.' }),
		ticks: bool.docs({ description: 'Show min and max below the track.' }),
		disabled: bool.docs({ description: 'Disable interaction.' }),
		confirm: bool.docs({ description: 'Server-confirmed value: :state(pending) while the local value differs from the server\'s value attribute (see revert()).' }),
		name: string.trim.docs({ description: 'Name reported in sb-change (e.g. the field of a command).' }),
	}),
	manifest: {
		events: [
			{ name: 'input', kind: 'event', bubbles: true, composed: true, description: 'While dragging (native, re-targeted to the host).' },
			{ name: 'change', kind: 'event', bubbles: true, composed: true, description: 'When the value is committed.' },
			{ name: 'sb-change', kind: 'custom-event', bubbles: true, composed: true, description: 'When the value is committed. detail: { name, value }: ready for a command.' },
		],
	},
	setup: ({ $$, action, adoptStyles, defineHostProp, effect, emit, host, observeProps, overrideProp, props }) => {
		adoptStyles(host, styles)
		const clamp = (v) => Math.min(props.max, Math.max(props.min, Number.isFinite(v) ? v : props.min))
		const decimals = () => (String(props.step).split('.')[1] || '').length
		$$.value = clamp(props.value)
		// A value attribute sent by the server wins when it changes (a morph
		// with a new value); re-sending the same markup changes nothing, so edits
		// survive re-renders. A *removed* attribute changes nothing either: morphs
		// also remove attributes that were only reflected (e.g. from a data-bind
		// write before the upgrade). To clear it, the server sends a new value.
		// A new min or max re-clamps the current value.
		observeProps((p, changes) => peek(() => ($$.value = clamp('value' in changes && host.hasAttribute('value') ? p.value : $$.value))), 'value', 'min', 'max')
		overrideProp('value', () => peek(() => $$.value), (v) => peek(() => ($$.value = clamp(Number(v)))))
		// Commands: the attribute is the server's value, $$.value the local one.
		// With confirm, :state(pending) marks an edit the server hasn't confirmed
		// yet; revert() returns to the server's value (e.g. a rejected command).
		const states = internalsOf(host).states
		const sync = () => peek(() => (props.confirm && $$.value !== clamp(props.value) ? states.add('pending') : states.delete('pending')))
		effect(() => ($$.value, sync()))
		observeProps(sync)
		defineHostProp('revert', { value: () => peek(() => (($$.value = clamp(props.value)), sync())) })
		$$.pct = () => ((($$.value - props.min) / (props.max - props.min || 1)) * 100).toFixed(2) + '%'
		$$.shown = () => Number($$.value).toFixed(decimals()) + props.unit
		action('commit', () => {
			emit('change')
			emit('sb-change', { name: props.name, value: $$.value })
		})
	},
	// The filled part of the track follows the value through a custom
	// property on the rail.
	onFirstRender: ({ $$, effect, host }) => {
		const rail = host.shadowRoot.querySelector('.rail')
		effect(() => rail.style.setProperty('--_p', $$.pct))
	},
	render: ({ html, props: { min, max, step, label, showValue, ticks, disabled, unit } }) => html`
		<label class="field">
			${label || showValue ? html`
				<span class="head">
					<span class="label" part="label">${label}</span>
					${showValue ? html`<output part="value" data-text="$$shown"></output>` : null}
				</span>` : null}
			<span class="rail">
				<input
					type="range"
					part="input"
					min="${min}"
					max="${max}"
					step="${step || 'any'}"
					aria-label="${label ? null : 'Value'}"
					disabled="${disabled}"
					data-effect="el.value != $$value && (el.value = $$value)"
					data-on:input="$$value = +el.value"
					data-on:change="@commit()"
				/>
			</span>
			${ticks ? html`<span class="ticks" aria-hidden="true"><span>${min}${unit}</span><span>${max}${unit}</span></span>` : null}
		</label>
	`,
})
