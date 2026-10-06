---
name: Context Menu
tag: sb-context-menu
category: navigation
summary: A menu of actions for the thing it opens on, by right click or a button.
author: derekr
license: Beerware
source: https://github.com/derekr/pd-rockets/tree/4f4111722763d2aa8f5851c397af026fcbe97e4f/rocket/context-menu
tags: [context menu, menu, right click, actions, keyboard, popover, pd rockets]
since: 2026-10-06
preview: |
  <style>
    .demo-menu-card { display: grid; gap: 6px; inline-size: 12rem; max-inline-size: 100%; margin: 0; padding: 0; list-style: none; }
    .demo-menu-card li { display: flex; justify-content: space-between; align-items: center; padding: 0.25rem 0.25rem 0.25rem 0.6rem; border: 1px solid var(--sb-border); background: var(--sb-surface-card); color: var(--sb-text-1); }
    .demo-menu-card button { border: 1px solid var(--sb-border); background: var(--sb-surface-raised); color: var(--sb-text-1); font: inherit; }
    #card-menu[role="menu"] { --_shadow: var(--sb-shadow-overlay, 0 12px 24px rgb(0 0 0 / 0.55)); min-inline-size: 10rem; padding: 0.3rem; border: 1px solid var(--sb-border); background: var(--sb-surface-raised); color: var(--sb-text-1); box-shadow: var(--_shadow); outline: none; }
    #card-menu [role="menuitem"] { display: block; inline-size: 100%; padding: 0.3rem 0.5rem; border: 0; background: none; color: inherit; font: inherit; text-align: start; white-space: nowrap; }
    #card-menu [role="menuitem"]:is(:hover, :focus) { background: var(--sb-surface-hover); outline: none; }
    #card-menu [role="menuitem"]:focus-visible { box-shadow: inset 2px 0 var(--sb-brand); }
    @media (forced-colors: active) {
      #card-menu [role="menuitem"]:is(:hover, :focus-visible) { outline: 2px solid Highlight; outline-offset: -2px; }
    }
  </style>
  <ul class="demo-menu-card">
    <li data-context-id="earth" data-menu-for="card-menu" data-menu-param-name="Earth">🪐 Earth <button type="button" data-menu-for="card-menu" aria-haspopup="menu" aria-label="Actions for Earth">…</button></li>
    <li data-context-id="mars" data-menu-for="card-menu" data-menu-param-name="Mars">🪐 Mars <button type="button" data-menu-for="card-menu" aria-haspopup="menu" aria-label="Actions for Mars">…</button></li>
  </ul>
  <sb-context-menu id="card-menu" aria-label="Planet actions"><template data-sb-menu><button type="button" role="menuitem" data-action="top">Move to the top</button><button type="button" role="menuitem" data-action="remove">Remove {name}</button></template></sb-context-menu>
usage: |
  <ul>
    <li tabindex="-1" data-context-id="42" data-menu-for="report-actions">Report 42
      <button type="button" data-menu-for="report-actions" aria-haspopup="menu" aria-label="Actions for report 42"
        data-preserve-attr="style aria-expanded">…</button>
    </li>
  </ul>
  <sb-context-menu id="report-actions" aria-label="Report actions" data-ignore-morph
    data-on:sb-menu-action="@post('/reports/' + evt.detail.contextId + '/' + evt.detail.action)">
    <template data-sb-menu>
      <button type="button" role="menuitem" data-action="archive">Archive</button>
      <button type="button" role="menuitem" data-action="delete">Delete</button>
    </template>
  </sb-context-menu>
---

