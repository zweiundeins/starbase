import { rocket, startPeeking, stopPeeking } from 'datastar'

// Host getters must not subscribe callers (e.g. data-bind's sync effect) to
// the internal signal, or that effect writes the stale bound value back.
// data-bind may set a property before the element is upgraded; adopt it.
const early = (host, name) => {
	const d = Object.getOwnPropertyDescriptor(host, name)
	if (!d || !('value' in d)) return undefined
	delete host[name]
	return d.value
}

const peek = (fn) => {
	startPeeking()
	try {
		return fn()
	} finally {
		stopPeeking()
	}
}

// Pixel corners: a polygon that notches every corner by one "pixel".
const notch = (p) => `polygon(${p} 0, calc(100% - ${p}) 0, calc(100% - ${p}) ${p}, 100% ${p}, 100% calc(100% - ${p}), calc(100% - ${p}) calc(100% - ${p}), calc(100% - ${p}) 100%, ${p} 100%, ${p} calc(100% - ${p}), 0 calc(100% - ${p}), 0 ${p}, ${p} ${p})`

const styles = /* css */ `
:host {
	--_track: var(--sb-border-strong, #3A4868);
	--_on: var(--sb-brand, #8C6BFF);
	--_knob: var(--sb-text-1, #F3F4FA);
	--_text: var(--sb-text-1, #F3F4FA);
	--_focus: var(--sb-brand-light, #B09AFF);
	display: inline-flex;
	vertical-align: middle;
}
:host([disabled]) { opacity: 0.5; pointer-events: none; }
label { display: inline-flex; align-items: center; gap: 0.75em; color: var(--_text); cursor: pointer; }
button {
	all: unset;
	position: relative;
	inline-size: calc(var(--_u) * 11);
	block-size: calc(var(--_u) * 6);
	background: var(--_track);
	clip-path: ${notch('var(--_u)')};
	cursor: pointer;
	transition: background 180ms steps(3, end);
}
.sm { --_u: 3px; }
.md { --_u: 4px; }
.lg { --_u: 7px; }
.knob {
	position: absolute;
	inset-block: var(--_u);
	inset-inline-start: var(--_u);
	inline-size: calc(var(--_u) * 4);
	background: var(--_knob);
	clip-path: ${notch('calc(var(--_u) / 2)')};
	transition: translate 180ms steps(5, end);
}
button.on { background: var(--_on); }
button.on .knob { translate: calc(var(--_u) * 5) 0; }
button:focus-visible { outline: 2px solid var(--_focus); outline-offset: 3px; clip-path: none; }
@media (prefers-reduced-motion: reduce) { .knob, button { transition: none; } }
`

rocket('sb-flip-switch', {
	props: ({ bool, oneOf, string }) => ({
		checked: bool.docs({ description: 'Initial state. Read the live state from the checked property.' }),
		disabled: bool.docs({ description: 'Disable interaction.' }),
		label: string.trim.docs({ description: 'Visible label next to the switch.' }),
		size: oneOf('sm', 'md', 'lg').default('md').docs({ description: 'Size of the switch.' }),
	}),
	manifest: {
		events: [
			{ name: 'change', kind: 'event', bubbles: true, composed: true, description: 'After the user flips the switch.' },
			{ name: 'sb-change', kind: 'custom-event', bubbles: true, composed: true, description: 'Same moment. detail: { checked }.' },
		],
	},
	setup: ({ $$, action, adoptStyles, emit, host, observeProps, overrideProp, props }) => {
		adoptStyles(host, styles)
		// Interaction state lives in a local signal, never in the attribute:
		// a server morph may re-send the original markup at any time.
		$$.on = props.checked
		const pre = early(host, 'checked')
		$$.dirty = pre !== undefined
		if ($$.dirty) $$.on = !!pre
		// Like a native <input>: the attribute is only the default. Once the
		// checked state is "dirty" (edited, or set as a property, e.g. by data-bind),
		// attribute changes, including a server morph removing a reflected
		// attribute, no longer touch it.
		observeProps(() => peek(() => !$$.dirty && ($$.on = props.checked)), 'checked')
		overrideProp('checked', () => peek(() => $$.on), (v) => peek(() => (($$.dirty = true), ($$.on = !!v))))
		action('toggle', () => {
			if (props.disabled) return
			$$.dirty = true
			$$.on = !$$.on
			emit('change')
			emit('sb-change', { checked: $$.on })
		})
	},
	render: ({ html, props: { label, size, disabled } }) => html`
		<label>
			<button
				type="button"
				role="switch"
				part="switch"
				class="${size}"
				aria-label="${label ? null : 'Toggle'}"
				disabled="${disabled}"
				data-class:on="$$on"
				data-attr:aria-checked="String($$on)"
				data-on:click="@toggle()"
			><span class="knob"></span></button>
			${label ? html`<span part="label">${label}</span>` : null}
		</label>
	`,
})
