import { rocket, startPeeking, stopPeeking } from 'datastar'

// Reads that must not subscribe the caller's effect to a signal.
const peek = (fn) => {
	startPeeking()
	try {
		return fn()
	} finally {
		stopPeeking()
	}
}

// Types by grapheme, so an emoji or an accented letter arrives in one piece.
const segmenter = new Intl.Segmenter(undefined, { granularity: 'grapheme' })
const graphemes = (s) => Array.from(segmenter.segment(s), (g) => g.segment)

// Every phrase is laid out in full, invisibly, in one grid cell: the element
// takes the size of the longest from the start. The live line sits in the
// same cell as the typed part plus the untyped rest, laid out but hidden, so
// its line breaks are the final ones and nothing on the page ever moves.
// The cursor is an overlay moved with translate to where the rest begins:
// anything that moved through the layout with every character (even a
// pseudo-element of the hidden rest) would count as a layout shift each time,
// and an inline-block would let the word being typed break in two.
const styles = /* css */ `
:host {
	--_cursor: var(--sb-typewriter-cursor, var(--sb-brand-light, #B09AFF));
}
:host([hidden]) { display: none; }
.stack { position: relative; display: inline-grid; }
.stack > :not(.cursor) { grid-area: 1 / 1; }
.sizer, .rest { visibility: hidden; }
.cursor {
	/* physical: place() measures from the stack's top left, in RTL too */
	position: absolute;
	top: 0;
	left: 0;
	inline-size: 0.55em;
	block-size: 0.95em;
	background: var(--_cursor);
	pointer-events: none;
}
.bar .cursor { inline-size: max(2px, 0.08em); block-size: 1.1em; }
.underscore .cursor { block-size: max(2px, 0.12em); }
.none .cursor, .cursor:not(.placed) { display: none; }
.sr { position: absolute; inline-size: 1px; block-size: 1px; overflow: hidden; clip-path: inset(50%); white-space: nowrap; }
@media (prefers-reduced-motion: no-preference) {
	.blink .cursor { animation: sb-typewriter-blink 1.06s steps(1, end) infinite; }
}
@keyframes sb-typewriter-blink { 50% { opacity: 0; } }
@media (forced-colors: active) {
	.cursor { forced-color-adjust: none; background: CanvasText; }
}
@media print {
	.rest { visibility: visible; }
	.cursor { display: none; }
}
`