From [PD rockets](https://github.com/derekr/pd-rockets) by derekr, where it is `pd-context-menu`, with its names in Starbase's `sb-` prefix. It carries the patches in `patches/pd-rockets`: they keep it working after a morph reaches it while open, focus the first item on open, hide submenus until they open, take labels and disabled actions from the trigger, close it on Tab or a second click on its button, and declare its events.

One menu for many things on a page. Any element marked `data-menu-for="<menu id>"` opens it with a right click, and a button marked the same way opens it with a click, Enter or Space, so keyboards and touch screens reach it too. The menu copies its items from a `<template>` the server renders, remembers which thing it opened for (the nearest `data-context-id`), and emits `sb-menu-action` with the chosen action and that id. What the action does is up to the server.

[`sb-dropdown`](/components/dropdown) is a menu button: one trigger, its items in a JSON prop or as markup (`slot="item"`), and the choice reported for that button. A context menu is one menu that every trigger naming it shares, and each trigger tells it which thing it acts on.

## Examples

### Planets with actions

Right-click a planet, or press its … button. The list's order is in `data-state`; the chosen action goes to `/demo/arrange/context-menu` with that order, and the server answers with the list rearranged. Remove a planet and the server adds a button that brings it back. Nothing is stored.

Each row gives the menu its planet's name (`data-menu-param-name`, for "Remove Mars") and the actions that don't apply to it (`data-menu-disabled`: the first planet can't move up). The menu itself never changes, so it carries `data-ignore-morph`, and the buttons keep the anchor it sets on them with `data-preserve-attr`: an answer that arrives while it is open leaves it open, next to its planet.

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
  .demo-menu :is(li, li > button, .demo-menu__restore):focus-visible { outline: 2px solid var(--sb-brand-light); outline-offset: 2px; }
  .demo-menu sb-context-menu { anchor-name: --demo-menu; }
  .demo-menu [role="menu"] {
    --_shadow: var(--sb-shadow-overlay, 0 12px 24px rgb(0 0 0 / 0.55));
    min-inline-size: 11rem;
    margin: 0;
    padding: 0.3rem;
    border: 1px solid var(--sb-border);
    background: var(--sb-surface-raised);
    color: var(--sb-text-1);
    box-shadow: var(--_shadow);
    outline: none;
  }
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
    outline: none;
  }
  .demo-menu [role^="menuitem"]:is(:hover, :focus, [aria-expanded="true"]):not([aria-disabled="true"]) { background: var(--sb-surface-hover); }
  .demo-menu [role^="menuitem"]:focus-visible { box-shadow: inset 2px 0 var(--sb-brand); }
  .demo-menu [role^="menuitem"][aria-disabled="true"] { color: var(--sb-text-muted); cursor: default; }
  .demo-menu [data-submenu]::after { content: "›"; }
  .demo-menu [role="separator"] { block-size: 1px; margin: 0.3rem 0.2rem; background: var(--sb-border); }
  /* On a phone the menu is a sheet at the bottom of the screen, and a submenu sits on top of it. */
  .demo-menu sb-context-menu[data-mobile-sheet] { inset: auto 0 0 !important; inline-size: auto; position-try-fallbacks: none !important; }
  .demo-menu [role="menu"] [role="menu"][data-mobile-sheet] {
    position-anchor: --demo-menu !important;
    inset: auto 0 calc(anchor(top) + 4px) !important;
    inline-size: auto;
    position-try-fallbacks: none !important;
  }
  @media (forced-colors: active) {
    .demo-menu [role^="menuitem"]:is(:hover, :focus-visible) { outline: 2px solid Highlight; outline-offset: -2px; }
    .demo-menu [role^="menuitem"][aria-disabled="true"] { color: GrayText; }
    .demo-menu [role="separator"] { background: CanvasText; }
    .demo-menu li > button[aria-expanded="true"] { border-color: Highlight; }
    .demo-menu :is(li, li > button, .demo-menu__restore):focus-visible { outline-color: Highlight; }
  }
</style>
<div id="planet-menus" class="demo-menu" data-state="mercury venus earth mars"
	data-on:sb-menu-action="@get('/demo/arrange/context-menu', {payload: {id: el.id, state: el.dataset.state, move: evt.detail}})">
	<ul aria-label="Planets">
		<li id="planet-menus-mercury" tabindex="-1" data-context-id="mercury" data-menu-for="planet-menus-menu" data-menu-param-name="Mercury" data-menu-disabled="top up">
			<span>🪐 Mercury</span>
			<button type="button" data-menu-for="planet-menus-menu" aria-haspopup="menu" aria-controls="planet-menus-menu" aria-label="Actions for Mercury" data-preserve-attr="style aria-expanded">…</button>
		</li>
		<li id="planet-menus-venus" tabindex="-1" data-context-id="venus" data-menu-for="planet-menus-menu" data-menu-param-name="Venus">
			<span>🪐 Venus</span>
			<button type="button" data-menu-for="planet-menus-menu" aria-haspopup="menu" aria-controls="planet-menus-menu" aria-label="Actions for Venus" data-preserve-attr="style aria-expanded">…</button>
		</li>
		<li id="planet-menus-earth" tabindex="-1" data-context-id="earth" data-menu-for="planet-menus-menu" data-menu-param-name="Earth">
			<span>🪐 Earth</span>
			<button type="button" data-menu-for="planet-menus-menu" aria-haspopup="menu" aria-controls="planet-menus-menu" aria-label="Actions for Earth" data-preserve-attr="style aria-expanded">…</button>
		</li>
		<li id="planet-menus-mars" tabindex="-1" data-context-id="mars" data-menu-for="planet-menus-menu" data-menu-param-name="Mars" data-menu-disabled="bottom down">
			<span>🪐 Mars</span>
			<button type="button" data-menu-for="planet-menus-menu" aria-haspopup="menu" aria-controls="planet-menus-menu" aria-label="Actions for Mars" data-preserve-attr="style aria-expanded">…</button>
		</li>
	</ul>
	<sb-context-menu id="planet-menus-menu" aria-label="Planet actions" data-ignore-morph>
		<template data-sb-menu>
			<button type="button" role="menuitem" data-action="top">Move to the top</button>
			<button type="button" role="menuitem" data-action="bottom">Move to the bottom</button>
			<button type="button" role="menuitem" data-submenu="planet-menus-step" aria-haspopup="menu" aria-expanded="false">Move one place</button>
			<div id="planet-menus-step" role="menu" aria-label="Move one place">
				<button type="button" role="menuitem" data-action="up">Up</button>
				<button type="button" role="menuitem" data-action="down">Down</button>
			</div>
			<div role="separator"></div>
			<button type="button" role="menuitem" data-action="remove">Remove {name}</button>
		</template>
	</sb-context-menu>
</div>
```

## Markup and events

- **The menu** is `<sb-context-menu id="…">` with a `<template data-sb-menu>` as its direct child, holding the items: elements with `role="menuitem"` (or `menuitemradio`) and the `data-action` they report. Buttons make good items: they take the focus and the clicks. A `role="separator"` element divides groups.
- **A submenu** is an element with an id and `role="menu"` inside the template, opened by an item with `data-submenu="<its id>"`. It stays hidden until then.
- **Triggers** are elements with `data-menu-for="<menu id>"`. A right click on one opens the menu at the pointer; a click on one that is a `<button>` or `<a>` opens it below, and a second click closes it. The menu reports the `data-context-id` of the nearest element that has one.
- **Placeholders:** `{contextId}` in an item's text, `aria-label`, `title` or `data-menu-param-*` attribute becomes that id when the menu opens. A `data-menu-param-<word>` on the trigger, or on the element with its context id, fills `{<word>}`: the demo's rows carry `data-menu-param-name` for "Remove {name}". Nothing else is filled in, and no other attribute is touched.
- **Actions that don't apply:** `data-menu-disabled="<action> …"` on a trigger, or on the element with its context id, gives the copied items with those `data-action`s `aria-disabled="true"`. The keyboard skips them and a click on one does nothing.
- **Content from the server:** instead of a template, the menu can hold a direct child marked `data-sb-menu-content` (items fetched for one thing), shown as it is. Until that child exists a trigger opens nothing and no event reports the press, so the page fetches the items with a handler of its own on the trigger (`data-on:click="@get(…)"`) and calls `openFor(trigger)` once they are in place.
- **Morphs:** the menu sets attributes the server doesn't render and adds the copied items, so a morph that reaches it while it is open closes it (the next open works as usual). A menu whose template doesn't change can carry `data-ignore-morph`, like the demo's; its trigger buttons then keep the anchor and `aria-expanded` it gives them with `data-preserve-attr="style aria-expanded"`.

| Event | Detail | When |
|---|---|---|
| `sb-menu-action` | `{ action, contextId }` | An item with `data-action` was chosen. The menu closes and gives the focus back to the trigger. |
| `sb-menu-scope` | `{ root, active }` | The menu opened (`active: true`) or closed: a page that has keyboard shortcuts of its own can pause them meanwhile. |

Both bubble. The element also has three methods:

- `openFor(trigger, point?, context?)` opens it for a trigger element: at `point` (`{ x, y }` in the viewport) when given, else below the trigger. `context` adds placeholder values, over the trigger's.
- `closeMenu(refocus?)` closes it, and with `true` puts the focus back on the trigger.
- `isOpen()` tells whether it is open.

## Keyboard

| Keys | Action |
|---|---|
| Enter or Space on a trigger button | Open the menu; its first item takes the focus |
| Shift+F10 or the menu key on a trigger | Open it at the trigger (the browser sends a context menu event) |
| Arrow Down / Arrow Up, or j / k | Focus the next / previous item, wrapping around and skipping disabled ones |
| Home / End | Focus the first / last item |
| Enter or Space | Choose the focused item |
| Arrow Right or l on a submenu's item | Open the submenu and focus its first item |
| Arrow Left, h or Backspace | Close the submenu |
| Escape | Close the innermost menu and give the focus back to what opened it |
| Tab or Shift+Tab | Close the menu, give the focus back to the trigger and move on from there |

On macOS, Ctrl + n and Ctrl + p also move the focus. Each action takes its keys from a `data-key-<action>` attribute on the menu, a space-separated list such as `data-key-focus-next="ArrowDown n"`; an empty value turns the action off. The actions are `focus-next`, `focus-previous`, `focus-left`, `focus-right`, `focus-first`, `focus-last` and `cancel`. The menu handles Escape itself and stops it there, so a dialog or drawer around it stays open.

## On the server

The server decides what an action does and renders the page again. This is the demo's handler, which keeps the order in the markup instead of a database. A removal or a restore takes the focused button away (with its row, or the restore button itself), so the new state marks the planet whose button takes the focus with a `*`, and the markup gives that button a `data-init` that focuses it when the focus fell back to the page:

```go source=internal/web/demo_arrange_menu.go#arrangeContextMenu,renderContextMenu
// arrangeContextMenu applies an sb-menu-action ({action, contextId}) to the
// list: the menu's actions, and restore, which the page's button sends. A
// removal or a restore takes away the focused button, so the new state marks
// the body whose button takes the focus instead.
func arrangeContextMenu(state string, move json.RawMessage) (string, error) {
	shown, removed, _, err := menuLists(state)
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
	focus := ""
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
		focus = shown[min(i, len(shown)-1)] // the body that took its place
	case "restore":
		if len(removed) == 0 {
			return "", fmt.Errorf("no body to bring back")
		}
		shown, removed, focus = append(shown, removed...), nil, removed[0]
	default:
		return "", fmt.Errorf("no action %q", m.Action)
	}
	if err != nil {
		return "", err
	}
	words := make([]string, 0, len(shown)+len(removed))
	for _, id := range shown {
		if id == focus {
			id = "*" + id
		}
		words = append(words, id)
	}
	for _, id := range removed {
		words = append(words, "-"+id)
	}
	return strings.Join(words, " "), nil
}

