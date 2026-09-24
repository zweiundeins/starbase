import { rocket, startPeeking, stopPeeking } from 'datastar'

// Reads of $$ outside render() go through peek(): host getters must not
// subscribe their caller (data-bind's sync effect) to an internal signal, and
// observeProps callbacks run inside the effect of whoever set the attribute.
const peek = (fn) => {
	startPeeking()
	try {
		return fn()
	} finally {
		stopPeeking()
	}
}

// One ElementInternals per element: attachInternals() works once, and setup
// runs again when the element is re-attached. Custom states (:state(busy)) live
// outside the attributes, so a server morph cannot reset them.
const internals = new WeakMap()
const internalsOf = (host) => internals.get(host) ?? internals.set(host, host.attachInternals()).get(host)

// What a move carries over: a morph moves elements (moveBefore), Rocket has no
// connectedMoveCallback, so a move runs cleanup and then setup again while the
// counted requests are still in flight. Cleanup tells a move (the host is
// already connected again) from a removal, which carries nothing.
const moved = new WeakMap()

// Pixel corners: notches every corner by p (2px times --sb-notch; at 0 the
// border radius takes over).
const notch = (p) => `polygon(${p} 0, calc(100% - ${p}) 0, calc(100% - ${p}) ${p}, 100% ${p}, 100% calc(100% - ${p}), calc(100% - ${p}) calc(100% - ${p}), calc(100% - ${p}) 100%, ${p} 100%, ${p} calc(100% - ${p}), 0 calc(100% - ${p}), 0 ${p}, ${p} ${p})`

// Notes on the styles, kept here where the minifier drops them:
// - size is styled from the host attribute (setup reflects a size that was set
//   as a property before the element connected).
// - .vh is the status region: always there and visually hidden, its text set
//   when the wait starts. A region that appears with its text already in it is
//   not reliably announced.
// - The bar's fill and sweep are physical (left to right), mirrored in RTL.
// - Forced colors drop backgrounds and box-shadows, which is all this draws:
//   the dots and the fill become CanvasText, the bar and the blocks get an
//   outline (inset, or the clip-path cuts it off).
const styles = /* css */ `
:host {
	--_fill: var(--sb-brand, #8C6BFF);
	--_edge: var(--sb-brand-light, #B09AFF);
	--_track: var(--sb-surface-inset, #0B1224);
	--_border: var(--sb-border, #283552);
	--_text: var(--sb-text-2, #AEBBDD);
	--_notch: var(--sb-notch, 1);
	--_n: calc(2px * var(--_notch));
	--_s: 24px;  /* spinner box */
	--_d: 5px;   /* spinner dot */
	--_h: 10px;  /* bar height */
	--_l: 12px;  /* skeleton line */
	display: block;
	inline-size: 100%;
}
/* The spinner sits in a line of content; the bar and the skeleton fill their
   column. Only :host can carry this, so set variant as an attribute. */
:host(:not([variant])), :host([variant="spinner"]) { display: inline-block; inline-size: auto; vertical-align: middle; }
:host([hidden]) { display: none; }
:host([size="sm"]) { --_s: 16px; --_d: 4px; --_h: 6px; --_l: 9px; }
:host([size="lg"]) { --_s: 40px; --_d: 8px; --_h: 16px; --_l: 18px; }
.wrap { display: grid; }
.wrap.labelled { gap: 0.5rem; }
.row { display: flex; align-items: center; gap: 0.5rem; }
.label { color: var(--_text); font-size: 0.8125rem; }
.vh { position: absolute; inline-size: 1px; block-size: 1px; margin: -1px; overflow: hidden; clip-path: inset(50%); white-space: nowrap; }

/* Spinner: eight pixel blocks, one lit at a time. */
.spinner { position: relative; flex: none; inline-size: var(--_s); block-size: var(--_s); }
.spinner span {
	position: absolute;
	inset: 0;
	margin: auto;
	inline-size: var(--_d);
	block-size: var(--_d);
	background: var(--_fill);
	opacity: 0.22;
	transform: rotate(calc(var(--i) * 45deg)) translateY(calc(var(--_s) / -2 + var(--_d) / 2));
	animation: sb-busy-blink 720ms steps(1, end) infinite;
	animation-delay: calc(var(--i) * 90ms);
}
@keyframes sb-busy-blink { 0% { opacity: 1; } 12.5%, 100% { opacity: 0.22; } }

/* Bar: a notched track with a fill; indeterminate sweeps a chunk across it. */
.bar {
	position: relative;
	overflow: hidden;
	block-size: var(--_h);
	background: var(--_track);
	box-shadow: 0 0 0 2px var(--_border);
	clip-path: ${notch('var(--_n)')};
	border-radius: calc(3px * (1 - var(--_notch)));
}
.bar .fill {
	position: absolute;
	inset-block: 0;
	left: 0;
	inline-size: var(--_p, 0%);
	background: var(--_fill);
	box-shadow: inset 0 -3px 0 var(--_edge);
	transition: inline-size 160ms steps(6, end);
}
.bar.sweeping .fill { inline-size: 34%; animation: sb-busy-sweep 1.1s steps(9, end) infinite; }
@keyframes sb-busy-sweep { from { translate: -105% 0; } to { translate: 320% 0; } }
:host(:dir(rtl)) .bar { scale: -1 1; }

/* Skeleton: notched blocks with a shimmer that steps across them. */
.skel { display: grid; gap: calc(var(--_l) * 0.7); }
.skel span {
	display: block;
	block-size: var(--_l);
	background-color: var(--_track);
	background-image: linear-gradient(90deg, transparent 0%, color-mix(in oklch, var(--_fill) 40%, transparent) 50%, transparent 100%);
	background-size: 180% 100%;
	background-repeat: no-repeat;
	box-shadow: 0 0 0 2px var(--_border);
	clip-path: ${notch('var(--_n)')};
	border-radius: calc(3px * (1 - var(--_notch)));
	animation: sb-busy-shimmer 1.4s steps(12, end) infinite;
	animation-delay: calc(var(--i) * 140ms);
}
.skel span:last-child:not(:first-child) { inline-size: 62%; }
@keyframes sb-busy-shimmer { from { background-position: -80% 0; } to { background-position: 180% 0; } }

/* No motion: every variant keeps a readable static state. */
@media (prefers-reduced-motion: reduce) {
	.spinner span { animation: none; opacity: 0.3; }
	.spinner span:first-child { opacity: 1; }
	.bar .fill { transition: none; }
	.bar.sweeping .fill { animation: none; inline-size: 100%; opacity: 0.5; }
	.skel span { animation: none; background-image: none; }
}
@media (forced-colors: active) {
	.spinner span, .bar .fill { forced-color-adjust: none; background: CanvasText; box-shadow: none; }
	.bar, .skel span { outline: 1px solid CanvasText; outline-offset: -1px; }
}
`

