import { rocket } from 'datastar'

// Internal resolution: models are rasterised into RES×RES pixels, then
// scaled up with image-rendering: pixelated for crisp 8-bit edges.
const RES = 128

const PALETTE = {
	white: [243, 244, 250],
	steel: [142, 151, 196],
	red: [229, 72, 77],
	violet: [140, 107, 255],
	cyan: [101, 191, 255],
	navy: [43, 76, 154],
	gold: [255, 199, 77],
	amber: [255, 159, 67],
	green: [76, 196, 106],
	ocean: [47, 123, 224],
	ice: [226, 240, 255],
	lilac: [176, 154, 255],
	flameY: [255, 224, 102],
	flameO: [255, 130, 58],
}
const EMISSIVE = new Set(['flameY', 'flameO'])
const OUTLINE = [8, 13, 29]

// Deterministic 3D value noise for planet continents.
const hash = (x, y, z) => {
	let n = (x * 374761393 + y * 668265263 + z * 1274126177) | 0
	n = Math.imul(n ^ (n >>> 13), 1274126177)
	return ((n ^ (n >>> 16)) >>> 0) / 4294967296
}
const noise = (x, y, z) => {
	const xi = Math.floor(x), yi = Math.floor(y), zi = Math.floor(z)
	const s = (t) => t * t * (3 - 2 * t)
	const u = s(x - xi), v = s(y - yi), w = s(z - zi)
	const l = (a, b, t) => a + (b - a) * t
	const c = (dx, dy, dz) => hash(xi + dx, yi + dy, zi + dz)
	return l(
		l(l(c(0, 0, 0), c(1, 0, 0), u), l(c(0, 1, 0), c(1, 1, 0), u), v),
		l(l(c(0, 0, 1), c(1, 0, 1), u), l(c(0, 1, 1), c(1, 1, 1), u), v),
		w,
	)
}

// Models are functions from a voxel coordinate to a palette key (or null),
// sampled over a bounding box.
const MODELS = {
	rocket: {
		box: [-6, 6, -4, 19, -6, 6],
		voxel(x, y, z) {
			const r = Math.hypot(x, z)
			if (y < 0) return r <= 1.6 + y * 0.35 ? (y > -2 ? 'flameY' : 'flameO') : null
			if (y <= 1) return r <= 1.7 ? 'steel' : null
			if (y === 2) return r <= 2.6 ? 'steel' : null
			const fin = (Math.abs(x) < 0.5 || Math.abs(z) < 0.5) && y <= 7 && r > 3 && r <= 3 + (7 - y) * 0.45 + 0.6
			if (fin) return 'red'
			if (y <= 13) {
				if (r > 3.2) return null
				if (y === 5 || y === 12) return 'violet'
				if (z >= 2.4 && Math.abs(x) <= 1.2 && (y === 9 || y === 10)) return 'cyan'
				if (z >= 2.4 && Math.abs(x) <= 1.8 && y >= 8 && y <= 11) return 'navy'
				return 'white'
			}
			const nose = [2.8, 2.3, 1.8, 1.2, 0.7][y - 14]
			if (nose === undefined || r > nose) return null
			return y >= 16 ? 'red' : 'white'
		},
	},
	satellite: {
		box: [-12, 12, -3, 7, -3, 3],
		voxel(x, y, z) {
			if (Math.abs(x) <= 2 && Math.abs(y) <= 2 && Math.abs(z) <= 2) return (x + y + z) % 2 ? 'gold' : 'amber'
			if (Math.abs(x) >= 4 && Math.abs(x) <= 12 && y === 0 && Math.abs(z) <= 2) return Math.abs(x) % 3 === 0 ? 'cyan' : 'navy'
			if (Math.abs(x) === 3 && y === 0 && z === 0) return 'steel'
			if (y >= 3 && y <= 4 && Math.hypot(x, z) <= 2.2 - (4 - y) * 0.8) return 'white'
			if (x === 0 && z === 0 && y >= 5 && y <= 7) return y === 7 ? 'red' : 'steel'
			return null
		},
	},
	planet: {
		box: [-13, 13, -8, 8, -13, 13],
		voxel(x, y, z) {
			const d = Math.hypot(x, y, z)
			if (d <= 7.6) {
				if (Math.abs(y) >= 6.3) return 'ice'
				return noise(x / 3.2 + 7, y / 3.2, z / 3.2) > 0.55 ? 'green' : 'ocean'
			}
			const r = Math.hypot(x, z)
			if (y === 0 && r >= 9.5 && r <= 12.5) return r > 11 ? 'violet' : 'lilac'
			return null
		},
	},
}

