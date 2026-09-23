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

const DIGITS = [0, 1, 2, 3, 4, 5, 6, 7, 8, 9]
const isDigit = (c) => c >= '0' && c <= '9'

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
		// Assistive tech gets the formatted value once; each strip holds all ten
		// digits, so the strips are hidden from it.
		return html`
			<span class="sr">${text}</span><span class="odo" id="${shapeKey(text)}" aria-hidden="true">${[...text].map((ch) =>
				isDigit(ch)
					? html`<span class="col" part="digit"><span class="strip" style="transform:translateY(-${Number(ch) * 10}%)">${DIGITS.map((d) => html`<span>${d}</span>`)}</span></span>`
					: html`<span class="static" part="separator">${ch}</span>`,
			)}</span>
		`
	},
})