rocket('sb-busy', {
	props: ({ bool, number, oneOf, string }) => ({
		busy: bool.docs({ description: 'The server forces the indicator on. It wins: while this is true the indicator stays up whatever the watched requests do (busy="false" to clear it).' }),
		variant: oneOf('spinner', 'bar', 'skeleton').default('spinner').docs({ description: 'Shape: a spinner, a progress bar or skeleton lines. Set it as an attribute (it also picks the host display).' }),
		value: string.trim.docs({ description: 'Bar only: 0-100 for a determinate bar. Empty means indeterminate.' }),
		lines: number.clamp(1, 20).step(1).default(3).docs({ description: 'Skeleton only: how many placeholder lines.' }),
		label: string.trim.default('Loading').docs({ description: 'What is being waited for, e.g. "Loading flight plan". Read out by the status region; visible with show-label.' }),
		showLabel: bool.docs({ description: 'Show the label next to the indicator instead of only reading it out.' }),
		size: oneOf('sm', 'md', 'lg').default('md').docs({ description: 'Spinner box, bar height and skeleton line height.' }),
		for: string.trim.docs({ description: 'Which requests to watch: a CSS selector matching the element that triggers them (or one of its ancestors), "*" for every request on the page, empty for any request from inside the host\'s parent element.' }),
		delay: number.clamp(0, 10000).step(10).docs({ description: 'Milliseconds to wait before showing, so a fast request never flashes. Try 200.' }),
		min: number.clamp(0, 10000).step(10).docs({ description: 'Milliseconds to stay up once shown, so it does not flicker. Try 400.' }),
	}),
	manifest: {
		events: [
			{ name: 'sb-busy-change', kind: 'custom-event', bubbles: true, composed: true, description: 'When the visible state flips, after delay and min. detail: { busy }: e.g. to disable a form while it is up.' },
		],
	},
	// The structure only depends on variant and lines. Everything else (the
	// visible state, the value, the label) is driven through signals, and the
	// size through the host attribute, so a morph that flips busy or value does
	// not rebuild the shadow DOM or restart the animations.
	renderOnPropChange: ({ changes }) => 'variant' in changes || 'lines' in changes,
	setup: ({ $$, adoptStyles, cleanup, defineHostProp, effect, emit, host, observeProps, props }) => {
		adoptStyles(host, styles)
		// A property write on a disconnected host is not reflected, and the size
		// is styled from the attribute.
		props.size != 'md' && (host.size = props.size)

		// --- presentation: props into signals ---
		// $$.v is the bar's value, 0-100, or false for an indeterminate bar
		// (data-attr drops false: no aria-valuenow, aria-valuetext or --_p).
		const sync = () => {
			const raw = String(props.value ?? '').trim(), n = +raw
			$$.v = raw && isFinite(n) ? Math.min(100, Math.max(0, n)) : false
			$$.label = props.label
			$$.showLabel = props.showLabel
		}
		sync()
		observeProps(() => peek(sync), 'value', 'label', 'showLabel')

		// --- local state: never reflected to an attribute ---
		// A morph resets attributes, so the watched requests in flight, the
		// timers and the visible state live in closure variables ($$.on mirrors
		// `on` for the template). flight maps each triggering element to its
		// requests in flight: a request is closed by the element that opened it,
		// even once that element has left the document (a Delete button whose row
		// the response removed).
		let [flight, on, shownAt] = moved.get(host) || [new Map(), false, 0]
		let show = 0, hide = 0
		let bailed = false // the last watched request failed: drop it without waiting for min
		let ready = false
		$$.on = on
		const wanted = () => props.busy || flight.size > 0
		const paint = (v) => {
			if (on === v) return
			$$.on = on = v
			if (v) shownAt = performance.now()
			// During setup the page has not bound its data-on listener on the host
			// yet: tell it a microtask later, if that is still the state.
			ready ? emit('sb-busy-change', { busy: v }) : queueMicrotask(() => on === v && emit('sb-busy-change', { busy: v }))
		}
		// The effective state is (server busy OR a watched request in flight), put
		// through delay and min. Called from listeners and observeProps: it peeks.
		const settle = () => {
			if (wanted()) {
				clearTimeout(hide)
				hide = 0
				if (on || show) return
				if (props.delay > 0) show = setTimeout(() => ((show = 0), paint(true)), props.delay)
				else paint(true)
				return
			}
			clearTimeout(show) // never shown: the request beat the delay
			show = 0
			if (!on) return
			const left = bailed ? 0 : props.min - (performance.now() - shownAt)
			if (left > 0) {
				if (!hide) hide = setTimeout(() => ((hide = 0), settle()), left)
				return
			}
			clearTimeout(hide)
			hide = 0
			bailed = false
			paint(false)
		}
		observeProps(() => peek(settle), 'busy', 'delay', 'min')
		peek(settle) // the server may have rendered busy on the very first markup
		ready = true

		// --- watching Datastar requests ---
		// datastar-fetch is dispatched on the document and reaches every listener,
		// so a request counts only when it starts from an element this one
		// watches. A document listener has no attribute form, hence
		// addEventListener (removed in cleanup).
		const mine = (el) => {
			const sel = props.for
			if (sel === '*') return true
			if (sel) {
				try {
					return el === host || !!el.closest(sel)
				} catch {
					return false // not a selector
				}
			}
			return el === host || !!host.parentElement?.contains(el)
		}
		const onFetch = (e) => {
			const { type, el } = e.detail || {}
			const n = flight.get(el)
			if (type === 'started') {
				if (!mine(el)) return
				flight.set(el, (n || 0) + 1)
				bailed = false
			} else if (!n) return
			// Datastar fires finished in a finally, so it alone closes a request;
			// error and retries-failed only mark that this one failed, which drops
			// the indicator without waiting out min. retrying changes nothing.
			else if (type === 'finished') n > 1 ? flight.set(el, n - 1) : flight.delete(el)
			else return void (bailed ||= type === 'error' || type === 'retries-failed')
			peek(settle)
		}
		document.addEventListener('datastar-fetch', onFetch)

		// --- what the outside sees ---
		// :state(busy), a custom state: a morph cannot reset it.
		const states = internalsOf(host).states
		effect(() => {
			$$.on ? states.add('busy') : states.delete('busy')
		})
		defineHostProp('visible', { get: () => on })

		cleanup(() => {
			document.removeEventListener('datastar-fetch', onFetch)
			clearTimeout(show)
			clearTimeout(hide)
			// A move carries on in the next setup. A removal is final: the page
			// hears that the indicator is gone ($$ is torn down already).
			moved.set(host, host.isConnected && [flight, on, shownAt])
			if (on && !host.isConnected) emit('sb-busy-change', { busy: (on = false) })
		})
	},
	render: ({ html, props: { variant, lines } }) => html`
		<span class="vh" role="status" data-text="$$on ? $$label : ''"></span>
		<div class="wrap" part="base" data-show="$$on" data-class:labelled="$$showLabel">
			${variant === 'skeleton'
				? html`<div class="skel" part="skeleton" aria-hidden="true">
						${Array.from({ length: lines }, (_, i) => html`<span style="--i: ${i}"></span>`)}
					</div>`
				: null}
			${variant === 'bar'
				? html`<div class="bar" part="bar" role="progressbar" aria-labelledby="label"
						aria-valuemin="0" aria-valuemax="100" data-class:sweeping="$$v === false"
						data-attr:aria-valuenow="$$v" data-attr:aria-valuetext="$$v !== false && $$v + '%'"
						data-attr:style="$$v !== false && '--_p: ' + $$v + '%'">
						<span class="fill" part="fill"></span>
					</div>`
				: null}
			<div class="row">
				${variant === 'spinner'
					? html`<span class="spinner" part="spinner" aria-hidden="true">
							${Array.from({ length: 8 }, (_, i) => html`<span style="--i: ${i}"></span>`)}
						</span>`
					: null}
				<span class="label" part="label" id="label" aria-hidden="true" data-show="$$showLabel" data-text="$$label"></span>
			</div>
		</div>
	`,
})
