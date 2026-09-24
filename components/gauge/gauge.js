import { rocket } from 'datastar'

const probe = document.createElement('canvas').getContext('2d', { willReadFrequently: true })
const rgbCache = new Map()
// Per-instance repaint, so setup's actions can reach the canvas code.
const kicks = new WeakMap()
// css is a computed colour, so always one the canvas can parse. Returns
// [r, g, b, a]; the dial uses r, g and b.
const rgbOf = (css) => {
	if (!rgbCache.has(css)) {
		probe.clearRect(0, 0, 1, 1)
		probe.fillStyle = css
		probe.fillRect(0, 0, 1, 1)
		rgbCache.set(css, probe.getImageData(0, 0, 1, 1).data)
	}
	return rgbCache.get(css)
}

// Dial raster size; scaled up with crisp pixels.
const W = 80, H = 44, CX = 40, CY = 40, R_OUT = 37, R_IN = 30

// The <i> holds the dial's colours, which paint reads: CSS resolves the
// tokens (fallbacks, light-dark()), and a change to any of them, from any
// cause (a theme switch, a theme scoped to a container, the system's light or
// dark mode), runs its transition, whose end repaints the dial (at
// transitionrun a browser may still compute the old colour). It is not a part,
// so a page's ::part() rules can't reach it, and never forced, like the canvas.
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
	/* The readout scales with the gauge: cqi is 1% of its width. */
	container-type: inline-size;
}
canvas { inline-size: 100%; aspect-ratio: ${W} / ${H}; image-rendering: pixelated; }
i {
	position: absolute;
	forced-color-adjust: none;
	color: var(--sb-ok, #6EF59A);
	border-color: var(--sb-warn, #F5C451) var(--sb-danger, #F2777A) var(--sb-border, #283552) var(--_text);
	outline-color: var(--sb-brand, #8C6BFF);
	transition: 1ms;
}
.readout { display: grid; justify-items: center; line-height: 1.2; }
/* 1.5rem and 0.75rem at the default 12rem, in proportion otherwise, never
   smaller than stays readable. The value's font is the page's unless a theme
   sets a display font. */
.value { color: var(--_text); font-family: var(--sb-font-display, inherit); font-size: max(0.875rem, 12.5cqi); font-weight: 700; font-variant-numeric: tabular-nums; }
.label { color: var(--_muted); font-size: max(0.625rem, 6.25cqi); letter-spacing: 0.08em; text-transform: uppercase; }
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
	// The template reads label, min and max; the readout follows value, decimals
	// and unit through $$.
	renderOnPropChange: ({ changes }) => 'label' in changes || 'min' in changes || 'max' in changes,
	setup: ({ $$, action, adoptStyles, host, observeProps, props }) => {
		adoptStyles(host, styles)
		// Colours are read at paint time, so a new theme only needs a repaint.
		action('repaint', () => kicks.get(host)?.())
		const sync = () => {
			// aria-valuenow stays in range, like a <meter>; the text is the reading.
			$$.now = Math.min(props.max, Math.max(props.min, props.value))
			$$.shown = Number(props.value).toFixed(props.decimals) + props.unit
		}
		sync()
		observeProps(sync)
	},
	render: ({ html, props: { label, min, max } }) => html`
		<canvas part="dial" width="${W}" height="${H}" aria-hidden="true"></canvas>
		<i data-on:transitionend="@repaint()"></i>
		<div class="readout" role="meter" aria-valuemin="${min}" aria-valuemax="${max}" aria-label="${label || 'Gauge'}"
			data-attr:aria-valuenow="$$now" data-attr:aria-valuetext="$$shown">
			<span class="value" part="value" data-text="$$shown"></span>
			${label ? html`<span class="label" part="label">${label}</span>` : null}
		</div>
	`,
	onFirstRender: ({ cleanup, host, observeProps, props }) => {
		const canvas = host.shadowRoot.querySelector('canvas')
		const colours = host.shadowRoot.querySelector('i')
		const ctx = canvas.getContext('2d')
		const img = ctx.createImageData(W, H)
		const reduced = matchMedia('(prefers-reduced-motion: reduce)')
		const frac = (v) => Math.max(0, Math.min(1, (v - props.min) / (props.max - props.min || 1)))
		let pos = frac(props.value), vel = 0, raf = 0, last = 0, seen = false

		const tone = (v) => {
			const { warn, danger } = props
			const bad = danger >= warn ? [v >= danger, v >= warn] : [v <= danger, v <= warn]
			return bad[0] ? 'danger' : bad[1] ? 'warn' : 'ok'
		}
		const paint = () => {
			const cs = getComputedStyle(colours)
			const C = {
				ok: rgbOf(cs.color),
				warn: rgbOf(cs.borderTopColor),
				danger: rgbOf(cs.borderRightColor),
				track: rgbOf(cs.borderBottomColor),
				needle: rgbOf(cs.borderLeftColor),
				hub: rgbOf(cs.outlineColor),
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
					// 0 at left, 1 at right; the row just below the centre mirrors the
					// one above it, so the arc's ends light up like the rest.
					const f = 1 - Math.atan2(Math.abs(dy), dx) / Math.PI
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
			paint()
			if (Math.abs(pos - target) > 0.0005 || Math.abs(vel) > 0.0005) kick()
			last = raf ? t : 0 // stopped (settled or offscreen): the next run starts afresh
		}
		// Frames run only while the gauge is on screen; scrolling in catches up.
		const kick = () => {
			if (seen && !raf) raf = requestAnimationFrame(tick)
		}
		observeProps(kick)
		kicks.set(host, kick)
		// The last entry is the current state: one callback can hold an enter and a leave.
		const io = new IntersectionObserver((es) => {
			seen = es.at(-1).isIntersecting
			kick()
		})
		io.observe(host)
		paint()
		cleanup(() => {
			cancelAnimationFrame(raf)
			io.disconnect()
		})
	},
})
