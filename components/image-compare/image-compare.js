import { rocket, startPeeking, stopPeeking } from 'datastar'

// Host getters must not subscribe callers (e.g. data-bind's sync effect) to
// the internal signal, or that effect writes the stale bound value back.
const peek = (fn) => {
	startPeeking()
	try {
		return fn()
	} finally {
		stopPeeking()
	}
}

// One ElementInternals per element: attachInternals() works once, and setup
// runs again when the element is re-attached. :state(expanded) is styleable
// from the page and morph-proof.
const internals = new WeakMap()
const internalsOf = (host) => internals.get(host) ?? internals.set(host, host.attachInternals()).get(host)

const clamp = (v) => Math.min(100, Math.max(0, Number.isFinite(v) ? v : 50))

// Pixel corners: notches every corner by --_n (2px times --sb-notch; at 0 the
// border-radius takes over).
const notch = `polygon(var(--_n) 0, calc(100% - var(--_n)) 0, calc(100% - var(--_n)) var(--_n), 100% var(--_n), 100% calc(100% - var(--_n)), calc(100% - var(--_n)) calc(100% - var(--_n)), calc(100% - var(--_n)) 100%, var(--_n) 100%, var(--_n) calc(100% - var(--_n)), 0 calc(100% - var(--_n)), 0 var(--_n), var(--_n) var(--_n))`

