---
name: Sortable List
tag: sb-sortable-list
category: data
summary: Reorder a list by dragging or with Alt and the arrow keys. The server applies each move.
author: derekr
source: https://github.com/derekr/pd-rockets/tree/4f4111722763d2aa8f5851c397af026fcbe97e4f/rocket/sortable-list
tags: [drag and drop, sortable, reorder, list, keyboard, pd rockets]
since: 2026-10-06
preview: |
  <style>
    .demo-sortable { display: grid; gap: 6px; inline-size: 12rem; }
    .demo-sortable [data-sortable-item] { padding: 0.4rem 0.7rem; border: 1px solid var(--sb-border); background: var(--sb-surface-card); color: var(--sb-text-1); }
  </style>
  <sb-sortable-list class="demo-sortable">
    <div data-sortable-item="earth" tabindex="0">🪐 Earth</div>
    <div data-sortable-item="mars" tabindex="0">🪐 Mars</div>
    <div data-sortable-item="venus" tabindex="0">🪐 Venus</div>
  </sb-sortable-list>
usage: |
  <sb-sortable-list data-on:sb-sortable-move="@post('/list/move', {payload: evt.detail})">
    <div data-sortable-item="a" tabindex="0">First</div>
    <div data-sortable-item="b" tabindex="0">Second</div>
  </sb-sortable-list>
---

