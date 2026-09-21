---
name: Code Playground
tag: sb-code-playground
category: utilities
summary: Edit a component's code and HTML side by side, with a live sandboxed preview.
author: zweiundeins
tags: [code, playground, sandbox, editor, preview, repl]
since: 2026-09-21
preview: |
  <sb-code-playground style="--sb-code-playground-height: 8.5rem; inline-size: 100%"><script type="text/plain" data-file="index.html"><sb-button variant="pixel">Hi!</sb-button></script></sb-code-playground>
playground:
  exclude: [deps, themes, runner, initial]
  content: <script type="text/plain" data-file="index.html"><p>Hello from the sandbox</p></script>
  style: "inline-size: 100%; --sb-code-playground-height: 18rem"
---

A small code sandbox: file tabs with `sb-code-editor`, a live preview in a sandboxed iframe, a theme picker and a console. Every run gets a fresh document, because custom elements can't be redefined. The user code runs in an opaque origin, so it can't touch the host page's cookies, storage or DOM.

It needs `sb-code-editor` on the page and a **runner** page for the iframe. This site's runner lives at `/playground/run`. The protocol is below if you want to host your own.

## Examples

### Files from the markup

Give it child `<script type="text/plain" data-file="…">` elements: `component.js`, `index.html` and optionally `style.css`. They are never parsed or executed on the host page.

```html preview
<sb-code-playground style="inline-size: 100%; --sb-code-playground-height: 22rem">
  <script type="text/plain" data-file="component.js">
    import { rocket } from 'datastar'

    rocket('sb-greeting', {
      props: ({ string }) => ({ name: string.default('astronaut') }),
      render: ({ html, props: { name } }) => html`<p>Hello, ${name}! 🚀</p>`,
    })

    console.log('defined <sb-greeting>')
  </script>
  <script type="text/plain" data-file="index.html">
    <sb-greeting name="Starbase"></sb-greeting>
  </script>
</sb-code-playground>
```

### Using other components

`deps` maps tags to module URLs to load in the preview. Tags that the edited code defines itself are skipped, so you can edit a component that its own demo also uses.

## Runner protocol

The iframe (`sandbox="allow-scripts"`) loads `runner` and exchanges `postMessage`s. Every message carries `source: "sb-runner"`.

| Direction | Message |
|---|---|
| runner → host | `{type: "ready"}` |
| host → runner | `{type: "run", files, deps: [url], theme}` |
| runner → host | `{type: "console", level, args}` · `{type: "error", message, line}` · `{type: "done"}` |

The runner should import the edited `component.js` (for example from a `blob:` URL) **before** the dependencies, then `datastar`, and resolve `'datastar'` through an import map.

## Accessibility

File tabs are real tabs, the editors are native textareas, the preview frame has a title, and the console is a `role="log"` live region. Ctrl/Cmd+Enter in an editor runs the code.
