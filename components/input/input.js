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

// The live state per element, for the same reason: re-attaching (e.g. a morph
// moving the element) runs setup again, and must not throw away an edit the
// server hasn't seen, or its pending and error state.
const kept = new WeakMap()

const styles = /* css */ `
:host {
	--_bg: var(--sb-control-bg, #0B1224);
	--_border: var(--sb-control-border, #283552);
	--_border-hover: var(--sb-control-border-hover, #3A4868);
	--_text: var(--sb-control-text, #F3F4FA);
	--_placeholder: var(--sb-control-placeholder, #7785A8);
	--_label: var(--sb-text-2, #AEBBDD);
	--_brand: var(--sb-brand, #8C6BFF);
	--_brand-light: var(--sb-brand-light, #B09AFF);
	--_brand-subtle: var(--sb-brand-subtle, rgb(140 107 255 / 0.14));
	--_danger: var(--sb-danger, #F2777A);
	--_on-brand: var(--sb-text-on-brand, #F3F4FA);
	--_radius: var(--sb-control-radius, 6px);
	display: block;
	inline-size: 100%;
	max-inline-size: 26rem;
}
:host([hidden]) { display: none; }
.field { display: grid; grid-template-columns: minmax(0, 1fr); gap: 0.4rem; }
label { color: var(--_label); font-size: 0.8125rem; font-weight: 600; }
.control { display: flex; gap: 0.5rem; }
input {
	all: unset;
	box-sizing: border-box;
	flex: 1;
	min-inline-size: 0;
	block-size: 2.75rem;
	padding-inline: 0.875rem;
	border: 1px solid var(--_border);
	border-radius: var(--_radius);
	background: var(--_bg);
	color: var(--_text);
	transition: border-color 120ms, box-shadow 120ms;
}
input::placeholder { color: var(--_placeholder); }
input:hover { border-color: var(--_border-hover); }
input:focus-visible { border-color: var(--_brand-light); box-shadow: 0 0 0 3px var(--_brand-subtle); }
.invalid input { border-color: var(--_danger); }
.invalid input:focus-visible { box-shadow: 0 0 0 3px color-mix(in oklch, var(--_danger) 20%, transparent); }
button {
	all: unset;
	display: grid;
	place-items: center;
	flex: none;
	inline-size: 2.75rem;
	block-size: 2.75rem;
	border: 1px solid var(--_brand-light);
	border-radius: var(--_radius);
	background: var(--_brand-subtle);
	color: var(--_text);
	cursor: pointer;
	transition: background 120ms, transform 120ms;
}
button:hover { background: var(--_brand); color: var(--_on-brand); transform: translate(1px); }
/* Right to left, the arrow and its nudge point the other way. */
:host(:dir(rtl)) button { scale: -1 1; }
button:focus-visible { box-shadow: 0 0 0 2px var(--_bg), 0 0 0 4px var(--_brand-light); }
/* The input and the button (the only focusable elements): invisible, except
   in forced colors, where the box-shadows are dropped. */
:focus-visible { outline: 2px solid transparent; outline-offset: 2px; }
svg { inline-size: 1.15rem; block-size: 1.15rem; }
.hint, .error { font-size: 0.75rem; }
.hint { color: var(--_placeholder); }
.error { color: var(--_danger); }
/* Out of the flow while empty, but never display: none: a live region
   announces reliably only when it is already there when its text changes. */
.error:empty { position: absolute; }
`