rocket('sb-typewriter', {
	props: ({ bool, json, number, oneOf, string }) => ({
		phrases: json.default(() => []).docs({ description: 'Phrases to type in turn, each erased before the next, as a JSON array of strings. Without it, the element\'s own text is typed once.' }),
		loop: bool.docs({ description: 'With phrases: start over after the last one, for as long as the element is on screen.' }),
		prompt: string.docs({ description: 'Text in front of the typed text that is there from the start, e.g. "$ ".' }),
		cursor: oneOf('block', 'bar', 'underscore', 'none').default('block').docs({ description: 'The cursor\'s shape. It blinks while it waits.' }),
		interval: number.min(1).default(45).docs({ description: 'Average milliseconds between two characters: lower types faster. Each varies a little, like a person typing; erasing takes half as long.' }),
		delay: number.min(0).default(400).docs({ description: 'Milliseconds before the first character, counted from when the element comes into view.' }),
		hold: number.min(0).default(1800).docs({ description: 'With phrases: milliseconds a typed phrase stays before it is erased.' }),
	}),
	manifest: {
		events: [
			{ name: 'sb-typed', kind: 'custom-event', bubbles: true, composed: true, description: 'When a phrase is fully typed. detail: { text, index }.' },
		],
	},
	// The text changes every few milliseconds through signals; props only
	// restart the typing.
	renderOnPropChange: false,
	setup: ({ $$, adoptStyles, host, observeProps, props }) => {
		adoptStyles(host, styles)
		const sync = () =>
			peek(() => {
				$$.prompt = props.prompt
				$$.shape = props.cursor
			})
		sync()
		observeProps(sync, 'prompt')
		observeProps(sync, 'cursor')
		$$.all = []
		$$.typed = ''
		$$.rest = ''
		$$.sr = ''
		$$.blink = true
		$$.at = ''
		$$.placed = false
	},
	onFirstRender: ({ $$, cleanup, emit, host, observeProps, props, refs: { stack, rest, cursor } }) => {
		const still = matchMedia('(prefers-reduced-motion: reduce)')
		let list = [] // [[graphemes of phrase 0], …]
		let k = 0 // the phrase
		let n = 0 // graphemes typed
		let phase = 'wait' // wait, type, hold, erase, done
		let timer = 0
		let visible = false

		// The phrases, else the element's own text (the server's, which is
		// also what readers without JavaScript and search engines get).
		const load = () => {
			const own = host.textContent.replace(/\s+/g, ' ').trim()
			const phrases = Array.isArray(props.phrases) ? props.phrases.map((p) => String(p ?? '')).filter(Boolean) : []
			const texts = phrases.length ? phrases : [own]
			list = texts.map(graphemes)
			$$.all = texts
			// Screen readers get the whole text at once, never the frames.
			$$.sr = own || texts.join(' ')
			k = 0
			restart()
		}
		// The cursor goes where the rest starts (an empty rest still has a
		// position), centred on that line; the underscore sits on its bottom.
		const place = () => {
			const box = stack.getBoundingClientRect()
			const r = rest.getClientRects()[0]
			if (!r) return
			const rtl = getComputedStyle(stack).direction === 'rtl'
			const w = cursor.offsetWidth || parseFloat(getComputedStyle(cursor).inlineSize) || 0
			const h = parseFloat(getComputedStyle(cursor).blockSize) || 0
			const x = rtl ? r.right - box.left - w : r.left - box.left
			const y = props.cursor === 'underscore' ? r.bottom - box.top - h - r.height * 0.08 : r.top - box.top + (r.height - h) / 2
			$$.at = `${Math.round(x * 2) / 2}px ${Math.round(y * 2) / 2}px`
			$$.placed = true
		}
		const show = () => {
			const g = list[k] || []
			$$.typed = g.slice(0, n).join('')
			$$.rest = g.slice(n).join('')
			requestAnimationFrame(place) // after the text is in the DOM
		}
		const wait = (ms) => {
			clearTimeout(timer)
			timer = setTimeout(step, ms)
		}
		const jitter = () => props.interval * (0.5 + Math.random())
		const step = () => {
			if (!visible) return // resumes when it comes back into view
			const g = list[k] || []
			if (phase === 'wait' || phase === 'type') {
				phase = 'type'
				$$.blink = false
				if (n < g.length) {
					n++
					show()
					return wait(jitter())
				}
				$$.blink = true
				emit('sb-typed', { text: g.join(''), index: k })
				if (list.length < 2 || (!props.loop && k === list.length - 1)) return void (phase = 'done')
				phase = 'hold'
				return wait(props.hold)
			}
			if (phase === 'hold') {
				phase = 'erase'
				$$.blink = false
			}
			if (phase === 'erase') {
				if (n > 0) {
					n--
					show()
					return wait(props.interval / 2)
				}
				k = (k + 1) % list.length
				phase = 'type'
				return wait(props.interval * 4)
			}
		}
		// Reduced motion: the whole (first) phrase at once, a still cursor.
		const restart = () => {
			clearTimeout(timer)
			n = 0
			phase = 'wait'
			if (still.matches) {
				const g = list[k] || []
				n = g.length
				phase = 'done'
				$$.blink = false
				show()
				// Shown at once, but complete all the same: listeners attached in
				// the same pass still hear it.
				return void queueMicrotask(() => emit('sb-typed', { text: g.join(''), index: k }))
			}
			show()
			$$.blink = true
			if (visible) wait(props.delay)
		}

		// Starts when it comes into view, and pauses while it is out of it.
		const io = new IntersectionObserver((entries) => {
			const was = visible
			visible = entries.at(-1).isIntersecting
			if (visible && !was && phase !== 'done') wait(phase === 'wait' ? props.delay : props.interval)
		})
		io.observe(host)
		// New line breaks (a resize, a web font arriving) move the cursor too.
		const ro = new ResizeObserver(place)
		ro.observe(stack)
		document.fonts?.ready.then(place)
		// New text from the server (a morph, data-text) types again.
		const watch = new MutationObserver(load)
		watch.observe(host, { childList: true, characterData: true, subtree: true })
		still.addEventListener('change', restart)
		// peek: the callback runs inside the effect of whoever set the
		// attribute (data-attr), which must not subscribe to what load reads.
		observeProps(() => peek(load), 'phrases')
		// A new cursor shape or prompt moves the cursor without a character typed.
		const replace = () => requestAnimationFrame(place)
		observeProps(replace, 'cursor')
		observeProps(replace, 'prompt')
		load()
		cleanup(() => {
			clearTimeout(timer)
			io.disconnect()
			ro.disconnect()
			watch.disconnect()
			still.removeEventListener('change', restart)
		})
	},
	render: ({ html }) => html`
		<span class="stack" part="text" data-ref:stack data-class:blink="$$blink" data-class:bar="$$shape === 'bar'" data-class:underscore="$$shape === 'underscore'" data-class:none="$$shape === 'none'">
			<template data-for="t in $$all">
				<span class="sizer" aria-hidden="true"><span part="prompt" data-text="$$prompt"></span><span data-text="t ?? ''"></span></span>
			</template>
			<span class="live" aria-hidden="true"><span class="prompt" part="prompt" data-text="$$prompt"></span><span data-text="$$typed"></span><span class="rest" data-ref:rest data-text="$$rest"></span></span>
			<span class="cursor" part="cursor" data-ref:cursor aria-hidden="true" data-style:translate="$$at" data-class:placed="$$placed"></span>
			<span class="sr" data-text="$$sr"></span>
		</span>
	`,
})