// Precompute exposed faces for a model: 4 corners, a normal and a colour.
const FACE_DIRS = [
	[1, 0, 0], [-1, 0, 0], [0, 1, 0], [0, -1, 0], [0, 0, 1], [0, 0, -1],
]
const cache = new Map()
const facesFor = (name) => {
	if (cache.has(name)) return cache.get(name)
	const m = MODELS[name]
	const [x0, x1, y0, y1, z0, z1] = m.box
	const grid = new Map()
	const key = (x, y, z) => `${x},${y},${z}`
	for (let x = x0; x <= x1; x++)
		for (let y = y0; y <= y1; y++)
			for (let z = z0; z <= z1; z++) {
				const c = m.voxel(x, y, z)
				if (c) grid.set(key(x, y, z), c)
			}
	const cx = (x0 + x1) / 2, cy = (y0 + y1) / 2, cz = (z0 + z1) / 2
	const faces = []
	for (const [k, color] of grid) {
		const [x, y, z] = k.split(',').map(Number)
		for (const n of FACE_DIRS) {
			if (grid.has(key(x + n[0], y + n[1], z + n[2]))) continue
			// Corners of the unit face centred on the voxel's face.
			const [a, b] = n[0] ? [[0, 1, 0], [0, 0, 1]] : n[1] ? [[1, 0, 0], [0, 0, 1]] : [[1, 0, 0], [0, 1, 0]]
			const fc = [x - cx + n[0] / 2, y - cy + n[1] / 2, z - cz + n[2] / 2]
			const corner = (s, t) => [fc[0] + (a[0] * s + b[0] * t) / 2, fc[1] + (a[1] * s + b[1] * t) / 2, fc[2] + (a[2] * s + b[2] * t) / 2]
			faces.push({ p: [corner(-1, -1), corner(1, -1), corner(1, 1), corner(-1, 1)], n, color })
		}
	}
	const extent = Math.max(x1 - x0, y1 - y0, z1 - z0) + 1
	const result = { faces, extent }
	cache.set(name, result)
	return result
}

