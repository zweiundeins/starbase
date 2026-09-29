---
name: Checkbox Group
tag: sb-checkbox-group
category: forms
summary: Several choices with pixel checkboxes, one array value and an optional select-all.
author: zweiundeins
tags: [checkbox, choice, form, multi-select]
since: 2026-09-29
preview: |
  <sb-checkbox-group value='["shields","sensors"]' orientation="horizontal">
    <sb-check value="shields">Shields</sb-check>
    <sb-check value="sensors">Sensors</sb-check>
    <sb-check value="cloak">Cloak</sb-check>
  </sb-checkbox-group>
playground:
  values: {label: Systems online, selectAll: All systems}
  attrs: {value: '["shields"]'}
  content: <sb-check value="shields">Shields</sb-check><sb-check value="sensors" description="Long range only">Sensors</sb-check><sb-check value="cloak" disabled>Cloak</sb-check>
  style: "inline-size: min(100%, 20rem)"
---

Several choices under one name, where each can be on or off. Write the choices as `sb-check` children, or let the server fill `options`. The group keeps one `value`, the array of checked values, emits one `sb-change` when the user checks or clears a choice, and follows the command contract, so it can drive a command as it is.

`sb-check` is **not** a component of its own. Like `<option>` inside a native `<select>`, it is markup the group reads: value, label, `description` and `disabled`. The value falls back to the label, then to the text, and a choice needs a non-empty value. The group renders every item in its own shadow root, so the repeated rows contain no custom elements and a Datastar morph over the group can't trip over a nested upgrade. For a single yes or no, use [`sb-checkbox`](/components/checkbox).

## Examples

### Basic

```html preview
<sb-checkbox-group label="Systems online" value='["shields","sensors"]'>
  <sb-check value="shields">Shields</sb-check>
  <sb-check value="sensors">Sensors</sb-check>
  <sb-check value="cloak" disabled>Cloak (in repair)</sb-check>
</sb-checkbox-group>
```

### Select all

`select-all` adds a leading checkbox that checks every enabled choice, or clears them all when they are all checked. It shows a dash while only some are. Its text is the attribute's value, "All" when it is empty. Disabled choices keep their state.

```html preview
<sb-checkbox-group label="Cargo to unload" select-all="Every bay" value='["b"]'>
  <sb-check value="a">Bay A</sb-check>
  <sb-check value="b">Bay B</sb-check>
  <sb-check value="c">Bay C</sb-check>
</sb-checkbox-group>
```

### Descriptions and a row

`description` adds a second line, and `orientation="horizontal"` lays the choices out in a row that wraps.

```html preview
<div style="display: grid; gap: 24px">
  <sb-checkbox-group label="Crew alerts" value='["hull"]'>
    <sb-check value="hull" description="Breaches and pressure drops">Hull</sb-check>
    <sb-check value="fuel" description="Below a quarter tank">Fuel</sb-check>
    <sb-check value="mail" description="Messages from home">Mail</sb-check>
  </sb-checkbox-group>
  <sb-checkbox-group label="Rations" orientation="horizontal" select-all>
    <sb-check value="soup">Soup</sb-check>
    <sb-check value="bread">Bread</sb-check>
    <sb-check value="tea">Tea</sb-check>
  </sb-checkbox-group>
</div>
```

### Choices from the server

`options` takes the same shape as `sb-select` and `sb-radio-group`: strings, or `{value, label, description?, disabled?}`. It is server data, so it only flows in. Children win over it when both are there.

```html preview
<sb-checkbox-group label="Docks" value='["b"]'
  options='[{"value":"a","label":"Dock A"},{"value":"b","label":"Dock B","description":"Closest to the lift"},{"value":"c","label":"Dock C","disabled":true}]'></sb-checkbox-group>
```

### In a signal

The live value is the `value` property, an array. Keep it in a signal with `sb-change`: `data-bind` treats array signals as native checkbox groups, which is a different thing.

```html preview
<div data-signals="{_crew: ['pilot']}">
  <sb-checkbox-group label="Crew on the bridge" value='["pilot"]'
    data-on:sb-change="$_crew = evt.detail.value">
    <sb-check value="pilot">Pilot</sb-check>
    <sb-check value="navigator">Navigator</sb-check>
    <sb-check value="engineer">Engineer</sb-check>
  </sb-checkbox-group>
  <p>On duty: <strong data-text="$_crew.join(', ') || 'nobody'"></strong></p>
</div>
```

