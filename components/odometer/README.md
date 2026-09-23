---
name: Odometer
tag: sb-odometer
category: data
summary: A number that rolls its digit wheels to each new value, like a car's odometer.
author: zweiundeins
tags: [number, counter, animation, odometer, intl]
since: 2026-09-23
preview: |
  <sb-odometer value="221180" style="font-size: 2rem">221180</sb-odometer>
playground:
  props:
    value: {min: 0, max: 999999}
    decimals: {min: 0, max: 3}
  values: {value: 221180}
  style: "font-size: 2rem"
---

Shows a number whose digits roll to each new value, one wheel per digit. Only the digits that change turn, so a readout that ticks up many times a second (a live distance, a counter, a price) stays calm to watch. It takes its font, size and colour from where you put it.

Put the server-formatted number inside it: that is what shows before the component loads, for search engines, and without JavaScript.

## Examples

### Rolling

```html preview
<div data-signals="{_km: 221180}" style="display: flex; gap: 1rem; align-items: center">
  <sb-odometer data-attr:value="$_km" data-preserve-attr="value" style="font-size: 2rem">221180</sb-odometer>
  <sb-button data-on:click="$_km += 1">+1</sb-button>
  <sb-button data-on:click="$_km += 37">+37</sb-button>
  <sb-button data-on:click="$_km += 999">+999</sb-button>
</div>
```

### Decimals

`decimals` fixes the digits after the decimal mark, so a whole-number step still shows its `.0` and the layout does not jump.

```html preview
<div data-signals="{_trip: 9.4}" style="display: flex; gap: 1rem; align-items: center">
  <sb-odometer decimals="1" data-attr:value="$_trip" data-preserve-attr="value" style="font-size: 2rem">9.4</sb-odometer>
  <sb-button data-on:click="$_trip = Math.round(($_trip + 0.3) * 10) / 10">+0.3 km</sb-button>
</div>
```

When the number gains a digit (9.9 → 10.0) the wheels snap to the new layout instead of rolling from the old one.

### Locales

Separators follow `lang`, or the page's language when it is not set. The digits themselves are always 0–9.

```html preview
<div style="display: grid; gap: 0.5rem; font-size: 1.5rem">
  <sb-odometer value="1234567.5" decimals="1" lang="de-CH">1234567.5</sb-odometer>
  <sb-odometer value="1234567.5" decimals="1" lang="fr-CH">1234567.5</sb-odometer>
  <sb-odometer value="1234567.5" decimals="1" lang="en">1234567.5</sb-odometer>
</div>
```

## Styling

The digits inherit the surrounding font; a monospaced or tabular face keeps the wheels evenly spaced. `--sb-odometer-duration` and `--sb-odometer-easing` tune the roll, and `::part(digit)` / `::part(separator)` reach the wheels and the static characters.

## Accessibility

Screen readers get the formatted value once, as text. The wheels are hidden from them, since each one holds all ten digits. With `prefers-reduced-motion: reduce` the digits change without rolling.