// Draw one frame into an ImageData with a z-buffer.
const draw = (img, zbuf, model, yaw, pitch, zoom, light) => {
	const { faces, extent } = facesFor(model)
	const data = img.data
	data.fill(0)
	zbuf.fill(-Infinity)
	const cy = Math.cos(yaw), sy = Math.sin(yaw), cp = Math.cos(pitch), sp = Math.sin(pitch)
	const rot = ([x, y, z]) => {
		const x1 = x * cy + z * sy
		const z1 = -x * sy + z * cy
		return [x1, y * cp - z1 * sp, y * sp + z1 * cp]
	}
	const scale = (RES * 0.78 * zoom) / extent
	const elev = 0.7
	const L = [Math.cos(light) * Math.cos(elev), Math.sin(elev), Math.sin(light) * Math.cos(elev)]
	const steps = [0.42, 0.62, 0.82, 1]
	const fill = (ax, ay, az, bx, by, bz, cx2, cy2, cz2, rgb) => {
		const minX = Math.max(0, Math.floor(Math.min(ax, bx, cx2)))
		const maxX = Math.min(RES - 1, Math.ceil(Math.max(ax, bx, cx2)))
		const minY = Math.max(0, Math.floor(Math.min(ay, by, cy2)))
		const maxY = Math.min(RES - 1, Math.ceil(Math.max(ay, by, cy2)))
		const area = (bx - ax) * (cy2 - ay) - (by - ay) * (cx2 - ax)
		if (Math.abs(area) < 1e-9) return
		for (let py = minY; py <= maxY; py++) {
			for (let px = minX; px <= maxX; px++) {
				const x = px + 0.5, y = py + 0.5
				let w0 = ((bx - x) * (cy2 - y) - (by - y) * (cx2 - x)) / area
				let w1 = ((cx2 - x) * (ay - y) - (cy2 - y) * (ax - x)) / area
				let w2 = 1 - w0 - w1
				if (w0 < -1e-6 || w1 < -1e-6 || w2 < -1e-6) continue
				const z = w0 * az + w1 * bz + w2 * cz2
				const i = py * RES + px
				if (z <= zbuf[i]) continue
				zbuf[i] = z
				data[i * 4] = rgb[0]
				data[i * 4 + 1] = rgb[1]
				data[i * 4 + 2] = rgb[2]
				data[i * 4 + 3] = 255
			}
		}
	}
	for (const f of faces) {
		const n = rot(f.n)
		if (n[2] <= 0) continue // facing away
		const base = PALETTE[f.color]
		let rgb = base
		if (!EMISSIVE.has(f.color)) {
			const d = Math.max(0, n[0] * L[0] + n[1] * L[1] + n[2] * L[2])
			const lvl = steps[Math.min(3, Math.floor((0.3 + 0.7 * d) * 4))]
			rgb = [base[0] * lvl, base[1] * lvl, base[2] * lvl]
		}
		const s = f.p.map((v) => {
			const r = rot(v)
			return [RES / 2 + r[0] * scale, RES / 2 - r[1] * scale, r[2]]
		})
		fill(...s[0], ...s[1], ...s[2], rgb)
		fill(...s[0], ...s[2], ...s[3], rgb)
	}
	// 1px silhouette outline, the pixel-art way.
	for (let y = 0; y < RES; y++) {
		for (let x = 0; x < RES; x++) {
			const i = y * RES + x
			if (data[i * 4 + 3]) continue
			const on = (xx, yy) => xx >= 0 && yy >= 0 && xx < RES && yy < RES && data[(yy * RES + xx) * 4 + 3] === 255
			if (on(x - 1, y) || on(x + 1, y) || on(x, y - 1) || on(x, y + 1)) {
				data[i * 4] = OUTLINE[0]
				data[i * 4 + 1] = OUTLINE[1]
				data[i * 4 + 2] = OUTLINE[2]
				data[i * 4 + 3] = 254
			}
		}
	}
}

const styles = /* css */ `
:host {
	--_size: var(--sb-voxel-size, 16rem);
	--_focus: var(--sb-brand-light, #B09AFF);
	display: inline-block;
	inline-size: var(--_size);
	max-inline-size: 100%;
	aspect-ratio: 1;
	vertical-align: middle;
}
canvas {
	display: block;
	inline-size: 100%;
	block-size: 100%;
	image-rendering: pixelated;
	cursor: grab;
	touch-action: none;
}
canvas:active { cursor: grabbing; }
canvas:focus-visible { outline: 2px solid var(--_focus); outline-offset: 4px; border-radius: 4px; }
`

const rad = (d) => (d * Math.PI) / 180

