---
title: Contribute
lede: Built something small and useful? Launch it here. A component is one folder and one pull request.
description: How to add a Rocket component to the Starbase community collection.
---

## Submit in two minutes (no tools needed)

**[Open the submission form →](/submit)**

Tip: build it in the **[Playground](/playground)** first. Hit *Save & share*, then *Submit as component*, and the form arrives prefilled with your playground link.

It's a GitHub issue form. Either:

- **Paste your component:** its code, a preview snippet and some docs, or
- **Link your repository:** paste the URL of the public GitHub repo (or folder) your component lives in. We take the file that calls `rocket('sb-…')` and the `README.md` next to it, and pin the exact commit. Editing the issue later re-syncs from the repo.

A bot validates the component, loads it in a headless browser to generate its API reference, and opens a pull request with you as the author. If something is wrong, it comments on the issue: fix the issue and it tries again. You can update your own component later the same way.

## Or open a pull request

Prefer git? Fork the repository and:

1. `go tool task new -- my-widget --category forms` scaffolds `components/my-widget/`.
2. Build it in `my-widget.js` and document it in `README.md`. `go tool task live` shows it at `/components/my-widget`.
3. `go tool task manifests` generates `manifest.json` (it needs Chrome or Chromium), and `go tool task test` validates everything.
4. Open the pull request. CI runs the same checks.

There is no Go code to touch: the gallery, search, categories, docs page and Playground all come from your folder.

## Anatomy of a component

```text
components/my-widget/
├── README.md       front matter + docs; html preview blocks become live demos
├── my-widget.js    rocket('sb-my-widget', { ... })
└── manifest.json   generated in dev from Rocket's manifest, commit it
```

The front matter drives the gallery card:

```yaml
---
name: My Widget
tag: sb-my-widget          # must start with sb-
category: forms            # forms · navigation · feedback · layout · media · data · utilities · experimental
summary: One line, at most 90 characters.
author: your-github-handle
tags: [keywords, for, search]
since: 2026-09-21
preview: |
  <sb-my-widget></sb-my-widget>
---
```

A fenced block tagged `html preview` renders twice: live on the page and as copyable source.

````markdown
```html preview
<sb-my-widget size="lg"></sb-my-widget>
```
````

Your page automatically gets a **Playground** built from the manifest. Add a `playground:` block to the front matter to set slider ranges (`props`), starting `values`, slotted `content`, a `style` or static `attrs` for the live element (useful for arrays and JSON props, which get no control), or to `exclude` props.

## House rules

- **Shadow DOM and tokens.** Use Rocket's default `open` mode. Style only through `--sb-*` semantic tokens, each with a fallback: `--_bg: var(--sb-surface-card, #141D32)`. That makes the [Themes](/themes) page work, and the component still works outside Starbase. Pixel corners scale with `--sb-notch` (`--_notch: var(--sb-notch, 1)`; at `0`, fall back to a normal border radius), so people can opt out of the 8-bit look.
- **State lives in `$$` signals, not attributes.** Datastar apps morph server HTML into the page. If your component writes interaction state back to its own attributes, the next morph resets it. Initialise `$$` from props, update `$$`, and expose live values with `overrideProp` getters.
- **Speak Datastar.** Emit bubbling, composed events with `emit()`, so `data-on:*` works on any ancestor. Expose a `value` or `checked` property and fire `change` or `input`, so `data-bind` works. In your docs, declare bound signals with `data-signals` first, and bind components with `__prop`: `data-bind:_x__prop.value`. Datastar binds before custom elements are upgraded, and without `__prop` it falls back to the `value` *attribute*, which the next server morph removes. Host getters and setters that read or write `$$` must wrap the access in `startPeeking()`/`stopPeeking()`. Otherwise `data-bind`'s own effect subscribes to your internal signal and writes stale values back. Also adopt a `value` that `data-bind` set before your element was upgraded (see `early()` in the slider).
- **Signal-driven attributes survive morphs only when preserved.** When a page drives a component with `data-attr:yaw="$_yaw"`, add `data-preserve-attr="yaw"` to the host. The server morph copies attributes from its own markup, and without this it would reset them on every frame.
- **Wire it declaratively.** In `render`, use `data-on:*` with local actions (`action('press', …)` → `data-on:click="@press()"`), and `data-bind`, `data-show`, `data-class`, `data-text` and `<template data-for>` on `$$` signals. Get elements through `data-ref` and `onFirstRender({ refs })`. Reach for `addEventListener` only where no attribute exists (`matchMedia`, observers).
- **Accessible by default.** Use native elements inside the shadow root, keyboard support, visible focus, and `prefers-reduced-motion`.
- **Document the API.** Give every prop `.docs({ description })` and declare slots and events in `manifest`. The API tables are generated from it.
- **Few dependencies.** Import from `'datastar'`, or relatively from files inside your own folder (for example a vendored ES module in `vendor/`, together with its licence). Keep it small.

## Review

A maintainer checks that the component renders in the gallery and on the Themes page, that `go test ./...` passes, and that the docs examples work. Components are published under the MIT licence with you as the author.
