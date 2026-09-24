import { createCodec, rocket } from 'datastar'

// A host for Apache ECharts that a server can drive: the page sends an ECharts
// option as JSON, and the element themes it from the page's tokens, keeps it in
// the reader's language, follows theme changes, fits it to its box and animates
// every new option into place.
//
// Ported from Libretto's <tco-chart>, which draws every chart in that app from
// options its Go server builds.

// ---- page-level extension points --------------------------------------------
//
// Both are exported for the page's own code (a component may import only
// 'datastar' and its own files, but a page may import this module). Every URL
// of this module is a module of its own (a pinned .min.js, the readable file, a
// bundle that exports nothing), and the page's import is rarely the one that
// defined the element: so what they set lives on the page, shared by every copy.
const reg = (globalThis[Symbol.for('sb-echarts')] ??= { kinds: new Map(), charts: new Set() })

// Kinds are charts whose shape is code. A custom series draws through a
// renderItem *function*, which no server can send as JSON; so the page defines
// a builder under a name, and an option { kind: 'name', ... } is expanded by it
// (with the context below) before theming. A chart that names a kind before the
// page has defined it draws as soon as it is.
export const defineChartKind = (name, build) => {
	reg.kinds.set(name, build)
	reg.charts.forEach((redraw) => redraw(name))
}

// The numbers ECharts writes itself – value-axis labels, tooltip rows – use
// Intl.NumberFormat in the element's language. A page whose server formats
// numbers its own way sets the same formatter here, so a chart never disagrees
// with the figures around it.
export const setNumberFormat = (fn) => void (reg.format = fn)

// ECharts itself (1.1 MB) is loaded the first time a chart comes near the
// screen, once per page.
let lib = null
const load = () => (lib ??= import('./vendor/echarts.esm.min.js'))

// ---- option JSON ------------------------------------------------------------

const optionCodec = createCodec({
	decode(v) {
		if (typeof v !== 'string' || !v.trim()) return null
		try {
			const o = JSON.parse(v)
			return o && typeof o === 'object' && !Array.isArray(o) && Object.keys(o).length ? o : null
		} catch (e) {
			console.error('<sb-echarts>: option is not valid JSON', e)
			return null
		}
	},
	encode: (v) => (v ? JSON.stringify(v) : ''),
})

const isObj = (v) => v !== null && typeof v === 'object' && !Array.isArray(v)

