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
  <sb-tooltip content="More info"><button type="button" aria-label="Help">?</button></sb-tooltip>
playground:
  content: <sb-button variant="outline" size="sm">Hover me</sb-button>
  values: {content: More info, open: true}
---

A short hint that appears on hover and keyboard focus, with a small pixel arrow. Wrap a focusable trigger, such as a button or a link, that has an accessible name of its own.

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
- **Stacking:** the tip sits at `--sb-z-tooltip` (70). It is positioned inside the component, not in the top layer, so an ancestor with `overflow: hidden` clips it.
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

- **Shown on hover and keyboard focus.** Keyboard focus keeps the tip up while the pointer comes and goes. A mouse click focuses the trigger without pinning the tip: it still goes when the pointer leaves.
- **Dismissible, hoverable, persistent** ([WCAG 1.4.13](https://www.w3.org/WAI/WCAG22/Understanding/content-on-hover-or-focus)). Escape hides it wherever the focus is; the pointer can move from the trigger onto the tip, which stays until the pointer leaves both, the focus moves on or Escape is pressed. A shown tip takes the pointer, so it covers what is under it.
- **Announced when it appears.** The tip lives in the component's shadow root, so it can't be the trigger's `aria-describedby`. Instead it appears inside a polite live region, which screen readers announce after the trigger's own name. While hidden it is out of the accessibility tree, so it isn't read as stray text.
- **The trigger carries the name.** Give it a name of its own: an icon button needs an `aria-label`, because the tip is read after the name, never as the name. A trigger that can't take focus, like a plain image, is out of reach for keyboard users.

Keep tooltips short and supplementary: never put the only copy of important information in one.
