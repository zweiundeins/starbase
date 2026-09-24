---
name: Voxel
tag: sb-voxel
category: media
summary: A tiny 3D voxel renderer you steer with signals. No WebGL.
author: zweiundeins
tags: [3d, canvas, animation, pixel, model]
since: 2026-09-21
preview: |
  <sb-voxel spin="40" pitch="18" style="--sb-voxel-size: 8.5rem"></sb-voxel>
usage: |
  <sb-voxel spin="40"></sb-voxel>
playground:
  props:
    yaw: {min: -180, max: 180}
    pitch: {min: -89, max: 89}
    zoom: {min: 0.25, max: 2.5, step: 0.05}
    spin: {min: -180, max: 180, step: 5}
    light: {min: 0, max: 360, step: 5}
  values: {spin: 30}
---

A software 3D renderer for voxel models: a rocket, a satellite and a ringed planet. It rasterises into a 128×128 buffer with a z-buffer, uses flat, stepped shading and a pixel outline, and is scaled up with crisp edges. Drag it or use the arrow keys to orbit. Every angle is an attribute, so Datastar signals can steer it.

## Examples

### Steered by sliders

Each slider is bound to a local signal, and each signal drives an attribute with `data-attr`. `data-preserve-attr` keeps those attributes when a server frame morphs the page.

```html preview
<div data-signals="{_yaw: 30, _pitch: 15, _zoom: 1, _light: 45}" style="display: flex; flex-wrap: wrap; gap: 24px; align-items: center; inline-size: 100%">
  <sb-voxel
    data-attr:yaw="$_yaw" data-attr:pitch="$_pitch" data-attr:zoom="$_zoom" data-attr:light="$_light"
    data-preserve-attr="yaw pitch zoom light"></sb-voxel>
  <div style="display: grid; gap: 16px; flex: 1; min-inline-size: 14rem">
    <sb-slider label="Yaw" min="-180" max="180" unit="°" data-bind:_yaw__prop.value></sb-slider>
    <sb-slider label="Pitch" min="-89" max="89" unit="°" data-bind:_pitch__prop.value></sb-slider>
    <sb-slider label="Zoom" min="0.25" max="2.5" step="0.05" data-bind:_zoom__prop.value></sb-slider>
    <sb-slider label="Light" min="0" max="360" step="5" unit="°" data-bind:_light__prop.value></sb-slider>
  </div>
</div>
```

### Models

```html preview
<sb-voxel model="rocket" spin="30" style="--sb-voxel-size: 10rem"></sb-voxel>
<sb-voxel model="satellite" spin="-20" pitch="25" style="--sb-voxel-size: 10rem"></sb-voxel>
<sb-voxel model="planet" spin="15" pitch="20" style="--sb-voxel-size: 10rem"></sb-voxel>
```

### Report the orbit

After a drag, `sb-orbit` reports the effective angles.

```html preview
<div data-signals:_orbit="'drag the satellite'">
  <sb-voxel model="satellite" data-on:sb-orbit="$_orbit = 'yaw ' + Math.round(evt.detail.yaw) + '°, pitch ' + Math.round(evt.detail.pitch) + '°'"></sb-voxel>
  <p data-text="$_orbit"></p>
</div>
```

## Performance

It only paints when something changes: a prop, a drag, or each frame while `spin` is set. Offscreen instances pause, and with `prefers-reduced-motion` the model doesn't spin.

## Accessibility

The canvas has `role="img"`, and its label always states the model and current angles. It is focusable, and the arrow keys orbit (Shift for bigger steps).
