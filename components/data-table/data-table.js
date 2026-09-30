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

// Cells: numbers in the reader's format (4,242.5), isolated so a minus sign
// stays in front in right-to-left text; the rest as text.
const format = new Intl.NumberFormat()
const text = (v) => (typeof v === 'number' ? '⁨' + format.format(v) + '⁩' : String(v ?? ''))
// A cell is a value, or an object {value, text, suffix, href, tone}.
const isCell = (v) => v !== null && typeof v === 'object' && !Array.isArray(v)
const raw = (v) => (isCell(v) ? v.value : v)
const tones = new Set(['info', 'success', 'warning', 'danger', 'neutral'])
// Only http, https and mailto links: javascript:, data: and the rest stay text.
const link = (href) => {
	try {
		const u = new URL(String(href), document.baseURI)
		return ['http:', 'https:', 'mailto:'].includes(u.protocol) && u.href
	} catch {
		return false
	}
}
// What a rich row's template shows: t the text, s the suffix, h a checked
// href, n the tone. Nothing here is ever parsed as markup.
const rich = (v) => {
	if (!isCell(v)) return { t: text(v) }
	const c = { t: v.text != null ? String(v.text) : text(v.value) }
	if (v.suffix != null && v.suffix !== '') c.s = String(v.suffix)
	if (v.href) c.h = link(v.href)
	if (tones.has(v.tone)) c.n = v.tone
	return c
}
// Local sorting: numbers by value, the rest as text in the reader's order
// (numeric: "Io 2" before "Io 10"); missing values last in both directions.
const collator = new Intl.Collator(undefined, { numeric: true })
const missing = (v) => v == null || v === ''
const sortKey = (v) => (isCell(v) ? (v.value ?? v.text) : v)
const keys = (v) => (Array.isArray(v) ? v.map(String) : [])
const compare =
	({ key, dir }) =>
	([a], [b]) => {
		const x = sortKey(a?.[key]), y = sortKey(b?.[key])
		if (missing(x) || missing(y)) return missing(x) - missing(y)
		return (dir === 'desc' ? -1 : 1) * (typeof x === 'number' && typeof y === 'number' ? x - y : collator.compare(String(x), String(y)))
	}
const align = (a) => (a === 'end' || a === 'right' ? 'end' : a === 'center' ? 'center' : 'start')

