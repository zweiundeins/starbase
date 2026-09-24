---
name: Odometer
tag: sb-odometer
category: data
summary: A number that rolls its digit wheels to each new value, like a car's odometer.
author: zweiundeins
tags: [number, counter, animation, odometer, intl]
since: 2026-09-23
preview: |
  <div data-signals="{_odo: 221180.4}" data-on-interval__duration.3600ms.leading="$_odo = Math.round(($_odo + 0.1) * 10) / 10">
    <sb-odometer drum decimals="1" data-attr:value="$_odo" data-preserve-attr="value" style="font-size: 2rem; --sb-odometer-duration: 3.6s; --sb-odometer-easing: linear">221180.4</sb-odometer>
  </div>
usage: |
  <sb-odometer value="221180.4" decimals="1">221180.4</sb-odometer>
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

### Driving

A tenth of a kilometre every 3.6 seconds is 100 km/h. Give the roll the same 3.6 seconds and a linear pace, and the last wheel never stops turning, like the one in a car; the kilometre wheel turns over with it on the last tenth.

```html preview
<div data-signals="{_odo: 221180.4}" data-on-interval__duration.3600ms.leading="$_odo = Math.round(($_odo + 0.1) * 10) / 10">
  <sb-odometer decimals="1" data-attr:value="$_odo" data-preserve-attr="value" style="font-size: 2.5rem; --sb-odometer-duration: 3.6s; --sb-odometer-easing: linear">221180.4</sb-odometer>
  km
</div>
```

### Drum

`drum` draws each wheel as a cylinder: you see the digit face-on and its neighbours curving away above and below, and as a wheel turns, the digit on it foreshortens like print on a real drum.

```html preview
<div data-signals="{_odo: 221180.4}" data-on-interval__duration.1200ms.leading="$_odo = Math.round(($_odo + 0.1) * 10) / 10" style="display: flex; gap: 0.5rem; align-items: center">
  <sb-odometer drum decimals="1" data-attr:value="$_odo" data-preserve-attr="value" style="font-size: 3rem; --sb-odometer-duration: 1.2s; --sb-odometer-easing: linear">221180.4</sb-odometer>
  <span>km</span>
</div>
```

`--sb-odometer-window` sets how much of the drum shows (default `1.5em`, the digit plus a sliver of its neighbours), `--sb-odometer-perspective` how strongly it curves (default `5em`; smaller is rounder).

### Flight

An altimeter moves fast, and both ways. Updating every 150 ms with a 150 ms linear roll, the lower wheels blur, the hundreds tick past steadily, and on the way down every wheel turns back under.

```html preview
<div data-signals="{_alt: 10000, _vs: 1}" data-on-interval__duration.150ms="$_vs = $_alt >= 35000 ? -1 : $_alt <= 10000 ? 1 : $_vs; $_alt += $_vs * 61" style="display: flex; gap: 1rem; align-items: baseline">
  <sb-odometer data-attr:value="$_alt" data-preserve-attr="value" style="font-size: 2.5rem; --sb-odometer-duration: 150ms; --sb-odometer-easing: linear">10000</sb-odometer>
  <span>ft</span>
  <span data-text="$_vs > 0 ? '▲ climbing' : '▼ descending'"></span>
</div>
```

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

The digits inherit the surrounding font; a monospaced or tabular face keeps the wheels evenly spaced. `::part(digit)` / `::part(separator)` reach the wheels and the static characters.

For a wheel you can see is round, add `drum` (see above).

The speed is yours: `--sb-odometer-duration` (default `0.55s`) is how long a wheel takes to reach its new digit, and `--sb-odometer-easing` how it gets there. A quick ease-out suits a counter that jumps; for a value that climbs steadily, set the duration to the interval between updates and the easing to `linear`, and the wheels move continuously. A wheel always turns the way the number moves: forward over the top when it climbs (9 → 0), back when it falls.

## Accessibility

Screen readers get the formatted value once, as text. The wheels are hidden from them, since each one holds all ten digits. With `prefers-reduced-motion: reduce` the digits change without rolling.
