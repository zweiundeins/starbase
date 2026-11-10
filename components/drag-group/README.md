---
name: Drag Group
tag: sb-drag-group
category: data
summary: Move items within and between lists by dragging or with Alt and the arrow keys.
author: derekr
source: https://github.com/derekr/pd-rockets/tree/4f4111722763d2aa8f5851c397af026fcbe97e4f/rocket/drag-group
license: Beerware
tags: [drag and drop, lists, move, reorder, kanban, keyboard, pd rockets]
since: 2026-10-06
preview: |
  <style>
    .demo-group-card { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 8px; inline-size: min(100%, 15rem); }
    .demo-group-card [data-drop-list] { position: relative; display: grid; align-content: start; gap: 4px; padding: 0.35rem; border: 1px dashed var(--sb-border); background: var(--sb-surface-inset); }
    .demo-group-card [data-drop-active] { border-color: var(--sb-brand); }
    .demo-group-card [data-drag-item], [data-drag-preview][data-drag-item] { padding: 0.3rem 0.5rem; border: 1px solid var(--sb-border); background: var(--sb-surface-card); color: var(--sb-text-1); font-size: 0.8125rem; white-space: nowrap; cursor: grab; touch-action: none; user-select: none; }
    .demo-group-card [data-drag-item] { position: relative; }
    .demo-group-card [data-dragging] { opacity: 0.35; }
    .demo-group-card [data-drop-before]::before, .demo-group-card [data-drop-end]::after { content: ""; display: block; block-size: 3px; background: var(--sb-brand); }
    .demo-group-card [data-drop-before]::before { position: absolute; inset: -4px 0 auto; }
    [data-drag-preview][data-drag-item] { box-shadow: var(--sb-shadow-overlay, 0 8px 16px rgb(0 0 0 / 0.35)); cursor: grabbing; }
    @media (forced-colors: active) {
      .demo-group-card [data-drop-active], [data-drag-preview][data-drag-item] { border-color: Highlight; }
      .demo-group-card [data-dragging] { border: 1px dashed CanvasText; }
      .demo-group-card [data-drop-before]::before, .demo-group-card [data-drop-end]::after { background: Highlight; }
    }
  </style>
  <sb-drag-group id="drag-group-card" data-ignore-morph class="demo-group-card" data-state="planets=earth,ceres dwarfs=pluto"
    data-on:sb-drag-group-move="@get('/demo/arrange/drag-group-card', {payload: {id: el.id, state: el.dataset.state, move: evt.detail}})">
    <section data-drop-list="planets">
      <div data-drag-item="earth">🪐 Earth</div>
      <div data-drag-item="ceres">🪨 Ceres</div>
    </section>
    <section data-drop-list="dwarfs">
      <div data-drag-item="pluto">🪨 Pluto</div>
    </section>
  </sb-drag-group>
usage: |
  <sb-drag-group data-on:sb-drag-group-move="@post('/cards/move', {payload: evt.detail})">
    <section data-drop-list="todo">
      <div data-drag-item="a" tabindex="0">Write the docs</div>
    </section>
    <section data-drop-list="done">
      <div data-drag-item="b" tabindex="0">Ship it</div>
    </section>
  </sb-drag-group>
---

