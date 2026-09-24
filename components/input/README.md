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
playground:
  values: {placeholder: "Your message...", action: true}
  props: {minlength: {min: 0, max: 20}}
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
  <sb-input label="Destination" data-bind:_dest__prop.value__event.input placeholder="Type a planet"></sb-input>
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

## With commands

Give it a `name`, and it emits `sb-change` with `{ name, value }` when a value is committed: ready to post as a command. With `confirm`, it sets `:state(pending)` until the server's re-rendered attribute matches, and `revert()` goes back to the server's value when a command is rejected. See [Commands and components](/contribute#commands-and-components) and the [Showcase](/showcase).

## Styling

Style it from your page's CSS — no need to change the component or import anything into it. Custom properties, inherited properties and `::part()` all reach into its shadow root.

- **Size:** it fills the width it is given, up to `26rem`; set `max-inline-size` on the element to change that. The field is `2.75rem` tall: set `block-size` on `::part(input)` (and `::part(button)`, with `action`).
- **Fonts:** the label, the text you type and the hint use your page's font.
- **Colours:** the field is `--sb-control-bg` with a `--sb-control-border` edge (`--sb-control-border-hover` on hover) and `--sb-control-text`; the placeholder and the hint are `--sb-control-placeholder`, the label `--sb-text-2`. Focus draws a `--sb-brand-light` edge with a `--sb-brand-subtle` glow, and errors are `--sb-danger`. The submit button has a `--sb-brand-light` edge on `--sb-brand-subtle`, `--sb-brand` on hover. Corners are `--sb-control-radius`.
- **Parts:** `label`, `input` and `button` (the submit arrow, with `action`). Your page's `::part()` rules win over the component's own, without `!important`.

```html preview
<style>
  .my-input { max-inline-size: 18rem; --sb-control-radius: 0; }
  .my-input::part(input) { block-size: 2.25rem; font-size: 0.875rem; }
  .my-input::part(label) { text-transform: uppercase; letter-spacing: 0.08em; }
</style>
<sb-input class="my-input" label="Call sign" placeholder="e.g. Maverick"></sb-input>
```

## Accessibility

The native `<input>` sits inside a `<label>`. Without a `label`, the placeholder becomes the accessible name. Errors are announced with `role="alert"`, and `aria-invalid` follows the validation state.
