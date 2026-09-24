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

// Decimals as written: 0.05 -> 2.
const decimals = (n) => (String(n).split('.')[1] || '').length

// A detached range input puts a value in range and on a step exactly as the
// rendered one does (with the browser's decimal arithmetic).
const probe = document.createElement('input')
probe.type = 'range'

// Pixel corners: notches every corner by --_n (2px times --sb-notch; at 0 the
// border-radius takes over).
const notch = `polygon(var(--_n) 0, calc(100% - var(--_n)) 0, calc(100% - var(--_n)) var(--_n), 100% var(--_n), 100% calc(100% - var(--_n)), calc(100% - var(--_n)) calc(100% - var(--_n)), calc(100% - var(--_n)) 100%, var(--_n) 100%, var(--_n) calc(100% - var(--_n)), 0 calc(100% - var(--_n)), 0 var(--_n), var(--_n) var(--_n))`

const styles = /* css */ `
:host {
	--_thumb: var(--sb-slider-thumb, var(--sb-text-1, #F3F4FA));
	--_thumb-edge: var(--sb-slider-thumb-edge, var(--sb-brand-light, #B09AFF));
	--_notch: var(--sb-notch, 1);
	--_n: calc(2px * var(--_notch));
	display: block;
	inline-size: 100%;
	min-inline-size: 8rem;
}
:host([hidden]) { display: none; }
.field { display: grid; gap: 0.5rem; }
.field:has(:disabled) { opacity: 0.5; pointer-events: none; }
.head { display: flex; justify-content: space-between; align-items: baseline; gap: 1rem; font-size: 0.8125rem; }
.label { color: var(--sb-text-2, #AEBBDD); font-weight: 600; }
.value { color: var(--sb-text-1, #F3F4FA); font-variant-numeric: tabular-nums; font-weight: 700; }
.rail { position: relative; block-size: 20px; display: grid; align-items: center; }
input {
	position: relative;
	inline-size: 100%;
	block-size: 20px;
	margin: 0;
	background: transparent;
	cursor: pointer;
}
input:focus-visible { outline: 2px solid var(--_thumb-edge); outline-offset: 4px; }
.ticks { display: flex; justify-content: space-between; color: var(--sb-text-muted, #7785A8); font-size: 0.6875rem; font-variant-numeric: tabular-nums; }
/* The pixel look. With forced colours (high contrast) the native control
   stays: it draws itself in system colours. */
@media (forced-colors: none) {
	/* Pixel track: notched ends, the filled part drawn with a gradient stop
	   at --_p (from the right in right-to-left text). */
	.rail::before {
		content: "";
		position: absolute;
		inset-inline: 0;
		block-size: 8px;
		background: linear-gradient(to right, var(--sb-brand, #8C6BFF) var(--_p), var(--sb-surface-inset, #0B1224) var(--_p));
		box-shadow: 0 0 0 2px var(--sb-border, #283552);
		clip-path: ${notch};
		border-radius: calc(4px * (1 - var(--_notch)));
	}
	.rail:dir(rtl)::before { scale: -1 1; }
	input { appearance: none; }
	input::-webkit-slider-runnable-track { background: transparent; block-size: 20px; }
	input::-moz-range-track { background: transparent; }
	input::-webkit-slider-thumb {
		appearance: none;
		inline-size: 14px;
		block-size: 20px;
		border: 0;
		background: var(--_thumb);
		box-shadow: inset 0 -3px 0 var(--_thumb-edge);
		clip-path: ${notch};
		border-radius: calc(7px * (1 - var(--_notch)));
		transition: translate 80ms;
	}
	input::-moz-range-thumb {
		inline-size: 14px;
		block-size: 20px;
		border: 0;
		background: var(--_thumb);
		box-shadow: inset 0 -3px 0 var(--_thumb-edge);
		clip-path: ${notch};
		border-radius: calc(7px * (1 - var(--_notch)));
	}
	input:active::-webkit-slider-thumb { translate: 0 1px; }
}
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
	setup: ({ $$, action, adoptStyles, cleanup, defineHostProp, effect, emit, host, observeProps, overrideProp, props }) => {
		adoptStyles(host, styles)
		const clamp = (v) => (Object.assign(probe, { min: props.min, max: props.max, step: props.step || 'any', value: Number.isFinite(v) ? v : props.min }), +probe.value)
		const states = internalsOf(host).states
		// With confirm: the values committed while pending. The server echoes
		// each one; an echo of an older one, with newer ones still on their
		// way, must not pull the thumb back. Emptied whenever the local value
		// and the server's agree.
		let sent = []
		$$.value = clamp(props.value)
		// Runs on every value and prop change. A new range or step moves the
		// value into it; from the value derive the fill, the text (with the
		// decimals of the step and of min, 2 when continuous) and, with confirm,
		// :state(pending) while the local value differs from the server's
		// (revert() goes back to it, e.g. when a command is rejected).
		const sync = () =>
			peek(() => {
				const v = ($$.value = clamp($$.value))
				$$.pct = ((v - props.min) / (props.max - props.min || 1)) * 100 + '%'
				$$.shown = v.toFixed(Math.max(decimals(props.step || 0.01), decimals(props.min))) + props.unit
				props.confirm && v !== clamp(props.value) ? states.add('pending') : (states.delete('pending'), (sent = []))
			})
		// (No value: the element is being disconnected, its signals are gone.)
		effect(() => $$.value != null && sync())
		observeProps(sync)
		// A value attribute sent by the server wins when it changes; re-sending
		// the same markup changes nothing, so edits survive re-renders. A
		// *removed* attribute changes nothing either: morphs also remove
		// attributes that were only reflected (e.g. from a data-bind write
		// before the upgrade). The attribute is watched, not the prop: value="0"
		// on a slider rendered without one decodes to the same 0. The callback
		// runs after the whole morph, when a new range has arrived as well.
		let served = host.hasAttribute('value') ? props.value : null
		const watch = new MutationObserver(() =>
			peek(() => {
				if (!host.hasAttribute('value')) return void (served = null)
				if (props.value === served) return
				served = props.value
				// Drop the commits up to this echo (all of them when it echoes none).
				sent = sent.slice(sent.indexOf(served) + 1 || sent.length)
				sent.length || ($$.value = clamp(served))
			}),
		)
		watch.observe(host, { attributeFilter: ['value'] })
		cleanup(() => watch.disconnect())
		overrideProp('value', () => peek(() => $$.value), (v) => peek(() => ($$.value = clamp(Number(v)))))
		defineHostProp('revert', { value: () => peek(() => ($$.value = clamp(props.value))) })
		action('commit', () => {
			states.has('pending') && sent.push($$.value)
			emit('change')
			emit('sb-change', { name: props.name, value: $$.value })
		})
	},
	// The fill follows the value through --_p on the rail (data-style sets it
	// again after a re-render). The input's effect names the range, so it runs
	// again once a re-render has given the input a new one.
	render: ({ html, host, props: { min, max, step, label, showValue, ticks, disabled, unit } }) => html`
		<label class="field">
			${label || showValue ? html`
				<span class="head">
					<span class="label" part="label">${label}</span>
					${showValue ? html`<span class="value" part="value" data-text="$$shown"></span>` : null}
				</span>` : null}
			<span class="rail" data-style:--_p="$$pct">
				<input
					type="range"
					part="input"
					min="${min}"
					max="${max}"
					step="${step || 'any'}"
					aria-label="${label || host.ariaLabel || 'Value'}"
					data-attr:aria-valuetext="$$shown"
					disabled="${disabled}"
					data-effect="${min},${max},${step}, el.value != $$value && (el.value = $$value)"
					data-on:input="$$value = +el.value"
					data-on:change="@commit()"
				/>
			</span>
			${ticks ? html`<span class="ticks" aria-hidden="true"><span>${min}${unit}</span><span>${max}${unit}</span></span>` : null}
		</label>
	`,
})
