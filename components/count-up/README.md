---
name: Count Up
tag: sb-count-up
category: data
summary: Counts a number up to its value when it scrolls into view.
author: zweiundeins
tags: [number, counter, animation, statistics, intl]
since: 2026-09-23
preview: |
  <sb-count-up value="4681" style="font-size: 2rem">4681</sb-count-up>
playground:
  props:
    value: {min: 0, max: 10000}
    from: {min: 0, max: 10000}
    duration: {min: 0, max: 5000, step: 100}
    decimals: {min: 0, max: 3}
  values: {value: 4681}
  style: "font-size: 2rem"
---

Counts a number to its value the first time it is on screen: for the statistics on a landing page, a total that just changed, anything that should arrive rather than appear. It takes its font, size and colour from where you put it.

Put the server-formatted number inside it. That is what search engines and people without JavaScript see, and screen readers always hear the final value rather than the frames of the count.

## Examples

### On scroll

```html preview
<p style="font-size: 1.25rem">
  <sb-count-up value="1113" style="font-size: 2.5rem; font-weight: 700">1113</sb-count-up>
  models, <sb-count-up value="56">56</sb-count-up> makes.
</p>
```

### Decimals and units

`decimals` is kept for every frame, so the width does not change while it counts. Units stay outside the element.

```html preview
<p style="font-size: 2rem">
  CHF <sb-count-up value="0.52" decimals="2" duration="2000">0.52</sb-count-up> / km
</p>
```

### A new value

When the server sends a new `value` while the number is on screen, it counts from where it stands.

```html preview
<div data-signals="{_total: 1200}" style="display: flex; gap: 1rem; align-items: center">
  <sb-count-up data-attr:value="$_total" data-preserve-attr="value" style="font-size: 2rem">1200</sb-count-up>
  <sb-button data-on:click="$_total += Math.round(Math.random() * 500)">Add an order</sb-button>
</div>
```

## Accessibility

Screen readers get the final value as text; the counting digits are hidden from them. With `prefers-reduced-motion: reduce` the number is shown at its value straight away, and a new value replaces it without counting.
