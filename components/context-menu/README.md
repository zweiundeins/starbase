---
name: Context Menu
tag: sb-context-menu
category: navigation
summary: A menu of actions for the thing it opens on, by right click or a button.
author: derekr
source: https://github.com/derekr/pd-rockets/tree/4f4111722763d2aa8f5851c397af026fcbe97e4f/rocket/context-menu
tags: [context menu, menu, right click, actions, keyboard, popover, pd rockets]
since: 2026-10-06
preview: |
  <style>
    .demo-menu-card { display: grid; gap: 6px; inline-size: 12rem; margin: 0; padding: 0; list-style: none; }
    .demo-menu-card li { display: flex; justify-content: space-between; align-items: center; padding: 0.25rem 0.25rem 0.25rem 0.6rem; border: 1px solid var(--sb-border); background: var(--sb-surface-card); color: var(--sb-text-1); }
    .demo-menu-card button { border: 1px solid var(--sb-border); background: var(--sb-surface-raised); color: var(--sb-text-1); font: inherit; }
    #card-menu[role="menu"] { min-inline-size: 10rem; padding: 0.3rem; border: 1px solid var(--sb-border); background: var(--sb-surface-raised); color: var(--sb-text-1); }
    #card-menu [role="menuitem"] { display: block; inline-size: 100%; padding: 0.3rem 0.5rem; border: 0; background: none; color: inherit; font: inherit; text-align: start; white-space: nowrap; }
  </style>
  <ul class="demo-menu-card">
    <li data-context-id="earth" data-menu-for="card-menu">🪐 Earth <button type="button" data-menu-for="card-menu" aria-haspopup="menu" aria-label="Actions for Earth">…</button></li>
    <li data-context-id="mars" data-menu-for="card-menu">🪐 Mars <button type="button" data-menu-for="card-menu" aria-haspopup="menu" aria-label="Actions for Mars">…</button></li>
  </ul>
  <sb-context-menu id="card-menu" aria-label="Planet actions"><template data-sb-menu><button type="button" role="menuitem" data-action="top">Move to the top</button><button type="button" role="menuitem" data-action="remove">Remove</button></template></sb-context-menu>
usage: |
  <ul>
    <li data-context-id="42" data-menu-for="report-actions">Report 42
      <button type="button" data-menu-for="report-actions" aria-haspopup="menu" aria-label="Actions for report 42">…</button>
    </li>
  </ul>
  <sb-context-menu id="report-actions" aria-label="Report actions"
    data-on:sb-menu-action="@post('/reports/' + evt.detail.contextId + '/' + evt.detail.action)">
    <template data-sb-menu>
      <button type="button" role="menuitem" data-action="archive">Archive</button>
      <button type="button" role="menuitem" data-action="delete">Delete</button>
    </template>
  </sb-context-menu>
---

