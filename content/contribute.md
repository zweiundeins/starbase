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
usage: |                   # optional
  <sb-my-widget label="Volume"></sb-my-widget>
---
```

`preview:` is the gallery card's live demo, so tune it to look good in a card. `usage:` is the smallest markup that shows how you'd use the component in your own page; the installation snippets use it, and fall back to `preview:`. Keep it free of card sizing, demo-only signals and state, and assets that only exist on Starbase.

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
- **Few dependencies.** Import from `'datastar'`, or relatively from files inside your own folder (for example a vendored ES module in `vendor/`, together with its licence). Submit those from a repository link: the bot brings along every file the component imports, up to 2 MB each. Vendored libraries must be unmodified files from an npm release, listed in a `vendor.json` next to the component (`{"vendor/lib.js": {"npm": "lib@1.2.3", "file": "dist/lib.esm.js"}}`). The bot verifies them byte for byte against the registry, and refuses minified code it can't verify. No imports from CDNs or other URLs. Keep it small.

## Commands and components

Starbase is built on CQRS: a change is a command sent to the server, and the page shows the server's state again. Components that hold a value follow one contract, so any of them can drive a command:

- **Intent:** when the user commits a value (release, blur, Enter; never every keystroke), emit `sb-change` with `{ name, value }`, and give the component a `name` prop. A page can then send it as is: `data-on:sb-change="@post('/cmd/…', {payload: evt.detail})"`.
- **The server's value is the attribute.** A `value` (or `checked`, `selected`…) attribute the server changes wins over local edits; re-sent identical markup changes nothing. A *removed* attribute is ignored, so to clear, the server sends `value=""` or `checked="false"`.
- **Pending and revert:** with the `confirm` attribute, the component sets `:state(pending)` (a CSS custom state, styleable from the page, safe from morphs) while its local value differs from the server's. `host.revert()` goes back to the server's value, for a rejected command:

```html
<sb-slider name="thrust" confirm value="40"
  data-on:sb-change="@post('/cmd/flight', {payload: {tabid: $tabid, ...evt.detail}})"
  data-on:datastar-fetch="evt.detail.el === el && evt.detail.type === 'error' && el.revert()"></sb-slider>
```

`datastar-fetch` reaches every listener, so the handler checks `evt.detail.el === el` first. The [Showcase](/showcase) has it running: a pending state, a server-normalized value and a rejected command.

**No optimistic updates.** Never show a result the server hasn't produced: counts, lists, derived values and success messages come only from its render. The user's own input stays as they left it, marked pending (`:state(pending)`, or a faded item for operations) until the server confirms it, and goes back with `revert()` if the command is rejected. This is the [Tao of Datastar](https://data-star.dev/guide/the_tao_of_datastar#optimistic-updates) applied to components.

### Lists that hold the keyboard

A component whose rows can be replaced by the server (a tree, a menu, a group of choices) has to defend the focus, or a morph throws the user out of it:

- The morph can **park a row before removing it**, so `focusout` fires while the row is still connected and `relatedTarget` is `null`. Treat that as "not leaving" and keep your "focus is inside" flag.
- Restore the DOM focus after a re-render only when it **fell on the floor** — `document.activeElement` is the body or the host. If the user moved on to something else, leave it there; a component that grabs focus back is worse than one that loses it.
- When the focused row is **gone from the new list**, focus its neighbour (the old index, clamped into the new list, skipping disabled rows), not the first row: jumping to the top turns one arrow key into a trip to the other end of the list.

`sb-tree`, `sb-radio-group` and `sb-dropdown` all do this; copy from whichever is closest in shape.

### Components with several values

- **One decision, one value.** Parts that change together (a range's start and end, a multi-select's picks, a colour's channels) are one structured `value`: a JSON attribute such as `value='{"start":20,"end":60}'`, one `sb-change` with the whole value, and one command the server accepts or rejects as a whole. Never one event per part: two commands could leave the server with half a change, or start after end.
- **View state is not the value.** What the user looks at rather than decides (a tree's `expanded` branches, an open panel) is a separate prop with its own event, e.g. `sb-toggle`. The server may keep it with its own command, but it isn't part of `pending` or `revert()`.
- **Server data only flows in.** Props the server fills (`results`, `options`, `loaded`) are never pending and never sent back.
- **Operations instead of a value.** When an intent is an operation, not a new value (painting pixels, moving a card, sending a message), emit an `sb-<verb>` event with the operation and show each item as pending until the server's re-render contains it, like the [pixel board](/components/pixel-board).

## Review

A maintainer checks that the component renders in the gallery and on the Themes page, that `go test ./...` passes, and that the docs examples work. Components are published under the MIT licence with you as the author.
