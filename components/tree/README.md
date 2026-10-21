---
name: Tree
tag: sb-tree
category: navigation
summary: A keyboard-friendly tree of links or items from the server's lists, with lazy branches.
author: zweiundeins
tags: [tree, hierarchy, lazy, navigation, sidebar, file browser]
since: 2026-09-22
preview: |
  <sb-tree style="inline-size: 13rem" value="earth">
    <ul>
      <li data-id="sol" data-open><svg data-icon viewBox="0 0 16 16" width="16" height="16" fill="currentColor" shape-rendering="crispEdges" aria-hidden="true"><path d="M6 5h4v1h1v4h-1v1H6v-1H5V6h1zM7 1h2v2H7zM7 13h2v2H7zM1 7h2v2H1zM13 7h2v2h-2z"/></svg> Sol <span data-meta>3</span>
        <ul>
          <li data-id="venus"><svg data-icon viewBox="0 0 16 16" width="16" height="16" fill="currentColor" shape-rendering="crispEdges" aria-hidden="true"><path d="M6 4h4v1h1v1h1v4h-1v1h-1v1H6v-1H5v-1H4V6h1V5h1z"/></svg> Venus</li>
          <li data-id="earth" data-open><svg data-icon viewBox="0 0 16 16" width="16" height="16" fill="currentColor" shape-rendering="crispEdges" aria-hidden="true"><path d="M6 4h4v1h1v1h1v1h3v2h-3v1h-1v1h-1v1H6v-1H5v-1H4V9H1V7h3V6h1V5h1z"/></svg> Earth <span data-meta>1</span>
            <ul><li data-id="moon"><svg data-icon viewBox="0 0 16 16" width="16" height="16" fill="currentColor" shape-rendering="crispEdges" aria-hidden="true"><path d="M6 2h4v1H8v1H7v2H6v4h1v2h1v1h2v1H6v-1H5v-1H4v-1H3V6h1V5h1V4h1z"/></svg> Moon</li></ul>
          </li>
          <li data-id="mars"><svg data-icon viewBox="0 0 16 16" width="16" height="16" fill="currentColor" shape-rendering="crispEdges" aria-hidden="true"><path d="M6 4h4v1h1v1h1v4h-1v1h-1v1H6v-1H5v-1H4V6h1V5h1z"/></svg> Mars</li>
        </ul>
      </li>
    </ul>
  </sb-tree>
usage: |
  <sb-tree label="Docs">
    <ul>
      <li><a href="/guide">Guide</a>
        <ul>
          <li><a href="/guide/install" aria-current="page">Install</a></li>
          <li><a href="/guide/deploy">Deploy <span data-meta>new</span></a></li>
        </ul>
      </li>
      <li data-id="api" data-lazy><a href="/api">API</a></li>
    </ul>
  </sb-tree>
playground:
  attrs:
    items: '[{"id":"sol","label":"Sol","icon":"☀️","children":[{"id":"venus","label":"Venus"},{"id":"earth","label":"Earth","icon":"🌍","children":[{"id":"moon","label":"Moon"}]}]}]'
  exclude: [items, loaded, value]
---

A tree of links or items: a docs sidebar, a file browser, categories, an org chart. The server renders it as nested lists inside the element, the way it would without JavaScript, and the component shows them as a tree with keyboard support, open and closed branches, icons and a note after each label. Items with an `<a href>` stay links: they open in a new tab, show their URL in the status bar and work before the script loads. A branch can also wait for the server: opening it asks for its children with an `sb-load` event, and the server patches a list of them into its `<li>`.

