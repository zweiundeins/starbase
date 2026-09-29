import { rocket, startPeeking, stopPeeking } from 'datastar'

// Host getters must not subscribe their caller to our signals, and attribute
// changes arrive inside the effect of whoever set them.
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

// A pixel arrow in the 8px gap between trigger and panel: pointing up in a
// 12×8 box, turned by f for the other sides.
const tip = [[4, 2], [8, 2], [8, 4], [10, 4], [10, 6], [12, 6], [12, 8], [0, 8], [0, 6], [2, 6], [2, 4], [4, 4]]
const arrow = (f) => `polygon(${tip.map(([x, y]) => f(x, y).join('px ') + 'px')})`
const UP = arrow((x, y) => [x, y])
const DOWN = arrow((x, y) => [x, 8 - y])
const LEFT = arrow((x, y) => [y, x])
const RIGHT = arrow((x, y) => [8 - y, x])
const GAP = 8

// Anchor positioning with anchored container queries (the arrow follows a
// flip); place() does the same in JS elsewhere.
const anchors = CSS.supports('container-type: anchored')

// The open popovers, innermost last: Escape closes only the top one.
const layers = []

// Where the focus goes on open: an autofocus element, else the first one that
// takes the focus, looking into the shadow roots of components on the way.
const FOCUSABLE = ':is(a[href], button, input, select, textarea, summary, [tabindex]):not(:disabled, [tabindex^="-"])'
const first = (els) => {
	const auto = els.map((el) => (el.matches('[autofocus]') ? el : el.querySelector('[autofocus]'))).find(Boolean)
	if (auto) return auto
	for (const el of els) {
		for (const x of [el, ...el.querySelectorAll('*')]) {
			if (x.matches(FOCUSABLE) && x.checkVisibility()) return x
			const inner = x.shadowRoot && first([...x.shadowRoot.children])
			if (inner) return inner
		}
	}
}

