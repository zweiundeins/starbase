import { rocket } from 'datastar'

// A 16-colour palette tuned to the site: space, steel, violet, sky, nature, fire.
const PALETTE = [
	'#080D1D', '#1B1F3B', '#283552', '#7785A8', '#AEBBDD', '#F3F4FA', '#8C6BFF', '#B09AFF',
	'#5B4FD6', '#65BFFF', '#2F7BE0', '#6EF59A', '#4CC46A', '#FFD84D', '#FF9F43', '#E5484D',
]
const PENDING_MS = 3000
const FLUSH_MS = 80

const hex = (c) => {
	const m = /^#?([0-9a-f]{6})$/i.exec(c)
	const n = m ? parseInt(m[1], 16) : 0
	return [n >> 16, (n >> 8) & 255, n & 255]
}

// Per-instance painter shared by setup and onFirstRender.
const painters = new WeakMap()

const styles = /* css */ `
:host {
	--_border: var(--sb-border, #283552);
	--_focus: var(--sb-brand-light, #B09AFF);
	--_surface: var(--sb-surface-inset, #0B1224);
	display: inline-block;
	inline-size: var(--sb-pixel-board-size, 24rem);
	max-inline-size: 100%;
	vertical-align: middle;
}
.board { display: grid; gap: 0.75rem; }
.frame { position: relative; aspect-ratio: 1; border: 1px solid var(--_border); background: var(--_surface); }
canvas { display: block; inline-size: 100%; block-size: 100%; image-rendering: pixelated; cursor: crosshair; touch-action: none; }
:host([readonly]) canvas { cursor: default; }
canvas:focus-visible { outline: 2px solid var(--_focus); outline-offset: 3px; }
.grid {
	position: absolute;
	inset: 0;
	pointer-events: none;
	background-image:
		linear-gradient(to right, rgb(255 255 255 / 0.06) 1px, transparent 1px),
		linear-gradient(to bottom, rgb(255 255 255 / 0.06) 1px, transparent 1px);
	background-size: calc(100% / var(--_n)) calc(100% / var(--_n));
}
.palette { display: grid; grid-template-columns: repeat(8, 1fr); gap: 4px; }
.palette button {
	all: unset;
	aspect-ratio: 1;
	border: 1px solid rgb(255 255 255 / 0.12);
	cursor: pointer;
	transition: translate 80ms;
}
.palette button:hover { translate: 0 -1px; }
.palette button[aria-checked="true"] { outline: 2px solid var(--_focus); outline-offset: 2px; }
.palette button:focus-visible { outline: 2px solid var(--_focus); outline-offset: 2px; }
`