From [PD rockets](https://github.com/derekr/pd-rockets) by derekr, where it is `pd-drag-group`, with its names in Starbase's `sb-` prefix. It carries the patches in `patches/pd-rockets`, which for this surface keep Alt and an arrow key on an item from reaching the browser, let the arrow keys reach empty lists, keep an item's place when Alt and the arrows take it to another list, list the move event in the manifest, let it nest in other PD rockets components, and skip the move animation when the reader prefers reduced motion.

Several lists whose items the server renders and arranges. Drag an item within its list or into another one, or focus it and hold Alt while you press the arrow keys. The component shows where the item would land, then emits `sb-drag-group-move` with the item, the list it comes from, the list it goes to and the item it goes before. It moves nothing itself: the server applies the move and sends the lists back, and the morph puts each item in its place with a short animation.

## Examples

### Sort the bodies into planets and dwarf planets

Ceres and Venus start in the wrong list. The server renders both lists with their contents in `data-state`. A move sends that arrangement with the event's detail to `/demo/arrange/drag-group`, which answers with the lists rearranged. Nothing is stored: try it in two tabs.

```html preview
<style>
  .demo-group { display: grid; grid-template-columns: repeat(auto-fit, minmax(10rem, 1fr)); gap: 12px; inline-size: min(100%, 26rem); }
  .demo-group [data-drop-list] {
    position: relative;
    display: grid;
    align-content: start;
    gap: 6px;
    min-block-size: 10rem;
    padding: 0.5rem;
    border: 1px dashed var(--sb-border);
    background: var(--sb-surface-inset);
  }
  .demo-group [data-drop-active] { border-color: var(--sb-brand); }
  .demo-group__title { color: var(--sb-text-2); font-size: 0.75rem; font-weight: 600; }
  .demo-group [data-drag-item], [data-drag-preview][data-drag-item] {
    padding: 0.5rem 0.75rem;
    border: 1px solid var(--sb-border);
    background: var(--sb-surface-card);
    color: var(--sb-text-1);
    cursor: grab;
    touch-action: none;
    user-select: none;
  }
  .demo-group [data-drag-item] { position: relative; }
  .demo-group :is([data-drag-item], [data-drop-list]):focus-visible { outline: 2px solid var(--sb-brand-light); outline-offset: 2px; }
  .demo-group [data-dragging] { opacity: 0.35; }
  .demo-group[data-key-staging] [data-drag-item]:focus { border-color: var(--sb-brand); }
  .demo-group [data-drop-before]::before, .demo-group [data-drop-end]::after {
    content: "";
    display: block;
    block-size: 3px;
    background: var(--sb-brand);
  }
  .demo-group [data-drop-before]::before { position: absolute; inset: -5px 0 auto; }
  [data-drag-preview][data-drag-item] { box-shadow: var(--sb-shadow-overlay, 0 8px 16px rgb(0 0 0 / 0.35)); cursor: grabbing; }
  @media (forced-colors: active) {
    .demo-group [data-drop-active], .demo-group[data-key-staging] [data-drag-item]:focus, [data-drag-preview][data-drag-item] { border-color: Highlight; }
    .demo-group [data-dragging] { border: 1px dashed CanvasText; }
    .demo-group [data-drop-before]::before, .demo-group [data-drop-end]::after { background: Highlight; }
    .demo-group :is([data-drag-item], [data-drop-list]):focus-visible { outline-color: Highlight; }
  }
</style>
<sb-drag-group id="sort-bodies" data-ignore-morph class="demo-group" data-state="planets=earth,ceres,mars,jupiter dwarfs=pluto,venus"
	data-on:sb-drag-group-move="@get('/demo/arrange/drag-group', {payload: {id: el.id, state: el.dataset.state, move: evt.detail}})">
	<p id="sort-bodies-keys" hidden>To move it, hold Alt and press the arrow keys.</p>
	<section data-drop-list="planets" tabindex="-1" aria-label="Planets">
		<span class="demo-group__title">Planets</span>
		<div id="sort-bodies-earth" data-drag-item="earth" tabindex="0" aria-describedby="sort-bodies-keys">🪐 Earth</div>
		<div id="sort-bodies-ceres" data-drag-item="ceres" tabindex="0" aria-describedby="sort-bodies-keys">🪨 Ceres</div>
		<div id="sort-bodies-mars" data-drag-item="mars" tabindex="0" aria-describedby="sort-bodies-keys">🪐 Mars</div>
		<div id="sort-bodies-jupiter" data-drag-item="jupiter" tabindex="0" aria-describedby="sort-bodies-keys">🪐 Jupiter</div>
	</section>
	<section data-drop-list="dwarfs" tabindex="-1" aria-label="Dwarf planets">
		<span class="demo-group__title">Dwarf planets</span>
		<div id="sort-bodies-pluto" data-drag-item="pluto" tabindex="0" aria-describedby="sort-bodies-keys">🪨 Pluto</div>
		<div id="sort-bodies-venus" data-drag-item="venus" tabindex="0" aria-describedby="sort-bodies-keys">🪐 Venus</div>
	</section>
</sb-drag-group>
```

## Markup and events

Every element with `data-drop-list="<id>"` inside the group is a list, and every element with `data-drag-item="<id>"` inside a list is an item. The ids are what the event reports: list ids name the server's destinations, and item ids must be unique within the group. Items can hold anything, and they need `tabindex="0"` to take the keyboard focus. Other content in a list, such as a title, is left alone.

The example also gives each item an `id`, so the morph moves the element, focus and all; without one, the morph rewrites the element in the old place, and the focus stays there, on another item. Each list has `tabindex="-1"`, so the arrow keys can focus it while it is empty without adding a Tab stop; they pass over an empty list that has no `tabindex`.

| Event | Detail | When |
|---|---|---|
| `sb-drag-group-move` | `{ itemId, fromList, toList, before }` | An item was dropped, or a keyboard move was committed. `fromList` and `toList` are list ids (the same one for a move within a list); `before` is the id of the item it now precedes in `toList`, or `""` for the end. |

Groups are independent: an item never leaves its group. A sortable list or another drag group inside an item handles its own items, since the nearest component owns each gesture. The event bubbles, so a handler that nested components reach should check `evt.target`.

## Keyboard

| Keys | Action |
|---|---|
| Arrow Down / Arrow Up, or j / k | Focus the next / previous item in the list |
| Arrow Left / Arrow Right, or h / l | Focus the item at the same place in the next list to the left / right that has items, or an empty list on the way that has a `tabindex` |
| Home / End | Focus the first / last item in the list |
| Alt + Arrow Down / Up, or Alt + j / k | Move the focused item down / up its list |
| Alt + Arrow Left / Right, or Alt + h / l | Move the focused item into the list to the left / right, at the same place (at the end of a shorter list) |
| Escape | Cancel a move before Alt is released |

Moves add up while Alt is held: each key moves the item on from where the last one put it, so Alt + Arrow Right and then Alt + Arrow Up places it in the next list, one place higher than it was in its own. Releasing Alt sends the move. On macOS, Ctrl + n and Ctrl + p also move the focus. Each action takes its keys from a `data-key-<action>` attribute on the group, a space-separated list such as `data-key-focus-next="ArrowDown n"`; an empty value turns the action off. A key with modifiers names them with `+`, as in `Alt+ArrowUp` or `Ctrl+n`. Give the move keys one, such as `data-key-move-up="Alt+ArrowUp Alt+w"`: its release sends the move, so a move key without a modifier sends each step on its own. The actions are `focus-next`, `focus-previous`, `focus-left`, `focus-right`, `focus-first`, `focus-last`, `move-up`, `move-down`, `move-left`, `move-right` and `cancel`.

## On the server

The server owns the lists. A handler applies the move and renders them again; this is the demo's, which keeps the arrangement in the markup instead of a database:

```go source=internal/web/demo_arrange_group.go#arrangeDragGroup,renderDragGroup
// arrangeDragGroup applies an sb-drag-group-move ({itemId, fromList,
// toList, before}) to lists of bodies.
func arrangeDragGroup(state string, move json.RawMessage) (string, error) {
	g, err := parseGroup(state)
	if err != nil {
		return "", err
	}
	var m struct {
		ItemID   string `json:"itemId"`
		FromList string `json:"fromList"`
		ToList   string `json:"toList"`
		Before   string `json:"before"`
	}
	if err := json.Unmarshal(move, &m); err != nil {
		return "", err
	}
	from, to := g.items[m.FromList], g.items[m.ToList]
	if !slices.Contains(g.names, m.ToList) || !slices.Contains(from, m.ItemID) {
		return "", fmt.Errorf("%q is not in %q, or %q is not a list", m.ItemID, m.FromList, m.ToList)
	}
	if m.FromList != m.ToList {
		g.items[m.FromList] = slices.DeleteFunc(slices.Clone(from), func(id string) bool { return id == m.ItemID })
		to = append(slices.Clone(to), m.ItemID)
	}
	if g.items[m.ToList], err = moveBefore(to, m.ItemID, m.Before); err != nil {
		return "", err
	}
	return g.String(), nil
}

// renderDragGroup is the group's markup: the host the morph replaces, with
// the arrangement in data-state, and a list per name with an item per body.
// Item ids make the morph move an item, focus and all; tabindex="-1" lets
// the arrow keys focus an empty list.
func renderDragGroup(id, state string) string {
	g, _ := parseGroup(state)
	var b strings.Builder
	fmt.Fprintf(&b, "<sb-drag-group id=\"%s\" data-ignore-morph class=\"demo-group\" data-state=\"%s\"\n\tdata-on:sb-drag-group-move=\"%s\">\n", id, state, arrangeOn("drag-group"))
	fmt.Fprintf(&b, "\t<p id=\"%s-keys\" hidden>To move it, hold Alt and press the arrow keys.</p>\n", id)
	for _, name := range g.names {
		fmt.Fprintf(&b, "\t<section data-drop-list=\"%s\" tabindex=\"-1\" aria-label=\"%s\">\n\t\t<span class=\"demo-group__title\">%s</span>\n", name, groupLists[name], groupLists[name])
		for _, item := range g.items[name] {
			fmt.Fprintf(&b, "\t\t<div id=\"%s-%s\" data-drag-item=\"%s\" tabindex=\"0\" aria-describedby=\"%s-keys\">%s</div>\n", id, item, item, id, label(item))
		}
		b.WriteString("\t</section>\n")
	}
	b.WriteString("</sb-drag-group>")
	return b.String()
}
```

## Styling

The component adds no styles: the lists and their items are your page's markup, styled by your page's CSS. While an item moves, it marks the elements involved:

- `data-drag-active` on the group and `data-dragging` on the item while you drag (the copy under the pointer has it too, so scope the rule to the group).
- `data-drop-active` on the list the item would land in, with `data-drop-before` on the item it would land before, or `data-drop-end` on the list when it would land at the end.
- `data-key-staging` on the group while a keyboard move waits for Alt to be released.
- The item under the pointer is a copy of the item in `<body>`, marked `data-drag-preview`, with the original's size in `--sb-source-width` and `--sb-source-height`. A `<template data-sb-preview>` that is a direct child of an item replaces the copy.
- A `<template data-sb-target="before">` (or `"end"`, or `""` for both) that is a direct child of the group is copied into the landing place as a drop marker, marked `data-sb-target-indicator`.

The example above shows one way to draw them.

## Accessibility

Lists and items are elements of your page, so their roles and names are yours: the example names each list with `aria-label`, and points every item's `aria-describedby` at a hidden paragraph that tells how to move it. Keyboard moves need no pointer. The component announces nothing itself: when the server sends the new arrangement, the moved item keeps the focus.

## Licence

The code is PD rockets', under its Beer-Ware licence, in `LICENSE-pd-rockets.txt` next to it. `go run ./cmd/vendorpd` brings in a newer release.
