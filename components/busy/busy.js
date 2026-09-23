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
// runs again when the element is re-attached. Custom states (:state(busy)) and
// the ARIA properties live outside the attributes, so a server morph cannot
// reset them.
const internals = new WeakMap()
const internalsOf = (host) => internals.get(host) ?? internals.set(host, host.attachInternals()).get(host)

// Pixel corners: notches every corner by p (2px times --sb-notch; at 0 the
// border radius takes over).
const notch = (p) => `polygon(${p} 0, calc(100% - ${p}) 0, calc(100% - ${p}) ${p}, 100% ${p}, 100% calc(100% - ${p}), calc(100% - ${p}) calc(100% - ${p}), calc(100% - ${p}) 100%, ${p} 100%, ${p} calc(100% - ${p}), 0 calc(100% - ${p}), 0 ${p}, ${p} ${p})`

const styles = /* css */ `
:host {
	--_fill: var(--sb-brand, #8C6BFF);
	--_edge: var(--sb-brand-light, #B09AFF);
	--_track: var(--sb-surface-inset, #0B1224);
	--_border: var(--sb-border, #283552);
	--_text: var(--sb-text-2, #AEBBDD);
	--_notch: var(--sb-notch, 1);
	--_n: calc(2px * var(--_notch));
	display: block;
	inline-size: 100%;
}
/* The spinner sits in a line of content; the bar and the skeleton fill their
   column. Only :host can carry this, so set variant as an attribute. */
:host(:not([variant])), :host([variant="spinner"]) { display: inline-block; inline-size: auto; vertical-align: middle; }
.wrap {
	--_s: 24px;  /* spinner box */
	--_d: 5px;   /* spinner dot */
	--_h: 10px;  /* bar height */
	--_l: 12px;  /* skeleton line */
	position: relative;
	display: grid;
	gap: 0;
}
.wrap.sm { --_s: 16px; --_d: 4px; --_h: 6px; --_l: 9px; }
.wrap.lg { --_s: 40px; --_d: 8px; --_h: 16px; --_l: 18px; }
.wrap.labelled { gap: 0.5rem; }
.row { display: flex; align-items: center; gap: 0.5rem; }
.label { color: var(--_text); font-size: 0.8125rem; }
/* Visually hidden, still read out: the status region needs its text. */
.label.quiet { position: absolute; inline-size: 1px; block-size: 1px; margin: -1px; padding: 0; overflow: hidden; clip-path: inset(50%); white-space: nowrap; }

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
	inset-inline-start: 0;
	inline-size: var(--_p, 0%);
	background: var(--_fill);
	box-shadow: inset 0 -3px 0 var(--_edge);
	transition: inline-size 160ms steps(6, end);
}
.bar.sweeping .fill { inline-size: 34%; animation: sb-busy-sweep 1.1s steps(9, end) infinite; }
@keyframes sb-busy-sweep { from { translate: -105% 0; } to { translate: 320% 0; } }

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
	// visible state, the value, the label, the size) is driven through signals,
	// so a morph that flips busy or value does not rebuild the shadow DOM or
	// restart the animations.
	renderOnPropChange: ({ changes }) => 'variant' in changes || 'lines' in changes,
	setup: ({ $$, adoptStyles, cleanup, defineHostProp, effect, emit, host, observeProps, props }) => {
		adoptStyles(host, styles)

		// --- presentation: props into signals ---
		const asValue = () => {
			const raw = String(props.value ?? '').trim()
			const n = Number(raw)
			return raw === '' || !Number.isFinite(n) ? null : Math.min(100, Math.max(0, n))
		}
		const sync = () => {
			const v = asValue()
			$$.determinate = v !== null
			$$.value = v ?? 0
			$$.label = props.label
			$$.showLabel = props.showLabel
			$$.size = props.size
		}
		$$.on = false
		sync()
		observeProps(() => peek(sync), 'value', 'label', 'showLabel', 'size')
		$$.now = () => ($$.determinate ? $$.value : false) // data-attr drops false: an indeterminate bar has no aria-valuenow
		$$.valuetext = () => ($$.determinate ? `${$$.value}%` : false)
		$$.barStyle = () => `--_p: ${$$.determinate ? $$.value : 0}%`

		// --- local state: never reflected to an attribute ---
		// A morph resets attributes, so how many watched requests are in flight,
		// the timers and the animation state stay in closure variables and $$.
		let flight = 0 // matching requests in flight (started minus finished)
		let show = 0, hide = 0, shownAt = 0
		let bailed = false // the last watched request failed: drop it without waiting for min
		const wanted = () => props.busy || flight > 0
		const paint = (on) => {
			if ($$.on === on) return
			$$.on = on
			if (on) shownAt = performance.now()
			emit('sb-busy-change', { busy: on })
		}
		// The effective state is (server busy OR a watched request in flight), put
		// through delay and min. Called from listeners and observeProps: it peeks.
		const settle = () => {
			if (wanted()) {
				clearTimeout(hide)
				hide = 0
				if ($$.on || show) return
				if (props.delay > 0) show = setTimeout(() => ((show = 0), paint(true)), props.delay)
				else paint(true)
				return
			}
			clearTimeout(show) // never shown: the request beat the delay
			show = 0
			if (!$$.on) return
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

		// --- watching Datastar requests ---
		// datastar-fetch is dispatched on the document and reaches every listener,
		// so the first thing to check is whose request it was. A document listener
		// has no attribute form, hence addEventListener (removed in cleanup).
		const mine = (el) => {
			if (!el) return false
			const sel = props.for
			if (sel === '*') return true
			if (sel) {
				try {
					return el === host || el.matches(sel) || !!el.closest(sel)
				} catch {
					return false // not a selector
				}
			}
			const region = host.parentElement
			return el === host || (!!region && region.contains(el))
		}
		const onFetch = (e) => {
			const { type, el } = e.detail || {}
			if (!mine(el)) return
			// finished always follows started (Datastar fires it in a finally), so it
			// alone closes a request; error and retries-failed only mark that this
			// one failed, which drops the indicator without waiting out min.
			// retrying keeps it up: the request is not over.
			if (type === 'started') {
				flight++
				bailed = false
			} else if (type === 'finished') flight = Math.max(0, flight - 1)
			else if (type === 'error' || type === 'retries-failed') bailed = true
			else if (type !== 'retrying') return
			peek(settle)
		}
		document.addEventListener('datastar-fetch', onFetch)

		// --- what the outside sees ---
		// aria-busy through ElementInternals: a property, so a morph cannot strip
		// it. The attribute is a convenience for CSS and tests; :state(busy) is the
		// morph-proof hook for pages.
		const states = internalsOf(host).states
		effect(() => {
			const on = $$.on
			on ? states.add('busy') : states.delete('busy')
			try {
				internalsOf(host).ariaBusy = on ? 'true' : 'false'
			} catch {}
			host.setAttribute('aria-busy', on ? 'true' : 'false')
		})
		defineHostProp('visible', { get: () => peek(() => $$.on) })

		cleanup(() => {
			document.removeEventListener('datastar-fetch', onFetch)
			clearTimeout(show)
			clearTimeout(hide)
			flight = 0
		})
	},
	render: ({ html, props: { variant, lines } }) => html`
		<div class="wrap" part="base" role="status" aria-live="polite" data-show="$$on"
			data-class="{sm: $$size === 'sm', lg: $$size === 'lg', labelled: $$showLabel}">
			${variant === 'skeleton'
				? html`<div class="skel" part="skeleton" aria-hidden="true">
						${Array.from({ length: lines }, (_, i) => html`<span style="--i: ${i}"></span>`)}
					</div>`
				: null}
			${variant === 'bar'
				? html`<div class="bar" part="bar" role="progressbar" aria-labelledby="label"
						aria-valuemin="0" aria-valuemax="100" data-class:sweeping="!$$determinate"
						data-attr:aria-valuenow="$$now" data-attr:aria-valuetext="$$valuetext" data-attr:style="$$barStyle">
						<span class="fill" part="fill"></span>
					</div>`
				: null}
			<div class="row">
				${variant === 'spinner'
					? html`<span class="spinner" part="spinner" aria-hidden="true">
							${Array.from({ length: 8 }, (_, i) => html`<span style="--i: ${i}"></span>`)}
						</span>`
					: null}
				<span class="label" part="label" id="label" data-class:quiet="!$$showLabel" data-text="$$label"></span>
			</div>
		</div>
	`,
})
