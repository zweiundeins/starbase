---
name: Pixel Board
tag: sb-pixel-board
category: experimental
summary: "A paintable pixel canvas, made for multiplayer: the server owns the cells."
author: zweiundeins
tags: [canvas, pixel, paint, multiplayer, realtime]
since: 2026-09-21
preview: |
  <sb-pixel-board local readonly size="16" cells="0000000000000000000000055000000000000055550000000000005775000000000005579550000000000559955000000000055775500000000005555550000000000566665000000000f555555f0000000ff555555ff000000f05333350f0000000003333000000000000eddd0000000000000ee00000000000000000000000" style="--sb-pixel-board-size: 7.5rem"></sb-pixel-board>
usage: |
  <sb-pixel-board local grid size="24"></sb-pixel-board>
playground:
  props:
    size: {min: 8, max: 64}
    color: {min: 0, max: 15}
  values: {local: true, size: 16, grid: true}
  exclude: [cells, palette]
  style: "--sb-pixel-board-size: 18rem"
---

A grid of chunky pixels to paint on, with a 16-colour palette. It's built for **server-driven multiplayer**: the board state is one attribute, `cells` (one hex digit per cell, row by row), which the server renders. Painting doesn't change `cells`. It emits `sb-paint`, your backend applies it, and the next server frame brings the new `cells` to everyone. While a pixel is in flight, it's drawn slightly faded.

## Examples

### Local mode

`local` paints the board's own buffer, so no server is needed. Drag to draw, pick a colour below, or focus the board and use the arrow keys plus Space.

```html preview
<sb-pixel-board local grid size="24" style="--sb-pixel-board-size: 20rem"></sb-pixel-board>
```

### Server-driven (the Showcase board)

This is the wiring the multiplayer board on the [Showcase](/showcase) page uses. The server renders `cells` into every frame of the page's render stream, and painting is a command:

```html
<sb-pixel-board id="board" size="48" cells="…rendered by the server…"
  data-on:sb-paint="@post('/cmd/paint', {payload: {tabid: $tabid, color: evt.detail.color, cells: evt.detail.cells}})">
</sb-pixel-board>
```

No `data-preserve-attr` here: `cells` belongs to the server, and every morph brings the latest board.

### Watch only

```html preview
<sb-pixel-board readonly size="16" cells="0000000000000000000000055000000000000055550000000000005775000000000005579550000000000559955000000000055775500000000005555550000000000566665000000000f555555f0000000ff555555ff000000f05333350f0000000003333000000000000eddd0000000000000ee00000000000000000000000" style="--sb-pixel-board-size: 12rem"></sb-pixel-board>
```

## Event

`sb-paint` bubbles out of the shadow root with `detail: { color, cells }`, where `cells` are the indices (`y * size + x`) touched since the last event. Events are batched every ~80 ms during a stroke, and fast drags are interpolated so strokes have no gaps.

## Styling

Style it from your page's CSS — no need to change the component or import anything into it. Custom properties, inherited properties and `::part()` all reach into its shadow root.

- **Size:** `--sb-pixel-board-size` (default `24rem`) is the width of the board, which stays square, with the palette below it. `size` sets the number of cells per side, not the size on screen.
- **Colours:** the paint colours are the `palette` prop (16 hex colours), not theme tokens. The board has a `--sb-border` frame around a `--sb-surface-inset` background; the chosen colour and the focus ring are `--sb-brand-light`.
- **Parts:** `board` (the canvas and the palette together), `canvas` and `palette`. Your page's `::part()` rules win over the component's own, without `!important`.

```html preview
<style>
  .my-board { --sb-pixel-board-size: 16rem; --sb-brand-light: var(--sb-accent); }
  .my-board::part(palette) { gap: 2px; }
</style>
<sb-pixel-board class="my-board" local grid size="16"></sb-pixel-board>
```

## Accessibility

The canvas is focusable: the arrow keys move a highlighted cursor, and Space or Enter paints. Its label always states the cursor position, that cell's colour and the selected colour. The palette is a radio group of labelled buttons.
