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
	transition: background 180ms steps(3);
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
	transition: inset-inline-start 180ms steps(5);
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
	setup: ({ $$, action, adoptStyles, cleanup, defineHostProp, effect, emit, host, observeProps, overrideProp, props }) => {
		adoptStyles(host, styles)
		// Kept across moves (Rocket drops $$ on disconnect, setup runs again on
		// re-attach) on the element's own ElementInternals: { on: the live
		// state, served: the server's last word }.
		const keep = internalsOf(host)
		// Interaction state lives in a local signal, never reflected.
		$$.on = keep.on ?? props.checked
		// A checked attribute sent by the server wins when it says something new
		// (a morph with a new value); re-sending the same markup changes nothing,
		// so edits survive re-renders. A *removed* attribute is ignored (morphs
		// also remove attributes that were only reflected, e.g. from a data-bind
		// write before the upgrade), but it clears the server's last word, so
		// sending the attribute again counts. To switch it off, the server sends
		// checked="false". Not observeProps: it only fires when the decoded value
		// changes, and checked="false" onto an element without the attribute
		// decodes to the same false. The same observer hands the host's
		// aria-label to the switch (a host label doesn't reach inside).
		const attrs = () =>
			peek(() => {
				$$.aria = host.getAttribute('aria-label') || 'Toggle'
				if (!host.hasAttribute('checked')) return void (keep.served = null)
				if (props.checked !== keep.served) $$.on = keep.served = props.checked
			})
		attrs() // also catches what the server said while the element was detached
		const watch = new MutationObserver(attrs)
		watch.observe(host, { attributeFilter: ['checked', 'aria-label'] })
		cleanup(() => watch.disconnect())
		overrideProp('checked', () => peek(() => $$.on), (v) => peek(() => ($$.on = !!v)))
		// Commands: the attribute is the server's value, $$.on the local one.
		// With confirm, :state(pending) marks an edit the server hasn't confirmed
		// yet; revert() returns to the server's value (e.g. a rejected command).
		const states = keep.states
		const sync = () => peek(() => (props.confirm && $$.on !== props.checked ? states.add('pending') : states.delete('pending')))
		// Rocket's disconnect wipes $$ (re-running this) before a move re-attaches.
		effect(() => ((keep.on = $$.on ?? keep.on), sync()))
		observeProps(sync)
		const revert = () => peek(() => (($$.on = props.checked), sync()))
		defineHostProp('revert', { value: revert })
		action('toggle', () => {
			if (props.disabled) return
			$$.on = !$$.on
			emit('change')
			emit('sb-change', { name: props.name, value: $$.on, checked: $$.on })
		})
		// Forms: until Rocket can make this element form-associated, join the
		// submissions and resets of the form it sits in. Like a checkbox without
		// a value: name=on when on, nothing when off. `formdata` also fires for
		// new FormData(form), so Datastar's contentType: 'form' posts include it.
		const form = host.closest('form')
		const root = host.getRootNode()
		const onData = (evt) => peek(() => props.name && !props.disabled && $$.on && evt.formData.append(props.name, 'on'))
		// A reset is revert() (the server's value, no change events), unless the
		// page cancelled it (e.g. a confirm() in data-on:reset), like a native
		// reset. Heard once it has bubbled up to the document (or shadow root),
		// so every listener on the form has run, whichever was bound first (a
		// form patched in binds its data-on:reset after this); a listener that
		// stops the reset's propagation hides it.
		const onReset = (evt) => evt.target == form && !evt.defaultPrevented && revert()
		form?.addEventListener('formdata', onData)
		root.addEventListener('reset', onReset)
		cleanup(() => (form?.removeEventListener('formdata', onData), root.removeEventListener('reset', onReset)))
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