const styles = /* css */ `
:host {
	--_bg: var(--sb-bg, #080D1D);
	--_text: var(--sb-text-1, #F3F4FA);
	--_frame: var(--sb-frame-color, #B09AFF);
	--_step: var(--sb-frame-step, 3px);
	--_line: var(--sb-image-compare-line, var(--sb-text-1, #F3F4FA));
	--_handle: var(--sb-image-compare-handle, var(--sb-text-1, #F3F4FA));
	--_handle-ink: var(--sb-image-compare-handle-ink, var(--sb-bg, #080D1D));
	--_edge: var(--sb-brand-light, #B09AFF);
	--_star: var(--sb-text-muted, #7785A8);
	--_star-bright: var(--sb-text-1, #F3F4FA);
	--_stars: var(--sb-image-compare-stars,
		linear-gradient(var(--_star-bright) 0 0) 82% 6% / 3px 3px no-repeat,
		linear-gradient(var(--_star) 0 0) 95% 79% / 2px 2px no-repeat,
		linear-gradient(var(--_star) 0 0) 80% 2% / 2px 2px no-repeat,
		linear-gradient(var(--_star) 0 0) 68% 9% / 2px 2px no-repeat,
		linear-gradient(var(--_star) 0 0) 8% 5% / 2px 2px no-repeat,
		linear-gradient(var(--_star-bright) 0 0) 77% 4% / 3px 3px no-repeat,
		linear-gradient(var(--_star) 0 0) 1% 85% / 2px 2px no-repeat,
		linear-gradient(var(--_star) 0 0) 11% 91% / 2px 2px no-repeat,
		linear-gradient(var(--_star) 0 0) 98% 30% / 2px 2px no-repeat,
		linear-gradient(var(--_star) 0 0) 4% 9% / 2px 2px no-repeat,
		linear-gradient(var(--_star-bright) 0 0) 73% 99% / 3px 3px no-repeat,
		linear-gradient(var(--_star) 0 0) 50% 9% / 2px 2px no-repeat,
		linear-gradient(var(--_star) 0 0) 3% 88% / 2px 2px no-repeat,
		linear-gradient(var(--_star) 0 0) 1% 28% / 2px 2px no-repeat,
		linear-gradient(var(--_star) 0 0) 27% 7% / 2px 2px no-repeat,
		linear-gradient(var(--_star-bright) 0 0) 54% 10% / 3px 3px no-repeat,
		linear-gradient(var(--_star) 0 0) 43% 2% / 2px 2px no-repeat,
		linear-gradient(var(--_star) 0 0) 53% 98% / 2px 2px no-repeat,
		linear-gradient(var(--_star) 0 0) 32% 91% / 2px 2px no-repeat,
		linear-gradient(var(--_star) 0 0) 13% 2% / 2px 2px no-repeat,
		linear-gradient(var(--_star-bright) 0 0) 8% 60% / 3px 3px no-repeat,
		linear-gradient(var(--_star) 0 0) 94% 99% / 2px 2px no-repeat,
		linear-gradient(var(--_star) 0 0) 1% 35% / 2px 2px no-repeat,
		linear-gradient(var(--_star) 0 0) 3% 27% / 2px 2px no-repeat);
	--_notch: var(--sb-notch, 1);
	--_n: calc(2px * var(--_notch));
	display: block;
}
:host([hidden]) { display: none; }

/* The frame is a popover, so that full screen puts the very same slotted
   images in the top layer. Closed, it is an ordinary block in the page. */
.frame:not(:popover-open) {
	display: block;
	position: relative;
	inset: auto;
	inline-size: auto;
	block-size: auto;
	margin: 0;
	padding: 0;
	border: 0;
	overflow: visible;
	background: none;
	color: inherit;
}
.frame:popover-open {
	box-sizing: border-box;
	position: fixed;
	inset: 0;
	inline-size: 100vw;
	block-size: 100dvh;
	max-inline-size: none;
	max-block-size: none;
	margin: 0;
	padding: 4rem 1.5rem 1.5rem;
	border: 0;
	display: flex;
	align-items: center;
	justify-content: center;
	overflow: hidden;
	touch-action: none;
	background: var(--_stars), var(--_bg);
	color: var(--_text);
}
.frame::backdrop { background: rgb(0 0 0 / 70%); }

/* The 8-bit frame: notched by stepped shadows in --sb-frame-color, with a
   hard drop shadow (like sb-button's pixel variant). --sb-frame-step: 1px
   tones it down to a hairline, 0 removes it. */
.stage {
	position: relative;
	margin: var(--_step);
	overflow: hidden;
	background: var(--_bg);
	box-shadow:
		0 calc(-1 * var(--_step)) 0 0 var(--_frame),
		0 var(--_step) 0 0 var(--_frame),
		calc(-1 * var(--_step)) 0 0 0 var(--_frame),
		var(--_step) 0 0 0 var(--_frame),
		calc(var(--_step) * 2) calc(var(--_step) * 2) 0 0 color-mix(in oklch, var(--_frame) 45%, black);
	/* Vertical swipes still scroll the page; horizontal ones move the divider. */
	touch-action: pan-y;
	cursor: ew-resize;
	user-select: none;
	-webkit-user-select: none;
}
.stage:has(input:focus-visible) { outline: 2px solid var(--_frame); outline-offset: calc(var(--_step) * 3); }
.frame:popover-open .stage {
	flex: none;
	inline-size: min(100% - 2 * var(--_step), (100dvh - 5.5rem - 2 * var(--_step)) * var(--_ratio, 1.6));
	touch-action: none;
}

/* Both sides are cut at the divider, so transparent images never show
   each other. The after side stays in the flow and sets the size. */
.after { position: relative; clip-path: inset(0 0 0 var(--_p, 50%)); }
.before { position: absolute; inset: 0; clip-path: inset(0 calc(100% - var(--_p, 50%)) 0 0); }
.stage:dir(rtl) .after { clip-path: inset(0 var(--_p, 50%) 0 0); }
.stage:dir(rtl) .before { clip-path: inset(0 0 0 calc(100% - var(--_p, 50%))); }
/* Slotted content is a picture, not a control: pointers belong to the stage. */
::slotted(*) { display: block; inline-size: 100%; block-size: auto; pointer-events: none; -webkit-user-drag: none; }
.before ::slotted(*) { block-size: 100%; object-fit: cover; object-position: left top; }

/* Each tag sits in its own layer, so the divider cuts it like the image. */
.tag {
	position: absolute;
	inset-block-start: 0.5rem;
	max-inline-size: calc(50% - 1rem);
	overflow: hidden;
	text-overflow: ellipsis;
	white-space: nowrap;
	padding: 0.375rem 0.5rem;
	font-family: var(--sb-font-display, inherit);
	font-size: 0.6875rem;
	font-weight: 700;
	letter-spacing: 0.1em;
	line-height: 1;
	text-transform: uppercase;
	color: var(--_text);
	background: color-mix(in oklch, var(--_bg) 82%, transparent);
	clip-path: ${notch};
	border-radius: calc(3px * (1 - var(--_notch)));
	pointer-events: none;
}
.tag:empty { display: none; }
.before .tag { inset-inline-start: 0.5rem; }
.after .tag { inset-inline-end: 0.5rem; }

.line {
	position: absolute;
	inset-block: 0;
	inset-inline-start: var(--_p, 50%);
	inline-size: 2px;
	translate: -1px 0;
	background: var(--_line);
	box-shadow: 0 0 0 1px color-mix(in oklch, var(--_bg) 60%, transparent);
	pointer-events: none;
}
.stage:dir(rtl) .line { translate: 1px 0; }
/* The grip: a light plate with a bevelled bottom edge, like sb-slider's thumb. */
.handle {
	position: absolute;
	inset-block-start: 50%;
	inset-inline-start: var(--_p, 50%);
	translate: -50% -50%;
	inline-size: 2.75rem;
	block-size: 2.75rem;
	display: grid;
	place-items: center;
	background: var(--_handle);
	color: var(--_handle-ink);
	box-shadow: inset 0 -3px 0 var(--_edge);
	clip-path: ${notch};
	border-radius: calc(50% * (1 - var(--_notch)));
	pointer-events: none;
}
.stage:dir(rtl) .handle { translate: 50% -50%; }
.stage:active .handle { margin-block-start: 1px; }

/* The native range input carries keyboard, screen reader and value text.
   Pointers go to the stage, so the input stays out of their way. */
input {
	position: absolute;
	inset: 0;
	inline-size: 100%;
	block-size: 100%;
	margin: 0;
	opacity: 0;
	pointer-events: none;
}

.expand {
	position: absolute;
	inset-block-end: calc(var(--_step) + 0.5rem);
	inset-inline-end: calc(var(--_step) + 0.5rem);
	inline-size: 2.75rem;
	block-size: 2.75rem;
	display: grid;
	place-items: center;
	border: 0;
	padding: 0;
	color: var(--_text);
	background: color-mix(in oklch, var(--_bg) 82%, transparent);
	clip-path: ${notch};
	border-radius: calc(3px * (1 - var(--_notch)));
	cursor: pointer;
}
.expand:hover { background: var(--_frame); color: var(--_bg); }
/* A transparent outline is what forced colours show, where shadows are dropped. */
.expand:focus-visible { box-shadow: inset 0 0 0 2px var(--_frame); outline: 2px solid transparent; }
.frame:popover-open .expand { position: fixed; inset-block: 1rem auto; inset-inline-end: 1.5rem; }
.expand .close, .frame:popover-open .expand .open { display: none; }
.frame:popover-open .expand .close { display: block; }

@media (forced-colors: active) {
	.stage { outline: 1px solid CanvasText; }
	.stage:has(input:focus-visible) { outline: 2px solid Highlight; }
	.line { background: CanvasText; box-shadow: none; }
	.handle, .tag, .expand { forced-color-adjust: none; background: Canvas; color: CanvasText; }
	.handle { box-shadow: inset 0 0 0 2px CanvasText; }
	.expand:hover { background: Highlight; color: HighlightText; }
	.expand:focus-visible { outline: 2px solid Highlight; }
}
`

