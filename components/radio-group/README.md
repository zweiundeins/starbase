---
name: Radio Group
tag: sb-radio-group
category: forms
summary: A grouped single choice with pixel radios and one value.
author: zweiundeins
tags: [radio, choice, form, single-select]
since: 2026-09-23
preview: |
  <sb-radio-group value="warp" orientation="horizontal">
    <sb-radio value="impulse">Impulse</sb-radio>
    <sb-radio value="warp">Warp</sb-radio>
  </sb-radio-group>
playground:
  values: {label: Drive mode, value: warp}
  content: <sb-radio value="impulse">Impulse</sb-radio><sb-radio value="warp" description="Faster than light">Warp</sb-radio><sb-radio value="tow" disabled>Tow</sb-radio>
  style: "inline-size: min(100%, 20rem)"
---

One choice out of a handful, the form primitive a `<select>` is too small for. Write the choices as `sb-radio` children, or let the server fill `options`. The group keeps a single `value`, emits `sb-change` when the user commits a choice, and follows the command contract, so it can drive a command as it is.

`sb-radio` is **not** a component of its own. Like `<option>` inside a native `<select>`, it is markup the group reads: value, label, `description` and `disabled`. The value falls back to the label, then to the text. A choice needs a non-empty value: `""` is the group's "nothing chosen", so `<sb-radio value="">Any</sb-radio>` is left out (give "Any" a value of its own, e.g. `value="any"`). The group renders every item in its own shadow root, so the repeated rows contain no custom elements, the roving focus never has to cross a shadow boundary, and a Datastar morph over the group can't trip over a nested upgrade.

## Examples

### Basic

```html preview
<sb-radio-group label="Drive mode" value="warp">
  <sb-radio value="impulse">Impulse</sb-radio>
  <sb-radio value="warp">Warp</sb-radio>
  <sb-radio value="tow" disabled>Tow (needs a tug)</sb-radio>
</sb-radio-group>
```

### Descriptions and a row

`description` adds a second line, and `orientation="horizontal"` lays the choices out in a row that wraps.

```html preview
<div style="display: grid; gap: 24px">
  <sb-radio-group label="Shield profile" value="balanced">
    <sb-radio value="balanced" description="Even coverage, no surprises">Balanced</sb-radio>
    <sb-radio value="forward" description="Everything to the bow">Forward</sb-radio>
    <sb-radio value="off" description="For when the sensors need silence">Off</sb-radio>
  </sb-radio-group>
  <sb-radio-group label="Rations" orientation="horizontal" value="double">
    <sb-radio value="single">Single</sb-radio>
    <sb-radio value="double">Double</sb-radio>
    <sb-radio value="feast">Feast</sb-radio>
  </sb-radio-group>
</div>
```

### Choices from the server

`options` takes the same shape as `sb-select`: strings, or `{value, label, description?, disabled?}`. It is server data, so it only flows in. Children win over it when both are there.

```html preview
<sb-radio-group label="Dock" value="b"
  options='[{"value":"a","label":"Bay A"},{"value":"b","label":"Bay B","description":"Closest to the lift"},{"value":"c","label":"Bay C","disabled":true}]'></sb-radio-group>
```

### Two-way binding

The live value is the `value` property, so `data-bind` works with the `__prop` and `__event` modifiers. Declare the signal first.

```html preview
<div data-signals:_drive="'impulse'">
  <sb-radio-group label="Drive mode" data-bind:_drive__prop.value__event.change>
    <sb-radio value="impulse">Impulse</sb-radio>
    <sb-radio value="warp">Warp</sb-radio>
  </sb-radio-group>
  <p>Engaged: <strong data-text="$_drive"></strong></p>
</div>
```

## With commands

Give it a `name`, and it emits `sb-change` with `{ name, value }` when the user commits a choice: ready to post as a command. With `confirm`, it sets `:state(pending)` until the server's re-rendered `value` attribute matches, and `revert()` goes back to the server's value when a command is rejected.

```html
<sb-radio-group name="drive" confirm value="impulse" label="Drive mode"
  data-on:sb-change="@post('/cmd/drive', {payload: {tabid: $tabid, ...evt.detail}})"
  data-on:datastar-fetch="evt.detail.el === el && evt.detail.type === 'error' && el.revert()">
  <sb-radio value="impulse">Impulse</sb-radio>
  <sb-radio value="warp">Warp</sb-radio>
</sb-radio-group>
```

Style the wait from the page:

```css
sb-radio-group:state(pending) { opacity: 0.7; }
```

The `value` attribute is the server's value, and it wins whenever it changes: a morph with a new value replaces the local choice, re-sent identical markup leaves the user's choice alone, and a *removed* attribute is ignored (to clear, the server sends `value=""`, which also works on a group that had no `value` before). Nothing else is server state: the live value before the echo, the roving focus, the hover and the pending state all live in local signals and are never reflected to attributes. See [Commands and components](/contribute#commands-and-components) and the [Showcase](/showcase).

## Forms

Inside a `<form>`, `sb-radio-group` submits the checked choice under its `name` (`name=value`), `new FormData(form)` and Datastar's `contentType: 'form'` include it, and a form reset brings back the server's value without `change` events. As with radios, nothing is submitted when no choice is checked or the checked one is disabled, and nothing without a `name` or while the group is `disabled`. It is not a form-associated element yet (Rocket can't declare one), so `required` and validity, `<fieldset disabled>`, `<label for>` and the `form` attribute don't reach it. With commands, `sb-change` carries `{ name, value }` (see [With commands](#with-commands)).

## Styling

Parts: `base`, `label`, `items`, `item`, `dot` and `description`. Colours come from the control and brand tokens (a checked choice is a `--sb-brand` dot in a `--sb-brand-light` ring, on a `--sb-brand-subtle` tint), the corners from `--sb-notch`, so `[data-sb-style="smooth"]` rounds them instead. A disabled group has `:state(disabled)`, from the decoded prop, so `disabled="false"` is not disabled.

```html preview
<sb-radio-group label="Hull paint" value="rust" style="--sb-brand: #F5C451; --sb-brand-light: #FFE08A; --sb-brand-subtle: rgb(245 196 81 / 0.16)">
  <sb-radio value="rust">Rust</sb-radio>
  <sb-radio value="chrome">Chrome</sb-radio>
</sb-radio-group>
```

## Accessibility

The element itself is the `radiogroup`, with `aria-orientation` (and `aria-disabled` when disabled), set through `ElementInternals` so a morph can't strip them. The `label` names it. Without one, give the element an `aria-label` or `aria-labelledby`: they name the group too, and win over the `label`. Each item is a `role="radio"` with `aria-checked`, and its description is part of its accessible name.

Only one item is in the tab order: the checked one, or the first enabled one when nothing is checked. Disabled items don't take the focus, not even from a click. Arrow keys move **and** select, as native radios do, wrapping around and skipping disabled items. Left and Right follow the reading direction (in right-to-left text, Left moves on), and arrows with Alt, Ctrl or Meta are left to the browser (Alt+Left goes back). Home and End jump to the first and last enabled choice; Space (and Enter) selects the focused one. The focus survives a re-render: the items are driven by signals, and when the focused choice is dropped or disabled, its neighbour takes the focus. Once the focus has left the group, a click on the page's text included, the group doesn't take it back.

In forced colours (Windows High Contrast), the checked dot is filled with the system's `Highlight` colour.
