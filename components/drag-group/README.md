---
name: Drag Group
tag: sb-drag-group
category: data
summary: Move items within and between lists by dragging or with Alt and the arrow keys.
author: derekr
source: https://github.com/derekr/pd-rockets/tree/4f4111722763d2aa8f5851c397af026fcbe97e4f/rocket/drag-group
tags: [drag and drop, lists, move, reorder, kanban, keyboard, pd rockets]
since: 2026-10-06
preview: |
  <style>
    .demo-group-card { display: grid; grid-template-columns: 1fr 1fr; gap: 8px; inline-size: 16rem; }
    .demo-group-card [data-drop-list] { display: grid; align-content: start; gap: 4px; padding: 0.35rem; border: 1px dashed var(--sb-border); background: var(--sb-surface-inset); }
    .demo-group-card [data-drag-item] { padding: 0.3rem 0.5rem; border: 1px solid var(--sb-border); background: var(--sb-surface-card); color: var(--sb-text-1); font-size: 0.8125rem; }
  </style>
  <sb-drag-group class="demo-group-card">
    <section data-drop-list="planets">
      <div data-drag-item="earth" tabindex="0">🪐 Earth</div>
      <div data-drag-item="ceres" tabindex="0">🪨 Ceres</div>
    </section>
    <section data-drop-list="dwarfs">
      <div data-drag-item="pluto" tabindex="0">🪨 Pluto</div>
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

From [PD rockets](https://github.com/derekr/pd-rockets) by derekr, where it is `pd-drag-group`: the same code, with its names in Starbase's `sb-` prefix.

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
  .demo-group [data-drag-item]:focus-visible { outline: 2px solid var(--sb-brand-light); outline-offset: 2px; }
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
</style>
<sb-drag-group id="sort-bodies" class="demo-group" data-state="planets=earth,ceres,mars,jupiter dwarfs=pluto,venus"
	data-on:sb-drag-group-move="@get('/demo/arrange/drag-group', {payload: {id: el.id, state: el.dataset.state, move: evt.detail}})">
	<section data-drop-list="planets" aria-label="Planets">
		<span class="demo-group__title">Planets</span>
		<div data-drag-item="earth" tabindex="0">🪐 Earth</div>
		<div data-drag-item="ceres" tabindex="0">🪨 Ceres</div>
		<div data-drag-item="mars" tabindex="0">🪐 Mars</div>
		<div data-drag-item="jupiter" tabindex="0">🪐 Jupiter</div>
	</section>
	<section data-drop-list="dwarfs" aria-label="Dwarf planets">
		<span class="demo-group__title">Dwarf planets</span>
		<div data-drag-item="pluto" tabindex="0">🪨 Pluto</div>
		<div data-drag-item="venus" tabindex="0">🪐 Venus</div>
	</section>
</sb-drag-group>
```

## Markup and events

Every element with `data-drop-list="<id>"` inside the group is a list, and every element with `data-drag-item="<id>"` inside a list is an item. The ids are what the event reports: list ids name the server's destinations, and item ids must be unique within the group. Items can hold anything, and they need `tabindex="0"` to take the keyboard focus. Other content in a list, such as a title, is left alone.

| Event | Detail | When |
|---|---|---|
| `sb-drag-group-move` | `{ itemId, fromList, toList, before }` | An item was dropped, or a keyboard move committed. `fromList` and `toList` are list ids (the same one for a move within a list); `before` is the id of the item it now precedes in `toList`, or `""` for the end. |

Groups are independent: an item never leaves its group. A sortable list or another drag group inside an item handles its own items, since the nearest component owns each gesture. The event bubbles, so a handler that nested components reach should check `evt.target`.

## Keyboard

| Keys | Action |
|---|---|
| Arrow Down / Arrow Up, or j / k | Focus the next / previous item in the list |
| Arrow Left / Arrow Right, or h / l | Focus the item at the same place in the list to the left / right |
| Home / End | Focus the first / last item in the list |
| Alt + Arrow Down / Up, or Alt + j / k | Move the focused item down / up its list |
| Alt + Arrow Left / Right, or Alt + h / l | Move the focused item to the end of the list to the left / right |
| Escape | Cancel a move before Alt is released |

Moves add up while Alt is held: Alt + Arrow Right and then Alt + Arrow Up places the item above the last one of the next list. Releasing Alt sends the move. On macOS, Ctrl + n and Ctrl + p also move the focus. Each action takes its keys from a `data-key-<action>` attribute on the group, a space-separated list such as `data-key-focus-next="ArrowDown n"`; an empty value turns the action off. The actions are `focus-next`, `focus-previous`, `focus-left`, `focus-right`, `focus-first`, `focus-last`, `move-up`, `move-down`, `move-left`, `move-right` and `cancel`.

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
func renderDragGroup(id, state string) string {
	g, _ := parseGroup(state)
	var b strings.Builder
	fmt.Fprintf(&b, "<sb-drag-group id=\"%s\" class=\"demo-group\" data-state=\"%s\"\n\tdata-on:sb-drag-group-move=\"%s\">\n", id, state, arrangeOn("drag-group"))
	for _, name := range g.names {
		fmt.Fprintf(&b, "\t<section data-drop-list=\"%s\" aria-label=\"%s\">\n\t\t<span class=\"demo-group__title\">%s</span>\n", name, groupLists[name], groupLists[name])
		for _, item := range g.items[name] {
			fmt.Fprintf(&b, "\t\t<div data-drag-item=\"%s\" tabindex=\"0\">%s</div>\n", item, label(item))
		}
		b.WriteString("\t</section>\n")
	}
	b.WriteString("</sb-drag-group>")
	return b.String()
}
```

## Styling

The component adds no styles: the lists and their items are your page's markup, styled by your page's CSS. While an item moves, it marks the elements involved:

- `data-drag-active` on the group and `data-dragging` on the item while you drag.
- `data-drop-active` on the list the item would land in, with `data-drop-before` on the item it would land before, or `data-drop-end` on the list when it would land at the end.
- `data-key-staging` on the group while a keyboard move waits for Alt to be released.
- The item under the pointer is a copy of the item in `<body>`, marked `data-drag-preview`, with the original's size in `--sb-source-width` and `--sb-source-height`. A `<template data-sb-preview>` inside an item replaces the copy.
- A `<template data-sb-target="before">` (or `"end"`) inside the group is copied into the landing place as a drop marker, marked `data-sb-target-indicator`.

The example above shows one way to draw them.

## Accessibility

Lists and items are elements of your page, so their roles and names are yours: the example names each list with `aria-label`. Keyboard moves need no pointer. The component announces nothing itself: when the server sends the new arrangement, the moved item keeps the focus.

## Licence

The code is PD rockets', under its Beer-Ware licence, in `LICENSE-pd-rockets.txt` next to it. `go run ./cmd/vendorpd` brings in a newer release.
