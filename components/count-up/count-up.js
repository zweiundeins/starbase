import { rocket } from 'datastar'

// Counts a number up (or down) to its value when it scrolls into view.
//
// Three readers, three answers. Without JavaScript, and for search engines,
// the server's text inside the element is the number. Screen readers always
// get the final value, never the frames in between. Everyone else sees it
// count, once, the first time it is on screen, and again from wherever it
// stands whenever the server sends a new value while it is visible.
//
// Ported from libretto.ch's <count-up>, which rendered nothing at all with
// JavaScript off; here the server's text is the fallback.

const formatters = new Map()
const formatterFor = (lang, decimals, grouping) => {
	const key = `${lang}|${decimals}|${grouping}`
	let f = formatters.get(key)
	if (!f) {
		f = new Intl.NumberFormat(lang, { minimumFractionDigits: decimals, maximumFractionDigits: decimals, useGrouping: grouping })
		formatters.set(key, f)
	}
	return f
}

const easeOut = (t) => 1 - (1 - t) ** 3

const styles = /* css */ `
:host { display: inline; font-variant-numeric: tabular-nums; }
.sr { position: absolute; inline-size: 1px; block-size: 1px; overflow: hidden; clip-path: inset(50%); white-space: nowrap; }
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
		const format = (n) => formatterFor(props.lang || host.closest('[lang]')?.lang || navigator.language, props.decimals, props.grouping).format(n)

		$$.final = format(props.value)
		$$.text = host.textContent.trim() || $$.final // the server's text, until counting takes over

		let shown = props.from // the number on screen
		let raf = 0
		let visible = false
		let counted = false
		const show = (n) => ((shown = n), ($$.text = format(n)))
		const settle = () => (cancelAnimationFrame(raf), (raf = 0), show(props.value))
		const count = (a, b) => {
			cancelAnimationFrame(raf)
			if (!props.duration || a === b) return settle()
			const start = performance.now()
			const step = (now) => {
				const t = Math.min((now - start) / props.duration, 1)
				show(a + (b - a) * easeOut(t))
				raf = t < 1 ? requestAnimationFrame(step) : 0
			}
			raf = requestAnimationFrame(step)
		}

		if (matchMedia('(prefers-reduced-motion: reduce)').matches) {
			settle()
			observeProps(() => (($$.final = format(props.value)), settle()))
			return
		}

		show(props.from)
		const io = new IntersectionObserver(
			([e]) => {
				visible = e.isIntersecting
				if (visible && !counted) (counted = true), count(props.from, props.value)
			},
			{ threshold: 0.4 },
		)
		io.observe(host)
		cleanup(() => (io.disconnect(), cancelAnimationFrame(raf)))

		observeProps(() => {
			$$.final = format(props.value)
			if (!counted) return show(props.from) // not seen yet: still waiting at the start
			visible ? count(shown, props.value) : settle()
		})
	},
	render: ({ html }) => html`<span class="sr" data-text="$$final"></span><span part="value" aria-hidden="true" data-text="$$text"></span>`,
})
