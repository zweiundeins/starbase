---
name: Toggle
tag: sb-toggle
category: forms
summary: A switch component with smooth Datastar interactions.
author: zweiundeins
tags: [switch, checkbox, boolean]
since: 2026-09-21
preview: |
  <sb-toggle checked size="lg" aria-label="Warp drive"></sb-toggle>
usage: |
  <sb-toggle label="Notifications"></sb-toggle>
playground:
  values: {label: Warp drive, checked: true}
---

An on/off switch with chunky pixel corners and stepped motion. It exposes a `checked` property and fires `change`, so Datastar can bind to it like a native checkbox.

## Examples

### Basic

```html preview
<sb-toggle label="Warp drive"></sb-toggle>
<sb-toggle label="Shields" checked></sb-toggle>
<sb-toggle label="Self-destruct" disabled></sb-toggle>
```

### Sizes

```html preview
<sb-toggle size="sm" checked aria-label="Small"></sb-toggle>
<sb-toggle size="md" checked aria-label="Medium"></sb-toggle>
<sb-toggle size="lg" checked aria-label="Large"></sb-toggle>
```

### Two-way binding

Bind the `checked` property with `data-bind` and the `__prop` and `__event` modifiers. If the signal can be true when the page loads and the toggle sits in a region the server morphs, add `data-preserve-attr="checked"`: a write made before the component is defined shows up as a `checked` attribute, and the next morph would remove it, which switches the toggle off.

```html preview
<div data-signals:_thrusters="false">
  <sb-toggle label="Thrusters" data-bind:_thrusters__prop.checked__event.change></sb-toggle>
  <p data-text="$_thrusters ? 'Thrusters engaged' : 'Thrusters idle'"></p>
</div>
```

## With commands

Give it a `name`, and it emits `sb-change` with `{ name, value }` when a value is committed: ready to post as a command. With `confirm`, it sets `:state(pending)` until the server's re-rendered attribute matches, and `revert()` goes back to the server's value when a command never reaches the server (an error status, or `retries-failed`). See [Commands and components](/contribute#commands-and-components) and the [Showcase](/showcase).

The server's word is the `checked` attribute, by presence, as on a native checkbox: adding it switches the toggle on and removing it switches it off, also after the user flipped it, while the same markup again leaves the user's switch alone. In templ that is `checked?={ on }`. `checked="false"` still reads as off, with a console warning: that spelling is deprecated and goes in a later release, so leave the attribute out.

## Forms

Inside a `<form>`, `sb-toggle` submits like a checkbox: `name=on` when it is on, nothing when it is off or `disabled`, and nothing without a `name`. `new FormData(form)` and Datastar's `contentType: 'form'` include it, and a form reset brings back the server's `checked` without firing `change` or `sb-change`. It is not a form-associated element yet (Rocket can't declare one), so `required` and validity, `<fieldset disabled>`, `<label for>` and the `form` attribute don't reach it. With commands, `sb-change` carries `{ name, value }` (see [With commands](#with-commands)).

## Styling

Style it from your page's CSS, without changing the component or importing anything into it. Custom properties, inherited properties and `::part()` all reach into its shadow root.

- **Size:** `size` (`sm`, `md`, `lg`) sets the switch.
- **Fonts:** the label uses your page's font, size and colour.
- **Colours:** the track is `--sb-border-strong` when off and `--sb-brand` when on; the knob is `--sb-toggle-knob` (default `--sb-text-1`, and `--sb-text-on-brand` on the brand track), the focus ring `--sb-brand-light`. `--sb-notch: 0` rounds the switch instead of notching it, and its knob and colour move smoothly instead of in steps.
- **Parts:** `switch` (the track) and `label`. The knob has no part: colour it with `--sb-toggle-knob`. Your page's `::part()` rules win over the component's own, without `!important`.

```html preview
<style>
  .my-toggle { --sb-brand: #16A34A; --sb-notch: 0; --sb-toggle-knob: #FFFFFF; }
  .my-toggle::part(label) { color: var(--sb-text-2); font-size: 0.875rem; }
</style>
<sb-toggle class="my-toggle" label="Autopilot" checked size="lg"></sb-toggle>
```

## Accessibility

The switch is a `<button role="switch">` with `aria-checked`, so it is focusable and toggles with Space and Enter. Give it a `label`, or an `aria-label` on `<sb-toggle>` when the name shouldn't show: the switch takes it over (`aria-labelledby` doesn't reach inside). Without either it is announced as "Toggle". In forced colours (Windows High Contrast) the track is outlined and the on state uses the system highlight colours.
