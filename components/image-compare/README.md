---
name: Image Compare
tag: sb-image-compare
category: media
summary: Two images on top of each other, with a divider to drag between before and after.
author: zweiundeins
tags: [image, compare, before-after, slider, media]
since: 2026-10-05
preview: |
  <sb-image-compare before-label="Raw" after-label="Processed" position="40" style="inline-size: 15rem">
    <img slot="before" src="/art/hero.svg" alt="" width="360" height="168" style="image-rendering: pixelated; filter: grayscale(1) contrast(1.4) brightness(0.8)">
    <img slot="after" src="/art/hero.svg" alt="" width="360" height="168" style="image-rendering: pixelated">
  </sb-image-compare>
usage: |
  <sb-image-compare before-label="Before" after-label="After" expandable>
    <img slot="before" src="before.png" alt="The page before the change">
    <img slot="after" src="after.png" alt="The page after the change">
  </sb-image-compare>
playground:
  props:
    position: {min: 0, max: 100}
  values: {beforeLabel: Raw, afterLabel: Processed}
  content: '<img slot="before" src="/art/hero.svg" alt="" width="360" height="168" style="image-rendering: pixelated; filter: grayscale(1) contrast(1.4) brightness(0.8)"><img slot="after" src="/art/hero.svg" alt="" width="360" height="168" style="image-rendering: pixelated">'
  style: "inline-size: min(100%, 28rem)"
  sync: {sb-position: [position]}
---

Two pictures of the same size, stacked, with a divider the reader drags to reveal one or the other: a raw and a processed telescope frame, a redesign, two profiler traces of the same page load. Drag anywhere on the picture, tap or click to jump, or focus the divider and use the arrow keys. With `expandable`, a button opens the comparison full screen, among the stars.

The sides are **slotted**, so the page renders them: `<img>` with `srcset`, `<picture>`, lazy loading and alt text stay yours, and any element works, not only images. The `after` side sets the size; the `before` side is cut to it. A `<picture>`'s inner `<img>` sits in the page's DOM, so the page sizes it (`sb-image-compare img { display: block; inline-size: 100% }`). Slotted content doesn't take pointer events: the whole picture is the control.

## Examples

### Raw and processed

```html preview
<sb-image-compare before-label="Raw" after-label="Processed" position="35" style="inline-size: min(100%, 30rem)">
  <img slot="before" src="/art/hero.svg" alt="The launch frame as the sensor recorded it, grey and flat" width="360" height="168" style="image-rendering: pixelated; filter: grayscale(1) contrast(1.4) brightness(0.8)">
  <img slot="after" src="/art/hero.svg" alt="The same frame in colour: a rocket leaving a violet planet, Earth and the moon behind" width="360" height="168" style="image-rendering: pixelated">
</sb-image-compare>
```

Without a label, the divider's value text says "Before" and "After"; with labels it names them, e.g. "Raw 35%, Processed 65%". Each tag sits on its own side, so the divider cuts it as it cuts the picture.

### Full screen

`expandable` adds a button in the corner. Full screen is a popover over a starfield: the same slotted images move to the top layer, scaled to fit the viewport, and the divider keeps its position. Escape or the button closes it; Tab stays between the divider and the button meanwhile. `sb-expand` reports both, and `:state(expanded)` styles the open element from the page. Full screen shows the images the browser already chose for the page: with `srcset`, list a candidate as wide as the screen and keep `sizes` honest, or the picture is upscaled.

```html preview
<sb-image-compare expandable before-label="Infrared" after-label="Visible" style="inline-size: min(100%, 30rem)">
  <img slot="before" src="/art/landscape.svg" alt="Planet X-9 in infrared: the hills glow, the sky is dark" width="256" height="144" style="image-rendering: pixelated; filter: hue-rotate(150deg) saturate(1.6)">
  <img slot="after" src="/art/landscape.svg" alt="Planet X-9 in visible light: a violet planet rising over violet hills" width="256" height="144" style="image-rendering: pixelated">
</sb-image-compare>
```

### Any two elements: two themes

The sides don't have to be images. Here the same card is rendered twice, once in Deep Space and once in Daylight, each scoped to its side with `data-sb-theme`.

```html preview
<sb-image-compare before-label="Deep Space" after-label="Daylight" style="inline-size: min(100%, 24rem)">
  <div slot="before" data-sb-theme="deep-space" style="padding: 20px; background: var(--sb-bg)">
    <sb-card heading="Planet X-9">
      <img slot="media" src="/art/landscape.svg" alt="" width="256" height="144">
      A cold, quiet world with excellent stargazing.
      <sb-button slot="footer" size="sm" variant="pixel">Visit</sb-button>
    </sb-card>
  </div>
  <div slot="after" data-sb-theme="daylight" style="padding: 20px; background: var(--sb-bg)">
    <sb-card heading="Planet X-9">
      <img slot="media" src="/art/landscape.svg" alt="" width="256" height="144">
      A cold, quiet world with excellent stargazing.
      <sb-button slot="footer" size="sm" variant="pixel">Visit</sb-button>
    </sb-card>
  </div>
</sb-image-compare>
```

