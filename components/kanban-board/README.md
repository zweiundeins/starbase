---
name: Kanban Board
tag: sb-kanban-board
category: data
summary: Lanes of cards to drag, or to move with Alt and the arrows. The server applies each move.
author: derekr
license: Beerware
source: https://github.com/derekr/pd-rockets/tree/4f4111722763d2aa8f5851c397af026fcbe97e4f/rocket/kanban
tags: [drag and drop, kanban, board, lanes, cards, keyboard, pd rockets]
since: 2026-10-06
preview: |
  <style>
    .demo-kanban-card { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 6px; inline-size: 100%; }
    .demo-kanban-card [data-kanban-lane] { display: grid; padding: 4px; border: 1px solid var(--sb-border); background: var(--sb-surface-inset); }
    .demo-kanban-card [data-kanban-lane][data-drop-active] { border-color: var(--sb-brand); }
    .demo-kanban-card [data-kanban-lane-cards] { display: grid; align-content: start; gap: 4px; min-block-size: 3.5rem; }
    .demo-kanban-card [data-kanban-card], [data-drag-preview][data-kanban-card] { position: relative; padding: 0.25rem 0.4rem; border: 1px solid var(--sb-border); background: var(--sb-surface-card); color: var(--sb-text-1); font-size: 0.75rem; white-space: nowrap; cursor: grab; touch-action: none; user-select: none; }
    .demo-kanban-card [data-kanban-card]:focus-visible, .demo-kanban-card [data-kanban-lane]:focus-visible { outline: 2px solid var(--sb-brand-light); outline-offset: 1px; }
    .demo-kanban-card [data-dragging] { opacity: 0.35; }
    .demo-kanban-card [data-drop-before]::before, .demo-kanban-card [data-drop-end]::after { content: ""; display: block; block-size: 2px; background: var(--sb-brand); }
    .demo-kanban-card [data-drop-before]::before { position: absolute; inset: -4px 0 auto; }
    @media (forced-colors: active) {
      .demo-kanban-card [data-kanban-lane][data-drop-active] { outline: 2px solid Highlight; outline-offset: -2px; }
      .demo-kanban-card [data-kanban-card]:focus-visible, .demo-kanban-card [data-kanban-lane]:focus-visible { outline-color: Highlight; }
      .demo-kanban-card [data-drop-before]::before, .demo-kanban-card [data-drop-end]::after { forced-color-adjust: none; background: Highlight; }
      .demo-kanban-card [data-dragging] { border: 1px dashed CanvasText; }
    }
  </style>
  <sb-kanban-board id="kanban-board-card" data-ignore-morph class="demo-kanban-card" data-state="4: mars io | 7: moon | 9:" data-on:sb-kanban-move="@get('/demo/arrange/kanban-board-card', {payload: {id: el.id, state: el.dataset.state, move: evt.detail}})">
    <section data-kanban-lane data-col="4" tabindex="-1" aria-label="To visit"><div data-kanban-lane-cards><article id="kanban-board-card-mars" data-kanban-card="mars" tabindex="0">Mars</article><article id="kanban-board-card-io" data-kanban-card="io" tabindex="-1">Io</article></div></section>
    <section data-kanban-lane data-col="7" tabindex="-1" aria-label="En route"><div data-kanban-lane-cards><article id="kanban-board-card-moon" data-kanban-card="moon" tabindex="-1">Moon</article></div></section>
    <section data-kanban-lane data-col="9" tabindex="-1" aria-label="Visited"><div data-kanban-lane-cards></div></section>
  </sb-kanban-board>
usage: |
  <sb-kanban-board data-on:sb-kanban-move="@post('/board/move', {payload: evt.detail, requestCancellation: 'disabled'})">
    <section data-kanban-lane data-col="1" tabindex="-1" aria-label="To do">
      <div data-kanban-lane-cards>
        <article id="card-a" data-kanban-card="a" tabindex="0">First</article>
      </div>
    </section>
    <section data-kanban-lane data-col="2" tabindex="-1" aria-label="Done">
      <div data-kanban-lane-cards></div>
    </section>
  </sb-kanban-board>
