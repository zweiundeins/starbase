import { rocket } from 'datastar'

const probe = document.createElement('canvas').getContext('2d', { willReadFrequently: true })
const rgbCache = new Map()
// Per-instance repaint, so setup's actions can reach the canvas code.
const kicks = new WeakMap()
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

// Dial raster size; scaled up with crisp pixels.
const W = 80, H = 44, CX = 40, CY = 40, R_OUT = 37, R_IN = 30

const styles = /* css */ `
:host {
	--_text: var(--sb-text-1, #F3F4FA);
	--_muted: var(--sb-text-2, #AEBBDD);
	display: inline-grid;
	justify-items: center;
	gap: 0.25rem;
	inline-size: var(--sb-gauge-size, 12rem);
	max-inline-size: 100%;
	vertical-align: middle;
}
canvas { inline-size: 100%; aspect-ratio: ${W} / ${H}; image-rendering: pixelated; }
.readout { display: grid; justify-items: center; line-height: 1.2; }
.value { color: var(--_text); font-family: var(--sb-font-display, ui-monospace, monospace); font-size: 1.5rem; font-weight: 700; font-variant-numeric: tabular-nums; }
.label { color: var(--_muted); font-size: 0.75rem; letter-spacing: 0.08em; text-transform: uppercase; }
`

rocket('sb-gauge', {
	props: ({ number, string }) => ({
		value: number.docs({ description: 'Current value; the needle eases toward it.' }),
		min: number.docs({ description: 'Minimum.' }),
		max: number.default(100).docs({ description: 'Maximum.' }),
		warn: number.default(70).docs({ description: 'Warning threshold. Below danger means low values are bad (like fuel).' }),
		danger: number.default(90).docs({ description: 'Danger threshold.' }),
		label: string.trim.docs({ description: 'Caption below the value.' }),
		unit: string.docs({ description: 'Unit after the value, e.g. "%" or " km/s".' }),
		decimals: number.clamp(0, 4).docs({ description: 'Decimals shown in the readout.' }),
	}),
	renderOnPropChange: ({ changes }) => 'label' in changes || 'unit' in changes,
	setup: ({ $$, action, adoptStyles, host, observeProps, props }) => {
		adoptStyles(host, styles)
		// Colours are read at paint time, so a new theme only needs a repaint.
		action('repaint', () => kicks.get(host)?.())
		const sync = () => {
			$$.now = props.value
			$$.shown = Number(props.value).toFixed(props.decimals) + props.unit
		}
		sync()
		observeProps(sync, 'value', 'decimals', 'unit')
	},
	render: ({ html, props: { label, min, max } }) => html`
		<canvas part="dial" width="${W}" height="${H}" aria-hidden="true" data-on:sb-theme-change__window="@repaint()"></canvas>
		<div class="readout" role="meter" aria-valuemin="${min}" aria-valuemax="${max}" aria-label="${label || 'Gauge'}"
			data-attr:aria-valuenow="$$now" data-attr:aria-valuetext="$$shown">
			<span class="value" part="value" data-text="$$shown"></span>
			${label ? html`<span class="label" part="label">${label}</span>` : null}
		</div>
	`,
	onFirstRender: ({ cleanup, host, observeProps, props }) => {
		const canvas = host.shadowRoot.querySelector('canvas')
		const ctx = canvas.getContext('2d')
		const img = ctx.createImageData(W, H)
		const reduced = matchMedia('(prefers-reduced-motion: reduce)')
		const frac = (v) => Math.max(0, Math.min(1, (v - props.min) / (props.max - props.min || 1)))
		let pos = frac(props.value), vel = 0, raf = 0, last = 0

		const tone = (v) => {
			const { warn, danger } = props
			const bad = danger >= warn ? [v >= danger, v >= warn] : [v <= danger, v <= warn]
			return bad[0] ? 'danger' : bad[1] ? 'warn' : 'ok'
		}
		const paint = () => {
			const cs = getComputedStyle(host)
			const col = (t, f) => rgbOf(cs.getPropertyValue(t).trim() || f)
			const C = {
				ok: col('--sb-ok', '#6EF59A'),
				warn: col('--sb-warn', '#F5C451'),
				danger: col('--sb-danger', '#F2777A'),
				track: col('--sb-border', '#283552'),
				needle: col('--sb-text-1', '#F3F4FA'),
				hub: col('--sb-brand', '#8C6BFF'),
			}
			const d = img.data
			d.fill(0)
			const set = (x, y, c, a = 255) => {
				if (x < 0 || y < 0 || x >= W || y >= H) return
				const i = (y * W + x) * 4
				d[i] = c[0], d[i + 1] = c[1], d[i + 2] = c[2], d[i + 3] = a
			}
			for (let y = 0; y < H; y++) {
				for (let x = 0; x < W; x++) {
					const dx = x + 0.5 - CX, dy = CY - (y + 0.5)
					const r = Math.hypot(dx, dy)
					if (dy < -0.5 || r < R_IN || r > R_OUT) continue
					const f = 1 - Math.atan2(dy, dx) / Math.PI // 0 at left, 1 at right
					const v = props.min + f * (props.max - props.min)
					// Zone colours are dim ahead of the needle and bright behind it.
					const lit = f <= pos
					const tick = Math.abs(f * 10 - Math.round(f * 10)) < 0.045 && r > R_OUT - 3
					if (tick) set(x, y, C.needle, 200)
					else set(x, y, lit ? C[tone(v)] : C.track, lit ? 255 : 150)
				}
			}
			// Needle: a stepped line from the hub.
			const a = Math.PI * (1 - pos)
			for (let i = 4; i <= R_IN - 2; i++) set(Math.round(CX + Math.cos(a) * i - 0.5), Math.round(CY - Math.sin(a) * i - 0.5), C.needle)
			for (let y = -2; y <= 1; y++) for (let x = -2; x <= 1; x++) set(CX + x, CY + y - 1, C.hub)
			ctx.putImageData(img, 0, 0)
		}
		// Slightly under-damped spring: the needle overshoots a hair, like hardware.
		const tick = (t) => {
			raf = 0
			const target = frac(props.value)
			if (reduced.matches) {
				pos = target
				vel = 0
			} else {
				const dt = last ? Math.min(0.05, (t - last) / 1000) : 1 / 60
				const k = 90, c = 2 * Math.sqrt(k) * 0.72
				vel += (-(pos - target) * k - vel * c) * dt
				pos += vel * dt
			}
			last = t
			paint()
			if (Math.abs(pos - target) > 0.0005 || Math.abs(vel) > 0.0005) raf = requestAnimationFrame(tick)
			else last = 0
		}
		const kick = () => {
			if (!raf) raf = requestAnimationFrame(tick)
		}
		observeProps(kick)
		kicks.set(host, kick)
		// Theme tokens are re-read on every paint; repaint when scrolled in, and when
		// the system flips under "auto" (a pick on <sb-theme-switch> arrives as
		// sb-theme-change, see the template).
		const io = new IntersectionObserver(([e]) => e.isIntersecting && kick())
		io.observe(host)
		const scheme = matchMedia('(prefers-color-scheme: dark)')
		scheme.addEventListener('change', kick)
		paint()
		cleanup(() => {
			cancelAnimationFrame(raf)
			io.disconnect()
			scheme.removeEventListener('change', kick)
		})
	},
})
