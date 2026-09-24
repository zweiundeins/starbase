import { rocket } from 'datastar'

// Counts a number up (or down) to its value when it scrolls into view.
//
// Three readers, three answers. Before the module loads, without JavaScript,
// and for search engines, the server's text inside the element is the number.
// Screen readers always get the final value, never the frames in between.
// Everyone else sees it count, once, the first time it is on screen (moving
// the element doesn't count it again), and again from wherever it stands
// whenever the server sends a new value while it is visible.
//
// Ported from libretto.ch's <count-up>, which rendered nothing at all with
// JavaScript off; here the server's text is the fallback.

// An invalid tag (en_US, a typo) would make Intl throw and render nothing.
const locale = (tag) => { try { return Intl.getCanonicalLocales(tag?.replace(/_/g, '-') || [])[0] } catch {} }

// Hosts that have counted: setup reruns on every connect.
const seen = new WeakSet()

// Both copies share one grid cell: the final value (invisible, for screen
// readers) holds the width, so the line doesn't move while the digits grow.
// Print has no scrolling to start a count: it shows the final value.
const styles = /* css */ `
:host { display: inline-grid; justify-items: end; font-variant-numeric: tabular-nums; }
:host([hidden]) { display: none; }
span { grid-area: 1/1; }
.sr { opacity: 0; }
@media print { .sr { opacity: 1; } [part] { visibility: hidden; } }
`

rocket('sb-count-up', {
	props: ({ bool, number, string }) => ({
		value: number.docs({ description: 'The number to arrive at.' }),
		from: number.docs({ description: 'Where the count starts the first time it is seen.' }),
		decimals: number.round.clamp(0, 7).docs({ description: 'Fixed digits after the decimal mark, kept for every frame so the width stays put.' }),
		duration: number.min(0).default(1400).docs({ description: 'Length of the count in milliseconds.' }),
		grouping: bool.default(true).docs({ description: 'Group thousands the way the locale does.' }),
		lang: string.trim.docs({ description: "Locale for the separators (default: the page's lang, then the browser's)." }),
	}),
	renderOnPropChange: false,
	setup: ({ $$, adoptStyles, cleanup, host, observeProps, props }) => {
		adoptStyles(host, styles)
		const still = matchMedia('(prefers-reduced-motion: reduce)').matches
		let nf
		const update = () => {
			// closest() stops at a shadow root: inside another component, the page's lang.
			nf = new Intl.NumberFormat(locale(props.lang || host.closest('[lang]')?.lang || document.documentElement.lang), {
				minimumFractionDigits: props.decimals,
				maximumFractionDigits: props.decimals,
				useGrouping: props.grouping,
			})
			$$.final = nf.format(props.value)
		}
		update()

		let counted = still || seen.has(host) // reduced motion: the value, straight away
		let shown // the number on screen
		let raf = 0
		let visible = false
		const show = (n) => ((shown = n), ($$.text = nf.format(n)))
		const settle = () => (cancelAnimationFrame(raf), (raf = 0), show(props.value))
		const count = (a, b) => {
			cancelAnimationFrame(raf)
			if (!props.duration || a === b) return settle()
			let start
			const step = (now) => {
				// The frame's own clock: a frame can start before the change arrived.
				const t = Math.min((now - (start ??= now)) / props.duration, 1)
				show(a + (b - a) * (1 - (1 - t) ** 3))
				raf = t < 1 ? requestAnimationFrame(step) : 0
			}
			raf = requestAnimationFrame(step)
		}

		show(counted ? props.value : props.from)
		const io = new IntersectionObserver(
			([e]) => {
				visible = e.isIntersecting
				if (visible && !counted) seen.add(host), (counted = true), count(props.from, props.value)
			},
			{ threshold: 0.4 },
		)
		io.observe(host)
		cleanup(() => (io.disconnect(), cancelAnimationFrame(raf)))

		observeProps(() => {
			update()
			if (!counted) return show(props.from) // not seen yet: still waiting at the start
			visible && !still ? count(shown, props.value) : settle()
		})
	},
	render: ({ html }) => html`<span class="sr" data-text="$$final"></span><span part="value" aria-hidden="true" data-text="$$text"></span>`,
})
