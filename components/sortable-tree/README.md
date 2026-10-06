---
name: Sortable Tree
tag: sb-sortable-tree
category: data
summary: Move files and folders by dragging or with Alt and the arrow keys. The server applies it.
author: derekr
license: Beerware
source: https://github.com/derekr/pd-rockets/tree/4f4111722763d2aa8f5851c397af026fcbe97e4f/rocket/sortable-tree
tags: [drag and drop, sortable, tree, file tree, folders, reorder, keyboard, pd rockets]
since: 2026-10-06
preview: |
  <style>
    .demo-tree-card { display: block; inline-size: 12rem; }
    .demo-tree-card [data-tree-children] { display: grid; gap: 2px; }
    .demo-tree-card [data-tree-children] [data-tree-children] { padding-inline-start: 1.1rem; }
    .demo-tree-card [data-tree-row], [data-drag-preview][data-tree-row] { padding: 0.25rem 0.5rem; background: var(--sb-surface-card); color: var(--sb-text-1); cursor: grab; touch-action: none; user-select: none; }
    .demo-tree-card [data-tree-row] { position: relative; }
    .demo-tree-card [data-tree-kind="folder"] > [data-tree-row] { font-weight: 600; }
    .demo-tree-card [data-tree-row]:focus-visible { outline: 2px solid var(--sb-brand-light); outline-offset: -2px; }
    .demo-tree-card [data-dragging] { opacity: 0.35; }
    .demo-tree-card [data-tree-into] { background: var(--sb-brand-subtle); outline: 1px solid var(--sb-brand); outline-offset: -1px; }
    .demo-tree-card [data-tree-before]::before, .demo-tree-card [data-tree-end]::after { content: ""; display: block; block-size: 3px; background: var(--sb-brand); }
    .demo-tree-card [data-tree-before]::before { position: absolute; inset: -3px 0 auto; }
    [data-drag-preview][data-tree-row] { box-shadow: var(--sb-shadow-overlay, 0 8px 16px rgb(0 0 0 / 0.35)); cursor: grabbing; }
    @media (forced-colors: active) {
      .demo-tree-card [data-tree-before]::before, .demo-tree-card [data-tree-end]::after { forced-color-adjust: none; background: Highlight; }
      .demo-tree-card [data-tree-into], .demo-tree-card [data-tree-row]:focus-visible { outline: 2px solid Highlight; outline-offset: -2px; }
      .demo-tree-card [data-dragging], [data-drag-preview][data-tree-row] { outline: 1px dashed CanvasText; outline-offset: -1px; }
    }
  </style>
  <sb-sortable-tree id="sortable-tree-card" class="demo-tree-card" data-state="earth(moon) mars(phobos)"
    data-on:sb-tree-move="@get('/demo/arrange/sortable-tree-card', {payload: {id: el.id, state: el.dataset.state, move: evt.detail}})">
    <div data-tree-children data-tree-parent="">
      <div id="sortable-tree-card-earth" data-tree-node="earth" data-tree-kind="folder">
        <div data-tree-row tabindex="0" aria-expanded="true">🪐 Earth</div>
        <div data-tree-children data-tree-parent="earth">
          <div id="sortable-tree-card-moon" data-tree-node="moon" data-tree-kind="file"><div data-tree-row tabindex="-1">🌑 Moon</div></div>
        </div>
      </div>
      <div id="sortable-tree-card-mars" data-tree-node="mars" data-tree-kind="folder">
        <div data-tree-row tabindex="-1" aria-expanded="true">🪐 Mars</div>
        <div data-tree-children data-tree-parent="mars">
          <div id="sortable-tree-card-phobos" data-tree-node="phobos" data-tree-kind="file"><div data-tree-row tabindex="-1">🌑 Phobos</div></div>
        </div>
      </div>
    </div>
  </sb-sortable-tree>