From [PD rockets](https://github.com/derekr/pd-rockets) by derekr, where it is `pd-context-menu`: the same code, with its names in Starbase's `sb-` prefix.

One menu for many things on a page. Any element marked `data-menu-for="<menu id>"` opens it with a right click, and a button marked the same way opens it with a click, Enter or Space, so keyboards and touch screens reach it too. The menu copies its items from a `<template>` the server renders, remembers which thing it opened for (the nearest `data-context-id`), and emits `sb-menu-action` with the chosen action and that id. What the action does is up to the server.

[`sb-dropdown`](/components/dropdown) is a menu button: one trigger, its items in a JSON prop, and the choice reported for that button. A context menu is one menu that every trigger naming it shares, with its items in your page's markup.

## Examples

### Planets with actions

Right-click a planet, or press its … button. The list's order is in `data-state`; the chosen action goes to `/demo/arrange/context-menu` with that order, and the server answers with the list rearranged. Remove a planet and the server adds a button that brings it back. Nothing is stored.

```html preview
<style>
  .demo-menu { display: grid; gap: 10px; justify-items: start; }
  .demo-menu ul { display: grid; gap: 6px; inline-size: min(100%, 18rem); margin: 0; padding: 0; list-style: none; }
  .demo-menu li {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 0.5rem;
    padding: 0.3rem 0.3rem 0.3rem 0.75rem;
    border: 1px solid var(--sb-border);
    background: var(--sb-surface-card);
    color: var(--sb-text-1);
  }
  .demo-menu li > button, .demo-menu__restore {
    padding: 0.2rem 0.6rem;
    border: 1px solid var(--sb-border);
    background: var(--sb-surface-raised);
    color: var(--sb-text-1);
    font: inherit;
    cursor: pointer;
  }
  .demo-menu li > button[aria-expanded="true"] { border-color: var(--sb-brand); }
  .demo-menu :is(li > button, .demo-menu__restore):focus-visible { outline: 2px solid var(--sb-brand-light); outline-offset: 2px; }
  .demo-menu [role="menu"] {
    min-inline-size: 11rem;
    margin: 0;
    padding: 0.3rem;
    border: 1px solid var(--sb-border);
    background: var(--sb-surface-raised);
    color: var(--sb-text-1);
    box-shadow: var(--sb-shadow-overlay, 0 12px 24px rgb(0 0 0 / 0.35));
    outline: none;
  }
  .demo-menu [role="menu"] [role="menu"]:not(:popover-open) { display: none; }
  .demo-menu [role^="menuitem"] {
    display: flex;
    justify-content: space-between;
    gap: 1rem;
    inline-size: 100%;
    padding: 0.4rem 0.6rem;
    border: 0;
    background: none;
    color: inherit;
    font: inherit;
    text-align: start;
    cursor: pointer;
  }
  .demo-menu [role^="menuitem"]:is(:hover, :focus, [aria-expanded="true"]) { background: var(--sb-surface-hover); outline: none; }
  .demo-menu [role^="menuitem"]:focus-visible { box-shadow: inset 2px 0 var(--sb-brand); }
  .demo-menu [data-submenu]::after { content: "›"; }
  .demo-menu [role="separator"] { block-size: 1px; margin: 0.3rem 0.2rem; background: var(--sb-border); }
  @media (forced-colors: active) {
    .demo-menu [role^="menuitem"]:is(:hover, :focus-visible) { outline: 2px solid Highlight; outline-offset: -2px; }
  }
</style>
<div id="planet-menus" class="demo-menu" data-state="mercury venus earth mars"
	data-on:sb-menu-action="@get('/demo/arrange/context-menu', {payload: {id: el.id, state: el.dataset.state, move: evt.detail}})">
	<ul aria-label="Planets">
		<li id="planet-menus-mercury" data-context-id="mercury" data-menu-for="planet-menus-menu">
			<span>🪐 Mercury</span>
			<button type="button" data-menu-for="planet-menus-menu" aria-haspopup="menu" aria-controls="planet-menus-menu" aria-label="Actions for Mercury">…</button>
		</li>
		<li id="planet-menus-venus" data-context-id="venus" data-menu-for="planet-menus-menu">
			<span>🪐 Venus</span>
			<button type="button" data-menu-for="planet-menus-menu" aria-haspopup="menu" aria-controls="planet-menus-menu" aria-label="Actions for Venus">…</button>
		</li>
		<li id="planet-menus-earth" data-context-id="earth" data-menu-for="planet-menus-menu">
			<span>🪐 Earth</span>
			<button type="button" data-menu-for="planet-menus-menu" aria-haspopup="menu" aria-controls="planet-menus-menu" aria-label="Actions for Earth">…</button>
		</li>
		<li id="planet-menus-mars" data-context-id="mars" data-menu-for="planet-menus-menu">
			<span>🪐 Mars</span>
			<button type="button" data-menu-for="planet-menus-menu" aria-haspopup="menu" aria-controls="planet-menus-menu" aria-label="Actions for Mars">…</button>
		</li>
	</ul>
	<sb-context-menu id="planet-menus-menu" aria-label="Planet actions">
		<template data-sb-menu>
			<button type="button" role="menuitem" data-action="top">Move to the top</button>
			<button type="button" role="menuitem" data-action="bottom">Move to the bottom</button>
			<button type="button" role="menuitem" data-submenu="planet-menus-step" aria-haspopup="menu" aria-expanded="false">Move one place</button>
			<div id="planet-menus-step" role="menu" aria-label="Move one place">
				<button type="button" role="menuitem" data-action="up">Up</button>
				<button type="button" role="menuitem" data-action="down">Down</button>
			</div>
			<div role="separator"></div>
			<button type="button" role="menuitem" data-action="remove">Remove</button>
		</template>
	</sb-context-menu>
</div>
```

## Markup and events

- **The menu** is `<sb-context-menu id="…">` with a `<template data-sb-menu>` that holds its items: elements with `role="menuitem"` (or `menuitemradio`) and the `data-action` they report. Buttons make good items: they take the focus and the clicks. A `role="separator"` element divides groups.
- **A submenu** is an element with an id and `role="menu"` inside the template, opened by an item with `data-submenu="<its id>"`.
- **Triggers** are elements with `data-menu-for="<menu id>"`. A right click on one opens the menu at the pointer; a click on one that is a `<button>` or `<a>` opens it below. The menu reports the `data-context-id` of the nearest element that has one.
- **Placeholders:** `{contextId}` in an item's text, `aria-label`, `title` or `data-menu-param-*` attribute becomes that id when the menu opens. Nothing else is filled in, and no other attribute is touched.
- **Content from the server:** instead of a template, the menu can hold a direct child marked `data-sb-menu-content` (items fetched for one thing). It is shown as it is; call `openFor(trigger)` once it is in place.

| Event | Detail | When |
|---|---|---|
| `sb-menu-action` | `{ action, contextId }` | An item with `data-action` was chosen. The menu closes and gives the focus back to the trigger. |
| `sb-menu-scope` | `{ root, active }` | The menu opened (`active: true`) or closed: a page that has keyboard shortcuts of its own can pause them meanwhile. |

Both bubble. The element also has three methods:

- `openFor(trigger, point?, context?)` opens it for a trigger element: at `point` (`{ x, y }` in the viewport) when given, else below the trigger. `context` adds placeholder values.
- `closeMenu(refocus?)` closes it, and with `true` puts the focus back on the trigger.
- `isOpen()` tells whether it is open.

## Keyboard

| Keys | Action |
|---|---|
| Enter or Space on a trigger button | Open the menu; the menu takes the focus |
| Shift+F10 or the menu key on a trigger | Open it at the trigger (the browser sends a context menu event) |
| Arrow Down / Arrow Up, or j / k | Focus the next / previous item, wrapping around |
| Home / End | Focus the first / last item |
| Enter or Space | Choose the focused item (or the first, while the menu itself has the focus) |
| Arrow Right or l on a submenu's item | Open the submenu |
| Arrow Left, h or Backspace | Close the submenu |
| Escape | Close the innermost menu and give the focus back to what opened it |

On macOS, Ctrl + n and Ctrl + p also move the focus. Each action takes its keys from a `data-key-<action>` attribute on the menu, a space-separated list such as `data-key-focus-next="ArrowDown n"`; an empty value turns the action off. The actions are `focus-next`, `focus-previous`, `focus-left`, `focus-right`, `focus-first`, `focus-last` and `cancel`. The menu handles Escape itself and stops it there, so a dialog or drawer around it stays open.

## On the server

The server decides what an action does and renders the page again. This is the demo's handler, which keeps the order in the markup instead of a database:

```go source=internal/web/demo_arrange_menu.go#arrangeContextMenu,renderContextMenu
// arrangeContextMenu applies an sb-menu-action ({action, contextId}) to the
// list: the menu's actions, and restore, which the page's button sends.
func arrangeContextMenu(state string, move json.RawMessage) (string, error) {
	shown, removed, err := menuLists(state)
	if err != nil {
		return "", err
	}
	var m struct {
		Action    string `json:"action"`
		ContextID string `json:"contextId"`
	}
	if err := json.Unmarshal(move, &m); err != nil {
		return "", err
	}
	i := slices.Index(shown, m.ContextID)
	if i < 0 && m.Action != "restore" {
		return "", fmt.Errorf("%q is not in the list", m.ContextID)
	}
	at := func(j int) string { // the body at j, or "" past the end
		if j < len(shown) {
			return shown[j]
		}
		return ""
	}
	switch m.Action {
	case "top":
		if i > 0 {
			shown, err = moveBefore(shown, m.ContextID, shown[0])
		}
	case "bottom":
		if i < len(shown)-1 {
			shown, err = moveBefore(shown, m.ContextID, "")
		}
	case "up":
		if i > 0 {
			shown, err = moveBefore(shown, m.ContextID, shown[i-1])
		}
	case "down":
		if i < len(shown)-1 {
			shown, err = moveBefore(shown, m.ContextID, at(i+2))
		}
	case "remove":
		if len(shown) == 1 {
			return "", fmt.Errorf("the last body stays")
		}
		shown, removed = slices.Delete(shown, i, i+1), append(removed, m.ContextID)
	case "restore":
		shown, removed = append(shown, removed...), nil
	default:
		return "", fmt.Errorf("no action %q", m.Action)
	}
	if err != nil {
		return "", err
	}
	for _, id := range removed {
		shown = append(shown, "-"+id)
	}
	return strings.Join(shown, " "), nil
}

// renderContextMenu is the demo's markup: the list, a row per body that the
// menu opens for (a right click on the row, or its button), the menu's
// template, and a button that brings removed bodies back.
func renderContextMenu(id, state string) string {
	shown, removed, _ := menuLists(state)
	menu := id + "-menu"
	var b strings.Builder
	fmt.Fprintf(&b, "<div id=\"%s\" class=\"demo-menu\" data-state=\"%s\"\n\tdata-on:sb-menu-action=\"%s\">\n\t<ul aria-label=\"Planets\">\n", id, state, arrangeOn("context-menu"))
	for _, item := range shown {
		fmt.Fprintf(&b, "\t\t<li id=\"%s-%s\" data-context-id=\"%s\" data-menu-for=\"%s\">\n\t\t\t<span>%s</span>\n", id, item, item, menu, label(item))
		fmt.Fprintf(&b, "\t\t\t<button type=\"button\" data-menu-for=\"%s\" aria-haspopup=\"menu\" aria-controls=\"%s\" aria-label=\"Actions for %s\">…</button>\n\t\t</li>\n", menu, menu, html.EscapeString(bodies()[item].Name))
	}
	b.WriteString("\t</ul>\n")
	if len(removed) > 0 {
		names := make([]string, len(removed))
		for i, r := range removed {
			names[i] = html.EscapeString(bodies()[r].Name)
		}
		fmt.Fprintf(&b, "\t<button type=\"button\" class=\"demo-menu__restore\"\n\t\tdata-on:click=\"@get('/demo/arrange/context-menu', {payload: {id: '%s', state: el.parentElement.dataset.state, move: {action: 'restore', contextId: ''}}})\">Bring back %s</button>\n", id, strings.Join(names, ", "))
	}
	fmt.Fprintf(&b, "\t<sb-context-menu id=\"%s\" aria-label=\"Planet actions\">\n\t\t<template data-sb-menu>\n", menu)
	for _, a := range [][2]string{{"top", "Move to the top"}, {"bottom", "Move to the bottom"}} {
		fmt.Fprintf(&b, "\t\t\t<button type=\"button\" role=\"menuitem\" data-action=\"%s\">%s</button>\n", a[0], a[1])
	}
	fmt.Fprintf(&b, "\t\t\t<button type=\"button\" role=\"menuitem\" data-submenu=\"%s-step\" aria-haspopup=\"menu\" aria-expanded=\"false\">Move one place</button>\n", id)
	fmt.Fprintf(&b, "\t\t\t<div id=\"%s-step\" role=\"menu\" aria-label=\"Move one place\">\n", id)
	b.WriteString("\t\t\t\t<button type=\"button\" role=\"menuitem\" data-action=\"up\">Up</button>\n")
	b.WriteString("\t\t\t\t<button type=\"button\" role=\"menuitem\" data-action=\"down\">Down</button>\n\t\t\t</div>\n")
	b.WriteString("\t\t\t<div role=\"separator\"></div>\n")
	b.WriteString("\t\t\t<button type=\"button\" role=\"menuitem\" data-action=\"remove\">Remove</button>\n")
	b.WriteString("\t\t</template>\n\t</sb-context-menu>\n</div>")
	return b.String()
}
```

## Styling

The component adds no styles: the menu, its items and the triggers are your page's markup. While it works, it sets:

- `popover="auto"`, `role="menu"` and `tabindex="-1"` on the element. The browser shows it in the top layer, above everything else.
- Its position, in its inline style: with CSS anchor positioning, an anchor named `--sb-menu-anchor-<n>` on the trigger (or on a point at the pointer); without, fixed coordinates.
- The copied items, in a `<div data-menu-content>` inside it, removed again when it closes.
- `popover="auto"` on a submenu when it first opens. Until then it is an ordinary element inside the menu, so hide it (the example's `[role="menu"] [role="menu"]:not(:popover-open)`).
- `aria-expanded` on the trigger, and on a submenu's item while that submenu is open.
- `data-mobile-sheet` on the menu and its submenus while the viewport is at most 650px wide, to style them as a sheet if you like. An element with `data-submenu-back` inside a submenu closes it.

Because the menu sits in the top layer, its background and shadow are yours to give it; the example draws the shadow from `var(--sb-shadow-overlay)`, like Starbase's own overlays.

## Accessibility

The menu is a `role="menu"` of `menuitem`s: opening it moves the focus into it, and closing it moves the focus back to the trigger. Give it a name with `aria-label`. Give trigger buttons `aria-haspopup="menu"` and a label that says what they act on ("Actions for Earth"); the component keeps their `aria-expanded` up to date. A right click opens the menu for mouse users; keyboard users press the button, or Shift+F10 on a focused trigger.

## Licence

The code is PD rockets', under its Beer-Ware licence, in `LICENSE-pd-rockets.txt` next to it. `go run ./cmd/vendorpd` brings in a newer release.
