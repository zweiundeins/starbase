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
usage: |
  <sb-meter label="Shields" value="75" unit="%"></sb-meter>
playground:
  props:
    value: {min: 0, max: 100}
    segments: {min: 2, max: 40}
    warn: {min: 0, max: 100}
    danger: {min: 0, max: 100}
    decimals: {min: 0, max: 3}
  values: {value: 58, label: Shields, unit: "%"}
---

A row of pixel blocks lit up to `value`, from `min` to `max`. Any value above `min` lights at least one block, and only `max` lights them all. When the value changes, blocks light up in a quick cascade from the edge that moves.

The whole bar changes colour at the `warn` and `danger` thresholds, given in value units. Left unset, they sit at 70% and 90% of the range (70 and 90 on 0–100). When `danger` is below `warn`, low values are the bad ones, as with fuel or batteries. To keep the bar in its normal colour, set both above `max`.

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

### Live from the server

```html preview
<div data-signals="{_tm: {fuel: 100}}" data-init="@get('/demo/telemetry')" style="inline-size: min(100%, 22rem)">
  <sb-meter label="Fuel" unit="%" warn="30" danger="15" segments="20" data-attr:value="Math.round($_tm.fuel)" data-preserve-attr="value"></sb-meter>
</div>
```

## Styling

Style it from your page's CSS — no need to change the component or import anything into it. Custom properties, inherited properties and `::part()` all reach into its shadow root.

- **Size:** `--sb-meter-width` (default `22rem`) is the widest it gets, `--sb-meter-height` (default `12px`) the height of the blocks; `segments` sets how many there are.
- **Fonts:** the label and the value use your page's font.
- **Colours:** lit blocks are `--sb-ok`, `--sb-warn` past the `warn` threshold and `--sb-danger` past `danger`. The track is `--sb-surface-inset` with a `--sb-border` edge, and unlit blocks are a tint of that border. The label is `--sb-text-2`, the value `--sb-text-1`.
- **Parts:** `label`, `value` and `bar`. Your page's `::part()` rules win over the component's own, without `!important`.

```html preview
<style>
  .my-meter { --sb-meter-width: 16rem; --sb-meter-height: 20px; --sb-ok: var(--sb-accent); }
  .my-meter::part(value) { font-size: 1.25rem; }
</style>
<sb-meter class="my-meter" label="Shields" value="45" unit="%"></sb-meter>
```

## Accessibility

The bar is a `role="meter"` with min, max, value and a readable value text. The colour change is backed by the number, so tone never carries the meaning alone.
