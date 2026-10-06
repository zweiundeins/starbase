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

const anchors = CSS.supports('anchor-name: --a')

// A heading without an id gets one from its text, unique in the document.
const slug = (text) =>
	text.normalize('NFKD').replace(/\p{Diacritic}/gu, '').toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/^-|-$/g, '') || 'section'
const uniqueId = (base) => {
	let id = base
	for (let n = 2; document.getElementById(id); n++) id = `${base}-${n}`
	return id
}

const styles = /* css */ `
:host {
	--_text: var(--sb-text-1, #F3F4FA);
	--_text-2: var(--sb-text-2, #AEBBDD);
	--_muted: var(--sb-text-muted, #7785A8);
	--_brand: var(--sb-brand, #8C6BFF);
	--_brand-light: var(--sb-brand-light, #B09AFF);
	--_border: var(--sb-border, #283552);
	--_surface: var(--sb-surface-card, #141D32);
	--_hover: var(--sb-surface-hover, #1A2540);
	--_step: var(--sb-frame-step, 3px);
	--_focus: var(--sb-focus-ring, 0 0 0 2px #080D1D, 0 0 0 4px #B09AFF);
	--_shadow: var(--sb-shadow-overlay, 0 12px 24px rgb(0 0 0 / 0.55));
	--_notch: var(--sb-notch, 1);
	--_n: calc(2px * var(--_notch));
	/* the gap between the progress bar's blocks: 2px in 8-bit, none when smooth */
	--_seg: calc(2px * var(--_notch));
	display: flex;
	flex-direction: column;
	min-block-size: 0;
	font-size: 0.875rem;
}
:host([hidden]) { display: none; }
nav { display: flex; flex-direction: column; gap: 0.75rem; min-block-size: 0; }

.head { display: flex; align-items: baseline; justify-content: space-between; gap: 1rem; }
.label, .count {
	font-family: var(--sb-font-display, inherit);
	font-size: 0.75rem;
	font-weight: 700;
	letter-spacing: 0.1em;
	text-transform: uppercase;
	color: var(--_muted);
}
.count { font-variant-numeric: tabular-nums; letter-spacing: 0.05em; }

/* Reading progress: blocks in 8-bit, a plain bar when smooth. */
.track { position: relative; block-size: 4px; background: var(--_border); overflow: hidden; }
.fill {
	position: absolute;
	inset-block: 0;
	inset-inline-start: 0;
	background: repeating-linear-gradient(to right, var(--_brand) 0 6px, transparent 6px calc(6px + var(--_seg)));
}

/* The list hangs on a rail; the current entry gets a pixel marker on it.
   The rail is a background inside the list (attached to its scrolling
   content), so the marker isn't cut off by the list's own scrolling. */
.panel:not([popover]) { display: flex; flex-direction: column; min-block-size: 0; }
ol {
	position: relative;
	min-block-size: 0;
	margin: 0;
	padding: 0 0 0 4px;
	list-style: none;
	overflow-y: auto;
	overscroll-behavior: contain;
	background: linear-gradient(var(--_border), var(--_border)) 2px 0 / 2px 100% no-repeat local;
}
ol:dir(rtl) { padding: 0 4px 0 0; background-position: right 2px top 0; }
li { margin: 0; }
li[hidden] { display: none; }
a {
	position: relative;
	display: block;
	padding: 0.3rem 0.75rem;
	padding-inline-start: calc(0.75rem + var(--_level, 0) * 0.875rem);
	color: var(--_text-2);
	text-decoration: none;
	line-height: 1.35;
	border-radius: 2px;
	outline: none;
}
a:hover { color: var(--_text); background: var(--_hover); }
a:focus-visible { box-shadow: var(--_focus); outline: 2px solid transparent; }
a[aria-current] { color: var(--_text); font-weight: 600; }
a[aria-current]::before {
	content: "";
	position: absolute;
	inset-inline-start: -4px;
	inset-block-start: calc(0.3rem + 0.675em - 3px);
	inline-size: 6px;
	block-size: 6px;
	background: var(--_brand);
	border-radius: calc(3px * (1 - var(--_notch)));
}

/* Compact: a bar with the current section, the list opens from it. */
.bar {
	all: unset;
	box-sizing: border-box;
	position: relative;
	display: grid;
	grid-template-columns: auto minmax(0, 1fr) auto auto;
	align-items: center;
	gap: 0.625rem;
	inline-size: calc(100% - 2 * var(--_step));
	margin: var(--_step);
	padding: 0.625rem 0.875rem 0.875rem;
	background: var(--_surface);
	color: var(--_text);
	cursor: pointer;
	anchor-name: --sb-toc;
	box-shadow:
		0 calc(-1 * var(--_step)) 0 0 var(--_border),
		0 var(--_step) 0 0 var(--_border),
		calc(-1 * var(--_step)) 0 0 0 var(--_border),
		var(--_step) 0 0 0 var(--_border);
}
.bar:hover { background: var(--_hover); }
.bar:focus-visible { outline: 2px solid var(--_brand-light); outline-offset: calc(var(--_step) * 2); }
.bar .now { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; font-weight: 600; }
.bar svg { inline-size: 12px; block-size: 8px; color: var(--_muted); transition: rotate 120ms; }
.bar[aria-expanded="true"] svg { rotate: 180deg; }
.bar .track { position: absolute; inset-inline: 0; inset-block-end: 0; block-size: 3px; }

/* The same stepped frame as the bar (a clip-path would cut the drop shadow). */
.panel[popover] {
	box-sizing: border-box;
	margin: var(--_step);
	padding: 0.5rem 0.5rem 0.5rem 0.75rem;
	border: 0;
	background: var(--_surface);
	color: inherit;
	max-block-size: min(60vh, 28rem);
	overflow: visible;
	flex-direction: column;
	box-shadow:
		0 calc(-1 * var(--_step)) 0 0 var(--_border),
		0 var(--_step) 0 0 var(--_border),
		calc(-1 * var(--_step)) 0 0 0 var(--_border),
		var(--_step) 0 0 0 var(--_border);
	filter: drop-shadow(var(--_shadow));
}
.panel[popover]:popover-open { display: flex; }
@supports (anchor-name: --a) {
	.panel[popover] {
		position-anchor: --sb-toc;
		inset: auto;
		position-area: block-end span-all;
		inline-size: calc(anchor-size(inline) - 2 * var(--_step));
		margin-block-start: calc(var(--_step) * 3 + 6px);
		position-try-fallbacks: flip-block;
	}
}

@media (prefers-reduced-motion: no-preference) {
	.fill { transition: inline-size 120ms linear; }
}
@media (forced-colors: active) {
	ol { border-inline-start: 2px solid CanvasText; }
	a[aria-current]::before, .fill { forced-color-adjust: none; background: Highlight; }
	a:focus-visible { outline: 2px solid Highlight; }
	.bar { border: 1px solid ButtonBorder; }
	.panel[popover] { outline: 1px solid CanvasText; outline-offset: -1px; }
}
`

