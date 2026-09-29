import { rocket, startPeeking, stopPeeking } from 'datastar'

// Host getters must not subscribe callers (e.g. data-bind's sync effect), and
// attribute changes arrive inside the effect of whoever set them.
const peek = (fn) => {
	startPeeking()
	try {
		return fn()
	} finally {
		stopPeeking()
	}
}

// One ElementInternals per element: attachInternals() works once, and
// onFirstRender runs again when the element is re-attached. Its custom states
// (:state(pending)) are styleable from the page and morph-proof.
const internals = new WeakMap()
const internalsOf = (host) => internals.get(host) ?? internals.set(host, host.attachInternals()).get(host)

// Pixel corners: notches every corner by p (2px times --sb-notch; at 0 the
// border-radius takes over).
const notch = (p) => `polygon(${p} 0, calc(100% - ${p}) 0, calc(100% - ${p}) ${p}, 100% ${p}, 100% calc(100% - ${p}), calc(100% - ${p}) calc(100% - ${p}), calc(100% - ${p}) 100%, ${p} 100%, ${p} calc(100% - ${p}), 0 calc(100% - ${p}), 0 ${p}, ${p} ${p})`

// A malformed tag (en_US) would make Intl throw: repair it, or use the default.
const locale = (tag) => { try { return Intl.getCanonicalLocales(tag?.replace(/_/g, '-') || [])[0] } catch {} }

// Dates are whole days since 1970-01-01 in UTC: no time zone or DST in the way.
const DAY = 864e5
const ymd = (y, m, d) => Date.UTC(y, m, d) / DAY // m from 0; overflow rolls over
const iso = (n) => new Date(n * DAY).toISOString().slice(0, 10)
const parts = (n) => ((n = new Date(n * DAY)), [n.getUTCFullYear(), n.getUTCMonth(), n.getUTCDate()])
// An ISO date to its day, or null: 2026-02-31 is no date.
const day = (s) => {
	const m = /^(\d{4})-(\d\d)-(\d\d)$/.exec(String(s ?? '').trim())
	const n = m && ymd(m[1], m[2] - 1, m[3])
	return m && iso(n) === m[0] ? n : null
}
// The same day k months on, or that month's last day when it is shorter.
const addMonths = (n, k) => {
	const [y, m, d] = parts(n)
	return ymd(y, m + k, Math.min(d, parts(ymd(y, m + k + 1, 0))[2]))
}
const today = () => ((n) => ymd(n.getFullYear(), n.getMonth(), n.getDate()))(new Date())
const lohi = (a, b) => (a > b ? [b, a] : [a, b])
const anchors = CSS.supports('anchor-name: --a')

