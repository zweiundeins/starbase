import { rocket } from 'datastar'

// Resolve any CSS colour (tokens, oklch, …) to [r, g, b] in 0..1.
const probe = document.createElement('canvas').getContext('2d', { willReadFrequently: true })
const rgbCache = new Map()
const rgbOf = (css) => {
	if (rgbCache.has(css)) return rgbCache.get(css)
	probe.clearRect(0, 0, 1, 1)
	probe.fillStyle = '#000'
	probe.fillStyle = css
	probe.fillRect(0, 0, 1, 1)
	const [r, g, b] = probe.getImageData(0, 0, 1, 1).data
	const rgb = [r / 255, g / 255, b / 255]
	rgbCache.set(css, rgb)
	return rgb
}
const luma = ([r, g, b]) => 2 * r + 7 * g + b

// Four stops per palette, darkest first: space, gas, glow, highlights/stars.
// On a light theme space comes out lightest: paint() then sorts them by lightness.
const PALETTES = {
	violet: [['--sb-surface-inset', '#0B1224'], ['--sb-brand', '#8C6BFF'], ['--sb-accent', '#65BFFF'], ['--sb-text-1', '#F3F4FA']],
	aurora: [['--sb-surface-inset', '#0B1224'], ['--sb-accent', '#65BFFF'], ['--sb-datastar', '#6EF59A'], ['--sb-text-1', '#F3F4FA']],
	ember: [['--sb-surface-inset', '#0B1224'], ['--sb-danger', '#F2777A'], ['--sb-warn', '#F5C451'], ['--sb-text-1', '#F3F4FA']],
	mono: [['--sb-surface-inset', '#0B1224'], ['--sb-border-strong', '#3A4868'], ['--sb-text-muted', '#7785A8'], ['--sb-text-1', '#F3F4FA']],
}

const VERT = `attribute vec2 a; void main() { gl_Position = vec4(a, 0.0, 1.0); }`

// Domain-warped fbm (after Inigo Quilez), posterized with a 4×4 Bayer
// dither so it reads as pixel art, plus a twinkling star layer.
// u_p is the props: seed, density, levels, stars.
const FRAG = `
#ifdef GL_FRAGMENT_PRECISION_HIGH
precision highp float;
#else
precision mediump float;
#endif
uniform vec2 u_res;
uniform vec2 u_ptr;
uniform float u_time;
uniform vec4 u_p;
uniform vec3 u_c0, u_c1, u_c2, u_c3;

float hash(vec2 p) {
	p = fract(p * vec2(123.34, 456.21));
	p += dot(p, p + 45.32);
	return fract(p.x * p.y);
}
float noise(vec2 p) {
	vec2 i = floor(p), f = fract(p);
	vec2 u = f * f * (3.0 - 2.0 * f);
	return mix(mix(hash(i), hash(i + vec2(1.0, 0.0)), u.x),
	           mix(hash(i + vec2(0.0, 1.0)), hash(i + vec2(1.0, 1.0)), u.x), u.y);
}
float fbm(vec2 p) {
	float v = 0.0, a = 0.5;
	mat2 m = mat2(1.6, 1.2, -1.2, 1.6);
	for (int i = 0; i < 5; i++) { v += a * noise(p); p = m * p; a *= 0.5; }
	return v;
}
float bayer2(vec2 a) { a = floor(a); return fract(dot(a, vec2(0.5, a.y * 0.75))); }
float bayer4(vec2 a) { return bayer2(0.5 * a) * 0.25 + bayer2(a); }

void main() {
	vec2 uv = (gl_FragCoord.xy - 0.5 * u_res) / u_res.y;
	float t = u_time;
	vec2 p = uv * 2.4 + u_ptr * 0.18 + u_p.x * 7.31;
	vec2 q = vec2(fbm(p + 0.05 * t), fbm(p + vec2(5.2, 1.3) - 0.04 * t));
	vec2 r = vec2(fbm(p + 3.5 * q + vec2(1.7, 9.2) + 0.08 * t), fbm(p + 3.5 * q + vec2(8.3, 2.8) + 0.06 * t));
	float f = fbm(p + 3.0 * r);

	float edge = mix(0.72, 0.28, u_p.y);
	float cloud = smoothstep(edge - 0.22, edge + 0.4, f);
	vec3 col = mix(u_c0, u_c1, clamp(cloud * 1.3, 0.0, 1.0));
	col = mix(col, u_c2, clamp(dot(q, q) * cloud * 1.1 - 0.15, 0.0, 1.0));
	col = mix(col, u_c3, smoothstep(0.62, 1.0, cloud * r.y * 1.5));

	// Stars sit behind the gas and move less with the pointer (parallax).
	vec2 cell = floor(gl_FragCoord.xy + u_ptr * 4.0);
	float s = hash(cell + u_p.x * 13.0);
	float star = step(0.9965, s) * (0.6 + 0.4 * sin(t * 2.0 + s * 6283.0));
	col = mix(col, u_c3, star * u_p.w * (1.0 - cloud * 0.8));

	if (u_p.z > 0.5) col = floor(col * u_p.z + bayer4(gl_FragCoord.xy)) / u_p.z;
	gl_FragColor = vec4(clamp(col, 0.0, 1.0), 1.0);
}`

