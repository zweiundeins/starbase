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
  exclude: [rev]
---

A text field with a label, help text, native constraint validation and an optional submit arrow. It exposes a live `value` property, so `data-bind` works as it does on a native `<input>`.

## Examples

### Label and hint

```html preview
<sb-input label="Call sign" placeholder="e.g. Maverick" hint="Shown to the rest of the crew."></sb-input>
```

### Validation

Validation starts once a value is committed (Enter, or leaving a changed field) or submitted, and then runs on every keystroke and on every new value, like the browser's `:user-invalid`. `required`, `minlength`, `pattern` and `type` use the browser's own rules. Set `error` to replace the default message.

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

`sb-submit` fires on Enter or on the arrow button, but only when the value is valid. Send it to your backend with `@post`. (Enter also commits the value, so `sb-change` follows.)

```html preview
<div data-signals:_log="''">
  <sb-input action required placeholder="Transmit a message..." data-on:sb-submit="$_log = 'Sent: ' + evt.detail.value"></sb-input>
  <p data-text="$_log"></p>
</div>
```

## With commands

Give it a `name`, and it emits `sb-change` with `{ name, value }` when a value is committed (Enter, or leaving a changed field): ready to post as a command. The `value` attribute is the server's value: a new one replaces what the user typed, and re-sent identical markup leaves an edit alone. To clear the field, the server sends `value=""`.

With `confirm`, it sets `:state(pending)` while the local value differs from the server's, and `revert()` goes back to the server's value when a command is rejected:

```html
<sb-input name="callsign" label="Call sign" confirm value="STARBASE-1" rev="7"
  data-on:sb-change="@post('/cmd/flight', {payload: evt.detail})"
  data-on:datastar-fetch="evt.detail.el === el && evt.detail.type === 'error' && el.revert()"></sb-input>
```

A server that answers with the value it already had (it upper-cased `starbase-1` back to `STARBASE-1`, or ignored a no-op) sends identical markup, so the edit would stay pending. Render `rev`, a revision that changes with every applied command for this field (a counter will do): a new `rev` means the server has answered, and its value wins even when it is unchanged. The same clears a chat-style field after its message was sent: `value=""` with a new `rev`.

See [Commands and components](/contribute#commands-and-components) and the [Showcase](/showcase), which runs this call sign.

## Forms

`sb-input` is not a form-associated element: a `<form>` doesn't submit it, `FormData` and Datastar's `contentType: 'form'` don't see it, and a form reset doesn't reset it. Send its value as a command instead: `sb-change` carries `{ name, value }` (see [With commands](#with-commands)).

## Styling

Style it from your page's CSS — no need to change the component or import anything into it. Custom properties, inherited properties and `::part()` all reach into its shadow root.

- **Size:** it fills the width it is given, up to `26rem`; set `max-inline-size` on the element to change that. The field is `2.75rem` tall: set `block-size` on `::part(input)` (and `::part(button)`, with `action`).
- **Fonts:** the label, the text you type and the hint use your page's font.
- **Colours:** the field is `--sb-control-bg` with a `--sb-control-border` edge (`--sb-control-border-hover` on hover) and `--sb-control-text`; the placeholder and the hint are `--sb-control-placeholder`, the label `--sb-text-2`. Focus draws a `--sb-brand-light` edge with a `--sb-brand-subtle` glow, and errors are `--sb-danger`. The submit button has a `--sb-brand-light` edge on `--sb-brand-subtle`, and is `--sb-brand` with a `--sb-text-on-brand` arrow on hover. Corners are `--sb-control-radius`.
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

`label` is a `<label>` for the native `<input>`; without one, the element's `aria-label` or the placeholder names it. The hint and the error describe it (`aria-describedby`). The error is a polite live region, announced after what the screen reader is saying, and `aria-invalid` follows the validation state. Right to left, the submit arrow points the other way. In forced colors, focus shows as an outline.
