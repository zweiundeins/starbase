import { rocket } from 'datastar'

const probe = document.createElement('canvas').getContext('2d', { willReadFrequently: true })
const rgbCache = new Map()
const rgbOf = (css) => {
	if (rgbCache.has(css)) return rgbCache.get(css)
	probe.clearRect(0, 0, 1, 1)
	probe.fillStyle = '#000'
	probe.fillStyle = css
	probe.fillRect(0, 0, 1, 1)
	const [r, g, b] = probe.getImageData(0, 0, 1, 1).data
	rgbCache.set(css, [r, g, b])
	return [r, g, b]
}

const TONES = {
	brand: ['--sb-brand-light', '#B09AFF'],
	ok: ['--sb-ok', '#6EF59A'],
	warn: ['--sb-warn', '#F5C451'],
	danger: ['--sb-danger', '#F2777A'],
	accent: ['--sb-accent', '#65BFFF'],
}
// Per-instance buffer shared by setup and onFirstRender.
const buffers = new WeakMap()

const STEP = 3 // pixels between points
const H = 24

const styles = /* css */ `
:host {
	--_muted: var(--sb-text-muted, #7785A8);
	--_text: var(--sb-text-1, #F3F4FA);
	display: inline-flex;
	align-items: center;
	gap: 0.75rem;
	inline-size: var(--sb-sparkline-width, 12rem);
	max-inline-size: 100%;
	vertical-align: middle;
}
canvas { flex: 1; min-inline-size: 0; block-size: auto; image-rendering: pixelated; }
.value { color: var(--_text); font-variant-numeric: tabular-nums; font-weight: 700; font-size: 0.875rem; white-space: nowrap; }
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
	}),
	renderOnPropChange: ({ changes }) => 'showValue' in changes,
	setup: ({ $$, adoptStyles, defineHostProp, host, observeProps, props }) => {
		adoptStyles(host, styles)
		// The data buffer is instance state, not an attribute.
		let data = [...props.values].slice(-props.length)
		$$.latest = ''
		const label = () => {
			const last = data.at(-1)
			$$.latest = last === undefined ? '' : Number(last).toFixed(props.decimals) + props.unit
		}
		label()
		const buf = {
			pushed: false,
			get data() {
				return data
			},
			push(v) {
				if (!Number.isFinite(v)) return
				buf.pushed = true
				data.push(v)
				if (data.length > props.length) data = data.slice(-props.length)
				label()
				buf.onchange?.()
			},
		}
		buffers.set(host, buf)
		observeProps(() => {
			data = [...props.values].slice(-props.length)
			label()
			buf.onchange?.()
		}, 'values')
		observeProps(() => buf.push(props.value), 'value')
		observeProps(() => (label(), buf.onchange?.()), 'length', 'decimals', 'unit', 'tone', 'scale', 'min', 'max')
		defineHostProp('push', { value: (v) => buf.push(Number(v)) })
		defineHostProp('data', { get: () => [...data] })
	},
	render: ({ html, props: { showValue } }) => html`
		<canvas part="line" height="${H}" role="img"></canvas>
		${showValue ? html`<span class="value" part="value" data-text="$$latest"></span>` : null}
	`,
	onFirstRender: ({ cleanup, host, props }) => {
		const canvas = host.shadowRoot.querySelector('canvas')
		const ctx = canvas.getContext('2d')
		let raf = 0
		const paint = () => {
			raf = 0
			const { data, pushed } = buffers.get(host)
			// Static data fills the width; pushed data scrolls in from the right.
			const slots = pushed ? props.length : Math.max(2, Math.min(props.length, data.length))
			const w = (slots - 1) * STEP + 3
			if (canvas.width !== w) canvas.width = w
			canvas.style.aspectRatio = `${w} / ${H}`
			const img = ctx.createImageData(w, H)
			const d = img.data
			const [token, fallback] = TONES[props.tone]
			const c = rgbOf(getComputedStyle(host).getPropertyValue(token).trim() || fallback)
			let lo = props.min, hi = props.max
			if (props.scale === 'auto' && data.length) {
				lo = Math.min(...data)
				hi = Math.max(...data)
				if (hi - lo < 1e-9) (lo -= 1), (hi += 1)
			}
			const yOf = (v) => Math.round((H - 3) - ((Math.max(lo, Math.min(hi, v)) - lo) / (hi - lo || 1)) * (H - 4))
			const set = (x, y, a) => {
				if (x < 0 || y < 0 || x >= w || y >= H) return
				const i = (y * w + x) * 4
				if (d[i + 3] >= a) return
				d[i] = c[0], d[i + 1] = c[1], d[i + 2] = c[2], d[i + 3] = a
			}
			// Right-aligned: the newest point sits at the right edge.
			const x0 = w - 2 - (data.length - 1) * STEP
			let prev = null
			data.forEach((v, i) => {
				const x = x0 + i * STEP, y = yOf(v)
				for (let yy = y + 1; yy < H; yy++) for (let xx = x - STEP + 1; xx <= x; xx++) set(xx, yy, 46) // area
				if (prev !== null) {
					for (let xx = x - STEP + 1; xx <= x; xx++) set(xx, prev, 255) // step: flat…
					for (let yy = Math.min(prev, y); yy <= Math.max(prev, y); yy++) set(x, yy, 255) // …then vertical
				}
				prev = y
			})
			if (data.length) {
				const x = x0 + (data.length - 1) * STEP, y = yOf(data.at(-1))
				for (let yy = -1; yy <= 1; yy++) for (let xx = -1; xx <= 1; xx++) set(x + xx, y + yy, 255)
			}
			ctx.putImageData(img, 0, 0)
			const mn = data.length ? Math.min(...data) : 0, mx = data.length ? Math.max(...data) : 0
			const fmt = (v) => Number(v).toFixed(props.decimals) + props.unit
			canvas.setAttribute('aria-label', data.length ? `Trend of ${data.length} points: latest ${fmt(data.at(-1))}, low ${fmt(mn)}, high ${fmt(mx)}` : 'No data yet')
		}
		buffers.get(host).onchange = () => {
			if (!raf) raf = requestAnimationFrame(paint)
		}
		paint()
		cleanup(() => cancelAnimationFrame(raf))
	},
})
