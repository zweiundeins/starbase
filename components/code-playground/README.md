---
name: Code Playground
tag: sb-code-playground
category: utilities
unlisted: true
summary: Edit a component's code and HTML side by side, with a live sandboxed preview.
author: zweiundeins
tags: [code, playground, sandbox, editor, preview, repl]
since: 2026-09-21
preview: |
  <sb-code-playground style="--sb-code-playground-height: 8.5rem; inline-size: 100%"><script type="text/plain" data-file="index.html"><sb-button variant="pixel">Hi!</sb-button></script></sb-code-playground>
usage: |
  <sb-code-playground runner="/your/runner.html">
    <script type="text/plain" data-file="index.html"><p>Hello!</p></script>
  </sb-code-playground>
playground:
  props: { delay: { min: 100, max: 5000, step: 100 } }
  exclude: [deps, themes, runner, initial, autoRun, theme]
  content: <script type="text/plain" data-file="index.html"><p>Hello from the sandbox</p></script>
  style: "inline-size: 100%; --sb-code-playground-height: 18rem"
---

A small code sandbox: file tabs with `sb-code-editor`, a live preview in a sandboxed iframe, a theme picker and a console. Every run gets a fresh iframe, because custom elements can't be redefined; it replaces the old one, so runs add no browser history and nothing from an earlier run keeps going. The user code runs in an opaque origin, so it can't touch the host page's cookies, storage or DOM.

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

### Your own things in the bar

Elements with `slot="bar"` go into the top bar, between the file tabs and the run status. `sb-change` (`detail.files`) tells you when the code changed, on every edit. The editors' own `change` and `sb-change` (`{ name, value }`) stay inside the playground. Starbase's `/playground` puts a live size line there: every edit (debounced) posts `component.js` to the server, which minifies and compresses it like the catalog's own modules.

```html
<sb-code-playground data-on:sb-change__debounce.400ms="@post('/size', {payload: {code: evt.detail.files['component.js']}})">
  <span slot="bar" data-text="$_size"></span>
</sb-code-playground>
```

## Properties and methods

- `files`: the current files as `{ "component.js": "…", … }`. Setting it merges the given files over the current ones, updates the editors and runs. Starbase's "Save & share" reads it.
- `run()`: run the current files, like the Run button.

## Runner protocol

The iframe (`sandbox="allow-scripts allow-modals"`) loads `runner` and exchanges `postMessage`s. Every message carries `source: "sb-runner"`.

| Direction | Message |
|---|---|
| runner → host | `{type: "ready"}` |
| host → runner | `{type: "run", files, deps: [url], theme, base}` |
| runner → host | `{type: "console", level, args}` · `{type: "error", message, line}` · `{type: "done"}` |

The runner should import the edited `component.js` (for example from a `blob:` URL) **before** the dependencies, then `datastar`, and resolve `'datastar'` through an import map. A `blob:` module has no folder, so the runner resolves relative imports (`./vendor/lib.js`) against `base`, the `base` attribute as an absolute URL (empty when unset).

"Ran in … ms" is the time from `ready` to `done`. If a run has not sent `done` 5 s after its runner page loaded, the console says so: "No answer after 5 s: an endless loop?", or "The runner did not answer" when not even `ready` came. The next run replaces the frame, which stops it. This site's runner stops endless loops itself: every `while` and `for (…; …; …)` in `component.js` checks the time, and a task that spends a second in loops throws "Endless loop? Stopped after 1 s" (with the loop's line).

## Styling

Style it from your page's CSS — no need to change the component or import anything into it. Custom properties, inherited properties and `::part()` all reach into its shadow root.

- **Size:** `--sb-code-playground-height` (default `34rem`) is the height of the whole playground. It fills the width it is given; the editor and the preview sit side by side, and stack below `48rem`.
- **Fonts:** the toolbar uses your page's font. The console and the code use `--sb-font-ui` when a site sets one, else a monospace font.
- **Colours:** the frame is `--sb-surface-card` with `--sb-border` lines; the selected file tab, the console and the theme picker are `--sb-surface-inset`. Text is `--sb-text-2` (`--sb-text-1` when active), the status `--sb-text-muted`. The Run button fills with `--sb-brand` (`--sb-brand-hover` on hover) and writes (and draws its triangle) in `--sb-text-on-brand`; focus rings are `--sb-brand-light`. Console errors are `--sb-danger`, warnings `--sb-warn`, and the preview's background is `--sb-bg`. Corners are `--sb-radius-lg`. The editors are `sb-code-editor`s, so their syntax colours (`--sb-code-tag`, `--sb-code-keyword`…) reach them too.
- **Parts:** `playground` (the frame), `run`, `preview` (the iframe) and `console`. Your page's `::part()` rules win over the component's own, without `!important`.

```html preview
<style>
  .my-playground { inline-size: 100%; --sb-code-playground-height: 16rem; --sb-code-tag: #DB2777; }
  .my-playground::part(run) { background: #15803D; border-color: #15803D; color: #FFFFFF; }
</style>
<sb-code-playground class="my-playground">
  <script type="text/plain" data-file="index.html">
    <p>Hello from a smaller playground.</p>
  </script>
</sb-code-playground>
```

A font you load yourself works inside the component too: load it in the page (a `<link>`, or an `@import` at the very top of your main stylesheet). A component's own styles are a constructed stylesheet, which can't `@import`.

## Accessibility

File tabs are real tabs: one Tab stop, arrow keys, Home and End switch files, and each editor (a native textarea) is its tab panel. The preview frame has a title. The console is a `role="log"` live region, so output, errors and a run that does not answer are announced; the run status ("Ran in … ms") is not, so auto-runs while typing stay quiet. Ctrl/Cmd+Enter in an editor runs the code.
