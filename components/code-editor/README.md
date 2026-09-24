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

## Examples

### Initial code

Put the code in a child `<script type="text/plain">`. Its text is never parsed as HTML and never executed, and common indentation is removed.

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
| Tab / Shift+Tab | Indent / outdent (the selected lines) |
| Enter | New line, keeping the indentation (plus one after `{`, `(` or `[`) |
| Ctrl/Cmd+Enter | Emit `sb-run` |
| Esc, then Tab | Leave the editor, so Tab is never a focus trap |

## With commands

Give it a `name`, and it emits `sb-change` with `{ name, value }` when a value is committed: ready to post as a command. With `confirm`, it sets `:state(pending)` until the server's re-rendered attribute matches, and `revert()` goes back to the server's value when a command is rejected. See [Commands and components](/contribute#commands-and-components) and the [Showcase](/showcase).

## Accessibility

The editable element is a native textarea with an accessible name (`label`, or "Code"). The highlighted layer and the gutter are `aria-hidden`.
