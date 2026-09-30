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

A small icon button that copies its `value` to the clipboard and confirms with a check mark. It also copies on plain `http://`, where browsers have no Clipboard API, and when nothing else works it selects the text for the user to copy. Every code block on this site uses it.

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

### Plain HTTP and refused clipboards

The button tries three ways, in this order:

1. The Clipboard API (`navigator.clipboard.writeText`). Browsers offer it only in a secure context: `https://` pages and `http://localhost`.
2. A textarea in the button's own shadow root, holding the value, selected and copied with `document.execCommand('copy')`. Without a Clipboard API this runs in the click's own task, while the browser still counts the click as a user action. When the API refuses (in an iframe without the `clipboard-write` permission, or when the document isn't focused), it runs once the refusal arrives, and a browser that no longer counts the click by then refuses this too. The textarea is in the button's layer (a modal dialog, a popover), never in inert content; it is removed straight after the copy command, in the same task, and the focus goes back to the button.
3. A panel under the button with the text selected and focused, and a message that says which keys copy it: "Press Ctrl+C to copy", or ⌘C on Apple systems (`manual-label`, where `{key}` stands for the keys). A copy of the whole text from the panel, with the keys or the system's copy menu on a phone, closes it and shows `copied-label`; a copy of only a part, or of nothing after a click into the field, leaves it open and reports nothing. Escape and a click elsewhere close it quietly. It opens in the top layer, so a code block's `overflow: hidden` doesn't cut it off.

`sb-copy` says which way worked in `method`: `"clipboard"`, `"execCommand"` or `"manual"`. `sb-copy-error` fires only when the first two ways failed, with `{ value, error }`: the Clipboard API's error name (`"NotAllowedError"` when it refused) or `"TypeError"` where there is none. It is cancelable: a page that offers its own way calls `preventDefault()`, and the button shows the cross and `failed-label` instead of the panel.

```html
<sb-copy-button value="go run ." data-on:sb-copy-error="evt.preventDefault(); console.warn('copy refused:', evt.detail.error)"></sb-copy-button>
```

A new `label` re-renders the button, which closes an open panel.

## Styling

Style it from your page's CSS, without changing the component or importing anything into it. Custom properties, inherited properties and `::part()` all reach into its shadow root.

- **Size:** the button is a `2rem` square inside a 1px border; set `inline-size` and `block-size` on `::part(button)` for another (the border comes on top), and the icon stays half of it.
- **Fonts:** the "Copied!" tip and the panel use your page's font. The panel's text field is at least 12pt, since iOS zooms in on a smaller field when it takes the focus.
- **Colours:** the button is `--sb-surface-raised` with a `--sb-border` edge and a `--sb-text-2` icon (on hover `--sb-text-1`, with a `--sb-border-strong` edge). After copying it turns `--sb-ok`; when copying failed and the page cancelled `sb-copy-error`, `--sb-danger`. The tip is `--sb-surface-raised` with a `--sb-border-strong` edge and `--sb-text-1` text; a failure fills it with `--sb-danger` and `--sb-text-on-danger`. The panel is `--sb-surface-raised` with a `--sb-border-strong` edge and `--sb-text-1` text, and its text field has a `--sb-border` edge. Corners are `--sb-radius-sm`, the focus ring `--sb-focus-ring`. In forced colors the panel gets a system-colour outline.
- **Parts:** `button`, `tip`, `manual` (the panel) and `manual-text` (its text field). Your page's `::part()` rules win over the component's own, without `!important`.
- **States:** the host has `:state(copied)` or `:state(failed)` while the check mark or the cross shows, and `:state(manual)` while the panel is open, e.g. `sb-copy-button:state(manual)`. They are CSS custom states, so a server morph cannot reset them.

```html preview
<style>
  .my-copy { --sb-ok: var(--sb-brand-light); }
  .my-copy::part(button) { inline-size: 2.5rem; block-size: 2.5rem; border-radius: 50%; }
</style>
<sb-copy-button class="my-copy" value="go run ."></sb-copy-button>
```

## Accessibility

The button's accessible name is `label`. The result ("Copied!" or "Copy failed") goes to a `role="status"` region next to the button, so screen readers announce it without moving focus, also after a copy from the panel. It sits outside the button on purpose: a button's children are presentational, so a live region inside one may never be read out. The same element is the visual tip (`::part(tip)`). Keyboard focus shows `--sb-focus-ring`, and in forced colors (Windows High Contrast), which drop that shadow, a system-colour outline.

When the panel opens, the focus moves to its text field with the whole text selected. The message ("Press Ctrl+C to copy") is the field's name, so a screen reader reads the instruction and then the text. Escape closes the panel, and inside a modal dialog or a drawer the first Escape closes only the panel. When the panel closes with the focus inside, the focus goes back to the button. The textarea of the second way holds the focus only for the copy itself, and then the button has it again. The button shows its focus ring if it had one when it was clicked, or if the panel was closed with a key (Escape, the copy shortcut), so a mouse click leaves none.
