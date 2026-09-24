---
name: Range
tag: sb-range
category: forms
summary: A two-thumb pixel slider for a start and an end, committed as one value.
author: zweiundeins
tags: [range, slider, interval, input, form]
since: 2026-09-22
preview: |
  <sb-range label="Orbit" value='{"start":300,"end":800}' min="160" max="2000" step="10" unit=" km" style="inline-size: 13rem"></sb-range>
usage: |
  <sb-range label="Price" value='{"start":20,"end":80}' min="0" max="100" unit=" €"></sb-range>
playground:
  props:
    min: {min: -100, max: 0}
    max: {min: 1, max: 200}
    step: {min: 0, max: 10}
  values: {label: Window}
  attrs: {value: '{"start":20,"end":60}'}
---

A slider with two thumbs, for a start and an end: a price band, a time window, an altitude range. It is the sibling of [`sb-slider`](/components/slider), and is built on two native range inputs, so arrow keys, Page Up/Down, Home/End and touch work on each thumb. The thumbs can meet but not cross. Where they meet, the direction you drag picks the thumb: left moves the start, right the end.

The range is **one value**: `value='{"start":20,"end":60}'` in, `{ start, end }` out. Both ends change together in one `sb-change` and one command. See [Components with several values](/contribute#components-with-several-values).

## Examples

### Basic

```html preview
<div style="display: grid; gap: 20px; inline-size: min(100%, 22rem)">
  <sb-range label="Orbit" value='{"start":300,"end":800}' min="160" max="2000" step="10" unit=" km" ticks></sb-range>
  <sb-range label="Launch window" value='{"start":9,"end":17}' min="0" max="24" unit=":00"></sb-range>
  <sb-range label="Signal" value='{"start":0.2,"end":0.8}' min="0" max="1" step="0.01"></sb-range>
</div>
```

A missing (or `null`) part is the bound: `value='{"end":40}'` starts at `min`, and without a `value` the whole range is selected. Values snap to `step` as the thumbs do, so the text, the fill and `sb-change` always match the thumbs; when `max` is off the step grid, the end stops at the last step below it. With `step="0"` (continuous) the text shows two decimals.

### The whole range in one event

`sb-change` fires once per committed drag or key press, with both ends. A key press that can't move a thumb (the other one is in the way) sends nothing:

```html preview
<div data-signals="{_window: ''}" style="display: grid; gap: 12px; inline-size: min(100%, 22rem)">
  <sb-range name="window" label="Launch window" value='{"start":9,"end":17}' min="0" max="24" unit=":00"
    data-on:sb-change="$_window = JSON.stringify(evt.detail)"></sb-range>
  <code data-text="$_window || 'Drag a thumb…'"></code>
</div>
```

The live value is the `value` property: `el.value` returns `{ start, end }`, and setting it accepts `{ start, end }` or `[start, end]`.

## With commands

Post the detail as it is. The server validates the range as a whole and renders the new `value` attribute:

```html
<sb-range name="window" confirm value='{"start":9,"end":17}' min="0" max="24"
  data-on:sb-change="@post('/cmd/window', {payload: {tabid: $tabid, ...evt.detail}})"
  data-on:datastar-fetch="evt.detail.el === el && evt.detail.type === 'error' && el.revert()"></sb-range>
```

With `confirm`, the host has `:state(pending)` while either end differs from the server's attribute, and `revert()` restores both ends. A command can never leave the server with only half a change. See [Commands and components](/contribute#commands-and-components).

```css
sb-range:state(pending) { opacity: 0.7; }
```

## Forms

`sb-range` is not a form-associated element: a `<form>` doesn't submit it, `FormData` and Datastar's `contentType: 'form'` don't see it, and a form reset doesn't reset it. Send its value as a command instead: `sb-change` carries `{ name, value }` (see [With commands](#with-commands)).

## Styling

Style it from your page's CSS — no need to change the component or import anything into it. Custom properties, inherited properties and `::part()` all reach into its shadow root.

- **Size:** it fills the width it is given (at least `8rem`); set `max-inline-size` on the element to cap it.
- **Fonts:** the label, the range and the tick labels use your page's font.
- **Colours:** the part of the track between the thumbs is `--sb-brand`, the rest `--sb-surface-inset` with a `--sb-border` edge. The thumbs are `--sb-slider-thumb` (default `--sb-text-1`) with a `--sb-slider-thumb-edge` edge (default `--sb-brand-light`), which is also the focus ring. They are the same properties as [`sb-slider`](/components/slider)'s, so one rule styles both. The label is `--sb-text-2`, the range `--sb-text-1`, the ticks `--sb-text-muted`. `--sb-notch: 0` rounds the track and the thumbs instead of notching them.
- **Parts:** `label`, `value` (the shown range), `rail` (the track) and `input` for both range inputs, plus `start` or `end` for one of them (`::part(input end)`). The thumbs are drawn inside the inputs and have no part: colour them with `--sb-slider-thumb` and `--sb-slider-thumb-edge`. Your page's `::part()` rules win over the component's own, without `!important`.

```html preview
<style>
  .my-range { max-inline-size: 20rem; --sb-brand: #F97316; --sb-brand-light: #FDBA74; --sb-notch: 0; --sb-slider-thumb: #FFFFFF; --sb-slider-thumb-edge: #F97316; }
  .my-range::part(value) { color: #F97316; }
</style>
<sb-range class="my-range" label="Burn window" value='{"start":20,"end":45}' max="60" unit=" s"></sb-range>
```

## Accessibility

The field is a group named by its `label`. Each thumb is a native slider, announced as "<label> start" and "<label> end" with its value and unit (`aria-valuetext`). Without a `label` they are "Range start" and "Range end". The range shown next to the label is plain text, not a live region, so a step is announced once, by the thumb that moved. The focused thumb gets a ring inside its edge.
