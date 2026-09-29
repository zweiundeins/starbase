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

// The check mark in 2px pixels, cut out of the box's 12px inside.
const check = 'path("M10 1h2v2h-2zM8 3h4v2H8zM0 5h2v2H0zM6 5h4v2H6zM0 7h8v2H0zM2 9h4v2H2z")'

// :state(disabled) comes from the decoded prop, so disabled="false" is not
// disabled. Forced colours paint every background Canvas: the box and its
// mark take the system's Highlight colours there.
const styles = /* css */ `
:host {
	--_bg: var(--sb-control-bg, #0B1224);
	--_border: var(--sb-control-border, #283552);
	--_border-hover: var(--sb-control-border-hover, #3A4868);
	--_text: var(--sb-control-text, #F3F4FA);
	--_brand: var(--sb-brand, #8C6BFF);
	--_brand-light: var(--sb-brand-light, #B09AFF);
	--_mark: var(--sb-text-on-brand, #F3F4FA);
	--_hover: var(--sb-surface-hover, #1A2540);
	--_radius: var(--sb-control-radius, 6px);
	--_notch: var(--sb-notch, 1);
	--_n: calc(2px * var(--_notch));
	display: inline-flex;
	vertical-align: middle;
}
:host([hidden]) { display: none; }
:host(:state(disabled)) { opacity: 0.5; pointer-events: none; }
.item {
	display: flex;
	align-items: flex-start;
	gap: 0.6rem;
	padding: 0.4rem 0.6rem;
	color: var(--_text);
	font-size: 0.875rem;
	cursor: pointer;
	clip-path: ${notch('var(--_n)')};
	border-radius: calc(var(--_radius) * (1 - var(--_notch)));
	transition: background 120ms;
}
@media (hover: hover) { .item:hover { background: var(--_hover); } }
.item:focus-visible { outline: 2px solid var(--_brand-light); outline-offset: 2px; clip-path: none; }
.box {
	position: relative;
	flex: none;
	box-sizing: border-box;
	width: 16px;
	height: 16px;
	margin-block-start: 0.1rem;
	border: 2px solid var(--_border);
	background: var(--_bg);
	clip-path: ${notch('var(--_n)')};
	border-radius: calc(4px * (1 - var(--_notch)));
	transition: border-color 120ms, background 120ms;
}
.item:hover .box { border-color: var(--_border-hover); }
.on .box, .mixed .box { border-color: var(--_brand-light); background: var(--_brand); }
.on .box::after, .mixed .box::after { content: ""; position: absolute; inset: 0; background: var(--_mark); clip-path: ${check}; }
.mixed .box::after { clip-path: inset(5px 2px); }
@media (prefers-reduced-motion: reduce) { .item, .box { transition: none; } }
@media (forced-colors: active) {
	.box { forced-color-adjust: none; border-color: CanvasText; background: Canvas; }
	.on .box, .mixed .box { border-color: Highlight; background: Highlight; }
	.box::after { forced-color-adjust: none; background: HighlightText; }
}
`

