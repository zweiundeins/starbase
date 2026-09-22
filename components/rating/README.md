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

A row of pixel hearts (or stars) for a score. Point to preview, click to pick, or use the arrow keys. The live value is the `value` property, so `data-bind` works, and a new `value` attribute from the server replaces it.

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

For showing a score, e.g. an average: `readonly` turns it into an image with a text alternative ("4.5 of 5").

```html preview
<sb-rating readonly value="4.5" precision="0.5" size="sm"></sb-rating>
```

### Clearable

With `clearable`, picking the current value again resets it to 0.

```html preview
<sb-rating clearable value="2"></sb-rating>
```

## With commands

Give it a `name`, and it emits `sb-change` with `{ name, value }` when a value is picked: ready to post as a command. With `confirm`, it sets `:state(pending)` until the server's re-rendered attribute matches, and `revert()` goes back to the server's value when a command is rejected. See [Commands and components](/contribute#commands-and-components).

## Theming

The filled colour is `--sb-rating-color` (hearts default to `--sb-danger`, stars to `--sb-warn`); empty units use `--sb-border-strong`.

## Accessibility

The row is a `slider` (Arrow keys change the value by one step, Home clears it, End fills it), named by `label` or "Rating", with `aria-valuetext` like "3 of 5". Read-only ratings are an `img` with the same text.
