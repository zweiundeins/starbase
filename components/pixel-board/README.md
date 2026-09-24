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

A grid of chunky pixels to paint on, with a 16-colour palette. It's built for **server-driven multiplayer**: the board state is one attribute, `cells` (one hex digit per cell, row by row), which the server renders. Painting doesn't change `cells`. It emits `sb-paint`, your backend applies it, and the next server frame brings the new `cells` to everyone. While a pixel is in flight, it's drawn slightly faded; if no frame confirms it within 3 seconds (say the server refused it), it goes back to the board's colour.

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
  data-on:sb-paint="@post('/cmd/paint', {
    payload: {color: evt.detail.color, cells: evt.detail.cells},
    requestCancellation: 'disabled'
  })"></sb-pixel-board>
```

`requestCancellation: 'disabled'` matters: a stroke posts every 80 ms, and by default Datastar cancels a request that is still running when the next one goes to the same URL, which would drop pixels mid-stroke on a slow connection. No `data-preserve-attr` here: `cells` belongs to the server, and every morph brings the latest board.

### Watch only

```html preview
<sb-pixel-board readonly size="16" cells="0000000000000000000000055000000000000055550000000000005775000000000005579550000000000559955000000000055775500000000005555550000000000566665000000000f555555f0000000ff555555ff000000f05333350f0000000003333000000000000eddd0000000000000ee00000000000000000000000" style="--sb-pixel-board-size: 12rem"></sb-pixel-board>
```

## Event

`sb-paint` bubbles out of the shadow root with `detail: { color, cells }`, where `cells` are the indices (`y * size + x`) touched since the last event. Events are batched every ~80 ms during a stroke, with at most 60 cells each (a quick stroke sends more events, not a bigger one, so a server can cap a command's size), and fast drags are interpolated so strokes have no gaps. A stroke that leaves the board and comes back continues where it re-enters, without a line across.

Only the primary button of one pointer paints: a right click doesn't, and a second finger doesn't join the stroke. On touch screens, two fingers zoom the page. A `readonly` board leaves touches to the page, which scrolls over it, and doesn't highlight the cell under the pointer.

## Styling

Style it from your page's CSS — no need to change the component or import anything into it. Custom properties, inherited properties and `::part()` all reach into its shadow root.

- **Size:** `--sb-pixel-board-size` (default `24rem`) is the width of the board, which stays square, with the palette below it. `size` sets the number of cells per side, not the size on screen.
- **Colours:** the paint colours are the `palette` prop (16 hex colours), not theme tokens. The board has a `--sb-border` frame around a `--sb-surface-inset` background; the chosen colour and the focus ring are `--sb-brand-light`. With `grid`, the cell lines are `--sb-pixel-board-grid`. They lie on the cells, not on the page, so by default they follow the blank cell colour (the palette's first): faint white on a dark board, faint black on a light one. The swatches are edged in `--sb-text-1` at 12%, so they stand out on light and dark pages alike.
- **Parts:** `board` (the canvas and the palette together), `canvas` and `palette`. Your page's `::part()` rules win over the component's own, without `!important`.

```html preview
<style>
  .my-board { --sb-pixel-board-size: 16rem; --sb-brand-light: var(--sb-accent); --sb-pixel-board-grid: rgb(255 255 255 / 0.2); }
  .my-board::part(palette) { gap: 2px; }
</style>
<sb-pixel-board class="my-board" local grid size="16"></sb-pixel-board>
```

## Accessibility

The canvas is focusable, with the role `application`, so a screen reader passes the keys on to it: the arrow keys move a highlighted cursor (starting in the middle), and Space or Enter paints. Other keys, like Tab, are left alone, and the cursor goes away when the board loses focus. The canvas label states the cursor position, that cell's colour and the selected colour, and follows all three, including a new colour choice and cells the server changes. The palette is a group of native radio buttons ("Colour 1" to "Colour 16"): one tab stop, and the arrow keys pick the colour. In forced colours (Windows High Contrast) the swatches keep their colours.