Both sides must render at the same size: give them the same markup, or fixed dimensions.

### Steered by Datastar

`position` is an attribute, so a signal can drive it, and `sb-position` reports where the reader left the divider. `data-preserve-attr` keeps the bound attribute when a server frame morphs the page.

```html preview
<div data-signals="{_phase: 50}" style="display: grid; gap: 16px; inline-size: min(100%, 18rem)">
  <sb-image-compare before-label="Night side" after-label="Day side"
    data-attr:position="$_phase" data-preserve-attr="position"
    data-on:sb-position="$_phase = evt.detail.position">
    <img slot="before" src="/art/moon.svg" alt="The moon's night side, almost black" width="210" height="210" style="image-rendering: pixelated; filter: brightness(0.3) saturate(0.4)">
    <img slot="after" src="/art/moon.svg" alt="The moon's day side, grey and cratered" width="210" height="210" style="image-rendering: pixelated">
  </sb-image-compare>
  <sb-slider label="Phase" unit="%" data-bind:_phase__prop.value></sb-slider>
</div>
```

A new `position` from the page or the server replaces the reader's, even one equal to the default; re-sending the same markup changes nothing. The live position is the `position` property.

Before the module has loaded, both sides show stacked. The autoloader's `sb-cloak` class on `<html>` hides undefined components until then; without it, hide `sb-image-compare:not(:defined) [slot="before"]` yourself.

## Styling

Style it from your page's CSS, without changing the component or importing anything into it. Custom properties, inherited properties and `::part()` all reach into its shadow root.

- **Size:** the comparison is as wide as its container and as tall as the `after` side at that width. Set `inline-size` (or `max-inline-size`) on the element.
- **8-bit details:** the picture sits in a notched pixel frame in `--sb-frame-color` with a hard drop shadow, `--sb-frame-step` thick (`1px` makes it a hairline, `0` removes it). `--sb-notch` notches the grip, the tags and the button (`0` rounds them), and the tags use `--sb-font-display`. `data-sb-style="smooth"` turns all three off.
- **Colours:** `--sb-image-compare-line` colours the divider, `--sb-image-compare-handle` and `--sb-image-compare-handle-ink` the grip and its arrows (by default `--sb-text-1` on `--sb-bg`), with a bevel in `--sb-brand-light`. The tags and the button are `--sb-text-1` on a translucent `--sb-bg`. Full screen is `--sb-bg` with stars in `--sb-text-muted` and `--sb-text-1`; `--sb-image-compare-stars: none` leaves the background plain.
- **Parts:** `frame` (also the full screen view), `stage` (the framed picture), `before`, `after`, `divider`, `handle`, `tag` (both tags; `before-tag`, `after-tag` for one), and `expand` (the button). Your page's `::part()` rules win over the component's own, without `!important`.

```html preview
<style>
  .my-compare {
    --sb-frame-color: #FFD166;
    --sb-image-compare-line: #FFD166;
    --sb-image-compare-handle: #FFD166;
    --sb-notch: 0;
    inline-size: min(100%, 20rem);
  }
  .my-compare::part(tag) { font-family: Georgia, serif; text-transform: none; letter-spacing: 0; }
</style>
<sb-image-compare class="my-compare" before-label="Mono" after-label="Colour">
  <img slot="before" src="/art/saturn.svg" alt="A ringed planet in grey" width="260" height="180" style="image-rendering: pixelated; filter: grayscale(1)">
  <img slot="after" src="/art/saturn.svg" alt="A violet planet with blue rings" width="260" height="180" style="image-rendering: pixelated">
</sb-image-compare>
```

## Accessibility

The divider is a native range input, invisible over the picture: Tab focuses it (the frame gets a focus ring), the arrow keys move it by 1 %, Page Up/Down by 10 %, Home and End to either edge. Its value text names both sides; `label` names the control ("Comparison" by default). Pointer drags move the same value, and after a drag or a tap the divider has the focus, so the keys continue from there.

On touch screens a horizontal drag moves the divider and a vertical swipe scrolls the page without moving it. Full screen is announced as a dialog named by `label`; `expand-label` and `close-label` name the button in your language. In right-to-left text the before side sits on the right.

Give each image its own alt text: a reader who can't see the pictures needs to know what differs.