rocket('sb-image-compare', {
	props: ({ bool, number, string }) => ({
		position: number.clamp(0, 100).default(50).docs({ description: 'Where the divider sits, in percent from the start edge. A new position from the page or the server replaces the reader\'s.' }),
		beforeLabel: string.trim.docs({ description: 'Tag over the before side (slot "before", on the start side of the divider).' }),
		afterLabel: string.trim.docs({ description: 'Tag over the after side (slot "after", on the end side of the divider).' }),
		label: string.trim.docs({ description: 'Accessible name of the divider, and of the full screen view. Without it, the element\'s aria-label, else "Comparison".' }),
		expandable: bool.docs({ description: 'Show a button that opens the comparison full screen (Escape closes it).' }),
		expanded: bool.docs({ description: 'Full screen, as view state: a changed attribute from the page or the server opens or closes it, a removed one is ignored. The reader\'s own opening and closing never touch it; sb-expand reports them.' }),
		expandLabel: string.default('Full screen').docs({ description: 'Accessible name of the full screen button.' }),
		closeLabel: string.default('Exit full screen').docs({ description: 'Accessible name of the button while full screen.' }),
	}),
	manifest: {
		slots: [
			{ name: 'before', description: 'The before side, on the start side of the divider: an <img>, a <picture> (the page sizes its <img>), or any element.' },
			{ name: 'after', description: 'The after side, on the end side. It sets the size of the comparison.' },
		],
		events: [
			{ name: 'sb-position', kind: 'custom-event', bubbles: true, composed: true, description: 'After a drag, a tap or a key moved the divider. detail: { position } (percent).' },
			{ name: 'sb-expand', kind: 'custom-event', bubbles: true, composed: true, description: 'When full screen opens or closes. detail: { expanded }.' },
		],
	},
	// A new position must not re-render: that would drop keyboard focus and
	// close full screen. Everything that changes is driven by signals.
	renderOnPropChange: false,
	setup: ({ $$, adoptStyles, cleanup, emit, host, observeProps, overrideProp, props }) => {
		adoptStyles(host, styles)
		// Moved or re-inserted while full screen (a morph does that): the browser
		// closed the popover without a toggle event, so say so here.
		const states = internalsOf(host).states
		if (states.has('expanded')) {
			states.delete('expanded')
			emit('sb-expand', { expanded: false })
		}
		const sync = () =>
			peek(() => {
				$$.before = props.beforeLabel
				$$.after = props.afterLabel
				$$.expandable = props.expandable
				$$.name = props.label || host.ariaLabel || 'Comparison'
				$$.expandText = props.expandLabel
				$$.closeText = props.closeLabel
			})
		sync()
		observeProps(sync)
		$$.pos = clamp(props.position)
		$$.ratio = ''
		$$.open = false
		// A position attribute from the page or the server wins when it
		// changes; re-sending the same markup changes nothing. The attribute is
		// watched, not the prop: position="50" on an element rendered without
		// one decodes to the same 50, and must still take back a drag.
		let served = host.hasAttribute('position') ? props.position : null
		const watch = new MutationObserver(() =>
			peek(() => {
				if (!host.hasAttribute('position')) return void (served = null)
				if (props.position === served) return
				served = props.position
				$$.pos = clamp(served)
			}),
		)
		watch.observe(host, { attributeFilter: ['position'] })
		cleanup(() => watch.disconnect())
		overrideProp('position', () => peek(() => $$.pos), (v) => peek(() => ($$.pos = clamp(Number(v)))))
	},
	onFirstRender: ({ $$, action, cleanup, emit, host, overrideProp, props, refs: { frame, stage, input, button } }) => {
		const states = internalsOf(host).states
		// Full screen from the page or the server: served is its last word, so
		// only a changed attribute wins and a removed one is ignored (morphs also
		// strip reflected attributes), as with sb-popover's open.
		const expand = (on) => {
			if (!frame.isConnected || on === frame.matches(':popover-open')) return
			on ? frame.showPopover() : frame.hidePopover()
		}
		let served = host.hasAttribute('expanded') ? props.expanded : null
		if (served) expand(true)
		const watchExpanded = new MutationObserver(() =>
			peek(() => {
				if (!host.hasAttribute('expanded')) return void (served = null)
				if (props.expanded === served) return
				expand((served = props.expanded))
			}),
		)
		watchExpanded.observe(host, { attributeFilter: ['expanded'] })
		cleanup(() => watchExpanded.disconnect())
		overrideProp('expanded', () => peek(() => $$.open), (v) => peek(() => expand(!!v && v !== 'false')))
		const report = () => emit('sb-position', { position: peek(() => Math.round($$.pos * 10) / 10) })
		const at = (x) => {
			const r = stage.getBoundingClientRect()
			const p = ((x - r.left) / (r.width || 1)) * 100
			$$.pos = clamp(stage.matches(':dir(rtl)') ? 100 - p : p)
		}
		// A mouse moves the divider on press, so a click jumps there. A finger
		// may be starting a vertical scroll, which the browser takes over
		// (pointercancel), and Chrome sends a pointermove or two before it
		// does: a finger moves the divider only once it has gone sideways
		// more than up or down, or on a tap.
		let drag = null
		const end = (moved) => {
			drag = null
			if (moved) {
				input.focus({ preventScroll: true, focusVisible: false })
				report()
			}
		}
		action('drag', ({ evt }) => {
			if (evt.type === 'pointerdown') {
				if (evt.button || !evt.isPrimary) return // the first finger, the main button
				// No mousedown default: it would take the focus off the divider
				// (and start a text selection or an image drag).
				evt.preventDefault()
				drag = { id: evt.pointerId, x: evt.clientX, y: evt.clientY, moved: false, scroll: false }
				stage.setPointerCapture(evt.pointerId)
				if (evt.pointerType === 'mouse') {
					at(evt.clientX)
					drag.moved = true
				}
			} else if (drag?.id !== evt.pointerId) {
				return // another pointer, or the capture ended after pointerup
			} else if (evt.type === 'pointermove') {
				if (!drag.moved) {
					const dx = Math.abs(evt.clientX - drag.x)
					const dy = Math.abs(evt.clientY - drag.y)
					if (dy > dx && dy > 4) drag.scroll = true
					if (drag.scroll || dx < 6) return
					drag.moved = true
				}
				at(evt.clientX)
			} else if (evt.type === 'pointerup') {
				if (drag.scroll) return end(false)
				if (!drag.moved) at(evt.clientX) // a tap
				end(true)
			} else {
				end(drag.moved) // pointercancel, lostpointercapture
			}
		})
		action('commit', report)
		// Full screen fits the stage to the viewport by its aspect ratio,
		// measured while it is still in the page.
		action('measure', ({ evt }) => {
			if (evt.newState === 'open') $$.ratio = String(stage.offsetWidth / (stage.offsetHeight || 1))
		})
		action('toggled', ({ evt }) => {
			const open = evt.newState === 'open'
			if (open === peek(() => $$.open)) return
			$$.open = open
			open ? states.add('expanded') : states.delete('expanded')
			emit('sb-expand', { expanded: open })
		})
		// Full screen is a popover, not a modal: keep Tab between the divider
		// and the close button, and the wheel from scrolling the page behind.
		action('keep', ({ evt }) => {
			if (!peek(() => $$.open)) return
			if (evt.type === 'wheel') return void evt.preventDefault()
			if (evt.key !== 'Tab') return
			const active = frame.getRootNode().activeElement
			const next = active === input ? button : input
			if (active !== input && active !== button) return void (evt.preventDefault(), input.focus())
			evt.preventDefault()
			next.focus()
		})
	},
	render: ({ html }) => html`
		<div class="frame" id="frame" part="frame" popover="auto" data-ref:frame
			data-attr:role="$$open ? 'dialog' : null"
			data-attr:aria-modal="$$open ? 'true' : null"
			data-attr:aria-label="$$open ? $$name : null"
			data-on:beforetoggle="@measure()"
			data-on:toggle="@toggled()"
			data-on:keydown="@keep()"
			data-on:wheel="@keep()">
			<div class="stage" part="stage" data-ref:stage
				data-style:--_p="$$pos + '%'"
				data-style:--_ratio="$$ratio"
				data-on:pointerdown="@drag()"
				data-on:pointermove="@drag()"
				data-on:pointerup="@drag()"
				data-on:pointercancel="@drag()"
				data-on:lostpointercapture="@drag()">
				<div class="after" part="after"><slot name="after"></slot><span class="tag" part="tag after-tag" data-text="$$after"></span></div>
				<div class="before" part="before"><slot name="before"></slot><span class="tag" part="tag before-tag" data-text="$$before"></span></div>
				<span class="line" part="divider"></span>
				<span class="handle" part="handle"><svg viewBox="0 0 12 8" width="24" height="16" shape-rendering="crispEdges" fill="currentColor" aria-hidden="true"><path d="M3 0h1v1h-1zM8 0h1v1h-1zM2 1h2v1h-2zM8 1h2v1h-2zM1 2h3v1h-3zM8 2h3v1h-3zM0 3h4v1h-4zM8 3h4v1h-4zM0 4h4v1h-4zM8 4h4v1h-4zM1 5h3v1h-3zM8 5h3v1h-3zM2 6h2v1h-2zM8 6h2v1h-2zM3 7h1v1h-1zM8 7h1v1h-1z"/></svg></span>
				<input type="range" min="0" max="100" step="1" data-ref:input
					data-attr:aria-label="$$name"
					data-attr:aria-valuetext="($$before || 'Before') + ' ' + Math.round($$pos) + '%, ' + ($$after || 'After') + ' ' + Math.round(100 - $$pos) + '%'"
					data-effect="el.value != $$pos && (el.value = $$pos)"
					data-on:input="$$pos = +el.value"
					data-on:change="@commit()" />
			</div>
			<button class="expand" part="expand" type="button" popovertarget="frame" data-ref:button
				data-attr:aria-label="$$open ? $$closeText : $$expandText"
				data-attr:title="$$open ? $$closeText : $$expandText"
				data-show="$$expandable">
				<svg class="open" viewBox="0 0 10 10" width="20" height="20" shape-rendering="crispEdges" fill="currentColor" aria-hidden="true"><path d="M0 0h4v1h-4zM6 0h4v1h-4zM0 1h1v1h-1zM9 1h1v1h-1zM0 2h1v1h-1zM9 2h1v1h-1zM0 3h1v1h-1zM9 3h1v1h-1zM0 6h1v1h-1zM9 6h1v1h-1zM0 7h1v1h-1zM9 7h1v1h-1zM0 8h1v1h-1zM9 8h1v1h-1zM0 9h4v1h-4zM6 9h4v1h-4z"/></svg>
				<svg class="close" viewBox="0 0 10 10" width="20" height="20" shape-rendering="crispEdges" fill="currentColor" aria-hidden="true"><path d="M1 1h2v1h-2zM7 1h2v1h-2zM2 2h2v1h-2zM6 2h2v1h-2zM3 3h4v1h-4zM4 4h2v1h-2zM3 5h4v1h-4zM2 6h2v1h-2zM6 6h2v1h-2zM1 7h2v1h-2zM7 7h2v1h-2zM1 8h1v1h-1zM8 8h1v1h-1z"/></svg>
			</button>
		</div>
	`,
})
