---
name: Bento Workspace
tag: sb-bento-workspace
category: experimental
summary: Move and resize tiles on CSS grids by pointer or keyboard. The server places them.
author: derekr
license: Beerware
source: https://github.com/derekr/pd-rockets/tree/4f4111722763d2aa8f5851c397af026fcbe97e4f/rocket/bento
tags: [drag and drop, dashboard, grid, bento, tiles, resize, keyboard, pd rockets]
since: 2026-10-06
preview: |
  <style>
    .demo-bento-card { display: block; inline-size: 100%; max-inline-size: 16rem; }
    .demo-bento-card [data-bento-grid] { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); grid-auto-rows: 2.5rem; gap: 6px; }
    .demo-bento-card [data-bento-item], [data-drag-preview][data-bento-item] { display: grid; place-items: center; min-inline-size: 0; border: 1px solid var(--sb-border); background: var(--sb-surface-card); color: var(--sb-text-1); font-size: 0.75rem; cursor: grab; touch-action: none; user-select: none; }
    .demo-bento-card :is([data-dragging], [data-bento-projecting]) { opacity: 0.35; }
    .demo-bento-card [data-bento-target] { border: 2px dashed var(--sb-brand); background: color-mix(in oklch, var(--sb-brand) 14%, transparent); }
    [data-drag-preview][data-bento-item] { box-shadow: var(--sb-shadow-overlay, 0 8px 16px rgb(0 0 0 / 0.35)); cursor: grabbing; }
    @media (prefers-reduced-motion: no-preference) { .demo-bento-card[data-drag-active] [data-bento-item] { transition: transform 160ms ease; } }
    @media (forced-colors: active) {
      .demo-bento-card [data-bento-target] { border-color: Highlight; }
      .demo-bento-card :is([data-dragging], [data-bento-projecting]) { opacity: 1; border: 2px dashed CanvasText; }
      [data-drag-preview][data-bento-item] { border: 2px solid Highlight; }
    }
  </style>
  <sb-bento-workspace id="bento-card" class="demo-bento-card" data-state="deck thrust.1.1.2.2 fuel.3.1.1.1 crew.3.2.1.1"
    data-on:sb-bento-move="@get('/demo/arrange/bento-card', {payload: {id: el.id, state: el.dataset.state, move: evt.detail}})">
    <div data-bento-grid="deck" data-columns="3">
      <div data-bento-item="thrust" data-bento-col="1" data-bento-row="1" data-bento-width="2" data-bento-height="2" style="grid-column: 1 / span 2; grid-row: 1 / span 2">Thrust</div>
      <div data-bento-item="fuel" data-bento-col="3" data-bento-row="1" data-bento-width="1" data-bento-height="1" style="grid-column: 3 / span 1; grid-row: 1 / span 1">Fuel</div>
      <div data-bento-item="crew" data-bento-col="3" data-bento-row="2" data-bento-width="1" data-bento-height="1" style="grid-column: 3 / span 1; grid-row: 2 / span 1">Crew</div>
    </div>
  </sb-bento-workspace>
usage: |
  <sb-bento-workspace data-on:sb-bento-move="@post('/tiles/move', {payload: evt.detail})"
    data-on:sb-bento-resize="@post('/tiles/resize', {payload: evt.detail})">
    <div data-bento-grid="main" data-columns="4" style="display: grid; grid-template-columns: repeat(4, 1fr); grid-auto-rows: 80px">
      <div data-bento-item="a" data-bento-col="1" data-bento-row="1" data-bento-width="2" data-bento-height="1"
        style="grid-column: 1 / span 2; grid-row: 1" tabindex="0">A</div>
    </div>
  </sb-bento-workspace>
---