const styles = /* css */ `
:host {
	--_bg: var(--sb-control-bg, #0B1224);
	--_border: var(--sb-control-border, #283552);
	--_border-hover: var(--sb-control-border-hover, #3A4868);
	--_text: var(--sb-control-text, #F3F4FA);
	--_placeholder: var(--sb-control-placeholder, #7785A8);
	--_label: var(--sb-text-2, #AEBBDD);
	--_muted: var(--sb-text-muted, #7785A8);
	--_panel: var(--sb-surface-raised, #10182B);
	--_hover: var(--sb-surface-hover, #1A2540);
	--_brand: var(--sb-brand, #8C6BFF);
	--_brand-light: var(--sb-brand-light, #B09AFF);
	--_brand-subtle: var(--sb-brand-subtle, rgb(140 107 255 / 0.14));
	--_on-brand: var(--sb-text-on-brand, #F3F4FA);
	--_danger: var(--sb-danger, #F2777A);
	--_radius: var(--sb-control-radius, 6px);
	--_notch: var(--sb-notch, 1);
	--_n: calc(2px * var(--_notch));
	--_dir: 1;
	display: grid;
	gap: 0.4rem;
	max-inline-size: 20rem;
}
:host([hidden]) { display: none; }
:host(:dir(rtl)) { --_dir: -1; }
:host(:state(disabled)) { opacity: 0.5; pointer-events: none; }
label { color: var(--_label); font-size: 0.8125rem; font-weight: 600; }
.control {
	display: flex;
	align-items: center;
	border: 1px solid var(--_border);
	border-radius: var(--_radius);
	background: var(--_bg);
	anchor-name: --sb-date;
	transition: border-color 120ms, box-shadow 120ms;
}
.control:hover { border-color: var(--_border-hover); }
.control:focus-within { border-color: var(--_brand-light); box-shadow: 0 0 0 3px var(--_brand-subtle); }
.invalid { border-color: var(--_danger); }
input {
	all: unset;
	flex: 1;
	min-inline-size: 0;
	block-size: 2.75rem;
	padding-inline: 0.875rem;
	color: var(--_text);
	font-variant-numeric: tabular-nums;
}
input::placeholder { color: var(--_placeholder); }
button {
	all: unset;
	display: grid;
	grid-auto-flow: column;
	place-content: center;
	gap: 1px;
	inline-size: 1.75rem;
	block-size: 1.75rem;
	border-radius: calc(var(--_radius) - 2px);
	color: var(--_muted);
	cursor: pointer;
}
button:hover { color: var(--_text); background: var(--_hover); }
.control button { inline-size: 2.25rem; block-size: 2.25rem; margin-inline-end: 0.25rem; }
button[aria-disabled=true] { opacity: 0.35; cursor: default; background: none; }
/* Rings: the control's glow shows the input's focus. Transparent outlines
   become visible in forced colours. */
:focus-visible { outline: 2px solid var(--_brand-light); outline-offset: -2px; }
input:focus-visible { outline-color: transparent; }
svg { inline-size: 1rem; block-size: 1rem; }
.error { color: var(--_danger); font-size: 0.75rem; }
/* Never display: none: a live region announces only once it is there. */
.error:empty { position: absolute; }
[popover] {
	margin: 0;
	padding: 0;
	border: 0;
	background: none;
	overflow: visible;
	filter: drop-shadow(0 12px 24px rgb(0 0 0 / 0.55));
}
@supports (anchor-name: --a) {
	[popover] { position-anchor: --sb-date; inset: auto; position-area: block-end span-inline-end; margin-block-start: 4px; position-try-fallbacks: flip-block, flip-inline; }
}
#cal { justify-self: start; }
.cal {
	padding: 0.5rem;
	background: var(--_panel);
	color: var(--_text);
	box-shadow: inset 0 0 0 1px var(--_border);
	clip-path: ${notch('var(--_n)')};
	border-radius: calc(var(--_radius) * (1 - var(--_notch)));
}
.head { display: flex; align-items: center; gap: 2px; }
.title { flex: 1; text-align: center; font-size: 0.875rem; font-weight: 600; }
/* Pixel arrows, pointing to the inline start (back) or end (on). */
i { inline-size: 4px; block-size: 7px; background: currentColor; clip-path: polygon(0 0, 1px 0, 1px 1px, 2px 1px, 2px 2px, 3px 2px, 3px 3px, 4px 3px, 4px 4px, 3px 4px, 3px 5px, 2px 5px, 2px 6px, 1px 6px, 1px 7px, 0 7px); scale: var(--_dir) 1; }
.back i { scale: calc(-1 * var(--_dir)) 1; }
table { border-collapse: collapse; table-layout: fixed; inline-size: 15.75rem; margin-block-start: 0.25rem; }
th { block-size: 1.75rem; padding: 0; overflow: hidden; color: var(--_muted); font-size: 0.6875rem; font-weight: 600; }
td {
	position: relative;
	inline-size: 2.25rem;
	block-size: 2.25rem;
	padding: 0;
	text-align: center;
	font-size: 0.8125rem;
	font-variant-numeric: tabular-nums;
	clip-path: ${notch('var(--_n)')};
	border-radius: calc((var(--_radius) - 2px) * (1 - var(--_notch)));
}
[part~=day] { cursor: pointer; }
[part~=day]:hover { background: var(--_hover); }
[part~=today] { color: var(--_brand-light); font-weight: 700; }
[part~=today]::after { content: ""; position: absolute; inset-inline: 35%; inset-block-end: 5px; block-size: 2px; background: currentColor; }
[part~=range] { background: var(--_brand-subtle); clip-path: none; border-radius: 0; }
.picking [part~=range] { background: var(--_hover); }
[part~=day][part~=selected] { background: var(--_brand); color: var(--_on-brand); clip-path: ${notch('var(--_n)')}; }
[part~=disabled] { color: var(--_muted); text-decoration: line-through; cursor: default; }
[part~=disabled]:hover { background: none; }
@media (prefers-reduced-motion: reduce) { .control { transition: none; } }
@media (forced-colors: active) {
	i, [part~=today]::after { forced-color-adjust: none; background: CanvasText; }
	[part~=range], [part~=day][part~=selected] { forced-color-adjust: none; background: Highlight; color: HighlightText; }
	[part~=disabled] { color: GrayText; }
	.cal { outline: 1px solid CanvasText; outline-offset: -1px; }
}
`

