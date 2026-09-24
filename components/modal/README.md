---
name: Modal
tag: sb-modal
category: feedback
summary: Accessible modal dialogs for any use case.
author: zweiundeins
tags: [dialog, overlay, popup, confirm]
since: 2026-09-21
preview: |
  <sb-modal inline heading="Mission Control" style="--sb-radius-lg: 8px"><span>Are you ready to launch?</span><sb-button slot="footer" size="sm" variant="outline" data-sb-close>Cancel</sb-button><sb-button slot="footer" size="sm" data-sb-close>Launch</sb-button></sb-modal>
usage: |
  <button data-on:click="$_confirm.show()">Delete…</button>
  <sb-modal data-ref:_confirm heading="Delete this file?">
    This cannot be undone.
    <button slot="footer" data-sb-close="cancel">Cancel</button>
    <button slot="footer" data-sb-close="delete">Delete</button>
  </sb-modal>
playground:
  content: Are you ready to launch?<sb-button slot="footer" size="sm" variant="outline" data-sb-close>Cancel</sb-button><sb-button slot="footer" size="sm" data-sb-close>Launch</sb-button>
  values: {heading: Mission Control, inline: true}
---

A modal built on the native `<dialog>`. The browser traps focus, closes on Escape and handles the top layer. Open it with `show()` and close it with `close()`, or mark footer buttons with `data-sb-close`.

## Examples

### Open from Datastar

`data-ref` gives you the element as a signal, so any Datastar expression can call its methods.

```html preview
<div>
  <sb-button data-on:click="$_launch.show()">Launch sequence</sb-button>
  <sb-modal data-ref:_launch heading="Mission Control" data-on:sb-close="$_answer = evt.detail.value">
    Are you ready to launch? This cannot be undone.
    <sb-button slot="footer" variant="ghost" data-sb-close="cancel">Cancel</sb-button>
    <sb-button slot="footer" data-sb-close="launch">Launch</sb-button>
  </sb-modal>
  <p data-signals:_answer="''" data-show="$_answer" data-text="'You chose: ' + $_answer"></p>
</div>
```

### Inline

The `inline` attribute renders the panel in place, which is handy for docs and previews.

```html preview
<sb-modal inline heading="Docking complete" closable="false">
  All systems nominal. Welcome aboard.
</sb-modal>
```

### Opened by the server

In a CQRS app the server owns "is the dialog open?". Render `<sb-modal open>` and your SSE stream decides. Datastar's morph keeps the element, and its state lives inside the component.

## Styling

Style it from your page's CSS — no need to change the component or import anything into it. Custom properties, inherited properties and `::part()` all reach into its shadow root.

- **Size:** the dialog is `28rem` wide (less on small screens); set `inline-size` on `::part(panel)` for another width.
- **Fonts:** the heading and the body use your page's font.
- **Colours:** the panel is `--sb-surface-raised` with a `--sb-border` edge and divider, the heading `--sb-text-1`, the body `--sb-text-2`. The backdrop is `--sb-surface-overlay`, slightly blurred. The close button's focus ring is `--sb-brand`; corners are `--sb-radius-lg`.
- **Parts:** `panel` (the dialog), `heading`, `body` and `footer`. Your page's `::part()` rules win over the component's own, without `!important`.

```html preview
<style>
  .my-modal::part(panel) { inline-size: min(22rem, 100%); border-radius: 0; }
  .my-modal::part(heading) { font-size: 1.25rem; }
</style>
<sb-modal class="my-modal" inline heading="Abort launch?">The countdown stops and the crew stands down.</sb-modal>
```

## Accessibility

The native modal `<dialog>` sets `aria-modal`, traps focus and restores it on close. The heading labels the dialog. Escape and a click on the backdrop both close it, and `sb-close` reports why in `detail.reason`.
