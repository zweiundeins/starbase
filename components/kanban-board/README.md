---
name: Kanban Board
tag: sb-kanban-board
category: data
summary: Lanes of cards to drag, or to move with Alt and the arrows. The server applies each move.
author: derekr
source: https://github.com/derekr/pd-rockets/tree/4f4111722763d2aa8f5851c397af026fcbe97e4f/rocket/kanban
tags: [drag and drop, kanban, board, lanes, cards, keyboard, pd rockets]
since: 2026-10-06
preview: |
  <style>
    .demo-kanban-card { display: grid; grid-template-columns: repeat(3, 6rem); gap: 6px; }
    .demo-kanban-card [data-kanban-lane] { display: grid; align-content: start; gap: 4px; padding: 4px; border: 1px solid var(--sb-border); background: var(--sb-surface-inset); font-size: 0.75rem; }
    .demo-kanban-card [data-kanban-card] { padding: 0.25rem 0.4rem; border: 1px solid var(--sb-border); background: var(--sb-surface-card); color: var(--sb-text-1); white-space: nowrap; }
  </style>
  <sb-kanban-board class="demo-kanban-card">
    <section data-kanban-lane data-col="0"><div data-kanban-lane-cards><article data-kanban-card="mars" tabindex="0">🪐 Mars</article></div></section>
    <section data-kanban-lane data-col="1"><div data-kanban-lane-cards><article data-kanban-card="europa" tabindex="0">🌑 Europa</article></div></section>
    <section data-kanban-lane data-col="2"><div data-kanban-lane-cards><article data-kanban-card="moon" tabindex="0">🌑 Moon</article></div></section>
  </sb-kanban-board>
usage: |
  <sb-kanban-board data-on:sb-kanban-move="@post('/board/move', {payload: evt.detail})">
    <section data-kanban-lane data-col="0" aria-label="To do">
      <div data-kanban-lane-cards>
        <article data-kanban-card="a" tabindex="0">First</article>
      </div>
    </section>
    <section data-kanban-lane data-col="1" aria-label="Done">
      <div data-kanban-lane-cards></div>
    </section>
  </sb-kanban-board>
---

