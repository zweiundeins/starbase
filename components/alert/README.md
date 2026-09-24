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

```html preview
<sb-alert variant="info" closable heading="New components landed">Check the gallery for fresh arrivals.</sb-alert>
```

### Server-driven

With Datastar the backend decides when to show an alert. Patch it into the page from an SSE response and the client needs no extra logic:

```html
<div id="flash">
  <sb-alert variant="success" closable heading="Saved">Mission plan stored.</sb-alert>
</div>
```

## Accessibility

Warning and danger alerts use `role="alert"` and are announced immediately. Info and success use the polite `role="status"`.
