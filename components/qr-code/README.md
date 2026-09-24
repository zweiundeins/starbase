---
name: QR Code
tag: sb-qr-code
category: media
summary: Pixel-perfect QR codes for any text or URL, with optional brand-coloured corners.
author: zweiundeins
tags: [qr, code, share, url, svg]
since: 2026-09-22
preview: |
  <sb-qr-code value="https://data-star.dev" accent style="inline-size: 7rem"></sb-qr-code>
usage: |
  <sb-qr-code value="https://data-star.dev"></sb-qr-code>
playground:
  props:
    border: {min: 0, max: 8}
  values: {value: "https://data-star.dev", accent: true}
---

Draws a QR code as a crisp SVG: every module is a pixel. It encodes with [uqr](https://github.com/unjs/uqr) (MIT), vendored unmodified from npm; `vendor.json` records the release it is verified against.

The code stays dark on white whatever the theme, since some scanners can't read inverted codes. Size it with CSS (`inline-size`), or set `--sb-qr-color`, `--sb-qr-background` and `--sb-qr-accent`.

## Examples

### Live from an input

```html preview
<div data-signals="{_qr: 'https://starbase.example'}" style="display: grid; gap: 16px; justify-items: start">
  <sb-input label="Text or URL" data-bind:_qr__prop.value style="inline-size: 20rem"></sb-input>
  <sb-qr-code accent data-attr:value="$_qr" data-preserve-attr="value"></sb-qr-code>
</div>
```

### Error correction

Higher levels survive more damage, like a logo over the middle, and need more modules.

```html preview
<div style="display: flex; gap: 16px; flex-wrap: wrap">
  <sb-qr-code value="Ad astra" ecc="L" style="inline-size: 6rem"></sb-qr-code>
  <sb-qr-code value="Ad astra" ecc="H" style="inline-size: 6rem"></sb-qr-code>
</div>
```

### Colours

```html preview
<sb-qr-code value="https://github.com/zweiundeins/starbase" accent
  style="inline-size: 8rem; --sb-qr-color: #2E2780; --sb-qr-accent: #E5484D"></sb-qr-code>
```

## From the server

Server-rendered values work as they are: `<sb-qr-code value="https://example.com/invite/X7Q2">`. When a stream re-renders the attribute, the code follows.

## Styling

Style it from your page's CSS — no need to change the component or import anything into it. Custom properties, inherited properties and `::part()` all reach into its shadow root.

- **Size:** `10rem` square by default; set `inline-size` on the element and it stays square.
- **Colours:** `--sb-qr-color` for the modules (near-black), `--sb-qr-background` behind them (white) and, with `accent`, `--sb-qr-accent` for the three corner squares (your `--sb-brand` unless set). Keep the contrast high: scanners need it.
- **Parts:** `svg`, `background`, `modules` and `corners`, for anything the tokens don't cover (they are SVG shapes, so they take `fill`). Your page's `::part()` rules win over the component's own, without `!important`.

```html preview
<style>
  .my-qr { inline-size: 7rem; --sb-qr-color: #1E3A8A; --sb-qr-accent: #DB2777; }
</style>
<sb-qr-code class="my-qr" accent value="https://data-star.dev"></sb-qr-code>
```

## Accessibility

The SVG is an `img` named "QR code: " plus the value (or `label`). Put the link or text next to the code as well, for people who can't scan it.
