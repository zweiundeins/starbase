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

// With `drum`, each wheel is a cylinder instead: its ten digits sit on faces
// 36° apart, and rolling turns the cylinder, so a digit foreshortens as it
// turns toward the top or bottom like print on a real wheel. The angle only
// ever accumulates (+36° a digit going up, −36° going down), which makes
// turning over the top natural and needs none of the strip's outer turns.
const DIGITS = [0, 1, 2, 3, 4, 5, 6, 7, 8, 9]
const STEP = 36 // degrees per digit
const mod10 = (n) => ((n % 10) + 10) % 10

// Per element: the last value, and each wheel's strip position and drum turn,
// counted from the right so the units wheel stays the units wheel when a
// digit is gained.
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

/* Drum: the window shows the digit face-on and the neighbours curving away
   above and below it, fading into the page. A face is 1em tall, so ten of them
   close into a cylinder of radius 0.5em / tan(18°). */
:host([drum]) {
	--_window: var(--sb-odometer-window, 1.5em);
	--_perspective: var(--sb-odometer-perspective, 5em);
	--_r: 1.5388em;
	align-items: center;
}
.drum {
	position: relative;
	inline-size: 1ch;
	block-size: var(--_window);
	overflow: hidden;
	perspective: var(--_perspective);
	-webkit-mask-image: linear-gradient(to bottom, transparent, #000 30%, #000 70%, transparent);
	mask-image: linear-gradient(to bottom, transparent, #000 30%, #000 70%, transparent);
}
.cyl { position: absolute; inset: 0; transform-style: preserve-3d; will-change: transform; }
.face {
	position: absolute;
	inset-inline: 0;
	inset-block-start: 50%;
	block-size: 1em;
	margin-block-start: -0.5em;
	line-height: 1em;
	text-align: center;
	backface-visibility: hidden;
}
@media (prefers-reduced-motion: no-preference) {
	.strip, .cyl { transition: transform var(--_duration) var(--_easing); }
}
`

rocket('sb-odometer', {
	props: ({ bool, number, string }) => ({
		value: number.docs({ description: 'The number to show. Change it and the digits roll to the new value.' }),
		decimals: number.round.clamp(0, 7).docs({ description: 'Fixed digits after the decimal mark, so a whole-number step still shows ".0" and the layout does not jump.' }),
		grouping: bool.default(true).docs({ description: 'Group thousands the way the locale does (221’180, 221,180, 221 180).' }),
		lang: string.trim.docs({ description: "Locale for the separators (default: the page's lang, then the browser's)." }),
		drum: bool.docs({ description: 'Draw each wheel as a 3D drum: digits curve away above and below, and foreshorten as they roll.' }),
	}),
	setup: ({ adoptStyles, host }) => {
		adoptStyles(host, styles)
	},
	render: ({ html, host, props: { value, decimals, grouping, lang, drum } }) => {
		const locale = lang || host.closest('[lang]')?.lang || navigator.language
		const text = formatterFor(locale, decimals, grouping).format(value)
		// The mode is part of the shape: switching it builds fresh wheels.
		const shape = shapeKey(text) + (drum ? '-drum' : '')
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
		// Drum turns: from where each wheel stands, forward to the new digit when
		// the number climbs and back when it falls, however many digits away.
		const turn = digits.map((d, i) => {
			const k = digits.length - 1 - i
			const t = same ? last.turn[k] : undefined
			if (t === undefined) return d
			if (up) return t + mod10(d - mod10(t))
			if (down) return t - mod10(mod10(t) - d)
			return t + (mod10(d - mod10(t)) <= 5 ? mod10(d - mod10(t)) : mod10(d - mod10(t)) - 10)
		})
		const byWheel = (list) => Object.fromEntries(list.map((v, i) => [digits.length - 1 - i, v]))
		wheels.set(host, { value, shape, pos: byWheel(pos), turn: byWheel(turn) })

		// Assistive tech gets the formatted value once; each strip holds all its
		// digits, so the strips are hidden from it.
		let n = 0
		const wheel = () => {
			const i = n++
			return drum
				? html`<span class="drum" part="digit"><span class="cyl" style="transform:translateZ(calc(-1 * var(--_r))) rotateX(${turn[i] * STEP}deg)">${DIGITS.map((d) => html`<span class="face" style="transform:rotateX(${-d * STEP}deg) translateZ(var(--_r))">${d}</span>`)}</span></span>`
				: html`<span class="col" part="digit"><span class="strip" style="transform:${offset(pos[i])}">${CELLS.map((d) => html`<span>${d}</span>`)}</span></span>`
		}
		return html`
			<span class="sr">${text}</span><span class="odo" id="${shape}" aria-hidden="true">${[...text].map((ch) =>
				isDigit(ch) ? wheel() : html`<span class="static" part="separator">${ch}</span>`,
			)}</span>
		`
	},
})
