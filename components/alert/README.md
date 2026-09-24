---
name: Alert
tag: sb-alert
category: feedback
summary: Contextual feedback messages for your app.
author: zweiundeins
tags: [notice, callout, status, message]
since: 2026-09-21
preview: |
  <sb-alert variant="success" heading="Launch successful!" closable style="inline-size: 100%">Your component is live.</sb-alert>
usage: |
  <sb-alert variant="success" heading="Saved" closable>Your changes are live.</sb-alert>
playground:
  content: Your component is live.
  values: {variant: success, heading: "Launch successful!", closable: true}
---

Alerts tell people what just happened. A pixel status light, a tinted surface and an optional dismiss button, in four tones.

## Examples

### Variants

```html preview
<div style="display: grid; gap: 12px; inline-size: 100%">
  <sb-alert heading="Heads up">Telemetry resumes in 30 seconds.</sb-alert>
  <sb-alert variant="success" heading="Launch successful!">Your component is live.</sb-alert>
  <sb-alert variant="warning" heading="Low fuel">Consider a gravity assist.</sb-alert>
  <sb-alert variant="danger" heading="Hull breach">Seal deck 7 immediately.</sb-alert>
</div>
```

### Dismissible

The close button hides the alert and emits `sb-close`. From script, `host.open` reads and sets whether it is shown, and `show()` / `hide()` do the same; none of them writes the `open` attribute.

```html preview
<div style="display: grid; gap: 8px; inline-size: 100%">
  <sb-alert data-ref:_news variant="info" closable heading="New components landed">Check the gallery for fresh arrivals.</sb-alert>
  <sb-button size="sm" variant="outline" style="justify-self: start" data-on:click="$_news.show()">show()</sb-button>
</div>
```

### Server-driven

With Datastar the backend decides when to show an alert: patch it into the page from an SSE response.

```html
<div id="flash">
  <sb-alert id="flash-42" variant="success" closable heading="Saved">Mission plan stored.</sb-alert>
</div>
```

The `open` attribute is the server's:

| The server sends | What happens |
| --- | --- |
| a **changed** `open` attribute | It wins, over a dismissal too: `open="false"` hides the alert; `open`, `open="true"` or no attribute shows it again |
| the **same** markup again | Nothing. A dismissed alert stays dismissed, even with a new message in it |

So a new message needs a new element: give each message its own `id` (as above), or patch it in with mode `replace`. A dismissal stays in the browser: `sb-close` (a plain event, no detail) fires for the close button only, not for `hide()`, `host.open` or the server, so listen to it if the server should know.

## Styling

Style it from your page's CSS — no need to change the component or import anything into it. Custom properties, inherited properties and `::part()` all reach into its shadow root.

- **Fonts:** the heading and the message use your page's font.
- **Colours:** each variant has a tone: `--sb-info`, `--sb-ok` (success), `--sb-warn` (warning) or `--sb-danger`. It colours the status light and the heading, and tints the border and the background. The box is `--sb-surface-inset` with a `--sb-border` edge, the message `--sb-text-2` (the close button turns `--sb-text-1` on hover), the corners `--sb-radius`.
- **Parts:** `alert` (the box), `heading` and `message`. Your page's `::part()` rules win over the component's own, without `!important`.
- **State:** a dismissed or hidden alert matches `sb-alert:state(closed)` and takes no space in the layout, like one with the `hidden` attribute.

```html preview
<style>
  .my-alert { --sb-info: var(--sb-brand-light); }
  .my-alert::part(alert) { border-radius: 0; padding: 1rem 1.25rem; }
  .my-alert::part(message) { font-size: 1rem; }
</style>
<sb-alert class="my-alert" heading="Docking window">Bay 3 opens at 14:00.</sb-alert>
```

## Accessibility

Warning and danger alerts use `role="alert"` and are announced as soon as they appear. Info and success use the polite `role="status"`, which screen readers announce reliably only when the region was on the page before its text arrived: one patched in together with its message may go unannounced. When an info or success message must be heard, patch it into a container that stays on the page and carries `role="status"` itself (`<div id="flash" role="status">`).
