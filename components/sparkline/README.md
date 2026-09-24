---
name: Sparkline
tag: sb-sparkline
category: data
summary: A tiny stepped pixel chart that grows with every pushed value.
author: zweiundeins
tags: [chart, trend, telemetry, dashboard]
since: 2026-09-21
preview: |
  <sb-sparkline values="[12,18,15,22,30,26,34,41,38,47,52,49,58,64,61,70]" length="16" tone="ok" show-value unit="%" style="--sb-sparkline-width: 12rem"></sb-sparkline>
usage: |
  <sb-sparkline values="[12,18,15,22,30,26,34,41]" show-value unit="%"></sb-sparkline>
playground:
  props:
    value: {min: 0, max: 100}
    length: {min: 4, max: 80}
    decimals: {min: 0, max: 3}
  values: {showValue: true, tone: accent, value: 58}
  attrs: {values: "[12,18,15,22,30,26,34,41,38,47,52,49]"}
  exclude: [min, max, label]
  style: "--sb-sparkline-width: 18rem"
---

A small inline chart drawn as a stepped pixel line with a soft area underneath. Give it a JSON `values` array, or use push mode: every time the `value` attribute changes, a point is appended (a starting `value` is the latest point, after any `values`). It keeps the last `length` points: fewer spread across the width, and once it's full the line scrolls. Point `value` at a signal, and the line follows it as the signal ticks, from the client or from the server.

A point is added when the value *changes*: the same value again adds nothing (`el.push()` records a repeat). A removed or empty `value` attribute adds nothing either, and neither does the same value put back. In a region the server morphs, add `data-preserve-attr="value"` next to `data-attr:value`, as in the examples: a morph resets attributes to the server's markup, and a `value` written there would be a new point.

## Examples

### Push mode

`data-on-interval` changes a signal every 400 ms. `data-attr` pushes each new value into the chart.

```html preview
<div data-signals:_alt="50" data-on-interval__duration.400ms="$_alt = Math.max(0, Math.min(100, $_alt + (Math.random() * 20 - 9)))">
  <sb-sparkline data-attr:value="$_alt" data-preserve-attr="value" show-value tone="accent" length="48" style="--sb-sparkline-width: 20rem"></sb-sparkline>
</div>
```

### Static data

```html preview
<sb-sparkline values="[3,5,4,8,6,9,12,10,14,13,17]" tone="ok" show-value></sb-sparkline>
<sb-sparkline values="[90,82,85,70,72,60,48,52,40,33,21]" tone="danger" show-value unit="%"></sb-sparkline>
<sb-sparkline values="[40,60,40,60,40,60,40,60]" scale="fixed" min="0" max="100" tone="brand"></sb-sparkline>
```

### Live from the server

The server pushes `$_tm.alt` four times a second. Push mode turns every change into a new point.

```html preview
<div data-signals="{_tm: {alt: 0}}" data-init="@get('/demo/telemetry')">
  <sb-sparkline data-attr:value="$_tm.alt" data-preserve-attr="value" length="80" show-value unit=" km" tone="accent" label="Altitude" style="--sb-sparkline-width: 22rem"></sb-sparkline>
</div>
```

### From JavaScript

`el.push(n)` appends a point, and `el.data` returns a copy of the buffer. The buffer survives moving the element in the page.

## Styling

Style it from your page's CSS — no need to change the component or import anything into it. Custom properties, inherited properties and `::part()` all reach into its shadow root.

- **Size:** `--sb-sparkline-width` (default `12rem`) is the width of the line and the value together, and `--sb-sparkline-height` (default `2.25rem`) the height of the line. Its pixels stay square: the line is 24 of them high and as many wide as fit.
- **Fonts:** the value uses your page's font.
- **Colours:** `tone` picks the line's token: `brand` is `--sb-brand-light`, `ok` `--sb-ok`, `warn` `--sb-warn`, `danger` `--sb-danger`, `accent` `--sb-accent`. The line repaints whenever that colour changes: a theme switch, the system's light or dark mode, or your own class that redefines the token. The value is `--sb-text-1`.
- **Parts:** `line` (the canvas) and `value`. Your page's `::part()` rules win over the component's own, without `!important`.

```html preview
<style>
  .my-spark { --sb-sparkline-width: 16rem; --sb-sparkline-height: 3rem; --sb-brand-light: #22C55E; }
  .my-spark::part(value) { font-size: 1.125rem; }
</style>
<sb-sparkline class="my-spark" values="[12,18,15,22,30,26,34,41]" show-value unit="%"></sb-sparkline>
```

## Accessibility

The canvas is `role="img"`, and its label summarises the data (point count, latest, low and high). Name what it shows with `label`, which comes first ("Altitude: Trend of 80 points: …"): an `aria-label` on the element itself doesn't reach the image. The shown value is hidden from screen readers, since the label has it. Use `show-value` when the latest number matters.
