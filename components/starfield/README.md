---
name: Starfield
tag: sb-starfield
category: media
summary: A warp-speed pixel starfield. Speed, density and colour are attributes.
author: zweiundeins
tags: [canvas, animation, background, space]
since: 2026-09-21
preview: |
  <sb-starfield speed="35" density="260" tint="violet" warp style="--sb-starfield-height: 7.5rem"></sb-starfield>
usage: |
  <sb-starfield></sb-starfield>
playground:
  props:
    speed: {min: 0, max: 100}
    density: {min: 20, max: 1500, step: 10}
  values: {speed: 40, warp: true}
---

Stars fly toward you in chunky pixels. Every knob is an attribute, so a slider (or the server) can throttle up to warp. Colours come from theme tokens, so the field follows the page's theme.

## Examples

### Throttle

```html preview
<div data-signals="{_speed: 20, _warp: false}" style="display: grid; gap: 16px; inline-size: 100%">
  <sb-starfield data-attr:speed="$_speed" data-attr:warp="$_warp" data-preserve-attr="speed warp" tint="cyan"></sb-starfield>
  <div style="display: flex; gap: 24px; align-items: end; flex-wrap: wrap">
    <sb-slider label="Speed" max="100" style="flex: 1; min-inline-size: 12rem" data-bind:_speed__prop.value></sb-slider>
    <sb-toggle label="Warp" data-bind:_warp__prop.checked></sb-toggle>
  </div>
</div>
```

### Tints

```html preview
<div style="display: grid; grid-template-columns: repeat(auto-fit, minmax(10rem, 1fr)); gap: 12px; inline-size: 100%">
  <sb-starfield tint="white" style="--sb-starfield-height: 6rem"></sb-starfield>
  <sb-starfield tint="violet" style="--sb-starfield-height: 6rem"></sb-starfield>
  <sb-starfield tint="cyan" style="--sb-starfield-height: 6rem"></sb-starfield>
  <sb-starfield tint="green" warp speed="60" style="--sb-starfield-height: 6rem"></sb-starfield>
</div>
```

## Performance

It renders at a third of the CSS resolution into one `ImageData`. It stops animating offscreen, when `speed` is 0, or under `prefers-reduced-motion`, where it shows a single still frame. Set the height with `--sb-starfield-height`.

## Accessibility

It is decorative motion: the canvas is labelled "Animated starfield". Don't rely on it to convey information.
