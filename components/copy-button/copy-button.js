import { rocket, startPeeking, stopPeeking } from 'datastar'

// observeProps callbacks run inside the effect of whoever set the attribute.
const peek = (fn) => {
	startPeeking()
	try {
		return fn()
	} finally {
		stopPeeking()
	}
}

// attachInternals() works once and setup reruns on re-attach; custom states
// (:state(copied)) live outside the attributes, so morphs keep them.
const internals = new WeakMap()
const internalsOf = (host) => internals.get(host) ?? internals.set(host, host.attachInternals()).get(host)

const anchors = CSS.supports('anchor-name: --a')
// userAgentData (secure contexts only) says "macOS", navigator.platform "MacIntel".
const KEY = /mac|iphone|ipad|ipod/i.test(navigator.userAgentData?.platform ?? navigator.platform) ? '⌘C' : 'Ctrl+C'

// Without field-sizing: as many rows as the wrapped text fills, up to 6,
// measured without a scrollbar, which would narrow the lines.
const fit = (text) => {
	text.style.overflow = 'hidden'
	text.rows = 2
	const two = text.clientHeight
	text.rows = 1
	const line = two - text.clientHeight
	text.rows = Math.min(6, Math.round((text.scrollHeight - text.clientHeight) / line) + 1)
	text.style.overflow = ''
}

// Tokens with fallbacks, so the component also works outside Starbase.
// Forced colors drop the focus ring (a box-shadow): the transparent outline
// shows there instead. The tip is centred with a physical left, like its
// translate: a logical inset would put it off-centre in RTL.
const styles = /* css */ `
:host {
	--_bg: var(--sb-surface-raised, #10182B);
	--_border: var(--sb-border, #283552);
	--_border-strong: var(--sb-border-strong, #3A4868);
	--_text: var(--sb-text-2, #AEBBDD);
	--_text-hover: var(--sb-text-1, #F3F4FA);
	--_ok: var(--sb-ok, #6EF59A);
	--_danger: var(--sb-danger, #F2777A);
	--_on-danger: var(--sb-text-on-danger, #1B0A0C);
	--_tip-bg: var(--sb-surface-raised, #10182B);
	--_radius: var(--sb-radius-sm, 6px);
	--_focus: var(--sb-focus-ring, 0 0 0 2px #080D1D, 0 0 0 4px #B09AFF);
	display: inline-block;
	position: relative;
	vertical-align: middle;
}
:host([hidden]) { display: none; }
button {
	all: unset;
	position: relative;
	display: inline-grid;
	place-items: center;
	inline-size: 2rem;
	block-size: 2rem;
	/* The icon is half the button, whatever size a page gives it (::part(button)). */
	container-type: size;
	anchor-name: --sb-copy;
	border: 1px solid var(--_border);
	border-radius: var(--_radius);
	background: var(--_bg);
	color: var(--_text);
	cursor: pointer;
	transition: color 120ms, border-color 120ms, background 120ms;
}
button:hover { color: var(--_text-hover); border-color: var(--_border-strong); }
:focus-visible { box-shadow: var(--_focus); outline: 2px solid transparent; outline-offset: 2px; }
button.copied { color: var(--_ok); border-color: color-mix(in oklch, var(--_ok) 50%, transparent); }
button.failed { color: var(--_danger); border-color: color-mix(in oklch, var(--_danger) 50%, transparent); }
svg { inline-size: 50cqi; block-size: 50cqi; } /* half the button (the 2rem box inside its border) */
.tip {
	position: absolute;
	inset-block-end: calc(100% + 6px);
	left: 50%;
	padding: 2px 8px;
	border: 1px solid var(--_border-strong);
	border-radius: 4px;
	background: var(--_tip-bg);
	color: var(--_text-hover);
	font-size: 0.6875rem;
	font-weight: 600;
	white-space: nowrap;
	pointer-events: none;
	opacity: 0;
	translate: -50% 4px;
	transition: opacity 120ms, translate 120ms;
}
.tip.failed { border-color: var(--_danger); background: var(--_danger); color: var(--_on-danger); }
.tip.shown { opacity: 1; translate: -50% 0; }
/* The copy source: selectable, so never display: none; 12pt keeps iOS from zooming in on its focus. */
.source { position: fixed; inset-block-start: 0; inset-inline-start: 0; opacity: 0; pointer-events: none; font-size: 12pt; }
[popover] {
	margin: 0;
	padding: 0.5rem;
	border: 1px solid var(--_border-strong);
	border-radius: var(--_radius);
	background: var(--_bg);
	color: var(--_text-hover);
	font-size: 0.8125rem;
}
@supports (anchor-name: --a) {
	[popover] { position-anchor: --sb-copy; inset: auto; position-area: block-end span-inline-start; margin-block-start: 6px; position-try-fallbacks: flip-block, flip-inline; }
}
[popover] p { margin: 0 0 0.375rem; font-weight: 600; }
/* At least 12pt: iOS zooms in on a smaller field when it takes the focus. */
[popover] textarea {
	display: block;
	box-sizing: border-box;
	inline-size: min(28rem, 90vw);
	field-sizing: content;
	max-block-size: calc(6lh + 0.75rem + 2px);
	margin: 0;
	padding: 0.375rem 0.5rem;
	border: 1px solid var(--_border);
	border-radius: var(--_radius);
	background: none;
	color: inherit;
	font: inherit;
	font-size: max(1rem, 12pt);
	resize: none;
}
@media (forced-colors: active) {
	[popover] { outline: 1px solid CanvasText; outline-offset: -1px; }
}
`

