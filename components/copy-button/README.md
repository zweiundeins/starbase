---
name: Copy Button
tag: sb-copy-button
category: utilities
summary: One-click copy to clipboard with a friendly confirmation.
author: zweiundeins
tags: [clipboard, copy, code]
since: 2026-09-21
preview: |
  <div style="display: flex; align-items: center; gap: 8px"><code>go run .</code><sb-copy-button value="go run ."></sb-copy-button></div>
usage: |
  <sb-copy-button value="npm install"></sb-copy-button>
playground:
  values: {value: go run .}
---

A small icon button that copies its `value` to the clipboard and confirms with a check mark, or says so when the browser refuses. Every code block on this site uses it.

## Examples

### Basic

```html preview
<code>&lt;sb-button&gt;Launch&lt;/sb-button&gt;</code>
<sb-copy-button value="<sb-button>Launch</sb-button>"></sb-copy-button>
```

### Copy a Datastar signal

Bind `value` to a signal with `data-attr`, and the button always copies the current value.

```html preview
<div data-signals:_coords="'51.4779° N, 0.0015° W'">
  <span data-text="$_coords"></span>
  <sb-copy-button data-attr:value="$_coords" label="Copy coordinates"></sb-copy-button>
</div>
```

### React to copies

The `sb-copy` event bubbles out of the shadow root, so `data-on` works on any ancestor.

```html preview
<div data-signals:_copies="0" data-on:sb-copy="$_copies++">
  <sb-copy-button value="ignition"></sb-copy-button>
  <span data-text="'Copied ' + $_copies + ' times'"></span>
</div>
```

### When copying fails

Browsers can refuse the clipboard: outside a secure context (plain `http://`), in an iframe without the `clipboard-write` permission, or when the document isn't focused. The button then shows a cross and `failed-label`, and emits `sb-copy-error` with `{ value, error }` (e.g. `"NotAllowedError"`), so a page can offer another way, like selecting the text:

```html
<sb-copy-button value="go run ." data-on:sb-copy-error="console.warn('copy refused:', evt.detail.error)"></sb-copy-button>
```

## Accessibility

The button's accessible name is `label`. The result ("Copied!" or "Copy failed") goes to a `role="status"` region next to the button, so screen readers announce it without moving focus. It sits outside the button on purpose: a button's children are presentational, so a live region inside one may never be read out. The same element is the visual tip (`::part(tip)`).
