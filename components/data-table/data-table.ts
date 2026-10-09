import { rocket, startPeeking, stopPeeking } from 'datastar'

// Host getters must not subscribe callers (e.g. data-bind's sync effect).
const peek = <T>(fn: () => T): T => {
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
const internals = new WeakMap<HTMLElement, ElementInternals>()
const internalsOf = (host: HTMLElement) => internals.get(host) ?? internals.set(host, host.attachInternals()).get(host)!

// Pixel corners: notches every corner by p (3px times --sb-notch; at 0 the
// border-radius takes over).
const notch = (p: string) => `polygon(${p} 0, calc(100% - ${p}) 0, calc(100% - ${p}) ${p}, 100% ${p}, 100% calc(100% - ${p}), calc(100% - ${p}) calc(100% - ${p}), calc(100% - ${p}) 100%, ${p} 100%, ${p} calc(100% - ${p}), 0 calc(100% - ${p}), 0 ${p}, ${p} ${p})`

// A row from the server: its values by column key. A value is a scalar, or
// a cell object {value, text, suffix, href, tone}.
type Row = Record<string, unknown>
type Cell = { value?: unknown; text?: unknown; suffix?: unknown; href?: unknown; tone?: unknown }
type Align = 'start' | 'center' | 'end'
type Column = { key: string; label: string; width?: unknown; align: Align; sortable: boolean }
type Sort = { key: string; dir: '' | 'asc' | 'desc' }
// A row as rendered, and its index in the server's order.
type Shown = [Row | undefined, number]

// Cells: numbers in the reader's format (4,242.5), isolated so a minus sign
// stays in front in right-to-left text; the rest as text.
const format = new Intl.NumberFormat()
const text = (v: unknown) => (typeof v === 'number' ? '⁨' + format.format(v) + '⁩' : String(v ?? ''))
const isCell = (v: unknown): v is Cell => v !== null && typeof v === 'object' && !Array.isArray(v)
const raw = (v: unknown) => (isCell(v) ? v.value : v)
const tones = new Set<unknown>(['info', 'success', 'warning', 'danger', 'neutral'])
// Only http, https and mailto links: javascript:, data: and the rest stay text.
const link = (href: unknown) => {
	try {
		const u = new URL(String(href), document.baseURI)
		return ['http:', 'https:', 'mailto:'].includes(u.protocol) && u.href
	} catch {
		return false
	}
}
// What a rich row's template shows: t the text, s the suffix, h a checked
// href, n the tone. Nothing here is ever parsed as markup.
type Rich = { t: string; s?: string; h?: string | false; n?: unknown }
const rich = (v: unknown): Rich => {
	if (!isCell(v)) return { t: text(v) }
	const c: Rich = { t: v.text != null ? String(v.text) : text(v.value) }
	if (v.suffix != null && v.suffix !== '') c.s = String(v.suffix)
	if (v.href) c.h = link(v.href)
	if (tones.has(v.tone)) c.n = v.tone
	return c
}
// Local sorting: numbers by value, the rest as text in the reader's order
// (numeric: "Io 2" before "Io 10"); missing values last in both directions.
const collator = new Intl.Collator(undefined, { numeric: true })
const missing = (v: unknown) => v == null || v === ''
const sortKey = (v: unknown) => (isCell(v) ? (v.value ?? v.text) : v)
const keys = (v: unknown): string[] => (Array.isArray(v) ? v.map(String) : [])
const compare =
	({ key, dir }: Sort) =>
	([a]: Shown, [b]: Shown) => {
		const x = sortKey(a?.[key]), y = sortKey(b?.[key])
		if (missing(x) || missing(y)) return Number(missing(x)) - Number(missing(y))
		return (dir === 'desc' ? -1 : 1) * (typeof x === 'number' && typeof y === 'number' ? x - y : collator.compare(String(x), String(y)))
	}
const align = (a: unknown): Align => (a === 'end' || a === 'right' ? 'end' : a === 'center' ? 'center' : 'start')
const anchors = CSS.supports('anchor-name: --a')
// A control in the toolbar slot counts while it shows: not hidden, nor display: none (data-show).
const visible = (slot: HTMLSlotElement | null | undefined) => !!slot?.assignedElements().some((e) => !(e as HTMLElement).hidden && (e as HTMLElement).style.display !== 'none')
// sb-virtual-scroll, which the grid is.
type VirtualScroll = HTMLElement & { scrollToIndex?: (i: number, options?: { block?: string }) => void }

// The windowing is sb-virtual-scroll's: the rows are its children, the header
// row sits in its header slot, and its role="grid" makes it the grid. Every
// row is a grid of the same tracks, at least as wide as their minimums, so the
// columns line up and a wide table scrolls sideways.
const styles = /* css */ `
:host {
	--_shadow: var(--sb-shadow-overlay, 0 12px 24px rgb(0 0 0 / 0.55));
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
.toolbar {
	flex: none;
	display: flex;
	flex-wrap: wrap;
	align-items: center;
	justify-content: flex-end;
	gap: 0.5rem;
	margin-block-end: 0.5rem;
}
/* Buttons the page puts in the toolbar look like the Columns button, unless
   the page styles them. */
.columns, ::slotted(button) {
	all: unset;
	box-sizing: border-box;
	display: inline-flex;
	align-items: center;
	block-size: 2rem;
	padding-inline: 0.75rem;
	border: 1px solid var(--_border);
	background: var(--_head);
	color: var(--_label);
	font-weight: 600;
	cursor: pointer;
	clip-path: ${notch('calc(2px * var(--_notch))')};
	border-radius: calc(var(--_radius) * (1 - var(--_notch)));
}
.columns { anchor-name: --sb-columns; }
.columns:hover, .columns[aria-expanded="true"], ::slotted(button:hover) { color: var(--_text); background: var(--_hover); }
::slotted(button:disabled) { opacity: 0.5; cursor: default; }
::slotted([hidden]) { display: none; }
.menu {
	margin: 0;
	padding: 0;
	border: 0;
	background: none;
	color: var(--_text);
	overflow: visible;
	filter: drop-shadow(var(--_shadow));
}
@supports (anchor-name: --a) {
	.menu { position-anchor: --sb-columns; inset: auto; position-area: block-end span-inline-start; margin-block-start: 4px; position-try-fallbacks: flip-block, flip-inline; }
}
.panel {
	display: grid;
	min-inline-size: 10rem;
	max-block-size: min(20rem, 70vh);
	overflow: auto;
	padding: 0.375rem;
	border: 1px solid var(--_border);
	background: var(--_head);
	clip-path: ${notch('var(--_n)')};
	border-radius: calc(var(--_radius) * (1 - var(--_notch)));
}
.option {
	display: flex;
	align-items: center;
	gap: 0.6rem;
	padding: 0.4rem 0.5rem;
	white-space: nowrap;
	cursor: pointer;
}
.option:hover { background: var(--_hover); }
.option input { flex: none; inline-size: 1rem; block-size: 1rem; margin: 0; accent-color: var(--_brand); cursor: inherit; }
.option:has(:disabled) { color: var(--_muted); cursor: default; }
.option:has(:disabled):hover { background: none; }
.columns:focus-visible, ::slotted(button:focus-visible) { outline: 2px solid var(--_focus); outline-offset: -2px; }
.option input:focus-visible { outline: 2px solid var(--_focus); outline-offset: 2px; }
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
/* The sort arrow, down for descending: faint until the column is the sort.
   Stepped at notch 1, a triangle at 0. */
.th button::after {
	content: "";
	flex: none;
	inline-size: 8px;
	block-size: 6px;
	background: currentColor;
	opacity: 0.35;
	clip-path: polygon(0px 0px, 8px 0px, calc(6.667px + 1.333px * var(--_notch)) 2px, calc(6.667px + -0.667px * var(--_notch)) 2px, calc(5.333px + 0.667px * var(--_notch)) 4px, calc(5.333px + -0.333px * var(--_notch)) 4px, calc(4px + 1px * var(--_notch)) 6px, calc(4px + -1px * var(--_notch)) 6px, calc(2.667px + 0.333px * var(--_notch)) 4px, calc(2.667px + -0.667px * var(--_notch)) 4px, calc(1.333px + 0.667px * var(--_notch)) 2px, calc(1.333px + -1.333px * var(--_notch)) 2px);
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
	.option:has(:disabled) { color: GrayText; }
	.columns:is(:hover, [aria-expanded="true"]) { forced-color-adjust: none; background: Highlight; color: HighlightText; outline-color: HighlightText; }
	.option:not(:has(:disabled)):hover { outline: 1px solid Highlight; outline-offset: -1px; }
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
		name: string.trim.docs({ description: 'Name reported in sb-change, sb-columns and sb-export (e.g. the field of a command).' }),
		hiddenColumns: json.default(() => []).docs({ description: 'View state: the keys of the columns not shown, as a JSON array. A changed list from the server wins; the same list again keeps the user\'s choice. The live list is the hiddenColumns property. At least one column always shows.' }),
		columnPicker: bool.docs({ description: 'Show a Columns button in the toolbar, with a menu that shows and hides columns (sb-columns).' }),
		columnsLabel: string.trim.default('Columns').docs({ description: 'The Columns button\'s text, and the accessible name of its menu.' }),
	}),
	manifest: {
		slots: [{ name: 'toolbar', description: 'Your own controls (e.g. export buttons), in a toolbar above the table, before the Columns button.' }],
		events: [
			{ name: 'sb-window', kind: 'custom-event', bubbles: true, composed: true, description: 'The view needs rows the table doesn\'t have (also on connect and on resize): detail { offset, count }, plus key and dir of the order to send them in. Answer with rows from that offset, offset and total.' },
			{ name: 'sb-sort', kind: 'custom-event', bubbles: true, composed: true, description: 'A sortable header was clicked. detail: { key, dir }, plus the window to answer with (offset 0, count). Answer with those rows in that order, and the new sort; a table that holds every row has sorted them already.' },
			{ name: 'sb-row-activate', kind: 'custom-event', bubbles: true, composed: true, description: 'Enter or a double click on a row (not on a link in it). detail: { key }.' },
			{ name: 'sb-cell-activate', kind: 'custom-event', bubbles: true, composed: true, description: 'A click or middle click on a link in a cell, or Enter on its cell. detail: { key, column, value, href }. Cancelable: preventDefault() stops the navigation, for an in-page action instead.' },
			{ name: 'change', kind: 'event', bubbles: true, composed: true, description: 'The selection changed.' },
			{ name: 'sb-change', kind: 'custom-event', bubbles: true, composed: true, description: 'The selection changed. detail: { name, value } (an array of row keys): ready for a command.' },
			{ name: 'sb-export', kind: 'custom-event', bubbles: true, composed: true, description: 'requestExport(format) was called. detail: { name, format, sort: { key, dir }, columns, selected }: the order on screen, the keys of the shown columns in order, and the local selection. The page asks the server for the file; the table never builds one.' },
			{ name: 'sb-columns', kind: 'custom-event', bubbles: true, composed: true, description: 'The user showed or hid a column in the column picker. detail: { name, hidden } (the keys of every hidden column). View state: not emitted for a change the server made or a property write.' },
		],
	},
	// Rendered once: rows, header and focus all flow through signals, so new
	// rows never rebuild the table (and take the keyboard focus with it).
	renderOnPropChange: false,
	setup: ({ $$, action, adoptStyles, cleanup, defineHostProp, effect, emit, emitCancellable, host, observeProps, overrideProp, props }) => {
		adoptStyles(host, styles)
		const $ = <E extends Element = HTMLElement>(s: string) => host.shadowRoot?.querySelector<E>(s) // in the rendered template
		const root = () => host.shadowRoot!
		// What the server sent, in plain variables (only $$.rows or $$.rich is rendered).
		// asked: the order an sb-sort of ours asked for, until rows in it arrive;
		// olds: the orders it replaced, until `until`; dropped: the rows of the
		// last answer not taken, which stay dropped.
		// shown: the rows as rendered, [row, index in the server's order]; said:
		// the server's last order, so only a new one replaces a local one.
		let data: Row[] = [], off = 0, cols: Column[] = [], sort: Sort = { key: '', dir: '' }, sorted: string | undefined, asked: Sort | null = null, expire = 0
		let olds = new Set<string>(), until = 0, dropped: unknown, shown: Shown[] = [], said: string | undefined
		// all: every column; cols: the ones shown; hidden: the local list (view
		// state); forced: a column shown although the list hides every one.
		let all: Column[] = [], hidden = keys(props.hiddenColumns), forced: string | false | undefined
		const orderOf = (o: Sort) => o.key + ' ' + o.dir
		const keyAt = (i: number) => {
			const [r, j] = shown[i - off] ?? []
			return r && String(raw(r[props.rowKey]) ?? j)
		}
		// Every row is here: the table sorts them itself.
		const whole = () => off === 0 && data.length >= props.total
		const show = () => {
			shown = data.map((r, j): Shown => [r, off + j])
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
			if (sorted !== undefined && orderOf(sort) !== sorted) $<VirtualScroll>('.grid')?.scrollToIndex?.(0)
			sorted = orderOf(sort)
		}
		$$.rows = '[]'
		$$.rich = '[]'
		$$.fr = -1 // the focus cell (roving tabindex): row (-1 the header), column
		$$.fc = 0
		$$.hasFocus = false
		$$.menu = false
		$$.slotted = false
		// slotchange doesn't report a control hidden or shown in place.
		const tools = new MutationObserver(() => ($$.slotted = visible($<HTMLSlotElement>('slot[name="toolbar"]'))))
		tools.observe(host, { subtree: true, attributeFilter: ['hidden', 'style'] })
		cleanup(() => tools.disconnect())
		action('toolbar', ({ el }) => ($$.slotted = visible(el as HTMLSlotElement)))

		// The menu's checkboxes follow the columns. A focused one stays with its
		// column; when the server drops that column, the neighbour takes the focus.
		const options = (before: Column[]) => {
			const menu = $('.menu')
			const box = menu?.matches(':popover-open') ? (root().activeElement as HTMLElement | null) : null
			const was = box && menu!.contains(box) && box.dataset.k != null ? { key: before[+box.dataset.k]?.key, j: +box.dataset.k } : null
			const on = new Set(cols.map((c) => c.key))
			$$.opts = JSON.stringify(all.map((c) => ({ l: c.label, h: !on.has(c.key), d: cols.length === 1 && on.has(c.key) })))
			if (was && menu)
				requestAnimationFrame(() => {
					const boxes = [...menu.querySelectorAll('input')]
					let j = all.findIndex((c) => c.key === was.key)
					if (j < 0) j = Math.min(was.j, boxes.length - 1)
					// A disabled neighbour is skipped; with none left, the menu holds the focus.
					const to = (boxes[j]?.disabled ? (boxes[j + 1] ?? boxes[j - 1]) : boxes[j]) ?? menu
					const at = root().activeElement, floor = !at && (!document.activeElement || document.activeElement === document.body)
					if (to !== at && (menu.contains(at) || floor)) to.focus()
				})
		}
		const layout = () => {
			const before = all
			all = (Array.isArray(props.columns) ? props.columns : [])
				.filter((c) => c?.key != null)
				.map((c) => ({ key: String(c.key), label: String(c.label ?? c.key), width: c.width, align: align(c.align), sortable: !!c.sortable }))
			const vis = all.filter((c) => !hidden.includes(c.key))
			forced = !vis.length && all[0]?.key
			cols = vis.length ? vis : all.slice(0, 1)
			$$.al = cols.map((c) => c.align)
			$$.tpl = cols.map((c) => (typeof c.width === 'number' ? c.width + 'px' : c.width || 'minmax(6rem, 1fr)')).join(' ')
			$$.fc = Math.max(0, Math.min($$.fc, cols.length - 1))
			$$.picker = props.columnPicker
			$$.clabel = props.columnsLabel
			if (!props.columnPicker && $('.menu')?.matches(':popover-open')) $('.menu')!.hidePopover()
			options(before)
		}

		// Moves the DOM focus to the tab stop (the header row while its row is
		// not rendered: a tabindex on the list would take its rows out of the
		// tab order), also after the rows re-rendered: only when the focus is in
		// the table or fell on the floor, never when the user moved on.
		const refocus = (scroll?: boolean) =>
			requestAnimationFrame(() => {
				const active = document.activeElement, inner = root().activeElement
				if (!$$.hasFocus || (active && active !== document.body && active !== host) || (inner && !$('.grid')!.contains(inner))) return
				const cell = $('.grid [tabindex="0"]')
				const el = cell ?? $('.head')
				if (el && root().activeElement !== el) el.focus({ preventScroll: true })
				if (scroll && cell && $$.fr >= 0) cell.scrollIntoView({ block: 'nearest', inline: 'nearest' })
			})

		const shape = () => cols.map((c) => c.key).join('\n')
		const take = () => {
			const was = shape()
			layout()
			$$.label = props.label
			$$.mode = props.selection
			$$.h = props.rowHeight
			$$.buf = props.buffer
			$$.loading = props.loading
			const key = String(props.sort?.key ?? '')
			const now: Sort = { key, dir: !key ? '' : props.sort.dir === 'desc' ? 'desc' : 'asc' }
			const order = orderOf(now)
			// While our sb-sort is unanswered, only its order is taken, and for a
			// while after it rows in an order it replaced are late answers to
			// earlier windows, which would put that order back. An order the
			// server sends on its own wins.
			if (asked ? order !== orderOf(asked) : olds.has(order) && performance.now() < until) {
				dropped = props.rows
				// Columns shown or hidden meanwhile show at once, with the rows at hand.
				if (shape() !== was) show()
			} else if (props.rows !== dropped) {
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
		const later = () => queued || ((queued = true), queueMicrotask(() => peek(() => ((queued = false), take()))))
		observeProps(later)
		cleanup(() => clearTimeout(expire))

		// Selection: $$.sel is the local one, the attribute the server's.
		$$.sel = keys(props.selected)
		overrideProp('selected', () => peek(() => [...$$.sel]), (v) => peek(() => ($$.sel = keys(v))))
		// Hidden columns are view state: never pending, and revert() leaves them.
		overrideProp('hiddenColumns', () => peek(() => [...hidden]), (v) => peek(() => ((hidden = keys(v)), take())))
		// A new selected or hidden-columns attribute from the server wins, even [] on a fresh element;
		// the same one again keeps the user's choice, and a removed one is ignored (morphs strip reflected ones).
		const words: Record<string, string | null> = {}
		const heard = (k: string) => {
			const a = host.getAttribute(k)
			const news = a !== null && a !== words[k]
			words[k] = a
			return news
		}
		heard('selected'), heard('hidden-columns')
		const watch = new MutationObserver(() =>
			peek(() => {
				if (heard('selected')) $$.sel = keys(props.selected)
				if (heard('hidden-columns')) (hidden = keys(props.hiddenColumns)), later()
			}),
		)
		watch.observe(host, { attributeFilter: ['selected', 'hidden-columns'] })
		cleanup(() => watch.disconnect())
		// With confirm, :state(pending) marks a selection the server hasn't
		// confirmed yet (compared as sets); revert() returns to the server's.
		const states = internalsOf(host).states
		const same = (a: string[], b: string[]) => a.toSorted().join('\n') === b.toSorted().join('\n')
		// keys(): the teardown deletes $$.sel, then runs the effect below once more.
		const sync = () => peek(() => states[props.confirm && !same(keys($$.sel), keys(props.selected)) ? 'add' : 'delete']('pending'))
		effect(() => (JSON.stringify($$.sel), sync()))
		observeProps(sync)
		defineHostProp('revert', { value: () => peek(() => (($$.sel = keys(props.selected)), sync())) })
		// The server has the rows and builds the file: the table only says what
		// is on screen (its order, not an unanswered sb-sort) and what is picked.
		defineHostProp('requestExport', {
			value: (format = 'csv') => peek(() => emit('sb-export', { name: props.name, format: String(format), sort: { key: sort.key, dir: sort.dir }, columns: cols.map((c) => c.key), selected: keys($$.sel) })),
		})

		const pick = (i: number, toggle: boolean) => {
			const k = keyAt(i)
			if (props.selection === 'none' || k == null) return
			const has = $$.sel.includes(k)
			if (has && !toggle) return
			$$.sel = props.selection === 'multiple' ? (has ? $$.sel.filter((x: string) => x !== k) : [...$$.sel, k]) : has ? [] : [k]
			emit('change')
			emit('sb-change', { name: props.name, value: [...$$.sel] })
		}
		const activate = (i: number) => {
			const k = keyAt(i)
			if (k != null) emit('sb-row-activate', { key: k })
		}

		// Rows in view: the list's height minus the header's.
		const inView = () => Math.max(1, Math.floor(($('.grid')!.clientHeight - $('.head')!.offsetHeight) / props.rowHeight))

		// The list asks for a window: ask the page, with the order to send it in.
		action('window', ({ evt }) => {
			const e = evt as CustomEvent<{ offset: number; count: number }>
			e.stopPropagation()
			emit('sb-window', { offset: e.detail.offset, count: e.detail.count, ...(asked ?? sort) })
		})
		action('sort', (_, j: number) => {
			const c = cols[j]
			$$.fr = -1
			$$.fc = j
			if (!c?.sortable) return
			const next: Sort = { key: c.key, dir: sort.key === c.key && sort.dir === 'asc' ? 'desc' : 'asc' }
			// The window to answer with: the top one, the view and its buffers.
			const count = inView() + 1 + 2 * Math.ceil(props.buffer / props.rowHeight)
			// With every row here, sort them now; the page still hears of it.
			if (whole()) return (sort = next), show(), top(), refocus(), emit('sb-sort', { ...next, offset: 0, count })
			olds.add(orderOf(sort)).add(orderOf(asked ?? sort))
			asked = next
			olds.delete(orderOf(next))
			until = performance.now() + 10000
			// A sort that never comes back (or comes back in another order) stops
			// holding the rows up after a while.
			clearTimeout(expire)
			expire = setTimeout(() => peek(() => ((asked = null), take())), 5000)
			emit('sb-sort', { ...asked, offset: 0, count })
		})
		// Clicks on rows, heard on the grid (a cell's focusin moved the focus
		// there). The clicks of a double click don't select twice.
		const rowOf = (evt: Event) => (evt.target as Element).closest<HTMLElement>('.row')?.dataset.r
		const follow = (evt: Event, a: HTMLAnchorElement) => {
			const i = +a.closest<HTMLElement>('.row')!.dataset.r!, c = cols[+a.closest<HTMLElement>('[data-c]')!.dataset.c!]
			const v = shown[i - off]?.[0]?.[c?.key]
			if (c && !emitCancellable('sb-cell-activate', { key: keyAt(i), column: c.key, value: raw(v) ?? null, href: a.href })) evt.preventDefault()
		}
		action('click', ({ evt }) => {
			const e = evt as MouseEvent
			const a = (e.target as Element).closest<HTMLAnchorElement>('a[href]')
			if (a) return follow(e, a)
			e.detail < 2 && rowOf(e) && pick(+rowOf(e)!, props.selection === 'multiple')
		})
		// A middle click opens a link in a new tab: the page may cancel that too.
		action('aux', ({ evt }) => { const e = evt as MouseEvent, a = e.button === 1 && (e.target as Element).closest<HTMLAnchorElement>('a[href]'); a && follow(e, a) })
		action('activate', ({ evt }) => !(evt!.target as Element).closest('a[href]') && rowOf(evt!) && activate(+rowOf(evt!)!))
		// Focus can also arrive by Tab or a click: keep the roving focus in step.
		action('focusin', ({ evt }) => {
			$$.hasFocus = true
			const c = (evt!.target as Element).closest?.<HTMLElement>('[data-c]')
			if (c) ($$.fr = +c.closest<HTMLElement>('[data-r]')!.dataset.r!), ($$.fc = +c.dataset.c!)
		})
		action('focusout', ({ evt }) => {
			// A cell re-rendered away also "loses" focus, but that's not leaving
			// (refocus picks it up): decide a frame later, when it is gone.
			const t = evt!.target as Element
			requestAnimationFrame(() => t.isConnected && !$('.grid')?.contains(host.shadowRoot?.activeElement ?? null) && ($$.hasFocus = false))
		})

		// The column menu: a popover (auto: light dismiss and Escape are the
		// browser's) that holds the focus like a dialog.
		action('menu', ({ el, evt }) =>
			peek(() => {
				const menu = el as HTMLElement, button = $('.columns')!
				// Closing with the focus inside: it goes back to the button, or to the
				// grid when the server took the picker away.
				if ((evt as ToggleEvent).newState !== 'open') return menu.contains(root().activeElement) && (props.columnPicker ? button : ($('.grid [tabindex="0"]') ?? $('.head')))!.focus()
				if (anchors) return
				const r = button.getBoundingClientRect()
				const rtl = host.matches(':dir(rtl)')
				Object.assign(menu.style, { position: 'fixed', inset: 'auto', top: r.bottom + 4 + 'px', [rtl ? 'left' : 'right']: (rtl ? r.left : document.documentElement.clientWidth - r.right) + 'px' })
			}),
		)
		action('menuToggle', ({ el, evt }) =>
			peek(() => {
				$$.menu = (evt as ToggleEvent).newState === 'open'
				if ($$.menu) (el!.querySelector<HTMLElement>('input:not(:disabled)') ?? (el as HTMLElement)).focus()
			}),
		)
		action('menuKey', ({ el, evt }) => {
			// The browser closes the menu. Stopped here, the Escape can't also
			// close a drawer or popover the table sits in.
			const e = evt as KeyboardEvent
			if (e.key === 'Escape') return e.stopPropagation()
			if (e.key !== 'Tab') return
			e.preventDefault()
			const f = [...el!.querySelectorAll<HTMLInputElement>('input:not(:disabled)')]
			const i = f.indexOf(root().activeElement as HTMLInputElement)
			f[i < 0 ? (e.shiftKey ? f.length - 1 : 0) : (i + (e.shiftKey ? -1 : 1) + f.length) % f.length]?.focus()
		})
		// A checkbox: the column shows or hides at once, and the page hears of
		// it. The list is the user's view, so a forced column counts as shown.
		action('pickColumn', ({ evt }) =>
			peek(() => {
				const box = evt!.target as HTMLInputElement, c = all[+box.dataset.k!]
				if (!c) return
				hidden = hidden.filter((k) => k !== c.key && k !== forced)
				if (!box.checked) hidden.push(c.key)
				take()
				emit('sb-columns', { name: props.name, hidden: [...hidden] })
			}),
		)
		// The grid keys of WAI-ARIA; moves past the rendered rows scroll there,
		// and the rows come with the next window.
		action('key', ({ evt: e }) => {
			const evt = e as KeyboardEvent, target = evt.target as Element
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
					if (r < 0) return target.localName === 'button' || evt.preventDefault()
					const a = evt.key === 'Enter' && target.closest('[data-c]')?.querySelector<HTMLAnchorElement>('a[href]')
					a ? a.click() : evt.key === 'Enter' ? activate(r) : pick(r, true)
					return evt.preventDefault()
				}
				default:
					return
			}
			evt.preventDefault()
			$$.fr = r = Math.max(-1, Math.min(r, $$.n - 1))
			$$.fc = Math.max(0, Math.min(c, last))
			if (r >= 0) $<VirtualScroll>('.grid')!.scrollToIndex?.(r, { block: 'nearest' })
			refocus(true)
		})
	},
	// Runs on every connect, after the render.
	onFirstRender: ({ $$, refs }) => {
		const { grid, parts, tools } = refs as { grid: HTMLElement; parts: HTMLElement; tools: HTMLSlotElement }
		grid.append(...parts.childNodes)
		$$.slotted = visible(tools)
	},
	render: ({ html }) => {
		// $$rows (plain cells) or $$rich (cell objects): the same rows either way.
		// r?. and || '[]': data-for can re-evaluate once with its signals gone.
		const loop = (list: string, content: DocumentFragment | null) => html`
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
		<div class="toolbar" part="toolbar" data-show="$$picker || $$slotted">
			<slot name="toolbar" data-ref:tools data-on:slotchange="@toolbar()"></slot>
			<button type="button" class="columns" part="columns-button" popovertarget="columns" aria-haspopup="dialog" aria-controls="columns"
				data-show="$$picker" data-attr:aria-expanded="String(!!$$menu)" data-text="$$clabel"></button>
		</div>
		<div id="columns" class="menu" part="columns-menu" popover="auto" role="dialog" aria-modal="true" tabindex="-1"
			data-attr:aria-label="$$clabel"
			data-on:beforetoggle="@menu()" data-on:toggle="@menuToggle()" data-on:keydown="@menuKey()"
			data-on:input="evt.stopPropagation()" data-on:change="@pickColumn()">
			<div class="panel">
				<template data-for="o, j in JSON.parse($$opts || '[]')">
					<label class="option" part="column-option"><input type="checkbox" data-attr="{'data-k': j, disabled: o?.d}" data-effect="el.checked = !o?.h"><span data-text="o?.l"></span></label>
				</template>
			</div>
		</div>
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
