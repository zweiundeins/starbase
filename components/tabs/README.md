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

A segmented tab bar with keyboard support. Labels come from a JSON attribute. Each panel is a slotted element whose `slot` is its label lowercased, with a hyphen for every run of anything but letters and digits: `slot="docs"` for "Docs", `slot="self-host"` for "Self-host", `slot="übersicht"` for "Übersicht". When two labels give the same slot ("C++" and "C#" both give `c`), the later one gets its index appended: `c-1`.

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

The live index is the `selected` property, and `input` fires on every move (a click or an arrow key), so `data-bind` follows it with the `__prop` modifier. Declare the signal first; setting it selects a tab.

```html preview
<div data-signals:_tab="1">
  <sb-tabs labels='["Home","Docs","API"]' data-bind:_tab__prop.selected></sb-tabs>
  <p>Selected index: <strong data-text="$_tab"></strong></p>
</div>
```

## With commands

Give it a `name`, and it emits `sb-change` with `{ name, value, label }` (`value` is the index) when the selection is committed: ready to post as a command. A click commits at once; the arrow keys commit after a 250 ms pause, so running through the tabs sends one command, not one per tab (whose answers would pull the selection back one by one). With `confirm`, it sets `:state(pending)` until the server's re-rendered `selected` attribute matches, and `revert()` goes back to the server's value when a command is rejected. The panels are light DOM, so events from their content (a field's `change`, another component's `sb-change`) bubble through `sb-tabs` too: check `evt.target === el`. See [Commands and components](/contribute#commands-and-components) and the [Showcase](/showcase).

## Styling

Style it from your page's CSS — no need to change the component or import anything into it. Custom properties, inherited properties and `::part()` all reach into its shadow root.

- **Fonts:** the tab labels use your page's font.
- **Colours:** the strip is `--sb-surface-inset` with a `--sb-border` edge. Tabs are `--sb-text-2` (`--sb-text-1` on hover) with a `--sb-border` outline; the selected tab fills with `--sb-brand-light` and writes in `--sb-bg`. The focus ring is `--sb-focus-ring`, corners `--sb-control-radius`. When the tabs don't fit, the strip scrolls sideways with a thin `--sb-border` scrollbar, and the selected tab scrolls into view.
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

It follows the WAI-ARIA tabs pattern: `role="tablist"`, `tab` and `tabpanel`, a roving `tabindex`, and arrow keys, Home and End for navigation; in a right-to-left layout the arrows follow the mirrored strip. Name the tab list with `aria-label` on `sb-tabs`: it is forwarded to the tablist inside (the site's own install tabs say `aria-label="Installation method"`). In forced-colors mode (Windows High Contrast) the selected tab is drawn in `Highlight` and the focus as an outline.
