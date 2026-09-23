import { rocket, startPeeking, stopPeeking } from 'datastar'

// Host getters must not subscribe callers (e.g. a data-attr effect that set an
// attribute) to our internal signals.
const peek = (fn) => {
	startPeeking()
	try {
		return fn()
	} finally {
		stopPeeking()
	}
}

// Pixel corners: notches every corner by p (2px times --sb-notch; at 0 the
// border-radius takes over).
const notch = (p) => `polygon(${p} 0, calc(100% - ${p}) 0, calc(100% - ${p}) ${p}, 100% ${p}, 100% calc(100% - ${p}), calc(100% - ${p}) calc(100% - ${p}), calc(100% - ${p}) 100%, ${p} 100%, ${p} calc(100% - ${p}), 0 calc(100% - ${p}), 0 ${p}, ${p} ${p})`

// Case- and accent-insensitive matching for the type-ahead.
const fold = (s) => s.normalize('NFD').replace(/\p{Diacritic}/gu, '').toLowerCase()

const anchors = typeof CSS !== 'undefined' && CSS.supports?.('anchor-name: --a')

// Items are strings, "-" for a divider, or
// {value, label?, description?, icon?, disabled?, danger?, divider?}.
const item = (it) => {
	if (it === null || it === undefined) return null
	if (typeof it !== 'object') {
		const s = String(it)
		return s === '-' || s === '---' ? { divider: true } : { divider: false, value: s, label: s, description: '', icon: '', disabled: false, danger: false }
	}
	if (it.divider) return { divider: true }
	const value = String(it.value ?? it.label ?? '')
	return {
		divider: false,
		value,
		label: String(it.label ?? it.value ?? ''),
		description: it.description ? String(it.description) : '',
		icon: it.icon ? String(it.icon) : '',
		disabled: !!it.disabled,
		danger: !!it.danger,
	}
}
const normalize = (list) => (Array.isArray(list) ? list : []).map(item).filter(Boolean)

