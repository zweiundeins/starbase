import { rocket } from 'datastar'

// Each CSS pixel block is PX device-independent pixels: chunky on purpose.
const PX = 3

// Resolve any CSS colour (tokens, oklch, …) to [r, g, b].
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

const TINTS = {
	white: ['--sb-text-1', '#F3F4FA'],
	violet: ['--sb-brand-light', '#B09AFF'],
	cyan: ['--sb-accent', '#65BFFF'],
	green: ['--sb-datastar', '#6EF59A'],
}

const styles = /* css */ `
:host {
	--_bg: var(--sb-surface-inset, #0B1224);
	--_radius: var(--sb-radius, 8px);
	display: block;
	inline-size: 100%;
	block-size: var(--sb-starfield-height, 12rem);
	border-radius: var(--_radius);
	overflow: hidden;
	background: var(--_bg);
	contain: strict;
}
canvas { display: block; inline-size: 100%; block-size: 100%; image-rendering: pixelated; }
`

rocket('sb-starfield', {
	props: ({ bool, number, oneOf }) => ({
		speed: number.clamp(0, 100).default(20).docs({ description: 'Flight speed.' }),
		density: number.clamp(20, 1500).default(300).docs({ description: 'Number of stars.' }),
		tint: oneOf('white', 'violet', 'cyan', 'green').default('white').docs({ description: 'Star colour (from theme tokens).' }),
		warp: bool.docs({ description: 'Draw streaks instead of dots.' }),
	}),
	renderOnPropChange: false,
	setup: ({ adoptStyles, host }) => adoptStyles(host, styles),
	render: ({ html }) => html`<canvas part="canvas" role="img" aria-label="Animated starfield"></canvas>`,
	onFirstRender: ({ cleanup, host, observeProps, props }) => {
		const canvas = host.shadowRoot.querySelector('canvas')
		const ctx = canvas.getContext('2d')
		const reduced = matchMedia('(prefers-reduced-motion: reduce)')
		let w = 1, h = 1, img = ctx.createImageData(1, 1)
		let stars = []
		let raf = 0, last = 0, visible = true

		const spawn = (s = {}) => {
			s.x = (Math.random() * 2 - 1) * 1.2
			s.y = (Math.random() * 2 - 1) * 1.2
			s.z = Math.random() * 0.95 + 0.05
			s.pz = s.z
			return s
		}
		const populate = () => {
			const n = Math.round(props.density)
			while (stars.length < n) stars.push(spawn())
			stars.length = n
		}
		const resize = () => {
			w = Math.max(1, Math.floor(host.clientWidth / PX))
			h = Math.max(1, Math.floor(host.clientHeight / PX))
			canvas.width = w
			canvas.height = h
			img = ctx.createImageData(w, h)
		}
		const plot = (x, y, rgb, a) => {
			x |= 0
			y |= 0
			if (x < 0 || y < 0 || x >= w || y >= h) return
			const i = (y * w + x) * 4
			if (img.data[i + 3] >= a) return
			img.data[i] = rgb[0]
			img.data[i + 1] = rgb[1]
			img.data[i + 2] = rgb[2]
			img.data[i + 3] = a
		}
		const line = (x0, y0, x1, y1, rgb, a) => {
			const n = Math.max(Math.abs(x1 - x0), Math.abs(y1 - y0), 1)
			for (let i = 0; i <= n; i++) plot(x0 + ((x1 - x0) * i) / n, y0 + ((y1 - y0) * i) / n, rgb, Math.round(a * (0.35 + (0.65 * i) / n)))
		}
		const paint = (dt) => {
			const [token, fallback] = TINTS[props.tint]
			const rgb = rgbOf(getComputedStyle(host).getPropertyValue(token).trim() || fallback)
			img.data.fill(0)
			const f = Math.min(w, h) * 0.9
			const v = (props.speed / 100) * 0.9 * dt
			for (const s of stars) {
				s.pz = s.z
				s.z -= v
				if (s.z <= 0.02) spawn(s), (s.z = 1), (s.pz = 1)
				const sx = w / 2 + (s.x / s.z) * f, sy = h / 2 + (s.y / s.z) * f
				if (sx < 0 || sy < 0 || sx >= w || sy >= h) {
					spawn(s)
					continue
				}
				// Nearer stars are brighter: four stepped levels.
				const a = [70, 130, 200, 255][Math.min(3, Math.floor((1 - s.z) * 4))]
				if (props.warp && v > 0) {
					const px = w / 2 + (s.x / s.pz) * f, py = h / 2 + (s.y / s.pz) * f
					line(px, py, sx, sy, rgb, a)
				} else {
					plot(sx, sy, rgb, a)
					if (s.z < 0.25) plot(sx + 1, sy, rgb, a), plot(sx, sy + 1, rgb, a), plot(sx + 1, sy + 1, rgb, a)
				}
			}
			ctx.putImageData(img, 0, 0)
		}
		const tick = (t) => {
			raf = 0
			const dt = last ? Math.min(0.05, (t - last) / 1000) : 0
			last = t
			paint(dt)
			if (visible && !reduced.matches && props.speed > 0) raf = requestAnimationFrame(tick)
			else last = 0
		}
		const kick = () => {
			if (!raf && visible) raf = requestAnimationFrame(tick)
		}

		resize()
		populate()
		const ro = new ResizeObserver(() => (resize(), kick()))
		ro.observe(host)
		const io = new IntersectionObserver(([e]) => {
			visible = e.isIntersecting
			kick()
		})
		io.observe(host)
		observeProps(() => (populate(), kick()))
		reduced.addEventListener('change', kick)
		kick()
		cleanup(() => {
			cancelAnimationFrame(raf)
			ro.disconnect()
			io.disconnect()
			reduced.removeEventListener('change', kick)
		})
	},
})