rocket('sb-date-picker', {
	props: ({ bool, json, oneOf, string }) => ({
		value: string.docs({ description: 'The date, ISO: 2026-09-29. With mode="range", JSON: {"start":"2026-09-29","end":"2026-10-03"}. A new value from the server replaces it (value="" clears); the live value is the value property.' }),
		mode: oneOf('single', 'range').default('single').docs({ description: 'One date, or a start and an end, committed together as one value.' }),
		min: string.docs({ description: 'Earliest date that can be picked (ISO).' }),
		max: string.docs({ description: 'Latest date that can be picked (ISO).' }),
		disabledDates: json.default(() => []).docs({ description: 'Server data: dates that can\'t be picked, as a JSON array of ISO dates (e.g. booked days). They can still be focused and read.' }),
		month: string.docs({ description: 'The month shown, YYYY-MM (view state). A changed attribute from the server moves the calendar there; the user paging months emits sb-month.' }),
		open: bool.docs({ description: 'The calendar popover is open. View state: a changed attribute from the server opens or closes it (open="false" closes); local toggling never reflects it.' }),
		inline: bool.docs({ description: 'Show the calendar on the page, always visible, without the text field.' }),
		label: string.trim.docs({ description: 'Visible label, and the accessible name of the calendar.' }),
		placeholder: string.docs({ description: 'Placeholder text (default: the locale\'s date pattern, e.g. dd.mm.yyyy).' }),
		error: string.docs({ description: 'Message shown when the typed text is not a date that can be picked.' }),
		lang: string.trim.docs({ description: 'Locale for month and weekday names, the first day of the week and the typed format (default: the page\'s lang, then the browser\'s).' }),
		disabled: bool.docs({ description: 'Disable the field and the calendar. A form leaves it out.' }),
		confirm: bool.docs({ description: 'Server-confirmed value: :state(pending) while the local value differs from the server\'s value attribute (see revert()).' }),
		name: string.trim.docs({ description: 'Name reported in sb-change, sb-month and sb-toggle (e.g. the field of a command), and submitted with its form.' }),
	}),
	manifest: {
		events: [
			{ name: 'change', kind: 'event', bubbles: true, composed: true, description: 'The value changed.' },
			{ name: 'sb-change', kind: 'custom-event', bubbles: true, composed: true, description: 'A date was picked or typed. detail: { name, value }: an ISO date ("" when cleared), or with mode="range" { start, end } (null when cleared). Ready for a command.' },
			{ name: 'sb-month', kind: 'custom-event', bubbles: true, composed: true, description: 'The user moved the calendar to another month. detail: { name, year, month } (month from 1): the moment to send that month\'s disabled-dates.' },
			{ name: 'sb-toggle', kind: 'custom-event', bubbles: true, composed: true, description: 'The popover opened or closed. detail: { name, open }. View state: not emitted for a change the server made.' },
		],
	},
	// Rendered once: the grid holds the keyboard focus, so everything flows
	// through signals.
	renderOnPropChange: false,
	render: ({ html }) => {
		const seven = [...Array(7).keys()]
		return html`
			<label part="label" for="i" data-show="$$label" data-text="$$label"></label>
			<div class="control" part="control" data-ref:control data-show="!$$inline" data-class:invalid="$$invalid"
				data-on:keydown="evt.key === 'Escape' && $$open && evt.stopPropagation()">
				<input id="i" part="input" autocomplete="off" spellcheck="false" aria-describedby="e"
					data-attr:aria-label="$$label ? null : $$aria"
					data-attr:placeholder="$$ph"
					data-attr:disabled="$$disabled"
					data-attr:aria-invalid="String($$invalid)"
					data-bind:text
					data-on:input="evt.stopPropagation()"
					data-on:change="@typed()"/>
				<button type="button" part="button" data-ref:button popovertarget="cal" aria-label="Choose date" aria-haspopup="dialog" aria-controls="cal"
					data-attr:aria-expanded="String(!!$$open)"
					data-attr:disabled="$$disabled"><svg viewBox="0 0 16 16" aria-hidden="true"><path fill="currentColor" fill-rule="evenodd" d="M2 3h12v11H2zM3 7h10v6H3zM4 1h2v2H4zm6 0h2v2h-2zM5 9h2v2H5z"/></svg></button>
			</div>
			<span class="error" part="error" id="e" aria-live="polite" data-text="$$invalid && $$error || ''"></span>
			<div id="cal" data-ref:cal
				data-attr:popover="$$inline ? null : 'auto'"
				data-attr:role="$$inline ? 'group' : 'dialog'"
				data-attr:aria-modal="$$inline ? null : 'true'"
				data-attr:aria-label="$$label || 'Choose date'"
				data-effect="el.popover && el.togglePopover(!!$$open)"
				data-on:beforetoggle="@before()"
				data-on:toggle="@toggled()"
				data-on:keydown="@key()">
				<div class="cal" part="calendar">
					<div class="head">
						<button type="button" class="back" part="nav" data-attr:disabled="$$disabled" aria-label="Previous year" data-attr:aria-disabled="$$back ? null : 'true'" data-on:click="@page(-12)"><i></i><i></i></button>
						<button type="button" class="back" part="nav" data-attr:disabled="$$disabled" aria-label="Previous month" data-attr:aria-disabled="$$back ? null : 'true'" data-on:click="@page(-1)"><i></i></button>
						<div class="title" part="title" id="t" aria-live="polite" data-text="$$title"></div>
						<button type="button" part="nav" data-attr:disabled="$$disabled" aria-label="Next month" data-attr:aria-disabled="$$on ? null : 'true'" data-on:click="@page(1)"><i></i></button>
						<button type="button" part="nav" data-attr:disabled="$$disabled" aria-label="Next year" data-attr:aria-disabled="$$on ? null : 'true'" data-on:click="@page(12)"><i></i><i></i></button>
					</div>
					<table role="grid" part="grid" aria-labelledby="t" data-class:picking="$$picking">
						<thead><tr>${seven.map((d) => html`<th scope="col" data-attr:abbr="$$w${d}.l" data-text="$$w${d}.s"></th>`)}</tr></thead>
						<tbody data-ref:days>
							${[...Array(6).keys()].map(
								(w) => html`<tr>${seven.map(
									(d) => html`<td
										data-attr:part="$$c${w * 7 + d}.p"
										data-attr:tabindex="$$c${w * 7 + d}.f"
										data-attr:aria-selected="$$c${w * 7 + d}.s"
										data-attr:aria-disabled="$$c${w * 7 + d}.d"
										data-attr:aria-current="$$c${w * 7 + d}.k"
										data-attr:aria-label="$$c${w * 7 + d}.l"
										data-text="$$c${w * 7 + d}.t"
										data-on:click="@tap(${w * 7 + d})"
										data-on:pointerover="@over(${w * 7 + d})"></td>`,
								)}</tr>`,
							)}
						</tbody>
					</table>
				</div>
			</div>
		`
	},
	onFirstRender: ({ $$, action, adoptStyles, cleanup, defineHostProp, effect, emit, host, observeProps, overrideProp, props, refs: { control, button, cal, days } }) => {
		adoptStyles(host, styles)
		const states = internalsOf(host).states
		const range = () => props.mode === 'range'

		// The value as one string, so values compare: "" or an ISO date, or with
		// range "" or {"start","end"} JSON in order (what a form submits).
		const norm = (v) => {
			if (!range()) return (v = day(v)) == null ? '' : iso(v)
			try {
				v = typeof v === 'string' ? JSON.parse(v) : v
			} catch {}
			const [a, b] = lohi(...(Array.isArray(v) ? v : [v?.start, v?.end]).map(day))
			return a == null || b == null ? '' : JSON.stringify({ start: iso(a), end: iso(b) })
		}
		const out = (v) => (range() ? (v ? JSON.parse(v) : null) : v)
		const ends = (v) => (range() ? (v ? Object.values(JSON.parse(v)) : []) : [v]).map(day).filter((n) => n != null)
		const text = (v) => ends(v).map((n) => fmt.format(n * DAY)).join(' – ')

		// The language: formats, the first day of the week, the typed order and
		// the placeholder. The typed format is the locale's numeric one
		// (dd.mm.yyyy, mm/dd/yyyy, yyyy/mm/dd…), in its own digits.
		let fmt, full, title, num, digits, order, first, lo, hi, off
		const speak = () => {
			let el = host, l
			while (!(l = el.closest('[lang]')) && (el = el.getRootNode().host));
			const lang = locale(l?.lang)
			const dtf = (o) => new Intl.DateTimeFormat(lang, { calendar: 'gregory', timeZone: 'UTC', ...o })
			fmt = dtf({ year: 'numeric', month: '2-digit', day: '2-digit' })
			full = dtf({ dateStyle: 'full' })
			title = dtf({ year: 'numeric', month: 'long' })
			num = new Intl.NumberFormat(lang, { useGrouping: false })
			digits = [...'0123456789'].map((d) => num.format(d))
			try {
				const w = new Intl.Locale(fmt.resolvedOptions().locale)
				first = (w.getWeekInfo?.() ?? w.weekInfo).firstDay % 7
			} catch {
				first = 1
			}
			const ps = fmt.formatToParts(0)
			order = ps.map((p) => p.type).filter((t) => t !== 'literal')
			const pattern = ps.map((p) => ({ day: 'dd', month: 'mm', year: 'yyyy' })[p.type] ?? p.value).join('')
			$$.ph = props.placeholder || (range() ? `${pattern} – ${pattern}` : pattern)
			// 1970-01-04 (day 3) was a Sunday.
			const wd = (o, d) => dtf({ weekday: o }).format((3 + ((first + d) % 7)) * DAY)
			for (let d = 0; d < 7; d++) {
				const s = wd('short', d)
				$$['w' + d] = { s: s.length > 4 ? wd('narrow', d) : s, l: wd('long', d) }
			}
		}
		// Bounds and ruled-out days.
		const bound = () => {
			lo = day(props.min) ?? -Infinity
			hi = day(props.max) ?? Infinity
			off = new Set([props.disabledDates].flat().map(day))
		}
		// The other props the template reads.
		const mirror = () => {
			for (const k of ['label', 'error', 'inline', 'disabled']) $$[k] = props[k]
			$$.aria = host.getAttribute('aria-label') || 'Date'
			states[props.disabled ? 'add' : 'delete']('disabled')
		}
		speak()
		bound()
		mirror()
		const ok = (n) => n >= lo && n <= hi && !off.has(n)
		const clamp = (n) => Math.min(Math.max(n, lo), hi)
		// Typed text: an ISO date, or three numbers in the locale's order with a
		// four-digit year. Anything else is not a date: never guess.
		const read = (s) => {
			s = s.replace(/\p{Nd}/gu, (c) => (digits.includes(c) ? digits.indexOf(c) : c))
			let n = day(s)
			const g = s.match(/\d+/g)
			if (n == null && g?.length === 3) {
				const o = Object.fromEntries(order.map((t, i) => [t, g[i]]))
				n = o.year?.length === 4 && o.month?.length < 3 && o.day?.length < 3 ? day(`${o.year}-${o.month.padStart(2, '0')}-${o.day.padStart(2, '0')}`) : null
			}
			return n != null && ok(n) ? n : null
		}

		// The month shown, the day with the grid's focus (always in that month),
		// the grid's first day, the first pick of a range and the day pointed at.
		let vy, vm, focus, g, f1, len
		let pick1 = null
		let hover = null
		const here = (n) => n >= f1 && n < f1 + len
		const show = (n, quiet) => {
			const [y, m] = parts(n)
			if (y === vy && m === vm) return
			;[vy, vm] = [y, m]
			quiet || emit('sb-month', { name: props.name, year: y, month: m + 1 })
		}
		// Every date is a cell with part="day today selected range start end
		// disabled", as they apply; the other months' cells are blank. false
		// drops an attribute: null would delete the signal its effect follows.
		const paint = () =>
			peek(() => {
				f1 = ymd(vy, vm, 1)
				len = ymd(vy, vm + 1, 1) - f1
				g = f1 - ((new Date(f1 * DAY).getUTCDay() - first + 7) % 7)
				const R = range()
				// The range on show: the value's, or while picking, from the first
				// pick to the day pointed at.
				const e = ends($$.v)
				const [a, b] = pick1 != null ? lohi(pick1, hover ?? focus) : [e[0], e.at(-1)]
				const t = today()
				for (let i = 0; i < 42; i++) {
					const n = g + i
					const sel = pick1 != null ? n === pick1 : R ? n === a || n === b : n === b
					$$['c' + i] = !here(n)
						? { t: '', l: false, p: false, f: false, s: false, d: false, k: false }
						: {
								t: num.format(n - f1 + 1),
								l: full.format(n * DAY),
								p: ['day', n === t && 'today', sel && 'selected', R && n >= a && n <= b && 'range', R && n === a && 'start', R && n === b && 'end', !ok(n) && 'disabled'].filter(Boolean).join(' '),
								f: n === focus && !props.disabled ? '0' : '-1',
								s: sel && 'true',
								d: !ok(n) && 'true',
								k: n === t && 'date',
							}
				}
				$$.title = title.format(f1 * DAY)
				$$.back = lo < f1
				$$.on = hi >= f1 + len
				$$.picking = pick1 != null
			})
		const cell = () => days.rows[((focus - g) / 7) | 0]?.cells[(focus - g) % 7]
		// Moves the grid's focus (and the month with it), within min and max.
		const move = (n, quiet) => {
			focus = clamp(n)
			hover = null
			show(focus, quiet)
			paint()
		}

		// The value: $$.v is the local one, the attribute the server's.
		const set = (v) => {
			$$.v = v
			$$.text = text(v)
			$$.invalid = false
			pick1 = null
			paint()
		}
		const commit = (v) => {
			const was = $$.v
			set(v)
			if (v !== was) {
				emit('change')
				emit('sb-change', { name: props.name, value: out(v) })
			}
		}
		const mon = (s) => (/^\d{4}-\d\d$/.test(s) ? day(s + '-01') : null)
		const toMonth = (n) => {
			const [y, m] = parts(n)
			move(addMonths(focus, (y - vy) * 12 + m - vm), true)
		}
		// The first month: month, else the value's, else today's. The focus is on
		// the value (or today) when it is in that month.
		const v0 = norm(props.value)
		focus = ends(v0)[0] ?? today()
		if (mon(props.month) != null && !iso(focus).startsWith(props.month)) focus = mon(props.month)
		focus = clamp(focus)
		;[vy, vm] = parts(focus)
		set(v0)

		const sync = () => peek(() => states[props.confirm && $$.v !== norm(props.value) ? 'add' : 'delete']('pending'))
		effect(() => $$.v != null && sync())
		// Only what a prop changes is rebuilt, and the grid is painted once.
		observeProps((p, changes) =>
			peek(() => {
				const has = (...k) => k.some((x) => x in changes)
				if (has('lang', 'placeholder', 'mode')) speak()
				if (has('min', 'max', 'disabledDates')) bound()
				if (p.disabled || p.inline) $$.open = false
				mirror()
				if (has('lang')) $$.invalid || ($$.text = text($$.v))
				if (has('mode')) set(norm(p.value))
				else if (has('lang', 'min', 'max', 'disabledDates', 'disabled')) paint()
				sync()
			}),
		)

		// The popover (auto: light dismiss and Escape are the browser's). $$.open
		// follows it, and the server's open drives it through the data-effect: a
		// toggle that finds $$.open already there is the server's, and quiet.
		action('before', ({ evt }) =>
			peek(() => {
				pick1 = null
				// Closing with the focus inside: it goes back to the button.
				if (evt.newState !== 'open') return cal.contains(host.shadowRoot.activeElement) && button.focus()
				// The user's open shows the value's month (or today's).
				$$.open ? paint() : move(ends($$.v)[0] ?? today())
				if (anchors) return
				const r = control.getBoundingClientRect()
				const rtl = host.matches(':dir(rtl)')
				Object.assign(cal.style, { position: 'fixed', inset: 'auto', top: r.bottom + 4 + 'px', [rtl ? 'right' : 'left']: (rtl ? document.documentElement.clientWidth - r.right : r.left) + 'px' })
			}),
		)
		action('toggled', ({ evt }) =>
			peek(() => {
				const o = evt.newState === 'open'
				if (o === !!$$.open) return
				$$.open = o
				emit('sb-toggle', { name: props.name, open: o })
				o && cell()?.focus()
			}),
		)

		// The server's word: a changed value, open or month attribute wins. A
		// removed one is ignored (morphs also strip reflected attributes), and
		// the same word again leaves the local state alone. Watched on the
		// attributes: observeProps is silent when the decoded value is the same.
		const said = {}
		const heard = (k) => {
			const a = host.getAttribute(k)
			const news = a !== null && a !== said[k]
			said[k] = a
			return news
		}
		for (const k of ['value', 'open', 'month']) heard(k)
		const watch = new MutationObserver(() =>
			peek(() => {
				if (heard('value')) set(norm(props.value))
				if (heard('open')) $$.open = props.open && !props.inline && !props.disabled
				if (!heard('month') || mon(props.month) == null) return
				const had = host.shadowRoot.activeElement?.localName === 'td'
				toMonth(mon(props.month))
				had && cell()?.focus()
			}),
		)
		watch.observe(host, { attributeFilter: ['value', 'open', 'month'] })

		overrideProp('value', () => peek(() => out($$.v)), (v) => peek(() => set(norm(v))))
		overrideProp('open', () => peek(() => !!$$.open), (v) => peek(() => cal.popover && !props.disabled && cal.togglePopover(!!v)))
		overrideProp('month', () => iso(ymd(vy, vm, 1)).slice(0, 7), (v) => peek(() => mon(v) != null && toMonth(mon(v))))
		const revert = () => peek(() => (set(norm(props.value)), sync()))
		defineHostProp('revert', { value: revert })

		// Forms: until Rocket can make this element form-associated, join the
		// submissions and resets of the form it sits in, heard on the root once
		// every listener on the form has run. The entry is the value's string;
		// a reset is revert(): the server's value, no events.
		const form = host.closest('form')
		const root = host.getRootNode()
		const onData = (evt) => evt.target === form && peek(() => props.name && !props.disabled && evt.formData.append(props.name, $$.v))
		const onReset = (evt) => evt.target === form && !evt.defaultPrevented && revert()
		root.addEventListener('formdata', onData)
		root.addEventListener('reset', onReset)
		cleanup(() => {
			root.removeEventListener('formdata', onData)
			root.removeEventListener('reset', onReset)
			watch.disconnect()
		})

		const choose = (n) => {
			if (!ok(n) || props.disabled) return
			if (range() && pick1 == null) return (pick1 = n), paint()
			const [a, b] = lohi(pick1 ?? n, n)
			commit(range() ? JSON.stringify({ start: iso(a), end: iso(b) }) : iso(n))
			$$.open && cal.hidePopover()
		}
		// The typed text, committed on change (Enter or leaving the field).
		action('typed', () =>
			peek(() => {
				const s = $$.text.trim()
				const ns = s ? (range() ? s.split(/\s+-\s+|\s*[–—]\s*/) : [s]).map(read) : []
				if (ns.includes(null) || (ns.length && ns.length !== (range() ? 2 : 1))) return ($$.invalid = true)
				const [a, b] = lohi(ns[0], ns[1])
				commit(!ns.length ? '' : range() ? JSON.stringify({ start: iso(a), end: iso(b) }) : iso(ns[0]))
			}),
		)
		action('page', (_, k) => move(addMonths(focus, k)))
		// The days: i is the cell's place in the grid.
		action('tap', (_, i) =>
			peek(() => {
				if (!here(g + i)) return
				focus = g + i
				paint()
				choose(focus)
			}),
		)
		action('over', (_, i) => peek(() => pick1 != null && here(g + i) && g + i !== hover && ((hover = g + i), paint())))
		action('key', ({ el, evt }) =>
			peek(() => {
				const k = evt.key
				if (k === 'Escape') {
					// The browser closes the popover. Stopped here, the Escape can't also
					// close a drawer or popover the picker sits in. Inline, it drops a
					// half-picked range.
					if ($$.open) return evt.stopPropagation()
					if (pick1 == null) return
					pick1 = null
					paint()
				} else if (k === 'Tab' && $$.open) {
					// A dialog: the focus stays in it.
					const f = [...el.querySelectorAll('button, [tabindex="0"]')]
					f[(f.indexOf(host.shadowRoot.activeElement) + (evt.shiftKey ? -1 : 1) + f.length) % f.length]?.focus()
				} else if (evt.target.localName === 'td') {
					const x = host.matches(':dir(rtl)') ? -1 : 1
					const w = (new Date(focus * DAY).getUTCDay() - first + 7) % 7 // place in the week
					const s = evt.shiftKey ? 12 : 1
					const to = { ArrowLeft: focus - x, ArrowRight: focus + x, ArrowUp: focus - 7, ArrowDown: focus + 7, Home: focus - w, End: focus + 6 - w, PageUp: addMonths(focus, -s), PageDown: addMonths(focus, s) }[k]
					if (to != null) move(to), queueMicrotask(() => cell()?.focus())
					else if (k === 'Enter' || k === ' ') choose(focus)
					else return
				} else return
				evt.preventDefault()
			}),
		)
		// A calendar the server rendered open shows (the data-effect), now that the
		// actions its toggle events call are there.
		$$.open = props.open && !props.inline && !props.disabled
	},
})
