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

// The element's ElementInternals: attachInternals() works only once, and setup
// runs again when the element is re-attached. Its custom states
// (:state(pending)) are styleable from the page and morph-proof.
const internals = new WeakMap()
const internalsOf = (host) => internals.get(host) ?? internals.set(host, host.attachInternals()).get(host)

// A detached range input puts a value in range and on the step grid exactly
// as the thumbs do, in decimal arithmetic (0.7 / 0.1 is 6.999… in floats).
const probe = document.createElement('input')
probe.type = 'range'

// The thumb, for WebKit and Gecko. Its focus ring (--_ring) is drawn inside:
// the clip-path cuts off anything outside.
const thumb = /* css */ `
	pointer-events: auto;
	inline-size: 14px;
	block-size: 20px;
	border: 0;
	background: var(--_thumb);
	box-shadow: inset 0 -3px 0 var(--_thumb-edge);
	clip-path: var(--_clip);
	border-radius: calc(7px * (1 - var(--_notch)));
	cursor: grab;
	outline: var(--_ring, 0);
	outline-offset: -3px;
`

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
	/* Pixel corners for the track and the thumbs: every corner notched by
	   --_n (at --sb-notch: 0 their border-radius takes over). */
	--_clip: polygon(var(--_n) 0, calc(100% - var(--_n)) 0, calc(100% - var(--_n)) var(--_n), 100% var(--_n), 100% calc(100% - var(--_n)), calc(100% - var(--_n)) calc(100% - var(--_n)), calc(100% - var(--_n)) 100%, var(--_n) 100%, var(--_n) calc(100% - var(--_n)), 0 calc(100% - var(--_n)), 0 var(--_n), var(--_n) var(--_n));
	display: block;
	inline-size: 100%;
	min-inline-size: 8rem;
}
:host([hidden]) { display: none; }
:host([disabled]) { opacity: 0.5; pointer-events: none; }
.field { display: grid; gap: 0.5rem; }
.head { display: flex; justify-content: space-between; align-items: baseline; gap: 1rem; font-size: 0.8125rem; }
.label { color: var(--_label); font-weight: 600; }
.value { color: var(--_value); font-variant-numeric: tabular-nums; font-weight: 700; }
/* Two native range inputs on one rail: only their thumbs take the pointer.
   The fill runs between the thumbs' centres (--_a, --_b are 0..1), which
   sit 7px (half a thumb) inside the ends; a stop at 0 starts where the one
   before it ends. */
