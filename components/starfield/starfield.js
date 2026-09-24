import { rocket } from 'datastar'

// Each CSS pixel block is PX device-independent pixels: chunky on purpose.
const PX = 3

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
	props: ({ bool, number, oneOf, string }) => ({
		speed: number.clamp(0, 100).default(20).docs({ description: 'Flight speed (0 pauses the field).' }),
		density: number.clamp(20, 1500).default(300).docs({ description: 'Number of stars.' }),
		tint: oneOf('white', 'violet', 'cyan', 'green').default('white').docs({ description: 'Star colour (from theme tokens).' }),
		warp: bool.docs({ description: 'Draw streaks instead of dots.' }),
		label: string.default('Animated starfield').docs({ description: 'Accessible label (empty for a decorative background).' }),
	}),
	renderOnPropChange: false,
	setup: ({ adoptStyles, host }) => adoptStyles(host, styles),
	render: ({ html }) => html`<canvas part="canvas" data-ref:canvas></canvas>`,
	onFirstRender: ({ cleanup, host, observeProps, props, refs: { canvas } }) => {
		// willReadFrequently: paint() reads each new star colour back from this canvas.
		const ctx = canvas.getContext('2d', { willReadFrequently: true })
		const reduced = matchMedia('(prefers-reduced-motion: reduce)')
		const scheme = matchMedia('(prefers-color-scheme: dark)')
		let w, h, css, rgb, img = ctx.createImageData(1, 1)
		let stars = []
		let raf = 0, last = 0, v = 0, visible = true

		// x and y are relative to the view: a star is on screen while |x| and
		// |y| are below z, so a resize stretches the field instead of cropping
		// it. Stars come from a box of ±1.2 seen at a focal length of 0.9 × the
		// shorter side, so there are none before the field has a size.
		const spawn = (s = {}) => {
			s.x = (Math.random() * 2 - 1) * 2.16 * Math.min(1, h / w)
			s.y = (Math.random() * 2 - 1) * 2.16 * Math.min(1, w / h)
			s.z = Math.random() * 0.95 + 0.05
			s.pz = s.z
			return s
		}
		const populate = () => {
			const n = w ? Math.round(props.density) : 0
			while (stars.length < n) stars.push(spawn())
			stars.length = n
		}
		// Without a size (not laid out yet, or hidden) it keeps the last one.
		const resize = () => {
			const x = Math.floor(host.clientWidth / PX), y = Math.floor(host.clientHeight / PX)
			if (x && y) (canvas.width = w = x), (canvas.height = h = y), (img = ctx.createImageData(x, y)), populate()
		}
		const plot = (x, y, a) => {
			x |= 0
			y |= 0
			if (x < 0 || y < 0 || x >= w || y >= h) return
			const i = (y * w + x) * 4
			if (img.data[i + 3] >= a) return
			img.data.set(rgb, i)
			img.data[i + 3] = a
		}
		const line = (x0, y0, x1, y1, a) => {
			const n = Math.max(Math.abs(x1 - x0), Math.abs(y1 - y0), 1)
			for (let i = 0; i <= n; i++) plot(x0 + ((x1 - x0) * i) / n, y0 + ((y1 - y0) * i) / n, Math.round(a * (0.35 + (0.65 * i) / n)))
		}
		// paint() without dt draws the stars where they are, as the last frame did.
		const paint = (dt) => {
			// Resolve the colour (tokens, oklch, …) to RGB on one pixel, once per
			// colour: putImageData below overwrites it.
			const [token, fallback] = TINTS[props.tint]
			const c = getComputedStyle(host).getPropertyValue(token).trim() || fallback
			if (c !== css) {
				ctx.clearRect(0, 0, 1, 1)
				ctx.fillStyle = '#000'
				ctx.fillStyle = css = c
				ctx.fillRect(0, 0, 1, 1)
				rgb = ctx.getImageData(0, 0, 1, 1).data
			}
			img.data.fill(0)
			if (dt >= 0) v = (props.speed / 100) * 0.9 * dt
			next: for (const s of stars) {
				if (dt >= 0) {
					s.pz = s.z
					s.z -= v
					if (s.z <= 0.02) spawn(s), (s.z = 1), (s.pz = 1)
				}
				// Off screen: a moving field shows the new star from the next frame
				// on; a still one draws until it has one on screen, so it shows them all.
				while (Math.abs(s.x) >= s.z || Math.abs(s.y) >= s.z) if ((spawn(s), v)) continue next
				const sx = (w / 2) * (1 + s.x / s.z), sy = (h / 2) * (1 + s.y / s.z)
				// Nearer stars are brighter: four stepped levels.
				const a = [70, 130, 200, 255][Math.min(3, Math.floor((1 - s.z) * 4))]
				if (props.warp && v > 0) {
					line((w / 2) * (1 + s.x / s.pz), (h / 2) * (1 + s.y / s.pz), sx, sy, a)
				} else {
					plot(sx, sy, a)
					if (s.z < 0.25) plot(sx + 1, sy, a), plot(sx, sy + 1, a), plot(sx + 1, sy + 1, a)
				}
			}
			ctx.putImageData(img, 0, 0)
		}
		// The frame it stops on is a still one (dt = 0), e.g. when reduced motion is turned on.
		const tick = (t) => {
			const go = visible && !reduced.matches && props.speed > 0
			paint(go && last ? Math.min(0.05, (t - last) / 1000) : 0)
			last = go && t
			raf = go && requestAnimationFrame(tick)
		}
		// A still field paints once more (dt = 0 moves nothing).
		const kick = () => {
			if (!raf && visible) raf = requestAnimationFrame(tick)
		}
		const sync = () => {
			canvas.role = props.label ? 'img' : 'none'
			canvas.ariaLabel = props.label || null
			populate()
			kick()
		}

		resize()
		sync()
		// Resizing clears the canvas after this frame's animation callbacks: draw
		// it again right away, or the frame shows it blank.
		const ro = new ResizeObserver(() => (resize(), paint()))
		ro.observe(host)
		const io = new IntersectionObserver(([e]) => {
			visible = e.isIntersecting
			kick()
		})
		io.observe(host)
		observeProps(sync)
		// Token colours change with the theme: repaint a still field too.
		reduced.addEventListener('change', kick)
		scheme.addEventListener('change', kick)
		addEventListener('sb-theme-change', kick)
		cleanup(() => {
			cancelAnimationFrame(raf)
			ro.disconnect()
			io.disconnect()
			reduced.removeEventListener('change', kick)
			scheme.removeEventListener('change', kick)
			removeEventListener('sb-theme-change', kick)
		})
	},
})
