---
title: Contribute
lede: Built something small and useful? Launch it here. A component is one folder and one pull request.
description: How to add a Rocket component to the Starbase community collection.
---

## The short version

1. Fork the repository and run `go tool task new -- my-widget --category forms`.
2. Build your component in `components/my-widget/my-widget.js` and document it in `README.md`.
3. Run `go tool task live`, open your component's page, and check its examples. The dev server writes `manifest.json` for you.
4. Run `go tool task test` and open a pull request.

That's it. There is no Go code to touch: the gallery, search, categories and the docs page come from your folder.

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

## House rules

- **Shadow DOM and tokens.** Use Rocket's default `open` mode. Style only through `--sb-*` semantic tokens, each with a fallback: `--_bg: var(--sb-surface-card, #141D32)`. That makes the [Themes](/themes) page work, and the component still works outside Starbase.
- **State lives in `$$` signals, not attributes.** Datastar apps morph server HTML into the page. If your component writes interaction state back to its own attributes, the next morph resets it. Initialise `$$` from props, update `$$`, and expose live values with `overrideProp` getters.
- **Speak Datastar.** Emit bubbling, composed events with `emit()`, so `data-on:*` works on any ancestor. Expose a `value` or `checked` property and fire `change` or `input`, so `data-bind` works. In your docs, declare bound signals with `data-signals` first, and bind components with `__prop`: `data-bind:_x__prop.value`. Datastar binds before custom elements are upgraded, and without `__prop` it falls back to the `value` *attribute*, which the next server morph removes. Host getters and setters that read or write `$$` must wrap the access in `startPeeking()`/`stopPeeking()`. Otherwise `data-bind`'s own effect subscribes to your internal signal and writes stale values back.
- **Signal-driven attributes survive morphs only when preserved.** When a page drives a component with `data-attr:yaw="$_yaw"`, add `data-preserve-attr="yaw"` to the host. The server morph copies attributes from its own markup, and without this it would reset them on every frame.
- **Accessible by default.** Use native elements inside the shadow root, keyboard support, visible focus, and `prefers-reduced-motion`.
- **Document the API.** Give every prop `.docs({ description })` and declare slots and events in `manifest`. The API tables are generated from it.
- **No dependencies.** Import only from `datastar`. Keep it small.

## Review

A maintainer checks that the component renders in the gallery and on the Themes page, that `go test ./...` passes, and that the docs examples work. Components are published under the MIT licence with you as the author.