rocket('sb-toc', {
	props: ({ bool, string }) => ({
		content: string.trim.docs({ description: 'CSS selector of the element whose headings it lists. Default: the closest <article>, else <main>, else the page. A list of links inside the element wins over it.' }),
		levels: string.trim.default('h2').docs({ description: 'The headings to list, e.g. "h2 h3". Deeper levels are indented.' }),
		label: string.trim.default('Contents').docs({ description: 'Heading of the list, and the accessible name of the navigation.' }),
		startLabel: string.trim.docs({ description: 'Adds a first entry that leads back to the start of the content, e.g. "Intro". It is the current one until the first heading is reached.' }),
		compact: string.trim.docs({ description: 'A media query. While it matches, the list folds into a bar with the current section and the reading progress, and opens from it, e.g. "(max-width: 64rem)".' }),
		progress: bool.docs({ description: 'Show the reading progress under the heading as well (the compact bar always shows it).' }),
	}),
	manifest: {
		events: [
			{ name: 'sb-section', kind: 'custom-event', bubbles: true, composed: true, description: 'When the reader scrolls into another section. detail: { id } (empty before the first heading).' },
		],
	},
	// Everything that changes goes through signals: a re-render would close
	// the compact list and drop the focus.
	renderOnPropChange: false,
	setup: ({ $$, adoptStyles, host, observeProps, props }) => {
		adoptStyles(host, styles)
		const sync = () =>
			peek(() => {
				$$.label = props.label || 'Contents'
				$$.start = props.startLabel
				$$.showProgress = props.progress
			})
		sync()
		observeProps(sync)
		$$.items = []
		$$.active = ''
		$$.current = ''
		$$.counter = ''
		$$.progress = 0
		$$.compact = false
		$$.open = false
	},
	onFirstRender: ({ $$, action, cleanup, emit, host, observeProps, props, refs: { bar, panel, list } }) => {
		// The sections: [{ id, text, level, el }], in plain closure state; the
		// template only gets what it shows.
		let items = []
		let content = null
		let shown = ''

		const resolve = () =>
			(props.content && document.querySelector(props.content)) || host.closest('article') || document.querySelector('main') || document.body

		// A list of links inside the element (the server's, and what readers
		// without JavaScript see) wins; else the headings of the content.
		const collect = () => {
			content = resolve()
			const links = [...host.querySelectorAll('a[href^="#"]')].filter((a) => a.hash.length > 1)
			if (links.length) {
				items = links.map((a) => {
					const id = decodeURIComponent(a.hash.slice(1))
					let level = -1
					for (let n = a.closest('li'); n && host.contains(n); n = n.parentElement.closest('li')) level++
					return { id, text: a.textContent.trim(), level: Math.max(level, 0), el: document.getElementById(id) }
				})
			} else {
				const found = content ? [...content.querySelectorAll(headings())].filter((h) => !h.closest('sb-toc')) : []
				const top = Math.min(...found.map((h) => +h.tagName[1]))
				items = found.map((h) => {
					const text = h.textContent.trim()
					if (!h.id) h.id = uniqueId(slug(text))
					return { id: h.id, text, level: +h.tagName[1] - top, el: h }
				})
			}
			items = items.filter((i) => i.el && i.text)
			const json = JSON.stringify(items.map(({ id, text, level }) => [id, text, level]))
			if (json !== shown) {
				shown = json
				$$.items = items.map(({ id, text, level }) => ({ id, text, level }))
			}
			measure(true)
		}

		// The current section: the last heading above a line 30% down the
		// viewport, below the page's scroll padding. At the very bottom of the
		// page, the last one, however short it is.
		let first = true
		const measure = (force) => {
			frame = 0
			const root = document.documentElement
			const pad = parseFloat(getComputedStyle(root).scrollPaddingTop) || 0
			const line = pad + (innerHeight - pad) * 0.3
			let idx = -1
			items.forEach((item, i) => item.el.getBoundingClientRect().top <= line && (idx = i))
			const last = items.at(-1)
			if (last && innerHeight + scrollY >= root.scrollHeight - 2 && last.el.getBoundingClientRect().top < innerHeight) idx = items.length - 1
			const start = peek(() => $$.start)
			const id = idx >= 0 ? items[idx].id : ''
			const total = items.length + (start ? 1 : 0)
			if (force || id !== peek(() => $$.active)) {
				$$.active = id
				$$.current = idx >= 0 ? items[idx].text : start || items[0]?.text || ''
				$$.counter = total ? `${Math.max(idx + (start ? 2 : 1), 1)}/${total}` : ''
				if (!first && !force) emit('sb-section', { id })
				requestAnimationFrame(reveal)
			}
			first = false
			// Progress: 0 when the content's top reaches the line, 100 when its
			// end comes into view (or, for content shorter than that, when its
			// end passes the line).
			if (content) {
				const r = content.getBoundingClientRect()
				const span = r.height - (innerHeight - line)
				const p = span > 0 ? (line - r.top) / span : (line - r.top) / (r.height || 1)
				$$.progress = Math.round(Math.min(1, Math.max(0, p)) * 1000) / 10
			}
		}
		// Keeps the current entry in view when the list scrolls (a long list
		// in a sticky sidebar).
		const reveal = () => {
			const a = list.querySelector('[aria-current]')
			if (!a || list.scrollHeight <= list.clientHeight) return
			const top = a.offsetTop
			if (top < list.scrollTop || top + a.offsetHeight > list.scrollTop + list.clientHeight) list.scrollTop = top - list.clientHeight / 3
		}

		let frame = 0
		const schedule = () => (frame ||= requestAnimationFrame(() => measure(false)))
		addEventListener('scroll', schedule, { passive: true })
		addEventListener('resize', schedule)

		// New markup collects again, once per frame: any change to the list
		// inside the element, and changes in the content that touch a heading
		// (busy content, like a live demo, would otherwise re-scan constantly).
		let pending = 0
		const again = () => (pending ||= requestAnimationFrame(() => ((pending = 0), collect())))
		const headings = () => (props.levels || 'h2').split(/[\s,]+/).filter((t) => /^h[1-6]$/i.test(t)).join(',') || 'h2'
		const touches = (r) => {
			if (host.contains(r.target)) return true
			const sel = headings()
			const hit = (n) => n.nodeType === 1 && (n.matches(sel) || n.querySelector(sel))
			if (r.type === 'childList') return [...r.addedNodes, ...r.removedNodes].some(hit) || !!r.target.closest?.(sel)
			return !!(r.target.nodeType === 1 ? r.target : r.target.parentElement)?.closest(sel)
		}
		const watch = new MutationObserver((records) => records.some(touches) && again())
		const observe = () => {
			watch.disconnect()
			watch.observe(host, { childList: true, subtree: true, characterData: true, attributes: true, attributeFilter: ['href'] })
			if (content && content !== document.body) watch.observe(content, { childList: true, subtree: true, characterData: true, attributes: true, attributeFilter: ['id'] })
		}
		collect()
		observe()
		const rescan = () => (collect(), observe())
		observeProps(rescan, 'content')
		observeProps(rescan, 'levels')
		observeProps(() => measure(true), 'startLabel')

		// Compact while the media query matches; leaving it closes the list.
		let mq = null
		const apply = () => {
			const on = !!mq?.matches
			if (!on && panel.matches(':popover-open')) panel.hidePopover()
			if (!on) panel.removeAttribute('style') // the hand placement, without anchor positioning
			$$.compact = on
		}
		const media = () => {
			mq?.removeEventListener('change', apply)
			mq = props.compact ? matchMedia(props.compact) : null
			mq?.addEventListener('change', apply)
			apply()
		}
		media()
		observeProps(media, 'compact')

		cleanup(() => {
			removeEventListener('scroll', schedule)
			removeEventListener('resize', schedule)
			cancelAnimationFrame(frame)
			cancelAnimationFrame(pending)
			watch.disconnect()
			mq?.removeEventListener('change', apply)
		})

		// The list closing with the focus inside (Escape) gives it back to the
		// bar; a picked section keeps it, the page moves on to that section.
		let refocus = false
		let picking = false
		const root = host.shadowRoot
		const close = () => {
			if (!panel.matches(':popover-open')) return
			picking = true // hidePopover() runs beforetoggle synchronously
			panel.hidePopover()
			picking = false
		}
		// A link scrolls the page as any anchor does (the page's scroll
		// padding and scroll-behavior apply) and closes the compact list.
		action('go', close)
		action('start', ({ evt }) => {
			evt.preventDefault()
			close()
			;(content && content !== document.body ? content : document.documentElement).scrollIntoView({ block: 'start' })
			history.replaceState(history.state, '', location.pathname + location.search)
		})
		action('toggled', ({ evt }) => {
			$$.open = evt.newState === 'open'
			if ($$.open) requestAnimationFrame(reveal)
			else if (refocus) bar.focus()
			refocus = false
		})
		// Without anchor positioning the list is placed under the bar by hand.
		action('place', ({ evt }) => {
			if (evt.newState === 'closed') refocus = !picking && panel.contains(root.activeElement)
			if (anchors || evt.newState !== 'open') return
			const r = bar.getBoundingClientRect()
			Object.assign(panel.style, { position: 'fixed', inset: 'auto', top: `${r.bottom + 6}px`, left: `${r.left}px`, width: `${r.width}px` })
		})
	},
	render: ({ html }) => html`
		<nav part="nav" data-attr:aria-label="$$label">
			<div class="head" part="head" data-show="!$$compact">
				<span class="label" part="label" data-text="$$label"></span>
				<span class="count" part="counter" aria-hidden="true" data-text="$$counter"></span>
			</div>
			<div class="track" part="progress" aria-hidden="true" data-show="!$$compact && $$showProgress">
				<span class="fill" data-style:inline-size="$$progress + '%'"></span>
			</div>
			<button class="bar" part="bar" type="button" popovertarget="panel" data-ref:bar data-show="$$compact"
				data-attr:aria-expanded="String($$open)">
				<span class="label" data-text="$$label"></span>
				<span class="now" data-text="$$current"></span>
				<span class="count" aria-hidden="true" data-text="$$counter"></span>
				<svg viewBox="0 0 6 4" shape-rendering="crispEdges" fill="currentColor" aria-hidden="true"><path d="M0 0h6v1H0zM1 1h4v1H1zM2 2h2v1H2z"/></svg>
				<span class="track" aria-hidden="true"><span class="fill" data-style:inline-size="$$progress + '%'"></span></span>
			</button>
			<div class="panel" id="panel" part="panel" data-ref:panel
				data-attr:popover="$$compact ? 'auto' : null"
				data-on:beforetoggle="@place()"
				data-on:toggle="@toggled()">
				<ol part="list" data-ref:list>
					<li data-attr:hidden="!$$start">
						<a href="#" part="link" data-attr:aria-current="$$start && $$active === '' ? 'location' : null" data-on:click="@start()" data-text="$$start"></a>
					</li>
					<!-- s?.: when the list shrinks, data-for can re-evaluate a removed row once with s undefined.
					     No ids on these repeated elements: the morph would park and move them. -->
					<template data-for="s in $$items">
						<li>
							<a part="link"
								data-attr:href="'#' + (s?.id ?? '')"
								data-style:--_level="s?.level ?? 0"
								data-attr:aria-current="s && $$active === s.id ? 'location' : null"
								data-on:click="@go()"
								data-text="s?.text"></a>
						</li>
					</template>
				</ol>
			</div>
		</nav>
	`,
})
