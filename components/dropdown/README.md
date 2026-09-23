---
name: Dropdown
tag: sb-dropdown
category: navigation
summary: An actions menu with submenus, whose items are commands you can post.
author: zweiundeins
tags: [dropdown, menu, actions, popover, keyboard, navigation]
since: 2026-09-23
preview: |
  <sb-dropdown data-signals:_sent="''" data-preserve-attr="label"
    data-attr:label="$_sent ? 'Sent: ' + $_sent : 'Ship actions'"
    data-on:sb-select="$_sent = evt.detail.value"
    items='[{"value":"refuel","label":"Refuel","icon":"⛽"},{"label":"Set course","icon":"🧭","children":[{"value":"mars","label":"Mars"},{"value":"europa","label":"Europa"}]},{"divider":true},{"value":"scuttle","label":"Scuttle","icon":"💥","danger":true}]'></sb-dropdown>
playground:
  attrs:
    "data-on:sb-select": '$_pg.label = "Sent: " + evt.detail.value'
    items: '[{"value":"refuel","label":"Refuel","icon":"⛽"},{"label":"Set course","icon":"🧭","children":[{"value":"mars","label":"Mars"},{"value":"europa","label":"Europa"},{"label":"Outer system","children":[{"value":"titan","label":"Titan"},{"value":"triton","label":"Triton"}]}]},{"divider":true},{"value":"scuttle","label":"Scuttle","icon":"💥","danger":true}]'
  values: {label: "Ship actions"}
  exclude: [items, name, open]
---

An actions menu: a trigger opens a list of things the user can *do*, with submenus where a choice needs one. Choosing an item emits `sb-select` with `{ name, value }`, so a page posts it as a command and shows whatever the server renders next. A menu holds no value, so there is nothing pending and nothing to revert.

**Nothing stays selected.** No checkmark, no highlighted row, and the trigger keeps its label: the menu hands over an intent and forgets it, so everything visible afterwards is the page reacting (the demos here write the value into a signal). If you want a value that sticks, reach for [sb-select](/components/select).

