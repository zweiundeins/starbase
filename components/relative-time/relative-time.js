import { rocket } from 'datastar'

// One ticker for every instance on the page: each registers when its text
// next changes, and only those are recomputed.
const watchers = new Set()
let ticker = 0
const tick = () => {
	const now = Date.now()
	for (const w of watchers) if (w.next <= now) w.update(now)
}
const watch = (w) => {
	watchers.add(w)
	ticker ||= setInterval(tick, 1000)
	return () => {
		watchers.delete(w)
		if (!watchers.size) (clearInterval(ticker), (ticker = 0))
	}
}

const SECOND = 1000, MINUTE = 60 * SECOND, HOUR = 60 * MINUTE, DAY = 24 * HOUR, WEEK = 7 * DAY, MONTH = 30 * DAY, YEAR = 365 * DAY
const UNITS = [
	['year', YEAR],
	['month', MONTH],
	['week', WEEK],
	['day', DAY],
	['hour', HOUR],
	['minute', MINUTE],
	['second', SECOND],
]

// parse accepts ISO 8601 and Unix time (seconds or milliseconds).
const parse = (v) => {
	const s = String(v ?? '').trim()
	if (!s) return NaN
	if (/^-?\d+(\.\d+)?$/.test(s)) {
		const n = Number(s)
		return Math.abs(n) < 1e11 ? n * 1000 : n
	}
	return Date.parse(s)
}

// relative picks the largest unit that fits, and the moment the text next
// changes: when the rounded count moves (abs crosses k + ½ units), or the
// unit does. For the past, abs grows with time; for the future, it shrinks.
const relative = (then, now, fmt) => {
	const diff = then - now
	const abs = Math.abs(diff)
	const past = diff <= 0
	let i = abs < SECOND ? UNITS.length - 1 : Math.max(0, UNITS.findIndex(([, ms]) => abs >= ms))
	// "60 minutes ago" reads better as "1 hour ago".
	if (i > 0 && Math.abs(Math.round(diff / UNITS[i][1])) * UNITS[i][1] >= UNITS[i - 1][1]) i--
	const [unit, ms] = UNITS[i]
	const n = Math.round(diff / ms)
	const x = abs / ms
	const edges = [Math.abs((past ? Math.floor(x - 0.5) + 1.5 : Math.ceil(x - 0.5) - 0.5) * ms - abs)]
	if (past && i > 0) edges.push(UNITS[i - 1][1] - abs) // grows into the next unit
	if (!past && abs >= ms) edges.push(abs - ms + 1) // shrinks below this unit
	const wait = Math.max(SECOND, Math.min(...edges.filter((e) => e > 0)))
	return { text: fmt.format(n, unit), next: now + wait }
}

const styles = /* css */ `
:host { display: inline; }
time { font: inherit; color: inherit; }
`

rocket('sb-relative-time', {
	props: ({ bool, number, oneOf, string }) => ({
		datetime: string.trim.docs({ description: 'The moment: ISO 8601 (2026-09-22T08:00:00Z) or Unix time (seconds or ms).' }),
		numeric: oneOf('auto', 'always').default('auto').docs({ description: '"auto" allows words like "yesterday"; "always" says "1 day ago".' }),
		format: oneOf('long', 'short', 'narrow').default('long').docs({ description: 'Length of the units ("3 minutes" / "3 min." / "3m").' }),
		lang: string.trim.docs({ description: 'Locale (default: the page\'s lang, then the browser\'s).' }),
		threshold: number.min(0).docs({ description: 'After this many days, show the date instead (0: always relative).' }),
		sync: bool.default(true).docs({ description: 'Keep the text current while the page is open.' }),
	}),
	renderOnPropChange: false,
	setup: ({ $$, adoptStyles, cleanup, host, observeProps, props }) => {
		adoptStyles(host, styles)
		$$.text = host.textContent.trim() // the server's fallback, until computed
		$$.iso = ''
		$$.title = ''
		const w = { next: Infinity, update: () => {} }
		w.update = (now = Date.now()) => {
			const then = parse(props.datetime)
			if (Number.isNaN(then)) {
				$$.text = host.textContent.trim()
				$$.iso = $$.title = ''
				w.next = Infinity
				return
			}
			const lang = props.lang || host.closest('[lang]')?.lang || navigator.language
			const date = new Date(then)
			$$.iso = date.toISOString()
			$$.title = new Intl.DateTimeFormat(lang, { dateStyle: 'full', timeStyle: 'short' }).format(date)
			if (props.threshold > 0 && Math.abs(then - now) > props.threshold * DAY) {
				$$.text = new Intl.DateTimeFormat(lang, { dateStyle: props.format === 'long' ? 'long' : 'medium' }).format(date)
				w.next = now + HOUR // re-check now and then (the page may stay open for days)
				return
			}
			const r = relative(then, now, new Intl.RelativeTimeFormat(lang, { numeric: props.numeric, style: props.format }))
			$$.text = r.text
			w.next = props.sync ? r.next : Infinity
		}
		w.update()
		observeProps(() => w.update())
		cleanup(watch(w))
	},
	render: ({ html }) => html`<time part="time" data-attr:datetime="$$iso || null" data-attr:title="$$title || null" data-text="$$text"></time>`,
})
