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
// 'datastar' and its own files, but a page may import this module).

// Kinds are charts whose shape is code. A custom series draws through a
// renderItem *function*, which no server can send as JSON; so the page defines
// a builder under a name, and an option { kind: 'name', ... } is expanded by it
// (with the context below) before theming.
const kinds = new Map()
export const defineChartKind = (name, build) => void kinds.set(name, build)

// The numbers ECharts writes itself – value-axis labels, tooltip rows – use
// Intl.NumberFormat in the element's language. A page whose server formats
// numbers its own way sets the same formatter here, so a chart never disagrees
// with the figures around it.
let numberFormat = null
export const setNumberFormat = (fn) => void (numberFormat = fn)

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

// deepMerge lays the page's option over the themed defaults: objects merge,
// everything else (arrays included) replaces.
const deepMerge = (base, over) => {
	const out = { ...base }
	for (const [k, v] of Object.entries(over)) out[k] = isObj(v) && isObj(out[k]) ? deepMerge(out[k], v) : v
	return out
}

// Axes and a grid belong only to charts that plot on one. Giving them to a pie
// or a gauge makes ECharts draw a phantom coordinate system.
const CARTESIAN = new Set(['bar', 'line', 'scatter', 'effectScatter', 'candlestick', 'boxplot', 'custom', 'pictorialBar', 'heatmap'])
const plotsOnGrid = (o) => {
	const s = Array.isArray(o.series) ? o.series : o.series ? [o.series] : []
	return !s.length || s.some((x) => CARTESIAN.has(x?.type))
}

