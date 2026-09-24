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

## Styling

Style it from your page's CSS — no need to change the component or import anything into it. Custom properties, inherited properties and `::part()` all reach into its shadow root.

- **Fonts:** the tab labels use your page's font.
- **Colours:** the strip is `--sb-surface-inset` with a `--sb-border` edge. Tabs are `--sb-text-2` (`--sb-text-1` on hover) with a `--sb-border` outline; the selected tab fills with `--sb-brand-light` and writes in `--sb-bg`. The focus ring is `--sb-focus-ring`, corners `--sb-control-radius`.
- **Parts:** `tablist` (the strip), `tab` (every tab) and `panel` (every panel). The selected tab is also `selected`, so `::part(tab selected)` styles only that one; it moves with the selection. Your page's `::part()` rules win over the component's own, without `!important`.

```html preview
<style>
  .my-tabs { --sb-brand-light: var(--sb-accent); --sb-control-radius: 0; }
  .my-tabs::part(tab) { padding: 0.35rem 0.75rem; font-size: 0.8125rem; }
  .my-tabs::part(tab selected) { text-decoration: underline; }
</style>
<sb-tabs class="my-tabs" labels='["Orbit","Crew"]'>
  <p slot="orbit">Low Earth orbit, 420 km.</p>
  <p slot="crew">Commander, pilot and one very good dog.</p>
</sb-tabs>
```

## Accessibility

It follows the WAI-ARIA tabs pattern: `role="tablist"`, `tab` and `tabpanel`, a roving `tabindex`, and arrow keys, Home and End for navigation.
