import { rocket } from 'datastar'

rocket('sb-signal-lamp', {
	props: ({ bool, oneOf }) => ({
		state: oneOf('off', 'on', 'blink').default('on').docs({ description: 'Lamp state.' }),
		large: bool.docs({ description: 'Double size.' }),
	}),
	setup: ({ adoptStyles, host }) => {
		adoptStyles(host, `
			:host { --_on: var(--sb-success, #4ADE80); --_off: var(--sb-border, #283552); display: inline-block; }
			i { display: block; inline-size: 12px; block-size: 12px; background: var(--_off); }
			:host([large]) i { inline-size: 24px; block-size: 24px; }
			.on, .blink { background: var(--_on); box-shadow: 0 0 8px var(--_on); }
			.blink { animation: b 1s steps(2) infinite; }
			@keyframes b { 50% { opacity: 0.2 } }
			@media (prefers-reduced-motion: reduce) { .blink { animation: none } }
		`)
	},
	render: ({ html, props: { state } }) => html`<i part="lamp" class="${state}" role="img" aria-label="${state}"></i>`,
})
