---
name: Sortable List
tag: sb-sortable-list
category: data
summary: Reorder a list by dragging or with Alt and the arrow keys. The server applies each move.
author: derekr
license: Beerware
source: https://github.com/derekr/pd-rockets/tree/4f4111722763d2aa8f5851c397af026fcbe97e4f/rocket/sortable-list
tags: [drag and drop, sortable, reorder, list, keyboard, pd rockets]
since: 2026-10-06
preview: |
  <style>
    .demo-sortable { display: grid; gap: 6px; inline-size: min(100%, 12rem); }
    .demo-sortable [data-sortable-item], [data-drag-preview][data-sortable-item] { padding: 0.4rem 0.7rem; border: 1px solid var(--sb-border); background: var(--sb-surface-card); color: var(--sb-text-1); cursor: grab; touch-action: none; user-select: none; }
    .demo-sortable [data-sortable-item] { position: relative; }
    .demo-sortable [data-sortable-item]:focus-visible { outline: 2px solid var(--sb-brand-light); outline-offset: 2px; }
    .demo-sortable [data-dragging] { opacity: 0.35; }
    .demo-sortable[data-key-staging] [data-sortable-item]:focus { border-color: var(--sb-brand); }
    .demo-sortable [data-drop-before]::before, .demo-sortable[data-drop-end]::after { content: ""; display: block; block-size: 3px; background: var(--sb-brand); }
    .demo-sortable [data-drop-before]::before { position: absolute; inset: -5px 0 auto; }
    [data-drag-preview][data-sortable-item] { box-shadow: var(--sb-shadow-overlay, 0 8px 16px rgb(0 0 0 / 0.35)); cursor: grabbing; }
    @media (forced-colors: active) {
      .demo-sortable [data-sortable-item]:focus-visible { outline-color: Highlight; }
      .demo-sortable [data-drop-before]::before, .demo-sortable[data-drop-end]::after { forced-color-adjust: none; background: Highlight; }
      .demo-sortable [data-dragging] { opacity: 1; border-style: dashed; }
      .demo-sortable[data-key-staging] [data-sortable-item]:focus, [data-drag-preview][data-sortable-item] { border-color: Highlight; }
    }
  </style>
  <p id="sortable-list-card-hint" class="visually-hidden">Drag a planet, or focus it and press the arrow keys while you hold Alt.</p>
  <sb-sortable-list id="sortable-list-card" data-ignore-morph class="demo-sortable" role="list" data-on:sb-sortable-move="@get('/demo/arrange/sortable-list', {payload: {id: el.id, state: el.dataset.state, move: evt.detail}})" data-state="earth mars venus">
    <div data-sortable-item="earth" role="listitem" tabindex="0" aria-describedby="sortable-list-card-hint">🪐 Earth</div>
    <div data-sortable-item="mars" role="listitem" tabindex="0" aria-describedby="sortable-list-card-hint">🪐 Mars</div>
    <div data-sortable-item="venus" role="listitem" tabindex="0" aria-describedby="sortable-list-card-hint">🪐 Venus</div>
  </sb-sortable-list>
usage: |
  <sb-sortable-list role="list" data-on:sb-sortable-move="@post('/list/move', {payload: evt.detail})">
    <div data-sortable-item="a" role="listitem" tabindex="0">First</div>
    <div data-sortable-item="b" role="listitem" tabindex="0">Second</div>
  </sb-sortable-list>
---