rocket('sb-voxel', {
	props: ({ number, oneOf }) => ({
		model: oneOf('rocket', 'satellite', 'planet').default('rocket').docs({ description: 'Which voxel model to show.' }),
		yaw: number.clamp(-180, 180).default(30).docs({ description: 'Rotation around the vertical axis, in degrees.' }),
		pitch: number.clamp(-89, 89).default(15).docs({ description: 'Tilt toward the viewer, in degrees.' }),
		zoom: number.clamp(0.25, 4).default(1).docs({ description: 'Scale factor.' }),
		spin: number.clamp(-360, 360).docs({ description: 'Auto-rotation speed, in degrees per second.' }),
		light: number.default(45).docs({ description: 'Light direction around the model, in degrees.' }),
	}),
	manifest: {
		events: [
			{ name: 'sb-orbit', kind: 'custom-event', bubbles: true, composed: true, description: 'After a drag or an arrow-key turn. detail: { yaw, pitch } (effective angles).' },
		],
	},
	// The canvas is repainted imperatively; props never re-render the DOM.
	renderOnPropChange: false,
	setup: ({ adoptStyles, host }) => adoptStyles(host, styles),
	// Everything else needs the canvas, so it starts once it is rendered.
	onFirstRender: ({ action, cleanup, emit, host, observeProps, props, refs: { canvas } }) => {
		const ctx = canvas.getContext('2d')
		const img = ctx.createImageData(RES, RES)
		const reduced = matchMedia('(prefers-reduced-motion: reduce)')
		const zbuf = new Float32Array(RES * RES)

		// The view: the page's angles, turned locally by drags and keys (never
		// written to attributes), plus the spin.
		let yaw = props.yaw, pitch = props.pitch, spun = 0, drag = null, turned = 0
		let visible = true, raf = 0, last = 0

		const angles = () => ({ yaw: ((((yaw + spun) % 360) + 540) % 360) - 180, pitch })
		const paint = () => {
			const a = angles()
			draw(img, zbuf, props.model, rad(a.yaw), rad(a.pitch), props.zoom, rad(props.light))
			ctx.putImageData(img, 0, 0)
			// While it spins, the name says so rather than changing every frame.
			const label = `Voxel ${props.model}, ${last ? 'spinning' : `yaw ${Math.round(a.yaw)}°, pitch ${Math.round(a.pitch)}°`}`
			if (canvas.ariaLabel !== label) canvas.ariaLabel = label
		}
		// Each frame paints; while spinning (and visible) it asks for the next one.
		const tick = (t) => {
			raf = 0
			const spinning = props.spin && !reduced.matches
			if (spinning) spun = (spun + (props.spin * (last ? t - last : 0)) / 1000) % 360
			last = spinning && visible ? t : 0
			if (last) invalidate()
			paint()
		}
		const invalidate = () => {
			if (!raf && visible) raf = requestAnimationFrame(tick)
		}
		observeProps(invalidate)
		// A new angle from the page wins over the local orbit.
		observeProps(() => { yaw = props.yaw; spun = 0 }, 'yaw')
		observeProps(() => { pitch = props.pitch }, 'pitch')
		// No declarative hook exists for these two: keep them imperative.
		reduced.addEventListener('change', invalidate)
		const io = new IntersectionObserver(([e]) => {
			if ((visible = e.isIntersecting)) invalidate()
		})
		io.observe(host)
		invalidate()

		// Turn the view; the pitch stops at ±89°. Only a real change is a turn.
		const turn = (dy, dp) => {
			const p = Math.max(-89, Math.min(89, pitch + dp))
			if (!dy && p === pitch) return
			yaw += dy
			pitch = p
			turned = 1
			invalidate()
		}
		// One action for both gestures: a drag (one pointer: down, move, up or
		// cancel) and an arrow key (down, repeats, up). Its end reports the turn.
		action('orbit', ({ el, evt }) => {
			if (evt.type === 'keydown') {
				const step = evt.shiftKey ? 15 : 5
				const move = { ArrowLeft: [-step, 0], ArrowRight: [step, 0], ArrowUp: [0, -step], ArrowDown: [0, step] }[evt.key]
				if (move) {
					evt.preventDefault()
					turn(...move)
				}
			} else if (evt.type === 'pointerdown') {
				if (evt.button || !evt.isPrimary) return // the first finger, the main button
				drag = { id: evt.pointerId, x: evt.clientX, y: evt.clientY }
				el.setPointerCapture(evt.pointerId)
			} else if (drag?.id !== evt.pointerId) {
				return // another pointer, or a key up during a drag
			} else if (evt.type === 'pointermove') {
				const k = 360 / el.clientWidth
				turn((evt.clientX - drag.x) * k, (evt.clientY - drag.y) * k * 0.5)
				drag.x = evt.clientX
				drag.y = evt.clientY
			} else {
				drag = null
				if (turned) emit('sb-orbit', angles())
				turned = 0
			}
		})

		cleanup(() => {
			cancelAnimationFrame(raf)
			io.disconnect()
			reduced.removeEventListener('change', invalidate)
		})
	},
	render: ({ html }) => html`
		<canvas part="canvas" width="${RES}" height="${RES}" tabindex="0" role="img"
			aria-description="Drag or use the arrow keys to turn it"
			data-ref:canvas
			data-on:pointerdown="@orbit()"
			data-on:pointermove="@orbit()"
			data-on:pointerup="@orbit()"
			data-on:pointercancel="@orbit()"
			data-on:keydown="@orbit()"
			data-on:keyup="@orbit()"></canvas>
	`,
})