// Month and weekday names in the element's language, from Intl, registered once
// per language. ECharts ships only a handful of locales of its own.
const locales = new Set()
const localeFor = (ec, lang) => {
	const code = 'sb-' + lang
	if (locales.has(code)) return code
	const fmt = (opt) => (d) => new Intl.DateTimeFormat(lang, { ...opt, timeZone: 'UTC' }).format(d)
	const months = Array.from({ length: 12 }, (_, m) => new Date(Date.UTC(2024, m, 1)))
	const days = Array.from({ length: 7 }, (_, d) => new Date(Date.UTC(2024, 0, 7 + d))) // a Sunday first
	ec.registerLocale(code, {
		time: { month: months.map(fmt({ month: 'long' })), monthAbbr: months.map(fmt({ month: 'short' })), dayOfWeek: days.map(fmt({ weekday: 'long' })), dayOfWeekAbbr: days.map(fmt({ weekday: 'short' })) },
		legend: { selector: { all: 'All', inverse: 'Inv' } },
		toolbox: {},
		series: { typeNames: {} },
		aria: {},
	})
	locales.add(code)
	return code
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

const styles = /* css */ `
:host {
	--_text: var(--sb-text-2, #AEBBDD);
	--_strong: var(--sb-text-1, #F3F4FA);
	--_muted: var(--sb-text-muted, #7785A8);
	--_line: var(--sb-border, #283552);
	--_grid: var(--sb-border-subtle, #1B2640);
	--_tip: var(--sb-surface-overlay, #141D32);
	--_font: var(--sb-font-body, system-ui, sans-serif);
	--_numbers: var(--sb-font-ui, var(--sb-font-body, system-ui, sans-serif));
	${PALETTE.map(([own, semantic, fallback], i) => `--_c${i + 1}: var(${own}, var(${semantic}, ${fallback}));`).join('\n\t')}
	display: block;
	position: relative;
	/* A chart fills the width it is given, also as a flex or grid item, where a
	   block whose only content is absolutely positioned would shrink to nothing. */
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

// One controller per element: setup creates it, onFirstRender hands it the plot.
const charts = new WeakMap()

// One ElementInternals per element: attachInternals() works once, and setup
// reruns when the element is re-attached. The ready state lives there rather
// than in an attribute, which the next server morph would strip.
const internals = new WeakMap()
const internalsOf = (host) => internals.get(host) ?? internals.set(host, host.attachInternals()).get(host)

class Chart {
	constructor(host, props, emit) {
		this.host = host
		this.props = props
		this.emit = emit
		this.plot = null // attached in onFirstRender
		this.visible = false
		this.ec = null
		this.chart = null
		this.legendGrid = ''
	}

	get lang() {
		return this.props.lang || this.host.closest('[lang]')?.lang || navigator.language
	}

	// A token's value as the page resolves it, in the element's own scope (so
	// properties set on the host apply). Colours go through a real `color`
	// property: getPropertyValue returns light-dark() and var() chains verbatim,
	// which a canvas cannot paint.
	css(token) {
		return getComputedStyle(this.host).getPropertyValue(token).trim()
	}
	color(value) {
		const probe = document.createElement('span')
		probe.className = 'probe'
		probe.style.color = value.startsWith('--') ? `var(${value})` : value
		this.host.shadowRoot.append(probe)
		const c = getComputedStyle(probe).color
		probe.remove()
		return c
	}

	formatNumber(n) {
		if (typeof n !== 'number' || !Number.isFinite(n)) return String(n)
		if (numberFormat) return numberFormat(n, this.lang)
		return new Intl.NumberFormat(this.lang, { maximumFractionDigits: 20 }).format(n)
	}

	// The context a kind builds with: everything it needs to draw in the page's
	// theme and language, without reaching into the element.
	context() {
		return {
			echarts: this.ec,
			color: (v) => this.color(v),
			css: (t) => this.css(t),
			formatNumber: (n) => this.formatNumber(n),
			lang: this.lang,
			width: this.plot.clientWidth,
			height: this.plot.clientHeight,
		}
	}

	build(input) {
		let option = input
		if (option.kind) {
			const kind = kinds.get(option.kind)
			if (!kind) {
				console.error(`<sb-echarts>: no chart kind "${option.kind}"; register it with defineChartKind()`)
				return null
			}
			option = kind(option, this.context())
		}
		const text = this.color('--_text'), muted = this.color('--_muted'), line = this.color('--_line'), grid = this.color('--_grid')
		const font = this.css('--_font'), numbers = this.css('--_numbers')
		const reduced = matchMedia('(prefers-reduced-motion: reduce)').matches
		const axis = {
			axisLine: { lineStyle: { color: line } },
			axisTick: { lineStyle: { color: line } },
			axisLabel: { color: muted, fontFamily: font },
			splitLine: { lineStyle: { color: grid } },
			nameTextStyle: { color: muted, fontFamily: font },
		}
		const base = {
			color: PALETTE.map((_, i) => this.color(`--_c${i + 1}`)),
			backgroundColor: 'transparent',
			animation: !reduced,
			animationDuration: 700,
			animationEasing: 'cubicOut',
			animationDurationUpdate: 450,
			animationEasingUpdate: 'cubicInOut',
			textStyle: { color: text, fontFamily: font },
			tooltip: {
				backgroundColor: this.color('--_tip'),
				borderColor: line,
				textStyle: { color: this.color('--_strong'), fontFamily: numbers, fontSize: 12 },
			},
			aria: { enabled: !this.host.querySelector(':scope > *') }, // a server table says it better
		}
		if (plotsOnGrid(option)) {
			base.xAxis = axis
			base.yAxis = { ...axis, axisLabel: { ...axis.axisLabel, fontFamily: numbers } }
			base.grid = { left: '3%', right: '4%', top: '10%', bottom: '12%' }
		}
		const merged = deepMerge(base, option)
		// Style a legend only where the option has one: in ECharts the mere presence
		// of a legend object switches one on.
		if (merged.legend) merged.legend = [merged.legend].flat().map((l) => deepMerge({ textStyle: { color: muted, fontFamily: font } }, l))
		// A pie's or funnel's outside labels get ECharts' own dark text outline,
		// which reads as a smudge on a dark ground: plain text in the theme's colour.
		for (const s of [merged.series].flat()) if (isObj(s) && (s.type === 'pie' || s.type === 'funnel')) s.label = { color: text, textBorderWidth: 0, ...s.label }
		// deepMerge replaced array axes wholesale; give each its themed defaults.
		for (const k of ['xAxis', 'yAxis']) if (Array.isArray(merged[k])) merged[k] = merged[k].map((a) => deepMerge(base[k] ?? axis, a))
		this.fitLegend(merged, font)
		this.localizeNumbers(merged)
		return this.resolveVars(merged)
	}

	// Strings that are exactly var(--token) become the colour they resolve to, so
	// a server can colour a series with the page's own tokens.
	resolveVars(v) {
		if (typeof v === 'string') return /^var\(--[\w-]+\)$/.test(v) ? this.color(v.slice(4, -1)) : v
		if (Array.isArray(v)) return v.map((x) => this.resolveVars(x))
		if (isObj(v)) return Object.fromEntries(Object.entries(v).map(([k, x]) => [k, this.resolveVars(x)]))
		return v
	}

	// Value axes and axis tooltips get the element's number format, where the
	// option has not set a formatter of its own (a unit-carrying axis, a custom
	// tooltip).
	localizeNumbers(o) {
		const fmt = (n) => this.formatNumber(n)
		const axis = (a, type) => {
			if (!isObj(a) || (a.type ?? type) !== 'value') return
			a.axisLabel ??= {}
			a.axisLabel.formatter ??= fmt
		}
		for (const [k, type] of [['xAxis', 'category'], ['yAxis', 'value']]) [o[k]].flat().forEach((a) => axis(a, type))
		;[o.tooltip].flat().forEach((t) => isObj(t) && (t.valueFormatter ??= fmt))
	}

	// ECharts reserves no room for its legend: on a narrow box a horizontal legend
	// wraps to several rows and runs into the axis names or tick labels. Measure
	// how many rows it needs at the current width and, only when it wraps, push
	// the grid's margin on that side out to clear it.
	fitLegend(o, font) {
		const legend = [o.legend].flat()[0]
		const labels = legend?.data
		if (!isObj(o.grid) || !Array.isArray(labels) || !labels.length || (legend.orient ?? 'horizontal') !== 'horizontal') return
		const near = (v) => v === 0 || v === '0' || (typeof v === 'string' && v.endsWith('%') && parseFloat(v) < 25) || (typeof v === 'number' && v < 60)
		const top = legend.top === 'top' || near(legend.top)
		const bottom = legend.bottom === 'bottom' || near(legend.bottom)
		const { clientWidth: w, clientHeight: h } = this.plot
		if ((!top && !bottom) || !w || !h) return
		const ctx = (this.measure ??= document.createElement('canvas').getContext('2d'))
		ctx.font = `12px ${font}`
		const gap = typeof legend.itemGap === 'number' ? legend.itemGap : 10
		const icon = typeof legend.itemWidth === 'number' ? legend.itemWidth : 25
		let rows = 1, x = 0
		for (const label of labels) {
			const lw = icon + 5 + ctx.measureText(String(isObj(label) ? label.name : label)).width + gap
			if (x > 0 && x + lw > w * 0.96) (rows++, (x = lw))
			else x += lw
		}
		if (rows < 2) return
		const px = (v) => (typeof v === 'number' ? v : typeof v === 'string' && v.endsWith('%') ? (h * parseFloat(v)) / 100 : parseFloat(v) || 0)
		const block = rows * 26 + 6
		if (top) {
			const named = [o.yAxis].flat().some((a) => isObj(a) && a.name)
			o.grid.top = Math.max(px(o.grid.top), Math.round((typeof legend.top === 'number' ? legend.top : 8) + block + (named ? 28 : 0)))
		}
		if (bottom) o.grid.bottom = Math.max(px(o.grid.bottom), Math.round((typeof legend.bottom === 'number' ? legend.bottom : 8) + block + 22))
	}

	async start() {
		if (this.chart || this.starting || !this.plot || !this.visible || !this.props.option) return
		this.starting = true
		this.ec = await load()
		this.starting = false
		if (!this.host.isConnected || this.chart) return
		this.chart = this.ec.init(this.plot, null, { renderer: this.props.renderer, locale: localeFor(this.ec, this.lang) })
		this.chart.on('click', (p) => this.emit('sb-chart-click', { seriesName: p.seriesName, seriesIndex: p.seriesIndex, name: p.name, value: p.value, dataIndex: p.dataIndex }))
		this.draw()
		internalsOf(this.host).states.add('ready')
	}

	// Every new option replaces the last one (notMerge), and ECharts animates the
	// difference, so a server push reads as the chart moving to its new state.
	draw({ quiet = false } = {}) {
		if (!this.chart) return
		if (!this.props.option) return void this.chart.clear()
		const option = this.build(this.props.option)
		if (!option) return
		if (quiet) option.animationDurationUpdate = 0
		this.chart.setOption(option, true)
		this.legendGrid = JSON.stringify([option.grid?.top, option.grid?.bottom])
	}

	// A new theme only needs the colours read again: nothing is re-created.
	retheme() {
		this.draw({ quiet: true })
	}

	resize() {
		if (!this.chart) return
		this.chart.resize()
		// A width change can change how many rows the legend wraps into: re-fit
		// just the grid margins, without resolving every colour again.
		const o = this.props.option
		if (!o?.legend) return
		const probe = { legend: o.legend, yAxis: o.yAxis, grid: { top: '10%', bottom: '12%', ...(isObj(o.grid) ? o.grid : {}) } }
		this.fitLegend(probe, this.css('--_font'))
		const key = JSON.stringify([probe.grid.top, probe.grid.bottom])
		if (key !== this.legendGrid) (this.legendGrid = key), this.chart.setOption({ grid: { top: probe.grid.top, bottom: probe.grid.bottom } })
	}

	dispose() {
		this.chart?.dispose()
		this.chart = null
		internalsOf(this.host).states.delete('ready')
	}
}

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
		const chart = new Chart(host, props, emit)
		charts.set(host, chart)
		action('retheme', () => chart.retheme())

		// Loaded and drawn only once it is about to be seen.
		const io = new IntersectionObserver(([e]) => ((chart.visible = e.isIntersecting), chart.start()), { rootMargin: '200px' })
		io.observe(host)
		const ro = new ResizeObserver(() => chart.resize())
		ro.observe(host)
		// A pick on <sb-theme-switch> arrives as sb-theme-change (see the template);
		// the system flipping under "auto" does not.
		const scheme = matchMedia('(prefers-color-scheme: dark)')
		const retheme = () => chart.retheme()
		scheme.addEventListener('change', retheme)
		observeProps(() => (chart.chart ? chart.draw() : chart.start()), 'option')
		// The renderer is chosen when ECharts starts, so a new one starts it again.
		observeProps(() => chart.chart && (chart.dispose(), chart.start()), 'renderer')

		cleanup(() => {
			io.disconnect()
			ro.disconnect()
			scheme.removeEventListener('change', retheme)
			chart.dispose()
			charts.delete(host)
		})
	},
	render: ({ html }) => html`
		<div class="plot" part="plot" data-ref:plot data-on:sb-theme-change__window="@retheme()"></div>
		<div class="fallback"><slot></slot></div>
	`,
	onFirstRender: ({ host, refs }) => {
		const chart = charts.get(host)
		chart.plot = refs.plot
		// With a server fallback in the slot, that is what screen readers get; the
		// drawing adds nothing for them.
		if (host.querySelector(':scope > *')) refs.plot.setAttribute('aria-hidden', 'true')
		chart.start()
	},
})
