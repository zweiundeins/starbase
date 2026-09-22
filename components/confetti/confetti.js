import { rocket } from 'datastar'
import confetti from './vendor/confetti.js'

rocket('sb-confetti', {
	props: ({ string }) => ({
		label: string.trim.default('Celebrate').docs({ description: 'Button text.' }),
	}),
	setup: ({ action, adoptStyles, host }) => {
		adoptStyles(host, `
			:host { display: inline-block; }
			button { all: unset; padding: 0.5rem 1rem; border: 1px solid var(--sb-border, #283552); border-radius: var(--sb-radius, 8px); background: var(--sb-surface-card, #141D32); color: var(--sb-text-1, #F3F4FA); cursor: pointer; }
			button:hover { border-color: var(--sb-brand, #8C6BFF); }
		`)
		action('fire', ({ el }) => {
			const r = el.getBoundingClientRect()
			confetti({ disableForReducedMotion: true, origin: { x: (r.left + r.width / 2) / innerWidth, y: (r.top + r.height / 2) / innerHeight } })
		})
	},
	render: ({ html, props: { label } }) => html`<button type="button" part="button" data-on:click="@fire()">${label}</button>`,
})
