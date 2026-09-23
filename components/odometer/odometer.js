import { rocket } from 'datastar'

// A mechanical odometer: each digit is a stacked 0–9 strip, positioned by a
// transform, so a change rolls the wheels instead of swapping the text.
//
// The roll is declarative. Rocket re-renders by morphing, so a changed digit
// keeps its strip element and only its style changes, which the CSS transition
// animates. A change of *shape* (a digit gained or lost, 9.9 → 10.0, or a new
// group separator) must snap instead of rolling from the old layout, so the
// wrapper is keyed by the shape: a new shape is a new id, which the morph
// replaces with fresh nodes rather than patching.
//
// Ported from Libretto's <odo-meter>, where it has run the live-drive distance
// readout since 2026-09.

// Each wheel is three turns of 0–9 stacked, resting on the middle one. A wheel
// that passes 9 → 0 while the number climbs rolls on into the turn below and
// one passing 0 → 9 while it falls rolls back into the turn above, the way a
// real odometer turns over, instead of spinning back through every digit. Before
// the next change, a wheel left in an outer turn moves to the same digit in the
// middle turn, which looks identical, so it always has room to roll either way.
const CELLS = Array.from({ length: 30 }, (_, i) => i % 10)
const REST = 10 // the middle turn
const isDigit = (c) => c >= '0' && c <= '9'
const offset = (pos) => `translateY(-${(pos * 100) / CELLS.length}%)`

// Per element: the last value, and each wheel's position, counted from the
// right so the units wheel stays the units wheel when a digit is gained.
const wheels = new WeakMap()

// The shape key: the same length and the same digit/static layout means a roll
// is valid. Statics (signs, separators, the decimal mark) are encoded by char
// code, so the key stays a plain id token.
const shapeKey = (s) => 'odo-' + [...s].map((c) => (isDigit(c) ? 'd' : c.charCodeAt(0).toString(36))).join('-')

// Formatters are cached per option set: Intl.NumberFormat is not cheap, and a
// readout can change many times a second.
const formatters = new Map()
const formatterFor = (lang, decimals, grouping) => {
	const key = `${lang}|${decimals}|${grouping}`
	let f = formatters.get(key)
	if (!f) {
		// Latin digits always: a locale's own numerals would not be 0–9 strips.
		f = new Intl.NumberFormat(lang, { minimumFractionDigits: decimals, maximumFractionDigits: decimals, useGrouping: grouping, numberingSystem: 'latn' })
		formatters.set(key, f)
	}
	return f
}

const styles = /* css */ `
:host {
	--_duration: var(--sb-odometer-duration, 0.55s);
	--_easing: var(--sb-odometer-easing, cubic-bezier(0.2, 0.85, 0.25, 1));
	display: inline-flex;
	align-items: flex-end;
	line-height: 1;
	font-variant-numeric: tabular-nums;
}
.odo { display: contents; }
.sr { position: absolute; inline-size: 1px; block-size: 1px; overflow: hidden; clip-path: inset(50%); white-space: nowrap; }
.col { display: inline-block; block-size: 1em; overflow: hidden; }
.strip { display: block; will-change: transform; }
.strip > span { display: block; block-size: 1em; line-height: 1em; text-align: center; }
.static { display: inline-block; }
@media (prefers-reduced-motion: no-preference) {
	.strip { transition: transform var(--_duration) var(--_easing); }
}
`

rocket('sb-odometer', {
	props: ({ bool, number, string }) => ({
		value: number.docs({ description: 'The number to show. Change it and the digits roll to the new value.' }),
		decimals: number.round.clamp(0, 7).docs({ description: 'Fixed digits after the decimal mark, so a whole-number step still shows ".0" and the layout does not jump.' }),
		grouping: bool.default(true).docs({ description: 'Group thousands the way the locale does (221’180, 221,180, 221 180).' }),
		lang: string.trim.docs({ description: "Locale for the separators (default: the page's lang, then the browser's)." }),
	}),
	setup: ({ adoptStyles, host }) => {
		adoptStyles(host, styles)
	},
	render: ({ html, host, props: { value, decimals, grouping, lang } }) => {
		const locale = lang || host.closest('[lang]')?.lang || navigator.language
		const text = formatterFor(locale, decimals, grouping).format(value)
		const shape = shapeKey(text)
		const digits = [...text].filter(isDigit).map(Number)

		const last = wheels.get(host)
		const same = last?.shape === shape
		const up = same && value > last.value
		const down = same && value < last.value
		if (same) {
			// Settle wheels still standing in an outer turn onto the middle one:
			// the same digit, so nothing visibly moves, but the roll that follows
			// starts from the middle and can turn over in either direction.
			host.shadowRoot?.querySelectorAll('.strip').forEach((strip, i) => {
				const k = digits.length - 1 - i
				const pos = last.pos[k]
				if (pos === undefined || (pos >= REST && pos < REST + 10)) return
				strip.style.transition = 'none'
				strip.style.transform = offset(REST + (pos % 10))
				void strip.offsetHeight
				strip.style.transition = ''
				last.pos[k] = REST + (pos % 10)
			})
		}
		const pos = digits.map((d, i) => {
			const k = digits.length - 1 - i
			const from = same ? last.pos[k] % 10 : d
			if (up && d < from) return REST + 10 + d // 9 → 0 going up: roll on over the top
			if (down && d > from) return d // 0 → 9 going down: roll back under
			return REST + d
		})
		wheels.set(host, { value, shape, pos: Object.fromEntries(pos.map((p, i) => [digits.length - 1 - i, p])) })

		// Assistive tech gets the formatted value once; each strip holds all its
		// digits, so the strips are hidden from it.
		let n = 0
		return html`
			<span class="sr">${text}</span><span class="odo" id="${shape}" aria-hidden="true">${[...text].map((ch) =>
				isDigit(ch)
					? html`<span class="col" part="digit"><span class="strip" style="transform:${offset(pos[n++])}">${CELLS.map((d) => html`<span>${d}</span>`)}</span></span>`
					: html`<span class="static" part="separator">${ch}</span>`,
			)}</span>
		`
	},
})