rocket('sb-checkbox', {
	props: ({ bool, string }) => ({
		checked: bool.docs({ description: 'Checked or not. A new state from the server wins (send checked="false" to clear it); the live state is the checked property.' }),
		indeterminate: bool.docs({ description: 'Shows the mixed state (the server says "some"). A click resolves it to checked; the live state is the indeterminate property.' }),
		label: string.trim.docs({ description: 'Visible label next to the box, and its accessible name.' }),
		disabled: bool.docs({ description: 'Disable interaction.' }),
		name: string.trim.docs({ description: 'Name reported in sb-change (e.g. the field of a command), and the field it submits in a form.' }),
		value: string.default('on').docs({ description: 'What a form submits under name while checked, like a native checkbox.' }),
		confirm: bool.docs({ description: 'Server-confirmed value: :state(pending) while the local state differs from the server\'s checked attribute (see revert()).' }),
	}),
	manifest: {
		events: [
			{ name: 'change', kind: 'event', bubbles: true, composed: true, description: 'After the user checks or clears the box.' },
			{ name: 'sb-change', kind: 'custom-event', bubbles: true, composed: true, description: 'Same moment. detail: { name, value, checked } (value is the checked state): ready for a command.' },
		],
	},
	// Rendered once: the state and the label flow through signals, so a new
	// prop never rebuilds the box and takes the keyboard focus with it.
	renderOnPropChange: false,
	setup: ({ $$, action, adoptStyles, cleanup, defineHostProp, effect, emit, host, observeProps, overrideProp, props }) => {
		adoptStyles(host, styles)
		// Kept across moves (Rocket drops $$ on disconnect, setup runs again on
		// re-attach) on the element's own ElementInternals: { on, mixed: the live
		// state, served, said: the server's last word on checked and indeterminate }.
		const keep = internalsOf(host)
		const states = keep.states
		const take = (p) => {
			$$.label = p.label
			$$.disabled = p.disabled
			states[p.disabled ? 'add' : 'delete']('disabled')
		}
		take(props)
		$$.on = keep.on ?? props.checked
		$$.mixed = keep.mixed ?? false
		// A checked attribute sent by the server wins when it says something new
		// (a morph with a new value); re-sending the same markup changes nothing,
		// so edits survive re-renders. A *removed* checked is ignored (morphs also
		// remove attributes that were only reflected), but it clears the server's
		// last word, so sending it again counts; checked="false" clears. Not
		// observeProps: it misses checked="false" onto an element without it.
		// indeterminate is the server's view state: any change of it wins. The
		// same observer hands the host's aria-label to the box.
		const attrs = () =>
			peek(() => {
				$$.aria = host.getAttribute('aria-label')
				if (props.indeterminate !== keep.said) $$.mixed = keep.said = props.indeterminate
				if (!host.hasAttribute('checked')) return void (keep.served = null)
				if (props.checked !== keep.served) $$.on = keep.served = props.checked
			})
		attrs() // also catches what the server said while the element was detached
		const watch = new MutationObserver(attrs)
		watch.observe(host, { attributeFilter: ['checked', 'indeterminate', 'aria-label'] })
		cleanup(() => watch.disconnect())
		overrideProp('checked', () => peek(() => $$.on), (v) => peek(() => ($$.on = !!v)))
		overrideProp('indeterminate', () => peek(() => $$.mixed), (v) => peek(() => ($$.mixed = !!v)))
		// Commands: the attribute is the server's value, $$.on the local one.
		// With confirm, :state(pending) marks an edit the server hasn't confirmed
		// yet; revert() returns to the server's word (e.g. a rejected command).
		const sync = () => peek(() => states[props.confirm && $$.on !== props.checked ? 'add' : 'delete']('pending'))
		// Rocket's disconnect wipes $$ (re-running this) before a move re-attaches.
		effect(() => ((keep.on = $$.on ?? keep.on), (keep.mixed = $$.mixed ?? keep.mixed), sync()))
		// peek: attribute changes arrive inside the effect of whoever set them.
		observeProps((p) => peek(() => (take(p), sync())))
		const revert = () => peek(() => (($$.on = props.checked), ($$.mixed = props.indeterminate), sync()))
		defineHostProp('revert', { value: revert })

		// Forms: until Rocket can make this element form-associated, join the
		// submissions and resets of the form it sits in. `formdata` also fires for
		// new FormData(form), so Datastar's contentType: 'form' posts include it.
		// Both are heard on the root (document or shadow root), once every
		// listener on the form has run: a reset the page cancelled (whenever its
		// listener was added) leaves the value, as it leaves native fields, and a
		// form nested in this one by a script, whose events bubble through it,
		// isn't taken for it. setup reruns on a re-attach, so a move follows.
		// Like a native checkbox: name=value when checked, nothing when not. A
		// reset is revert(): the server's state, no change events.
		const form = host.closest('form')
		const root = host.getRootNode()
		const onData = (evt) => evt.target === form && peek(() => props.name && !props.disabled && $$.on && evt.formData.append(props.name, props.value))
		const onReset = (evt) => evt.target === form && !evt.defaultPrevented && revert()
		root.addEventListener('formdata', onData)
		root.addEventListener('reset', onReset)
		cleanup(() => (root.removeEventListener('formdata', onData), root.removeEventListener('reset', onReset)))

		// From mixed, a click checks; otherwise it flips.
		const toggle = () => {
			if (props.disabled) return
			$$.on = $$.mixed || !$$.on
			$$.mixed = false
			emit('change')
			emit('sb-change', { name: props.name, value: $$.on, checked: $$.on })
		}
		action('toggle', toggle)
		action('key', ({ evt }) => evt.key === ' ' && (evt.preventDefault(), evt.repeat || toggle()))
	},
	render: ({ html }) => html`
		<div class="item" part="base" role="checkbox"
			data-attr:tabindex="$$disabled ? null : 0"
			data-attr:aria-checked="$$mixed ? 'mixed' : String($$on)"
			data-attr:aria-disabled="$$disabled ? 'true' : null"
			data-attr:aria-label="$$label ? null : $$aria"
			data-class:on="$$on"
			data-class:mixed="$$mixed"
			data-on:click="@toggle()"
			data-on:keydown="@key()">
			<span class="box" part="box" aria-hidden="true"></span>
			<span part="label" data-show="$$label" data-text="$$label"></span>
		</div>
	`,
})
