---
name: Toggle
tag: sb-toggle
category: forms
summary: A switch component with smooth Datastar interactions.
author: zweiundeins
tags: [switch, checkbox, boolean]
since: 2026-09-21
preview: |
  <sb-toggle checked size="lg"></sb-toggle>
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
<sb-toggle size="sm" checked></sb-toggle>
<sb-toggle size="md" checked></sb-toggle>
<sb-toggle size="lg" checked></sb-toggle>
```

### Two-way binding

Bind the `checked` property with `data-bind` and the `__prop` and `__event` modifiers.

```html preview
<div data-signals:_thrusters="false">
  <sb-toggle label="Thrusters" data-bind:_thrusters__prop.checked__event.change></sb-toggle>
  <p data-text="$_thrusters ? 'Thrusters engaged' : 'Thrusters idle'"></p>
</div>
```

## With commands

Give it a `name`, and it emits `sb-change` with `{ name, value }` when a value is committed: ready to post as a command. With `confirm`, it sets `:state(pending)` until the server's re-rendered attribute matches, and `revert()` goes back to the server's value when a command is rejected. See [Commands and components](/contribute#commands-and-components) and the [Showcase](/showcase).

## Styling

Style it from your page's CSS — no need to change the component or import anything into it. Custom properties, inherited properties and `::part()` all reach into its shadow root.

- **Size:** `size` (`sm`, `md`, `lg`) sets the switch.
- **Fonts:** the label uses your page's font, size and colour.
- **Colours:** the track is `--sb-border-strong` when off and `--sb-brand` when on; the knob is `--sb-toggle-knob` (default `--sb-text-1`, and `--sb-text-on-brand` on the brand track), the focus ring `--sb-brand-light`. `--sb-notch: 0` rounds the switch instead of notching it.
- **Parts:** `switch` (the track) and `label`. The knob has no part: colour it with `--sb-toggle-knob`. Your page's `::part()` rules win over the component's own, without `!important`.

```html preview
<style>
  .my-toggle { --sb-brand: #16A34A; --sb-notch: 0; --sb-toggle-knob: #FFFFFF; }
  .my-toggle::part(label) { color: var(--sb-text-2); font-size: 0.875rem; }
</style>
<sb-toggle class="my-toggle" label="Autopilot" checked size="lg"></sb-toggle>
```

## Accessibility

The switch is a `<button role="switch">` with `aria-checked`, so it is focusable and toggles with Space and Enter. Give it a `label`. Without one it is announced as "Toggle". In forced colours (Windows High Contrast) the track is outlined and the on state uses the system highlight colours.