const styles = /* css */ `
:host {
	--_bg: var(--sb-control-bg, #0B1224);
	--_border: var(--sb-control-border, #283552);
	--_border-hover: var(--sb-control-border-hover, #3A4868);
	--_text: var(--sb-text-1, #F3F4FA);
	--_muted: var(--sb-text-muted, #7785A8);
	--_panel: var(--sb-surface-raised, #141D32);
	--_hover: var(--sb-surface-hover, #1A2540);
	--_brand: var(--sb-brand, #8C6BFF);
	--_brand-light: var(--sb-brand-light, #B09AFF);
	--_brand-subtle: var(--sb-brand-subtle, rgb(140 107 255 / 0.14));
	--_danger: var(--sb-danger, #F2777A);
	--_radius: var(--sb-control-radius, 6px);
	--_notch: var(--sb-notch, 1);
	--_n: calc(2px * var(--_notch));
	display: inline-block;
	vertical-align: middle;
}
:host([disabled]) { opacity: 0.5; }

/* Trigger: a notched plate with a pixel caret. */
.trigger {
	all: unset;
	box-sizing: border-box;
	display: inline-flex;
	align-items: center;
	gap: 0.5em;
	min-block-size: 2.5rem;
	padding-inline: 0.9rem;
	background: var(--_bg);
	box-shadow: inset 0 0 0 1px var(--_border);
	clip-path: ${notch('var(--_n)')};
	border-radius: calc(var(--_radius) * (1 - var(--_notch)));
	color: var(--_text);
	font: inherit;
	font-size: 0.875rem;
	font-weight: 600;
	line-height: 1;
	white-space: nowrap;
	cursor: pointer;
	anchor-name: --sb-dropdown;
	transition: box-shadow 120ms, background 120ms;
}
.trigger:hover { box-shadow: inset 0 0 0 1px var(--_border-hover); background: var(--_hover); }
.trigger:focus-visible { box-shadow: inset 0 0 0 2px var(--_brand-light); }
.trigger[aria-expanded="true"] { box-shadow: inset 0 0 0 1px var(--_brand-light); background: var(--_hover); }
.trigger[disabled] { cursor: default; }
.caret {
	inline-size: 8px;
	block-size: 6px;
	flex: none;
	background: currentColor;
	opacity: 0.7;
	clip-path: polygon(0 0, 8px 0, 8px 2px, 6px 2px, 6px 4px, 4px 4px, 4px 6px, 2px 6px, 2px 4px, 0 4px, 0 2px);
	transition: rotate 120ms steps(2, end);
}
.trigger[aria-expanded="true"] .caret { rotate: 180deg; }

/* The menu is a top-layer popover, so no ancestor can clip it. */
[popover] {
	position: fixed;
	inset: auto;
	margin: 0;
	box-sizing: border-box;
	min-inline-size: 10rem;
	max-inline-size: min(22rem, 100vw - 1rem);
	max-block-size: min(20rem, 60dvh);
	overflow: auto;
	padding: 4px;
	border: 0;
	background: var(--_panel);
	box-shadow: inset 0 0 0 1px var(--_border);
	clip-path: ${notch('var(--_n)')};
	border-radius: calc(var(--_radius) * (1 - var(--_notch)));
	/* A filter shadow follows the notched silhouette; box-shadow would be clipped. */
	filter: drop-shadow(0 12px 24px rgb(0 0 0 / 0.55));
	color: var(--_text);
	font-size: 0.875rem;
}
/* CSS anchor positioning where it exists; place() does the same in JS elsewhere. */
@supports (anchor-name: --a) {
	[popover] {
		position-anchor: --sb-dropdown;
		min-inline-size: anchor-size(width);
		position-try-fallbacks: flip-block, flip-inline;
	}
	[data-place="bottom-start"] { position-area: block-end span-inline-end; margin-block-start: 4px; }
	[data-place="bottom"] { position-area: block-end center; margin-block-start: 4px; }
	[data-place="bottom-end"] { position-area: block-end span-inline-start; margin-block-start: 4px; }
	[data-place="top-start"] { position-area: block-start span-inline-end; margin-block-end: 4px; }
	[data-place="top"] { position-area: block-start center; margin-block-end: 4px; }
	[data-place="top-end"] { position-area: block-start span-inline-start; margin-block-end: 4px; }
}
.anim:popover-open { animation: pop 110ms cubic-bezier(0.2, 0, 0, 1); }
@keyframes pop { from { opacity: 0; translate: 0 -3px; } }

[role="menuitem"] {
	display: flex;
	align-items: center;
	gap: 0.5rem;
	padding: 0.4rem 0.6rem;
	border-radius: calc((var(--_radius) - 2px) * (1 - var(--_notch)));
	cursor: pointer;
	outline: none;
}
[role="menuitem"]:hover { background: var(--_hover); }
[role="menuitem"]:focus { background: var(--_hover); box-shadow: inset 2px 0 0 var(--_brand); }
[role="menuitem"]:focus-visible { outline: 2px solid var(--_brand-light); outline-offset: -2px; }
[role="menuitem"][aria-disabled="true"] { opacity: 0.45; cursor: default; }
[role="menuitem"][aria-disabled="true"]:hover { background: none; }
.danger { color: var(--_danger); }
.danger:focus { box-shadow: inset 2px 0 0 var(--_danger); }
.icon { flex: none; inline-size: 1.15rem; text-align: center; }
.body { min-inline-size: 0; }
.label { font-weight: 600; }
.desc { color: var(--_muted); font-size: 0.75rem; font-weight: 400; }
.label, .desc { display: block; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }

[role="separator"] { block-size: 1px; margin: 4px 2px; padding: 0; background: var(--_border); }
.empty { padding: 0.4rem 0.6rem; color: var(--_muted); }
/* Slotted items are read as data and re-rendered inside the menu. */
slot[name="item"] { display: none; }
@media (prefers-reduced-motion: reduce) {
	.anim:popover-open { animation: none; }
	.caret, .trigger { transition: none; }
}
`