rocket('sb-pixel-board', {
	props: ({ array, bool, number, string }) => ({
		size: number.clamp(8, 128).step(1).default(48).docs({ description: 'Cells per side.' }),
		cells: string.docs({ description: 'Board state: one hex digit (palette index) per cell, row by row. Usually server-owned.' }),
		palette: array(string.trim).default(() => PALETTE).docs({ description: '16 colours as a JSON array of #rrggbb.' }),
		color: number.clamp(0, 15).default(7).docs({ description: 'Initially selected palette index.' }),
		local: bool.docs({ description: 'Paint the local buffer directly (offline mode). Otherwise painted cells stay pending until cells echoes them.' }),
		readonly: bool.docs({ description: 'Watch only: no painting, no palette.' }),
		grid: bool.docs({ description: 'Show faint cell lines.' }),
	}),
	manifest: {
		events: [
			{ name: 'sb-paint', kind: 'custom-event', bubbles: true, composed: true, description: 'Painted cells, batched every ~80 ms during a stroke. detail: { color, cells: [index…] }.' },
		],
	},
	renderOnPropChange: ({ changes }) => 'readonly' in changes || 'grid' in changes,
	setup: ({ $$, action, adoptStyles, cleanup, emit, host, observeProps, props }) => {
		adoptStyles(host, styles)
		$$.color = props.color
		$$.palette = props.palette
		$$.label = 'Pixel board'

		const n = () => props.size
		let buf = new Uint8Array(n() * n())
		const pending = new Map() // idx → { color, at }
		const queue = new Set()
		let ctx = null, img = null, raf = 0, flushTimer = 0, expiry = 0
		let drawing = false, last = -1, hover = -1, cursor = -1

		const decode = () => {
			const size = n() * n()
			if (buf.length !== size) buf = new Uint8Array(size)
			const s = props.cells || ''
			for (let i = 0; i < size; i++) {
				const v = parseInt(s[i] ?? '0', 16)
				buf[i] = Number.isNaN(v) ? 0 : v
			}
			// Pixels the server has confirmed are no longer pending.
			for (const [i, p] of pending) if (buf[i] === p.color) pending.delete(i)
		}
		const paint = () => {
			raf = 0
			if (!ctx) return
			const size = n()
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
				if (i === hover || i === cursor) c = c.map((v) => Math.round(v + (255 - v) * 0.35))
				d[i * 4] = c[0]
				d[i * 4 + 1] = c[1]
				d[i * 4 + 2] = c[2]
				d[i * 4 + 3] = 255
			}
			ctx.putImageData(img, 0, 0)
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
		const expire = () => {
			const now = performance.now()
			for (const [i, p] of pending) if (now - p.at > PENDING_MS) pending.delete(i)
			redraw()
			expiry = pending.size ? setTimeout(expire, 500) : 0
		}
		const put = (i) => {
			if (i < 0 || props.readonly) return
			if (props.local) buf[i] = $$.color
			else {
				pending.set(i, { color: $$.color, at: performance.now() })
				if (!expiry) expiry = setTimeout(expire, 500)
			}
			queue.add(i)
			if (!flushTimer) flushTimer = setTimeout(flush, FLUSH_MS)
			redraw()
		}
		// Every cell on the grid line between two cells, so fast drags leave no gaps.
		const line = (a, b) => {
			const size = n()
			let x0 = a % size, y0 = (a / size) | 0
			const x1 = b % size, y1 = (b / size) | 0
			const dx = Math.abs(x1 - x0), dy = -Math.abs(y1 - y0), sx = x0 < x1 ? 1 : -1, sy = y0 < y1 ? 1 : -1
			let err = dx + dy
			for (;;) {
				put(y0 * size + x0)
				if (x0 === x1 && y0 === y1) break
				const e2 = 2 * err
				if (e2 >= dy) (err += dy), (x0 += sx)
				if (e2 <= dx) (err += dx), (y0 += sy)
			}
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

		// One gesture: down, move, up, cancel, leave.
		action('stroke', ({ el, evt }) => {
			const i = cellAt(el, evt)
			switch (evt.type) {
				case 'pointerdown':
					if (props.readonly || i < 0) return
					el.setPointerCapture(evt.pointerId)
					drawing = true
					last = i
					put(i)
					break
				case 'pointermove':
					if (hover !== i) (hover = i), redraw()
					if (drawing && i >= 0 && i !== last) line(last, i), (last = i)
					break
				default: // pointerup, pointercancel, pointerleave
					if (evt.type === 'pointerleave') (hover = -1), redraw()
					if (drawing && evt.type !== 'pointerleave') (drawing = false), flush()
			}
		})
		action('key', ({ evt }) => {
			const size = n()
			if (cursor < 0) cursor = ((size / 2) | 0) * size + ((size / 2) | 0)
			const x = cursor % size, y = (cursor / size) | 0
			const moves = { ArrowLeft: [-1, 0], ArrowRight: [1, 0], ArrowUp: [0, -1], ArrowDown: [0, 1] }
			if (evt.key in moves) {
				evt.preventDefault()
				const nx = Math.min(size - 1, Math.max(0, x + moves[evt.key][0]))
				const ny = Math.min(size - 1, Math.max(0, y + moves[evt.key][1]))
				cursor = ny * size + nx
			} else if (evt.key === ' ' || evt.key === 'Enter') {
				evt.preventDefault()
				put(cursor)
				flush()
			} else return
			$$.label = describe(cursor)
			redraw()
		})

		observeProps(() => (decode(), redraw()), 'cells', 'size')
		observeProps(() => ($$.palette = props.palette, redraw()), 'palette')
		observeProps(() => ($$.color = props.color), 'color')
		decode()
		painters.set(host, (canvas) => {
			ctx = canvas.getContext('2d')
			img = null
			redraw()
		})
		cleanup(() => {
			cancelAnimationFrame(raf)
			clearTimeout(flushTimer)
			clearTimeout(expiry)
		})
	},
	render: ({ html, props: { size, readonly, grid } }) => html`
		<div class="board" part="board">
			<div class="frame" style="--_n: ${size}">
				<canvas part="canvas" width="${size}" height="${size}" tabindex="0" role="img"
					data-ref:canvas
					data-attr:aria-label="$$label"
					data-on:pointerdown="@stroke()"
					data-on:pointermove="@stroke()"
					data-on:pointerup="@stroke()"
					data-on:pointercancel="@stroke()"
					data-on:pointerleave="@stroke()"
					data-on:keydown="@key()"></canvas>
				${grid ? html`<div class="grid" aria-hidden="true"></div>` : null}
			</div>
			${readonly ? null : html`
				<div class="palette" part="palette" role="radiogroup" aria-label="Colour">
					<template data-for="c, i in $$palette">
						<button type="button" role="radio"
							data-attr:aria-checked="String($$color === i)"
							data-attr:aria-label="'Colour ' + (i + 1)"
							data-style:background="c"
							data-on:click="$$color = i"></button>
					</template>
				</div>`}
		</div>
	`,
	// The canvas exists now; hand it to the painter.
	onFirstRender: ({ host, refs }) => painters.get(host)(refs.canvas),
})
