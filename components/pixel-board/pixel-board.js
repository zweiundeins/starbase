import { rocket } from 'datastar'

// A 16-colour palette tuned to the site: space, steel, violet, sky, nature, fire.
const PALETTE = [
	'#080D1D', '#1B1F3B', '#283552', '#7785A8', '#AEBBDD', '#F3F4FA', '#8C6BFF', '#B09AFF',
	'#5B4FD6', '#65BFFF', '#2F7BE0', '#6EF59A', '#4CC46A', '#FFD84D', '#FF9F43', '#E5484D',
]
const PENDING_MS = 3000
const FLUSH_MS = 80
const BATCH = 60 // cells per sb-paint at most (the Showcase server's burst)

// gridOn is the default line colour (--_grid-auto) for a board whose blank cells are c:
// the lines lie on the cells, not on the page, so faint white on a dark board, faint black on a light one.
// (Comments inside the CSS string would ship: keep them out here.)
const gridOn = ([r, g, b]) => (0.2126 * r + 0.7152 * g + 0.0722 * b > 140 ? 'rgb(0 0 0 / 0.1)' : 'rgb(255 255 255 / 0.06)')

const hex = (c) => {
	const m = /^#?([0-9a-f]{6})$/i.exec(c)
	const n = m ? parseInt(m[1], 16) : 0
	return [n >> 16, (n >> 8) & 255, n & 255]
}

// host → [buffer, the cells it was decoded from]: a re-attach (a move, a morph) keeps a local drawing.
const boards = new WeakMap()

// A readonly board renders no palette, so its frame is an only child: styled from what is rendered,
// not from the attribute (el.readonly = false writes readonly="false").
const styles = /* css */ `
:host {
	--_border: var(--sb-border, #283552);
	--_focus: var(--sb-brand-light, #B09AFF);
	--_surface: var(--sb-surface-inset, #0B1224);
	--_swatch-edge: color-mix(in srgb, var(--sb-text-1, #F3F4FA) 12%, transparent);
	display: inline-block;
	inline-size: var(--sb-pixel-board-size, 24rem);
	max-inline-size: 100%;
	vertical-align: middle;
}
:host([hidden]) { display: none; }
.board { display: grid; gap: 0.75rem; container-type: inline-size; }
.frame { position: relative; aspect-ratio: 1; border: 1px solid var(--_border); background: var(--_surface); }
canvas { display: block; inline-size: 100%; block-size: 100%; image-rendering: pixelated; cursor: crosshair; touch-action: pinch-zoom; }
.frame:only-child canvas { cursor: default; touch-action: auto; }
canvas:focus-visible { outline: 2px solid var(--_focus); outline-offset: 3px; }
.grid {
	position: absolute;
	inset: 0;
	pointer-events: none;
	background-image:
		linear-gradient(to right, var(--_grid) 1px, transparent 1px),
		linear-gradient(to bottom, var(--_grid) 1px, transparent 1px);
	--_grid: var(--sb-pixel-board-grid, var(--_grid-auto, rgb(255 255 255 / 0.06)));
	background-size: calc(100% / var(--_n)) calc(100% / var(--_n));
}
.palette { display: grid; grid-template-columns: repeat(8, 1fr); gap: 4px; }
@container (width > 22rem) { .palette { grid-template-columns: repeat(16, 1fr); } }
.palette input {
	all: unset;
	aspect-ratio: 1;
	border: 1px solid var(--_swatch-edge);
	cursor: pointer;
	transition: translate 80ms;
	forced-color-adjust: none;
}
.palette input:hover { translate: 0 -1px; }
.palette :checked { outline: 2px solid var(--_focus); outline-offset: 2px; }
.palette :focus-visible { outline: 2px solid var(--_focus); outline-offset: 2px; }
`