From [PD rockets](https://github.com/derekr/pd-rockets) by derekr, where it is `pd-sortable-list`, with its names in Starbase's `sb-` prefix. It carries the patches in `patches/pd-rockets`, which for this list take a drop just above or below it, skip the move animation when the reader prefers reduced motion, keep nesting working when surfaces come in separate bundles, and declare its event in the manifest.

A list whose items the server renders and orders. Drag an item, or focus it and hold Alt while you press the arrow keys. The component marks where the item would land, then emits `sb-sortable-move` with the item and the one it goes before. It moves nothing itself: the server applies the move and sends the list back in its new order, and the morph puts each item in its place with a short animation.

## Examples

### Put the inner planets in order

The server renders the list with its order in `data-state`. A move sends that order with the event's detail to `/demo/arrange/sortable-list`, which answers with the list in the new order. Nothing is stored: try it in two tabs.

```html preview
<style>
  .demo-sortable-box { display: grid; gap: 0.75rem; inline-size: min(100%, 16rem); }
  .demo-sortable-box p { margin: 0; color: var(--sb-text-muted); font-size: 0.875rem; }
  .demo-sortable { display: grid; gap: 6px; }
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
  @media (forced-colors: active) {
    .demo-sortable [data-sortable-item]:focus-visible { outline-color: Highlight; }
    .demo-sortable [data-drop-before]::before, .demo-sortable[data-drop-end]::after { forced-color-adjust: none; background: Highlight; }
    .demo-sortable [data-dragging] { opacity: 1; border-style: dashed; }
    .demo-sortable[data-key-staging] [data-sortable-item]:focus, [data-drag-preview][data-sortable-item] { border-color: Highlight; }
  }
</style>
<div class="demo-sortable-box">
<p id="inner-planets-hint">Drag a planet, or focus it and press the arrow keys while you hold Alt. Releasing Alt sends the move.</p>
<sb-sortable-list id="inner-planets" data-ignore-morph class="demo-sortable" role="list" data-state="earth mercury mars venus"
	data-on:sb-sortable-move="@get('/demo/arrange/sortable-list', {payload: {id: el.id, state: el.dataset.state, move: evt.detail}})">
	<div data-sortable-item="earth" role="listitem" tabindex="0" aria-describedby="inner-planets-hint">🪐 Earth</div>
	<div data-sortable-item="mercury" role="listitem" tabindex="0" aria-describedby="inner-planets-hint">🪐 Mercury</div>
	<div data-sortable-item="mars" role="listitem" tabindex="0" aria-describedby="inner-planets-hint">🪐 Mars</div>
	<div data-sortable-item="venus" role="listitem" tabindex="0" aria-describedby="inner-planets-hint">🪐 Venus</div>
</sb-sortable-list>
</div>
```

## Markup and events

Every element with `data-sortable-item="<id>"` inside the list is an item; the id is what the event reports. Items can hold anything, and they need `tabindex="0"` to take the keyboard focus. A list inside an item (a nested `sb-sortable-list`, or a drag group) handles its own items.

When an item is dropped, or a keyboard move commits, the list emits `sb-sortable-move` with `{ itemId, before }`: `before` is the id of the item it now precedes, or `""` for the end. A drop counts over the list and up to an item's height above or below it, so a drop just under the last item moves it to the end. The event bubbles: when lists are nested, check `evt.target` in a handler that several of them reach.

## Keyboard

| Keys | Action |
|---|---|
| Arrow Down / Arrow Up, or j / k | Focus the next / previous item |
| Home / End | Focus the first / last item |
| Alt + Arrow Down / Up, or Alt + j / k | Move the focused item |
| Escape | Cancel the move |

A keyboard move waits while you hold Alt, so several presses make one move. Releasing Alt sends it, and so does leaving the window. Escape, a click in the list or moving the focus to another element cancels it.

On macOS, Ctrl + n and Ctrl + p also move the focus. Each action takes its keys from a `data-key-<action>` attribute on the list, a space-separated list such as `data-key-focus-next="ArrowDown n"`. A key takes its modifiers with `+`: `Alt+k`, `Ctrl+n`, `Shift+ArrowDown`, `Meta+j` (or `Cmd+j`). A move bound with a modifier waits for that modifier's release, one without for the key's. An empty value turns the action off. The actions are `focus-next`, `focus-previous`, `focus-first`, `focus-last`, `move-up`, `move-down` and `cancel`.

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
// with the arrangement in data-state and an item per body. Every item points
// at the page's keyboard hint, the element with the id <id>-hint.
func renderSortableList(id, state string) string {
	id, state = html.EscapeString(id), html.EscapeString(state)
	var b strings.Builder
	fmt.Fprintf(&b, "<sb-sortable-list id=\"%s\" data-ignore-morph class=\"demo-sortable\" role=\"list\" data-state=\"%s\"\n\tdata-on:sb-sortable-move=\"%s\">\n", id, state, arrangeOn("sortable-list"))
	for _, item := range strings.Fields(state) {
		fmt.Fprintf(&b, "\t<div data-sortable-item=\"%s\" role=\"listitem\" tabindex=\"0\" aria-describedby=\"%s-hint\">%s</div>\n", item, id, label(item))
	}
	b.WriteString("</sb-sortable-list>")
	return b.String()
}
```

`moveBefore` is the move itself, shared by every sortable demo. It refuses an item that isn't in the list and a `before` that isn't either:

```go source=internal/web/demo_arrange.go#moveBefore
// moveBefore takes item out of list and puts it back before another
// ("": at the end), as every sortable surface's move does.
func moveBefore(list []string, item, before string) ([]string, error) {
	out := make([]string, 0, len(list))
	found := false
	for _, id := range list {
		if id == item {
			found = true
		} else {
			out = append(out, id)
		}
	}
	if !found || item == before {
		return nil, fmt.Errorf("can't move %q before %q", item, before)
	}
	at := len(out)
	for i, id := range out {
		if id == before {
			at = i
		}
	}
	if before != "" && at == len(out) {
		return nil, fmt.Errorf("%q is not in the list", before)
	}
	return append(out[:at], append([]string{item}, out[at:]...)...), nil
}
```

Before an arranger runs, the handler (`demoArrange`) refuses an element id that isn't a plain identifier, and `ids` refuses a state with anything but bodies from the example dataset. The render escapes what it writes all the same.

## Styling

The list and its items are your page's markup, styled by your page's CSS. Give items `touch-action: none`, so a touch drag moves the item instead of scrolling the page, and `user-select: none`, so a mouse drag doesn't select their text. While an item moves, the component marks the elements involved:

- `data-drag-active` on the list and `data-dragging` on the item while you drag.
- `data-drop-before` on the item the moving one would land before, and `data-drop-end` on the list when it would land at the end.
- `data-key-staging` on the list while a keyboard move waits for Alt to be released.
- A copy of the item follows the pointer, appended to `<body>` and marked `data-drag-preview`, with the original's size in `--sb-source-width` and `--sb-source-height`. A `<template data-sb-preview>` that is a direct child of an item replaces the copy.
- A `<template data-sb-target="before">` (or `"end"`, or `""` for both) that is a direct child of the list is copied into the landing place as a drop marker, marked `data-sb-target-indicator`: into the item the moving one would land before, or into the list for the end.

The component sets inline styles only on what it adds. The copy is `position: fixed` at the pointer, above the page (`z-index: 1000`), and takes the original's size unless a template replaces it. The marker is `position: absolute`, so the item or list it lands in needs a positioning context. Neither takes pointer events. The example above shows one way to draw the rest, with a `forced-colors` block that keeps the drop line, the moving item and the focus visible in high contrast.

## Accessibility

Items are focusable elements of your page, so their roles and names are yours. The example gives the list `role="list"` and each item `role="listitem"`, so assistive technology sees a list of items instead of loose text, and points every item at a hint with `aria-describedby`, which tells keyboard users about Alt and the arrow keys. Keyboard moves need no pointer. The component announces nothing itself: when the server sends the new order, the moved item keeps the focus. To announce the result, have the server render it into a live region next to the list.

## Licence

The code is PD rockets', under its [Beer-Ware licence](https://github.com/derekr/pd-rockets/blob/v2026-09-28-2/LICENSE), in `LICENSE-pd-rockets.txt` next to it.