The menu is a native `popover` in the top layer, positioned with CSS anchor positioning where the browser has it (and by hand, flipping and shifting, where it doesn't). No ancestor can clip it.

## Examples

### Actions

`items` is a JSON array. An item is a string, `{value, label?, description?, icon?, disabled?, danger?}`, or `{"divider": true}` — the string `"-"` is a divider too.

```html preview
<div data-signals:_log="''" style="display: grid; gap: 12px; justify-items: start">
  <sb-dropdown name="ship" label="Ship actions"
    items='["Refuel", {"value":"scan","label":"Long-range scan","icon":"📡","description":"Takes a while"}, "-", {"value":"dock","label":"Dock","disabled":true}, {"value":"scuttle","label":"Scuttle","icon":"💥","danger":true}]'
    data-on:sb-select="$_log = evt.detail.name + ' → ' + evt.detail.value"></sb-dropdown>
  <code data-text="$_log || 'Pick something…'"></code>
</div>
```

### Submenus

An item with `children` opens a submenu instead of reporting a value: the children win, so a parent never emits `sb-select` even when it carries a `value`. Nest up to five levels; anything deeper is dropped, so a runaway tree cannot build a menu nobody can reach.

```html preview
<div data-signals:_log="''" style="display: grid; gap: 12px; justify-items: start">
  <sb-dropdown name="doc" label="Document"
    items='[{"value":"rename","label":"Rename","icon":"✏️"},
            {"label":"Export","icon":"📦","children":[
              {"value":"csv","label":"CSV"},
              {"value":"json","label":"JSON","description":"Pretty printed"},
              {"label":"Archive","children":[{"value":"zip","label":"ZIP"},{"value":"tar","label":"TAR"}]}]},
            {"label":"Move to","icon":"🗂","children":[{"value":"drafts","label":"Drafts"},{"value":"sent","label":"Sent"},{"value":"trash","label":"Trash","danger":true}]},
            {"divider":true},
            {"value":"delete","label":"Delete","icon":"💥","danger":true}]'
    data-on:sb-select="$_log = evt.detail.value"></sb-dropdown>
  <code data-text="$_log || 'Nothing chosen yet'"></code>
</div>
```

A submenu opens to the inline end of its parent item and flips to the start when there is no room. Only one submenu per level is open at a time, and choosing a leaf closes every level at once. Submenus are view state: they raise no events and the server never hears about them.

### Placement

`placement` is the preferred side and alignment: `bottom-start`, `bottom`, `bottom-end`, `top-start`, `top` or `top-end`. The menu flips to the other side and shifts back into the viewport when there is no room.

```html preview
<div style="display: flex; gap: 12px; flex-wrap: wrap">
  <sb-dropdown label="Bottom end" placement="bottom-end" items='["Rename", "Duplicate", "-", "Delete"]'></sb-dropdown>
  <sb-dropdown label="Top start" placement="top-start" items='["Rename", "Duplicate", "-", "Delete"]'></sb-dropdown>
</div>
```

### Your own trigger

The `trigger` slot fills the trigger with your own content — text, an icon, an `sb-…` component. The button itself stays ours, so `aria-haspopup`, `aria-expanded` and the keyboard live in the shadow root and no morph can strip them. Keep the slot free of interactive elements (a button inside a button), and name the menu with `label`.

```html preview
<sb-dropdown label="More" placement="bottom-end" items='[{"value":"copy","label":"Copy link","icon":"🔗"},{"value":"share","label":"Share","icon":"📤"},{"divider":true},{"value":"remove","label":"Remove","danger":true}]'>
  <svg slot="trigger" width="16" height="4" viewBox="0 0 16 4" fill="currentColor" aria-hidden="true"><rect x="0" y="0" width="4" height="4"/><rect x="6" y="0" width="4" height="4"/><rect x="12" y="0" width="4" height="4"/></svg>
</sb-dropdown>
```

### Items as markup

Instead of `items`, write the menu as light DOM. The items are read **as data** (label, `value`, `disabled`, `data-icon`, `data-description`, `data-danger`, and `<hr>` for a divider) and rendered inside the menu, so the component never writes roles or `tabindex` into your markup, where the next morph would strip them. `items` wins whenever it is not empty.


```html preview
<div data-signals:_picked="''" style="display: grid; gap: 12px; justify-items: start">
  <sb-dropdown label="Crew" data-on:sb-select="$_picked = evt.detail.value">
    <button slot="item" value="ada" data-icon="👩‍🚀" data-description="Flight engineer">Ada</button>
    <button slot="item" value="yuri" data-icon="🧑‍🚀">Yuri</button>
    <hr slot="item">
    <button slot="item" value="eject" data-danger disabled>Eject</button>
  </sb-dropdown>
  <code data-text="$_picked || 'Nobody yet'"></code>
</div>
```

A slotted item opens a submenu with `data-children='[…]'`, the same JSON as `items`. Markup stops being the clearer form once a menu nests, so a deep tree belongs in `items`:

```html
<button slot="item" data-children='[{"value":"csv","label":"CSV"},{"value":"json","label":"JSON"}]'>Export</button>
```

### Server data

The menu is server data: a new `items` array replaces the whole tree whenever the server likes, and an open menu stays open — the open state is local and lives in a signal, so nothing about it is reset by a morph.

```html preview
<div data-signals="{_alt: false}" style="display: grid; gap: 12px; justify-items: start">
  <sb-dropdown label="Fleet" data-preserve-attr="items"
    data-attr:items="JSON.stringify($_alt ? ['Recall', 'Refit', '-', 'Decommission'] : ['Launch', 'Hold', '-', 'Scrub'])"></sb-dropdown>
  <button type="button" data-on:click="$_alt = !$_alt">Swap the items (try it while the menu is open)</button>
</div>
```

On a real page the same thing happens without the client signal: the server renders `<sb-dropdown items='…'>` again and the morph brings the new array. `data-preserve-attr` is only needed when a *signal* drives the attribute, as in the demo above.

## With commands

Give it a `name` and post the detail as it is:

```html
<sb-dropdown name="ship" label="Ship actions" items='[{"value":"refuel","label":"Refuel"}]'
  data-on:sb-select="@post('/cmd/ship', {payload: {tabid: $tabid, ...evt.detail}})"></sb-dropdown>
```

An item is an intent, not a value: there is no `confirm` and no `revert()`, and the component shows no result of its own. The server decides, and the page re-renders — see [Commands and components](/contribute#commands-and-components). When an item starts something slow, let the server render the pending state (a disabled item, a spinner in the page), never the menu.

## Open and closed

Opening and closing is local state. Clicks, the keyboard, an outside click and Escape only ever change a `$$` signal, so a server morph can never re-open a menu the user just closed.

The server still gets a say through the `open` attribute, with the usual rule:

- The first `open` sets the initial state — `<sb-dropdown open items='…'>` is open on the first paint, without the opening animation.
- A **changed** attribute wins over the local state: `open="false"` closes the menu, `open` re-opens it.
- Re-sent identical markup changes nothing, because the morph never touches an attribute it already agrees with. That is what makes the attribute safe to render on every frame.
- A **removed** attribute is ignored, like every other reflected attribute (morphs strip those). To close from the server, send `open="false"`.

From the client, use the property and the methods instead — they never touch the attribute:

```js
el.open          // true or false, live
el.show()        // open, and move the focus to the first item
el.hide()        // close, and leave the focus where it is
```

`sb-open` fires when it opens, `sb-close` with `{ reason }` (`item`, `escape`, `outside`, `scroll`, `tab`, `trigger`, `server` or `api`) when it closes.

```html preview
<div data-signals="{_state: 'closed'}" style="display: grid; gap: 12px; justify-items: start">
  <sb-dropdown label="Watch me" items='["Rename", "Duplicate", "-", "Delete"]'
    data-on:sb-open="$_state = 'open'"
    data-on:sb-close="$_state = 'closed (' + evt.detail.reason + ')'"></sb-dropdown>
  <code data-text="$_state"></code>
</div>
```

## Styling

Colours come from `--sb-control-bg`, `--sb-control-border`, `--sb-surface-raised`, `--sb-surface-hover`, `--sb-brand`, `--sb-text-muted` and `--sb-danger`; `--sb-notch: 0` rounds the pixel corners of trigger and menu. Parts: `trigger`, `menu` (every level) and `item`.

```css
sb-dropdown::part(trigger) { font-weight: 700; }
sb-dropdown::part(menu) { --sb-surface-raised: #1B1030; }
```

## Accessibility

It follows the WAI-ARIA menu button pattern:

- **Structure:** the trigger is a `button` with `aria-haspopup="menu"` and `aria-expanded`; the menu is a `menu` named by `label`, its rows are `menuitem`s, dividers are `separator`s and disabled items are `aria-disabled`.
- **Keys:** Enter, Space and Down open the menu at the first item, Up at the last one. Up and Down move, Home and End jump, typing a few letters jumps to a matching item (the buffer never leaks from one level into another). Enter and Space choose, Escape and Tab close and hand the focus back to the trigger.
- **Submenus:** Right (Left in a right-to-left page), Enter or Space on a parent opens its submenu and moves the focus to its first item; Left or Escape closes it again and puts the focus back on the parent item, so the keyboard walks in and out without ever leaving the menu. Escape at the root closes the whole thing. A parent item is a `menuitem` with `aria-haspopup="menu"` and `aria-expanded`, and its submenu is a `menu` named after it.
- **Pointer:** hovering a parent opens its submenu after a moment and leaving closes it a little later, so a diagonal path from the item into the submenu keeps it. Tapping a parent opens its submenu and tapping it again closes it, which is the only way back on a touch screen.
- **Focus:** real DOM focus moves onto the row inside the shadow root, so screen readers announce it and nothing in your markup is touched. When the server replaces the items while the menu is open, a row that is gone hands the focus to its neighbour, and the focus is only picked back up when it fell on the floor — see [Lists that hold the keyboard](/contribute#lists-that-hold-the-keyboard). A submenu whose parent is no longer a parent closes itself.
- **Pointer:** an outside click closes the menu, disabled items ignore clicks.
- **Motion:** the opening animation is skipped under `prefers-reduced-motion`, and for a menu that is already open on the first render.