From [PD rockets](https://github.com/derekr/pd-rockets) by derekr, where it is `pd-bento-workspace`, with its names in Starbase's `sb-` prefix and the patches in `patches/pd-rockets`. For this surface they let it nest with the other PD rockets surfaces, drop its slide under reduced motion, declare its events and keep the focus on a tile moved by pointer. Upstream calls it experimental.

Tiles on one or more CSS grids, which the server renders and places. Drag a tile to another cell or grid, drag its corner to resize it, or do both from the keyboard. The component shows the layout the change would give: tiles in the way move down to the next free row. Then it emits `sb-bento-move` or `sb-bento-resize` with every position that changed. It places nothing itself: the server applies the positions and sends the tiles back, and the morph puts each one in its place. When a tile changes grid, the tiles slide to their new places; a move within a grid lands at once.

## Examples

### A mission dashboard on two grids

Each tile holds another component (a gauge, two meters, a sparkline). The server renders the dashboard with its layout in `data-state`. A move or a resize sends that layout with the event's detail to `/demo/arrange/bento-workspace`, which checks the new positions (every tile inside its grid and its first 30 rows, none overlapping) and answers with the dashboard rendered again. A move it refuses, such as one past row 30, falls back after 2 seconds. Nothing is stored: try it in two tabs.

```html preview
<style>
  .demo-bento { display: flex; flex-wrap: wrap; gap: 12px; inline-size: 100%; }
  .demo-bento__panel { flex: 2 1 18rem; min-inline-size: 0; display: grid; gap: 6px; align-content: start; }
  .demo-bento__panel:last-child { flex: 1 1 9rem; }
  .demo-bento__label { color: var(--sb-text-2); font-size: 0.8125rem; font-weight: 600; }
  .demo-bento [data-bento-grid] {
    display: grid;
    grid-auto-rows: 4.5rem;
    gap: 8px;
    min-block-size: calc(2 * 4.5rem + 8px);
    padding: 8px;
    border: 1px dashed var(--sb-border);
    background: var(--sb-surface-inset);
  }
  .demo-bento [data-bento-grid="deck"] { grid-template-columns: repeat(4, minmax(0, 1fr)); }
  .demo-bento [data-bento-grid="shelf"] { grid-template-columns: repeat(2, minmax(0, 1fr)); }
  .demo-bento [data-bento-item], [data-drag-preview][data-bento-item] {
    position: relative;
    display: grid;
    grid-template-columns: minmax(0, 1fr);
    place-items: center;
    min-inline-size: 0;
    padding: 8px;
    overflow: hidden;
    border: 1px solid var(--sb-border);
    background: var(--sb-surface-card);
    color: var(--sb-text-1);
    cursor: grab;
    touch-action: none;
    user-select: none;
  }
  :is(.demo-bento, [data-drag-preview][data-bento-item]) :is(sb-meter, sb-sparkline) { inline-size: 100%; }
  :is(.demo-bento, [data-drag-preview][data-bento-item]) sb-gauge { --sb-gauge-size: 8rem; }
  :is(.demo-bento [data-bento-item], [data-drag-preview][data-bento-item]) p { margin: 0; }
  .demo-bento [data-bento-item]:focus-visible { outline: 2px solid var(--sb-brand-light); outline-offset: 2px; }
  .demo-bento[data-key-staging] [data-bento-item]:focus { border-color: var(--sb-brand); }
  .demo-bento :is([data-dragging], [data-bento-projecting]) { opacity: 0.35; }
  .demo-bento [data-bento-target] {
    border: 2px dashed var(--sb-brand);
    background: color-mix(in oklch, var(--sb-brand) 14%, transparent);
  }
  .demo-bento [data-bento-resize] {
    position: absolute;
    inset: auto 2px 2px auto;
    inline-size: 12px;
    block-size: 12px;
    background: linear-gradient(135deg, transparent 50%, var(--sb-text-muted) 50%);
    cursor: nwse-resize;
  }
  [data-drag-preview][data-bento-item] { box-shadow: var(--sb-shadow-overlay, 0 8px 16px rgb(0 0 0 / 0.35)); cursor: grabbing; }
  @media (prefers-reduced-motion: no-preference) {
    .demo-bento:is([data-drag-active], [data-resize-active], [data-key-staging]) [data-bento-item] { transition: transform 160ms ease; }
  }
  @media (forced-colors: active) {
    .demo-bento [data-bento-resize] { forced-color-adjust: none; background: linear-gradient(135deg, transparent 50%, CanvasText 50%); }
    .demo-bento [data-bento-item]:focus-visible { outline-color: Highlight; }
    .demo-bento[data-key-staging] [data-bento-item]:focus { border: 2px solid Highlight; }
    .demo-bento [data-bento-target] { border-color: Highlight; }
    .demo-bento :is([data-dragging], [data-bento-projecting]) { opacity: 1; border: 2px dashed CanvasText; }
    [data-drag-preview][data-bento-item] { border: 2px solid Highlight; }
  }
</style>
<sb-bento-workspace id="mission-deck" class="demo-bento" data-state="deck thrust.1.1.2.2 fuel.3.1.2.1 speed.3.2.2.1 shelf shields.1.1.2.1 crew.1.2.1.1"
	data-on:sb-bento-move="@get('/demo/arrange/bento-workspace', {payload: {id: el.id, state: el.dataset.state, move: evt.detail}})"
	data-on:sb-bento-resize="@get('/demo/arrange/bento-workspace', {payload: {id: el.id, state: el.dataset.state, move: evt.detail}})">
	<div class="demo-bento__panel">
		<span class="demo-bento__label">Deck</span>
		<div data-bento-grid="deck" data-columns="4" role="group" aria-label="Deck">
			<article data-bento-item="thrust" data-bento-col="1" data-bento-row="1" data-bento-width="2" data-bento-height="2"
				style="grid-column: 1 / span 2; grid-row: 1 / span 2" tabindex="0" aria-label="Thrust">
				<sb-gauge value="72" label="Thrust" unit="%"></sb-gauge>
				<span data-bento-resize aria-hidden="true"></span>
			</article>
			<article data-bento-item="fuel" data-bento-col="3" data-bento-row="1" data-bento-width="2" data-bento-height="1"
				style="grid-column: 3 / span 2; grid-row: 1 / span 1" tabindex="0" aria-label="Fuel">
				<sb-meter label="Fuel" value="64" unit="%"></sb-meter>
				<span data-bento-resize aria-hidden="true"></span>
			</article>
			<article data-bento-item="speed" data-bento-col="3" data-bento-row="2" data-bento-width="2" data-bento-height="1"
				style="grid-column: 3 / span 2; grid-row: 2 / span 1" tabindex="0" aria-label="Speed">
				<sb-sparkline values="[12,18,15,22,30,26,34,41]" show-value unit=" km/s"></sb-sparkline>
				<span data-bento-resize aria-hidden="true"></span>
			</article>
		</div>
	</div>
	<div class="demo-bento__panel">
		<span class="demo-bento__label">Shelf</span>
		<div data-bento-grid="shelf" data-columns="2" role="group" aria-label="Shelf">
			<article data-bento-item="shields" data-bento-col="1" data-bento-row="1" data-bento-width="2" data-bento-height="1"
				style="grid-column: 1 / span 2; grid-row: 1 / span 1" tabindex="0" aria-label="Shields">
				<sb-meter label="Shields" value="88" unit="%"></sb-meter>
				<span data-bento-resize aria-hidden="true"></span>
			</article>
			<article data-bento-item="crew" data-bento-col="1" data-bento-row="2" data-bento-width="1" data-bento-height="1"
				style="grid-column: 1 / span 1; grid-row: 2 / span 1" tabindex="0" aria-label="Crew">
				<p><strong>7</strong> aboard</p>
				<span data-bento-resize aria-hidden="true"></span>
			</article>
		</div>
	</div>
</sb-bento-workspace>
```

## Markup and events

Each element with `data-bento-grid="<id>"` inside the workspace is a grid, and each element with `data-bento-item="<id>"` inside a grid is a tile. The component reads a tile's place from `data-bento-col`, `data-bento-row`, `data-bento-width` and `data-bento-height` (cells, counted from 1), and a grid's width from `data-columns` (default 4). Your markup places the tiles to match, for example with `grid-column` and `grid-row`, and gives each tile `tabindex="0"`. An element with `data-bento-resize` inside a tile is its resize handle.

The pointer geometry needs CSS grids with `grid-template-columns` of that many columns and a fixed `grid-auto-rows`. The component sets no limit on rows (the demo's server refuses rows past 30); a tile is at most 5 rows high.

| Event | Detail | When |
|---|---|---|
| `sb-bento-move` | `{ itemId, fromGrid, toGrid, updates }` | A tile was dropped on another cell or grid, or a keyboard move committed. |
| `sb-bento-resize` | `{ itemId, grid, updates }` | A tile was resized by its handle or the keyboard. |

`updates` lists every tile whose place changes in the grid the tile lands in, the tile itself included: `{ itemId, grid, col, row, width, height }`. Tiles in the way move down to the next free row; the grid they leave stays as it is. Both events bubble.

## Keyboard

| Keys | Action |
|---|---|
| Arrow keys, or h / j / k / l | Focus the nearest tile in that direction; at a grid's edge, Left and Up go to the previous grid, Right and Down to the next |
| Home / End | Focus the first / last tile |
| Alt + arrow keys, or Alt + h / j / k / l | Move the focused tile one cell; past a grid's left or right edge it goes to the neighbouring grid. Releasing Alt sends the move |
| Shift + arrow keys | Resize the focused tile; releasing Shift sends the resize |
| Alt + Page Up / Page Down | Move the focused tile to the first cell of the previous / next grid |
| Escape | Cancel a move or resize before the key is released |

On macOS, Ctrl + n and Ctrl + p also move the focus. Each action takes its keys from a `data-key-<action>` attribute on the workspace, a space-separated list such as `data-key-move-right="Alt+ArrowRight Alt+l"`; an empty value turns the action off. The actions are `focus-next`, `focus-previous`, `focus-left`, `focus-right`, `focus-first`, `focus-last`, `move-up`, `move-down`, `move-left`, `move-right`, `resize-up`, `resize-down`, `resize-left`, `resize-right`, `grid-previous`, `grid-next` and `cancel`.

## On the server

The server owns the layout. A handler checks the positions, applies them and renders the tiles again; this is the demo's, which keeps the layout in the markup instead of a database:

```go source=internal/web/demo_arrange_bento.go#bentoDemo.arrange,renderBento
// arrange applies an sb-bento-move ({itemId, fromGrid, toGrid, updates})
// or an sb-bento-resize ({itemId, grid, updates}): updates are the new places
// of every tile that changed in the grid the tile lands in.
func (d bentoDemo) arrange(state string, move json.RawMessage) (string, error) {
	tiles, err := d.parse(state)
	if err != nil {
		return "", err
	}
	var m struct {
		ItemID   string        `json:"itemId"`
		FromGrid string        `json:"fromGrid"` // a move
		ToGrid   string        `json:"toGrid"`
		Grid     string        `json:"grid"` // a resize
		Updates  []bentoUpdate `json:"updates"`
	}
	if err := json.Unmarshal(move, &m); err != nil {
		return "", err
	}
	at := slices.IndexFunc(tiles, func(t bentoTile) bool { return t.id == m.ItemID })
	from, to := cmp.Or(m.FromGrid, m.Grid), cmp.Or(m.ToGrid, m.Grid)
	moved := slices.ContainsFunc(m.Updates, func(u bentoUpdate) bool { return u.ItemID == m.ItemID })
	if at < 0 || tiles[at].grid != from || len(m.Updates) > len(tiles) || !moved {
		return "", fmt.Errorf("not a move of %q", m.ItemID)
	}
	for _, u := range m.Updates {
		i := slices.IndexFunc(tiles, func(t bentoTile) bool { return t.id == u.ItemID })
		if i < 0 || u.Grid != to {
			return "", fmt.Errorf("an update of %q outside the grid it lands in", u.ItemID)
		}
		tiles[i] = bentoTile{u.ItemID, u.Grid, u.Col, u.Row, u.Width, u.Height}
	}
	if err := d.check(tiles); err != nil {
		return "", err
	}
	return d.format(tiles), nil
}

// renderBento is the dashboard's markup: the host the morph replaces, a
// grid per panel, and each tile at its place.
func renderBento(id, state string) string {
	d := bentoDashboard
	tiles, err := d.parse(state)
	if err != nil {
		return "invalid bento state: " + err.Error()
	}
	var b strings.Builder
	fmt.Fprintf(&b, "<sb-bento-workspace id=\"%s\" class=\"demo-bento\" data-state=\"%s\"\n\tdata-on:sb-bento-move=\"%s\"\n\tdata-on:sb-bento-resize=\"%[3]s\">\n", id, state, arrangeOn("bento-workspace"))
	for _, g := range d.grids {
		fmt.Fprintf(&b, "\t<div class=\"demo-bento__panel\">\n\t\t<span class=\"demo-bento__label\">%s</span>\n", g.label)
		fmt.Fprintf(&b, "\t\t<div data-bento-grid=\"%s\" data-columns=\"%d\" role=\"group\" aria-label=\"%s\">\n", g.id, g.columns, g.label)
		for _, t := range tiles {
			if t.grid != g.id {
				continue
			}
			fmt.Fprintf(&b, "\t\t\t<article data-bento-item=\"%s\" data-bento-col=\"%d\" data-bento-row=\"%d\" data-bento-width=\"%d\" data-bento-height=\"%d\"\n", t.id, t.col, t.row, t.width, t.height)
			fmt.Fprintf(&b, "\t\t\t\tstyle=\"grid-column: %d / span %d; grid-row: %d / span %d\" tabindex=\"0\" aria-label=\"%s\">\n", t.col, t.width, t.row, t.height, d.tiles[t.id].label)
			fmt.Fprintf(&b, "\t\t\t\t%s\n\t\t\t\t<span data-bento-resize aria-hidden=\"true\"></span>\n\t\t\t</article>\n", d.tiles[t.id].body)
		}
		b.WriteString("\t\t</div>\n\t</div>\n")
	}
	b.WriteString("</sb-bento-workspace>")
	return b.String()
}
```

## Styling

The component adds no styles: the workspace, its grids and its tiles are your page's markup, styled by your page's CSS. While a tile moves or grows, it marks the elements involved:

- `data-drag-active` on the workspace and `data-dragging` on the tile while you drag it; `data-resize-active` on the workspace and `data-bento-resizing` on the tile while you drag its handle.
- `data-bento-projecting` on the tile whose new place is shown, and a `data-bento-target` element in the grid at that place. The tiles in the way move there with an inline `transform`, and a grid that needs more rows gets a `min-height`. Transition that `transform` only while `data-drag-active`, `data-resize-active` or `data-key-staging` is on the workspace, as the example does: the morph swaps it for the new grid place at once, and a transition would then start the tile a row too far.
- `data-key-staging` on the workspace while a keyboard move or resize waits for its key to be released.
- The tile under the pointer is a copy of it in `<body>`, marked `data-drag-preview`, with the original's size in `--sb-source-width` and `--sb-source-height`. Rules scoped to the workspace don't reach it: the example scopes its tile rules with `:is(.demo-bento, [data-drag-preview][data-bento-item])`. A `<template data-sb-preview>` that is a direct child of a tile replaces the copy.
- A `<template data-sb-target="cell">` that is a direct child of the workspace is copied into the target cell, marked `data-sb-target-indicator`.

The shown layout stays for up to 2 seconds, or until the server's answer replaces it. The example above shows one way to draw all of it.

## Accessibility

Tiles are focusable elements of your page, so their roles and names are yours; the example gives each grid `role="group"` and a name, and each tile an `aria-label`. Moving and resizing need no pointer. The component announces nothing itself. A tile you move or resize, by keyboard or pointer, keeps the focus if it had it, when the server's answer comes within 2 seconds and puts the tile where the component showed it.

## Licence

The code is PD rockets', under its Beer-Ware licence, in `LICENSE-pd-rockets.txt` next to it. `go run ./cmd/vendorpd` brings in a newer release.
