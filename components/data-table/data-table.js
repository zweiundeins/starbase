import { rocket, startPeeking, stopPeeking } from 'datastar'

// Host getters must not subscribe callers (e.g. data-bind's sync effect).
const peek = (fn) => {
	startPeeking()
	try {
		return fn()
	} finally {
		stopPeeking()
	}
}

// One ElementInternals per element: attachInternals() works once, and setup
// runs again when the element is re-attached. Its custom states
// (:state(pending)) are styleable from the page and morph-proof.
const internals = new WeakMap()
const internalsOf = (host) => internals.get(host) ?? internals.set(host, host.attachInternals()).get(host)

// Pixel corners: notches every corner by p (3px times --sb-notch; at 0 the
// border-radius takes over).
const notch = (p) => `polygon(${p} 0, calc(100% - ${p}) 0, calc(100% - ${p}) ${p}, 100% ${p}, 100% calc(100% - ${p}), calc(100% - ${p}) calc(100% - ${p}), calc(100% - ${p}) 100%, ${p} 100%, ${p} calc(100% - ${p}), 0 calc(100% - ${p}), 0 ${p}, ${p} ${p})`

// Browsers cap an element's height (Firefox near 1.8e7 px): a taller table
// keeps its spacer at MAX and moves the rows faster than the scrollbar.
const MAX = 1e7

// Server-driven virtual scroll, after Anders Murphy's hyperlith. Every row has
// the same height; a spacer gives the scroller the height of all of them, and
// the rows at hand sit at translateY(first row × height). It wants the rows in
// view plus a buffer on each side, moves on only when the view is halfway into
// the buffer, and asks for wanted rows it doesn't have (onWindow). Kept in one
// piece, so it can move into sb-virtual-scroll.
const virtualScroll = (onRange, onWindow) => {
	const at = { height: 36, buffer: 4000, offset: 0, length: 0, total: 0 }
	let el, head, timer
	let from = 0, to = -1 // the wanted rows
	let asked = '' // the last window asked for: the same one is asked once
	let k = 1, view = 0 // scroll speed (1 below MAX), the height of the rows in view
	// Nothing at hand yet: ask as if there were no end.
	const count = () => (at.length || at.total ? Math.max(at.total, at.offset + at.length) : Infinity)
	const has = (a, b) => a >= at.offset && Math.min(b, count()) <= at.offset + at.length
	const covered = () => has(from, to)
	const update = (again) => {
		if (!el) return
		const h = at.height, n = count(), s = el.scrollTop
		const rows = Math.ceil(view / h) || 1, b = Math.ceil(at.buffer / h)
		const full = n === Infinity ? 0 : n * h, space = Math.min(full, MAX)
		k = full > space ? (full - view) / (space - view) : 1
		const first = Math.floor((s * k) / h)
		if (again || to < 0 || to > n || (from && first < from + b / 2) || (to < n && first + rows > to - b / 2)) {
			from = Math.max(0, Math.min(first - b, n - 2 * b - rows))
			to = Math.min(n, from + 2 * b + rows)
		}
		const start = Math.max(from, at.offset)
		const end = Math.max(start, Math.min(to, at.offset + at.length))
		onRange({ start, end, space, y: start * h - s * (k - 1), first, busy: !has(first, first + rows) })
		// Asked a task later: the page's listener may not be attached yet, and
		// inside another effect (a morph) the @get it starts would be tracked.
		if (!covered() && !timer)
			timer = setTimeout(() => {
				timer = 0
				const w = { offset: from, count: to - from }
				if (!covered() && asked !== (asked = w.offset + ',' + w.count)) onWindow(w)
			})
	}
	// Sizes are read when they change (a ResizeObserver), not on every scroll.
	const measure = () => {
		view = Math.max(0, el.clientHeight - head.offsetHeight)
		update()
	}
	return {
		attach: (grid, header) => ((el = grid), (head = header), measure()),
		measure,
		set: (o) => {
			const again = o.height !== at.height || o.buffer !== at.buffer
			Object.assign(at, o)
			update(again)
		},
		update: () => update(),
		forget: () => (asked = ''),
		// Scrolls the least that shows row i whole.
		scrollToIndex: (i) => {
			if (!el) return
			const h = at.height, y = i * h - el.scrollTop * k
			if (y < 0) el.scrollTop = (i * h) / k
			else if (y + h > view) el.scrollTop = ((i + 1) * h - view) / k
			update()
		},
		page: () => Math.max(1, Math.floor(view / at.height) - 1),
		size: () => Math.max(1, to - from), // rows in a window
		stop: () => clearTimeout(timer),
	}
}

