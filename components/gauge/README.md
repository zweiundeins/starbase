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
playground:
  props:
    value: {min: 0, max: 100}
    warn: {min: 0, max: 100}
    danger: {min: 0, max: 100}
    decimals: {min: 0, max: 3}
  values: {value: 64, label: Thrust, unit: "%"}
---

A half-dial drawn in pixels. The needle eases toward `value` on a slightly bouncy spring, and the arc behind it is coloured by zone: ok, warn or danger. When `danger` is below `warn`, low values are the bad ones, as with fuel.

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

## Accessibility

The readout is a `role="meter"` with min, max and current value. The dial canvas is decorative. Under `prefers-reduced-motion` the needle jumps straight to the value.
