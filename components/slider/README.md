---
name: Slider
tag: sb-slider
category: forms
summary: A pixel range control that steers anything through signals.
author: zweiundeins
tags: [range, input, number, control]
since: 2026-09-21
preview: |
  <sb-slider label="Thrust" value="72" unit="%" style="inline-size: 13rem"></sb-slider>
playground:
  props:
    value: {min: 0, max: 100}
    min: {min: -100, max: 0}
    max: {min: 1, max: 200}
    step: {min: 0, max: 10}
---

A range control with a chunky pixel thumb. It wraps a native `<input type="range">`, so keyboard, touch and screen readers work out of the box. It exposes a numeric `value` property for `data-bind`.

## Examples

### Basic

```html preview
<div style="display: grid; gap: 20px; inline-size: min(100%, 22rem)">
  <sb-slider label="Thrust" value="72" unit="%"></sb-slider>
  <sb-slider label="Heading" min="-180" max="180" value="30" unit="°" ticks></sb-slider>
  <sb-slider label="Precision" min="0" max="1" step="0.01" value="0.5"></sb-slider>
</div>
```

### Steer anything with signals

Declare a signal, bind the slider to it, and drive other elements' attributes with `data-attr`. Nothing goes to the server: signals that start with `_` stay in the browser.

```html preview
<div data-signals:_size="48" style="display: grid; gap: 16px; inline-size: min(100%, 22rem)">
  <sb-slider label="Size" min="16" max="96" unit="px" data-bind:_size__prop.value></sb-slider>
  <img src="/art/info.svg" alt="" data-attr:width="$_size" data-attr:height="$_size" style="image-rendering: pixelated">
</div>
```

When the driven element is a component inside a server-rendered region, add `data-preserve-attr` for those attributes. Then a server morph won't reset them:

```html
<sb-gauge data-attr:value="$_fuel" data-preserve-attr="value"></sb-gauge>
```

## With commands

Give it a `name`, and it emits `sb-change` with `{ name, value }` when a value is committed: ready to post as a command. With `confirm`, it sets `:state(pending)` until the server's re-rendered attribute matches, and `revert()` goes back to the server's value when a command is rejected. See [Commands and components](/contribute#commands-and-components) and the [Showcase](/showcase).

## Accessibility

The native range input provides the role, value and keyboard support (arrow keys, Page Up/Down, Home/End). Give it a `label`; without one it is announced as "Value".
