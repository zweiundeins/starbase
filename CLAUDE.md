# Starbase: notes for Claude

Community gallery for Datastar Rocket web components. Go 1.26, templ, SQLite (modernc, pure Go), Datastar v1.0.4 + Rocket (vendored in `static/vendor/datastar-rocket.js`).

## Commands
- `go tool task live`: dev server (air, `-tags dev`, live reload via `/dev/reload`)
- `go tool task test`: `go vet` + `go test ./...`. Run `go tool templ generate` after editing `.templ` (the `_templ.go` files are committed).
- `go tool task new -- <slug> --category <cat>`: scaffold a component
- `go tool task manifests`: regenerate all `manifest.json` in headless Chrome (`cmd/manifests`, runs the app in-process); `--check` in CI

## Architecture rules (CQRS)
- State changes are **commands** (`internal/commands`): structs with `Apply(ctx, *sql.Tx)`, optional `Validate()`, and `Scope()` (session id) when only that session's views change. Handlers call `bus.Send` and return 204. Commands never render HTML.
- Only the `cqrs.Bus` writes to the database (single writer, `db.W`). Queries use `db.R` through `queries.Queries.View` (one read tx per render).
- Pages are `pageFunc`s in `internal/web` registered with `s.page(mux, pattern, fn)`, which gives the GET document and the POST render stream. A page must be a pure function of `renderCtx`.
- Per-tab UI state goes in `tab_state` via commands, never in handler memory or client-only signals.

## Frontend rules
- CSS: `@layer reset, tokens, theme, base, layout, components, utilities`. Use semantic `--sb-*` tokens (theme.css) in components, not primitives. Prefer container queries over media queries.
- **Wire components declaratively:** events use `data-on:*` with local `action()`s (`@name()`), state is `$$` signals with `data-bind`/`data-show`/`data-class`/`data-attr`/`data-text`/`data-effect`, lists use `<template data-for>`, and elements come from `data-ref:x` → `onFirstRender({ refs })` (refs are not `$$` signals). `data-bind:x` in rendered markup binds the local `$$x`. Plain `addEventListener` is only for things without an attribute form (`matchMedia`, Intersection/ResizeObserver).
- Rocket components (`components/<slug>/`): shadow DOM (default mode), `adoptStyles(host, css)`, `--_x: var(--sb-x, fallback)` locals, interaction state in `$$` signals (never reflected to attributes, because server morphs reset attributes), `emit()` for events, `.docs()` on every prop.
- **Known upstream issue:** Datastar's morph is not re-entrant, and Rocket renders synchronously in `connectedCallback`. Never put `id`s on repeated or reordered elements that contain Rocket components (e.g. gallery cards). The morph would park and move them and crash (`Maximum call stack` / `moveBefore` HierarchyRequestError). Minimal repro and a proven fix are in `docs/repro/rocket-morph-reentrancy/`.
- `data-bind` on a custom element only writes to an existing signal. Declare it with `data-signals` first, and use `__prop.value` / `__prop.checked`: Datastar binds before Rocket upgrades the element, and otherwise falls back to the attribute, which morphs strip.
- Component host getters and setters over `$$` (`overrideProp`) must use `startPeeking()`/`stopPeeking()`, or `data-bind`'s effect subscribes to the internal signal and writes stale values back.
- Bindable components must adopt a property set before upgrade (`data-bind` writes `host.value` early): read and delete the own data property in `setup` (see `early()` in `components/slider/slider.js`).
- Hosts whose attributes are driven by signals (`data-attr:x=...`) need `data-preserve-attr="x ..."`: the morph resets every attribute to the server markup.
- `tab_state` is for server-owned UI state (filters, sort, theme). High-frequency, ephemeral demo state (sliders, playgrounds) stays in local `_`-prefixed signals.
- Component pages render an auto Playground from the manifest (`internal/catalog/playground.go`, `ui.Playground`). It is a `data-ignore-morph` island driven by local `$_pg` signals, tuned by `playground:` front matter.
- `GET /demo/telemetry` (`internal/web/demo.go`) is a stateless query stream of `$_tm` signal patches for live demos.
- Canvas components: `renderOnPropChange: false`, repaint on `observeProps`, read theme tokens at paint time (via a 1×1 canvas probe), pause offscreen (IntersectionObserver) and honour `prefers-reduced-motion`.
- The CSP uses a nonce (import map) plus `'unsafe-eval'` (Datastar). No inline `<script>` without `s.Nonce`.

## Submissions
- `.github/ISSUE_TEMPLATE/new-component.yml` is the "Submit a component" form (`/submit` redirects to it). Its labels must match `internal/submission` (a test enforces it).
- `.github/workflows/component-from-issue.yml`: `build` (read-only, runs untrusted code only there: `cmd/fromissue` then `cmd/manifests`) → `pull-request` (write, never executes submitted code) / `report-failure` (comments on the issue).
- Submissions may link a public GitHub repo; `submission.Fetch` pins the commit, recorded as `source:` front matter (shown as a Source link).
