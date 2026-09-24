---
name: Gauge
tag: sb-gauge
category: data
summary: A pixel dial whose needle springs to its value, with warning zones.
author: zweiundeins
tags: [meter, dial, dashboard, telemetry]
since: 2026-09-21
preview: |
  <sb-gauge value="72" label="Thrust" unit="%" style="--sb-gauge-size: 9.5rem"></sb-gauge>
usage: |
  <sb-gauge value="72" label="Thrust" unit="%"></sb-gauge>
playground:
  props:
    value: {min: 0, max: 100}
    warn: {min: 0, max: 100}
    danger: {min: 0, max: 100}
    decimals: {min: 0, max: 3}
  values: {value: 64, label: Thrust, unit: "%"}
---

A half-dial drawn in pixels. The needle eases toward `value` on a slightly bouncy spring, and the arc behind it is coloured by zone: ok, warn or danger. When `danger` is below `warn`, low values are the bad ones, as with fuel. The dial only animates while it is on screen: one that scrolls back in eases to the latest value.

## Examples

### Driven by a slider

```html preview
<div data-signals:_thrust="64" style="display: flex; gap: 32px; align-items: center; flex-wrap: wrap">
  <sb-gauge label="Thrust" unit="%" data-attr:value="$_thrust" data-preserve-attr="value"></sb-gauge>
  <sb-slider label="Throttle" unit="%" style="inline-size: 14rem" data-bind:_thrust__prop.value></sb-slider>
</div>
```

### Low is bad

```html preview
<sb-gauge label="Fuel" unit="%" value="18" warn="35" danger="15"></sb-gauge>
<sb-gauge label="Hull temp" unit="°" value="940" min="0" max="1200" warn="800" danger="1000"></sb-gauge>
```

### Live from the server

This site's demo endpoint streams telemetry as signal patches. The gauge follows `$_tm.vel` without any client code.

```html preview
<div data-signals="{_tm: {vel: 0, temp: 18}}" data-init="@get('/demo/telemetry')" style="display: flex; gap: 32px; flex-wrap: wrap">
  <sb-gauge label="Velocity" unit=" km/s" max="8" warn="9" danger="10" decimals="2" data-attr:value="$_tm.vel" data-preserve-attr="value"></sb-gauge>
  <sb-gauge label="Hull" unit="°C" max="1200" warn="800" danger="1000" data-attr:value="$_tm.temp" data-preserve-attr="value"></sb-gauge>
</div>
```

## Styling

Style it from your page's CSS — no need to change the component or import anything into it. Custom properties, inherited properties and `::part()` all reach into its shadow root.

- **Size:** `--sb-gauge-size` (default `12rem`) sets the width. The dial keeps its proportions and the value and label scale with it (never below a readable size).
- **Fonts:** the value and the label use your page's font. A site that sets a display font (`--sb-font-display`, like Starbase's pixel font) gets it for the value; set `--sb-font-display` on the gauge to choose the value's font on its own.
- **Colours:** the dial paints with theme tokens and repaints whenever one of them changes, whatever changed it: a theme switch, a theme scoped to a container, the system's light or dark mode, or your own override on the gauge. `--sb-ok`, `--sb-warn` and `--sb-danger` colour the zones, `--sb-border` the track, `--sb-text-1` the needle and the value, `--sb-brand` the hub, in any colour syntax, `light-dark()` included. The label is `--sb-text-2`.
- **Parts:** `value`, `label` and `dial` (the canvas), for anything the tokens don't cover. Your page's `::part()` rules win over the component's own, without `!important`.

```html preview
<style>
  .my-gauge {
    --sb-gauge-size: 14rem;
    font-family: Georgia, serif;
    --sb-font-display: Georgia, serif; /* only needed where a display font is set, like here */
  }
  .my-gauge::part(label) { text-transform: none; letter-spacing: 0; }
</style>
<sb-gauge class="my-gauge" value="64" label="Oxygen" unit="%"></sb-gauge>
```

A font you load yourself works inside the component too: load it in the page (a `<link>`, or an `@import` at the very top of your main stylesheet). A component's own styles are a constructed stylesheet, which can't `@import`.

## Accessibility

The readout is a `role="meter"` with min, max and current value. Like a native `<meter>`, the current value it reports stays within min and max; its text is the reading as shown, with the unit. The dial canvas is decorative. Under `prefers-reduced-motion` the needle jumps straight to the value.