// The host's background shows wherever the canvas doesn't paint: without
// WebGL, or once the context is lost. Where it paints, it is opaque.
const styles = /* css */ `
:host {
	--_bg: var(--sb-surface-inset, #0B1224);
	--_glow: var(--sb-brand, #8C6BFF);
	--_radius: var(--sb-radius, 8px);
	display: block;
	inline-size: 100%;
	block-size: var(--sb-nebula-height, 14rem);
	border-radius: var(--_radius);
	overflow: hidden;
	background: radial-gradient(ellipse at 35% 45%, color-mix(in oklch, var(--_glow) 55%, transparent), transparent 60%), var(--_bg);
	contain: strict;
}
:host([hidden]) { display: none; }
canvas { display: block; inline-size: 100%; block-size: 100%; image-rendering: pixelated; touch-action: pan-y; }
`

rocket('sb-nebula', {
	props: ({ bool, number, oneOf, string }) => ({
		speed: number.clamp(0, 5).default(1).docs({ description: 'Drift speed (0 is still).' }),
		density: number.clamp(0, 1).default(0.5).docs({ description: 'How much of the sky the gas covers, 0 to 1.' }),
		palette: oneOf('violet', 'aurora', 'ember', 'mono').default('violet').docs({ description: 'Colour scheme (from theme tokens).' }),
		pixel: number.clamp(1, 16).default(4).docs({ description: 'Size of one pixel, in CSS pixels.' }),
		levels: number.clamp(0, 16).default(6).docs({ description: 'Colour levels per channel, dithered (0 for smooth).' }),
		stars: bool.default(true).docs({ description: 'Draw twinkling stars.' }),
		seed: number.docs({ description: 'Picks a different nebula.' }),
		parallax: bool.default(true).docs({ description: 'Shift with the pointer.' }),
		label: string.default('Animated nebula').docs({ description: 'Accessible label.' }),
	}),
	renderOnPropChange: false,
	render: ({ html }) => html`
		<canvas
			part="canvas"
			role="img"
			data-ref:canvas
			data-on:pointermove="@point()"
			data-on:pointerleave="@leave()"
		></canvas>
	`,
	// All of it runs here rather than in setup: onFirstRender gets the same
	// context plus the canvas, and runs again on every connect.
	onFirstRender: ({ action, adoptStyles, cleanup, host, observeProps, props, refs: { canvas } }) => {
		adoptStyles(host, styles)
		const reduced = matchMedia('(prefers-reduced-motion: reduce)')
		const scheme = matchMedia('(prefers-color-scheme: dark)')
		let gl, loc
		let raf = 0, last = 0, time = 0, visible = true
		let px = 0, py = 0, tx = 0, ty = 0 // eased pointer and its target, -1..1

		const compile = () => {
			const sh = (type, src) => {
				const s = gl.createShader(type)
				gl.shaderSource(s, src)
				gl.compileShader(s)
				if (!gl.getShaderParameter(s, gl.COMPILE_STATUS) && !gl.isContextLost()) throw new Error('sb-nebula: ' + gl.getShaderInfoLog(s))
				return s
			}
			const prog = gl.createProgram()
			gl.attachShader(prog, sh(gl.VERTEX_SHADER, VERT))
			gl.attachShader(prog, sh(gl.FRAGMENT_SHADER, FRAG))
			gl.bindAttribLocation(prog, 0, 'a')
			gl.linkProgram(prog)
			gl.useProgram(prog)
			gl.bindBuffer(gl.ARRAY_BUFFER, gl.createBuffer())
			gl.bufferData(gl.ARRAY_BUFFER, new Float32Array([-1, -1, 3, -1, -1, 3]), gl.STATIC_DRAW)
			gl.enableVertexAttribArray(0)
			gl.vertexAttribPointer(0, 2, gl.FLOAT, false, 0, 0)
			loc = {}
			for (const n of ['res', 'ptr', 'time', 'p', 'c0', 'c1', 'c2', 'c3']) loc[n] = gl.getUniformLocation(prog, 'u_' + n)
		}
		const paint = () => {
			if (!gl || gl.isContextLost()) return
			const css = getComputedStyle(host)
			const c = PALETTES[props.palette].map(([token, fallback]) => rgbOf(css.getPropertyValue(token).trim() || fallback))
			if (luma(c[0]) > luma(c[3])) c.sort((a, b) => luma(a) - luma(b))
			c.forEach((rgb, i) => gl.uniform3fv(loc['c' + i], rgb))
			gl.viewport(0, 0, canvas.width, canvas.height)
			gl.uniform2f(loc.res, canvas.width, canvas.height)
			gl.uniform2f(loc.ptr, px, py)
			gl.uniform1f(loc.time, time)
			// Any seed picks a nebula: large ones would run out of float precision.
			gl.uniform4f(loc.p, props.seed % 101, props.density, props.levels, props.stars)
			gl.drawArrays(gl.TRIANGLES, 0, 3)
		}
		const tick = (t) => {
			raf = 0
			const dt = last ? Math.min(0.05, (t - last) / 1000) : 0
			last = t
			const still = reduced.matches
			if (!still) time += dt * props.speed
			// Ease toward the pointer; stop once settled.
			const on = props.parallax && !still, gx = on ? tx : 0, gy = on ? ty : 0
			const k = 1 - Math.exp(-dt * 6)
			px += (gx - px) * k
			py += (gy - py) * k
			const easing = Math.abs(gx - px) + Math.abs(gy - py) > 0.002
			paint()
			if (visible && ((!still && props.speed > 0) || easing)) raf = requestAnimationFrame(tick)
			else last = 0
		}
		const kick = () => {
			if (!raf && visible && gl && !gl.isContextLost()) raf = requestAnimationFrame(tick)
		}
		// Resizing the canvas clears it: paint again before the frame shows.
		const refresh = () => {
			canvas.width = Math.max(1, Math.ceil(host.clientWidth / props.pixel))
			canvas.height = Math.max(1, Math.ceil(host.clientHeight / props.pixel))
			paint()
			kick()
		}
		// An empty label makes it decoration, hidden from assistive technology.
		const name = () => {
			canvas.ariaLabel = props.label
			canvas.ariaHidden = !props.label || null
		}

		// After a GPU reset the context comes back (webglcontextrestored); one
		// the browser took for its cap on live contexts stays lost, and the
		// host's background shows. addEventListener, not data-on: cleanup()'s
		// loseContext() fires webglcontextlost after Rocket has torn the
		// element down, when a declarative binding would find no action.
		canvas.addEventListener('webglcontextlost', (evt) => {
			evt.preventDefault()
			cancelAnimationFrame(raf)
			raf = 0
		})
		canvas.addEventListener('webglcontextrestored', () => (compile(), kick()))
		gl = canvas.getContext('webgl', { antialias: false, alpha: false, depth: false, stencil: false, powerPreference: 'low-power' })
		if (gl) compile()
		name()

		action('point', ({ el, evt }) => {
			const r = el.getBoundingClientRect()
			tx = ((evt.clientX - r.left) / r.width) * 2 - 1
			ty = 1 - ((evt.clientY - r.top) / r.height) * 2
			kick()
		})
		action('leave', () => {
			tx = ty = 0
			kick()
		})

		const ro = new ResizeObserver(refresh) // also the first paint
		ro.observe(host)
		const io = new IntersectionObserver(([e]) => {
			visible = e.isIntersecting
			kick()
		})
		io.observe(host)
		// Colours are read at paint time: repaint a still nebula when they
		// change (a pick on <sb-theme-switch>, or the system's scheme for "auto").
		addEventListener('sb-theme-change', kick)
		for (const m of [reduced, scheme]) m.addEventListener('change', kick)
		observeProps(name, 'label')
		observeProps(refresh)
		cleanup(() => {
			cancelAnimationFrame(raf)
			ro.disconnect()
			io.disconnect()
			removeEventListener('sb-theme-change', kick)
			for (const m of [reduced, scheme]) m.removeEventListener('change', kick)
			gl?.getExtension('WEBGL_lose_context')?.loseContext() // free the slot
			canvas.remove() // a lost context stays lost: the next connect renders a new canvas
		})
	},
})