rocket('sb-input', {
	props: ({ bool, number, oneOf, string }) => ({
		value: string.docs({ description: 'The server\'s value. A new one replaces local edits (to clear, send value=""); the live value is the value property.' }),
		rev: string.docs({ description: 'Revision of the server\'s value, for commands: change it whenever the server applies a command for this field, and its value wins even when it is unchanged (e.g. an edit normalised back to it).' }),
		label: string.trim.docs({ description: 'Visible label.' }),
		placeholder: string.docs({ description: 'Placeholder text.' }),
		type: oneOf('text', 'email', 'search', 'url', 'tel', 'password').docs({ description: 'Input type.' }),
		name: string.trim.docs({ description: 'Name reported in sb-change and sb-submit (e.g. the field of a command).' }),
		required: bool.docs({ description: 'Value must not be empty.' }),
		minlength: number.min(0).docs({ description: 'Minimum length.' }),
		pattern: string.docs({ description: 'Regular expression the value must match.' }),
		hint: string.docs({ description: 'Help text below the field.' }),
		error: string.docs({ description: 'Message shown when invalid (defaults to the browser message).' }),
		action: bool.docs({ description: 'Show a submit arrow button.' }),
		confirm: bool.docs({ description: 'Server-confirmed value: :state(pending) while the local value differs from the server\'s value attribute (see revert()).' }),
	}),
	manifest: {
		events: [
			{ name: 'input', kind: 'event', bubbles: true, composed: true, description: 'On every keystroke (native, re-targeted to the host).' },
			{ name: 'change', kind: 'event', bubbles: true, composed: true, description: 'When the value is committed.' },
			{ name: 'sb-change', kind: 'custom-event', bubbles: true, composed: true, description: 'When the value is committed (Enter, or leaving a changed field). detail: { name, value }: ready for a command.' },
			{ name: 'sb-submit', kind: 'custom-event', bubbles: true, composed: true, description: 'Enter or arrow button with a valid value. detail: { name, value }.' },
		],
	},
	setup: ({ $$, action, adoptStyles, cleanup, defineHostProp, effect, emit, host, observeProps, overrideProp, props }) => {
		adoptStyles(host, styles)
		let served = host.getAttribute('value')
		let rev = host.getAttribute('rev')
		// The state kept from before a re-attach, if the server's value and rev
		// are still the ones it had seen; else a fresh one.
		const k = kept.get(host)
		;[$$.value, $$.touched, $$.invalid, $$.message] = k?.[4] === served && k[5] === rev ? k : [props.value, false, false, '']

		// Validation starts with the first commit or submit, and then follows
		// every new value.
		const validate = () => {
			$$.touched = true
			const el = host.shadowRoot?.querySelector('input')
			if (!el) return true
			const ok = el.checkValidity()
			$$.invalid = !ok
			$$.message = ok ? '' : props.error || el.validationMessage
			return ok
		}
		// Every new value goes through here, so the error always describes the
		// value shown (once validation has started).
		const set = (v) => (($$.value = v), $$.touched && validate())

		// The server's value is the attribute; $$.value the local one. A value
		// attribute the server *changes* wins; re-sent identical markup changes
		// nothing, so edits survive re-renders. A *removed* attribute is ignored:
		// morphs also remove attributes that were only reflected (e.g. from a
		// data-bind write before the upgrade). To clear, the server sends
		// value="". A new rev means the server has answered a command, so its
		// value wins even when it is the same as before. Why not observeProps:
		// it only fires when the decoded value changes, so value="" on an element
		// that had no value attribute would go unnoticed.
		const watch = new MutationObserver(() =>
			peek(() => {
				const v = host.getAttribute('value')
				const r = host.getAttribute('rev')
				if ((v !== null && v !== served) || r !== rev) set(props.value)
				served = v
				rev = r
			}),
		)
		watch.observe(host, { attributeFilter: ['value', 'rev'] })
		// Detached: what the kept state had seen of the server. A change not yet
		// observed, or made while detached, then starts afresh on re-attach.
		cleanup(() => (watch.disconnect(), kept.get(host)?.push(served, rev)))
		overrideProp('value', () => peek(() => $$.value), (v) => peek(() => set(String(v ?? ''))))

		// Commands: with confirm, :state(pending) marks an edit the server hasn't
		// confirmed yet; revert() returns to the server's value (e.g. a rejected
		// command).
		const states = internalsOf(host).states
		const sync = () => peek(() => (props.confirm && $$.value !== props.value ? states.add('pending') : states.delete('pending')))
		// (Rocket clears the signals on disconnect, which runs this once more
		// with no value: that is not a state to keep.)
		effect(() => ($$.value != null && kept.set(host, [$$.value, $$.touched, $$.invalid, $$.message]), sync()))
		observeProps(sync)
		defineHostProp('revert', { value: () => peek(() => set(props.value)) })

		action('input', ({ el }) => set(el.value))
		action('commit', () => {
			validate()
			emit('change')
			emit('sb-change', { name: props.name, value: $$.value })
		})
		// Enter or the arrow. Enter also commits: the native change event
		// follows. keyCode 13, not key: the Enter that picks a word in an input
		// method is 229 (and isComposing, where the browser says so).
		action('submit', () => {
			if (validate()) emit('sb-submit', { name: props.name, value: $$.value })
		})
	},
	// The label names the input, the hint (or the error) describes it. The
	// error is a polite live region: it can change on every keystroke.
	render: ({ html, host, props: { label, placeholder, type, required, minlength, pattern, hint, action } }) => html`
		<div class="field">
			${label ? html`<label part="label" for="i">${label}</label>` : null}
			<span class="control" data-class:invalid="$$invalid">
				<input
					id="i"
					part="input"
					type="${type}"
					placeholder="${placeholder || null}"
					aria-label="${label ? null : host.getAttribute('aria-label')}"
					aria-describedby="e h"
					required="${required}"
					minlength="${minlength || null}"
					pattern="${pattern || null}"
					data-attr:aria-invalid="String($$invalid)"
					data-effect="el.value !== $$value && (el.value = $$value)"
					data-on:input="@input()"
					data-on:change="@commit()"
					data-on:keydown="evt.keyCode == 13 && !evt.isComposing && @submit()"
				/>
				${action ? html`<button type="button" part="button" aria-label="Submit" data-on:click="@submit()"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M5 12h14"/><path d="m12 5 7 7-7 7"/></svg></button>` : null}
			</span>
			<span class="error" id="e" aria-live="polite" data-text="$$message"></span>
			${hint ? html`<span class="hint" id="h" data-show="!$$invalid">${hint}</span>` : null}
		</div>
	`,
})
