import { rocket } from 'datastar'

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
	--_radius: var(--sb-control-radius, 6px);
	display: block;
	inline-size: 100%;
	max-inline-size: 26rem;
}
.field { display: grid; grid-template-columns: minmax(0, 1fr); gap: 0.4rem; }
.label { color: var(--_label); font-size: 0.8125rem; font-weight: 600; }
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
	font: inherit;
	transition: border-color 120ms, box-shadow 120ms;
}
input::placeholder { color: var(--_placeholder); }
input:hover { border-color: var(--_border-hover); }
input:focus-visible { border-color: var(--_brand-light); box-shadow: 0 0 0 3px var(--_brand-subtle); }
.invalid input { border-color: var(--_danger); }
.invalid input:focus-visible { box-shadow: 0 0 0 3px color-mix(in oklch, var(--_danger) 20%, transparent); }
.go {
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
	transition: background 120ms, translate 120ms;
}
.go:hover { background: var(--_brand); translate: 1px 0; }
.go:focus-visible { box-shadow: 0 0 0 2px var(--_bg), 0 0 0 4px var(--_brand-light); }
.go svg { inline-size: 1.15rem; block-size: 1.15rem; }
.hint, .error { font-size: 0.75rem; }
.hint { color: var(--_placeholder); }
.error { color: var(--_danger); }
`

rocket('sb-input', {
	props: ({ bool, number, oneOf, string }) => ({
		value: string.docs({ description: 'Initial value. Read the live value from the value property.' }),
		label: string.trim.docs({ description: 'Visible label.' }),
		placeholder: string.docs({ description: 'Placeholder text.' }),
		type: oneOf('text', 'email', 'search', 'url', 'tel', 'password').default('text').docs({ description: 'Input type.' }),
		name: string.trim.docs({ description: 'Name reported in the sb-submit event.' }),
		required: bool.docs({ description: 'Value must not be empty.' }),
		minlength: number.min(0).docs({ description: 'Minimum length.' }),
		pattern: string.docs({ description: 'Regular expression the value must match.' }),
		hint: string.docs({ description: 'Help text below the field.' }),
		error: string.docs({ description: 'Message shown when invalid (defaults to the browser message).' }),
		action: bool.docs({ description: 'Show a submit arrow button.' }),
	}),
	manifest: {
		events: [
			{ name: 'input', kind: 'event', bubbles: true, composed: true, description: 'On every keystroke (native, re-targeted to the host).' },
			{ name: 'change', kind: 'event', bubbles: true, composed: true, description: 'When the value is committed.' },
			{ name: 'sb-submit', kind: 'custom-event', bubbles: true, composed: true, description: 'Enter or arrow button with a valid value. detail: { name, value }.' },
		],
	},
	setup: ({ $$, action, adoptStyles, emit, host, observeProps, overrideProp, props }) => {
		adoptStyles(host, styles)
		$$.value = props.value
		$$.touched = false
		$$.invalid = false
		$$.message = ''
		observeProps(() => ($$.value = props.value), 'value')
		overrideProp('value', () => $$.value, (v) => ($$.value = String(v ?? '')))

		const field = () => host.shadowRoot?.querySelector('input')
		const validate = () => {
			const el = field()
			if (!el) return true
			const ok = el.checkValidity()
			$$.invalid = !ok
			$$.message = ok ? '' : props.error || el.validationMessage
			return ok
		}
		const submit = () => {
			$$.touched = true
			if (validate()) emit('sb-submit', { name: props.name, value: $$.value })
		}
		action('input', ({ el }) => {
			$$.value = el.value
			if ($$.touched) validate()
		})
		action('commit', () => {
			$$.touched = true
			validate()
			emit('change')
		})
		action('key', ({ evt }) => {
			if (evt.key === 'Enter') {
				evt.preventDefault()
				submit()
			}
		})
		action('submit', submit)
	},
	render: ({ html, props: { label, placeholder, type, required, minlength, pattern, hint, action } }) => html`
		<label class="field">
			${label ? html`<span class="label" part="label">${label}</span>` : null}
			<span class="control" data-class:invalid="$$invalid">
				<input
					part="input"
					type="${type}"
					placeholder="${placeholder || null}"
					aria-label="${label ? null : placeholder || 'Text'}"
					required="${required}"
					minlength="${minlength || null}"
					pattern="${pattern || null}"
					data-attr:aria-invalid="String($$invalid)"
					data-effect="el.value !== $$value && (el.value = $$value)"
					data-on:input="@input()"
					data-on:change="@commit()"
					data-on:keydown="@key()"
				/>
				${action ? html`<button class="go" type="button" part="button" aria-label="Submit" data-on:click="@submit()"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M5 12h14"/><path d="m12 5 7 7-7 7"/></svg></button>` : null}
			</span>
			<span class="error" role="alert" data-show="$$invalid" data-text="$$message"></span>
			${hint ? html`<span class="hint" data-show="!$$invalid">${hint}</span>` : null}
		</label>
	`,
})
