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
// the next change, a wheel sent into an outer turn moves a whole turn back
// towards the middle one, which looks identical, so it always has room to roll
// either way.
const CELLS = Array.from({ length: 30 }, (_, i) => i % 10)
const REST = 10 // the middle turn
const isDigit = (c) => c >= '0' && c <= '9'
const offset = (pos) => `translateY(${(pos * -100) / CELLS.length}%)`

// With `drum`, each wheel is a cylinder instead: its ten digits sit on faces
// 36° apart, and rolling turns the cylinder, so a digit foreshortens as it
// turns toward the top or bottom like print on a real wheel. The angle only
// ever accumulates (+36° a digit going up, −36° going down), which makes
// turning over the top natural and needs none of the strip's outer turns.
const DIGITS = [0, 1, 2, 3, 4, 5, 6, 7, 8, 9]
const STEP = 36 // degrees per digit
const mod10 = (n) => ((n % 10) + 10) % 10

// Per element: the last value, and each wheel's strip position and drum turn.
// A roll only reads them for the same shape, so the same wheels.
const wheels = new WeakMap()

// The shape key: the same length and the same digit/static layout means a roll
// is valid. The statics (signs, separators, the decimal mark) stay as they are:
// the morph only matches ids, never uses them in a selector.
const shapeKey = (s) => 'odo' + s.replace(/\d/g, 'd')

// A tag Intl would throw on falls back to the default locale (en_US reads as en-US).
const locale = (tag) => { try { return Intl.getCanonicalLocales(tag?.replace(/_/g, '-') || [])[0] } catch {} }

const styles = /* css */ `
:host {
	--_duration: var(--sb-odometer-duration, 0.55s);
	--_easing: var(--sb-odometer-easing, cubic-bezier(0.2, 0.85, 0.25, 1));
	display: inline-flex;
	align-items: flex-end;
	line-height: 1;
	font-variant-numeric: tabular-nums;
	direction: ltr; /* a number reads left to right in right-to-left text too */
}
:host([hidden]) { display: none; }
/* Every wheel holds all its digits: a copy takes the .sr text instead. */
.odo { display: contents; -webkit-user-select: none; user-select: none; }
.sr { position: absolute; inline-size: 1px; block-size: 1px; overflow: hidden; clip-path: inset(50%); white-space: nowrap; }
.col { display: inline-block; block-size: 1em; overflow: hidden; }
.strip { display: block; will-change: transform; }
.strip > span { display: block; block-size: 1em; line-height: 1em; text-align: center; }
.static { display: inline-block; }

/* Drum (.d, from the decoded prop: drum="false" is no drum): the window shows
   the digit face-on and the neighbours curving away above and below it, fading
   into the page. A face is 1em tall, so ten of them close into a cylinder of
   radius 0.5em / tan(18°). */
.d > * {
	--_window: var(--sb-odometer-window, 1.5em);
	--_perspective: var(--sb-odometer-perspective, 5em);
	--_r: 1.5388em;
	align-self: center;
}
.drum {
	position: relative;
	inline-size: 1ch;
	block-size: var(--_window);
	overflow: hidden;
	perspective: var(--_perspective);
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
	display: grid;
	place-items: center;
	backface-visibility: hidden;
}
/* A digit sits high in its line box – room for ascenders above it, descenders
   below – so on a drum it would ride above its face's centre: the neighbour
   above slides out of the window and the one below into it. Trimming the box
   to the cap height centres every digit on its face. The separators get the
   same trim, so the decimal mark keeps the digits' baseline. Where text-box is
   not supported the drum simply looks as it did without it. */
.face > span,
.d > .static > span {
	display: block;
	text-box: trim-both cap alphabetic;
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
		// The nearest lang, also outside the shadow roots it sits in, then the browser's.
		let tag = lang
		for (let el = host; !tag && el; el = el.getRootNode().host) tag = el.closest('[lang]')?.lang
		// Latin digits always: a locale's own numerals would not be 0–9 strips. A
		// new formatter per render costs a few µs, next to milliseconds of morph.
		const text = new Intl.NumberFormat(locale(tag || navigator.language), { minimumFractionDigits: decimals, maximumFractionDigits: decimals, useGrouping: grouping, numberingSystem: 'latn' }).format(value)
		// The mode is part of the shape: switching it builds fresh wheels.
		const shape = shapeKey(text) + (drum ? '-drum' : '')
		const digits = [...text].filter(isDigit).map(Number)

		const last = wheels.get(host)
		const same = last?.shape === shape
		// The wheels show the magnitude, and the same shape has the same sign:
		// below zero, they turn back as the number climbs.
		const up = same && Math.abs(value) > Math.abs(last.value)
		const down = same && Math.abs(value) < Math.abs(last.value)
		if (same) {
			// Settle wheels sent into an outer turn a whole turn back, from where
			// they are now: the same digit, so nothing visibly moves and a roll
			// still in flight carries on, but the roll that follows counts from
			// the middle turn and can turn over in either direction. A wheel more
			// than a turn behind (updates far faster than the roll) is caught up
			// to the end of its strip rather than shown past it.
			host.shadowRoot?.querySelectorAll('.strip').forEach((strip, i) => {
				const shift = last.pos[i] - REST - (last.pos[i] % 10)
				if (!shift) return
				// Where it is now, from matrix(1, 0, 0, 1, 0, y) in px. Not rendered, the
				// transform is none (or not a matrix): NaN, and the write is ignored.
				const at = (parseFloat(getComputedStyle(strip).transform.split(',')[5]) * -CELLS.length) / strip.offsetHeight
				strip.style.transition = 'none'
				strip.style.transform = offset(Math.min(Math.max(at - shift, 0), 29))
				void strip.offsetHeight
				strip.style.transition = ''
			})
		}
		const pos = digits.map((d, i) => {
			const from = same ? last.pos[i] % 10 : d
			if (up && d < from) return REST + 10 + d // 9 → 0 going up: roll on over the top
			if (down && d > from) return d // 0 → 9 going down: roll back under
			return REST + d
		})
		// Drum turns: from where each wheel stands, forward to the new digit when
		// the number climbs and back when it falls, however many digits away.
		// Neither means the same value, so the same digits.
		const turn = digits.map((d, i) => {
			const t = same ? last.turn[i] : d
			return up ? t + mod10(d - t) : down ? t - mod10(t - d) : t
		})
		wheels.set(host, { value, shape, pos, turn })

		// Assistive tech gets the formatted value once; each strip holds all its
		// digits, so the strips are hidden from it.
		let n = 0
		const wheel = () => {
			const i = n++
			return drum
				? html`<span class="drum" part="digit"><span class="cyl" style="transform:translateZ(calc(-1 * var(--_r))) rotateX(${turn[i] * STEP}deg)">${DIGITS.map((d) => html`<span class="face" style="transform:rotateX(${-d * STEP}deg) translateZ(var(--_r))"><span>${d}</span></span>`)}</span></span>`
				: html`<span class="col" part="digit"><span class="strip" style="transform:${offset(pos[i])}">${CELLS.map((d) => html`<span>${d}</span>`)}</span></span>`
		}
		return html`
			<span class="sr">${text}</span><span class="odo${drum ? ' d' : ''}" id="${shape}" aria-hidden="true">${[...text].map((ch) =>
				isDigit(ch) ? wheel() : html`<span class="static" part="separator"><span>${ch}</span></span>`,
			)}</span>
		`
	},
})
