---
name: Meter
tag: sb-meter
category: feedback
summary: A segmented pixel bar for levels and resources.
author: zweiundeins
tags: [level, bar, battery, fuel]
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
- **Parts:** `label`, `value` and `bar`, and `segment` for every block; the lit ones are also `lit`: `::part(segment lit)`. Your page's `::part()` rules win over the component's own, without `!important`.
- **States:** the host has `:state(ok)`, `:state(warn)` or `:state(danger)`, so the tone can restyle any part: `sb-meter:state(danger)::part(label)`.

```html preview
<style>
  .my-meter { --sb-meter-width: 16rem; --sb-meter-height: 20px; --sb-ok: var(--sb-accent); }
  .my-meter::part(value) { font-size: 1.25rem; }
</style>
<sb-meter class="my-meter" label="Shields" value="45" unit="%"></sb-meter>
```

## Accessibility

The element itself is the meter: `role="meter"` with `min`, `max`, the value (held within the range) and the shown text, such as `75%`, as its value text. They are set through `ElementInternals`, so page morphs can't strip them. Its name is `label`; an `aria-label` on the element wins over it, and without either the name is "Meter". The caption and number above the bar are hidden from assistive technology, so nothing is read twice.

The fill and the number show the level; the tone is colour only, and assistive technology gets the value, not whether it is past `warn` or `danger`. Where that matters, say it in the label or next to the meter. In forced-colours mode (Windows High Contrast), the lit blocks are drawn in the text colour inside an outlined frame.

`sb-meter` shows a level within a known range, like the native `<meter>`. For the progress of a task, use a progress bar (`role="progressbar"`).
