---
name: Tooltip
tag: sb-tooltip
category: feedback
summary: Helpful hints, right where you need them.
author: zweiundeins
tags: [hint, popover, help]
since: 2026-09-21
preview: |
  <sb-tooltip content="More info" open><img src="/art/info.svg" alt="Info" width="32" height="32" style="image-rendering: pixelated"></sb-tooltip>
usage: |
  <sb-tooltip content="More info"><button type="button">?</button></sb-tooltip>
playground:
  content: <sb-button variant="outline" size="sm">Hover me</sb-button>
  values: {content: More info, open: true}
---

A short hint that appears on hover and keyboard focus, with a small pixel arrow. Wrap any trigger element.

## Examples

### Placement

```html preview
<div style="display: flex; gap: 24px; padding-block: 48px">
  <sb-tooltip content="Top" placement="top"><sb-button variant="outline" size="sm">Top</sb-button></sb-tooltip>
  <sb-tooltip content="Bottom" placement="bottom"><sb-button variant="outline" size="sm">Bottom</sb-button></sb-tooltip>
  <sb-tooltip content="Left" placement="left"><sb-button variant="outline" size="sm">Left</sb-button></sb-tooltip>
  <sb-tooltip content="Right" placement="right"><sb-button variant="outline" size="sm">Right</sb-button></sb-tooltip>
</div>
```

### Dynamic content

`content` is a normal attribute, so `data-attr` keeps it live.

```html preview
<div data-signals:_fuel="72" style="padding-block-start: 48px">
  <sb-tooltip data-attr:content="'Fuel: ' + $_fuel + '%'">
    <sb-button size="sm" data-on:click="$_fuel = Math.max(0, $_fuel - 9)">Burn fuel</sb-button>
  </sb-tooltip>
</div>
```

## Styling

Style it from your page's CSS — no need to change the component or import anything into it. Custom properties, inherited properties and `::part()` all reach into its shadow root.

- **Size:** the tip is at most `16rem` wide; set `max-inline-size` on `::part(tip)` for another limit.
- **Fonts:** the tip uses your page's font.
- **Colours:** the tip is `--sb-surface-raised` with `--sb-text-1` text, and `--sb-border-strong` draws its edge and the arrow. These tokens reach your trigger too, so set the text colour on the part rather than the token.
- **Parts:** `tip`. Your page's `::part()` rules win over the component's own, without `!important`.

```html preview
<style>
  .my-tip { --sb-surface-raised: var(--sb-brand); --sb-border-strong: var(--sb-brand); }
  .my-tip::part(tip) { color: var(--sb-text-on-brand); font-weight: 400; max-inline-size: 12rem; }
</style>
<div style="padding-block-end: 4rem">
  <sb-tooltip class="my-tip" open placement="bottom" content="Fuel is topped up before every launch."><sb-button size="sm" variant="outline">Fuel</sb-button></sb-tooltip>
</div>
```

## Accessibility

The tooltip shows on focus as well as hover, and Escape dismisses it. Keep tooltips short and supplementary: never put the only copy of important information in one.