.rail { position: relative; block-size: 20px; }
.rail::before {
	content: "";
	position: absolute;
	inset-inline: 0;
	inset-block-start: 6px;
	block-size: 8px;
	background: linear-gradient(to right, var(--_track) calc(7px + (100% - 14px) * var(--_a)), var(--_fill) 0 calc(7px + (100% - 14px) * var(--_b)), var(--_track) 0);
	box-shadow: 0 0 0 2px var(--_border);
	clip-path: var(--_clip);
	border-radius: calc(4px * (1 - var(--_notch)));
}
.rail:dir(rtl)::before { scale: -1 1; }
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
input:focus-visible { outline: none; --_ring: 3px solid var(--_thumb-edge); }
.ticks { display: flex; justify-content: space-between; color: var(--_muted); font-size: 0.6875rem; font-variant-numeric: tabular-nums; }
`

rocket('sb-range', {
	props: ({ bool, json, number, string }) => ({
		value: json.docs({
			description: 'The range, as JSON: {"start": 20, "end": 60}; a missing or null part is min or max. A new value from the server replaces it; the live value is the value property.',
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
		name: string.trim.docs({ description: 'Name reported in sb-change (e.g. the field of a command) and submitted with a <form>.' }),
	}),
	manifest: {
		events: [
			{ name: 'input', kind: 'event', bubbles: true, composed: true, description: 'While dragging either thumb (native, re-targeted to the host).' },
			{ name: 'change', kind: 'event', bubbles: true, composed: true, description: 'When the range is committed.' },
			{ name: 'sb-change', kind: 'custom-event', bubbles: true, composed: true, description: 'When the range is committed. detail: { name, value: { start, end } }: the whole range, one command.' },
		],
	},
	setup: ({ $$, action, adoptStyles, cleanup, defineHostProp, effect, emit, host, observeProps, overrideProp, props }) => {
		adoptStyles(host, styles)
		// Decimals of the step grid (step and min); continuous shows two.
		const dec = (n) => (String(n).split('.')[1] || '').length
		const places = () => (props.step ? Math.max(dec(props.step), dec(props.min)) : 2)
		// In bounds and on the step grid, like the thumbs (whose last stop can
		// be below max). ±Infinity (a missing part) and NaN become a bound first:
		// the probe would take them as its middle.
		const clamp = (v, { min, max, step } = props) => (Object.assign(probe, { min, max, step: step || 'any', value: isFinite(v) ? v : v > 0 ? max : min }), +probe.value)
		// A range in order and in bounds; accepts {start, end} or [start, end].
		const norm = (v) => {
			const [a, b] = Array.isArray(v) ? v : [v?.start, v?.end]
			const s = clamp(Number(a ?? -Infinity))
			const e = clamp(Number(b ?? Infinity))
			return s <= e ? { start: s, end: e } : { start: e, end: s }
		}
		const cur = () => peek(() => ({ start: $$.start, end: $$.end }))
		const set = (v) => peek(() => ({ start: $$.start, end: $$.end } = norm(v)))
		set(props.value)
		// The server's value attribute wins when it changes; a removed attribute
		// changes nothing (morphs also strip reflected ones). New bounds or step
		// re-clamp a local edit, and re-apply the server's range otherwise: a
		// morph sets attributes one by one, so the value can come before its
		// bounds. (Only for these props, not merged with sync's observer: inside
		// a Datastar batch, e.g. el.value = …; el.unit = … in one handler, mine
		// is still stale, and the server's range would replace the edit.)
		let mine
		observeProps((p, changes) => set(host.hasAttribute('value') && ('value' in changes || !mine) ? p.value : cur()), 'value', 'min', 'max', 'step')
		overrideProp('value', cur, set)
		// Commands: the attribute is the server's range, $$ the local one. Both
		// ends are one value: pending while either differs, revert() restores both.
		const states = internalsOf(host).states
		// Also keeps the shown texts current: props (unit, step) aren't signals.
		const sync = () =>
			peek(() => {
				const s = norm(props.value)
				const fmt = (n) => n.toFixed(places()) + props.unit
				mine = s.start !== $$.start || s.end !== $$.end
				states[props.confirm && mine ? 'add' : 'delete']('pending')
				const a = ($$.startText = fmt($$.start))
				const b = ($$.endText = fmt($$.end))
				$$.shown = a === b ? a : `${a} – ${b}`
			})
		// (No start: the element is being removed and its signals are gone;
		// writing the texts would bring them back.)
		effect(() => $$.start != null && ($$.end, sync()))
		observeProps(sync)
		const revert = () => set(props.value)
		defineHostProp('revert', { value: revert })
		// Forms: until Rocket can make this element form-associated, join the
		// submissions and resets of the form it sits in. `formdata` also fires for
		// new FormData(form), so Datastar's contentType: 'form' posts include it.
		// The entry is the local range in the value attribute's JSON; a reset is
		// revert(): the server's range, no events, like a native reset.
		const form = host.closest('form')
		const onData = (evt) => props.name && !props.disabled && evt.formData.append(props.name, JSON.stringify(cur()))
		form?.addEventListener('formdata', onData)
		form?.addEventListener('reset', revert)
		cleanup(() => (form?.removeEventListener('formdata', onData), form?.removeEventListener('reset', revert)))
		// Both inputs report here. A pointer that grabs the thumbs where they
		// meet ($$tie) picks the part with its first move: down moves the start,
		// up the end. When that is the other input's part, the inputs swap what
		// they show ($$swap) until the commit, so the dragged thumb shows it.
		// Such a drag can end where it began, without a change event, so
		// pointerup (and pointercancel) commit too. A commit only emits if the
		// range differs from the one before the drag or key press: a thumb
		// stopped by the other one still fires change.
		$$.swap = false // before the render, so the inputs' effects follow it
		let from
		action('slide', ({ evt: { target: t } }) =>
			peek(() => {
				const v = +t.value
				from ??= cur()
				if ($$.tie === true) $$.tie = v < $$.start ? 'start' : 'end'
				const k = $$.tie || t.name
				$$[k] = k === 'end' ? Math.max(v, $$.start) : Math.min(v, $$.end)
				$$.swap = k !== t.name
				t.value = $$[k]
			}),
		)
		action('commit', () => {
			const v = cur()
			if (from && (from.start !== v.start || from.end !== v.end)) {
				emit('change')
				emit('sb-change', { name: props.name, value: v })
			}
			from = $$.tie = null
			$$.swap = false
		})
	},
	render: ({ html, props: { min, max, step, label, showValue, ticks, disabled, unit } }) => html`
		<div class="field" role="group" aria-label="${label || null}">
			${label || showValue ? html`
				<span class="head">
					<span class="label" part="label">${label}</span>
					${showValue ? html`<span class="value" part="value" data-text="$$shown"></span>` : null}
				</span>` : null}
			<span
				class="rail"
				part="rail"
				data-style:--_a="($$start - ${min}) / ${max - min || 1}"
				data-style:--_b="($$end - ${min}) / ${max - min || 1}"
				data-on:pointerdown="$$tie = $$start == $$end"
				data-on:pointerup="@commit()"
				data-on:pointercancel="@commit()"
				data-on:input="@slide()"
				data-on:change="@commit()"
			>
				<input
					type="range"
					name="start"
					part="input start"
					min="${min}"
					max="${max}"
					step="${step || 'any'}"
					aria-label="${label || 'Range'} start"
					disabled="${disabled}"
					data-class:top="$$start >= ${(min + max) / 2}"
					data-attr:aria-valuetext="$$startText"
					data-effect="${min}, ${max}, ${step}, el.value = $$swap ? $$end : $$start"
				/>
				<input
					type="range"
					name="end"
					part="input end"
					min="${min}"
					max="${max}"
					step="${step || 'any'}"
					aria-label="${label || 'Range'} end"
					disabled="${disabled}"
					data-attr:aria-valuetext="$$endText"
					data-effect="${min}, ${max}, ${step}, el.value = $$swap ? $$start : $$end"
				/>
			</span>
			${ticks ? html`<span class="ticks" aria-hidden="true"><span>${min}${unit}</span><span>${max}${unit}</span></span>` : null}
		</div>
	`,
})
