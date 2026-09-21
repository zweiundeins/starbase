---
name: Meter
tag: sb-meter
category: feedback
summary: A segmented pixel bar for levels, progress and resources.
author: zweiundeins
tags: [progress, level, bar, battery, fuel]
since: 2026-09-21
preview: |
  <sb-meter label="Shields" value="75" unit="%" style="--sb-meter-width: 13rem"></sb-meter>
playground:
  props:
    value: {min: 0, max: 100}
    segments: {min: 2, max: 40}
    warn: {min: 0, max: 100}
    danger: {min: 0, max: 100}
    decimals: {min: 0, max: 3}
  values: {value: 58, label: Shields, unit: "%"}
---

A row of pixel blocks lit up to `value`. The whole bar changes colour at the `warn` and `danger` thresholds, and blocks light up in a quick cascade. When `danger` is below `warn`, low values are the bad ones, as with fuel or batteries.

## Examples

### Driven by a signal

```html preview
<div data-signals:_shields="58" style="display: grid; gap: 16px; inline-size: min(100%, 22rem)">
  <sb-meter label="Shields" unit="%" data-attr:value="$_shields" data-preserve-attr="value"></sb-meter>
  <sb-slider label="Power to shields" unit="%" data-bind:_shields__prop.value></sb-slider>
</div>
```

### Thresholds

```html preview
<div style="display: grid; gap: 16px; inline-size: min(100%, 22rem)">
  <sb-meter label="Cargo" value="40" unit="t" max="60" warn="45" danger="55"></sb-meter>
  <sb-meter label="Fuel" value="12" unit="%" warn="30" danger="15" segments="20"></sb-meter>
  <sb-meter label="Heat" value="92" unit="%" segments="8"></sb-meter>
</div>
```

## Accessibility

The bar is a `role="meter"` with min, max, value and a readable value text. The colour change is backed by the number, so tone never carries the meaning alone.