// Cells: numbers in the reader's format (4,242.5), isolated so a minus sign
// stays in front in right-to-left text; the rest as text.
const format = new Intl.NumberFormat()
const text = (v) => (typeof v === 'number' ? '\u2068' + format.format(v) + '\u2069' : String(v ?? ''))
const align = (a) => (a === 'end' || a === 'right' ? 'end' : a === 'center' ? 'center' : 'start')

const views = new WeakMap() // host → its virtual scroll, from setup to onFirstRender

// Placeholders are bars painted on the spacer, one per row, under the rows at
// hand (their background covers them). Forced colours drop gradients, and
// paint the selection with Highlight.
const styles = /* css */ `
:host {
	--_bg: var(--sb-surface-card, #141D32);
	--_head: var(--sb-surface-raised, #10182B);
	--_border: var(--sb-border, #283552);
	--_line: var(--sb-border-subtle, #1B2640);
	--_text: var(--sb-text-1, #F3F4FA);
	--_label: var(--sb-text-2, #AEBBDD);
	--_hover: var(--sb-surface-hover, #1A2540);
	--_sel: var(--sb-brand-subtle, rgb(140 107 255 / 0.14));
	--_brand: var(--sb-brand, #8C6BFF);
	--_focus: var(--sb-brand-light, #B09AFF);
	--_radius: var(--sb-radius, 8px);
	--_notch: var(--sb-notch, 1);
	--_n: calc(3px * var(--_notch));
	display: flex;
	flex-direction: column;
	inline-size: 100%;
	max-block-size: 24rem;
	color: var(--_text);
	font-size: 0.875rem;
}
:host([hidden]) { display: none; }
.grid {
	flex: 1 1 auto;
	min-block-size: 0;
	overflow: auto;
	overflow-anchor: none;
	overscroll-behavior: contain;
	border: 1px solid var(--_border);
	background: var(--_bg);
	clip-path: ${notch('var(--_n)')};
	border-radius: calc(var(--_radius) * (1 - var(--_notch)));
}
.grid:focus-visible { outline: 2px solid var(--_focus); outline-offset: -2px; }
.table, .head, .body, .win, .row { display: grid; grid-template-columns: subgrid; grid-column: 1 / -1; }
.table { grid-template-columns: var(--cols); }
.head {
	position: sticky;
	inset-block-start: 0;
	z-index: 1;
	background: var(--_head);
	color: var(--_label);
	font-weight: 600;
	border-block-end: 1px solid var(--_border);
}
.body { position: relative; align-content: start; }
.body::before {
	content: "";
	position: absolute;
	inset: 0 0.75rem;
	background: linear-gradient(transparent 32%, var(--_line) 0 68%, transparent 0) 0 var(--y) / 100% var(--h);
}
.busy .body::before { animation: pulse 0.8s steps(2) infinite alternate; }
@keyframes pulse { to { opacity: 0.4; } }
.win { translate: 0 var(--y); grid-auto-rows: var(--h); }
.row { box-sizing: border-box; border-block-end: 1px solid var(--_line); background: var(--_bg); }
.row:hover { background-color: var(--_hover); }
.row[aria-selected] { cursor: pointer; }
.row[aria-selected="true"] { background-image: linear-gradient(var(--_sel) 0 0); box-shadow: inset 2px 0 var(--_brand); }
.row[aria-selected="true"]:dir(rtl) { box-shadow: inset -2px 0 var(--_brand); }
.th, .cell {
	min-inline-size: 0;
	padding-inline: 0.75rem;
	line-height: calc(var(--h) - 1px);
	overflow: hidden;
	white-space: nowrap;
	text-overflow: ellipsis;
}
.th.sortable { padding: 0; }
.th button { all: unset; box-sizing: border-box; display: flex; gap: 0.4rem; align-items: center; inline-size: 100%; padding-inline: 0.75rem; cursor: pointer; }
.th button span { overflow: hidden; text-overflow: ellipsis; }
[data-align="center"] { text-align: center; }
[data-align="end"] { text-align: end; }
[data-align="center"] button { justify-content: center; }
[data-align="end"] button { justify-content: flex-end; }
/* The sort arrow, down for descending: faint until the column is the sort. */
.th button::after {
	content: "";
	flex: none;
	inline-size: 8px;
	block-size: 6px;
	background: currentColor;
	opacity: 0.35;
	clip-path: polygon(0 0, 8px 0, 8px 2px, 6px 2px, 6px 4px, 5px 4px, 5px 6px, 3px 6px, 3px 4px, 2px 4px, 2px 2px, 0 2px);
}
[aria-sort] button::after { opacity: 1; color: var(--_focus); }
[aria-sort="ascending"] button::after { rotate: 180deg; }
.th:focus-visible, .th button:focus-visible, .cell:focus-visible { outline: 2px solid var(--_focus); outline-offset: -2px; }
@media (prefers-reduced-motion: reduce) { .busy .body::before { animation: none; } }
@media (forced-colors: active) {
	.th button::after { forced-color-adjust: none; background: CanvasText; }
	.row[aria-selected="true"] { forced-color-adjust: none; background: Highlight; color: HighlightText; }
}
`

