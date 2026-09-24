---
name: Card
tag: sb-card
category: layout
summary: Flexible container for content and actions.
author: zweiundeins
tags: [container, panel, tile]
since: 2026-09-21
preview: |
  <sb-card variant="inset" style="inline-size: 13rem"><img slot="media" src="/art/landscape.svg" alt="A violet planet rising over pixel hills" width="256" height="144"></sb-card>
usage: |
  <sb-card heading="Planet X-9">A cold, quiet world with excellent stargazing.</sb-card>
playground:
  content: <img slot="media" src="/art/landscape.svg" alt="" width="256" height="144">A cold, quiet world with excellent stargazing.
  values: {heading: Planet X-9}
  style: "inline-size: min(100%, 18rem)"
---

A surface with optional media, heading, body and footer. Sections that get no content collapse (whitespace doesn't count), and `href` makes the whole card one link: its heading, stretched over the card.

## Examples

### Media, body and footer

```html preview
<sb-card heading="Planet X-9" style="inline-size: 18rem">
  <img slot="media" src="/art/landscape.svg" alt="A violet planet rising over pixel hills" width="256" height="144">
  A cold, quiet world with excellent stargazing.
  <sb-button slot="footer" size="sm">Visit</sb-button>
  <sb-button slot="footer" size="sm" variant="ghost">Details</sb-button>
</sb-card>
```

### Linked card

```html preview
<sb-card heading="Read the Rocket reference" href="https://data-star.dev/reference/rocket" variant="glow" style="inline-size: 18rem">
  Props, codecs, slots and scoping, straight from the source.
</sb-card>
```

## Styling

Style it from your page's CSS — no need to change the component or import anything into it. Custom properties, inherited properties and `::part()` all reach into its shadow root.

- **Size:** it fills the width it is given (a grid cell, say) and tightens its padding when that is under `14rem`; in a flex row or any other fit-content layout it takes the width of its content. Set `max-inline-size` on the card to cap it. Cards in one grid row stretch to the same height, with their footers at the bottom.
- **Body:** normal flow, like a `<div>`: inline markup stays in its line, and block children (paragraphs, lists) keep their own margins.
- **Media:** scaled to the card's width and clipped to its corners. Nothing else is clipped, so tooltips and badges can reach outside the card. Raster pixel art that should stay crisp when scaled needs `image-rendering: pixelated` on your `<img>`.
- **Fonts:** the heading and the body use your page's font.
- **Colours:** the surface is `--sb-surface-card` (`--sb-surface-inset` for `variant="inset"`) with a `--sb-border` edge; the heading is `--sb-text-1` and the body `--sb-text-2`. `--sb-brand` tints the border of the `glow` variant and of a linked card on hover and keyboard focus; keyboard focus also rings the card with `--sb-focus-ring`. Corners are `--sb-radius-lg`.
- **Motion:** a linked card lifts by 2px on hover and focus, except with `prefers-reduced-motion`.
- **Parts:** `card` (the frame), `media`, `heading`, `body` and `footer`. Your page's `::part()` rules win over the component's own, without `!important`.

```html preview
<style>
  .my-card { max-inline-size: 20rem; --sb-surface-card: var(--sb-brand-subtle); --sb-radius-lg: 0; }
  .my-card::part(body) { padding: 1.5rem; }
  .my-card::part(heading) { font-size: 1.25rem; }
</style>
<sb-card class="my-card" heading="Planet X-9">A cold, quiet world with excellent stargazing.</sb-card>
```

## Accessibility

A linked card renders one real `<a>` in the heading and stretches its click area over the card, so screen readers hear a single link; without a `heading` there is no link. Keyboard focus rings the whole card. Links and buttons placed directly in the body, and everything in the footer, stay clickable on their own; give other interactive content in a linked card `position: relative`, so it sits above the card's link. Always give media images `alt` text.
