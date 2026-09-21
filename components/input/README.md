---
name: Input
tag: sb-input
category: forms
summary: Styled input with built-in reactivity and validation.
author: zweiundeins
tags: [text, field, form, validation]
since: 2026-09-21
preview: |
  <sb-input placeholder="Your message..." action></sb-input>
---

A text field with a label, help text, native constraint validation and an optional submit arrow. It exposes a live `value` property, so `data-bind` works as it does on a native `<input>`.

## Examples

### Label and hint

```html preview
<sb-input label="Call sign" placeholder="e.g. Maverick" hint="Shown to the rest of the crew."></sb-input>
```

### Validation

Validation runs after the first blur and then on every keystroke. `required`, `minlength`, `pattern` and `type` use the browser's own rules. Set `error` to replace the default message.

```html preview
<sb-input label="Email" type="email" required placeholder="you@starbase.dev"></sb-input>
<sb-input label="Launch code" pattern="[0-9]{4}" error="Four digits, commander." placeholder="0000"></sb-input>
```

### Two-way binding

Declare the signal first, because `data-bind` only writes to a signal that already has a value.

```html preview
<div data-signals:_dest="''">
  <sb-input label="Destination" data-bind:_dest__event.input placeholder="Type a planet"></sb-input>
  <p data-text="$_dest ? 'Plotting course to ' + $_dest + '…' : 'Awaiting destination'"></p>
</div>
```

### Submit with the arrow

`sb-submit` fires on Enter or on the arrow button, but only when the value is valid. Send it to your backend with `@post`.

```html preview
<div data-signals:_log="''">
  <sb-input action required placeholder="Transmit a message..." data-on:sb-submit="$_log = 'Sent: ' + evt.detail.value"></sb-input>
  <p data-text="$_log"></p>
</div>
```

## Accessibility

The native `<input>` sits inside a `<label>`. Without a `label`, the placeholder becomes the accessible name. Errors are announced with `role="alert"`, and `aria-invalid` follows the validation state.
