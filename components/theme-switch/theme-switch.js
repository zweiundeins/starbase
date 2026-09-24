import { rocket, startPeeking, stopPeeking } from 'datastar'

// Icons for the well-known names, as CSS masks painted in currentColor:
// the markup only picks a class (no HTML injected). Other themes get the
// palette icon on the menu button, and text only elsewhere.
const ICON_PATHS = {
	auto: '<rect x="3" y="4" width="18" height="12" rx="2"/><path d="M8 20h8M12 16v4"/>',
	dark: '<path d="M20 14.5A8 8 0 0 1 9.5 4a8 8 0 1 0 10.5 10.5Z"/>',
	light: '<circle cx="12" cy="12" r="4"/><path d="M12 2v2M12 20v2M4.9 4.9l1.4 1.4M17.7 17.7l1.4 1.4M2 12h2M20 12h2M4.9 19.1l1.4-1.4M17.7 6.3l1.4-1.4"/>',
	palette: '<path d="M12 3a9 9 0 1 0 0 18c1.1 0 2-.9 2-2 0-.5-.2-1-.5-1.3-.3-.4-.5-.8-.5-1.3 0-1.1.9-2 2-2h2.4A4.6 4.6 0 0 0 21 9.8C21 6 17 3 12 3Z"/><circle cx="7.5" cy="10.5" r="1"/><circle cx="10.5" cy="7" r="1"/><circle cx="15" cy="7" r="1"/>',
}
// The SVG goes into the data URL as it is: inside url('…') a data URL takes
// <, > and spaces literally, and these icons hold no ', #, % or \ to escape.
const iconCSS = Object.entries(ICON_PATHS)
	.map(([name, paths]) => `.icon.${name} { --_mask: url('data:image/svg+xml,<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="black" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">${paths}</svg>'); }`)
	.join('')
const iconOf = (theme) => (theme in ICON_PATHS && theme !== 'palette' ? theme : '')