From [PD rockets](https://github.com/derekr/pd-rockets) by derekr, where it is `pd-kanban-board`: the same code, with its names in Starbase's `sb-` prefix.

A board of lanes whose cards the server renders and places. Drag a card within its lane or into another one, or focus it and hold Alt while you press the arrow keys. The component shows where the card would land, then emits `sb-kanban-move` with the card, the lane and the card it goes before. It moves nothing itself: the server applies the move and sends the board back, and the morph puts each card in its place with a short animation.

## Examples

### Plan a mission

The server renders the board with its arrangement in `data-state`: each lane's cards, lanes separated by `|`. A move sends that arrangement with the event's detail to `/demo/arrange/kanban-board`, which answers with the board as it is after the move. Nothing is stored: try it in two tabs.

```html preview
<style>
  .demo-kanban { display: grid; grid-template-columns: repeat(auto-fit, minmax(9rem, 1fr)); gap: 10px; inline-size: 100%; }
  .demo-kanban [data-kanban-lane] {
    display: grid;
    grid-template-rows: auto 1fr;
    gap: 8px;
    padding: 8px;
    border: 1px solid var(--sb-border);
    background: var(--sb-surface-inset);
  }
  .demo-kanban [data-kanban-lane][data-drop-active] { border-color: var(--sb-brand); }
  .demo-kanban__title { margin: 0; color: var(--sb-text-2); font-size: 0.75rem; font-weight: 700; text-transform: uppercase; letter-spacing: 0.05em; }
  .demo-kanban [data-kanban-lane-cards] { display: grid; align-content: start; gap: 6px; min-block-size: 3rem; }
  .demo-kanban [data-kanban-card], [data-drag-preview][data-kanban-card] {
    padding: 0.5rem 0.75rem;
    border: 1px solid var(--sb-border);
    background: var(--sb-surface-card);
    color: var(--sb-text-1);
    cursor: grab;
    touch-action: none;
    user-select: none;
  }
  .demo-kanban [data-kanban-card] { position: relative; }
  .demo-kanban [data-kanban-card]:focus-visible { outline: 2px solid var(--sb-brand-light); outline-offset: 2px; }
  .demo-kanban [data-dragging] { opacity: 0.35; }
  .demo-kanban[data-key-staging] [data-kanban-card]:focus { border-color: var(--sb-brand); }
  .demo-kanban [data-drop-before]::before, .demo-kanban [data-drop-end]::after {
    content: "";
    display: block;
    block-size: 3px;
    background: var(--sb-brand);
  }
  .demo-kanban [data-drop-before]::before { position: absolute; inset: -5px 0 auto; }
  [data-drag-preview][data-kanban-card] { box-shadow: var(--sb-shadow-overlay, 0 8px 16px rgb(0 0 0 / 0.35)); cursor: grabbing; }
</style>
<sb-kanban-board id="mission-board" class="demo-kanban" data-state="mars jupiter neptune | europa titan | moon"
	data-on:sb-kanban-move="@get('/demo/arrange/kanban-board', {payload: {id: el.id, state: el.dataset.state, move: evt.detail}})">
	<section data-kanban-lane data-col="0" aria-label="To visit">
		<p class="demo-kanban__title">To visit</p>
		<div data-kanban-lane-cards>
			<article data-kanban-card="mars" tabindex="0">🪐 Mars</article>
			<article data-kanban-card="jupiter" tabindex="0">🪐 Jupiter</article>
			<article data-kanban-card="neptune" tabindex="0">🪐 Neptune</article>
		</div>
	</section>
	<section data-kanban-lane data-col="1" aria-label="En route">
		<p class="demo-kanban__title">En route</p>
		<div data-kanban-lane-cards>
			<article data-kanban-card="europa" tabindex="0">🌑 Europa</article>
			<article data-kanban-card="titan" tabindex="0">🌑 Titan</article>
		</div>
	</section>
	<section data-kanban-lane data-col="2" aria-label="Visited">
		<p class="demo-kanban__title">Visited</p>
		<div data-kanban-lane-cards>
			<article data-kanban-card="moon" tabindex="0">🌑 Moon</article>
		</div>
	</section>
</sb-kanban-board>
```

## Markup and events

A lane is an element with `data-kanban-lane` and its column number in `data-col`; its cards sit in an element with `data-kanban-lane-cards`, which is also where a card dropped at the end of the lane goes. A card is an element with `data-kanban-card="<id>"` and `tabindex="0"`. Its content is yours: a button or link inside a card keeps its own clicks and keys, unless it carries `data-kanban-card-main`, which makes it the card's handle for dragging and the keyboard.

| Event | Detail | When |
|---|---|---|
| `sb-kanban-move` | `{ cardId, col, before }` | A card was dropped, or a keyboard move committed. `col` is the lane's `data-col` as a number, and `before` the id of the card it now precedes in that lane, or `""` for the end. |
| `sb-kanban-select` | `{ cardId }` | The keyboard moved the focus to another card. |

Both events bubble. When boards are nested, check `evt.target` in a handler that several of them reach.

## Keyboard

| Keys | Action |
|---|---|
| Arrow Down / Arrow Up, or j / k | Focus the next / previous card, across lanes in board order |
| Arrow Left / Arrow Right, or h / l | Focus the card in the same row of the previous / next lane |
| Home / End | Focus the first / last card on the board |
| Alt + Arrow Down / Up, or Alt + j / k | Move the focused card down / up in its lane |
| Alt + Arrow Left / Right, or Alt + h / l | Move the focused card to the end of the previous / next lane |
| Escape | Cancel a move before Alt is released |

Releasing Alt sends the move. On macOS, Ctrl + n and Ctrl + p also move the focus. Each action takes its keys from a `data-key-<action>` attribute on the board, a space-separated list such as `data-key-focus-next="ArrowDown n"`; an empty value turns the action off. The actions are `focus-next`, `focus-previous`, `focus-left`, `focus-right`, `focus-first`, `focus-last`, `move-up`, `move-down`, `move-left`, `move-right` and `cancel`. The older names `select-next`, `select-previous`, `select-left` and `select-right` still work when the `focus-` attribute is absent.

## On the server

The server owns the board. A handler applies the move and renders the board again; this is the demo's, which keeps the arrangement in the markup instead of a database:

```go source=internal/web/demo_arrange_kanban.go#arrangeKanban,renderKanban
// arrangeKanban applies an sb-kanban-move ({cardId, col, before}): the card
// leaves its lane and goes into lane col, before another card ("": last).
func arrangeKanban(state string, move json.RawMessage) (string, error) {
	lanes, err := kanbanState(state)
	if err != nil {
		return "", err
	}
	var m struct {
		CardID string `json:"cardId"`
		Col    int    `json:"col"`
		Before string `json:"before"`
	}
	if err := json.Unmarshal(move, &m); err != nil {
		return "", err
	}
	if m.Col < 0 || m.Col >= len(lanes) {
		return "", fmt.Errorf("no lane %d", m.Col)
	}
	from := -1
	for i, lane := range lanes {
		for j, id := range lane {
			if id == m.CardID {
				from = i
				lanes[i] = append(lane[:j:j], lane[j+1:]...)
				break
			}
		}
	}
	if from < 0 {
		return "", fmt.Errorf("no card %q", m.CardID)
	}
	// In at the end, then before its neighbour: moveBefore checks it.
	if lanes[m.Col], err = moveBefore(append(lanes[m.Col], m.CardID), m.CardID, m.Before); err != nil {
		return "", err
	}
	out := make([]string, len(lanes))
	for i, lane := range lanes {
		out[i] = strings.Join(lane, " ")
	}
	return strings.Join(out, " | "), nil
}

// renderKanban is the board's markup: the host the morph replaces, with the
// arrangement in data-state, a lane per column and a card per body.
func renderKanban(id, state string) string {
	lanes, _ := kanbanState(state)
	var b strings.Builder
	fmt.Fprintf(&b, "<sb-kanban-board id=\"%s\" class=\"demo-kanban\" data-state=\"%s\"\n\tdata-on:sb-kanban-move=\"%s\">\n", id, state, arrangeOn("kanban-board"))
	for i, title := range kanbanLanes {
		fmt.Fprintf(&b, "\t<section data-kanban-lane data-col=\"%d\" aria-label=\"%s\">\n\t\t<p class=\"demo-kanban__title\">%s</p>\n\t\t<div data-kanban-lane-cards>\n", i, title, title)
		if i < len(lanes) {
			for _, card := range lanes[i] {
				fmt.Fprintf(&b, "\t\t\t<article data-kanban-card=\"%s\" tabindex=\"0\">%s</article>\n", card, label(card))
			}
		}
		b.WriteString("\t\t</div>\n\t</section>\n")
	}
	b.WriteString("</sb-kanban-board>")
	return b.String()
}
```

## Styling

The component adds no styles: the lanes and cards are your page's markup, styled by your page's CSS. While a card moves, it marks the elements involved:

- `data-drag-active` on the board and `data-dragging` on the card while you drag.
- `data-drop-active` on the lane the card would land in, `data-drop-before` on the card it would land before, and `data-drop-end` on the lane's `data-kanban-lane-cards` element when it would land at the end.
- `data-key-staging` on the board while a keyboard move waits for Alt to be released.
- The card under the pointer is a copy of the card in `<body>`, marked `data-drag-preview`, with the original's size in `--sb-source-width` and `--sb-source-height`. A `<template data-sb-preview>` inside a card replaces the copy.
- A `<template data-sb-target="before">` (or `"end"`) inside the board is copied into the landing place as a drop marker, marked `data-sb-target-indicator`.

The example above shows one way to draw them.

## Accessibility

Lanes and cards are elements of your page, so their roles and names are yours: give each lane an `aria-label`, as the example does. Keyboard moves need no pointer. The component announces nothing itself: when the server sends the new board, the moved card keeps the focus.

## Licence

The code is PD rockets', under its Beer-Ware licence, in `LICENSE-pd-rockets.txt` next to it. `go run ./cmd/vendorpd` brings in a newer release.