const styles = /* css */ `
:host {
	--_bg: var(--sb-surface-raised, #10182B);
	--_border: var(--sb-border-strong, #3A4868);
	--_body: var(--sb-text-2, #AEBBDD);
	--_text: var(--sb-text-1, #F3F4FA);
	--_ctl: var(--sb-control-bg, #0B1224);
	--_ctl-border: var(--sb-control-border, #283552);
	--_ctl-hover: var(--sb-control-border-hover, #3A4868);
	--_hover: var(--sb-surface-hover, #1A2540);
	--_focus: var(--sb-brand-light, #B09AFF);
	--_radius: var(--sb-radius, 8px);
	--_ctl-radius: var(--sb-control-radius, 6px);
	--_notch: var(--sb-notch, 1);
	--_n: calc(2px * var(--_notch));
	display: inline-block;
	vertical-align: middle;
}
:host([hidden]) { display: none; }
.anchor { display: inline-flex; anchor-name: --sb-popover; }

.trigger {
	all: unset;
	box-sizing: border-box;
	display: inline-flex;
	align-items: center;
	min-block-size: 2.5rem;
	padding-inline: 0.9rem;
	background: var(--_ctl);
	box-shadow: inset 0 0 0 1px var(--_ctl-border);
	clip-path: ${notch('var(--_n)')};
	border-radius: calc(var(--_ctl-radius) * (1 - var(--_notch)));
	color: var(--_text);
	font-size: 0.875rem;
	font-weight: 600;
	line-height: 1;
	white-space: nowrap;
	cursor: pointer;
	transition: box-shadow 120ms, background 120ms;
}
.trigger:hover { box-shadow: inset 0 0 0 1px var(--_ctl-hover); background: var(--_hover); }
.trigger[aria-expanded="true"] { box-shadow: inset 0 0 0 1px var(--_focus); background: var(--_hover); }
.trigger:focus-visible { box-shadow: inset 0 0 0 2px var(--_focus); outline: 2px solid transparent; outline-offset: -2px; }

/* A top-layer popover, so no ancestor clips it. No transform or filter on
   it: either would become the containing block of the fixed arrow. */
.pop {
	position: fixed;
	inset: auto;
	inline-size: max-content;
	margin: 0;
	padding: 0;
	border: 0;
	overflow: visible;
	background: none;
	color: var(--_body);
	border-radius: calc(var(--_radius) * (1 - var(--_notch)));
	box-shadow: 0 12px 24px -6px rgb(0 0 0 / 0.55);
	--_aw: 12px;
	--_ah: 8px;
}
.panel {
	box-sizing: border-box;
	max-inline-size: min(20rem, 100vw - 1rem);
	max-block-size: min(24rem, 80dvh);
	overflow: auto;
	padding: 0.75rem 0.875rem;
	background: var(--_bg);
	box-shadow: inset 0 0 0 1px var(--_border);
	clip-path: ${notch('var(--_n)')};
	border-radius: inherit;
	font-size: 0.875rem;
	line-height: 1.5;
}
/* The arrow points at the trigger's middle, even when the panel shifts. */
.arrow { position: fixed; inset: auto; margin: 0; width: var(--_aw); height: var(--_ah); background: var(--_border); clip-path: var(--_c); }
[data-side="top"] { --_a: self-block-start; --_a2: self-block-end; --_c: ${DOWN}; --_c2: ${UP}; }
[data-side="bottom"] { --_a: self-block-end; --_a2: self-block-start; --_c: ${UP}; --_c2: ${DOWN}; }
[data-side="start"] { --_a: self-inline-start; --_a2: self-inline-end; --_c: ${RIGHT}; --_c2: ${LEFT}; }
[data-side="end"] { --_a: self-inline-end; --_a2: self-inline-start; --_c: ${LEFT}; --_c2: ${RIGHT}; }
:is([data-side="start"], [data-side="end"]) { --_aw: 8px; --_ah: 12px; }
:host(:dir(rtl)) :is([data-side="start"], [data-side="end"]) .arrow { scale: -1 1; }
/* A flip across the edge moves the panel to the other side, and the arrow with
   it; a flip along the edge only swaps the alignment (bottom-start, bottom-end). */
@supports (container-type: anchored) {
	.pop { position-anchor: --sb-popover; container-type: anchored; position-try-fallbacks: var(--_try); }
	.arrow { position-anchor: --sb-popover; position-area: var(--_a) center; }
	[data-side="top"] { margin-block-end: ${GAP}px; }
	[data-side="bottom"] { margin-block-start: ${GAP}px; }
	[data-side="start"] { margin-inline-end: ${GAP}px; }
	[data-side="end"] { margin-inline-start: ${GAP}px; }
	:is([data-side="top"], [data-side="bottom"]) { --_try: flip-block, flip-inline, flip-block flip-inline; }
	:is([data-side="start"], [data-side="end"]) { --_try: flip-inline, flip-block, flip-block flip-inline; }
	@container anchored(fallback: flip-block flip-inline) { .arrow { --_a: var(--_a2); --_c: var(--_c2); } }
	@container anchored(fallback: flip-block) { :is([data-side="top"], [data-side="bottom"]) .arrow { --_a: var(--_a2); --_c: var(--_c2); } }
	@container anchored(fallback: flip-inline) { :is([data-side="start"], [data-side="end"]) .arrow { --_a: var(--_a2); --_c: var(--_c2); } }
}
.anim:popover-open { animation: fade 110ms ease-out; }
@keyframes fade { from { opacity: 0; } }
@media (prefers-reduced-motion: reduce) {
	.anim:popover-open { animation: none; }
	.trigger { transition: none; }
}
@media (forced-colors: active) {
	.trigger { outline: 1px solid ButtonBorder; outline-offset: -1px; }
	.panel { outline: 1px solid CanvasText; outline-offset: -1px; }
	.arrow { forced-color-adjust: none; background: CanvasText; }
}
`

