---
name: Tree
tag: sb-tree
category: navigation
summary: A keyboard-friendly tree whose branches can load lazily from the server.
author: zweiundeins
tags: [tree, hierarchy, lazy, navigation, file browser]
since: 2026-09-22
preview: |
  <sb-tree style="inline-size: 13rem" value="earth" items='[{"id":"sol","label":"Sol","icon":"☀️","children":[{"id":"venus","label":"Venus","icon":"🟠"},{"id":"earth","label":"Earth","icon":"🌍","children":[{"id":"moon","label":"Moon","icon":"🌕"}]},{"id":"mars","label":"Mars","icon":"🔴"}]}]'></sb-tree>
playground:
  attrs:
    items: '[{"id":"sol","label":"Sol","icon":"☀️","children":[{"id":"venus","label":"Venus"},{"id":"earth","label":"Earth","icon":"🌍","children":[{"id":"moon","label":"Moon"}]}]}]'
  exclude: [items, loaded, value]
---

A tree of items (files, categories, an org chart…). Give it the items as JSON, or let the server fill in branches when they are opened: a lazy item asks for its children with an `sb-load` event, and you answer by patching a signal.

## Examples

### Lazy loading from the server

Every branch below loads when you first open it, from the site's example dataset (`/demo/data/children`). Datastar does the wiring:

1. `sb-load` runs `@get('/demo/data/children?parent=…&into=_sky')`.
2. The server answers with a signal patch, `{_sky: {"sol": [children…]}}`. Signal patches merge, so each loaded branch adds to the rest.
3. `data-attr:loaded` hands `$_sky` back to the tree.

`expanded` opens the path to the Moon from the start: each level loads once its parent has arrived.

```html preview
<div data-signals="{_sky: {}, _picked: ''}" style="display: grid; gap: 12px; inline-size: min(100%, 22rem)">
  <sb-tree label="The sky" expanded="milkyway solarsystem earth"
    items='[{"id":"milkyway","label":"Milky Way","icon":"🌌","lazy":true},{"id":"andromeda","label":"Andromeda","icon":"🌌","lazy":true}]'
    data-attr:loaded="JSON.stringify($_sky)" data-preserve-attr="loaded"
    data-on:sb-load="@get('/demo/data/children?parent=' + evt.detail.id + '&into=_sky&delay=250')"
    data-bind:_picked__prop.value></sb-tree>
  <span>Selected: <b data-text="$_picked || 'nothing'"></b></span>
</div>
```

The server side is a plain Datastar handler (Go here; any language works):

```go
func children(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("parent")
	datastar.NewSSE(w, r).MarshalAndPatchSignals(map[string]any{
		"_sky": map[string]any{id: childrenOf(id)}, // [{id, label, icon?, lazy?}]
	})
}
```

Without Datastar signals, the server can also re-render the element with a new `loaded` (or `items`, `value`, `expanded`) attribute: a changed attribute always wins.

### Multiple selection

```html preview
<sb-tree selection="multiple" value="io europa" style="inline-size: 16rem"
  items='[{"id":"jupiter","label":"Jupiter","icon":"🪐","children":[{"id":"io","label":"Io"},{"id":"europa","label":"Europa"},{"id":"ganymede","label":"Ganymede"},{"id":"callisto","label":"Callisto"}]}]'></sb-tree>
```

## With commands

Give it a `name`, and it emits `sb-change` with `{ name, value }` when the selection changes: ready to post as a command. With `confirm`, it sets `:state(pending)` until the server's re-rendered `value` matches, and `revert()` goes back to the server's selection when a command is rejected. `sb-toggle` reports opened and closed branches, so the server can keep `expanded` too. See [Commands and components](/contribute#commands-and-components).

## Items

Each item is `{id, label, icon?, children?, lazy?}`:

- `id`: unique across the tree, and a plain name (letters, digits and `_`), since it becomes a key in the signal.
- `icon`: a short text or emoji.
- `children`: nested items, given up front.
- `lazy: true` without `children`: the item can be opened, and asks for its children with `sb-load`.

## Accessibility

It follows the WAI-ARIA tree pattern:

- **Structure:** `tree` and `treeitem` roles, with level, position and expanded state.
- **Focus:** one item at a time is in the tab order.
- **Keys:** Up and Down move, Right opens (or moves to the first child), Left closes (or moves to the parent), Home and End jump, and Enter or Space selects.
- **Loading:** a branch that is loading is marked `aria-busy`.