rocket('sb-pixel-board', {
	props: ({ array, bool, number, string }) => ({
		size: number.clamp(8, 128).step(1).default(48).docs({ description: 'Cells per side.' }),
		cells: string.docs({ description: 'Board state: one hex digit (palette index) per cell, row by row. Usually server-owned.' }),
		palette: array(string.trim).default(() => PALETTE).docs({ description: '16 colours as a JSON array of #rrggbb.' }),
		color: number.clamp(0, 15).default(7).docs({ description: 'Selected palette index (the attribute sets it; the user can change it).' }),
		local: bool.docs({ description: 'Paint the local buffer directly (offline mode). Otherwise painted cells stay pending until cells echoes them.' }),
		readonly: bool.docs({ description: 'Watch only: no painting, no palette.' }),
		grid: bool.docs({ description: 'Show faint cell lines.' }),
	}),
	manifest: {
		events: [
			{ name: 'sb-paint', kind: 'custom-event', bubbles: true, composed: true, description: 'Painted cells, batched every ~80 ms during a stroke. detail: { color, cells: [index…] }.' },
		],
	},
	renderOnPropChange: ({ changes }) => 'readonly' in changes || 'grid' in changes || 'size' in changes,
	setup: ({ $$, action, adoptStyles, cleanup, emit, host, observeProps, props }) => {
		adoptStyles(host, styles)
		$$.color = props.color
		$$.palette = props.palette
		// ?. because disconnecting deletes the signals first, and a throw here would leave the host half-disconnected.
		$$.gridAuto = () => gridOn(hex($$.palette?.[0] ?? PALETTE[0]))
		$$.cur = -1 // the keyboard cursor
		$$.v = 0 // bumped on every paint, so the label follows the cells

		const n = () => props.size
		let [buf, from] = boards.get(host) || []
		const pending = new Map() // idx → { color }
		const queue = new Set()
		let ctx = null, img = null, raf = 0, flushTimer = 0
		let drawing = false, pid, last = -1, hover = -1

		const decode = () => {
			const size = n() * n()
			if (buf?.length !== size) (buf = new Uint8Array(size)), ($$.cur = -1)
			boards.set(host, [buf, (from = props.cells)])
			const s = from || ''
			for (let i = 0; i < size; i++) {
				const v = parseInt(s[i] ?? '0', 16)
				buf[i] = Number.isNaN(v) ? 0 : v
			}
			// Pixels the server has confirmed are no longer pending.
			for (const [i, p] of pending) if (buf[i] === p.color) pending.delete(i)
		}
		const paint = () => {
			raf = 0
			ctx ||= host.shadowRoot.querySelector('canvas')?.getContext('2d')
			if (!ctx) return
			const size = n(), cur = $$.cur
			if (!img || img.width !== size) {
				ctx.canvas.width = ctx.canvas.height = size
				img = ctx.createImageData(size, size)
			}
			const rgb = $$.palette.map(hex)
			const d = img.data
			for (let i = 0; i < size * size; i++) {
				const p = pending.get(i)
				let c = rgb[p ? p.color : buf[i]] || rgb[0]
				if (p) c = c.map((v, k) => Math.round(v * 0.6 + rgb[buf[i]][k] * 0.4)) // in flight
				if (i === hover || i === cur) c = c.map((v) => Math.round(v + (255 - v) * 0.35))
				d[i * 4] = c[0]
				d[i * 4 + 1] = c[1]
				d[i * 4 + 2] = c[2]
				d[i * 4 + 3] = 255
			}
			ctx.putImageData(img, 0, 0)
			$$.v++
		}
		const redraw = () => {
			if (!raf) raf = requestAnimationFrame(paint)
		}
		const flush = () => {
			flushTimer = 0
			if (!queue.size) return
			emit('sb-paint', { color: $$.color, cells: [...queue] })
			queue.clear()
		}
		const put = (i) => {
			if (props.readonly) return
			if (props.local) buf[i] = $$.color
			else {
				const p = { color: $$.color }
				pending.set(i, p)
				setTimeout(() => pending.get(i) === p && (pending.delete(i), redraw()), PENDING_MS)
			}
			queue.add(i)
			if (queue.size >= BATCH) flush()
			else if (!flushTimer) flushTimer = setTimeout(flush, FLUSH_MS)
			redraw()
		}
		// Every cell on the grid line from a to b (a itself is painted already), so fast drags leave no gaps.
		const line = (a, b) => {
			const s = n(), x = a % s, y = (a / s) | 0, dx = (b % s) - x, dy = ((b / s) | 0) - y
			const k = Math.max(Math.abs(dx), Math.abs(dy))
			for (let t = 1; t <= k; t++) put(Math.round(y + (dy * t) / k) * s + Math.round(x + (dx * t) / k))
		}
		const cellAt = (el, evt) => {
			const r = el.getBoundingClientRect()
			const size = n()
			const x = Math.floor(((evt.clientX - r.left) / r.width) * size)
			const y = Math.floor(((evt.clientY - r.top) / r.height) * size)
			return x < 0 || y < 0 || x >= size || y >= size ? -1 : y * size + x
		}
		const describe = (i) => {
			if (i < 0) return 'Pixel board'
			const size = n()
			return `Pixel board, cell ${(i % size) + 1},${((i / size) | 0) + 1}, colour ${buf[i] + 1}; painting colour ${$$.color + 1}`
		}

		// One gesture: the primary button of one pointer, from down until its capture ends (up or cancel).
		action('stroke', ({ el, evt }) => {
			const i = cellAt(el, evt)
			switch (evt.type) {
				case 'pointerdown':
					if (props.readonly || i < 0 || evt.button || drawing) return
					el.setPointerCapture((pid = evt.pointerId))
					drawing = true
					put((last = i))
					break
				case 'pointermove':
					if (hover !== i && !props.readonly) (hover = i), redraw()
					// Leaving the board ends the segment: coming back starts a new one, not a line across.
					if (drawing && evt.pointerId === pid && i !== last) i < 0 ? (last = -1) : (last < 0 ? put(i) : line(last, i), (last = i))
					break
				case 'pointerleave':
					hover = -1
					redraw()
					break
				default: // lostpointercapture
					if (evt.pointerId === pid) (drawing = false), flush()
			}
		})
		action('key', ({ evt }) => {
			// Only these keys arm the cursor: a move, or 0 for paint. Tab and the rest pass.
			const size = n(), m = { ArrowLeft: [-1, 0], ArrowRight: [1, 0], ArrowUp: [0, -1], ArrowDown: [0, 1], ' ': 0, Enter: 0 }[evt.key]
			if (m === undefined) return
			evt.preventDefault()
			let c = $$.cur
			if (c < 0) c = ((size / 2) | 0) * (size + 1)
			if (m) c = Math.min(size - 1, Math.max(0, ((c / size) | 0) + m[1])) * size + Math.min(size - 1, Math.max(0, (c % size) + m[0]))
			else put(c), flush()
			$$.cur = c
			redraw()
		})
		action('blur', () => (($$.cur = -1), redraw()))

		observeProps(() => (decode(), redraw()), 'cells', 'size')
		observeProps(() => ($$.palette = props.palette, redraw()), 'palette')
		observeProps(() => ($$.color = props.color), 'color')
		if (from !== props.cells || buf?.length !== n() * n()) decode()
		$$.label = () => ($$.v, describe($$.cur))
		redraw()
		cleanup(() => {
			cancelAnimationFrame(raf)
			clearTimeout(flushTimer)
			pending.clear() // the per-cell timers find nothing left to expire
		})
	},
	render: ({ html, props: { size, readonly, grid } }) => html`
		<div class="board" part="board">
			<div class="frame" style="--_n: ${size}" data-style:--_grid-auto="$$gridAuto">
				<canvas part="canvas" width="${size}" height="${size}" tabindex="0" role="application"
					data-attr:aria-label="$$label"
					data-on:pointerdown="@stroke()"
					data-on:pointermove="@stroke()"
					data-on:pointerleave="@stroke()"
					data-on:lostpointercapture="@stroke()"
					data-on:keydown="@key()"
					data-on:blur="@blur()"></canvas>
				${grid ? html`<div class="grid" aria-hidden="true"></div>` : null}
			</div>
			${readonly ? null : html`
				<div class="palette" part="palette" role="radiogroup" aria-label="Colour">
					<template data-for="c, i in $$palette">
						<input type="radio" name="c"
							data-attr:aria-label="'Colour ' + (i + 1)"
							data-style:background="c"
							data-effect="el.checked = $$color === i"
							data-on:change="$$color = i">
					</template>
				</div>`}
		</div>
	`,
})
