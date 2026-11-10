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

// The server's word on a boolean is its attribute's presence. "false" still
// reads as absent until that spelling goes (deprecated, one warning per element).
const warned = new WeakSet()
const boolWord = (host, attr) => {
	const v = host.getAttribute(attr)
	if (v === 'false' && !warned.has(host)) {
		warned.add(host)
		console.warn(`${host.localName}: ${attr}="false" is deprecated; leave the attribute out.`, host)
	}
	return v !== null && v !== 'false'
}

// Pixel corners: a polygon that notches every corner by one "pixel"
// (times --sb-notch; 0 leaves the rectangle, rounded by border-radius).
const notch = (p) => `polygon(${p} 0, calc(100% - ${p}) 0, calc(100% - ${p}) ${p}, 100% ${p}, 100% calc(100% - ${p}), calc(100% - ${p}) calc(100% - ${p}), calc(100% - ${p}) 100%, ${p} 100%, ${p} calc(100% - ${p}), 0 calc(100% - ${p}), 0 ${p}, ${p} ${p})`

const styles = /* css */ `
:host {
	--_track: var(--sb-border-strong, #3A4868);
	--_on: var(--sb-brand, #8C6BFF);
	--_knob: var(--sb-toggle-knob, var(--sb-text-1, #F3F4FA));
	--_knob-on: var(--sb-toggle-knob, var(--sb-text-on-brand, #F3F4FA));
	--_focus: var(--sb-brand-light, #B09AFF);
	--_notch: var(--sb-notch, 1);
	display: inline-flex;
	vertical-align: middle;
}
:host([hidden]) { display: none; }
label { display: inline-flex; align-items: center; gap: 0.75em; cursor: pointer; }
label:has(:disabled) { opacity: 0.5; pointer-events: none; }
button {
	all: unset;
	position: relative;
	inline-size: calc(var(--_u) * 11);
	block-size: calc(var(--_u) * 6);
	background: var(--_track);
	clip-path: ${notch('calc(var(--_u) * var(--_notch))')};
	border-radius: calc(var(--_u) * 3 * (1 - var(--_notch)));
	/* cursor: the label's (all: unset inherits it) */
	transition: background 180ms steps(calc(3 + 997 * (1 - var(--_notch)))); /* stepped at notch 1, smooth at 0 */
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
	transition: inset-inline-start 180ms steps(calc(5 + 995 * (1 - var(--_notch))));
}
button.on { background: var(--_on); }
button.on .knob { inset-inline-start: calc(var(--_u) * 6); background: var(--_knob-on); }
button:focus-visible { outline: 2px solid var(--_focus); outline-offset: 3px; clip-path: none; }
@media (prefers-reduced-motion: reduce) { .knob, button { transition: none; } }
@media (forced-colors: active) {
	button { outline: 1px solid; outline-offset: -1px; }
	button.on { background: Highlight; }
	.knob { forced-color-adjust: none; background: CanvasText; }
	button.on .knob { background: HighlightText; }
}
`

rocket('sb-toggle', {
	props: ({ bool, oneOf, string }) => ({
		checked: bool.docs({ description: 'On when present. The server\'s word: adding or removing the attribute wins over a local edit, identical markup keeps it; the live state is the checked property. checked="false" reads as off but is deprecated: leave the attribute out.' }),
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
	setup: ({ $$, action, adoptStyles, cleanup, defineHostProp, effect, emit, host, observeProps, overrideProp, props }) => {
		adoptStyles(host, styles)
		// Kept across moves (Rocket drops $$ on disconnect, setup runs again on
		// re-attach) on the element's own ElementInternals: { on: the live
		// state, served: the server's last word }.
		const keep = internalsOf(host)
		// Interaction state lives in a local signal, never reflected.
		$$.on = keep.on ?? boolWord(host, 'checked')
		// Commands: the attribute is the server's value, $$.on the local one.
		// With confirm, :state(pending) marks an edit the server hasn't confirmed
		// yet; revert() returns to the server's value (e.g. a rejected command).
		const states = keep.states
		const sync = () => peek(() => (props.confirm && $$.on !== boolWord(host, 'checked') ? states.add('pending') : states.delete('pending')))
		// A change of the checked attribute wins over a local edit: added says on,
		// removed says off. Re-sending the same markup changes nothing, so edits
		// survive re-renders. Not observeProps: the patched build decodes
		// checked="false" onto an element without it to the same false. The same
		// observer hands the host's aria-label to the switch.
		const attrs = () =>
			peek(() => {
				$$.aria = host.getAttribute('aria-label') || 'Toggle'
				const now = boolWord(host, 'checked')
				if (now !== keep.served) $$.on = keep.served = now
				sync()
			})
		attrs() // also catches what the server said while the element was detached
		const watch = new MutationObserver(attrs)
		watch.observe(host, { attributeFilter: ['checked', 'aria-label'] })
		cleanup(() => watch.disconnect())
		overrideProp('checked', () => peek(() => $$.on), (v) => peek(() => ($$.on = !!v)))
		// Rocket's disconnect wipes $$ (re-running this) before a move re-attaches.
		effect(() => ((keep.on = $$.on ?? keep.on), sync()))
		observeProps(sync)
		const revert = () => peek(() => (($$.on = boolWord(host, 'checked')), sync()))
		defineHostProp('revert', { value: revert })
		action('toggle', () => {
			if (props.disabled) return
			$$.on = !$$.on
			emit('change')
			emit('sb-change', { name: props.name, value: $$.on, checked: $$.on })
		})
		// Forms: until Rocket can make this element form-associated, join the
		// submissions and resets of the form it sits in. `formdata` also fires for
		// new FormData(form), so Datastar's contentType: 'form' posts include it.
		// Both are heard on the root (document or shadow root), once every
		// listener on the form has run: a reset the page cancelled (whenever its
		// listener was added) leaves the value, as it leaves native fields, and a
		// form nested in this one by a script, whose events bubble through it,
		// isn't taken for it. setup reruns on a re-attach, so a move follows.
		// Like a checkbox without a value: name=on when on, nothing when off. A
		// reset is revert(): the server's value, no change events.
		const form = host.closest('form')
		const root = host.getRootNode()
		const onData = (evt) => evt.target === form && peek(() => props.name && !props.disabled && $$.on && evt.formData.append(props.name, 'on'))
		const onReset = (evt) => evt.target === form && !evt.defaultPrevented && revert()
		root.addEventListener('formdata', onData)
		root.addEventListener('reset', onReset)
		cleanup(() => (root.removeEventListener('formdata', onData), root.removeEventListener('reset', onReset)))
	},
	render: ({ html, props: { label, size, disabled } }) => html`
		<label>
			<button
				type="button"
				role="switch"
				part="switch"
				class="${size}"
				data-attr:aria-label="${label ? null : '$$aria'}"
				disabled="${disabled}"
				data-class:on="$$on"
				data-attr:aria-checked="String($$on)"
				data-on:click="@toggle()"
			><span class="knob"></span></button>
			${label ? html`<span part="label">${label}</span>` : null}
		</label>
	`,
})