rocket('sb-copy-button', {
	props: ({ string, number }) => ({
		value: string.docs({ description: 'The text copied to the clipboard.' }),
		label: string.default('Copy to clipboard').docs({ description: 'Accessible label of the button. A new label re-renders the button and closes an open panel.' }),
		copiedLabel: string.default('Copied!').docs({ description: 'Shown and announced after copying.' }),
		failedLabel: string.default('Copy failed').docs({ description: 'Shown and announced when copying failed and the page cancelled sb-copy-error (otherwise the panel opens).' }),
		manualLabel: string.default('Press {key} to copy').docs({ description: 'The panel\'s message, and the name of its text field, when the browser can\'t copy: {key} becomes ⌘C on Apple systems, else Ctrl+C.' }),
		resetMs: number.min(300).default(1600).docs({ description: 'How long the copied or failed state lasts, in ms (at least 300).' }),
	}),
	manifest: {
		events: [
			{ name: 'sb-copy', kind: 'custom-event', bubbles: true, composed: true, description: 'After copying. detail: { value, method }: "clipboard" (the Clipboard API), "execCommand" (a hidden textarea, e.g. on plain http://) or "manual" (the user copied the whole text from the panel).' },
			{ name: 'sb-copy-error', kind: 'custom-event', bubbles: true, composed: true, description: 'Cancelable. Neither the Clipboard API nor the hidden textarea could copy. detail: { value, error } (the Clipboard API\'s error name: "NotAllowedError" when refused, "TypeError" where there is none, e.g. on plain http://). Not cancelled, a panel selects the text for the user to copy; preventDefault() shows the cross and failed-label instead.' },
		],
	},
	setup: ({ $$, action, adoptStyles, cleanup, effect, emit, emitCancellable, host, observeProps, props }) => {
		adoptStyles(host, styles)
		$$.state = '' // '', 'copied', 'failed' or 'manual'
		// The message is read out by the status region (and shown in the tip).
		$$.message = () => ($$.state === 'copied' ? props.copiedLabel : $$.state === 'failed' ? props.failedLabel : '')
		$$.hint = () => props.manualLabel.replaceAll('{key}', KEY)
		const states = internalsOf(host).states
		effect(() => {
			const s = $$.state
			for (const k of ['copied', 'failed', 'manual']) k === s ? states.add(k) : states.delete(k)
		})
		const $ = (id) => host.shadowRoot.getElementById(id) // in the rendered template
		let timer = 0
		let panelValue = ''
		// A script focus that leaves a text field shows the ring, also after a
		// mouse click: the button gets back the ring it had when clicked.
		let ring = false
		const back = () => $('b').focus({ preventScroll: true, focusVisible: ring })
		const show = (state) => {
			clearTimeout(timer)
			$$.state = state
			if (state !== 'manual') timer = setTimeout(() => ($$.state = ''), props.resetMs)
		}
		const copied = (value, method) => {
			show('copied')
			emit('sb-copy', { value, method })
		}

		// In the shadow root the textarea is in the button's layer (a modal
		// dialog, a popover), never in inert content.
		const viaTextarea = (value) => {
			const source = document.createElement('textarea')
			source.className = 'source'
			source.readOnly = true
			source.value = value
			host.shadowRoot.append(source)
			let ok = false
			try {
				source.focus({ preventScroll: true })
				source.select()
				source.setSelectionRange(0, source.value.length)
				ok = document.execCommand('copy')
			} catch {}
			source.remove()
			back()
			return ok
		}

		// Without anchor positioning: fixed under the button (over it when there
		// is no room), ending where it ends, kept on screen.
		const place = (panel) => {
			if (anchors) return
			const r = $('b').getBoundingClientRect()
			const { width: w, height: h } = panel.getBoundingClientRect()
			const { clientWidth: vw, clientHeight: vh } = document.documentElement
			const x = host.matches(':dir(rtl)') ? r.left : r.right - w
			const y = r.bottom + 6 + h <= vh ? r.bottom + 6 : r.top - 6 - h
			Object.assign(panel.style, { position: 'fixed', inset: 'auto', left: Math.max(8, Math.min(x, vw - w - 8)) + 'px', top: Math.max(8, y) + 'px' })
		}
		const open = (value) => {
			const panel = $('manual')
			const text = $('text')
			panelValue = value
			text.value = value
			show('manual')
			if (!panel.matches(':popover-open')) panel.showPopover()
			if (getComputedStyle(text).fieldSizing !== 'content') fit(text)
			place(panel)
			text.focus({ preventScroll: true })
			text.select()
			text.setSelectionRange(0, text.value.length)
		}
		const close = () => $('manual')?.matches(':popover-open') && $('manual').hidePopover()

		action('copy', async () => {
			const value = props.value
			let error = 'TypeError'
			ring = $('b').matches(':focus-visible')
			if (navigator.clipboard?.writeText) {
				// Removed while the browser was writing: its signals are gone, don't bring them back.
				try {
					await navigator.clipboard.writeText(value)
					if (host.isConnected) copied(value, 'clipboard')
					return
				} catch (err) {
					if (!host.isConnected) return
					error = err?.name || String(err)
				}
			}
			// Without the API this runs in the click's own task, while it still
			// counts as a user activation.
			if (viaTextarea(value)) return copied(value, 'execCommand')
			const manual = emitCancellable('sb-copy-error', { value, error })
			if (host.isConnected) manual ? open(value) : show('failed')
		})
		// Only a copy of the whole text counts. The browser copies the selection
		// after this event, and a focus move now would lose it: close a task later.
		action('copied', ({ el }) => {
			if (el.selectionStart !== 0 || el.selectionEnd !== el.value.length) return
			copied(panelValue, 'manual')
			setTimeout(close)
		})
		// The last input in the panel decides whether the focus goes back with a ring.
		action('last', ({ evt }) => (ring = evt.type === 'keydown'))
		// Escape, light dismiss, hidePopover() or a copy.
		action('closing', ({ el, evt }) => {
			if (evt.newState !== 'closed') return
			if (el.contains(host.shadowRoot.activeElement)) back()
			if ($$.state === 'manual') $$.state = ''
		})
		// The re-render a new label brings would leave the panel open, or drop it
		// without a toggle event.
		observeProps(() => peek(close), 'label')
		cleanup(() => clearTimeout(timer))
	},
	// The template reads only label: a new value (e.g. data-attr'd on every
	// keystroke) needn't re-render and morph the shadow root.
	renderOnPropChange: ({ changes }) => 'label' in changes,
	// The status region sits outside the button: a button's children are
	// presentational, so a live region inside it may never be announced.
	render: ({ html, props: { label } }) => html`
		<button
			type="button"
			id="b"
			part="button"
			aria-label="${label}"
			data-on:click="@copy()"
			data-class:copied="$$state === 'copied'"
			data-class:failed="$$state === 'failed'"
		>
			<svg data-show="$$state !== 'copied' && $$state !== 'failed'" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><rect width="14" height="14" x="8" y="8" rx="2"/><path d="M4 16c-1.1 0-2-.9-2-2V4c0-1.1.9-2 2-2h10c1.1 0 2 .9 2 2"/></svg>
			<svg data-show="$$state === 'copied'" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M20 6 9 17l-5-5"/></svg>
			<svg data-show="$$state === 'failed'" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M18 6 6 18M6 6l12 12"/></svg>
		</button>
		<span class="tip" part="tip" role="status" data-class:shown="$$message" data-class:failed="$$state === 'failed'" data-text="$$message"></span>
		<div id="manual" part="manual" popover="auto" data-on:beforetoggle="@closing()" data-on:keydown="@last()" data-on:pointerdown="@last()">
			<p id="hint" data-text="$$hint"></p>
			<textarea id="text" part="manual-text" readonly spellcheck="false" aria-labelledby="hint" data-on:copy="@copied()"></textarea>
		</div>
	`,
})