From [PD rockets](https://github.com/derekr/pd-rockets) by derekr, where it is `pd-sortable-list`: the same code, with its names in Starbase's `sb-` prefix.

A list whose items the server renders and orders. Drag an item, or focus it and hold Alt while you press the arrow keys. The component shows where the item would land, then emits `sb-sortable-move` with the item and the one it goes before. It moves nothing itself: the server applies the move and sends the list back in its new order, and the morph puts each item in its place with a short animation.

## Examples

### Put the inner planets in order

The server renders the list with its order in `data-state`. A move sends that order with the event's detail to `/demo/arrange/sortable-list`, which answers with the list in the new order. Nothing is stored: try it in two tabs.

```html preview
<style>
  .demo-sortable { display: grid; gap: 6px; inline-size: min(100%, 16rem); }
  .demo-sortable [data-sortable-item], [data-drag-preview][data-sortable-item] {
    padding: 0.5rem 0.75rem;
    border: 1px solid var(--sb-border);
    background: var(--sb-surface-card);
    color: var(--sb-text-1);
    cursor: grab;
    touch-action: none;
    user-select: none;
  }
  .demo-sortable [data-sortable-item] { position: relative; }
  .demo-sortable [data-sortable-item]:focus-visible { outline: 2px solid var(--sb-brand-light); outline-offset: 2px; }
  .demo-sortable [data-dragging] { opacity: 0.35; }
  .demo-sortable[data-key-staging] [data-sortable-item]:focus { border-color: var(--sb-brand); }
  .demo-sortable [data-drop-before]::before, .demo-sortable[data-drop-end]::after {
    content: "";
    display: block;
    block-size: 3px;
    background: var(--sb-brand);
  }
  .demo-sortable [data-drop-before]::before { position: absolute; inset: -5px 0 auto; }
  [data-drag-preview][data-sortable-item] { box-shadow: var(--sb-shadow-overlay, 0 8px 16px rgb(0 0 0 / 0.35)); cursor: grabbing; }
</style>
<sb-sortable-list id="inner-planets" class="demo-sortable" data-state="earth mercury mars venus"
	data-on:sb-sortable-move="@get('/demo/arrange/sortable-list', {payload: {id: el.id, state: el.dataset.state, move: evt.detail}})">
	<div data-sortable-item="earth" tabindex="0">🪐 Earth</div>
	<div data-sortable-item="mercury" tabindex="0">🪐 Mercury</div>
	<div data-sortable-item="mars" tabindex="0">🪐 Mars</div>
	<div data-sortable-item="venus" tabindex="0">🪐 Venus</div>
</sb-sortable-list>
```

## Markup and events

Every element with `data-sortable-item="<id>"` inside the list is an item; the id is what the event reports. Items can hold anything, and they need `tabindex="0"` to take the keyboard focus. A list inside an item (a nested `sb-sortable-list`, or a drag group) handles its own items.

| Event | Detail | When |
|---|---|---|
| `sb-sortable-move` | `{ itemId, before }` | An item was dropped, or a keyboard move committed. `before` is the id of the item it now precedes, or `""` for the end. |

The event bubbles. When lists are nested, check `evt.target` in a handler that several of them reach.

## Keyboard

| Keys | Action |
|---|---|
| Arrow Down / Arrow Up, or j / k | Focus the next / previous item |
| Home / End | Focus the first / last item |
| Alt + Arrow Down / Up, or Alt + j / k | Move the focused item; releasing Alt sends the move |
| Escape | Cancel a move before Alt is released |

On macOS, Ctrl + n and Ctrl + p also move the focus. Each action takes its keys from a `data-key-<action>` attribute on the list, a space-separated list such as `data-key-focus-next="ArrowDown n"`; an empty value turns the action off. The actions are `focus-next`, `focus-previous`, `focus-first`, `focus-last`, `move-up`, `move-down` and `cancel`.

## On the server

The server owns the order. A handler applies the move and renders the list again; this is the demo's, which keeps the order in the markup instead of a database:

```go source=internal/web/demo_arrange_list.go#arrangeSortableList,renderSortableList
// arrangeSortableList applies an sb-sortable-move ({itemId, before}) to a
// list of bodies.
func arrangeSortableList(state string, move json.RawMessage) (string, error) {
	list, err := ids(state)
	if err != nil {
		return "", err
	}
	var m struct {
		ItemID string `json:"itemId"`
		Before string `json:"before"`
	}
	if err := json.Unmarshal(move, &m); err != nil {
		return "", err
	}
	if list, err = moveBefore(list, m.ItemID, m.Before); err != nil {
		return "", err
	}
	return strings.Join(list, " "), nil
}

// renderSortableList is the list's markup: the host the morph replaces,
// with the arrangement in data-state and an item per body.
func renderSortableList(id, state string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "<sb-sortable-list id=\"%s\" class=\"demo-sortable\" data-state=\"%s\"\n\tdata-on:sb-sortable-move=\"%s\">\n", id, state, arrangeOn("sortable-list"))
	for _, item := range strings.Fields(state) {
		fmt.Fprintf(&b, "\t<div data-sortable-item=\"%s\" tabindex=\"0\">%s</div>\n", item, label(item))
	}
	b.WriteString("</sb-sortable-list>")
	return b.String()
}
```

## Styling

The component adds no styles: the list and its items are your page's markup, styled by your page's CSS. While an item moves, it marks the elements involved:

- `data-drag-active` on the list and `data-dragging` on the item while you drag.
- `data-drop-before` on the item the moving one would land before, and `data-drop-end` on the list when it would land at the end.
- `data-key-staging` on the list while a keyboard move waits for Alt to be released.
- The item under the pointer is a copy of the item in `<body>`, marked `data-drag-preview`, with the original's size in `--sb-source-width` and `--sb-source-height`. A `<template data-sb-preview>` inside an item replaces the copy.
- A `<template data-sb-target="before">` (or `"end"`) inside the list is copied into the landing place as a drop marker, marked `data-sb-target-indicator`.

The example above shows one way to draw them.

## Accessibility

Items are focusable elements of your page, so their roles and names are yours: a plain list of `<div>`s reads as text. Keyboard moves need no pointer. The component announces nothing itself: when the server sends the new order, the moved item keeps the focus.

## Licence

The code is PD rockets', under its Beer-Ware licence, in `LICENSE-pd-rockets.txt` next to it. `go run ./cmd/vendorpd` brings in a newer release.
