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

The live state is the `checked` property, so `data-bind` works with the `__prop` and `__event` modifiers. Declare the signal first. If it can be true when the page loads and the checkbox sits in a region the server morphs, add `data-preserve-attr="checked"` (see sb-toggle's [Two-way binding](/components/toggle#two-way-binding)).

```html preview
<div data-signals:_beacon="false">
  <sb-checkbox label="Distress beacon" data-bind:_beacon__prop.checked__event.change></sb-checkbox>
  <p data-text="$_beacon ? 'Beacon on' : 'Beacon off'"></p>
</div>
```

## With commands

Give it a `name`, and it emits `sb-change` with `{ name, value }` when the user flips it (`value` is the checked state): ready to post as a command. With `confirm`, it sets `:state(pending)` until the server's re-rendered `checked` matches, and `revert()` goes back to the server's state when a command never reaches the server (an error status, or `retries-failed`).

```html
<sb-checkbox name="beacon" confirm label="Distress beacon"
  data-on:sb-change="@post('/cmd/beacon', {payload: {tabid: $tabid, ...evt.detail}})"
  data-on:datastar-fetch="evt.detail.el === el && ['error', 'retries-failed'].includes(evt.detail.type) && el.revert()"></sb-checkbox>
```

Style the wait from the page:

```css
sb-checkbox:state(pending) { opacity: 0.7; }
```

The server's word is the `checked` attribute, by presence, as on a native checkbox: adding it checks the box and removing it clears it, also after the user's click, while the same markup again leaves that click alone. In templ that is `checked?={ on }`. Any value means checked, `checked="false"` too, as in HTML: leave the attribute out for not checked. `indeterminate` is the server's too: a new one wins, a removed one clears the dash, and `revert()` brings back both. The live state before the echo and the pending state live in local signals and are never reflected to attributes. See [Commands and components](/contribute#commands-and-components) and the [Showcase](/showcase).

**Refusals without page code:** give it `rev`, a number the server bumps with every command it answers for this field, applied or refused (a counter in the tab's state will do; templ: `Rev: sb.Int(n)`). A new `rev` makes the server's `checked` win even when it is unchanged, so a refused click goes back and `:state(pending)` ends, without a `revert()` from the page; the same `rev` again changes nothing. When both change in one morph, the new state applies once. It also brings back the server's `indeterminate`, like `revert()`. The page's op ids (see [Commands and components](/contribute#commands-and-components)) do the same for every component; `rev` needs nothing on the page. `rev` only helps where the server renders the outcome into this element's own frame. A view rendered once for every tab, and an outcome no command records (a rate limit, a failed request, being offline), need `revert()` instead: see [When `rev` is enough](/contribute#when-rev-is-enough).

**Pending from the page:** `pending` is the page's word that a command is on its way. While the attribute is there, the component has `:state(pending)` and `aria-busy` (on its ElementInternals, so morphs keep them), whatever `confirm` says: `data-attr:pending="$_saving"`, with a signal the page sets when it sends and the server's answer clears. The two add up: removing `pending` leaves an unconfirmed edit pending, and `revert()` leaves the page's `pending`. In a region the server morphs, a signal-driven `pending` needs `data-preserve-attr="pending"`.

## Forms

Inside a `<form>`, `sb-checkbox` submits like a native checkbox: `name=value` while it is checked (`value` defaults to `on`), nothing when it isn't, nothing without a `name` or while `disabled`. The mixed state doesn't change what is submitted. `new FormData(form)` and Datastar's `contentType: 'form'` include it, and a form reset brings back the server's state without `change` events. It is not a form-associated element yet (Rocket can't declare one), so `required` and validity, `<fieldset disabled>`, `<label for>` and the `form` attribute don't reach it. With commands, `sb-change` carries `{ name, value }` (see [With commands](#with-commands)).

## Styling

Parts: `base` (the clickable row), `box` and `label`. A checked box is `--sb-brand` in a `--sb-brand-light` frame with a `--sb-text-on-brand` mark; an empty one is `--sb-control-bg` in `--sb-control-border` (`--sb-control-border-hover` under the pointer, on a `--sb-surface-hover` row). The label is `--sb-control-text` at 0.875rem in the page's font. The corners come from `--sb-notch`, so `[data-sb-style="smooth"]` rounds them instead. A disabled checkbox has `:state(disabled)`, from the decoded prop, never from `:host([disabled])`.

```html preview
<sb-checkbox label="Gold plating" checked style="--sb-brand: #F5C451; --sb-brand-light: #FFE08A; --sb-text-on-brand: #1B1300"></sb-checkbox>
```

## Accessibility

The row is a `role="checkbox"` with `aria-checked` (`mixed` for the dash) and one tab stop; Space checks and clears it. A disabled checkbox has `aria-disabled` and leaves the tab order. The `label` names it. Without one, give `<sb-checkbox>` an `aria-label`: the box takes it over (`aria-labelledby` doesn't reach inside). In forced colours (Windows High Contrast) the checked box uses the system's `Highlight` colour and the mark `HighlightText`.
