---
name: Checkbox
tag: sb-checkbox
category: forms
summary: A pixel checkbox with a label, a mixed state and the command contract.
author: zweiundeins
tags: [checkbox, boolean, form, indeterminate]
since: 2026-09-29
preview: |
  <sb-checkbox label="Shields" checked></sb-checkbox>
usage: |
  <sb-checkbox name="shields" label="Shields"></sb-checkbox>
playground:
  values: {label: Dock at the station, checked: true}
---

A checkbox with a pixel box and a label, for a yes or no that belongs in a form. It keeps a local `checked`, emits `sb-change` when the user flips it, and follows the command contract, so it can drive a command as it is. For several choices under one name, use [`sb-checkbox-group`](/components/checkbox-group).

## Examples

### Basic

```html preview
<div style="display: grid; justify-items: start">
  <sb-checkbox label="Shields" checked></sb-checkbox>
  <sb-checkbox label="Cloaking device"></sb-checkbox>
  <sb-checkbox label="Self-destruct" disabled></sb-checkbox>
</div>
```

### Mixed

`indeterminate` shows a dash: the server says "some", for example when some of the items below are checked and others aren't. It is the server's view state, not a value. A click resolves it to checked, and the server's next `indeterminate` wins again.

```html preview
<sb-checkbox label="All cargo bays" indeterminate></sb-checkbox>
```

### Two-way binding

The live state is the `checked` property, so `data-bind` works with the `__prop` and `__event` modifiers. Declare the signal first.

```html preview
<div data-signals:_beacon="false">
  <sb-checkbox label="Distress beacon" data-bind:_beacon__prop.checked__event.change></sb-checkbox>
  <p data-text="$_beacon ? 'Beacon on' : 'Beacon off'"></p>
</div>
```

## With commands

Give it a `name`, and it emits `sb-change` with `{ name, value }` when the user flips it (`value` is the checked state): ready to post as a command. With `confirm`, it sets `:state(pending)` until the server's re-rendered `checked` matches, and `revert()` goes back to the server's state when a command is rejected.

```html
<sb-checkbox name="beacon" confirm label="Distress beacon"
  data-on:sb-change="@post('/cmd/beacon', {payload: {tabid: $tabid, ...evt.detail}})"
  data-on:datastar-fetch="evt.detail.el === el && evt.detail.type === 'error' && el.revert()"></sb-checkbox>
```

Style the wait from the page:

```css
sb-checkbox:state(pending) { opacity: 0.7; }
```

The server's `checked` wins when it sends a new one, even onto a checkbox rendered without it; the same markup again leaves the user's click alone. A removed attribute changes nothing, so send `checked="false"` to clear it. `indeterminate` is the server's too: a new one wins, a removed one clears the dash, and `revert()` brings back both. Write booleans as `checked`, `checked="true"` or `checked="false"`. The live state before the echo and the pending state live in local signals and are never reflected to attributes. See [Commands and components](/contribute#commands-and-components) and the [Showcase](/showcase).

## Forms

Inside a `<form>`, `sb-checkbox` submits like a native checkbox: `name=value` while it is checked (`value` defaults to `on`), nothing when it isn't, nothing without a `name` or while `disabled`. The mixed state doesn't change what is submitted. `new FormData(form)` and Datastar's `contentType: 'form'` include it, and a form reset brings back the server's state without `change` events. It is not a form-associated element yet (Rocket can't declare one), so `required` and validity, `<fieldset disabled>`, `<label for>` and the `form` attribute don't reach it. With commands, `sb-change` carries `{ name, value }` (see [With commands](#with-commands)).

## Styling

Parts: `base` (the clickable row), `box` and `label`. A checked box is `--sb-brand` in a `--sb-brand-light` frame with a `--sb-text-on-brand` mark; an empty one is `--sb-control-bg` in `--sb-control-border` (`--sb-control-border-hover` under the pointer, on a `--sb-surface-hover` row). The label is `--sb-control-text` at 0.875rem in the page's font. The corners come from `--sb-notch`, so `[data-sb-style="smooth"]` rounds them instead. A disabled checkbox has `:state(disabled)`, from the decoded prop, so `disabled="false"` is not disabled.

```html preview
<sb-checkbox label="Gold plating" checked style="--sb-brand: #F5C451; --sb-brand-light: #FFE08A; --sb-text-on-brand: #1B1300"></sb-checkbox>
```

## Accessibility

The row is a `role="checkbox"` with `aria-checked` (`mixed` for the dash) and one tab stop; Space checks and clears it. A disabled checkbox has `aria-disabled` and leaves the tab order. The `label` names it. Without one, give `<sb-checkbox>` an `aria-label`: the box takes it over (`aria-labelledby` doesn't reach inside). In forced colours (Windows High Contrast) the checked box uses the system's `Highlight` colour and the mark `HighlightText`.
