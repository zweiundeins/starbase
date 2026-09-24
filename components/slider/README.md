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
usage: |
  <sb-slider label="Volume" value="60" unit="%"></sb-slider>
playground:
  props:
    value: {min: 0, max: 100}
    min: {min: -100, max: 0}
    max: {min: 1, max: 200}
    step: {min: 0, max: 10}
---

A range control with a chunky pixel thumb. It wraps a native `<input type="range">`, so keyboard, touch and screen readers work out of the box. It exposes a numeric `value` property for `data-bind`.

Whichever way a value arrives (the attribute, the property, `data-bind`), it is put in range and on a step, like the native input's thumb. With `step="0"` the slider is continuous and shows two decimals.

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

Give it a `name`, and it emits `sb-change` with `{ name, value }` when a value is committed: ready to post as a command. With `confirm`, it sets `:state(pending)` until the server's re-rendered attribute matches, and `revert()` goes back to the server's value when a command is rejected. Every step taken with the keyboard is a commit, and the server echoes each one: with `confirm`, the echo of an older commit doesn't pull the thumb back while newer ones are on their way, so holding an arrow key loses no steps. Any other new `value` from the server wins, even `value="0"` on a slider first rendered without one. See [Commands and components](/contribute#commands-and-components) and the [Showcase](/showcase).

## Forms

Inside a `<form>`, `sb-slider` submits its value under its `name` (`name=72`: the number, without the unit), `new FormData(form)` and Datastar's `contentType: 'form'` include it, and a form reset brings back the server's value. A `disabled` slider submits nothing. It is not a form-associated element yet (Rocket can't declare one), so `required` and validity, `<fieldset disabled>`, `<label for>` and the `form` attribute don't reach it. With commands, `sb-change` carries `{ name, value }` (see [With commands](#with-commands)).

## Styling

Style it from your page's CSS — no need to change the component or import anything into it. Custom properties, inherited properties and `::part()` all reach into its shadow root.

- **Size:** it fills the width it is given (at least `8rem`); set `max-inline-size` on the element to cap it.
- **Direction:** in right-to-left text the minimum is on the right, and the track fills from there.
- **Fonts:** the label, the value and the tick labels use your page's font.
- **Colours:** the filled part of the track is `--sb-brand`, the rest `--sb-surface-inset` with a `--sb-border` edge. The thumb is `--sb-slider-thumb` (default `--sb-text-1`) with a `--sb-slider-thumb-edge` edge (default `--sb-brand-light`), which is also the focus ring. The label is `--sb-text-2`, the value `--sb-text-1`, the ticks `--sb-text-muted`. `--sb-notch: 0` rounds the track and the thumb instead of notching them.
- **Parts:** `label`, `value` and `input` (the range input). The thumb is drawn inside the input and has no part: colour it with `--sb-slider-thumb` and `--sb-slider-thumb-edge`. Your page's `::part()` rules win over the component's own, without `!important`.

```html preview
<style>
  .my-slider { max-inline-size: 20rem; --sb-brand: #F97316; --sb-brand-light: #FDBA74; --sb-notch: 0; --sb-slider-thumb: #FFFFFF; --sb-slider-thumb-edge: #F97316; }
  .my-slider::part(value) { color: #F97316; }
</style>
<sb-slider class="my-slider" label="Thrust" value="70" unit="%"></sb-slider>
```

## Accessibility

The native range input provides the role, value and keyboard support (arrow keys, Page Up/Down, Home/End). Its name is the `label` (a click on the label focuses the slider), else the element's `aria-label`, else "Value": give it one of the first two. The value is announced with its unit (`aria-valuetext`, e.g. "30°"). With forced colours (e.g. Windows High Contrast) it shows the native control, drawn in the system's colours.