usage: |
  <sb-sortable-tree id="files" data-on:sb-tree-move="@post('/files/move', {payload: evt.detail})">
    <div data-tree-children data-tree-parent="">
      <div id="file-docs" data-tree-node="docs" data-tree-kind="folder">
        <div data-tree-row tabindex="0">docs</div>
        <div data-tree-children data-tree-parent="docs">
          <div id="file-readme" data-tree-node="readme"><div data-tree-row tabindex="0">README.md</div></div>
        </div>
      </div>
      <div id="file-license" data-tree-node="license"><div data-tree-row tabindex="0">LICENSE</div></div>
    </div>
  </sb-sortable-tree>
---

From [PD rockets](https://github.com/derekr/pd-rockets) by derekr, where it is `pd-sortable-tree`, with its names in Starbase's `sb-` prefix and the patches in `patches/pd-rockets`. For this tree they open and close a folder with a click on its row, declare its move event for the API table below, skip the move animation when the reader prefers reduced motion, and let it nest inside the other PD rockets surfaces, or they inside it.

Folders and files the server renders and arranges, in compact rows like a file explorer. Drag a row, or focus it and hold Alt while you press the arrow keys, to move it among its siblings, into a folder or out of one. The component shows where the row would land, then emits `sb-tree-move` with the row, the folder it leaves, the folder it goes to and the row it goes before. It moves nothing itself: the server applies the move and sends the tree back, and the morph puts each row in its place with a short animation.

[sb-tree](/components/tree) is a different thing: a tree view that renders its items from JSON, selects them and loads children lazily. Here the server writes the whole tree as markup, and the component only moves it around.

## Examples

### Put the moons back with their planets

The server renders the tree with its arrangement in `data-state`: each folder's children in parentheses after it. A move sends that arrangement with the event's detail to `/demo/arrange/sortable-tree`, which answers with the tree after the move. Phobos belongs to Mars and Callisto to Jupiter. Nothing is stored: try it in two tabs.

```html preview
<style>
  .demo-tree { display: block; inline-size: min(100%, 18rem); }
  .demo-tree [data-tree-children] { display: grid; gap: 2px; }
  .demo-tree [data-tree-children] [data-tree-children] { padding-inline-start: 1.25rem; }
  .demo-tree [data-tree-row], [data-drag-preview][data-tree-row] {
    padding: 0.3rem 0.6rem;
    border: 1px solid transparent;
    background: var(--sb-surface-card);
    color: var(--sb-text-1);
    cursor: grab;
    touch-action: none;
    user-select: none;
  }
  .demo-tree [data-tree-row] { position: relative; }
  .demo-tree [data-tree-kind="folder"] > [data-tree-row] { font-weight: 600; }
  .demo-tree [data-tree-row][aria-expanded="false"] { color: var(--sb-text-2); }
  .demo-tree [data-tree-row]:hover { background: var(--sb-surface-hover); }
  .demo-tree [data-tree-row]:focus-visible { outline: 2px solid var(--sb-brand-light); outline-offset: -2px; }
  .demo-tree [data-dragging] { opacity: 0.35; }
  .demo-tree[data-key-staging] [data-tree-row]:focus { border-color: var(--sb-brand); }
  .demo-tree [data-tree-into] { border-color: var(--sb-brand); background: var(--sb-brand-subtle); }
  .demo-tree [data-tree-before]::before, .demo-tree [data-tree-end]::after {
    content: "";
    display: block;
    block-size: 3px;
    background: var(--sb-brand);
  }
  .demo-tree [data-tree-before]::before { position: absolute; inset: -3px 0 auto; }
  [data-drag-preview][data-tree-row] { box-shadow: var(--sb-shadow-overlay, 0 8px 16px rgb(0 0 0 / 0.35)); cursor: grabbing; }
  @media (forced-colors: active) {
    .demo-tree [data-tree-before]::before, .demo-tree [data-tree-end]::after { forced-color-adjust: none; background: Highlight; }
    .demo-tree [data-tree-into], .demo-tree [data-tree-row]:focus-visible { outline: 2px solid Highlight; outline-offset: -2px; }
    .demo-tree [data-dragging], [data-drag-preview][data-tree-row] { border-style: dashed; }
  }
</style>
<sb-sortable-tree id="solar-moons" class="demo-tree" data-state="earth(moon phobos) mars(deimos) jupiter(europa io) callisto"
	data-on:sb-tree-move="@get('/demo/arrange/sortable-tree', {payload: {id: el.id, state: el.dataset.state, move: evt.detail}})">
	<div data-tree-children data-tree-parent="" aria-label="Solar System">
		<div id="solar-moons-earth" data-tree-node="earth" data-tree-kind="folder">
			<div data-tree-row tabindex="0" aria-label="folder: Earth" aria-expanded="true">🪐 Earth</div>
			<div data-tree-children data-tree-parent="earth">
				<div id="solar-moons-moon" data-tree-node="moon" data-tree-kind="file">
					<div data-tree-row tabindex="0" aria-label="file: Moon">🌑 Moon</div>
				</div>
				<div id="solar-moons-phobos" data-tree-node="phobos" data-tree-kind="file">
					<div data-tree-row tabindex="0" aria-label="file: Phobos">🌑 Phobos</div>
				</div>
			</div>
		</div>
		<div id="solar-moons-mars" data-tree-node="mars" data-tree-kind="folder">
			<div data-tree-row tabindex="0" aria-label="folder: Mars" aria-expanded="true">🪐 Mars</div>
			<div data-tree-children data-tree-parent="mars">
				<div id="solar-moons-deimos" data-tree-node="deimos" data-tree-kind="file">
					<div data-tree-row tabindex="0" aria-label="file: Deimos">🌑 Deimos</div>
				</div>
			</div>
		</div>
		<div id="solar-moons-jupiter" data-tree-node="jupiter" data-tree-kind="folder">
			<div data-tree-row tabindex="0" aria-label="folder: Jupiter" aria-expanded="true">🪐 Jupiter</div>
			<div data-tree-children data-tree-parent="jupiter">
				<div id="solar-moons-europa" data-tree-node="europa" data-tree-kind="file">
					<div data-tree-row tabindex="0" aria-label="file: Europa">🌑 Europa</div>
				</div>
				<div id="solar-moons-io" data-tree-node="io" data-tree-kind="file">
					<div data-tree-row tabindex="0" aria-label="file: Io">🌑 Io</div>
				</div>
			</div>
		</div>
		<div id="solar-moons-callisto" data-tree-node="callisto" data-tree-kind="file">
			<div data-tree-row tabindex="0" aria-label="file: Callisto">🌑 Callisto</div>
		</div>
	</div>
</sb-sortable-tree>
```

## Markup and events

The tree is your page's markup, in four kinds of element:

- A list, `data-tree-children`: the top level directly inside the tree, and one in each folder, with `data-tree-parent="<the folder's id>"`. The top level's parent is `""`.
- A node, `data-tree-node="<id>"`, in a list. With `data-tree-kind="folder"` it is a folder; any other node is a file.
- A row, `data-tree-row`, directly inside its node: what you see, drag and focus. Rows need a `tabindex` to take the keyboard focus: `"0"` on every row puts each in the Tab order, as in the demo; `"0"` on the first and `"-1"` on the rest makes the tree one Tab stop, and the arrow keys still reach every row.
- A folder's list, directly inside the folder's node, after its row.

Give each node a unique `id`. The morph then moves nodes when the server's order changes, and the component sees the move and closes the folders the reader closed again. Without ids the morph rewrites nodes in their places, and a closed folder can open again after an answer.

Datastar's official release (v1.0.4) crashes on a morph that moves a node with an id when the node holds a Rocket component ([#1209](https://github.com/starfederation/datastar/issues/1209), fixed for its next release). The build the install snippets load has the fix, so with it a row may hold one, such as an `sb-relative-time` for a file's date. On the official release, keep Rocket components out of the rows, or the ids off the nodes.

| Event | Detail | When |
|---|---|---|
| `sb-tree-move` | `{ itemId, fromParent, toParent, before }` | A row was dropped, or a keyboard move committed. `fromParent` and `toParent` are folder ids, `""` for the top level; `before` is the id of the node it now precedes, or `""` for the end of the folder. |

A drop on the middle of a folder's row puts the node at the end of that folder; on the upper or lower part of a row, before or after it. The component never offers a folder a place inside itself, nor a file as a parent. The event bubbles: when trees are nested, check `evt.target` in a handler that several of them reach.

A click on a folder's row opens or closes it, and so do Arrow Right and Arrow Left on the focused row; a click on a control inside the row (a link, a button, a field) is left to that control. Folders open and close in the browser alone: the component hides a closed folder's list and keeps `aria-expanded` on its row, and the server never hears of it. A move into a closed folder opens it once the server has answered.

## Keyboard

| Keys | Action |
|---|---|
| Arrow Down / Arrow Up, or j / k | Focus the next / previous visible row |
| Arrow Right, or l | Open a closed folder; on an open one, focus its first row |
| Arrow Left, or h | Close an open folder; elsewhere, focus the folder that holds the row |
| Home / End | Focus the first / last visible row |
| Alt + Arrow Down / Up, or Alt + j / k | Move the row among its siblings; past the first or last, out of the folder, before or after it |
| Alt + Arrow Right, or Alt + l | Move the row into the nearest folder above it, at the end |
| Alt + Arrow Left, or Alt + h | Move the row out of its folder, right after it |
| Escape | Cancel a move before Alt is released |

Moves add up while Alt is held; releasing Alt sends the move. On macOS, Ctrl + n and Ctrl + p also move the focus. Each action takes its keys from a `data-key-<action>` attribute on the tree, a space-separated list such as `data-key-focus-next="ArrowDown n"`; an empty value turns the action off. The actions are `focus-next`, `focus-previous`, `focus-left`, `focus-right`, `focus-first`, `focus-last`, `move-up`, `move-down`, `move-left`, `move-right` and `cancel`.

## On the server

The server owns the tree. A handler checks the move, applies it and renders the tree again; this is the demo's, which keeps the arrangement in the markup instead of a database:

```go source=internal/web/demo_arrange_tree.go#arrangeSortableTree,renderSortableTree
// arrangeSortableTree applies an sb-tree-move ({itemId, fromParent,
// toParent, before}; "" is the top level) to the tree in a state.
func arrangeSortableTree(state string, move json.RawMessage) (string, error) {
	root, err := parseTree(state)
	if err != nil {
		return "", err
	}
	var m struct {
		ItemID     string `json:"itemId"`
		FromParent string `json:"fromParent"`
		ToParent   string `json:"toParent"`
		Before     string `json:"before"`
	}
	if err := json.Unmarshal(move, &m); err != nil {
		return "", err
	}
	// Every node's list and the folder it is in.
	lists := map[string]*[]*treeNode{"": &root}
	parent := map[string]string{}
	var item *treeNode
	var walk func(folder string, list []*treeNode)
	walk = func(folder string, list []*treeNode) {
		for _, n := range list {
			parent[n.id] = folder
			if n.id == m.ItemID {
				item = n
			}
			if n.folder {
				lists[n.id] = &n.children
				walk(n.id, n.children)
			}
		}
	}
	walk("", root)
	to, ok := lists[m.ToParent]
	switch {
	case item == nil || parent[m.ItemID] != m.FromParent:
		return "", fmt.Errorf("%q is not in %q", m.ItemID, m.FromParent)
	case !ok:
		return "", fmt.Errorf("%q is not a folder", m.ToParent)
	}
	for f := m.ToParent; f != ""; f = parent[f] {
		if f == m.ItemID {
			return "", fmt.Errorf("%q can't go into itself", m.ItemID)
		}
	}
	// Out of its folder, at the end of the other, then before its new neighbour.
	from := lists[m.FromParent]
	*from = remove(*from, item)
	order := make([]string, 0, len(*to)+1)
	byID := map[string]*treeNode{item.id: item}
	for _, n := range *to {
		order = append(order, n.id)
		byID[n.id] = n
	}
	if order, err = moveBefore(append(order, item.id), item.id, m.Before); err != nil {
		return "", err
	}
	*to = (*to)[:0]
	for _, id := range order {
		*to = append(*to, byID[id])
	}
	state = formatTree(root)
	if _, err := parseTree(state); err != nil {
		return "", fmt.Errorf("%q in %q nests the tree too deep", m.ItemID, m.ToParent)
	}
	return state, nil
}

// renderSortableTree is the tree's markup: the host the morph replaces,
// with the tree in data-state, folders open, and a row per body. Each node
// has an id, so the morph moves it rather than rewriting the node in its
// place, and the component sees the move (it closes its closed folders
// again when the tree's children change).
func renderSortableTree(id, state string) string {
	root, _ := parseTree(state)
	var b strings.Builder
	fmt.Fprintf(&b, "<sb-sortable-tree id=\"%s\" class=\"demo-tree\" data-state=\"%s\"\n\tdata-on:sb-tree-move=\"%s\">\n", id, state, arrangeOn("sortable-tree"))
	b.WriteString("\t<div data-tree-children data-tree-parent=\"\" aria-label=\"Solar System\">\n")
	var nodes func(list []*treeNode, indent string)
	nodes = func(list []*treeNode, indent string) {
		for _, n := range list {
			kind, expanded := "file", ""
			if n.folder {
				kind, expanded = "folder", ` aria-expanded="true"`
			}
			name := html.EscapeString(bodies()[n.id].Name)
			fmt.Fprintf(&b, "%s<div id=\"%s-%s\" data-tree-node=\"%s\" data-tree-kind=\"%s\">\n", indent, id, n.id, n.id, kind)
			fmt.Fprintf(&b, "%s\t<div data-tree-row tabindex=\"0\" aria-label=\"%s: %s\"%s>%s</div>\n", indent, kind, name, expanded, label(n.id))
			if n.folder {
				fmt.Fprintf(&b, "%s\t<div data-tree-children data-tree-parent=\"%s\">\n", indent, n.id)
				nodes(n.children, indent+"\t\t")
				fmt.Fprintf(&b, "%s\t</div>\n", indent)
			}
			fmt.Fprintf(&b, "%s</div>\n", indent)
		}
	}
	nodes(root, "\t\t")
	b.WriteString("\t</div>\n</sb-sortable-tree>")
	return b.String()
}
```

## Styling

The component adds no styles: the tree is your page's markup, styled by your page's CSS. While a row moves, it marks the elements involved:

- `data-drag-active` on the tree and `data-dragging` on the row while you drag.
- `data-tree-before` on the row the moving one would land before, `data-tree-into` on the row of the folder it would go into, and `data-tree-end` on the list it would land at the end of.
- `data-key-staging` on the tree while a keyboard move waits for Alt to be released.
- `hidden` on a closed folder's list, and `aria-expanded` on every folder's row.
- The row under the pointer is a copy of the row in `<body>`, marked `data-drag-preview`, with the original's size in `--sb-source-width` and `--sb-source-height`. A `<template data-sb-preview>` that is a direct child of the row replaces the copy: its content goes in a `<div>` with the template's class. One deeper inside the row is ignored.
- A `<template data-sb-target="before">` (or `"into"`, `"end"`; `data-sb-target=""` serves every kind without its own) is copied into the marked row or list as a drop marker: a `<span data-sb-target-indicator="<kind>">` with `position: absolute`, which your CSS places. The templates must be direct children of the tree; one inside a list or a row is ignored.

The example above shows one way to draw them.

## Accessibility

Rows are focusable elements of your page, so their names are yours: the demo labels each one, "folder: Earth" or "file: Moon". The markup has no tree roles, like PD rockets' own examples, so assistive technology reads the rows as a list of labelled items, and the `aria-expanded` the component keeps on folder rows only counts on an element whose role supports it (a `treeitem` or a `button`). Keyboard moves need no pointer. When the server's answer arrives, the moved row keeps the focus, and a folder it went into opens.

## Licence

The code is PD rockets', with Starbase's patches, under its Beer-Ware licence, in `LICENSE-pd-rockets.txt` next to it. `go run ./cmd/vendorpd` brings in a newer release.