rocket('sb-data-table', {
	props: ({ bool, json, number, oneOf, string }) => ({
		columns: json.default(() => []).docs({ description: 'The columns: [{key, label?, width?, align?, sortable?}]. width is a CSS grid track (a number is px; default minmax(6rem, 1fr)); align is start, center or end.' }),
		rows: json.default(() => []).docs({ description: 'The rows at hand, objects keyed by column: all of them, or the window the server sent, starting at row offset.' }),
		offset: number.min(0).docs({ description: 'Index of the first row in rows.' }),
		total: number.min(0).docs({ description: 'How many rows there are (at least offset plus the rows given).' }),
		rowKey: string.trim.default('id').docs({ description: 'The field that identifies a row, for the selection and sb-row-activate.' }),
		rowHeight: number.clamp(16, 200).default(36).docs({ description: 'The height of every row, in px: fixed, so the scroll position tells which rows are in view.' }),
		buffer: number.clamp(0, 20000).default(4000).docs({ description: 'How much to keep ready above and below the view, in px of rows.' }),
		sort: json.default(() => ({})).docs({ description: 'The order the rows are in: {key, dir: "asc" | "desc"}. The server sends it with the rows; a header click only asks for it (sb-sort).' }),
		loading: bool.docs({ description: 'Rows are on their way (bind it to data-indicator): the placeholders pulse.' }),
		selection: oneOf('none', 'single', 'multiple').default('none').docs({ description: 'How many rows can be selected.' }),
		selected: json.default(() => []).docs({ description: 'The selection: a JSON array of row keys. A new list from the server replaces it; the live value is the selected property.' }),
		label: string.trim.default('Table').docs({ description: 'Accessible name.' }),
		confirm: bool.docs({ description: 'Server-confirmed selection: :state(pending) while the local selection differs from the server\'s selected attribute (see revert()).' }),
		name: string.trim.docs({ description: 'Name reported in sb-change (e.g. the field of a command).' }),
	}),
	manifest: {
		events: [
			{ name: 'sb-window', kind: 'custom-event', bubbles: true, composed: true, description: 'The view needs rows the table doesn\'t have (also on connect and on resize): detail { offset, count }, plus key and dir of the current sort. Answer with rows from that offset, offset and total.' },
			{ name: 'sb-sort', kind: 'custom-event', bubbles: true, composed: true, description: 'A sortable header was clicked. detail: { key, dir }, plus the window to answer with (offset 0, count). Answer with those rows in that order, and the new sort.' },
			{ name: 'sb-row-activate', kind: 'custom-event', bubbles: true, composed: true, description: 'Enter or a double click on a row. detail: { key }.' },
			{ name: 'change', kind: 'event', bubbles: true, composed: true, description: 'The selection changed.' },
			{ name: 'sb-change', kind: 'custom-event', bubbles: true, composed: true, description: 'The selection changed. detail: { name, value } (an array of row keys): ready for a command.' },
		],
	},
	// Rendered once: rows, header and focus all flow through signals, so new
	// rows never rebuild the table (and take the keyboard focus with it).
	renderOnPropChange: false,
	setup: ({ $$, action, adoptStyles, cleanup, defineHostProp, effect, emit, host, observeProps, overrideProp, props }) => {
		adoptStyles(host, styles)
		// What the server sent, in plain variables (only $$.rows is rendered).
		let data = [], off = 0, cols = [], sort = {}, sorted
		const keyAt = (i) => {
			const r = data[i - off]
			return r && String(r[props.rowKey] ?? i)
		}
		$$.rows = '[]'
		$$.space = $$.y = 0
		$$.busy = false
		$$.fr = -1 // the focus cell (roving tabindex): row (-1 the header), column
		$$.fc = 0
		$$.hasFocus = false

		// The rows in the range the virtual scroll renders, re-built only when the
		// range or the rows change.
		let shown = '', stale = true
		const vs = virtualScroll(
			({ start, end, space, y, first, busy }) => {
				$$.space = space
				$$.y = y
				$$.busy = busy || props.loading
				if (!stale && shown === start + ',' + end) return
				stale = false
				shown = start + ',' + end
				// As JSON: data-for clones plain rows fast, and a signal's rows
				// would be proxies, which structuredClone refuses (a slow fallback).
				$$.rows = JSON.stringify(data.slice(start - off, end - off).map((r, j) => ({ i: start + j, k: keyAt(start + j), c: cols.map((c) => text(r?.[c.key])) })))
				// Out of focus, the tab stop stays on a row that is rendered.
				if (!$$.hasFocus && $$.fr >= 0 && ($$.fr < start || $$.fr >= end)) $$.fr = end > start ? Math.min(Math.max(first, start), end - 1) : -1
				refocus()
			},
			// With the order the rows are in, so a handler can pass it all on.
			(w) => emit('sb-window', { ...w, ...sort }),
		)
		views.set(host, vs)
		cleanup(vs.stop)

		const take = () => {
			cols = (Array.isArray(props.columns) ? props.columns : [])
				.filter((c) => c?.key != null)
				.map((c) => ({ key: String(c.key), label: String(c.label ?? c.key), width: c.width, align: align(c.align), sortable: !!c.sortable }))
			data = Array.isArray(props.rows) ? props.rows : []
			off = props.offset
			const key = String(props.sort?.key ?? '')
			sort = { key, dir: !key ? '' : props.sort.dir === 'desc' ? 'desc' : 'asc' }
			$$.cols = JSON.stringify(cols.map((c) => ({ ...c, sort: sort.key === c.key ? (sort.dir === 'desc' ? 'descending' : 'ascending') : '' })))
			$$.al = cols.map((c) => c.align)
			$$.tpl = cols.map((c) => (typeof c.width === 'number' ? c.width + 'px' : c.width || 'minmax(6rem, 1fr)')).join(' ')
			$$.label = props.label
			$$.mode = props.selection
			$$.h = props.rowHeight
			$$.n = Math.max(props.total, off + data.length)
			$$.fc = Math.max(0, Math.min($$.fc, cols.length - 1))
			stale = true
			vs.set({ height: props.rowHeight, buffer: props.buffer, offset: off, length: data.length, total: props.total })
			// A new order from the server starts at the top.
			const now = sort.key + ' ' + sort.dir
			if (sorted !== undefined && now !== sorted) vs.forget(), vs.scrollToIndex(0)
			sorted = now
		}
		take()
		// One take() for every attribute a morph or a signal patch changes at once.
		let queued = false
		observeProps(() => queued || ((queued = true), queueMicrotask(() => peek(() => ((queued = false), take())))))

		// Selection: $$.sel is the local one, the attribute the server's.
		const keys = (v) => (Array.isArray(v) ? v.map(String) : [])
		$$.sel = keys(props.selected)
		overrideProp('selected', () => peek(() => [...$$.sel]), (v) => peek(() => ($$.sel = keys(v))))
		// A new selected attribute from the server wins, also one that decodes
		// like the last ([] onto an element that never had one). The same one
		// again keeps the user's selection; a removed one is ignored (morphs
		// also strip reflected attributes).
		let served = host.getAttribute('selected')
		const watch = new MutationObserver(() =>
			peek(() => {
				const v = host.getAttribute('selected')
				if (v !== served && (served = v) !== null) $$.sel = keys(props.selected)
			}),
		)
		watch.observe(host, { attributeFilter: ['selected'] })
		cleanup(() => watch.disconnect())
		// With confirm, :state(pending) marks a selection the server hasn't
		// confirmed yet (compared as sets); revert() returns to the server's.
		const states = internalsOf(host).states
		const same = (a, b) => a.toSorted().join('\n') === b.toSorted().join('\n')
		const sync = () => peek(() => states[props.confirm && !same([...$$.sel], keys(props.selected)) ? 'add' : 'delete']('pending'))
		effect(() => (JSON.stringify($$.sel), sync()))
		observeProps(sync)
		defineHostProp('revert', { value: () => peek(() => (($$.sel = keys(props.selected)), sync())) })

		const pick = (i, toggle) => {
			const k = keyAt(i)
			if (props.selection === 'none' || k == null) return
			const has = $$.sel.includes(k)
			if (has && !toggle) return
			$$.sel = props.selection === 'multiple' ? (has ? $$.sel.filter((x) => x !== k) : [...$$.sel, k]) : has ? [] : [k]
			emit('change')
			emit('sb-change', { name: props.name, value: [...$$.sel] })
		}
		const activate = (i) => {
			const k = keyAt(i)
			if (k != null) emit('sb-row-activate', { key: k })
		}

		// Moves the DOM focus to the tab stop (the grid itself while its row is
		// not rendered), also after the rows re-rendered: only when the focus is
		// in the table or fell on the floor, never when the user moved on.
		const refocus = (scroll) =>
			requestAnimationFrame(() => {
				const sr = host.shadowRoot
				const active = document.activeElement
				if (!$$.hasFocus || !sr || (active && active !== document.body && active !== host)) return
				const cell = sr.querySelector('.table [tabindex="0"]')
				const el = cell ?? sr.querySelector('.grid')
				if (sr.activeElement !== el) el.focus({ preventScroll: true })
				if (scroll && cell) cell.scrollIntoView({ block: 'nearest', inline: 'nearest' })
			})

		action('scroll', vs.update)
		action('sort', (_, j) => {
			const c = cols[j]
			$$.fr = -1
			$$.fc = j
			// The window to answer with: the top one, in the new order.
			if (c?.sortable) emit('sb-sort', { key: c.key, dir: sort.key === c.key && sort.dir === 'asc' ? 'desc' : 'asc', offset: 0, count: vs.size() })
		})
		// Clicks on rows, heard on the grid (a cell's focusin moved the focus there).
		const rowOf = (evt) => evt.target.closest('.row')?.dataset.r
		action('click', ({ evt }) => rowOf(evt) && pick(+rowOf(evt), props.selection === 'multiple'))
		action('activate', ({ evt }) => rowOf(evt) && activate(+rowOf(evt)))
		// Focus can also arrive by Tab or a click: keep the roving focus in step.
		action('focusin', ({ evt }) => {
			$$.hasFocus = true
			const c = evt.target.closest?.('[data-c]')
			if (c) ($$.fr = +c.closest('[data-r]').dataset.r), ($$.fc = +c.dataset.c)
		})
		action('focusout', ({ evt }) => {
			// A cell re-rendered away also "loses" focus, but that's not leaving
			// (refocus picks it up): decide a frame later, when it is gone.
			const t = evt.target
			requestAnimationFrame(() => t.isConnected && !host.shadowRoot?.activeElement && ($$.hasFocus = false))
		})
		// The grid keys of WAI-ARIA; moves past the rendered rows scroll there,
		// and the rows come with the next window.
		action('key', ({ evt }) => {
			const last = cols.length - 1
			if (evt.altKey || evt.metaKey || last < 0) return
			let r = $$.fr, c = $$.fc
			switch (evt.key) {
				case 'ArrowDown': r++; break
				case 'ArrowUp': r--; break
				case 'ArrowRight':
				case 'ArrowLeft':
					c += (evt.key === 'ArrowLeft') === (getComputedStyle(host).direction === 'rtl') ? 1 : -1
					break
				case 'Home':
					if (evt.ctrlKey) r = -1
					c = 0
					break
				case 'End':
					if (evt.ctrlKey) r = $$.n - 1
					c = last
					break
				case 'PageDown': r += vs.page(); break
				case 'PageUp': r = r < 0 ? r : Math.max(0, r - vs.page()); break
				case ' ':
				case 'Enter':
					// A header's sort button clicks itself.
					if (r < 0) return evt.target.localName === 'button' || evt.preventDefault()
					evt.key === 'Enter' ? activate(r) : pick(r, true)
					return evt.preventDefault()
				default:
					return
			}
			evt.preventDefault()
			$$.fr = r = Math.max(-1, Math.min(r, $$.n - 1))
			$$.fc = Math.max(0, Math.min(c, last))
			if (r >= 0) vs.scrollToIndex(r)
			refocus(true)
		})
	},
	onFirstRender: ({ cleanup, host, refs: { grid, head } }) => {
		const vs = views.get(host)
		vs.attach(grid, head)
		const ro = new ResizeObserver(vs.measure)
		ro.observe(grid)
		ro.observe(head)
		cleanup(() => ro.disconnect())
	},
	render: ({ html }) => html`
		<div class="grid" part="grid" role="grid" tabindex="-1" aria-readonly="true" data-ref:grid
			data-class:busy="$$busy"
			data-attr:aria-label="$$label"
			data-attr:aria-rowcount="$$n + 1"
			data-attr:aria-multiselectable="$$mode === 'multiple' && 'true'"
			data-attr:aria-busy="$$busy && 'true'"
			data-style:--h="$$h + 'px'"
			data-style:--y="$$y + 'px'"
			data-on:scroll="@scroll()" data-on:keydown="@key()" data-on:focusin="@focusin()" data-on:focusout="@focusout()"
			data-on:click="@click()" data-on:dblclick="@activate()">
			<div class="table" data-style:--cols="$$tpl">
				<div class="head" part="header" role="row" aria-rowindex="1" data-r="-1" data-ref:head>
					<template data-for="c, j in JSON.parse($$cols)">
						<div class="th" part="column" role="columnheader"
							data-class:sortable="c?.sortable"
							data-attr:data-c="j"
							data-attr:data-align="c?.align"
							data-attr:aria-sort="c?.sort || null"
							data-attr:tabindex="c?.sortable ? null : $$fr < 0 && j === $$fc ? 0 : -1">
							<button type="button" data-show="c?.sortable" data-attr:tabindex="$$fr < 0 && j === $$fc ? 0 : -1" data-on:click="@sort(j)"><span data-text="c?.label"></span></button>
							<span data-show="!c?.sortable" data-text="c?.label"></span>
						</div>
					</template>
				</div>
				<div class="body" data-style:block-size="$$space + 'px'">
					<div class="win">
						<!-- r?.: data-for can re-evaluate a removed row once with r undefined.
						     No ids on these repeated elements: the morph would park and move them.
						     Few bindings per cell: each one is compiled again for every row. -->
						<template data-for="r in JSON.parse($$rows)">
							<div class="row" part="row" role="row"
								data-attr:part="$$mode !== 'none' && $$sel.includes(r?.k) ? 'row selected' : 'row'"
								data-attr="{'data-r': r?.i, 'aria-rowindex': r?.i + 2, 'aria-selected': $$mode === 'none' ? null : String($$sel.includes(r?.k))}">
								<template data-for="v, j in r?.c">
									<div class="cell" part="cell" role="gridcell"
										data-attr="{'data-c': j, 'data-align': $$al.at(j)}"
										data-attr:tabindex="j === $$fc && r?.i === $$fr ? 0 : -1"
										data-text="v"></div>
								</template>
							</div>
						</template>
					</div>
				</div>
			</div>
		</div>
	`,
})
