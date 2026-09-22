---
name: Tabs
tag: sb-tabs
category: navigation
summary: Organize content with style.
author: zweiundeins
tags: [tablist, segmented, switcher]
since: 2026-09-21
preview: |
  <sb-tabs labels='["Home","Docs","API"]'></sb-tabs>
playground:
  props: {selected: {min: 0, max: 2}}
---

A segmented tab bar with keyboard support. Labels come from a JSON attribute. Each panel is a slotted element whose `slot` is the label in kebab-case.

## Examples

### With panels

```html preview
<sb-tabs labels='["Overview","Specs","Crew"]' style="inline-size: 100%">
  <p slot="overview">A light, fast shuttle built for orbital hops.</p>
  <p slot="specs">Payload 4t · Δv 9.4 km/s · 2 engines</p>
  <p slot="crew">Commander, pilot and one very good dog.</p>
</sb-tabs>
```

### Follow the selection

```html preview
<div data-signals:_tab="'Home'">
  <sb-tabs labels='["Home","Docs","API"]' data-on:sb-tab-change="$_tab = evt.detail.label"></sb-tabs>
  <p data-text="'Current: ' + $_tab"></p>
</div>
```

## With commands

Give it a `name`, and it emits `sb-change` with `{ name, value }` when a value is committed: ready to post as a command. With `confirm`, it sets `:state(pending)` until the server's re-rendered attribute matches, and `revert()` goes back to the server's value when a command is rejected. See [Commands and components](/contribute#commands-and-components) and the [Showcase](/showcase).

## Accessibility

It follows the WAI-ARIA tabs pattern: `role="tablist"`, `tab` and `tabpanel`, a roving `tabindex`, and arrow keys, Home and End for navigation.