// renderContextMenu is the demo's markup: the list, a row per body that the
// menu opens for (a right click on the row, or its button), with its name
// for the menu's labels and the actions that don't apply to it, the menu's
// template, and a button that brings removed bodies back. The menu ignores
// morphs and the buttons keep what it sets on them (their anchor and
// aria-expanded), so an answer that arrives while it is open leaves it alone.
func renderContextMenu(id, state string) string {
	shown, removed, focus, _ := menuLists(state)
	menu := id + "-menu"
	var b strings.Builder
	fmt.Fprintf(&b, "<div id=\"%s\" class=\"demo-menu\" data-state=\"%s\"\n\tdata-on:sb-menu-action=\"%s\">\n\t<ul aria-label=\"Planets\">\n", id, state, arrangeOn("context-menu"))
	for k, item := range shown {
		name := html.EscapeString(bodies()[item].Name)
		var off []string
		if k == 0 {
			off = append(off, "top", "up")
		}
		if k == len(shown)-1 {
			off = append(off, "bottom", "down")
		}
		if len(shown) == 1 {
			off = append(off, "remove")
		}
		fmt.Fprintf(&b, "\t\t<li id=\"%s-%s\" tabindex=\"-1\" data-context-id=\"%s\" data-menu-for=\"%s\" data-menu-param-name=\"%s\"", id, item, item, menu, name)
		if len(off) > 0 {
			fmt.Fprintf(&b, " data-menu-disabled=\"%s\"", strings.Join(off, " "))
		}
		fmt.Fprintf(&b, ">\n\t\t\t<span>%s</span>\n", label(item))
		fmt.Fprintf(&b, "\t\t\t<button type=\"button\" data-menu-for=\"%s\" aria-haspopup=\"menu\" aria-controls=\"%s\" aria-label=\"Actions for %s\" data-preserve-attr=\"style aria-expanded\"", menu, menu, name)
		if item == focus { // only where the answer took the focus away
			gone := "" // naming the removed row makes each removal's data-init new, so it runs on the same button again
			if len(removed) > 0 {
				gone = fmt.Sprintf("!document.getElementById('%s-%s') && ", id, removed[len(removed)-1])
			}
			fmt.Fprintf(&b, "\n\t\t\t\tdata-init=\"%sdocument.activeElement === document.body && el.focus()\"", gone)
		}
		b.WriteString(">…</button>\n\t\t</li>\n")
	}
	b.WriteString("\t</ul>\n")
	if len(removed) > 0 {
		names := make([]string, len(removed))
		for i, r := range removed {
			names[i] = html.EscapeString(bodies()[r].Name)
		}
		fmt.Fprintf(&b, "\t<button type=\"button\" class=\"demo-menu__restore\"\n\t\tdata-on:click=\"@get('/demo/arrange/context-menu', {payload: {id: '%s', state: el.parentElement.dataset.state, move: {action: 'restore', contextId: ''}}})\">Bring back %s</button>\n", id, strings.Join(names, ", "))
	}
	fmt.Fprintf(&b, "\t<sb-context-menu id=\"%s\" aria-label=\"Planet actions\" data-ignore-morph>\n\t\t<template data-sb-menu>\n", menu)
	for _, a := range [][2]string{{"top", "Move to the top"}, {"bottom", "Move to the bottom"}} {
		fmt.Fprintf(&b, "\t\t\t<button type=\"button\" role=\"menuitem\" data-action=\"%s\">%s</button>\n", a[0], a[1])
	}
	fmt.Fprintf(&b, "\t\t\t<button type=\"button\" role=\"menuitem\" data-submenu=\"%s-step\" aria-haspopup=\"menu\" aria-expanded=\"false\">Move one place</button>\n", id)
	fmt.Fprintf(&b, "\t\t\t<div id=\"%s-step\" role=\"menu\" aria-label=\"Move one place\">\n", id)
	b.WriteString("\t\t\t\t<button type=\"button\" role=\"menuitem\" data-action=\"up\">Up</button>\n")
	b.WriteString("\t\t\t\t<button type=\"button\" role=\"menuitem\" data-action=\"down\">Down</button>\n\t\t\t</div>\n")
	b.WriteString("\t\t\t<div role=\"separator\"></div>\n")
	b.WriteString("\t\t\t<button type=\"button\" role=\"menuitem\" data-action=\"remove\">Remove {name}</button>\n")
	b.WriteString("\t\t</template>\n\t</sb-context-menu>\n</div>")
	return b.String()
}
```

## Styling

The component adds no styles: the menu, its items and the triggers are your page's markup. While it works, it sets:

- `popover="auto"`, `role="menu"` and `tabindex="-1"` on the element. The browser shows it in the top layer, above everything else.
- Its position, in its inline style: with CSS anchor positioning, an anchor named `--sb-menu-anchor-<n>` on the trigger (or on a point at the pointer); without, fixed coordinates.
- The copied items, in a `<div data-menu-content>` inside it, removed again when it closes.
- `popover="auto"` on every submenu when the menu opens, which hides a submenu until it opens.
- `aria-expanded` on a trigger button, and on a submenu's item while that submenu is open. A row opened by a right click gets none.
- `aria-disabled="true"` on the copied items whose actions the trigger turns off.
- `data-mobile-sheet` on the menu and its submenus while the viewport is at most 650px wide. Their position stays in the inline style, so a rule that moves them needs `!important`: that way the example turns the menu into a sheet at the bottom of the screen and puts a submenu on top of it (through an `anchor-name` on the menu), where it covers none of the menu's items. An element with `data-submenu-back` inside a submenu closes it.

Because the menu sits in the top layer, its background and shadow are yours to give it; the example draws the shadow from `var(--sb-shadow-overlay)` through a `--_shadow` local, like Starbase's own overlays.

## Accessibility

The menu is a `role="menu"` of `menuitem`s. Opening it moves the focus to its first item that isn't disabled. Choosing an item, Escape and Tab give the focus back to the trigger, so a row that opens the menu on a right click needs `tabindex="-1"` to take it, as in the demo. When the server's answer removes the trigger (the demo's Remove), the focus falls back to the page, and the answer decides where it goes. Give the menu a name with `aria-label`. Give trigger buttons `aria-haspopup="menu"` and a label that says what they act on ("Actions for Earth"); the component keeps their `aria-expanded` up to date. A right click opens the menu for mouse users; keyboard users press the button, or Shift+F10 on a focused trigger. iOS Safari sends no `contextmenu` event on a long press, so give every row a button.

## Licence

The code is PD rockets', under its Beer-Ware licence, in `LICENSE-pd-rockets.txt` next to it. `go run ./cmd/vendorpd` brings in a newer release.