rocket('sb-popover', {
	props: ({ bool, oneOf, string }) => ({
		label: string.trim.default('Details').docs({ description: 'Text of the default trigger, and the accessible name of the panel (an aria-label on the element wins).' }),
		mode: oneOf('click', 'hover').default('click').docs({ description: 'click: a dialog that opens on click and closes on an outside click or Escape. hover: a hover card that opens after a short delay on hover or keyboard focus and describes the trigger.' }),
		placement: oneOf('bottom', 'bottom-start', 'bottom-end', 'top', 'top-start', 'top-end', 'start', 'start-start', 'start-end', 'end', 'end-start', 'end-end').default('bottom').docs({ description: 'Side of the trigger, and the alignment along it. start and end are logical, so they mirror in right-to-left text. The panel flips and shifts to stay on screen.' }),
		arrow: bool.docs({ description: 'Draw a pixel arrow that points at the middle of the trigger.' }),
		open: bool.docs({ description: 'Open. View state the server may own: a changed attribute wins (open="false" closes), a removed one is ignored. Never reflected: use the open property, show() and hide() from the client.' }),
		name: string.trim.docs({ description: 'Name reported in sb-open and sb-close (e.g. the field of a command).' }),
	}),
	manifest: {
		slots: [
			{ name: 'trigger', description: 'Your own trigger: a native button (or a link, in hover mode). It gets aria-expanded and aria-haspopup (click mode) or an aria-describedby to the content (hover mode), put back after every morph. Without it, a button shows label.' },
			{ name: 'default', description: 'The panel content.' },
		],
		events: [
			{ name: 'sb-open', kind: 'custom-event', bubbles: true, composed: true, description: 'The panel opened. detail: { name, reason }, reason being trigger, server or api.' },
			{ name: 'sb-close', kind: 'custom-event', bubbles: true, composed: true, description: 'The panel closed. detail: { name, reason }: trigger, outside, escape, leave (hover mode), server or api.' },
		],
	},
	// Rendered once: the trigger and the panel hold the focus.
	renderOnPropChange: false,
	setup: ({ $$, action, adoptStyles, cleanup, defineHostProp, emit, host, observeProps, overrideProp, props }) => {
		adoptStyles(host, styles)
		let isOpen = props.open // plain too: cleanup runs after the signals are cleared
		$$.open = isOpen
		$$.anim = false // no fade-in for a panel that starts open
		$$.label = props.label
		$$.name = host.getAttribute('aria-label') || props.label
		$$.hover = props.mode === 'hover'
		$$.arrow = props.arrow
		$$.side = props.placement.split('-')[0]

		const $ = (s) => host.shadowRoot?.querySelector(s)
		const pop = () => $('.pop')
		const content = () => $('slot:not([name])')?.assignedElements({ flatten: true }) ?? []
		const slotted = () => $('slot[name="trigger"]')?.assignedElements()[0]
		const trigger = () => slotted() ?? $('.trigger')

		// Without anchored container queries: fixed coordinates, flipped to the
		// other side when only that one has room, shifted back into the viewport.
		const place = () => {
			const p = pop()
			const a = $('.anchor')
			if (!p || !a) return
			let [side, align] = props.placement.split('-')
			const block = side === 'top' || side === 'bottom'
			if (anchors) {
				$$.side = side
				const at = { top: 'block-start', bottom: 'block-end', start: 'inline-start', end: 'inline-end' }[side]
				// Centred: the whole row (or column) is the area, so a panel wider
				// than the trigger still fits and the fallbacks get their turn.
				p.style.positionArea = `self-${at} ${align ? `span-self-${block ? 'inline' : 'block'}-${align === 'start' ? 'end' : 'start'}` : 'span-all'}`
				p.style.placeSelf = align ? '' : block ? 'normal anchor-center' : 'anchor-center normal'
				return
			}
			if (!isOpen) return
			const r = a.getBoundingClientRect()
			const rtl = getComputedStyle(host).direction === 'rtl'
			const w = p.offsetWidth
			const h = p.offsetHeight
			// The viewport without its scrollbars.
			const { clientWidth: vw, clientHeight: vh } = document.documentElement
			const pad = 8
			const fits = (v, size, max) => v >= pad && v + size <= max - pad
			// Across the edge: after the trigger (or before it), or the other way
			// round when only that fits. Along it: aligned to lo (-1), hi (1) or centred.
			const across = (lo, hi, size, max, after) => {
				const [x, y] = after ? [hi + GAP, lo - GAP - size] : [lo - GAP - size, hi + GAP]
				return fits(x, size, max) || !fits(y, size, max) ? x : y
			}
			const along = (lo, hi, size, al) => (al ? (al < 0 ? lo : hi - size) : (lo + hi - size) / 2)
			const shift = (v, size, max) => Math.round(Math.min(Math.max(pad, v), Math.max(pad, max - size - pad))) + 'px'
			let left
			let top
			if (block) {
				top = across(r.top, r.bottom, h, vh, side === 'bottom')
				left = along(r.left, r.right, w, align && ((align === 'start') !== rtl ? -1 : 1))
				side = top > r.top ? 'bottom' : 'top'
			} else {
				left = across(r.left, r.right, w, vw, (side === 'end') !== rtl)
				top = along(r.top, r.bottom, h, align && (align === 'start' ? -1 : 1))
				side = (left > r.left) !== rtl ? 'end' : 'start'
			}
			$$.side = side
			p.style.left = shift(left, w, vw)
			p.style.top = shift(top, h, vh)
			Object.assign($('.arrow').style, {
				left: (block ? r.left + r.width / 2 - 6 : left > r.left ? r.right : r.left - GAP) + 'px',
				top: (block ? (side === 'bottom' ? r.bottom : r.top - GAP) : r.top + r.height / 2 - 6) + 'px',
			})
		}

		// Your trigger is page markup that a morph resets: the watcher below calls
		// this again, and it writes only what differs (no mutation, no loop).
		const aria = () => {
			const t = slotted()
			if (!t) return
			const want = { 'aria-haspopup': $$.hover ? null : 'dialog', 'aria-expanded': $$.hover ? null : String(isOpen) }
			for (const k in want) if (t.getAttribute(k) !== want[k]) want[k] === null ? t.removeAttribute(k) : t.setAttribute(k, want[k])
			// An id can't point into the shadow root, but an element reference to
			// the slotted content (same tree as the trigger) can.
			const d = content()
			const had = t.ariaDescribedByElements ?? []
			if ($$.hover && (had.length !== d.length || d.some((el, i) => el !== had[i]))) t.ariaDescribedByElements = d
		}

		// No attribute form for these: a press anywhere in the document, Escape
		// wherever the focus is, and scrolling while we position by hand.
		const onDown = (evt) => evt.composedPath().includes(host) || setOpen(false, 'outside')
		const onKey = (evt) => {
			if (evt.key !== 'Escape' || evt.defaultPrevented || layers.at(-1) !== host) return
			evt.preventDefault()
			setOpen(false, 'escape')
			// Dismissed: it stays closed until the pointer or the focus comes back.
			hov = foc = false
			clearTimeout(timer)
		}
		const onMove = () => place()
		let bound = false
		const bind = (on) => {
			if (on === bound) return
			bound = on
			const m = on ? 'addEventListener' : 'removeEventListener'
			document[m]('pointerdown', onDown, true)
			document[m]('keydown', onKey)
			if (!anchors) {
				window[m]('scroll', onMove, { capture: true, passive: true })
				window[m]('resize', onMove)
			}
		}

		const apply = () => {
			const p = pop()
			if (!p) return // before the first render
			try {
				p.togglePopover(isOpen)
			} catch {}
			const i = layers.indexOf(host)
			if (i >= 0) layers.splice(i, 1)
			if (isOpen) layers.push(host)
			place()
			aria()
			bind(isOpen)
		}

		let timer = 0
		const setOpen = (next, reason, focus) => {
			clearTimeout(timer)
			if (next === isOpen) return
			// The focus goes back to the trigger when it was inside the panel,
			// unless a press outside is taking it somewhere else.
			const back = !next && reason !== 'outside' && pop()?.matches(':focus-within')
			if (next) $$.anim = true
			$$.open = isOpen = next
			apply()
			if (focus && next && !$$.hover) first(content())?.focus()
			if (back) trigger()?.focus()
			emit(next ? 'sb-open' : 'sb-close', { name: props.name, reason })
		}

		// Hover mode: open a moment after the pointer or the focus came to trigger
		// or panel, close a moment after both left, so the pointer can cross the gap.
		let hov = false
		let foc = false
		const hovering = () => {
			if (!$$.hover) return
			clearTimeout(timer)
			const want = hov || foc
			if (want !== isOpen) timer = setTimeout(() => peek(() => setOpen(want, want ? 'trigger' : 'leave')), want ? 400 : 300)
		}

		// served is the server's last word on open: only a changed one wins, and a
		// removed attribute is ignored (morphs also strip reflected ones).
		let served = host.hasAttribute('open') ? props.open : null
		const serverOpen = () => {
			if (!host.hasAttribute('open')) return void (served = null)
			if (props.open === served) return
			setOpen((served = props.open), 'server')
		}
		const watch = new MutationObserver(() =>
			peek(() => {
				serverOpen()
				$$.name = host.getAttribute('aria-label') || props.label
				aria()
			}),
		)
		watch.observe(host, { attributes: true, subtree: true, attributeFilter: ['open', 'aria-label', 'aria-expanded', 'aria-haspopup', 'aria-describedby'] })

		observeProps((p) =>
			peek(() => {
				$$.label = p.label
				$$.name = host.getAttribute('aria-label') || p.label
				$$.hover = p.mode === 'hover'
				$$.arrow = p.arrow
				place()
				aria()
			}),
		)

		overrideProp('open', () => peek(() => $$.open), (v) => peek(() => setOpen(!!v && v !== 'false', 'api', true)))
		defineHostProp('show', { value: () => peek(() => setOpen(true, 'api', true)) })
		defineHostProp('hide', { value: () => peek(() => setOpen(false, 'api')) })

		cleanup(() => {
			// Removed or moved while open: it closes, and says so, unless the
			// server's open attribute brings it straight back.
			if (isOpen && !served) setTimeout(() => emit('sb-close', { name: props.name, reason: 'api' }))
			bind(false)
			watch.disconnect()
			clearTimeout(timer)
			const i = layers.indexOf(host)
			if (i >= 0) layers.splice(i, 1)
			try {
				pop()?.hidePopover()
			} catch {}
		})
		// A panel the server rendered open shows right after the first render.
		queueMicrotask(() => peek(apply))

		action('aria', () => peek(aria))
		// Hover mode takes a tap (no hover on a touch screen), not a mouse click.
		action('click', ({ evt }) => peek(() => (!$$.hover || evt.pointerType === 'touch') && setOpen(!isOpen, 'trigger', true)))
		action('point', ({ evt }, on) => peek(() => evt.pointerType !== 'touch' && ((hov = on), hovering())))
		// The focus holds a hover card open, except a mouse click's on the
		// trigger (it isn't :focus-visible, and the pointer is on it anyway).
		action('focus', ({ evt }, on, panel) =>
			peek(() => {
				foc = on && (panel || !hov || evt.composedPath()[0].matches(':focus-visible'))
				hovering()
			}),
		)
	},
	render: ({ html }) => html`
		<span class="anchor"
			data-on:click="@click()"
			data-on:pointerenter="@point(true)"
			data-on:pointerleave="@point(false)"
			data-on:focusin="@focus(true)"
			data-on:focusout="@focus(false)"><slot name="trigger" data-on:slotchange="@aria()"><button class="trigger" part="trigger" type="button"
				data-attr:aria-haspopup="$$hover ? null : 'dialog'"
				data-attr:aria-expanded="$$hover ? null : String($$open)"
				data-attr:aria-controls="$$hover ? null : 'pop'"
				data-attr:aria-describedby="$$hover ? 'pop' : null"
				data-text="$$label"></button></slot></span>
		<div id="pop" class="pop" popover="manual"
			data-attr:data-side="$$side"
			data-attr:role="$$hover ? null : 'dialog'"
			data-attr:aria-label="$$hover ? null : $$name"
			data-class:anim="$$anim"
			data-on:pointerenter="@point(true)"
			data-on:pointerleave="@point(false)"
			data-on:focusin="@focus(true, true)"
			data-on:focusout="@focus(false)">
			<div class="panel" part="panel"><slot data-on:slotchange="@aria()"></slot></div>
			<span class="arrow" part="arrow" aria-hidden="true" data-show="$$arrow"></span>
		</div>
	`,
})
