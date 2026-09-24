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
  exclude: [min, max]
  style: "--sb-sparkline-width: 18rem"
---

A small inline chart drawn as a stepped pixel line with a soft area underneath. Give it a JSON `values` array, or use push mode: the starting `value` is the first point, and every time the `value` attribute changes, a point is appended. The chart is always sized for `length` points, so its height is fixed from the start: fewer points spread across the width, and once it's full the line scrolls. Point `value` at a signal, and the line follows it as the signal ticks, from the client or from the server.

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

The server pushes `$_tm.alt` four times a second. Push mode turns every update into a new point.

```html preview
<div data-signals="{_tm: {alt: 0}}" data-init="@get('/demo/telemetry')">
  <sb-sparkline data-attr:value="$_tm.alt" data-preserve-attr="value" length="80" show-value unit=" km" tone="accent" style="--sb-sparkline-width: 22rem"></sb-sparkline>
</div>
```

### From JavaScript

`el.push(n)` appends a point, and `el.data` returns a copy of the buffer.

## Accessibility

The canvas is `role="img"`, and its label summarises the data (point count, latest, low and high). Use `show-value` when the latest number matters.
