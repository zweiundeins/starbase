---
name: Theme Switch
tag: sb-theme-switch
category: utilities
summary: Auto, dark or light (or your own themes), remembered in a cookie, with no flash.
author: zweiundeins
tags: [theme, dark mode, color scheme, preferences]
since: 2026-09-22
preview: |
  <sb-theme-switch cookie="sb-theme-demo" attribute="data-demo-theme"></sb-theme-switch>
usage: |
  <sb-theme-switch></sb-theme-switch>
playground:
  exclude: [themes, labels, cookie, attribute, domain]
  attrs: {cookie: sb-theme-demo, attribute: data-demo-theme}
---

Lets people pick a theme. The choice goes on `<html>` as an attribute (`data-theme="dark"` by default), and "auto" removes the attribute so your CSS can follow `prefers-color-scheme`. It is remembered in a cookie, so the next page can render the right theme before the first paint.

## Examples

### Auto, dark, light

```html preview
<sb-theme-switch cookie="sb-theme-demo" attribute="data-demo-theme"></sb-theme-switch>
```

### Compact

Icons only; the names stay available to screen readers and as tooltips.

```html preview
<sb-theme-switch compact cookie="sb-theme-demo" attribute="data-demo-theme"></sb-theme-switch>
```

### Your own themes

Any names work, and `labels` sets the visible names. `variant="select"` keeps a long list compact.

```html preview
<sb-theme-switch variant="select" cookie="sb-theme-demo2" attribute="data-demo-theme"
  themes='["auto", "deep-space", "nebula", "terminal", "daylight"]'></sb-theme-switch>
```

### Menu

An icon button that opens a menu, for headers. The icon shows the current choice (a palette for themes other than auto, dark and light). Starbase's own header uses this.

```html preview
<sb-theme-switch variant="menu" cookie="sb-theme-demo2" attribute="data-demo-theme"
  themes='["auto", "deep-space", "nebula", "terminal", "daylight"]'></sb-theme-switch>
```

## No flash of the wrong theme

The theme has to be on `<html>` before the browser paints, so JavaScript that runs after the page loads is too late. There are two ways, both reading the cookie:

**With a server** (best, and no script at all): read the `sb-theme` cookie and render it into the page:

```html
<html data-theme="dark">  <!-- the cookie's value; leave the attribute out for "auto" -->
```

**Static sites:** put this first in `<head>`, before the stylesheets. With a strict CSP, give it your nonce.

```html
<script>
  const t = (document.cookie.match(/(?:^|; )sb-theme=([^;]*)/) || [])[1]
  if (t && t !== 'auto') document.documentElement.dataset.theme = decodeURIComponent(t)
</script>
```

Then style the themes, with "auto" as the absence of the attribute:

```css
:root { color-scheme: light; /* light tokens */ }
@media (prefers-color-scheme: dark) { :root:not([data-theme]) { color-scheme: dark; /* dark tokens */ } }
[data-theme="dark"] { color-scheme: dark; /* dark tokens */ }
```

Also set `<meta name="color-scheme" content="light dark">`, so the browser's own background matches before your CSS arrives.

### Across subdomains

`domain` writes the cookie for a whole domain, so one choice covers the marketing site, the app and anything else under it. Leave it out and the cookie stays on the host that set it.

```html
<sb-theme-switch domain=".example.com"></sb-theme-switch>
```

Only a domain the page itself belongs to is accepted, and every subdomain can then read and overwrite the value — fine for a preference, so keep anything else out of this cookie. A domain the browser refuses would otherwise drop the choice without a word, so the component reads the cookie back and falls back to this host, reporting the mismatch through `reportError`.

## Following the theme from a canvas

CSS follows a theme on its own. Code that draws (a canvas, WebGL, a chart library) resolved its colours once and has to be told to draw again, for two different reasons:

- **Someone picks a theme.** The switch fires `sb-theme-change` on the window, with `detail.scheme` set to `"light"` or `"dark"`: what the page now paints in, worked out from the theme's own `color-scheme`, so you never need a list of theme names.
- **The system flips while "auto" is chosen.** Nothing is clicked, so no event fires. Listen to `prefers-color-scheme` as well.

Read your colours again when you repaint, and both cases are covered:

```js
const repaint = () => requestAnimationFrame(paint)
addEventListener('sb-theme-change', repaint)
matchMedia('(prefers-color-scheme: dark)').addEventListener('change', repaint)
```

Inside a Rocket component, the first one has an attribute form in the template, `data-on:sb-theme-change__window="@repaint()"`; `sb-sparkline` and `sb-gauge` do exactly this.

## Why a cookie

A cookie reaches the server with the request, so the server can render the theme straight away. Local storage would need a script on every page. The cookie holds only the theme name, for a year, on the whole site (`Path=/`, `SameSite=Lax`, `Secure` on https, and `Domain` when `domain` is set). Several switches on one page stay in sync.

## Styling

Style it from your page's CSS — no need to change the component or import anything into it. Custom properties, inherited properties and `::part()` all reach into its shadow root.

- **Fonts:** the options use your page's font.
- **Colours:** the control is `--sb-control-bg` with a `--sb-control-border` edge. Options are `--sb-text-2` (`--sb-text-1` when hovered or chosen); the chosen one fills with `--sb-brand-subtle` inside a `--sb-brand` edge. The focus ring is `--sb-brand-light`, corners `--sb-control-radius`. The `menu` variant's list uses the same tokens.
- **Parts:** `group` (the segmented control), `select` (with `variant="select"`), and `button` and `menu` (with `variant="menu"`). Your page's `::part()` rules win over the component's own, without `!important`.

```html preview
<style>
  .my-switch { --sb-brand: var(--sb-accent); --sb-control-radius: 999px; }
  .my-switch::part(group) { padding: 4px; }
</style>
<sb-theme-switch class="my-switch" cookie="sb-theme-demo" attribute="data-demo-theme"></sb-theme-switch>
```

## Accessibility

The segmented variant and the menu are radio groups (arrow keys move between themes); the select is a native `<select>`. All carry `label` ("Theme") as their accessible name; the menu button also says the current theme. The menu is a native popover: Escape and clicking outside close it.
