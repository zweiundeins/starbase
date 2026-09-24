---
name: Nebula
tag: sb-nebula
category: media
summary: A drifting WebGL nebula in dithered pixels. Speed, density and palette are attributes.
author: mbolli
tags: [webgl, shader, background, space, animation]
since: "2026-09-22"
preview: |
    <sb-nebula speed="1.2" density="0.6" style="--sb-nebula-height: 7.5rem"></sb-nebula>
usage: |
  <sb-nebula></sb-nebula>
playground:
    props:
        density:
            max: 1
            min: 0
            step: 0.05
        levels:
            max: 16
            min: 0
        pixel:
            max: 16
            min: 1
        seed:
            max: 100
            min: 0
        speed:
            max: 5
            min: 0
            step: 0.1
    values:
        density: 0.6
        speed: 1.5
---

Clouds of gas and a few twinkling stars, rendered by a fragment shader. The shader draws domain-warped noise at low resolution and posterizes it with an ordered dither, so it looks like pixel art. Colours come from theme tokens, and the nebula leans toward the pointer.

## Examples

### Controls

```html preview
<div data-signals="{_speed: 1, _density: 0.5, _pixel: 4}" style="display: grid; gap: 16px; inline-size: 100%">
  <sb-nebula data-attr:speed="$_speed" data-attr:density="$_density" data-attr:pixel="$_pixel" data-preserve-attr="speed density pixel"></sb-nebula>
  <div style="display: flex; gap: 24px; flex-wrap: wrap">
    <sb-slider label="Speed" max="5" step="0.1" style="flex: 1; min-inline-size: 10rem" data-bind:_speed__prop.value></sb-slider>
    <sb-slider label="Density" max="1" step="0.05" style="flex: 1; min-inline-size: 10rem" data-bind:_density__prop.value></sb-slider>
    <sb-slider label="Pixel size" min="1" max="16" style="flex: 1; min-inline-size: 10rem" data-bind:_pixel__prop.value></sb-slider>
  </div>
</div>
```

### Palettes

`palette` picks the colours and `seed` the clouds. Any number is a seed; seeds 101 apart give the same sky.

```html preview
<div style="display: grid; grid-template-columns: repeat(auto-fit, minmax(10rem, 1fr)); gap: 12px; inline-size: 100%">
  <sb-nebula palette="violet" style="--sb-nebula-height: 7rem"></sb-nebula>
  <sb-nebula palette="aurora" seed="7" style="--sb-nebula-height: 7rem"></sb-nebula>
  <sb-nebula palette="ember" seed="3" density="0.7" style="--sb-nebula-height: 7rem"></sb-nebula>
  <sb-nebula palette="mono" seed="12" style="--sb-nebula-height: 7rem"></sb-nebula>
</div>
```

### Smooth

`levels="0"` turns off the posterizing, and `pixel="1"` renders at full resolution.

```html preview
<sb-nebula levels="0" pixel="2" speed="0.6" style="--sb-nebula-height: 9rem"></sb-nebula>
```

### Decoration with a pause switch

`label=""` makes it decoration, hidden from assistive technology. Motion that goes on for more than five seconds next to other content needs a way to stop it: here a switch sets `speed` to 0.

```html preview
<div data-signals:_paused="false" style="display: grid; gap: 12px; inline-size: 100%">
  <sb-nebula label="" data-attr:speed="$_paused ? 0 : 1" data-preserve-attr="speed" style="--sb-nebula-height: 8rem"></sb-nebula>
  <sb-toggle label="Pause the nebula" data-bind:_paused__prop.checked></sb-toggle>
</div>
```

## Performance

One full-screen triangle and a single shader, rendered at a quarter of the CSS resolution by default (`pixel="4"`). It stops animating offscreen and when `speed` is 0. Each nebula has its own WebGL context, and frees it when removed. Browsers keep only so many contexts live on a page (Chrome about 16) and take the oldest away beyond that; such a context doesn't come back, while one lost to a GPU reset does. Without WebGL, or once its context is gone, it shows a still `--sb-brand` glow. Set the height with `--sb-nebula-height`.

## Styling

Style it from your page's CSS — no need to change the component or import anything into it. Custom properties, inherited properties and `::part()` all reach into its shadow root.

- **Size:** it fills the width it is given; `--sb-nebula-height` (default `14rem`) sets the height.
- **Colours:** the cloud is painted from theme tokens, read whenever it paints: every frame while it drifts, and at once on a theme change (a pick on `<sb-theme-switch>`, or the system turning light or dark), even when it is still. An override in your own CSS shows with the next paint. `palette` picks them: `violet` uses `--sb-brand`, `--sb-accent` and `--sb-text-1`; `aurora` `--sb-accent`, `--sb-datastar` and `--sb-text-1`; `ember` `--sb-danger`, `--sb-warn` and `--sb-text-1`; `mono` `--sb-border-strong`, `--sb-text-muted` and `--sb-text-1`. All of them sit on `--sb-surface-inset`. On a light theme, where `--sb-surface-inset` is lighter than `--sb-text-1`, it sorts the colours by lightness, so the sky stays dark and the stars light. Without WebGL, or with its context lost, it shows a still `--sb-brand` glow instead. Corners are `--sb-radius`.
- **Parts:** `canvas`. Your page's `::part()` rules win over the component's own, without `!important`.

```html preview
<style>
  .my-nebula { --sb-nebula-height: 8rem; --sb-radius: 0; --sb-brand: #EC4899; }
</style>
<sb-nebula class="my-nebula"></sb-nebula>
```

## Accessibility

It is an image named "Animated nebula"; change the name with `label`. Where it is only decoration, set `label=""`: it is then hidden from assistive technology. Under `prefers-reduced-motion` it shows one still frame and ignores the pointer. Otherwise it drifts for as long as it is on screen, so next to other content give people a way to pause it (WCAG 2.2.2): `speed="0"` holds it still, as in [the pause switch above](#decoration-with-a-pause-switch).