The tree can also be given as JSON in `items` (see [Items as JSON](#items-as-json)).

## Examples

### A navigation tree

Links, an SVG icon per item, a note after the label (`data-meta`) and the page you are on (`aria-current="page"`, whose branch opens by itself). Ideas has no link: it is a plain item, which can be selected.

```html preview
<sb-tree label="Components" style="inline-size: min(100%, 18rem)">
  <ul>
    <li data-id="navigation">
      <a href="/?cat=navigation"><svg data-icon viewBox="0 0 16 16" width="16" height="16" fill="currentColor" shape-rendering="crispEdges" aria-hidden="true"><path d="M1 3h5v1h1v1h8v8H1z"/></svg> Navigation <span data-meta>3</span></a>
      <ul>
        <li><a href="/components/tabs"><svg data-icon viewBox="0 0 16 16" width="16" height="16" fill="currentColor" shape-rendering="crispEdges" aria-hidden="true"><path d="M3 1h7v1h1v1h1v1h1v11H3z"/></svg> Tabs</a></li>
        <li><a href="/components/toc"><svg data-icon viewBox="0 0 16 16" width="16" height="16" fill="currentColor" shape-rendering="crispEdges" aria-hidden="true"><path d="M3 1h7v1h1v1h1v1h1v11H3z"/></svg> Table of Contents</a></li>
        <li><a href="/components/tree" aria-current="page"><svg data-icon viewBox="0 0 16 16" width="16" height="16" fill="currentColor" shape-rendering="crispEdges" aria-hidden="true"><path d="M3 1h7v1h1v1h1v1h1v11H3z"/></svg> Tree</a></li>
      </ul>
    </li>
    <li data-id="forms">
      <a href="/?cat=forms"><svg data-icon viewBox="0 0 16 16" width="16" height="16" fill="currentColor" shape-rendering="crispEdges" aria-hidden="true"><path d="M1 3h5v1h1v1h8v8H1z"/></svg> Forms <span data-meta>2</span></a>
      <ul>
        <li><a href="/components/select"><svg data-icon viewBox="0 0 16 16" width="16" height="16" fill="currentColor" shape-rendering="crispEdges" aria-hidden="true"><path d="M3 1h7v1h1v1h1v1h1v11H3z"/></svg> Select</a></li>
        <li><a href="/components/date-picker"><svg data-icon viewBox="0 0 16 16" width="16" height="16" fill="currentColor" shape-rendering="crispEdges" aria-hidden="true"><path d="M3 1h7v1h1v1h1v1h1v11H3z"/></svg> Date Picker</a></li>
      </ul>
    </li>
    <li data-id="ideas"><svg data-icon viewBox="0 0 16 16" width="16" height="16" fill="currentColor" shape-rendering="crispEdges" aria-hidden="true"><path d="M6 1h4v1h1v1h1v5h-1v1h-1v2H6V9H5V8H4V3h1V2h1zM6 12h4v1H6zM7 14h2v1H7z"/></svg> Ideas <span data-meta>draft</span></li>
  </ul>
</sb-tree>
```

Without JavaScript this is a nested list of links with their icons, and crawlers read it as one. Once the module runs, the list itself is no longer shown: the component draws its own rows from it, styled inside the component, so the page needs no stylesheet for them.

### Lazy loading from the server

Every branch below loads when you first open it, from the site's example dataset (`/demo/data/children`). Datastar does the wiring:

1. `sb-load` runs `@get('/demo/data/children?parent=…&tree=sky')`.
2. The server answers with the children as a `<ul>`, patched into the end of the parent's `<li>` (an element patch with the selector `#sky li[data-id="…"]` and the mode `append`).
3. The tree reads the new list, and the branch stops showing "Loading".

`expanded` opens the path to the Moon from the start: each level loads once its parent has arrived. The number after a label is how many children an item has.

```html preview
<div data-signals="{_picked: ''}" style="display: grid; gap: 12px; inline-size: min(100%, 22rem)">
  <sb-tree id="sky" label="The sky" expanded="milkyway solarsystem earth"
    data-on:sb-load="@get('/demo/data/children?parent=' + evt.detail.id + '&tree=sky&delay=250')"
    data-bind:_picked__prop.value>
    <ul>
      <li data-id="milkyway" data-lazy><svg data-icon viewBox="0 0 16 16" width="16" height="16" fill="currentColor" shape-rendering="crispEdges" aria-hidden="true"><path d="M7 7h2v2H7zM3 4h5v1h1v1H3zM8 10h5v1h-1v1H8zM2 6h1v3H2zM13 7h1v3h-1z"/></svg> Milky Way</li>
      <li data-id="andromeda" data-lazy><svg data-icon viewBox="0 0 16 16" width="16" height="16" fill="currentColor" shape-rendering="crispEdges" aria-hidden="true"><path d="M7 7h2v2H7zM3 4h5v1h1v1H3zM8 10h5v1h-1v1H8zM2 6h1v3H2zM13 7h1v3h-1z"/></svg> Andromeda</li>
    </ul>
  </sb-tree>
  <span>Selected: <b data-text="$_picked || 'nothing'"></b></span>
</div>
```

The handler behind this demo is Go with the [Datastar SDK](https://data-star.dev/reference/sdks); any SDK works the same way. A server that keeps the tree's state can also send the whole tree again, with the loaded branches in it: a morph of the element keeps the branches the user opened.

```go source=internal/web/demo_data.go#Server.demoChildren,Server.demoTreeMarkup
// demoChildren answers with the children of ?parent= (the top level without one).
// With &tree=<id>, as markup for the sb-tree with that id (demoTreeMarkup).
func (s *Server) demoChildren(w http.ResponseWriter, r *http.Request) {
	parent := r.URL.Query().Get("parent")
	bodies, err := s.q.ViewValue(r.Context(), func(rd *queries.Reader) ([]queries.DemoBody, error) {
		return rd.DemoChildren(r.Context(), parent)
	})
	if err != nil {
		s.fail(w, r, err)
		return
	}
	if tree := r.URL.Query().Get("tree"); tree != "" {
		s.demoTreeMarkup(w, r, tree, parent, bodies)
		return
	}
	items := demoItems(bodies)
	var patch any = items
	if parent != "" {
		patch = map[string]any{parent: items}
	}
	s.demoAnswer(w, r, "_tree", items, patch)
}

// demoTreeMarkup answers sb-tree's sb-load with the children as a list,
// appended to the parent's <li> (for the top level, the tree's whole list).
// Bodies with children of their own are lazy, and show how many they have.
func (s *Server) demoTreeMarkup(w http.ResponseWriter, r *http.Request, tree, parent string, bodies []queries.DemoBody) {
	if !elementIDRe.MatchString(tree) || !demoParentRe.MatchString(parent) {
		http.Error(w, "tree must be an element id, parent an item id", http.StatusBadRequest)
		return
	}
	w.Header().Set("Access-Control-Allow-Origin", "*") // public; used from the playground sandbox
	if !demoWait(r) {
		return
	}
	var b strings.Builder
	b.WriteString("<ul>")
	for _, body := range bodies {
		fmt.Fprintf(&b, `<li data-id="%s"`, html.EscapeString(body.ID))
		if body.Children > 0 {
			b.WriteString(" data-lazy")
		}
		fmt.Fprintf(&b, `>%s %s`, demoIcon(body.Kind), html.EscapeString(body.Name))
		if body.Children > 0 {
			fmt.Fprintf(&b, ` <span data-meta>%d</span>`, body.Children)
		}
		b.WriteString("</li>")
	}
	b.WriteString("</ul>")
	p := sse.Patch{Selector: "#" + tree, Mode: sse.Inner}
	if parent != "" {
		p = sse.Patch{Selector: fmt.Sprintf(`#%s li[data-id="%s"]`, tree, parent), Mode: sse.Append}
	}
	if stream, err := sse.Open(w, r, sse.Options{}); err == nil {
		stream.CloseWith(sse.PatchElements([]byte(b.String()), p))
	}
}
```

### Multiple selection

```html preview
<sb-tree selection="multiple" value="io europa" style="inline-size: 16rem">
  <ul>
    <li data-id="jupiter" data-open>Jupiter
      <ul>
        <li data-id="io">Io</li>
        <li data-id="europa">Europa</li>
        <li data-id="ganymede">Ganymede</li>
        <li data-id="callisto">Callisto</li>
      </ul>
    </li>
  </ul>
</sb-tree>
```

## Markup

Put a `<ul>` (or `<ol>`) directly inside the element, with a `<li>` per item and a nested list for its children. In a `<li>`:

- **`<a href>`** makes the item a link. Its row is an `<a>` with the same `href`, `target`, `rel` and `title`, so a click, Enter, a middle click and Ctrl or Cmd with a click do what they do on any link. Without a link, the item is plain text (bare, or in a `<span>`), and it can be selected.
- **`data-id`** on the `<li>` is the item's id, used by `value`, `expanded`, `sb-load`, `sb-toggle` and `sb-change`. Without it, the id is the link's `href`, else the labels on the way to the item. Use `data-id`, not `id`: ids are unique in the whole page.
- **`data-icon`** marks the item's icon: an inline `<svg>` (in `currentColor`, it takes the row's colour), an `<img>`, or a `<span>` with a glyph. The tree shows a copy of it at the size of the text; give an SVG `width` and `height` for the page without JavaScript.
- **`data-meta`** marks a short text shown muted after the label: a count, "draft", "new". It is not part of the label.
- **`aria-current`** on the link (usually `"page"`) marks the page you are on. Its row is highlighted, it is the row Tab lands on, and the branches above it open when it first shows or moves.
- **`data-open`** on the `<li>` opens the branch.
- **`data-lazy`** on a `<li>` without a list makes it a branch whose children come later: opening it emits `sb-load` with its id, and the server adds a `<ul>` to that `<li>`. An empty `<ul>` says it has none.

The tree keeps reading the list, so a morph or a patch can change any of it. Which branches are open is the tree's own state, not an attribute on the list, so the user's open and closed branches and the keyboard focus survive a morph of the list. The server's word wins when it changes: a new `data-open` opens the branch, and a new `expanded` replaces the open branches; the same markup again leaves the user's choices alone. A focused row that the server drops hands the focus to the row that takes its place.

When the element holds a list and also has `items`, the list wins.

## Items as JSON

The same tree as JSON in `items`: `[{id, label, icon?, children?, lazy?}]`.

- `id`: unique across the tree, and a plain name (letters, digits and `_`), since it becomes a key in the signal.
- `icon`: a short text or emoji.
- `children`: nested items, given up front.
- `lazy: true` without `children`: the item can be opened, and asks for its children with `sb-load`.

JSON items are plain items: no links, icons only as text, no meta, and nothing to show before the module runs. Children of lazy items come in `loaded`, `{"<id>": [items]}`. With Datastar, patch them into a signal and hand it to the tree with `data-attr:loaded`:

```html preview
<div data-signals="{_sky: {}}" style="inline-size: min(100%, 22rem)">
  <sb-tree label="The sky (JSON)" expanded="milkyway solarsystem"
    items='[{"id":"milkyway","label":"Milky Way","icon":"🌌","lazy":true},{"id":"andromeda","label":"Andromeda","icon":"🌌","lazy":true}]'
    data-attr:loaded="JSON.stringify($_sky)" data-preserve-attr="loaded"
    data-on:sb-load="@get('/demo/data/children?parent=' + evt.detail.id + '&into=_sky&delay=250')"></sb-tree>
</div>
```

Here the server answers with a signal patch, `{_sky: {"sol": [children…]}}`. Signal patches merge, so each loaded branch adds to the rest. `demoAnswer` also serves plain JSON and the demo's `delay`:

```go source=internal/web/demo_data.go#Server.demoAnswer
// demoAnswer writes the list as JSON, or patches it into the requested signal.
func (s *Server) demoAnswer(w http.ResponseWriter, r *http.Request, defaultSignal string, list, patch any) {
	w.Header().Set("Access-Control-Allow-Origin", "*") // public; used from the playground sandbox
	if !demoWait(r) {
		return
	}
	// Datastar's own requests accept JSON too: they always get the patch.
	if r.Header.Get("Datastar-Request") == "" && strings.Contains(r.Header.Get("Accept"), "application/json") {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "public, max-age=300")
		json.NewEncoder(w).Encode(list)
		return
	}
	into := r.URL.Query().Get("into")
	if into == "" {
		into = defaultSignal
	}
	if !signalNameRe.MatchString(into) {
		http.Error(w, "into must be a signal name", http.StatusBadRequest)
		return
	}
	js, err := json.Marshal(map[string]any{into: patch})
	if err != nil {
		s.fail(w, r, err)
		return
	}
	// Through lith's sse, which compresses: the edge leaves text/event-stream to it.
	if stream, err := sse.Open(w, r, sse.Options{}); err == nil {
		stream.CloseWith(sse.PatchSignals(js, false))
	}
}
```

Without Datastar signals, the server can also re-render the element with a new `loaded` (or `items`, `value`, `expanded`) attribute: a changed attribute always wins, and a removed `value` or `expanded` is ignored. Children can come in either: a lazy item stops loading once `items` or `loaded` has its children. To clear the selection or close every branch, send `value=""` or `expanded=""`.

## With commands

Give it a `name`, and it emits `sb-change` with `{ name, value }` when the selection changes: ready to post as a command. With `confirm`, it sets `:state(pending)` until the server's re-rendered `value` matches, and `revert()` goes back to the server's selection when a command is rejected. `sb-toggle` reports opened and closed branches, so the server can keep `expanded` too. See [Commands and components](/contribute#commands-and-components).

Links are followed, not selected: a tree of links marks the page with `aria-current`, and the server moves it when it renders the next page.

## Forms

`sb-tree` is not a form-associated element: a `<form>` doesn't submit it, `FormData` and Datastar's `contentType: 'form'` don't see it, and a form reset doesn't reset it. Send its value as a command instead: `sb-change` carries `{ name, value }` (see [With commands](#with-commands)).

## Styling

Style it from your page's CSS, without changing the component or importing anything into it. Custom properties, inherited properties and `::part()` all reach into its shadow root.

- **Size:** set `font-size` on the element (default `0.875rem`) to scale the text and the icons; rows are at least `2rem` tall (`min-block-size` on `::part(item)`).
- **Fonts:** the rows use your page's font.
- **Colours:** text is `--sb-text-1`; the carets, the meta and "loading" are `--sb-text-muted`. Rows turn `--sb-surface-hover` on hover; the selected row and the current page are `--sb-brand-subtle` with a `--sb-brand` edge. The focus ring is `--sb-brand-light`, corners `--sb-radius-sm`. `--sb-notch: 0` turns the stepped pixel carets into plain triangles that turn smoothly.
- **Parts:** `tree`, `item` (every row), and in a row `caret`, `icon`, `label` and `meta`. Selected rows are also `selected` and the current page `current`, so `::part(item selected)` styles only those; they move with the selection. Your page's `::part()` rules win over the component's own, without `!important`.
- **Without JavaScript:** the list is the page's to style, for example with `sb-tree:not(:defined) ul { list-style: none }`.

```html preview
<style>
  .my-tree { font-size: 1rem; --sb-brand: var(--sb-accent); }
  .my-tree::part(item) { min-block-size: 2.5rem; }
  .my-tree::part(item selected) { font-weight: 700; }
  .my-tree::part(meta) { color: var(--sb-accent); }
</style>
<sb-tree class="my-tree" label="Files" value="main">
  <ul>
    <li data-id="src" data-open>src <span data-meta>2</span>
      <ul>
        <li data-id="main">main.go</li>
        <li data-id="util">util.go</li>
      </ul>
    </li>
    <li data-id="readme">README.md</li>
  </ul>
</sb-tree>
```

## Accessibility

It follows the WAI-ARIA tree pattern, and for links its navigation tree variant:

- **Structure:** `tree` and `treeitem` roles, with level, position and expanded state. A link row is a link with the `treeitem` role; the current page has `aria-current`.
- **Focus:** one item at a time is in the tab order. Tab lands on the selected item, else the current page (the first one visible), otherwise on the item focused last, or the first.
- **Keys:** Up and Down move, Right opens (or moves to the first child), Left closes (or moves to the parent), Home and End jump. Enter or Space selects a plain item and follows a link. Keys with Alt, Ctrl or Cmd are the browser's (Alt+Left goes back).
- **Loading:** a branch that is loading is marked `aria-busy`.
