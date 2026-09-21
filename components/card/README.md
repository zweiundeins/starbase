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
---

A surface with optional media, heading, body and footer. Slots that get no content collapse, and `href` turns the whole card into one link.

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

## Accessibility

A linked card renders one real `<a>` in the heading and stretches its click area over the card, so screen readers hear a single link. Always give media images `alt` text.
