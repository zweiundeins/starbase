---
name: Rating
tag: sb-rating
category: forms
summary: Pixel hearts or stars for a score, with half steps, hover preview and keys.
author: zweiundeins
tags: [rating, stars, hearts, score, form]
since: 2026-09-22
preview: |
  <sb-rating value="3.5" precision="0.5" size="lg"></sb-rating>
playground:
  props:
    value: {min: 0, max: 10, step: 0.5}
    max: {min: 1, max: 10}
  values: {value: 3, max: 5}
---

A row of pixel hearts (or stars) for a score. Point to preview, click to pick (on touch, tap or drag along the row), or use the arrow keys. The live value is the `value` property, so `data-bind` works, and a new `value` attribute from the server replaces it, including `value="0"` to clear it.

## Examples

### Bound to a signal

```html preview
<div data-signals="{_score: 3}" style="display: grid; gap: 12px">
  <sb-rating label="How was the launch?" data-bind:_score__prop.value></sb-rating>
  <span>Score: <b data-text="$_score"></b></span>
</div>
```

### Half steps and stars

```html preview
<div style="display: grid; gap: 12px">
  <sb-rating icon="star" precision="0.5" value="2.5"></sb-rating>
  <sb-rating icon="star" precision="0.5" value="4" max="10" size="sm"></sb-rating>
</div>
```

### Read only

For showing a score, e.g. an average: `readonly` keeps the value ("4.5 of 5") for screen readers, and takes it out of the tab order.

```html preview
<sb-rating readonly value="4.5" precision="0.5" size="sm"></sb-rating>
```

### Clearable

With `clearable`, picking the current value again resets it to 0. Only picking clears: an arrow key or End at the maximum leaves it there.

```html preview
<sb-rating clearable value="2"></sb-rating>
```

## With commands

Give it a `name`, and it emits `sb-change` with `{ name, value }` when a value is picked: ready to post as a command. With `confirm`, it sets `:state(pending)` until the server's re-rendered attribute matches, and `revert()` goes back to the server's value when a command is rejected. See [Commands and components](/contribute#commands-and-components).

## Forms

Inside a `<form>`, `sb-rating` submits its value under its `name` (`name=3.5`, `name=0` when unrated), `new FormData(form)` and Datastar's `contentType: 'form'` include it, and a form reset brings back the server's value. It is not a form-associated element yet (Rocket can't declare one), so `required` and validity, `<fieldset disabled>`, `<label for>` and the `form` attribute don't reach it. With commands, `sb-change` carries `{ name, value }` (see [With commands](#with-commands)).

## Styling

Style it from your page's CSS — no need to change the component or import anything into it. Custom properties, inherited properties and `::part()` all reach into its shadow root.

- **Size:** `size` (`sm`, `md`, `lg`) sets the icons to `1rem`, `1.5rem` or `2.25rem`. For another size, set `inline-size` and `block-size` on `::part(unit)`.
- **Fonts:** the label uses your page's font.
- **Colours:** `--sb-rating-color` fills the icons (by default `--sb-danger` for hearts, `--sb-warn` for stars) and `--sb-rating-shine` is their highlight; empty icons are `--sb-border-strong`. The label is `--sb-text-2`, the focus ring `--sb-brand-light`.
- **Parts:** `label`, `base` (the row of icons) and `unit` (each icon). Your page's `::part()` rules win over the component's own, without `!important`.

```html preview
<style>
  .my-rating { --sb-rating-color: var(--sb-brand-light); }
  .my-rating::part(unit) { inline-size: 2rem; block-size: 2rem; }
</style>
<sb-rating class="my-rating" icon="star" label="Crew comfort" value="4"></sb-rating>
```

## Accessibility

The row is a `slider` (Arrow keys change the value by one step, Home clears it, End fills it), named by `label` or "Rating", with `aria-valuetext` like "3 of 5". Read-only ratings are a slider with `aria-readonly`, outside the tab order, with the same value and text.

Right to left (`dir="rtl"`), the row starts at the right: half units fill from the right, and Left and Right follow the row (Left raises the value), like a native range input.
