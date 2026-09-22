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
playground:
  exclude: [themes, labels, cookie, attribute]
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

## Why a cookie

A cookie reaches the server with the request, so the server can render the theme straight away. Local storage would need a script on every page. The cookie holds only the theme name, for a year, on the whole site (`Path=/`, `SameSite=Lax`, and `Secure` on https). Several switches on one page stay in sync.

## Accessibility

The segmented variant and the menu are radio groups (arrow keys move between themes); the select is a native `<select>`. All carry `label` ("Theme") as their accessible name; the menu button also says the current theme. The menu is a native popover: Escape and clicking outside close it.
