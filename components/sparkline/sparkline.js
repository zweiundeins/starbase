import { rocket, startPeeking, stopPeeking } from 'datastar'

// The bitmap is H pixels high and as wide as the box at square pixels, so
// the height is a token and the line keeps its pixel look at any `length`.
const H = 24
// Pushed points survive a move in the DOM (setup reruns on every connect).
const kept = new WeakMap()

const styles = /* css */ `
:host {
	--_c: var(--sb-brand-light, #B09AFF);
	display: inline-flex;
	align-items: center;
	gap: 0.75rem;
	inline-size: var(--sb-sparkline-width, 12rem);
	max-inline-size: 100%;
	vertical-align: middle;
}
:host([hidden]) { display: none; }
:host([tone=ok]) { --_c: var(--sb-ok, #6EF59A); }
:host([tone=warn]) { --_c: var(--sb-warn, #F5C451); }
:host([tone=danger]) { --_c: var(--sb-danger, #F2777A); }
:host([tone=accent]) { --_c: var(--sb-accent, #65BFFF); }
/* The line paints in the canvas's colour. The transition turns any change of
   it (theme switch, class, media query, tone) into a transitionend: a repaint. */
canvas { flex: 1; min-inline-size: 0; block-size: var(--sb-sparkline-height, 2.25rem); image-rendering: pixelated; color: var(--_c); transition: color 1ms; }
.value { color: var(--sb-text-1, #F3F4FA); font-variant-numeric: tabular-nums; font-weight: 700; font-size: 0.875rem; white-space: nowrap; }
`

rocket('sb-sparkline', {
	props: ({ array, bool, number, oneOf, string }) => ({
		values: array(number).docs({ description: 'Initial data points, as a JSON array.' }),
		value: number.docs({ description: 'Push mode: every change appends a point.' }),
		length: number.clamp(4, 200).default(40).docs({ description: 'Points kept (older ones scroll off).' }),
		scale: oneOf('auto', 'fixed').default('auto').docs({ description: 'auto fits the data; fixed uses min and max.' }),
		min: number.docs({ description: 'Bottom of the fixed scale.' }),
		max: number.default(100).docs({ description: 'Top of the fixed scale.' }),
		tone: oneOf('brand', 'ok', 'warn', 'danger', 'accent').default('brand').docs({ description: 'Line colour (theme token).' }),
		showValue: bool.docs({ description: 'Show the latest value next to the line.' }),
		unit: string.docs({ description: 'Unit after the shown value.' }),
		decimals: number.clamp(0, 4).docs({ description: 'Decimals of the shown value.' }),
		label: string.trim.docs({ description: 'What the chart shows, named first in its accessible label.' }),
	}),
	renderOnPropChange: false,
	setup: ({ $$, action, adoptStyles, cleanup, defineHostProp, host, observeProps, props }) => {
		adoptStyles(host, styles)
		const fmt = (v) => Number(v).toFixed(props.decimals) + props.unit
		const read = () => parseFloat(host.getAttribute('value'))
		let raf = 0
		const paint = () => {
			raf = 0
			const c = host.shadowRoot.querySelector('canvas'), ctx = c.getContext('2d'), n = data.length
			const w = (c.width = Math.round((H * c.clientWidth) / c.clientHeight))
			// Colours are read at paint time.
			ctx.fillStyle = getComputedStyle(c).color
			let lo = props.min, hi = props.max, px = 0, py
			if (props.scale == 'auto' && n) {
				lo = Math.min(...data)
				hi = Math.max(...data)
				if (hi - lo < 1e-9) lo--, hi++
			}
			data.forEach((v, i) => {
				// Fewer than `length` points spread across the width; once full,
				// the line scrolls with the newest point at the right.
				const x = 1 + Math.round((i * (w - 3)) / Math.max(1, n - 1))
				const y = Math.round(H - 3 - ((Math.max(lo, Math.min(hi, v)) - lo) / (hi - lo || 1)) * (H - 4))
				py ??= y
				ctx.globalAlpha = 0.18
				ctx.fillRect(px + 1, py + 1, x - px, H) // area under the step
				ctx.globalAlpha = 1
				ctx.fillRect(px + 1, py, x - px, 1) // step: flat…
				ctx.fillRect(x, Math.min(py, y), 1, Math.abs(py - y) + 1) // …then vertical
				px = x
				py = y
			})
			if (n) ctx.fillRect(px - 1, py - 1, 3, 3)
		}
		const draw = () => {
			// Callers can run inside an effect (data-attr sets a prop, data-effect
			// calls push): writing $$ there must not subscribe it.
			startPeeking()
			try {
				data.splice(0, data.length - props.length)
				const n = data.length, last = n && fmt(data.at(-1))
				$$.show = props.showValue
				$$.latest = last || ''
				$$.label = (props.label && props.label + ': ') + (n ? `Trend of ${n} points: latest ${last}, low ${fmt(Math.min(...data))}, high ${fmt(Math.max(...data))}` : 'No data yet')
				raf ||= requestAnimationFrame(paint)
			} finally {
				stopPeeking()
			}
		}
		const push = (v) => Number.isFinite(v) && (data.push(v), draw())
		let served = read()
		let data = kept.get(host)
		if (!data) {
			kept.set(host, (data = [...props.values]))
			// The starting value is the latest point, after the history in `values`.
			if (served !== data.at(-1)) push(served)
		}
		// Push mode watches the attribute: a removed or empty one adds nothing,
		// and neither does the same value put back (a morph strips and restores it).
		const watch = new MutationObserver(() => {
			const v = read()
			if (v === v && v !== served) push((served = v))
		})
		watch.observe(host, { attributeFilter: ['value'] })
		const ro = new ResizeObserver(draw)
		queueMicrotask(() => ro.observe(host.shadowRoot.querySelector('canvas')))
		cleanup(() => (cancelAnimationFrame(raf), watch.disconnect(), ro.disconnect()))
		observeProps((_, changes) => {
			if ('values' in changes) kept.set(host, (data = [...props.values]))
			draw()
		})
		action('draw', draw)
		defineHostProp('push', { value: (v) => push(Number(v)) })
		defineHostProp('data', { get: () => [...data] })
		draw()
	},
	render: ({ html }) => html`
		<canvas part="line" height="${H}" role="img" data-attr:aria-label="$$label" data-on:transitionend="@draw()"></canvas>
		<span class="value" part="value" aria-hidden="true" data-show="$$show" data-text="$$latest"></span>
	`,
})