---

From [PD rockets](https://github.com/derekr/pd-rockets) by derekr, where it is `pd-kanban-board`, with its names in Starbase's `sb-` prefix and the patches in `patches/pd-rockets` (0002 to 0005 for every surface, 0030 to 0039 for this one). For this board they add lane moves, let a card be dragged by a link or an image, keep Alt and the arrows from leaving the page at the first and last lane, keep a card's row when it changes lanes, add Alt + Home and Alt + End, let the arrows reach empty lanes, send `sb-kanban-select` on a click too, let a board that is one Tab stop move it with the focus, and list the events in the component's manifest.

A board of lanes whose cards the server renders and places. Drag a card within its lane or into another one, or focus it and hold Alt while you press the arrow keys. The component shows where the card would land, then emits `sb-kanban-move` with the card, the lane and the card it goes before. It moves nothing itself: the server applies the move and sends the board back, and the morph moves each card to its place with a short animation (none when the reader's system asks for reduced motion).

## Examples

### Plan a mission

The server renders the board with its arrangement in `data-state`: the lanes in their order, separated by `|`, each its id and its cards. A move sends that arrangement with the event's detail to `/demo/arrange/kanban-board`, which answers with the board as it is after the move. Nothing is stored: try it in two tabs. Lanes move too: drag one by the ↔ grip next to its title, or focus the grip and use Alt and the arrows.

Since each request carries the arrangement the page had when it left, two moves made faster than the server answers both start from the same board, and Datastar cancels the first request when the second one starts: the board then shows the second move only. A server that keeps the board applies both.

```html preview
<style>
  .demo-kanban { display: grid; grid-template-columns: repeat(auto-fit, minmax(9rem, 1fr)); gap: 10px; inline-size: 100%; }
  .demo-kanban [data-kanban-lane], [data-drag-preview][data-kanban-lane] {
    display: grid;
    grid-template-rows: auto 1fr;
    gap: 8px;
    padding: 8px;
    border: 1px solid var(--sb-border);
    background: var(--sb-surface-inset);
  }
  .demo-kanban [data-kanban-lane][data-drop-active] { border-color: var(--sb-brand); }
  .demo-kanban__head { display: flex; align-items: center; gap: 4px; }
  .demo-kanban__grip { padding: 0 4px; border: 0; background: none; color: var(--sb-text-muted); font: inherit; cursor: grab; touch-action: none; }
  .demo-kanban__title { margin: 0; color: var(--sb-text-2); font-size: 0.75rem; font-weight: 700; text-transform: uppercase; letter-spacing: 0.05em; }
  .demo-kanban [data-kanban-lane-cards], [data-drag-preview] [data-kanban-lane-cards] { display: grid; align-content: start; gap: 6px; min-block-size: 3rem; }
  .demo-kanban [data-kanban-card], [data-drag-preview][data-kanban-card], [data-drag-preview] [data-kanban-card] {
    padding: 0.5rem 0.75rem;
    border: 1px solid var(--sb-border);
    background: var(--sb-surface-card);
    color: var(--sb-text-1);
    cursor: grab;
    touch-action: none;
    user-select: none;
  }
  .demo-kanban [data-kanban-card] { position: relative; }
  .demo-kanban [data-kanban-card]:focus-visible, .demo-kanban [data-kanban-lane]:focus-visible, .demo-kanban__grip:focus-visible { outline: 2px solid var(--sb-brand-light); outline-offset: 2px; }
  .demo-kanban [data-dragging], .demo-kanban [data-lane-dragging] { opacity: 0.35; }
  .demo-kanban [data-lane-drop-target] { outline: 2px dashed var(--sb-brand); outline-offset: 3px; }
  .demo-kanban[data-key-staging] [data-kanban-card]:focus { border-color: var(--sb-brand); }
  .demo-kanban [data-drop-before]::before, .demo-kanban [data-drop-end]::after {
    content: "";
    display: block;
    block-size: 3px;
    background: var(--sb-brand);
  }
  .demo-kanban [data-drop-before]::before { position: absolute; inset: -5px 0 auto; }
  [data-drag-preview][data-kanban-card], [data-drag-preview][data-kanban-lane] { box-shadow: var(--sb-shadow-overlay, 0 8px 16px rgb(0 0 0 / 0.35)); cursor: grabbing; }
  @media (forced-colors: active) {
    .demo-kanban [data-kanban-lane][data-drop-active] { outline: 2px solid Highlight; outline-offset: -2px; }
    .demo-kanban[data-key-staging] [data-kanban-card]:focus { border-color: Highlight; }
    .demo-kanban [data-kanban-card]:focus-visible, .demo-kanban [data-kanban-lane]:focus-visible, .demo-kanban__grip:focus-visible { outline-color: Highlight; }
    .demo-kanban [data-lane-drop-target] { outline-color: Highlight; }
    .demo-kanban [data-drop-before]::before, .demo-kanban [data-drop-end]::after { forced-color-adjust: none; background: Highlight; }
    .demo-kanban [data-dragging], .demo-kanban [data-lane-dragging] { border: 1px dashed CanvasText; }
    [data-drag-preview][data-kanban-card], [data-drag-preview][data-kanban-lane] { border: 2px solid Highlight; }
  }
</style>
<sb-kanban-board id="mission-board" data-ignore-morph class="demo-kanban" data-state="4: mars jupiter neptune | 7: europa titan | 9: moon"
	data-on:sb-kanban-move="@get('/demo/arrange/kanban-board', {payload: {id: el.id, state: el.dataset.state, move: evt.detail}})"
	data-on:sb-kanban-lane-move="@get('/demo/arrange/kanban-board', {payload: {id: el.id, state: el.dataset.state, move: evt.detail}})">
	<section id="mission-board-lane-4" data-kanban-lane data-col="4" tabindex="-1" aria-label="To visit">
		<div class="demo-kanban__head">
			<button type="button" class="demo-kanban__grip" data-kanban-lane-grip aria-label="Move the To visit lane">↔</button>
			<p class="demo-kanban__title">To visit</p>
		</div>
		<div data-kanban-lane-cards>
			<article id="mission-board-mars" data-kanban-card="mars" tabindex="0">🪐 Mars</article>
			<article id="mission-board-jupiter" data-kanban-card="jupiter" tabindex="0">🪐 Jupiter</article>
			<article id="mission-board-neptune" data-kanban-card="neptune" tabindex="0">🪐 Neptune</article>
		</div>
	</section>
	<section id="mission-board-lane-7" data-kanban-lane data-col="7" tabindex="-1" aria-label="En route">
		<div class="demo-kanban__head">
			<button type="button" class="demo-kanban__grip" data-kanban-lane-grip aria-label="Move the En route lane">↔</button>
			<p class="demo-kanban__title">En route</p>
		</div>
		<div data-kanban-lane-cards>
			<article id="mission-board-europa" data-kanban-card="europa" tabindex="0">🌑 Europa</article>
			<article id="mission-board-titan" data-kanban-card="titan" tabindex="0">🌑 Titan</article>
		</div>
	</section>
	<section id="mission-board-lane-9" data-kanban-lane data-col="9" tabindex="-1" aria-label="Visited">
		<div class="demo-kanban__head">
			<button type="button" class="demo-kanban__grip" data-kanban-lane-grip aria-label="Move the Visited lane">↔</button>
			<p class="demo-kanban__title">Visited</p>
		</div>
		<div data-kanban-lane-cards>
			<article id="mission-board-moon" data-kanban-card="moon" tabindex="0">🌑 Moon</article>
		</div>
	</section>
</sb-kanban-board>
```

## Markup and events

A lane is an element with `data-kanban-lane` and the lane's id, a number, in `data-col` (4, 7 and 9 in the example): an id, not a position, so it stays with the lane when lanes are added or reordered. Its cards sit in an element with `data-kanban-lane-cards`, which is also where a card dropped at the end of the lane goes. A card is an element with `data-kanban-card="<id>"` and `tabindex="0"`, and an `id` that is unique on the page, so that the morph moves the card's element instead of writing another card into it. The card's content is yours: a button or link inside a card keeps its own clicks and keys, unless it carries `data-kanban-card-main`, which makes it the card's handle for dragging and the keyboard. The handle can be a link: a press that drags the card cancels the browser's own drag of the link (or of an image in the card), and a click still follows the link.

| Event | Detail | When |
|---|---|---|
| `sb-kanban-move` | `{ cardId, col, before }` | A card was dropped, or a keyboard move committed. `col` is the lane's `data-col` as a number, and `before` the id of the card it now precedes in that lane, or `""` for the end. |
| `sb-kanban-select` | `{ cardId }` | The arrow keys moved the focus to another card, or the pointer pressed a card (not a button or link inside it). Focusing an empty lane sends nothing. |
| `sb-kanban-lane-move` | `{ col, before }` | A lane was dropped by its grip, moved with Alt and the arrows, or stepped (see [Moving lanes](#moving-lanes)). `col` is its `data-col` as a number, and `before` the `data-col` of the lane it now precedes, or `""` for the end. |

The events bubble. When boards are nested, check `evt.target` in a handler that several of them reach.

## Keyboard

| Keys | Action |
|---|---|
| Arrow Down / Arrow Up, or j / k | Focus the next / previous card, across lanes in board order |
| Arrow Left / Arrow Right, or h / l | Focus the card in the same row of the previous / next lane |
| Home / End | Focus the first / last card on the board |
| Alt + Arrow Down / Up, or Alt + j / k | Move the focused card down / up in its lane |
| Alt + Arrow Left / Right, or Alt + h / l | Move the focused card into the previous / next lane, in the same row |
| Alt + Home / End | Move the focused card to the top / bottom of its lane |
| Alt + Arrow Left / Right on a lane's grip | Move the lane one place left / right |
| Alt + Home / End on a lane's grip | Move the lane to the first / last place |
| Escape | Cancel a move before Alt is released, or a lane drag |

Releasing Alt sends the move, and until then each key moves the card or lane on from where the last one put it. At the first or last lane, Alt + Arrow Left or Right does nothing, so the browser doesn't go back or forward in its history either.

A lane with `tabindex="-1"`, as in the example, takes the focus while it has no cards: the arrows stop there, so a keyboard user learns that the lane exists. Without the attribute, the arrows pass over an empty lane. Alt and the arrows move a card into an empty lane either way, and so does a drag.

With `tabindex="0"` on every card, as in the example, each card is a Tab stop. For a board that is one Tab stop, render one card with `tabindex="0"` and the others with `tabindex="-1"`, as the gallery card does. The stop then follows the focus to whichever card or empty lane the arrows reach, so Tab and Shift+Tab leave the board from there. The server can render the same values every time: when a morph puts them back, the board moves the stop to the focus again before Tab acts.

On macOS, Ctrl + n and Ctrl + p also move the focus. Each action takes its keys from a `data-key-<action>` attribute on the board, a space-separated list such as `data-key-focus-next="ArrowDown n"`; an empty value turns the action off. The actions are `focus-next`, `focus-previous`, `focus-left`, `focus-right`, `focus-first`, `focus-last`, `move-up`, `move-down`, `move-left`, `move-right`, `move-first`, `move-last` and `cancel`. The older names `select-next`, `select-previous`, `select-left` and `select-right` still work when the `focus-` attribute is absent.

## On the server

The server owns the board. A handler applies the move and renders the board again. Since `col` names the lane by its id, a move lands in the right lane even when another user added or reordered lanes in the meantime. A server that stores the board should send the move with `requestCancellation: 'disabled'`, as in the usage example: Datastar otherwise cancels a request when the next one from the same board starts, and a move can be lost. This is the demo's handler, which keeps the arrangement in the markup instead of a database:

```go source=internal/web/demo_arrange_kanban.go#kanbanLane,kanbanLanes,kanbanColumn,arrangeKanban,renderKanban
// A kanbanLane is a lane of the demo board. Its ID is its data-col: a move
// names the lane by id, which stays the same wherever the lane is.
type kanbanLane struct {
	ID    int
	Title string
}

// kanbanLanes are the demo board's lanes.
var kanbanLanes = []kanbanLane{{4, "To visit"}, {7, "En route"}, {9, "Visited"}}

// A kanbanColumn is a lane in a board's arrangement: its id and its cards.
type kanbanColumn struct {
	ID    int
	Cards []string
}

// arrangeKanban applies an sb-kanban-move ({cardId, col, before}): the card
// leaves its lane and goes into the lane whose id is col, before another
// card ("": last). A move without a card moves a lane (arrangeKanbanLane).
func arrangeKanban(state string, move json.RawMessage) (string, error) {
	cols, err := kanbanState(state)
	if err != nil {
		return "", err
	}
	var m struct {
		CardID *string `json:"cardId"`
		Col    int     `json:"col"`
		Before string  `json:"before"`
	}
	if err := json.Unmarshal(move, &m); err != nil {
		return "", err
	}
	if m.CardID == nil {
		return arrangeKanbanLane(cols, m.Col, m.Before)
	}
	to := slices.IndexFunc(cols, func(c kanbanColumn) bool { return c.ID == m.Col })
	if to < 0 {
		return "", fmt.Errorf("no lane %d", m.Col)
	}
	from := -1
	for i, c := range cols {
		if j := slices.Index(c.Cards, *m.CardID); j >= 0 {
			from = i
			cols[i].Cards = slices.Delete(c.Cards, j, j+1)
		}
	}
	if from < 0 {
		return "", fmt.Errorf("no card %q", *m.CardID)
	}
	// In at the end, then before its neighbour: moveBefore checks it.
	if cols[to].Cards, err = moveBefore(append(cols[to].Cards, *m.CardID), *m.CardID, m.Before); err != nil {
		return "", err
	}
	return kanbanFormat(cols), nil
}

// renderKanban is the board's markup: the host the morph replaces, with the
// arrangement in data-state, a lane per column and a card per body. Lanes
// and cards have ids, so the morph moves them (and their focus) instead of
// rewriting one in another's place, and each lane has a grip to drag it by.
func renderKanban(id, state string) string {
	cols, _ := kanbanState(state)
	esc := html.EscapeString
	var b strings.Builder
	on := arrangeOn("kanban-board")
	fmt.Fprintf(&b, "<sb-kanban-board id=\"%s\" data-ignore-morph class=\"demo-kanban\" data-state=\"%s\"\n\tdata-on:sb-kanban-move=\"%s\"\n\tdata-on:sb-kanban-lane-move=\"%s\">\n", esc(id), esc(state), on, on)
	for _, c := range cols {
		title := esc(kanbanTitle(c.ID))
		fmt.Fprintf(&b, "\t<section id=\"%s-lane-%d\" data-kanban-lane data-col=\"%d\" tabindex=\"-1\" aria-label=\"%s\">\n", esc(id), c.ID, c.ID, title)
		fmt.Fprintf(&b, "\t\t<div class=\"demo-kanban__head\">\n\t\t\t<button type=\"button\" class=\"demo-kanban__grip\" data-kanban-lane-grip aria-label=\"Move the %s lane\">↔</button>\n\t\t\t<p class=\"demo-kanban__title\">%s</p>\n\t\t</div>\n\t\t<div data-kanban-lane-cards>\n", title, title)
		for _, card := range c.Cards {
			fmt.Fprintf(&b, "\t\t\t<article id=\"%s-%s\" data-kanban-card=\"%s\" tabindex=\"0\">%s</article>\n", esc(id), esc(card), esc(card), label(card))
		}
		b.WriteString("\t\t</div>\n\t</section>\n")
	}
	b.WriteString("</sb-kanban-board>")
	return b.String()
}
```

## Moving lanes

Lanes move when their markup offers a way, and stay where the server put them otherwise. A grip, an element with `data-kanban-lane-grip` inside the lane and outside its cards, drags the lane, and takes Alt and the arrows while it has the focus. A button with `data-kanban-lane-step="-1"` or `"1"` moves its lane one place left or right. The component emits `sb-kanban-lane-move` and moves nothing itself: the server applies the move, and the morph puts the lanes in their new order with the same short animation as the cards.

```html
<sb-kanban-board
  data-on:sb-kanban-move="@post('/board/move', {payload: evt.detail, requestCancellation: 'disabled'})"
  data-on:sb-kanban-lane-move="@post('/board/lanes', {payload: evt.detail, requestCancellation: 'disabled'})">
  <section id="lane-12" data-kanban-lane data-col="12" tabindex="-1" aria-label="To do">
    <button type="button" data-kanban-lane-grip aria-label="Move the To do lane">↔</button>
    <button type="button" data-kanban-lane-step="-1" aria-label="Move To do left" disabled>◀</button>
    <button type="button" data-kanban-lane-step="1" aria-label="Move To do right">▶</button>
    <div data-kanban-lane-cards>…</div>
  </section>
  …
</sb-kanban-board>
```

`col` and `before` are lane ids, not positions. When someone else added or removed a lane between the render and the drop, the move still names the lanes it meant, and a server refuses an id its board no longer has instead of placing the lane next to the wrong one. Give each lane an `id`, as the example does, so that the morph moves the lane's element and its grip keeps the focus; keyed lanes work on the official Datastar release too, since they hold no Rocket component. The grip needs `touch-action: none`, or a finger on it scrolls the page instead of dragging the lane. Send lane moves with `requestCancellation: 'disabled'`, for the same reason as card moves. The demo applies one to the order of its lanes:

```go source=internal/web/demo_arrange_kanban.go#arrangeKanbanLane
// arrangeKanbanLane applies an sb-kanban-lane-move ({col, before}): the lane
// whose id is col goes before the lane whose id is before ("": last). An id
// the board doesn't have is refused, since the page that sent it is out of
// date: it never lands the lane next to the wrong neighbour.
func arrangeKanbanLane(cols []kanbanColumn, col int, before string) (string, error) {
	order := make([]string, len(cols))
	for i, c := range cols {
		order[i] = strconv.Itoa(c.ID)
	}
	order, err := moveBefore(order, strconv.Itoa(col), before)
	if err != nil {
		return "", fmt.Errorf("no lane move %d before %q: %w", col, before, err)
	}
	moved := make([]kanbanColumn, len(order))
	for i, id := range order {
		moved[i] = cols[slices.IndexFunc(cols, func(c kanbanColumn) bool { return strconv.Itoa(c.ID) == id })]
	}
	return kanbanFormat(moved), nil
}
```

## Styling

The component adds no styles: the lanes and cards are your page's markup, styled by your page's CSS. While a card moves, it marks the elements involved:

- `data-drag-active` on the board and `data-dragging` on the card while you drag.
- `data-drop-active` on the lane the card would land in, `data-drop-before` on the card it would land before, and `data-drop-end` on the lane's `data-kanban-lane-cards` element when it would land at the end.
- `data-key-staging` on the board while a keyboard move waits for Alt to be released.
- `data-lane-dragging` on a lane while it is dragged or its keyboard move waits, and `data-lane-drop-target` on the lane whose place it would take.
- The card under the pointer is a copy of the card in `<body>`, marked `data-drag-preview`, with the original's size in `--sb-source-width` and `--sb-source-height`. A `<template data-sb-preview>` that is a direct child of a card replaces the copy. A dragged lane's copy is marked the same way, without the ids of the lane and its cards, and a direct child `<template data-sb-preview>` of the lane replaces it.
- A `<template data-sb-target="before">` (or `"end"`) that is a direct child of the board is copied into the landing place as a drop marker, marked `data-sb-target-indicator`.

The example above shows one way to draw them.

## Accessibility

Lanes and cards are elements of your page, so their roles and names are yours: give each lane an `aria-label` and each grip a label that names its lane, as the example does. Keyboard moves need no pointer. The component announces nothing itself: when the server sends the new board, the moved card keeps the focus, after a drag too when the cards have ids. In forced colours, the example draws the drop marker, the target lane, the lane a dragged lane would replace and the focus ring in `Highlight`.

## Licence

The code is PD rockets', under its Beer-Ware licence, in `LICENSE-pd-rockets.txt` next to it. `go run ./cmd/vendorpd` brings in a newer release.