// A value ECharts treats as absent: its '-' placeholder, null, undefined or NaN.
const missing = (v) => v == null || v === '-' || Number.isNaN(v)
// The value an axis tooltip row shows, as ECharts picks it: the number itself,
// or the last dimension the series puts on the other axis – of an [x, y] pair,
// or of a dataset row, which carries every series' column.
const VALUE_AXIS = { x: 'y', y: 'x', angle: 'radius', radius: 'angle' }
const valueOf = (p) => {
	const v = p.value, i = p.encode?.[VALUE_AXIS[p.axisDim]]?.at(-1)
	return !v || typeof v !== 'object' ? v : Array.isArray(v) ? v[i ?? v.length - 1] : v[p.dimensionNames?.[i]]
}
// Series names come from the server and a tooltip formatter returns HTML.
const escapeHTML = (s) => String(s).replace(/[&<>"']/g, (c) => `&#${c.charCodeAt(0)};`)

// An axis tooltip lists every series at the hovered point, including those
// with no value there – a projection before it starts, costs in a month not
// yet lived – which would read as a row of "-". Leave them out, drawing the
// rest in ECharts' own layout, each value in its series' format if it has one.
const axisTooltip = (params, format, series) => {
	const rows = [params].flat().filter((p) => !missing(valueOf(p)))
	if (!rows.length) return ''
	const row = (p) =>
		`<div style="margin:10px 0 0;line-height:1">${p.marker ?? ''}<span style="margin-left:2px">${escapeHTML(p.seriesName ?? '')}</span>` +
		`<span style="float:right;margin-left:20px;font-weight:700">${escapeHTML((series[p.seriesIndex]?.tooltip?.valueFormatter ?? format)(valueOf(p)))}</span><div style="clear:both"></div></div>`
	return `<div style="line-height:1;opacity:.72">${escapeHTML(rows[0].axisValueLabel ?? '')}</div>${rows.map(row).join('')}`
}

// deepMerge lays the page's option over the themed defaults: objects merge,
// everything else (arrays included) replaces.
const deepMerge = (base, over) => {
	const out = { ...base }
	for (const [k, v] of Object.entries(over)) out[k] = isObj(v) && isObj(out[k]) ? deepMerge(out[k], v) : v
	return out
}

// A language tag as Intl takes it (en_US, as some CMSs write it, is en-US), or
// nothing for one it cannot read.
const locale = (tag) => { try { return Intl.getCanonicalLocales(tag?.replace(/_/g, '-') || [])[0] } catch {} }

// Month and weekday names in the element's language, from Intl, registered once
// per language. ECharts ships only a handful of locales of its own, and fills
// in everything else (legend, toolbox, aria) from its English one.
const locales = new Set()
const localeFor = (ec, lang) => {
	const code = 'sb-' + lang
	if (locales.has(code)) return code
	const fmt = (opt) => (d) => new Intl.DateTimeFormat(lang, { ...opt, timeZone: 'UTC' }).format(d)
	const months = Array.from({ length: 12 }, (_, m) => new Date(Date.UTC(2024, m, 1)))
	const days = Array.from({ length: 7 }, (_, d) => new Date(Date.UTC(2024, 0, 7 + d))) // a Sunday first
	ec.registerLocale(code, {
		time: { month: months.map(fmt({ month: 'long' })), monthAbbr: months.map(fmt({ month: 'short' })), dayOfWeek: days.map(fmt({ weekday: 'long' })), dayOfWeekAbbr: days.map(fmt({ weekday: 'short' })) },
	})
	locales.add(code)
	return code
}

// ---- colours ------------------------------------------------------------------

// Every colour handed to ECharts is plain rgb()/rgba(). getComputedStyle returns
// modern syntax (oklch(), color(srgb …)) for tokens built with oklch or
// color-mix, which a canvas paints but ECharts' own colour parser does not – and
// ECharts parses an item's colour to lighten it for the hover state. With one it
// cannot read, the hovered bar is drawn with no colour at all and flickers in
// and out under the pointer. A 1×1 canvas turns anything the browser understands
// into sRGB.
const pixel = document.createElement('canvas').getContext('2d', { willReadFrequently: true })
const rgba = (css) => {
	pixel.clearRect(0, 0, 1, 1)
	pixel.fillStyle = '#000'
	pixel.fillStyle = css
	pixel.fillRect(0, 0, 1, 1)
	const [r, g, b, a] = pixel.getImageData(0, 0, 1, 1).data
	return a === 255 ? `rgb(${r}, ${g}, ${b})` : `rgba(${r}, ${g}, ${b}, ${+(a / 255).toFixed(3)})`
}

// ---- styles -------------------------------------------------------------------

const PALETTE = [
	['--sb-chart-1', '--sb-brand-light', '#B09AFF'],
	['--sb-chart-2', '--sb-accent', '#65BFFF'],
	['--sb-chart-3', '--sb-ok', '#6EF59A'],
	['--sb-chart-4', '--sb-warn', '#F5C451'],
	['--sb-chart-5', '--sb-danger', '#F2777A'],
	['--sb-chart-6', '--sb-info', '#7DD3FC'],
	['--sb-chart-7', '--sb-text-2', '#AEBBDD'],
	['--sb-chart-8', '--sb-brand', '#8C6BFF'],
]

// Without font tokens the chart uses the element's own font, inherited from
// the page (font() below), so --_font and --_numbers have no fallbacks. A chart
// fills the width it is given, also as a flex or grid item, where a block whose
// only content is absolutely positioned would shrink to nothing.
const styles = /* css */ `
:host {
	--_text: var(--sb-text-2, #AEBBDD);
	--_strong: var(--sb-text-1, #F3F4FA);
	--_muted: var(--sb-text-muted, #7785A8);
	--_line: var(--sb-border, #283552);
	--_grid: var(--sb-border-subtle, #1B2640);
	--_tip: var(--sb-surface-overlay, #141D32);
	--_font: var(--sb-font-body);
	--_numbers: var(--sb-font-ui, var(--sb-font-body));
	${PALETTE.map(([own, semantic, fallback], i) => `--_c${i + 1}: var(${own}, var(${semantic}, ${fallback}));`).join('\n\t')}
	display: block;
	position: relative;
	inline-size: 100%;
	min-inline-size: 0;
	block-size: 18rem;
}
.plot { position: absolute; inset: 0; }
.fallback { position: absolute; inline-size: 1px; block-size: 1px; overflow: hidden; clip-path: inset(50%); white-space: nowrap; }
:host(:not(:state(ready))) .fallback { position: static; inline-size: auto; block-size: auto; clip-path: none; white-space: normal; }
.probe { position: absolute; visibility: hidden; }
`

// ---- the chart ----------------------------------------------------------------

// setup keeps each element's chart in its closure; onFirstRender hands it the plot.
const plots = new WeakMap()

// One ElementInternals per element: attachInternals() works once, and setup
// reruns when the element is re-attached. The ready state lives there rather
// than in an attribute, which the next server morph would strip.
const internals = new WeakMap()
const internalsOf = (host) => internals.get(host) ?? internals.set(host, host.attachInternals()).get(host)

// The keyboard's steps: [series, item].
const KEYS = { ArrowRight: [0, 1], ArrowLeft: [0, -1], ArrowDown: [1, 0], ArrowUp: [-1, 0], Enter: [0, 0] }
const clamp = (v, max) => Math.max(0, Math.min(v, max))

rocket('sb-echarts', {
	props: ({ oneOf, string }) => ({
		option: optionCodec.docs({ description: 'The ECharts option, as JSON. Strings that are exactly var(--token) become that colour. { "kind": "name", ... } is expanded by a builder the page defined with defineChartKind().' }),
		renderer: oneOf('canvas', 'svg').default('canvas').docs({ description: 'Draw on a canvas, or as SVG (sharper when printed, slower with many points).' }),
		lang: string.trim.docs({ description: "Locale for numbers, months and weekdays (default: the page's lang, then the browser's)." }),
	}),
	manifest: {
		slots: [{ name: '', description: 'A fallback for readers without the chart: shown until ECharts has drawn, kept for screen readers after that. A table of the same data is ideal.' }],
		events: [{ name: 'sb-chart-click', kind: 'custom-event', bubbles: true, composed: true, description: 'A click on a data item. detail: { seriesName, seriesIndex, name, value, dataIndex }.' }],
	},
	renderOnPropChange: false,
	setup: ({ action, adoptStyles, cleanup, emit, host, observeProps, props }) => {
		adoptStyles(host, styles)
		let plot = null // from onFirstRender
		let alive = true, visible = false, starting = false
		let ec = null, chart = null, lang = ''
		// The last built option before its grid was fitted (for a resize to fit
		// again), and the margins it got; sized: its kind read the plot's size.
		let unfitted = null, fitted = '', measure = null, sized = false
		let si = 0, at = -1 // the series and item the keyboard is on
		plots.set(host, (p) => ((plot = p), start()))

		const langOf = () => locale(props.lang || host.closest('[lang]')?.lang) ?? navigator.language
		// A token's value as the page resolves it, in the element's own scope (so
		// properties set on the host apply). Colours go through a real `color`
		// property: getPropertyValue returns light-dark() and var() chains verbatim,
		// which a canvas cannot paint.
		const css = (token) => getComputedStyle(host).getPropertyValue(token).trim()
		// The font the canvas draws with: the token when a page sets one, else
		// whatever font the element inherits from the page.
		const font = () => css('--_font') || getComputedStyle(host).fontFamily
		const color = (value) => {
			const probe = document.createElement('span')
			probe.className = 'probe'
			probe.style.color = value.startsWith('--') ? `var(${value})` : value
			host.shadowRoot.append(probe)
			const c = getComputedStyle(probe).color
			probe.remove()
			return rgba(c)
		}
		// ECharts passes a missing value as null, undefined or its own '-'
		// placeholder, and draws '-' for it when no formatter is set: say the same,
		// never "undefined".
		const formatNumber = (n) =>
			missing(n) ? '-'
			: typeof n !== 'number' || !Number.isFinite(n) ? String(n)
			: reg.format ? reg.format(n, lang)
			: new Intl.NumberFormat(lang, { maximumFractionDigits: 20 }).format(n)

		// Strings that are exactly var(--token) become the colour they resolve to, so
		// a server can colour a series with the page's own tokens.
		const resolveVars = (v) =>
			typeof v === 'string' ? (/^var\(--[\w-]+\)$/.test(v) ? color(v.slice(4, -1)) : v)
			: Array.isArray(v) ? v.map(resolveVars)
			: isObj(v) ? Object.fromEntries(Object.entries(v).map(([k, x]) => [k, resolveVars(x)]))
			: v

		const build = (input) => {
			lang = langOf()
			let option = input
			if (option.kind) {
				const kind = reg.kinds.get(option.kind)
				if (!kind) return void console.error(`<sb-echarts>: no chart kind "${option.kind}"; register it with defineChartKind()`)
				// The context a kind builds with: everything it needs to draw in the
				// page's theme and language, without reaching into the element. A kind
				// that reads the size is built again when the size changes.
				option = kind(option, { echarts: ec, color, css, formatNumber, lang, get width() { return (sized = true), plot.clientWidth }, get height() { return (sized = true), plot.clientHeight } })
			}
			const text = color('--_text'), muted = color('--_muted'), line = color('--_line'), f = font(), numbers = css('--_numbers') || f
			const axis = {
				axisLine: { lineStyle: { color: line } },
				axisTick: { lineStyle: { color: line } },
				axisLabel: { color: muted, fontFamily: f },
				splitLine: { lineStyle: { color: color('--_grid') } },
				nameTextStyle: { color: muted, fontFamily: f },
			}
			const base = {
				color: PALETTE.map((_, i) => color(`--_c${i + 1}`)),
				backgroundColor: 'transparent',
				animation: !matchMedia('(prefers-reduced-motion: reduce)').matches,
				animationDuration: 700,
				animationEasing: 'cubicOut',
				animationDurationUpdate: 450,
				animationEasingUpdate: 'cubicInOut',
				textStyle: { color: text, fontFamily: f },
				tooltip: {
					backgroundColor: color('--_tip'),
					borderColor: line,
					textStyle: { color: color('--_strong'), fontFamily: numbers, fontSize: 12 },
				},
				aria: { enabled: !host.querySelector(':scope > *') }, // a server table says it better
			}
			// Axes and a grid belong only to charts that plot on one, which ECharts
			// draws only with axes in the option. Giving them to a pie, a gauge, or
			// a bar on polar or calendar coordinates draws a phantom coordinate system.
			if (option.xAxis || option.yAxis) {
				base.xAxis = axis
				base.yAxis = { ...axis, axisLabel: { ...axis.axisLabel, fontFamily: numbers } }
				base.grid = { left: '3%', right: '4%', top: '10%', bottom: '12%' }
			}
			// resolveVars rebuilds every object and array: from here on the option is
			// a copy, and nothing below writes into the prop.
			const o = resolveVars(deepMerge(base, option))
			// Style a legend only where the option has one: in ECharts the mere presence
			// of a legend object switches one on.
			if (o.legend) o.legend = [o.legend].flat().map((l) => deepMerge({ textStyle: { color: muted, fontFamily: f } }, l))
			// A pie's or funnel's outside labels get ECharts' own dark text outline,
			// which reads as a smudge on a dark ground: plain text in the theme's colour.
			for (const s of [o.series].flat()) if (isObj(s) && (s.type === 'pie' || s.type === 'funnel')) s.label = { color: text, textBorderWidth: 0, ...s.label }
			// deepMerge replaced array axes wholesale; give each its themed defaults.
			for (const k of ['xAxis', 'yAxis']) if (Array.isArray(o[k])) o[k] = o[k].map((a) => deepMerge(base[k] ?? axis, a))
			// Value axes and tooltips get the element's number format, where the
			// option has not set a formatter of its own (a unit-carrying axis, a
			// custom tooltip).
			for (const [k, type] of [['xAxis', 'category'], ['yAxis', 'value']])
				for (const a of [o[k]].flat()) if (isObj(a) && (a.type ?? type) === 'value') (a.axisLabel ??= {}).formatter ??= formatNumber
			// Candlesticks and boxplots keep ECharts' axis tooltip, which lists each
			// of their values by name.
			const series = [o.series].flat()
			const own = series.some((s) => /candle|box|^k$/.test(s?.type))
			for (const t of [o.tooltip].flat())
				if (isObj(t)) {
					t.valueFormatter ??= formatNumber
					if (t.trigger === 'axis' && !own) t.formatter ??= (ps) => axisTooltip(ps, t.valueFormatter, series)
				}
			return o
		}

		// ECharts reserves no room for its legend: on a narrow box a horizontal
		// legend wraps to several rows and runs into the axis names or tick labels,
		// and at the bottom, where ECharts puts it unless told otherwise, even one
		// row covers the tick labels; so does a slider. Measure how many rows the
		// legend needs at the current width and push the grid's margin on that side
		// out to clear it (at the top only when it wraps).
		const margins = (o) => {
			let { top, bottom } = o.grid
			const legend = [o.legend].flat()[0]
			const { clientWidth: w, clientHeight: h } = plot
			const px = (v) => (typeof v === 'number' ? v : typeof v === 'string' && v.endsWith('%') ? (h * parseFloat(v)) / 100 : parseFloat(v) || 0)
			const num = (v, or) => (typeof v === 'number' ? v : or)
			const labels = isObj(legend) && legend.show !== false && (legend.orient ?? 'horizontal') === 'horizontal' ? (legend.data ?? [o.series].flat().map((s) => s?.name).filter((n) => n != null)) : []
			if (labels.length && w && h) {
				const near = (v) => v === 0 || v === '0' || (typeof v === 'string' && v.endsWith('%') && parseFloat(v) < 25) || (typeof v === 'number' && v < 60)
				const atTop = legend.top === 'top' || near(legend.top)
				const atBottom = legend.bottom === 'bottom' || near(legend.bottom) || (legend.top == null && legend.bottom == null)
				const ctx = (measure ??= document.createElement('canvas').getContext('2d'))
				ctx.font = `12px ${font()}`
				const gap = num(legend.itemGap, 10), icon = num(legend.itemWidth, 25)
				let rows = 1, x = 0
				// A scrolling legend keeps to one row.
				if (legend.type !== 'scroll')
					for (const label of labels) {
						const lw = icon + 5 + ctx.measureText(String(isObj(label) ? label.name : label)).width + gap
						if (x > 0 && x + lw > w * 0.96) (rows++, (x = lw))
						else x += lw
					}
				const block = rows * 26 + 6
				if (atTop && rows > 1) {
					const named = [o.yAxis].flat().some((a) => isObj(a) && a.name)
					top = Math.max(px(top), Math.round(num(legend.top, 8) + block + (named ? 28 : 0)))
				}
				if (atBottom) bottom = Math.max(px(bottom), Math.round(num(legend.bottom ?? 15, 8) + block + 22))
			}
			// A horizontal slider sits 15px from the bottom, 30px high plus its handle.
			for (const z of [o.dataZoom].flat()) if (z?.type === 'slider' && (z.orient ? z.orient === 'horizontal' : z.yAxisIndex == null)) bottom = Math.max(px(bottom), px(z.bottom ?? 15) + 66)
			return { top, bottom }
		}

		const click = (p) => emit('sb-chart-click', { seriesName: p.seriesName, seriesIndex: p.seriesIndex, name: p.name, value: p.value, dataIndex: p.dataIndex })

		const start = async () => {
			if (chart || starting || !plot || !visible || !props.option) return
			starting = true
			ec = await load()
			starting = false
			// Moved or removed while ECharts loaded: this setup is over, and a new
			// one (if any) starts its own chart.
			if (!alive || chart) return
			chart = ec.init(plot, null, { renderer: props.renderer, locale: localeFor(ec, langOf()) })
			chart.on('click', click)
			draw()
		}

		// Every new option replaces the last one (notMerge), and ECharts animates the
		// difference, so a server push reads as the chart moving to its new state.
		const draw = (quiet) => {
			if (!chart) return
			unfitted = null
			sized = false
			// Nothing to draw (no option, or a kind the page hasn't defined yet):
			// the fallback shows until there is.
			const option = props.option && build(props.option)
			if (!option) return void (chart.clear(), internalsOf(host).states.delete('ready'))
			if (quiet) option.animationDurationUpdate = 0
			if (option.xAxis && isObj(option.grid)) {
				unfitted = { ...option }
				const m = margins(option)
				fitted = JSON.stringify(m)
				option.grid = { ...option.grid, ...m }
			}
			// An option ECharts cannot draw (a bar without axes, say) throws halfway
			// and leaves the instance refusing every later option: start afresh with
			// the next one.
			try {
				chart.setOption(option, true)
			} catch (e) {
				dispose()
				return console.error('<sb-echarts>: ECharts cannot draw this option', e)
			}
			internalsOf(host).states.add('ready')
		}

		// A new theme reads the colours again and redraws without animating.
		const retheme = () => draw(true)

		const resize = () => {
			if (!chart) return
			chart.resize()
			// A kind that shaped the chart to its size: build it again.
			if (sized) return draw(true)
			// A width change can change how many rows the legend wraps into: fit
			// just the grid margins again, without resolving every colour again.
			if (!unfitted) return
			const m = margins(unfitted), key = JSON.stringify(m)
			if (key !== fitted) (fitted = key), chart.setOption({ grid: m })
		}

		const dispose = () => {
			chart?.dispose()
			chart = null
			internalsOf(host).states.delete('ready')
		}

		// The keyboard reaches what the pointer does: ←/→ step through a series'
		// items and ↑/↓ from one series to the next, each shown with its tooltip,
		// and Enter clicks the item.
		action('key', ({ evt }) => {
			const model = chart?.getModel(), n = model?.getSeriesCount(), [ds, di] = KEYS[evt.key] ?? []
			if (!n || ds == null || (!ds && !di && at < 0)) return
			evt.preventDefault()
			const s = model.getSeriesByIndex((si = clamp(si + ds, n - 1))), data = s.getData(), count = data.count()
			if (!count) return
			at = clamp(at + di, count - 1)
			if (ds || di) chart.dispatchAction({ type: 'showTip', seriesIndex: si, dataIndex: data.getRawIndex(at) })
			else click(s.getDataParams(at))
		})
		action('blur', () => chart?.dispatchAction({ type: 'hideTip' }))

		// Loaded and drawn only once it is about to be seen.
		const io = new IntersectionObserver(([e]) => ((visible = e.isIntersecting), start()), { rootMargin: '200px' })
		io.observe(host)
		const ro = new ResizeObserver(resize)
		ro.observe(host)
		// A pick on <sb-theme-switch> arrives as sb-theme-change; the system
		// flipping under "auto" does not.
		const scheme = matchMedia('(prefers-color-scheme: dark)')
		scheme.addEventListener('change', retheme)
		addEventListener('sb-theme-change', retheme)
		// A kind the page defines later.
		const onKind = (name) => props.option?.kind === name && draw()
		reg.charts.add(onKind)
		observeProps(() => (chart ? draw() : start()), 'option')
		// The renderer and the month names are chosen when ECharts starts, so a new
		// one starts it again.
		observeProps(() => chart && (dispose(), start()), 'renderer', 'lang')

		cleanup(() => {
			alive = false
			io.disconnect()
			ro.disconnect()
			scheme.removeEventListener('change', retheme)
			removeEventListener('sb-theme-change', retheme)
			reg.charts.delete(onKind)
			dispose()
			plots.delete(host)
		})
	},
	render: ({ html }) => html`
		<div class="plot" part="plot" data-ref:plot data-on:keydown="@key()" data-on:blur="@blur()"></div>
		<div class="fallback"><slot></slot></div>
	`,
	onFirstRender: ({ host, refs: { plot } }) => {
		// With a server fallback in the slot, that is what screen readers get; the
		// drawing adds nothing for them. Without one, the drawing is what readers
		// get, keyboard users included.
		if (host.querySelector(':scope > *')) plot.setAttribute('aria-hidden', 'true')
		else plot.tabIndex = 0
		plots.get(host)(plot)
	},
})