rocket('sb-dropdown', {
	props: ({ bool, json, oneOf, string }) => ({
		items: json.default(() => []).docs({ description: 'The menu, as JSON: ["Rename", "-", {"value":"delete","label":"Delete","danger":true}]. Items are {value, label?, description?, icon?, disabled?, danger?} or {"divider":true} ("-" works too). Server data: a new array from the server replaces the menu, open or not.' }),
		label: string.trim.default('Actions').docs({ description: 'Text of the default trigger, and the accessible name of trigger and menu.' }),
		placement: oneOf('bottom-start', 'bottom', 'bottom-end', 'top-start', 'top', 'top-end').default('bottom-start').docs({ description: 'Preferred side and alignment of the menu; it flips and shifts when there is no room.' }),
		open: bool.docs({ description: 'Open on first render. A changed attribute from the server opens or closes the menu (open="false" closes); re-sent identical markup leaves the local state alone. Never reflected: use the open property, show() and hide() from the client.' }),
		disabled: bool.docs({ description: 'Disable the trigger (and close the menu).' }),
		name: string.trim.docs({ description: 'Name reported in sb-select (e.g. the field of a command).' }),
	}),
	manifest: {
		slots: [
			{ name: 'trigger', description: 'Content of the trigger button (text, an icon). The component provides the button itself, with all the ARIA on it.' },
			{ name: 'item', description: 'Menu items as markup instead of items: <button slot="item" value="x" disabled data-icon="🛰" data-description="…" data-danger>Label</button>, or <hr slot="item"> for a divider. They are read as data; items wins when it is not empty.' },
		],
		events: [
			{ name: 'sb-select', kind: 'custom-event', bubbles: true, composed: true, description: 'An item was chosen. detail: { name, value }: ready for a command.' },
			{ name: 'sb-open', kind: 'custom-event', bubbles: true, composed: true, description: 'The menu opened.' },
			{ name: 'sb-close', kind: 'custom-event', bubbles: true, composed: true, description: 'The menu closed. detail: { reason }: item, escape, outside, scroll, tab, trigger, server or api.' },
		],
	},
	// Rendered once: the items, the open state and the focused row all flow
	// through signals, so new items never rebuild the trigger or take the focus.
	renderOnPropChange: false,
	setup: ({ $$, action, adoptStyles, cleanup, defineHostProp, effect, emit, host, observeProps, overrideProp, props }) => {
		adoptStyles(host, styles)
		// Open/closed, the focused row and the type-ahead buffer are local: they
		// are never written back to an attribute, which a morph would reset anyway.
		$$.open = props.open
		$$.active = -1 // index into $$.rows, -1 for none
		$$.inside = false // keyboard focus is in the menu
		$$.rows = []
		$$.icons = false
		$$.label = props.label
		$$.place = props.placement
		$$.disabled = props.disabled
		$$.anim = false // no opening animation for a menu that starts open

		const menu = () => host.shadowRoot?.querySelector('[popover]')
		const trigger = () => host.shadowRoot?.querySelector('.trigger')

		// Markup items are read as data (label, value, flags) and rendered inside
		// the menu: we never write roles or tabindex into the page's own DOM,
		// where the next morph would strip them.
		const slotted = () =>
			(host.shadowRoot?.querySelector('slot[name="item"]')?.assignedElements() ?? []).map((el) =>
				el.localName === 'hr' || el.hasAttribute('data-divider')
					? { divider: true }
					: {
							value: el.getAttribute('value') ?? el.textContent.trim(),
							label: el.textContent.trim(),
							description: el.getAttribute('data-description') ?? '',
							icon: el.getAttribute('data-icon') ?? '',
							disabled: el.hasAttribute('disabled') || el.getAttribute('aria-disabled') === 'true',
							danger: el.hasAttribute('data-danger'),
						},
			)

		const pickable = (i) => {
			const r = $$.rows[i]
			return !!r && !r.divider && !r.disabled
		}
		const step = (from, dir) => {
			const n = $$.rows.length
			for (let k = 1; k <= n; k++) {
				const i = (from + dir * k + n * (k + 1)) % n
				if (pickable(i)) return i
			}
			return -1
		}
		const edge = (dir) => step(dir > 0 ? -1 : 0, dir)

		// Move DOM focus back onto the focused row, also after the rows
		// re-rendered (new items from the server replace the elements).
		const refocus = () =>
			requestAnimationFrame(() => {
				if (!$$.open || !$$.inside || $$.active < 0) return
				const el = menu()?.querySelector(`[data-idx="${$$.active}"]`)
				if (el && host.shadowRoot.activeElement !== el) el.focus()
			})
		const focusRow = (i) => {
			if (i < 0) return
			$$.active = i
			$$.inside = true
			refocus()
		}

		let shown = '' // what the rows currently render, to skip idle rebuilds
		const rebuild = () => {
			const rows = normalize(props.items?.length ? props.items : slotted())
			const key = JSON.stringify(rows)
			if (key === shown) return
			shown = key
			// Whether the menu had the focus has to be read before the rows go:
			// removing the focused row fires focusout on the way out.
			const had = $$.inside
			$$.rows = rows
			// One icon column for all rows as soon as one item has an icon.
			$$.icons = rows.some((r) => r.icon)
			// Keep the focus on a row that still exists and can be chosen.
			if (!pickable($$.active)) $$.active = had && $$.open ? edge(1) : -1
			$$.inside = had
			refocus()
		}
		rebuild()
		// Markup items change without a slotchange when a morph only rewrites a
		// label or a flag: no attribute form for watching that.
		const watch = new MutationObserver(() => peek(rebuild))
		watch.observe(host, { childList: true, subtree: true, characterData: true, attributes: true, attributeFilter: ['value', 'disabled', 'aria-disabled', 'slot', 'data-icon', 'data-description', 'data-danger', 'data-divider'] })

		const place = () => {
			const m = menu()
			const t = trigger()
			if (anchors || !m || !t) return
			const r = t.getBoundingClientRect()
			m.style.minInlineSize = r.width + 'px'
			const w = m.offsetWidth
			const h = m.offsetHeight
			const gap = 4
			const pad = 8
			const [side, align = 'center'] = props.placement.split('-')
			const fits = (y) => y >= pad && y + h <= innerHeight - pad
			let top = side === 'top' ? r.top - h - gap : r.bottom + gap
			// Flip to the other side when this one has no room.
			if (!fits(top)) {
				const other = side === 'top' ? r.bottom + gap : r.top - h - gap
				if (fits(other)) top = other
			}
			let left = align === 'end' ? r.right - w : align === 'center' ? r.left + (r.width - w) / 2 : r.left
			// Shift back into the viewport.
			left = Math.min(Math.max(pad, left), Math.max(pad, innerWidth - w - pad))
			top = Math.min(Math.max(pad, top), Math.max(pad, innerHeight - h - pad))
			m.style.left = Math.round(left) + 'px'
			m.style.top = Math.round(top) + 'px'
		}

		// No attribute form for these: a click anywhere in the document, and the
		// scrolling of any ancestor while we position by hand.
		const onDown = (evt) => {
			if (!evt.composedPath().includes(host)) close('outside', false)
		}
		// Positioning by hand: follow the trigger, and give up when it has
		// scrolled out of sight (anchor positioning does this for us elsewhere).
		const onMove = () => {
			const r = trigger()?.getBoundingClientRect()
			if (r && (r.bottom < 0 || r.top > innerHeight)) return close('scroll', false)
			place()
		}
		let bound = false
		const bind = (on) => {
			if (on === bound) return
			bound = on
			if (on) {
				document.addEventListener('pointerdown', onDown, true)
				if (!anchors) {
					window.addEventListener('scroll', onMove, { capture: true, passive: true })
					window.addEventListener('resize', onMove)
				}
			} else {
				document.removeEventListener('pointerdown', onDown, true)
				window.removeEventListener('scroll', onMove, { capture: true })
				window.removeEventListener('resize', onMove)
			}
		}

		let type = '' // type-ahead buffer
		let typer = 0
		const setOpen = (next, { focus = null, reason = 'api', defer = false } = {}) => {
			if (next && props.disabled) return
			if (!!next === !!$$.open) return
			clearTimeout(typer)
			type = ''
			// Opening animates from here on; a menu the server rendered open does
			// not (the class has to be there before the popover shows, or adding
			// it later would start the animation a frame too late).
			if (next) $$.anim = true
			$$.open = !!next
			$$.inside = !!(next && focus)
			$$.active = next ? (focus ? edge(focus === 'last' ? -1 : 1) : -1) : -1
			const fire = () => emit(next ? 'sb-open' : 'sb-close', next ? undefined : { reason })
			// An open change that came from the server arrives inside a morph's
			// effect: emit in a later task, so a page's @post isn't tracked by it.
			defer ? setTimeout(fire) : fire()
		}
		const close = (reason, back = true) => {
			if (!$$.open) return
			setOpen(false, { reason })
			if (back) trigger()?.focus()
		}
		const choose = (i) => {
			const r = $$.rows[i]
			if (!pickable(i)) return
			// A menu holds no value, so nothing is pending: the command is an
			// intent, and the page shows whatever the server renders next.
			emit('sb-select', { name: props.name, value: r.value })
			close('item')
		}

		// The open attribute: the first one sets the initial state, a changed one
		// from the server wins over the local state (open="false" closes). A
		// *removed* attribute is ignored: morphs also strip attributes that were
		// only reflected. served is the last value the server stated, so a
		// re-written identical attribute leaves the local state alone: a morph
		// can never re-open a menu the user just closed.
		let served = props.open
		observeProps((p, changes) =>
			peek(() => {
				if ('open' in changes && host.hasAttribute('open') && p.open !== served) {
					served = p.open
					setOpen(p.open, { reason: 'server', defer: true })
				}
				if (p.disabled && $$.open) setOpen(false, { reason: 'api', defer: true })
				$$.label = p.label
				$$.place = p.placement
				$$.disabled = p.disabled
				rebuild()
			}),
		)

		// host.open / show() / hide(): the live state, never an attribute.
		overrideProp('open', () => peek(() => $$.open), (v) => peek(() => setOpen(!!v && v !== 'false', { focus: v ? 'first' : null })))
		defineHostProp('show', { value: () => peek(() => setOpen(true, { focus: 'first' })) })
		defineHostProp('hide', { value: () => peek(() => close('api', false)) })

		cleanup(() => {
			bind(false)
			watch.disconnect()
			clearTimeout(typer)
			try {
				menu()?.hidePopover()
			} catch {}
		})

		// Shows and hides the popover, positions it and keeps the document
		// listeners in step. A plain effect, not an action in data-effect: an
		// action would be looked up on a host that may already be gone.
		const apply = (open) => {
			const m = menu()
			if (!m) return // before the first render
			try {
				if (open) {
					if (!m.matches(':popover-open')) m.showPopover()
					place()
					refocus()
				} else if (m.matches(':popover-open')) {
					m.hidePopover()
				}
			} catch {}
			bind(open)
		}
		effect(() => {
			const open = $$.open
			peek(() => apply(open))
		})
		// The effect's first run happens before the first render, so a menu the
		// server rendered open is shown right after it (setup, then render, then
		// this microtask).
		queueMicrotask(() => peek(() => $$.open && apply(true)))
		action('items', () => peek(rebuild))
		action('toggle', () => peek(() => ($$.open ? close('trigger', false) : setOpen(true, { focus: 'first' }))))
		action('choose', (_, i) => peek(() => choose(i)))
		action('focusin', ({ evt }) =>
			peek(() => {
				$$.inside = true
				const i = Number(evt.target?.dataset?.idx)
				if (Number.isInteger(i)) $$.active = i
			}),
		)
		action('focusout', ({ el, evt }) =>
			peek(() => {
				// A row that re-rendered away also "loses" focus, and focus that
				// goes nowhere (relatedTarget null) is usually that: not leaving.
				if (!evt.target.isConnected || !evt.relatedTarget) return
				if (!el.contains(evt.relatedTarget)) $$.inside = false
			}),
		)
		// Keys on the trigger: open downwards or upwards, or close.
		action('triggerKey', ({ evt }) =>
			peek(() => {
				if (evt.key === 'ArrowDown' || evt.key === 'ArrowUp') setOpen(true, { focus: evt.key === 'ArrowUp' ? 'last' : 'first' })
				else if (evt.key === 'Escape' && $$.open) close('escape')
				else return
				evt.preventDefault()
			}),
		)
		// Keys in the menu. Enter and Space choose, Escape and Tab close, the
		// arrows and printable characters move.
		action('key', ({ evt }) =>
			peek(() => {
				const i = $$.active
				switch (evt.key) {
					case 'ArrowDown':
						focusRow(step(i, 1))
						break
					case 'ArrowUp':
						focusRow(step(i, -1))
						break
					case 'Home':
						focusRow(edge(1))
						break
					case 'End':
						focusRow(edge(-1))
						break
					case 'Enter':
					case ' ':
						choose(i)
						break
					case 'Escape':
						close('escape')
						break
					case 'Tab':
						// Close and hand the focus back, then let the browser tab on.
						close('tab')
						return
					default: {
						if (evt.key.length !== 1 || evt.altKey || evt.ctrlKey || evt.metaKey) return
						clearTimeout(typer)
						typer = setTimeout(() => (type = ''), 700)
						type += fold(evt.key)
						const n = $$.rows.length
						// Search from the row after the focused one, and wrap.
						for (let k = 1; k <= n; k++) {
							const j = (i + k + n) % n
							if (pickable(j) && fold($$.rows[j].label).startsWith(type)) {
								focusRow(j)
								break
							}
						}
						break
					}
				}
				evt.preventDefault()
			}),
		)
	},
	render: ({ html }) => html`
		<button class="trigger" part="trigger" type="button" aria-haspopup="menu" aria-controls="menu"
			data-attr:aria-expanded="String($$open)"
			data-attr:aria-label="$$label || null"
			data-attr:disabled="$$disabled"
			data-on:click="@toggle()"
			data-on:keydown="@triggerKey()"><slot name="trigger"><span data-text="$$label"></span></slot><span class="caret" aria-hidden="true"></span></button>
		<div id="menu" part="menu" role="menu" popover="manual"
			data-attr:data-place="$$place"
			data-attr:aria-label="$$label || null"
			data-class:anim="$$anim"
			data-on:keydown="@key()"
			data-on:focusin="@focusin()"
			data-on:focusout="@focusout()">
			<!-- r?.: when the list shrinks, data-for can re-evaluate a removed row once with r undefined. -->
			<template data-for="r, i in $$rows">
				<div part="item"
					data-show="!!r"
					data-attr:role="!r ? null : r.divider ? 'separator' : 'menuitem'"
					data-attr:data-idx="r?.divider ? null : i"
					data-attr:tabindex="r?.divider ? null : -1"
					data-attr:aria-disabled="r?.disabled ? 'true' : null"
					data-class:danger="r?.danger"
					data-on:click="@choose(i)">
					<span class="icon" aria-hidden="true" data-show="$$icons && !r?.divider" data-text="r?.icon"></span>
					<span class="body" data-show="!r?.divider">
						<span class="label" data-text="r?.label"></span>
						<span class="desc" data-show="r?.description" data-text="r?.description"></span>
					</span>
				</div>
			</template>
			<div class="empty" aria-disabled="true" data-attr:role="$$rows.length ? null : 'menuitem'" data-show="!$$rows.length">Nothing here</div>
		</div>
		<slot name="item" data-on:slotchange="@items()"></slot>
	`,
})
