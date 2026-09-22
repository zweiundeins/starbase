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

// Pixel corners: a polygon that notches every corner by one "pixel"
// (times --sb-notch; 0 leaves the rectangle, rounded by border-radius).
const notch = (p) => `polygon(${p} 0, calc(100% - ${p}) 0, calc(100% - ${p}) ${p}, 100% ${p}, 100% calc(100% - ${p}), calc(100% - ${p}) calc(100% - ${p}), calc(100% - ${p}) 100%, ${p} 100%, ${p} calc(100% - ${p}), 0 calc(100% - ${p}), 0 ${p}, ${p} ${p})`

const styles = /* css */ `
:host {
	--_track: var(--sb-border-strong, #3A4868);
	--_on: var(--sb-brand, #8C6BFF);
	--_knob: var(--sb-text-1, #F3F4FA);
	--_text: var(--sb-text-1, #F3F4FA);
	--_focus: var(--sb-brand-light, #B09AFF);
	--_notch: var(--sb-notch, 1);
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
	clip-path: ${notch('calc(var(--_u) * var(--_notch))')};
	border-radius: calc(var(--_u) * 3 * (1 - var(--_notch)));
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
	clip-path: ${notch('calc(var(--_u) / 2 * var(--_notch))')};
	border-radius: calc(var(--_u) * 2 * (1 - var(--_notch)));
	transition: translate 180ms steps(5, end);
}
button.on { background: var(--_on); }
button.on .knob { translate: calc(var(--_u) * 5) 0; }
button:focus-visible { outline: 2px solid var(--_focus); outline-offset: 3px; clip-path: none; }
@media (prefers-reduced-motion: reduce) { .knob, button { transition: none; } }
`

rocket('sb-toggle', {
	props: ({ bool, oneOf, string }) => ({
		checked: bool.docs({ description: 'On or off. A new state from the server wins (send checked="false" to switch it off); the live state is the checked property.' }),
		disabled: bool.docs({ description: 'Disable interaction.' }),
		label: string.trim.docs({ description: 'Visible label next to the switch.' }),
		size: oneOf('sm', 'md', 'lg').default('md').docs({ description: 'Size of the switch.' }),
		confirm: bool.docs({ description: 'Server-confirmed value: :state(pending) while the local value differs from the server\'s value attribute (see revert()).' }),
		name: string.trim.docs({ description: 'Name reported in sb-change (e.g. the field of a command).' }),
	}),
	manifest: {
		events: [
			{ name: 'change', kind: 'event', bubbles: true, composed: true, description: 'After the user flips the switch.' },
			{ name: 'sb-change', kind: 'custom-event', bubbles: true, composed: true, description: 'Same moment. detail: { name, value, checked } (value is the checked state): ready for a command.' },
		],
	},
	setup: ({ $$, action, adoptStyles, defineHostProp, effect, emit, host, observeProps, overrideProp, props }) => {
		adoptStyles(host, styles)
		// Interaction state lives in a local signal, never reflected.
		$$.on = props.checked
		// A checked attribute sent by the server wins when it changes (a morph
		// with a new value); re-sending the same markup changes nothing, so edits
		// survive re-renders. A *removed* attribute changes nothing either: morphs
		// also remove attributes that were only reflected (e.g. from a data-bind
		// write before the upgrade). To clear it, the server sends checked="false".
		observeProps(() => peek(() => host.hasAttribute('checked') && ($$.on = props.checked)), 'checked')
		overrideProp('checked', () => peek(() => $$.on), (v) => peek(() => ($$.on = !!v)))
		// Commands: the attribute is the server's value, $$.on the local one.
		// With confirm, :state(pending) marks an edit the server hasn't confirmed
		// yet; revert() returns to the server's value (e.g. a rejected command).
		const states = internalsOf(host).states
		const sync = () => peek(() => (props.confirm && $$.on !== props.checked ? states.add('pending') : states.delete('pending')))
		effect(() => ($$.on, sync()))
		observeProps(sync)
		defineHostProp('revert', { value: () => peek(() => (($$.on = props.checked), sync())) })
		action('toggle', () => {
			if (props.disabled) return
			$$.on = !$$.on
			emit('change')
			emit('sb-change', { name: props.name, value: $$.on, checked: $$.on })
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
