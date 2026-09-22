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

Every branch below is loaded when you first open it. Datastar does the wiring:

1. `sb-load` runs `@get('/demo/tree?id=…')`.
2. The server answers with a signal patch, `{_tree: {"sol": [children…]}}`. Signal patches merge, so each loaded branch adds to the rest.
3. `data-attr:loaded` hands `$_tree` back to the tree.

```html preview
<div data-signals="{_tree: {}, _picked: ''}" style="display: grid; gap: 12px; inline-size: min(100%, 22rem)">
  <sb-tree label="The sky"
    items='[{"id":"milkyway","label":"Milky Way","icon":"🌌","lazy":true},{"id":"andromeda","label":"Andromeda","icon":"🌀","lazy":true}]'
    data-attr:loaded="JSON.stringify($_tree)" data-preserve-attr="loaded"
    data-on:sb-load="@get('/demo/tree?id=' + evt.detail.id)"
    data-bind:_picked__prop.value></sb-tree>
  <span>Selected: <b data-text="$_picked || 'nothing'"></b></span>
</div>
```

The server side is a plain Datastar handler (Go here; any language works):

```go
func tree(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	datastar.NewSSE(w, r).MarshalAndPatchSignals(map[string]any{
		"_tree": map[string]any{id: childrenOf(id)}, // [{id, label, icon?, lazy?}]
	})
}
```

### Multiple selection

```html preview
<sb-tree selection="multiple" value="io europa" style="inline-size: 16rem"
  items='[{"id":"jupiter","label":"Jupiter","icon":"🪐","children":[{"id":"io","label":"Io"},{"id":"europa","label":"Europa"},{"id":"ganymede","label":"Ganymede"},{"id":"callisto","label":"Callisto"}]}]'></sb-tree>
```

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
