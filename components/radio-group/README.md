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

`sb-radio` is **not** a component of its own. Like `<option>` inside a native `<select>`, it is markup the group reads: value, label, `description` and `disabled`. The group renders every item in its own shadow root, so the repeated rows contain no custom elements, the roving focus never has to cross a shadow boundary, and a Datastar morph over the group can't trip over a nested upgrade.

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
  data-on:sb-change="@post('/cmd/flight', {payload: {tabid: $tabid, ...evt.detail}})"
  data-on:datastar-fetch="evt.detail.el === el && evt.detail.type === 'error' && el.revert()">
  <sb-radio value="impulse">Impulse</sb-radio>
  <sb-radio value="warp">Warp</sb-radio>
</sb-radio-group>
```

Style the wait from the page:

```css
sb-radio-group:state(pending) { opacity: 0.7; }
```

The `value` attribute is the server's value, and it wins whenever it changes: a morph with a new value replaces the local choice, re-sent identical markup leaves the user's choice alone, and a *removed* attribute is ignored (to clear, the server sends `value=""`). Nothing else is server state: the live value before the echo, the roving focus, the hover and the pending state all live in local signals and are never reflected to attributes. See [Commands and components](/contribute#commands-and-components) and the [Showcase](/showcase).

## Styling

Parts: `base`, `label`, `items`, `item`, `dot` and `description`. Colours come from the control and brand tokens, the corners from `--sb-notch`, so `[data-sb-style="smooth"]` rounds them instead.

```html preview
<sb-radio-group label="Hull paint" value="rust" style="--sb-brand: #F5C451; --sb-brand-subtle: rgb(245 196 81 / 0.16)">
  <sb-radio value="rust">Rust</sb-radio>
  <sb-radio value="chrome">Chrome</sb-radio>
</sb-radio-group>
```

## Accessibility

The items sit in a `role="radiogroup"` with `aria-orientation`, named by the `label` (without one it is announced as "Choice"). Each item is a `role="radio"` with `aria-checked`, and its description is part of its accessible name.

Only one item is in the tab order: the checked one, or the first enabled one when nothing is checked. Arrow keys move **and** select, as native radios do, wrapping around and skipping disabled items; Home and End jump to the first and last enabled choice; Space (and Enter) selects the focused one. The focus survives a re-render: the items are driven by signals, and the roving focus is restored after the rows change.
