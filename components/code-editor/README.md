---
name: Code Editor
tag: sb-code-editor
category: utilities
summary: A light code field with pixel-theme highlighting, smart tabs and line numbers.
author: zweiundeins
tags: [code, editor, textarea, syntax, highlight]
since: 2026-09-21
preview: |
  <sb-code-editor language="js" line-numbers style="--sb-code-editor-height: 7.5rem"><script type="text/plain">
  rocket('sb-hello', {
    render: ({ html }) => html`<b>hi</b>`,
  })
  </script></sb-code-editor>
usage: |
  <sb-code-editor language="js" label="script.js" value="const answer = 42"></sb-code-editor>
playground:
  props:
    tabSize: {min: 1, max: 8}
  content: <script type="text/plain">const answer = 42 // edit me</script>
  style: "inline-size: min(100%, 32rem)"
---

A real `<textarea>` layered over a highlighted `<pre>`. You get native editing (undo, IME, mobile keyboards, screen readers) with syntax colours from the theme tokens. It understands JavaScript, HTML and CSS, keeps indentation on Enter, indents and outdents selections with Tab and Shift+Tab, and emits `sb-run` on Ctrl/Cmd+Enter.

The highlighter (Prism) loads with the first editor on a page, so pages without one never download it. Until it arrives, the code shows uncoloured.

## Examples

### Initial code

Put the code in a child `<script type="text/plain">`. Its text is never parsed as HTML and never executed, and common indentation is removed. It counts as the server's value (for `confirm` and `revert()`), and it is read once: to change the code later, the server sends a `value` attribute.

```html preview
<sb-code-editor language="html" label="index.html" style="inline-size: 100%">
  <script type="text/plain">
    <sb-button variant="pixel" caret>Blast off</sb-button>
    <!-- edit me -->
  </script>
</sb-code-editor>
```

### Bound to a signal

`value` is a live property, so `data-bind` works (with `__prop.value`, and with the signal declared first).

```html preview
<div data-signals:_css="'.planet { color: #8C6BFF; }'" style="display: grid; gap: 12px; inline-size: 100%">
  <sb-code-editor language="css" data-bind:_css__prop.value></sb-code-editor>
  <p data-text="$_css.length + ' characters'"></p>
</div>
```

### Run with Ctrl+Enter

```html preview
<div data-signals:_runs="0" style="display: grid; gap: 12px; inline-size: 100%">
  <sb-code-editor language="js" data-on:sb-run="$_runs++"><script type="text/plain">console.log('press Ctrl+Enter')</script></sb-code-editor>
  <p data-text="'Runs: ' + $_runs"></p>
</div>
```

## Keyboard

| Keys | Action |
|---|---|
| Tab / Shift+Tab | Indent / outdent the selected lines (without a selection, Tab inserts a tab and Shift+Tab outdents the line) |
| Enter | New line, keeping the indentation (plus one after `{`, `(` or `[`) |
| Ctrl/Cmd+Enter | Emit `sb-run` |
| Esc, then Tab | Leave the editor, so Tab is never a focus trap |

## With commands

Give it a `name`, and it emits `sb-change` with `{ name, value }` when a value is committed: ready to post as a command. With `confirm`, it sets `:state(pending)` until the server's re-rendered attribute matches, and `revert()` goes back to the server's value when a command is rejected. See [Commands and components](/contribute#commands-and-components) and the [Showcase](/showcase).

A new `value` from the server replaces the code; the same markup sent again leaves the user's edits alone. To clear it, the server sends `value=""` (a removed attribute changes nothing). Moving the editor elsewhere in the page keeps its edits, like a `<textarea>`.

## Forms

`sb-code-editor` is not a form-associated element: a `<form>` doesn't submit it, `FormData` and Datastar's `contentType: 'form'` don't see it, and a form reset doesn't reset it. Send its value as a command instead: `sb-change` carries `{ name, value }` (see [With commands](#with-commands)).

## Styling

Style it from your page's CSS — no need to change the component or import anything into it. Custom properties, inherited properties and `::part()` all reach into its shadow root.

- **Size:** it fills the width it is given. `--sb-code-editor-height` (default `28rem`) is the most it grows before it scrolls, `--sb-code-editor-min-height` (default `0`) the least it shrinks to. `--sb-code-editor-font-size` (default `0.8125rem`) sets the size of the code and the line numbers.
- **Fonts:** the code uses `--sb-font-ui` when a site sets one, else JetBrains Mono or the system's monospace font; set `--sb-font-ui` on the editor to choose it. The label uses your page's font. The highlighted code is drawn under a transparent textarea, and the two must line up to the pixel, so change the code's font and size through `--sb-font-ui` and `--sb-code-editor-font-size`, which set both, not with `::part(textarea)`.
- **Colours:** the syntax colours are `--sb-code-keyword`, `--sb-code-function`, `--sb-code-tag`, `--sb-code-string` and `--sb-code-number`; comments, punctuation and the line numbers are `--sb-text-muted`, plain code `--sb-text-1`. The box is `--sb-surface-inset` with a `--sb-border` edge (`--sb-brand-light` while you type), the selection `--sb-selection`, the label `--sb-text-2`, the corners `--sb-radius`.
- **Parts:** `editor` (the scrolling box), `label` and `textarea` (the layer you type in). Your page's `::part()` rules win over the component's own, without `!important`.

```html preview
<style>
  .my-editor {
    --sb-code-editor-height: 8rem;
    --sb-code-editor-font-size: 1rem;
    --sb-font-ui: "Courier New", monospace;
    --sb-code-keyword: #DB2777;
  }
</style>
<sb-code-editor class="my-editor" language="js" label="orbit.js" value="const period = 2 * Math.PI * Math.sqrt(a ** 3 / mu)"></sb-code-editor>
```

A font you load yourself works inside the component too: load it in the page (a `<link>`, or an `@import` at the very top of your main stylesheet). A component's own styles are a constructed stylesheet, which can't `@import`.

## Accessibility

The editable element is a native textarea with an accessible name: `label`, else the host's `aria-label`, else "Code". The visible label is a `<label>` for it, and `focus()` on the element focuses the textarea. The highlighted layer and the gutter are `aria-hidden`.
