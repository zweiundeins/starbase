---
name: Flip Switch
tag: sb-flip-switch
category: forms
summary: A pixel on/off switch.
author: mbolli
tags: []
since: "2026-09-22"
preview: |
    <sb-flip-switch label="Warp drive"></sb-flip-switch>
    <sb-flip-switch label="Shields" checked></sb-flip-switch>
    <sb-flip-switch label="Self-destruct" disabled></sb-flip-switch>
---

## Examples

```html preview
<sb-flip-switch label="Warp drive"></sb-flip-switch>
<sb-flip-switch label="Shields" checked></sb-flip-switch>
<sb-flip-switch label="Self-destruct" disabled></sb-flip-switch>
```

```html preview
<sb-flip-switch size="sm" checked></sb-flip-switch>
<sb-flip-switch size="md" checked></sb-flip-switch>
<sb-flip-switch size="lg" checked></sb-flip-switch>
```

```html preview
<div data-signals:_thrusters="false">
  <sb-flip-switch label="Thrusters" data-bind:_thrusters__prop.checked__event.change></sb-flip-switch>
  <p data-text="$_thrusters ? 'Thrusters engaged' : 'Thrusters idle'"></p>
</div>
```
