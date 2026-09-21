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

## Accessibility

The switch is a `<button role="switch">` with `aria-checked`, so it is focusable and toggles with Space and Enter. Give it a `label`. Without one it is announced as "Toggle".