// The windowing is sb-virtual-scroll's: the rows are its children, the header
// row sits in its header slot, and its role="grid" makes it the grid. Every
// row is a grid of the same tracks, at least as wide as their minimums, so the
// columns line up and a wide table scrolls sideways.
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
	--_muted: var(--sb-text-muted, #7785A8);
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
	block-size: calc(var(--rows) * var(--h) + 2px);
	border: 1px solid var(--_border);
	background: var(--_bg);
	clip-path: ${notch('var(--_n)')};
	border-radius: calc(var(--_radius) * (1 - var(--_notch)));
}
.grid[aria-busy] { cursor: progress; }
.head, .row {
	display: grid;
	grid-template-columns: var(--cols);
	min-inline-size: min-content;
	box-sizing: border-box;
	block-size: var(--h);
}
.head { background: var(--_head); color: var(--_label); font-weight: 600; border-block-end: 1px solid var(--_border); }
.row { border-block-end: 1px solid var(--_line); background: var(--_bg); }
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
.head:focus-visible, .th:focus-visible, .th button:focus-visible, .cell:focus-visible { outline: 2px solid var(--_focus); outline-offset: -2px; }
/* Rich cells stay one line of inline boxes, so the row height never changes. */
.text { color: inherit; text-decoration: none; }
.text[href] {
	color: var(--_focus);
	text-decoration: underline;
	text-decoration-color: color-mix(in oklch, currentColor 40%, transparent);
	text-underline-offset: 0.2em;
}
.text[href]:hover { text-decoration-color: currentColor; }
.suffix { margin-inline-start: 0.35em; color: var(--_muted); }
.suffix:empty { display: none; }
[data-tone] {
	--_tone: var(--sb-info, #65BFFF);
	padding-inline: 0.4em;
	border: 1px solid color-mix(in oklch, var(--_tone) 45%, transparent);
	background: color-mix(in oklch, var(--_tone) 16%, transparent);
	color: light-dark(color-mix(in oklab, var(--_tone) 70%, var(--_text)), var(--_tone));
	clip-path: ${notch('calc(2px * var(--_notch))')};
	border-radius: calc(4px * (1 - var(--_notch)));
}
[data-tone="success"] { --_tone: var(--sb-ok, #6EF59A); }
[data-tone="warning"] { --_tone: var(--sb-warn, #F5C451); }
[data-tone="danger"] { --_tone: var(--sb-danger, #F2777A); }
[data-tone="neutral"] { --_tone: var(--_muted); }
@media (forced-colors: active) {
	.th button::after { forced-color-adjust: none; background: CanvasText; }
	.row[aria-selected="true"] { forced-color-adjust: none; background: Highlight; color: HighlightText; }
	.text[href] { color: LinkText; }
	[data-tone] { background: none; border-color: CanvasText; }
	.row[aria-selected="true"] :is(.text, .suffix) { color: inherit; border-color: currentColor; }
}
`

rocket('sb-data-table', {
	props: ({ bool, json, number, oneOf, string }) => ({
		columns: json.default(() => []).docs({ description: 'The columns: [{key, label?, width?, align?, sortable?}]. width is a CSS grid track (a number is px; default minmax(6rem, 1fr)); align is start, center or end. A row\'s value for a key is a scalar or a cell object {value, text?, suffix?, href?, tone?}.' }),
		rows: json.default(() => []).docs({ description: 'The rows at hand, objects keyed by column: all of them, or the window the server sent, starting at row offset. A value is a scalar, or a cell object {value, text, suffix, href, tone}: text replaces the formatted value, suffix follows it muted, href (http, https or mailto) makes it a link, tone (info, success, warning, danger, neutral) a badge. value is the sort key. Nothing is rendered as HTML.' }),
		offset: number.min(0).docs({ description: 'Index of the first row in rows.' }),
		total: number.min(0).docs({ description: 'How many rows there are (at least offset plus the rows given).' }),
		rowKey: string.trim.default('id').docs({ description: 'The field that identifies a row, for the selection and sb-row-activate.' }),
		rowHeight: number.clamp(16, 200).default(36).docs({ description: 'The height of every row, in px: fixed, so the scroll position tells which rows are in view.' }),
		buffer: number.clamp(0, 20000).default(4000).docs({ description: 'How much to keep ready above and below the view, in px of rows.' }),
		sort: json.default(() => ({})).docs({ description: 'The order the rows are in: {key, dir: "asc" | "desc"}. The server sends it with the rows. When the table holds every row (offset 0, no more than total), a header click sorts them here too; otherwise it asks for the order (sb-sort). A new sort from the server wins.' }),
		loading: bool.docs({ description: 'Rows are on their way (bind it to data-indicator): the table is aria-busy.' }),
		selection: oneOf('none', 'single', 'multiple').default('none').docs({ description: 'How many rows can be selected.' }),
		selected: json.default(() => []).docs({ description: 'The selection: a JSON array of row keys. A new list from the server replaces it; the live value is the selected property.' }),
		label: string.trim.default('Table').docs({ description: 'Accessible name.' }),
		confirm: bool.docs({ description: 'Server-confirmed selection: :state(pending) while the local selection differs from the server\'s selected attribute (see revert()).' }),
		name: string.trim.docs({ description: 'Name reported in sb-change (e.g. the field of a command).' }),
	}),
	manifest: {
		events: [
			{ name: 'sb-window', kind: 'custom-event', bubbles: true, composed: true, description: 'The view needs rows the table doesn\'t have (also on connect and on resize): detail { offset, count }, plus key and dir of the order to send them in. Answer with rows from that offset, offset and total.' },
			{ name: 'sb-sort', kind: 'custom-event', bubbles: true, composed: true, description: 'A sortable header was clicked. detail: { key, dir }, plus the window to answer with (offset 0, count). Answer with those rows in that order, and the new sort; a table that holds every row has sorted them already.' },
			{ name: 'sb-row-activate', kind: 'custom-event', bubbles: true, composed: true, description: 'Enter or a double click on a row (not on a link in it). detail: { key }.' },
			{ name: 'sb-cell-activate', kind: 'custom-event', bubbles: true, composed: true, description: 'A click or middle click on a link in a cell, or Enter on its cell. detail: { key, column, value, href }. Cancelable: preventDefault() stops the navigation, for an in-page action instead.' },
			{ name: 'change', kind: 'event', bubbles: true, composed: true, description: 'The selection changed.' },
			{ name: 'sb-change', kind: 'custom-event', bubbles: true, composed: true, description: 'The selection changed. detail: { name, value } (an array of row keys): ready for a command.' },
		],
	},
	// Rendered once: rows, header and focus all flow through signals, so new
	// rows never rebuild the table (and take the keyboard focus with it).
	renderOnPropChange: false,
	setup: ({ $$, action, adoptStyles, cleanup, defineHostProp, effect, emit, emitCancellable, host, observeProps, overrideProp, props }) => {
		adoptStyles(host, styles)
		const $ = (s) => host.shadowRoot?.querySelector(s) // in the rendered template
		// What the server sent, in plain variables (only $$.rows or $$.rich is rendered).
		// asked: the order an sb-sort of ours asked for, until rows in it arrive;
		// olds: the orders it replaced, until `until`; dropped: the rows of the
		// last answer not taken, which stay dropped.
		// shown: the rows as rendered, [row, index in the server's order]; said:
		// the server's last order, so only a new one replaces a local one.
		let data = [], off = 0, cols = [], sort = { key: '', dir: '' }, sorted, asked = null, expire = 0
		let olds = new Set(), until = 0, dropped, shown = [], said
		const orderOf = (o) => o.key + ' ' + o.dir
		const keyAt = (i) => {
			const [r, j] = shown[i - off] ?? []
			return r && String(raw(r[props.rowKey]) ?? j)
		}
		// Every row is here: the table sorts them itself.
		const whole = () => off === 0 && data.length >= props.total
		const show = () => {
			shown = data.map((r, j) => [r, off + j])
			if (whole() && sort.key) shown.sort(compare(sort))
			// As JSON: data-for clones plain rows fast, and a signal's rows would
			// be proxies, which structuredClone refuses (a slow fallback).
			// Rows with a cell object take the second loop, whose cells cost more.
			const fancy = shown.some(([r]) => cols.some((c) => isCell(r?.[c.key])))
			const out = JSON.stringify(shown.map(([r], j) => ({ i: off + j, k: keyAt(off + j), c: cols.map((c) => (fancy ? rich : text)(r?.[c.key])) })))
			if (fancy) ($$.rows = '[]'), ($$.rich = out)
			else ($$.rich = '[]'), ($$.rows = out)
			$$.cols = JSON.stringify(cols.map((c) => ({ ...c, sort: sort.key === c.key ? (sort.dir === 'desc' ? 'descending' : 'ascending') : '' })))
		}
		// A new order starts at the top.
		const top = () => {
			if (sorted !== undefined && orderOf(sort) !== sorted) $('.grid')?.scrollToIndex?.(0)
			sorted = orderOf(sort)
		}
		$$.rows = '[]'
		$$.rich = '[]'
		$$.fr = -1 // the focus cell (roving tabindex): row (-1 the header), column
		$$.fc = 0
		$$.hasFocus = false

		// Moves the DOM focus to the tab stop (the header row while its row is
		// not rendered: a tabindex on the list would take its rows out of the
		// tab order), also after the rows re-rendered: only when the focus is in
		// the table or fell on the floor, never when the user moved on.
		const refocus = (scroll) =>
			requestAnimationFrame(() => {
				const active = document.activeElement
				if (!$$.hasFocus || (active && active !== document.body && active !== host)) return
				const cell = $('.grid [tabindex="0"]')
				const el = cell ?? $('.head')
				if (el && host.shadowRoot.activeElement !== el) el.focus({ preventScroll: true })
				if (scroll && cell && $$.fr >= 0) cell.scrollIntoView({ block: 'nearest', inline: 'nearest' })
			})

		const take = () => {
			cols = (Array.isArray(props.columns) ? props.columns : [])
				.filter((c) => c?.key != null)
				.map((c) => ({ key: String(c.key), label: String(c.label ?? c.key), width: c.width, align: align(c.align), sortable: !!c.sortable }))
			$$.al = cols.map((c) => c.align)
			$$.tpl = cols.map((c) => (typeof c.width === 'number' ? c.width + 'px' : c.width || 'minmax(6rem, 1fr)')).join(' ')
			$$.label = props.label
			$$.mode = props.selection
			$$.h = props.rowHeight
			$$.buf = props.buffer
			$$.loading = props.loading
			$$.fc = Math.max(0, Math.min($$.fc, cols.length - 1))
			const key = String(props.sort?.key ?? '')
			const now = { key, dir: !key ? '' : props.sort.dir === 'desc' ? 'desc' : 'asc' }
			const order = orderOf(now)
			// While our sb-sort is unanswered, only its order is taken, and for a
			// while after it rows in an order it replaced are late answers to
			// earlier windows, which would put that order back. An order the
			// server sends on its own wins.
			if (asked ? order !== orderOf(asked) : olds.has(order) && performance.now() < until) dropped = props.rows
			else if (props.rows !== dropped) {
				asked = null
				clearTimeout(expire)
				if (order !== said) (said = order), (sort = now)
				data = Array.isArray(props.rows) ? props.rows : []
				off = props.offset
				show()
				$$.n = Math.max(props.total, off + data.length)
				$$.off = off
				// No rows: no total, so the list asks for its first window.
				$$.tot = $$.n || false
				// Out of focus, the tab stop stays on a row that is rendered.
				if (!$$.hasFocus && ($$.fr < off || $$.fr >= off + data.length)) $$.fr = -1
				top()
				refocus()
			} else show() // new columns
		}
		take()
		// One take() for every attribute a morph or a signal patch changes at once.
		let queued = false
		observeProps(() => queued || ((queued = true), queueMicrotask(() => peek(() => ((queued = false), take())))))
		cleanup(() => clearTimeout(expire))

		// Selection: $$.sel is the local one, the attribute the server's.
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

		// Rows in view: the list's height minus the header's.
		const inView = () => Math.max(1, Math.floor(($('.grid').clientHeight - $('.head').offsetHeight) / props.rowHeight))

		// The list asks for a window: ask the page, with the order to send it in.
		action('window', ({ evt }) => {
			evt.stopPropagation()
			emit('sb-window', { offset: evt.detail.offset, count: evt.detail.count, ...(asked ?? sort) })
		})
		action('sort', (_, j) => {
			const c = cols[j]
			$$.fr = -1
			$$.fc = j
			if (!c?.sortable) return
			const next = { key: c.key, dir: sort.key === c.key && sort.dir === 'asc' ? 'desc' : 'asc' }
			// The window to answer with: the top one, the view and its buffers.
			const count = inView() + 1 + 2 * Math.ceil(props.buffer / props.rowHeight)
			// With every row here, sort them now; the page still hears of it.
			if (whole()) return (sort = next), show(), top(), refocus(), emit('sb-sort', { ...next, offset: 0, count })
			olds.add(orderOf(sort)).add(orderOf(asked ?? sort))
			asked = next
			olds.delete(orderOf(asked))
			until = performance.now() + 10000
			// A sort that never comes back (or comes back in another order) stops
			// holding the rows up after a while.
			clearTimeout(expire)
			expire = setTimeout(() => peek(() => ((asked = null), take())), 5000)
			emit('sb-sort', { ...asked, offset: 0, count })
		})
		// Clicks on rows, heard on the grid (a cell's focusin moved the focus
		// there). The clicks of a double click don't select twice.
		const rowOf = (evt) => evt.target.closest('.row')?.dataset.r
		const follow = (evt, a) => {
			const i = +a.closest('.row').dataset.r, c = cols[+a.closest('[data-c]').dataset.c]
			const v = shown[i - off]?.[0]?.[c?.key]
			if (c && !emitCancellable('sb-cell-activate', { key: keyAt(i), column: c.key, value: raw(v) ?? null, href: a.href })) evt.preventDefault()
		}
		action('click', ({ evt }) => {
			const a = evt.target.closest('a[href]')
			if (a) return follow(evt, a)
			evt.detail < 2 && rowOf(evt) && pick(+rowOf(evt), props.selection === 'multiple')
		})
		// A middle click opens a link in a new tab: the page may cancel that too.
		action('aux', ({ evt }) => { const a = evt.button === 1 && evt.target.closest('a[href]'); a && follow(evt, a) })
		action('activate', ({ evt }) => !evt.target.closest('a[href]') && rowOf(evt) && activate(+rowOf(evt)))
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
				case 'PageDown': r += inView() - 1 || 1; break
				case 'PageUp': r = r < 0 ? r : Math.max(0, r - (inView() - 1 || 1)); break
				case ' ':
				case 'Enter': {
					// A header's sort button clicks itself, and so does a cell's link
					// (its click emits sb-cell-activate).
					if (r < 0) return evt.target.localName === 'button' || evt.preventDefault()
					const a = evt.key === 'Enter' && evt.target.closest('[data-c]')?.querySelector('a[href]')
					a ? a.click() : evt.key === 'Enter' ? activate(r) : pick(r, true)
					return evt.preventDefault()
				}
				default:
					return
			}
			evt.preventDefault()
			$$.fr = r = Math.max(-1, Math.min(r, $$.n - 1))
			$$.fc = Math.max(0, Math.min(c, last))
			if (r >= 0) $('.grid').scrollToIndex?.(r, { block: 'nearest' })
			refocus(true)
		})
	},
	// Runs on every connect, after the render.
	onFirstRender: ({ refs: { grid, parts } }) => grid.append(...parts.childNodes),
	render: ({ html }) => {
		// $$rows (plain cells) or $$rich (cell objects): the same rows either way.
		// r?. and || '[]': data-for can re-evaluate once with its signals gone.
		const loop = (list, content) => html`
			<!-- No ids on these repeated elements: the morph would park and move them.
			     Few bindings per cell: each one is compiled again for every row. -->
			<template data-for="r in JSON.parse($$${list} || '[]')">
				<div class="row" part="row" role="row"
					data-attr:part="$$mode !== 'none' && $$sel.includes(r?.k) ? 'row selected' : 'row'"
					data-attr="{'data-r': r?.i, 'aria-rowindex': r?.i + 2, 'aria-selected': $$mode === 'none' ? null : String($$sel.includes(r?.k))}">
					<template data-for="v, j in r?.c">
						<div class="cell" part="cell" role="gridcell"
							data-attr="{'data-c': j, 'data-align': $$al.at(j)}"
							data-attr:tabindex="j === $$fc && r?.i === $$fr ? 0 : -1"
							data-text="${content ? null : 'v'}">${content}</div>
					</template>
				</div>
			</template>
		`
		return html`
		<sb-virtual-scroll class="grid" part="grid" role="grid" aria-readonly="true" data-ref:grid
			data-attr="{'item-size': $$h, buffer: $$buf, offset: $$off, total: $$tot, 'aria-label': $$label, 'aria-rowcount': $$n + 1, 'aria-multiselectable': $$mode === 'multiple' && 'true', 'aria-busy': $$loading && 'true'}"
			data-style:--h="$$h + 'px'" data-style:--rows="$$n + 1" data-style:--cols="$$tpl"
			data-on:sb-window="@window()" data-on:keydown="@key()" data-on:focusin="@focusin()" data-on:focusout="@focusout()"
			data-on:click="@click()" data-on:auxclick="@aux()" data-on:dblclick="@activate()"></sb-virtual-scroll>
		<!-- The header and the rows are rendered here and handed to the list (onFirstRender): a list
		     upgraded before this template is scoped would take their $$ bindings for its own. -->
		<div hidden data-ref:parts>
			<div slot="header" class="head" part="header" role="row" aria-rowindex="1" data-r="-1" tabindex="-1">
				<template data-for="c, j in JSON.parse($$cols || '[]')">
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
			${loop('rows', null)}
			${loop('rich', html`<a class="text" data-attr="{href: v?.h || false, tabindex: v?.h ? '-1' : false, 'data-tone': v?.n || false}" data-attr:part="'text' + (v?.h ? ' link' : '') + (v?.n ? ' badge' : '')" data-text="v?.t"></a><span class="suffix" part="suffix" data-text="v?.s || ''"></span>`)}
		</div>
	`
	},
})
