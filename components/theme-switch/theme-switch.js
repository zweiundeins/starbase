import { rocket } from 'datastar'

// Icons for the well-known names; other themes show their label only.
const icon = (paths) => `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">${paths}</svg>`
const ICONS = {
	auto: icon('<rect x="3" y="4" width="18" height="12" rx="2"/><path d="M8 20h8M12 16v4"/>'),
	dark: icon('<path d="M20 14.5A8 8 0 0 1 9.5 4a8 8 0 1 0 10.5 10.5Z"/>'),
	light: icon('<circle cx="12" cy="12" r="4"/><path d="M12 2v2M12 20v2M4.9 4.9l1.4 1.4M17.7 17.7l1.4 1.4M2 12h2M20 12h2M4.9 19.1l1.4-1.4M17.7 6.3l1.4-1.4"/>'),
}
// Any other theme (the menu button shows it for them).
const PALETTE = icon('<path d="M12 3a9 9 0 1 0 0 18c1.1 0 2-.9 2-2 0-.5-.2-1-.5-1.3-.3-.4-.5-.8-.5-1.3 0-1.1.9-2 2-2h2.4A4.6 4.6 0 0 0 21 9.8C21 6 17 3 12 3Z"/><circle cx="7.5" cy="10.5" r="1"/><circle cx="10.5" cy="7" r="1"/><circle cx="15" cy="7" r="1"/>')

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
	--_notch: var(--sb-notch, 1);
	display: inline-flex;
	vertical-align: middle;
}
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
	cursor: pointer;
	transition: background 120ms, color 120ms;
}
label:hover { color: var(--_active); }
label:has(:checked) { background: var(--_brand-subtle); color: var(--_active); box-shadow: inset 0 0 0 1px var(--_brand); }
label:has(:focus-visible) { outline: 2px solid var(--_focus); outline-offset: 1px; }
input { position: absolute; opacity: 0; inset: 0; margin: 0; cursor: inherit; }
svg { inline-size: 1.05rem; block-size: 1.05rem; flex: none; }
.compact .text { position: absolute; inline-size: 1px; block-size: 1px; overflow: hidden; clip-path: inset(50%); white-space: nowrap; }
.compact .iconless .text { position: static; inline-size: auto; block-size: auto; clip-path: none; }
select {
	min-block-size: 2rem;
	padding: 0 2rem 0 0.75rem;
	border: 1px solid var(--_border);
	border-radius: var(--_radius);
	background: var(--_bg) url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 24 24' fill='none' stroke='%23AEBBDD' stroke-width='2'%3E%3Cpath d='m6 9 6 6 6-6'/%3E%3C/svg%3E") no-repeat right 0.5rem center / 1rem;
	color: var(--_active);
	font: inherit;
	font-size: 0.8125rem;
	appearance: none;
	cursor: pointer;
}
select:focus-visible { outline: 2px solid var(--_focus); outline-offset: 1px; }
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
.trigger svg { inline-size: 1.25rem; block-size: 1.25rem; }
.menu {
	margin: 0;
	padding: 4px;
	border: 1px solid var(--_border);
	border-radius: var(--_radius);
	background: var(--_bg);
	color: var(--_text);
	box-shadow: 0 12px 32px -12px rgb(0 0 0 / 0.6);
	/* Below the button, right-aligned (anchor positioning; centred where unsupported). */
	position-anchor: --sb-theme-trigger;
	inset: auto;
	position-area: bottom span-left;
	margin-block-start: 6px;
	position-try-fallbacks: flip-block, flip-inline;
}
.menu:popover-open { display: grid; gap: 2px; min-inline-size: 10rem; }
.menu label { justify-content: flex-start; padding-inline: 0.6rem 1rem; }
`

rocket('sb-theme-switch', {
	props: ({ array, bool, oneOf, string }) => ({
		themes: array(string.trim).default(() => ['auto', 'dark', 'light']).docs({ description: 'Theme names, in order. "auto" follows the system (prefers-color-scheme).' }),
		labels: array(string.trim).default(() => []).docs({ description: 'Visible names, in the order of themes (default: from the names).' }),
		attribute: string.trim.default('data-theme').docs({ description: 'Attribute set on <html> to the chosen theme. "auto" removes it.' }),
		cookie: string.trim.default('sb-theme').docs({ description: 'Cookie that remembers the choice (a year, whole site). Servers can read it to render the theme, with no flash.' }),
		variant: oneOf('segmented', 'select', 'menu').default('segmented').docs({ description: 'Radio buttons, a select, or an icon button with a menu (for headers).' }),
		compact: bool.docs({ description: 'Segmented only: icons without text for auto, dark and light.' }),
		label: string.trim.default('Theme').docs({ description: 'Accessible name of the control.' }),
	}),
	manifest: {
		events: [{ name: 'sb-theme-change', kind: 'custom-event', bubbles: true, composed: true, description: 'After the user picks a theme. detail: { theme, cookie }.' }],
	},
	setup: ({ $$, action, adoptStyles, emit, host, props }) => {
		adoptStyles(host, styles)
		const valid = (t) => props.themes.includes(t)
		const fallback = () => (valid('auto') ? 'auto' : props.themes[0])
		const saved = readCookie(props.cookie)
		$$.theme = valid(saved) ? saved : fallback()
		$$.options = props.themes.map((value, i) => ({ value, label: props.labels[i] || nameOf(value), icon: ICONS[value] || '' }))
		$$.icon = () => ICONS[$$.theme] || PALETTE
		$$.current = () => $$.options.find((o) => o.value === $$.theme)?.label ?? ''

		const root = document.documentElement
		const apply = (t) => (t === 'auto' ? root.removeAttribute(props.attribute) : root.setAttribute(props.attribute, t))
		// Normally the server (or a head snippet, see the docs) already set the
		// attribute before the first paint; this only repairs a page that didn't.
		if ((root.getAttribute(props.attribute) ?? 'auto') !== $$.theme) apply($$.theme)

		action('pick', ({ el }) => {
			const t = el.value
			if (!valid(t) || t === $$.theme) return
			$$.theme = t
			apply(t)
			const secure = location.protocol === 'https:' ? '; Secure' : ''
			document.cookie = `${props.cookie}=${encodeURIComponent(t)}; Path=/; Max-Age=31536000; SameSite=Lax${secure}`
			emit('sb-theme-change', { theme: t, cookie: props.cookie })
		})
		// Other switches for the same cookie follow along.
		action('sync', ({ evt }) => {
			const d = evt.detail
			if (evt.target !== host && d?.cookie === props.cookie && valid(d.theme)) $$.theme = d.theme
		})
	},
	render: ({ html, props: { variant, compact, label } }) =>
		variant === 'menu'
			? html`
				<button type="button" class="trigger" part="button" popovertarget="menu"
					data-attr:aria-label="'${label}: ' + $$current" data-attr:title="'${label}: ' + $$current"
					data-effect="el.innerHTML = $$icon"></button>
				<div id="menu" class="menu" part="menu" popover role="radiogroup" aria-label="${label}"
					data-on:sb-theme-change__window="@sync()">
					<template data-for="o in $$options">
						<label>
							<input type="radio" name="theme"
								data-attr:value="o.value"
								data-effect="el.checked = $$theme === o.value"
								data-on:change="@pick(); el.closest('[popover]').hidePopover()"/>
							<span data-show="o.icon" data-effect="el.innerHTML = o.icon"></span>
							<span class="text" data-text="o.label"></span>
						</label>
					</template>
				</div>`
			: variant === 'select'
			? html`
				<select part="select" aria-label="${label}"
					data-on:change="@pick()"
					data-on:sb-theme-change__window="@sync()">
					<template data-for="o in $$options">
						<option data-attr:value="o.value" data-text="o.label" data-effect="el.selected = $$theme === o.value"></option>
					</template>
				</select>`
			: html`
				<div class="group ${compact ? 'compact' : ''}" part="group" role="radiogroup" aria-label="${label}"
					data-on:sb-theme-change__window="@sync()">
					<template data-for="o in $$options">
						<label data-class:iconless="!o.icon" data-attr:title="o.label">
							<input type="radio" name="theme"
								data-attr:value="o.value"
								data-effect="el.checked = $$theme === o.value"
								data-on:change="@pick()"/>
							<span data-show="o.icon" data-effect="el.innerHTML = o.icon"></span>
							<span class="text" data-text="o.label"></span>
						</label>
					</template>
				</div>`,
})