## With commands

Give it a `name`, and it emits `sb-change` with `{ name, value }` when the user checks or clears a choice, with the whole array as `value`: one command for the whole decision. With `confirm`, it sets `:state(pending)` until the server's re-rendered `value` attribute matches (the order doesn't matter), and `revert()` goes back to the server's value when a command is rejected.

```html
<sb-checkbox-group name="alerts" confirm value='["hull"]' label="Crew alerts"
  data-on:sb-change="@post('/cmd/alerts', {payload: {tabid: $tabid, ...evt.detail}})"
  data-on:datastar-fetch="evt.detail.el === el && evt.detail.type === 'error' && el.revert()">
  <sb-check value="hull">Hull</sb-check>
  <sb-check value="fuel">Fuel</sb-check>
</sb-checkbox-group>
```

Style the wait from the page:

```css
sb-checkbox-group:state(pending) { opacity: 0.7; }
```

The `value` attribute is the server's value, a JSON array, and it wins whenever it changes: a morph with a new value replaces the local checks, re-sent identical markup leaves the user's checks alone, and a *removed* attribute is ignored (to clear, the server sends `value="[]"`, which also works on a group that had no `value` before). A new value keeps the choices' order; a value the choices don't offer stays in it. The live value before the echo, the focus, the select-all state and the pending state all live in local signals and are never reflected to attributes. See [Commands and components](/contribute#commands-and-components) and the [Showcase](/showcase).

## Forms

Inside a `<form>`, `sb-checkbox-group` submits like native checkboxes that share its `name`: one `name=value` entry per checked choice, in their order, and none when nothing is checked. A disabled choice is not submitted, even when it is checked, and nothing is without a `name` or while the group is `disabled`. `new FormData(form)` and Datastar's `contentType: 'form'` include it, and a form reset brings back the server's value without `change` events. It is not a form-associated element yet (Rocket can't declare one), so `required` and validity, `<fieldset disabled>`, `<label for>` and the `form` attribute don't reach it. With commands, `sb-change` carries `{ name, value }` (see [With commands](#with-commands)).

## Styling

Parts: `base`, `label`, `items`, `item` (every row, the select-all one included), `all` (the select-all row), `box` and `description`. A checked box is `--sb-brand` in a `--sb-brand-light` frame with a `--sb-text-on-brand` mark; an empty one is `--sb-control-bg` in `--sb-control-border` (`--sb-control-border-hover` under the pointer, on a `--sb-surface-hover` row). The group label is `--sb-text-2`, descriptions are `--sb-text-muted`. The corners come from `--sb-notch`, so `[data-sb-style="smooth"]` rounds them instead. A disabled group has `:state(disabled)`, from the decoded prop, so `disabled="false"` is not disabled.

```html preview
<style>
  .gold { --sb-brand: #F5C451; --sb-brand-light: #FFE08A; --sb-text-on-brand: #1B1300; }
  .gold::part(all) { font-weight: 600; }
</style>
<sb-checkbox-group class="gold" label="Hull paint" select-all value='["rust"]'>
  <sb-check value="rust">Rust</sb-check>
  <sb-check value="chrome">Chrome</sb-check>
</sb-checkbox-group>
```

## Accessibility

The choices sit in a `role="group"` named by the `label` (`aria-labelledby`). Without one, give `<sb-checkbox-group>` an `aria-label`: the group takes it over. Each choice is a `role="checkbox"` with `aria-checked`, and its description is part of its accessible name. The select-all box is a checkbox too, `aria-checked="mixed"` while only some choices are checked.

Like native checkboxes, every enabled choice is its own tab stop, and Space checks or clears the focused one; there is no arrow-key navigation. Disabled choices have `aria-disabled` and leave the tab order. The focus survives a re-render: the items are driven by signals, and when the focused choice is dropped or disabled, its neighbour takes the focus. Once the focus has left the group, a click on the page's text included, the group doesn't take it back.

In forced colours (Windows High Contrast), checked boxes use the system's `Highlight` colour and their marks `HighlightText`.
