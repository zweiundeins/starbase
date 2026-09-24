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

A slider with two thumbs, for a start and an end: a price band, a time window, an altitude range. It is the sibling of [`sb-slider`](/components/slider), and is built on two native range inputs, so arrow keys, Page Up/Down, Home/End and touch work on each thumb. The thumbs can meet but not cross.

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

A missing part is the bound: `value='{"end":40}'` starts at `min`, and without a `value` the whole range is selected.

### The whole range in one event

`sb-change` fires once per committed drag, with both ends:

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

## Styling

The thumbs and the fill follow `--sb-text-1`, `--sb-brand-light` and `--sb-brand`; `--sb-notch: 0` gives rounded corners. Parts: `label`, `value`, `rail`, and `input` (plus `start` / `end`) for each native input.

## Accessibility

The field is a group named by its `label`. Each thumb is a native slider, announced as "<label> start" and "<label> end" with its value and unit (`aria-valuetext`). Without a `label` they are "Range start" and "Range end".
