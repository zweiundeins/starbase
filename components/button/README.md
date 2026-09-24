---
name: Button
tag: sb-button
category: forms
summary: A versatile button component with Datastar interactions.
author: zweiundeins
tags: [button, action, link, cta]
since: 2026-09-21
preview: |
  <sb-button variant="pixel" size="lg" caret>Blast off</sb-button>
playground:
  content: Blast off
  values: {variant: pixel, caret: true}
---

Buttons start things. `sb-button` comes in five variants, three sizes, and renders a real `<a>` when given an `href`. Clicks bubble out of the shadow root, so any Datastar `data-on:click` just works.

## Examples

### Variants

```html preview
<sb-button>Primary</sb-button>
<sb-button variant="outline">Outline</sb-button>
<sb-button variant="ghost">Ghost</sb-button>
<sb-button variant="danger">Abort</sb-button>
<sb-button variant="pixel" caret>Blast off</sb-button>
```

### Sizes

```html preview
<sb-button size="sm">Small</sb-button>
<sb-button size="md">Medium</sb-button>
<sb-button size="lg">Large</sb-button>
```

### With Datastar

Local signals start with an underscore, so they stay in the browser and are never sent to the server.

```html preview
<div data-signals:_count="0">
  <sb-button data-on:click="$_count++">Launches: <span data-text="$_count"></span></sb-button>
  <sb-button variant="ghost" data-on:click="$_count = 0" data-attr:disabled="$_count === 0">Reset</sb-button>
</div>
```

Talking to the backend works the same way: `data-on:click="@post('/launch')"`.

### As a link

```html preview
<sb-button href="https://data-star.dev" variant="outline" caret>Read the Datastar docs</sb-button>
```

### While the action runs

`loading` shows an inline spinner in the button, blocks clicks and Enter, and sets `aria-busy`. The button keeps its focus, so nobody is thrown out of the page mid-action, and it takes no extra space until it is loading.

```html preview
<div data-signals="{_sync: []}" style="display: flex; align-items: center; gap: 16px">
  <sb-button
    data-indicator:_syncing
    data-attr:loading="$_syncing"
    data-preserve-attr="loading"
    data-on:click="@get('/demo/data/children?into=_sync&delay=900')"
  >Sync catalog</sb-button>
  <span class="muted" data-text="$_sync.length ? 'synced ' + $_sync.length + ' galaxies' : 'not synced yet'"></span>
</div>
```

`data-indicator` sets the signal while that element's request is in flight, and `data-preserve-attr` keeps the attribute through a server morph. For a wait that is not a button — a panel, a table, a whole region — use [`sb-busy`](/components/busy), which also knows `delay`, `min` and progress shapes.

## Styling

Style it from your page's CSS — no need to change the component or import anything into it. Custom properties, inherited properties and `::part()` all reach into its shadow root.

- **Size:** `size` (`sm`, `md`, `lg`) sets the height, the padding and the text size. For anything in between, set `block-size`, `padding-inline` or `font-size` on `::part(button)`: icons and the loading spinner are sized in `em`, so they follow the text.
- **Fonts:** the label uses your page's font. The `pixel` variant uses `--sb-font-display` when a site sets one (like Starbase's pixel font); set it on the button to choose that font on its own.
- **Colours:** `primary` fills with `--sb-brand` (`--sb-brand-hover` on hover) and writes in `--sb-text-on-brand`; `outline` draws a `--sb-brand-light` border and hovers with `--sb-brand-subtle`; `ghost` hovers with `--sb-surface-hover`; `danger` fills with `--sb-danger` and writes in `--sb-text-on-danger`. The other variants write in `--sb-text-1`. `pixel` is a `--sb-text-1` plate with `--sb-bg` text in a `--sb-frame-color` frame, `--sb-frame-step` thick. Corners are `--sb-control-radius`, the focus ring `--sb-focus-ring`.
- **Parts:** `button` (the `<button>`, or the `<a>` with `href`) and `spinner` (while `loading`). Your page's `::part()` rules win over the component's own, without `!important`.

```html preview
<style>
  .my-button { --sb-brand: #0F766E; --sb-brand-hover: #115E59; --sb-text-on-brand: #FFFFFF; --sb-control-radius: 999px; }
  .my-button::part(button) { font-size: 1.125rem; padding-inline: 2rem; }
</style>
<sb-button class="my-button">Launch</sb-button>
```

A font you load yourself works inside the component too: load it in the page (a `<link>`, or an `@import` at the very top of your main stylesheet). A component's own styles are a constructed stylesheet, which can't `@import`.

## Accessibility

A native `<button>` (or `<a>`) inside the shadow root does the work: keyboard focus, Enter and Space, and screen reader semantics. Disabled buttons are removed from the tab order.