// Text inside a single-quoted string of a Datastar expression.
const quote = (s) => s.replace(/[\\']/g, '\\$&')

// schemeOf is the colour scheme the page actually paints in, whatever the
// theme is called: a theme that sets its own color-scheme (deep-space → dark,
// daylight → light) decides, and only "light dark" (or none) defers to the
// system. Consumers get it in sb-theme-change, so a canvas needs no list of
// theme names to pick its palette.
const schemeOf = (el) => {
	const cs = getComputedStyle(el).colorScheme
	const dark = /\bdark\b/.test(cs)
	const decided = dark !== /\blight\b/.test(cs)
	return (decided ? dark : matchMedia('(prefers-color-scheme: dark)').matches) ? 'dark' : 'light'
}

const nameOf = (t) => t.charAt(0).toUpperCase() + t.slice(1).replaceAll('-', ' ')

const readCookie = (name) => {
	const c = document.cookie.split('; ').find((c) => c.startsWith(name + '='))
	try {
		return c ? decodeURIComponent(c.slice(name.length + 1)) : ''
	} catch {
		return ''
	}
}

const styles = /* css */ `
:host {
	--_bg: var(--sb-control-bg, #0B1224);
	--_border: var(--sb-control-border, #283552);
	--_text: var(--sb-text-2, #AEBBDD);
	--_active: var(--sb-text-1, #F3F4FA);
	--_brand: var(--sb-brand, #8C6BFF);
	--_brand-subtle: var(--sb-brand-subtle, rgb(140 107 255 / 0.14));
	--_focus: var(--sb-brand-light, #B09AFF);
	--_radius: var(--sb-control-radius, 6px);
	display: inline-flex;
	vertical-align: middle;
}
:host([hidden]) { display: none; }
.group {
	display: inline-flex;
	gap: 2px;
	padding: 2px;
	border: 1px solid var(--_border);
	border-radius: var(--_radius);
	background: var(--_bg);
}
label {
	position: relative;
	display: inline-flex;
	align-items: center;
	gap: 0.4em;
	min-block-size: 2rem;
	padding-inline: 0.6rem;
	border-radius: calc(var(--_radius) - 2px);
	color: var(--_text);
	font-size: 0.8125rem;
	line-height: 1;
	cursor: pointer;
	transition: background 120ms, color 120ms;
}
label:hover { color: var(--_active); }
label:has(:checked) { background: var(--_brand-subtle); color: var(--_active); box-shadow: inset 0 0 0 1px var(--_brand); }
label:has(:focus-visible) { outline: 2px solid var(--_focus); outline-offset: 1px; }
input { position: absolute; opacity: 0; inset: 0; margin: 0; cursor: inherit; }
/* Always a flex or grid item, so it is a block without saying so. */
.icon { flex: none; inline-size: 1.05rem; block-size: 1.05rem; background: currentColor; mask: var(--_mask) center / contain no-repeat; }
/* Compact: a name with an icon is for screen readers (and the tooltip) only. */
.compact .text { overflow: hidden; white-space: nowrap; }
.compact :not(.iconless) > .text { position: absolute; inline-size: 1px; block-size: 1px; clip-path: inset(50%); }
select {
	min-block-size: 2rem;
	padding: 0;
	padding-inline: 0.75rem 2rem;
	border: 1px solid var(--_border);
	border-radius: var(--_radius);
	background: var(--_bg);
	color: var(--_active);
	font: inherit;
	font-size: 0.8125rem;
	appearance: none;
	cursor: pointer;
}
select:focus-visible { outline: 2px solid var(--_focus); outline-offset: 1px; }
/* The select's arrow, drawn over it in the options' text colour, so it follows the theme. */
.picker { position: relative; display: inline-flex; }
.picker::after {
	content: "";
	position: absolute;
	inset-inline-end: calc(0.5rem + 1px); /* inside the select's border */
	inset-block-start: 50%;
	inline-size: 1rem;
	block-size: 1rem;
	translate: 0 -50%;
	background: var(--_text);
	mask: url('data:image/svg+xml,<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="black" stroke-width="2"><path d="m6 9 6 6 6-6"/></svg>') center / contain no-repeat;
	pointer-events: none;
}
.trigger {
	all: unset;
	display: grid;
	place-items: center;
	inline-size: 2.25rem;
	block-size: 2.25rem;
	border-radius: var(--_radius);
	color: var(--_text);
	cursor: pointer;
	anchor-name: --sb-theme-trigger;
}
.trigger:hover, .trigger:has(+ :popover-open) { color: var(--_active); background: var(--_brand-subtle); }
.trigger:focus-visible { outline: 2px solid var(--_focus); outline-offset: 1px; }
.trigger .icon { inline-size: 1.25rem; block-size: 1.25rem; }
.menu {
	padding: 4px;
	border: 1px solid var(--_border);
	border-radius: var(--_radius);
	background: var(--_bg);
	color: var(--_text);
	box-shadow: 0 12px 32px -12px #0009;
}
/* Below the button, right-aligned (anchor positioning; centred where unsupported). */
@supports (anchor-name: --a) {
	.menu {
		position-anchor: --sb-theme-trigger;
		inset: auto;
		position-area: bottom span-left;
		margin: 0;
		margin-block-start: 6px;
		position-try-fallbacks: flip-block, flip-inline;
	}
}
.menu:popover-open { display: grid; gap: 2px; min-inline-size: 10rem; }
.menu label { padding-inline: 0.6rem 1rem; }
@media (forced-colors: active) {
	.icon, .picker::after { forced-color-adjust: none; background: CanvasText; }
	label:has(:checked) { outline: 2px solid Highlight; }
}
`

rocket('sb-theme-switch', {
	props: ({ array, bool, oneOf, string }) => ({
		themes: array(string.trim).default(() => ['auto', 'dark', 'light']).docs({ description: 'Theme names, in order. "auto" follows the system (prefers-color-scheme).' }),
		labels: array(string.trim).default(() => []).docs({ description: 'Visible names, in the order of themes (default: from the names).' }),
		attribute: string.trim.default('data-theme').docs({ description: 'Attribute set on <html> to the chosen theme. "auto" removes it.' }),
		cookie: string.trim.default('sb-theme').docs({ description: 'Cookie that remembers the choice (a year, whole site). Servers can read it to render the theme, with no flash.' }),
		domain: string.trim.docs({ description: 'Cookie domain, e.g. ".example.com" to share the choice with every subdomain. Default: this host only.' }),
		variant: oneOf('segmented', 'select', 'menu').default('segmented').docs({ description: 'Radio buttons, a select, or an icon button with a menu (for headers).' }),
		compact: bool.docs({ description: 'Segmented only: icons without text for auto, dark and light.' }),
		label: string.trim.default('Theme').docs({ description: 'Accessible name of the control.' }),
	}),
	manifest: {
		events: [{ name: 'sb-theme-change', kind: 'custom-event', bubbles: true, composed: true, description: 'After the user picks a theme. detail: { theme, cookie, scheme }, where scheme is "light" or "dark": what the page now paints in.' }],
	},
	setup: ({ $$, action, adoptStyles, cleanup, emit, host, observeProps, props }) => {
		adoptStyles(host, styles + iconCSS)
		const valid = (t) => props.themes.includes(t)
		// A saved theme this switch doesn't list (another switch on the site has a
		// longer list) stays: no option is checked, and the page is left alone.
		$$.theme = readCookie(props.cookie) || (valid('auto') ? 'auto' : props.themes[0])
		// Peeking: observeProps runs this inside the effect of whoever set the
		// attribute (data-attr), which must not subscribe to $$.options.
		const list = () => {
			startPeeking()
			try {
				$$.options = props.themes.map((value, i) => ({ value, label: props.labels[i] || nameOf(value), icon: iconOf(value) }))
			} finally {
				stopPeeking()
			}
		}
		list()
		observeProps(list, 'themes', 'labels')
		$$.icon = () => iconOf($$.theme) || 'palette'
		// ?. because disconnecting clears $$ and this runs once more.
		$$.current = () => $$.options?.find((o) => o.value === $$.theme)?.label ?? nameOf($$.theme || '')

		const root = document.documentElement
		const apply = (t) => (t === 'auto' ? root.removeAttribute(props.attribute) : root.setAttribute(props.attribute, t))
		// Normally the server (or a head snippet, see the docs) already set the
		// attribute before the first paint; this only repairs a page that didn't.
		if (valid($$.theme) && (root.getAttribute(props.attribute) ?? 'auto') !== $$.theme) apply($$.theme)

		// A domain widens the choice to every subdomain that shares it; without one
		// the cookie stays on this host, which is the safe default. A domain the
		// page does not belong to is refused by the browser without a word, so the
		// write is read back and falls back to this host rather than losing the
		// choice silently.
		const save = (t) => {
			const secure = location.protocol === 'https:' ? '; Secure' : ''
			const value = `${props.cookie}=${encodeURIComponent(t)}; Path=/; Max-Age=31536000; SameSite=Lax${secure}`
			if (!props.domain) return void (document.cookie = value)
			// Drop a host-only cookie of the same name first: both would be sent,
			// and which one the server reads is undefined.
			document.cookie = `${props.cookie}=; Path=/; Max-Age=0; SameSite=Lax${secure}`
			document.cookie = `${value}; Domain=${props.domain}`
			if (readCookie(props.cookie) === t) return
			document.cookie = value
			reportError(new Error(`<sb-theme-switch> domain="${props.domain}" was refused by the browser (this page is ${location.hostname}); the theme is remembered for this host only`))
		}

		action('pick', ({ el }) => {
			const t = el.value
			if (!valid(t) || t === $$.theme) return
			$$.theme = t
			apply(t)
			save(t)
			emit('sb-theme-change', { theme: t, cookie: props.cookie, scheme: schemeOf(root) })
		})
		// Other switches for the same cookie follow along, also to a theme this
		// one doesn't list (it then shows no choice).
		const sync = ({ detail: d, target }) => target !== host && d?.cookie === props.cookie && ($$.theme = d.theme)
		addEventListener('sb-theme-change', sync)
		cleanup(() => removeEventListener('sb-theme-change', sync))
	},
	render: ({ html, props: { variant, compact, label } }) =>
		variant === 'menu'
			? html`
				<button type="button" class="trigger" part="button" popovertarget="menu"
					data-attr:aria-label="'${quote(label)}: ' + $$current" data-attr:title="'${quote(label)}: ' + $$current"
					><span data-attr:class="'icon ' + $$icon" aria-hidden="true"></span></button>
				<div id="menu" class="menu" part="menu" popover role="radiogroup" aria-label="${label}"
					data-on:click="evt.detail && el.hidePopover()"
					data-on:keyup="['Enter', ' '].includes(evt.key) && (evt.target.click(), el.hidePopover())"
					data-on:beforetoggle="el.matches(':focus-within') && el.previousElementSibling.focus()">
					<template data-for="o in $$options">
						<label data-attr:part="$$theme === o?.value ? 'option selected' : 'option'">
							<input type="radio" name="theme"
								data-attr:value="o?.value"
								data-effect="el.checked = $$theme === o?.value"
								data-on:change="@pick()"/>
							<span aria-hidden="true" data-show="o?.icon" data-attr:class="'icon ' + o?.icon"></span>
							<span class="text" data-text="o?.label"></span>
						</label>
					</template>
				</div>`
			: variant === 'select'
			? html`
				<span class="picker"><select part="select" aria-label="${label}"
					data-on:change="@pick()">
					<option hidden></option>
					<template data-for="o in $$options">
						<option data-attr:value="o?.value" data-text="o?.label" data-effect="el.selected = $$theme === o?.value"></option>
					</template>
				</select></span>`
			: html`
				<div class="group ${compact ? 'compact' : ''}" part="group" role="radiogroup" aria-label="${label}">
					<template data-for="o in $$options">
						<label data-class:iconless="!o?.icon" data-attr:title="o?.label" data-attr:part="$$theme === o?.value ? 'option selected' : 'option'">
							<input type="radio" name="theme"
								data-attr:value="o?.value"
								data-effect="el.checked = $$theme === o?.value"
								data-on:change="@pick()"/>
							<span aria-hidden="true" data-show="o?.icon" data-attr:class="'icon ' + o?.icon"></span>
							<span class="text" data-text="o?.label"></span>
						</label>
					</template>
				</div>`,
})
