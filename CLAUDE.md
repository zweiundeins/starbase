# Starbase: notes for Claude

Community gallery for Datastar Rocket web components. Go 1.27, templ, SQLite (modernc, pure Go), Datastar v1.0.4 + Rocket (vendored in `static/vendor/datastar-rocket.js`).

## Commands
- `go tool task live`: dev server (air, `-tags dev`, live reload via `/dev/reload`)
- `go tool task test`: `go vet` + `go test ./...`. Run `go tool templ generate` after editing `.templ` (the `_templ.go` files are committed).
- `go tool task new -- <slug> --category <cat>`: scaffold a component
- `go tool task manifests`: regenerate all `manifest.json` in headless Chrome (`cmd/manifests`, runs the app in-process); `--check` in CI

## Deploy
- `deploy/`: host files (systemd unit, env, Caddy snippet), `starbase-deploy` (root; verifies, restarts, checks `/healthz`, rolls back) and `setup-host.sh` (the restricted `starbase-deploy` user with a forced-command key). `.github/workflows/deploy.yml` runs after CI on `main`, gated by the repository variable `DEPLOY_HOST`.

## Architecture rules (CQRS)
- State changes are **commands** (`internal/commands`): structs with `Apply(ctx, *sql.Tx)`, optional `Validate()`, and `Scope()` (session id) when only that session's views change. Handlers call `bus.Send` and return 204. Commands never render HTML.
- Only the `cqrs.Bus` writes to the database (single writer, `db.W`). Queries use `db.R` through `queries.Queries.View` (one read tx per render).
- Pages are `pageFunc`s in `internal/web` registered with `s.page(mux, pattern, fn)`, which gives the GET document and the POST render stream. A page must be a pure function of `renderCtx`.
- Per-tab UI state goes in `tab_state` via commands, never in handler memory or client-only signals.

## Frontend rules
- 8-bit details are opt-out: pixel-corner clip-paths scale with `--sb-notch` (1 or 0; at 0 use a border radius), frames with `--sb-frame-step`, display text uses `--sb-font-display`. `[data-sb-style="smooth"]` (theme.css) sets all three; the Themes page switch is `tab_state.PreviewSmooth`.
- CSS: `@layer reset, tokens, theme, base, layout, components, utilities`. Use semantic `--sb-*` tokens (theme.css) in components, not primitives. Prefer container queries over media queries.
- **Wire components declaratively:** events use `data-on:*` with local `action()`s (`@name()`), state is `$$` signals with `data-bind`/`data-show`/`data-class`/`data-attr`/`data-text`/`data-effect`, lists use `<template data-for>`, and elements come from `data-ref:x` → `onFirstRender({ refs })` (refs are not `$$` signals). `data-bind:x` in rendered markup binds the local `$$x`. Plain `addEventListener` is only for things without an attribute form (`matchMedia`, Intersection/ResizeObserver).
- Rocket components (`components/<slug>/`): shadow DOM (default mode), `adoptStyles(host, css)`, `--_x: var(--sb-x, fallback)` locals, interaction state in `$$` signals (never reflected to attributes, because server morphs reset attributes), `emit()` for events, `.docs()` on every prop.
- **Known upstream issue:** Datastar's morph is not re-entrant, and Rocket renders synchronously in `connectedCallback`. Never put `id`s on repeated or reordered elements that contain Rocket components (e.g. gallery cards). The morph would park and move them and crash (`Maximum call stack` / `moveBefore` HierarchyRequestError). Minimal repro and a proven fix are in `docs/repro/rocket-morph-reentrancy/`.
- `data-bind` on a custom element only writes to an existing signal. Declare it with `data-signals` first, and use `__prop.value` / `__prop.checked`: Datastar binds before Rocket upgrades the element, and otherwise falls back to the attribute, which morphs strip.
- Component host getters and setters over `$$` (`overrideProp`) must use `startPeeking()`/`stopPeeking()`, or `data-bind`'s effect subscribes to the internal signal and writes stale values back.
- Bindable components must adopt a property set before upgrade (`data-bind` writes `host.value` early): read and delete the own data property in `setup` (see `early()` in `components/slider/slider.js`).
- Form-like components (value/checked) follow native dirty semantics: the attribute is only the default; after an edit or a property write (`$$dirty`), attribute changes are ignored. Otherwise a morph that removes a reflected `value` attribute wipes the live value.
- Hosts whose attributes are driven by signals (`data-attr:x=...`) need `data-preserve-attr="x ..."`: the morph resets every attribute to the server markup.
- `tab_state` is for server-owned UI state (filters, sort, theme). High-frequency, ephemeral demo state (sliders, playgrounds) stays in local `_`-prefixed signals.
- Component pages render an auto Playground from the manifest (`internal/catalog/playground.go`, `ui.Playground`). It is a `data-ignore-morph` island driven by local `$_pg` signals, tuned by `playground:` front matter.
- `GET /demo/telemetry` (`internal/web/demo.go`) is a stateless query stream of `$_tm` signal patches for live demos.
- Canvas components: `renderOnPropChange: false`, repaint on `observeProps`, read theme tokens at paint time (via a 1×1 canvas probe), pause offscreen (IntersectionObserver) and honour `prefers-reduced-motion`.
- The CSP uses a nonce (import map) plus `'unsafe-eval'` (Datastar). No inline `<script>` without `s.Nonce`.

## Submissions
- `.github/ISSUE_TEMPLATE/new-component.yml` is the "Submit a component" form (`/submit` redirects to it). Its labels must match `internal/submission` (a test enforces it).
- `.github/workflows/component-from-issue.yml`: `build` (read-only, runs untrusted code only there: `cmd/fromissue` then `cmd/manifests`) → `pull-request` (write, never executes submitted code) / `report-failure` (comments on the issue).
- Submissions may link a public GitHub repo; `submission.Fetch` pins the commit, recorded as `source:` front matter (shown as a Source link). Playground links set no `source:`. Imports are checked for every submission (`submission.Imports`): only `'datastar'` and relative files inside the component folder; with a repo link the bot vendors every file the component reaches (≤ 2 MB each, 4 MB total) and lists them in the PR notes.
- The playground resolves relative imports in `component.js` against `base` (`/c/<slug>/`, or `/playground/preview/<commit>/<slug>/` which proxies the pinned commit's `.js` files).
- Every submission PR lists similar catalog components (`submission.Similar`, in the PR body) and gets a preview comment per revision: `/playground?preview=<commit>/<slug>` (`internal/web/preview.go`) fetches that commit's files from the repo on raw.githubusercontent.com, memoized per commit (immutable).

## Code playground
- `/playground` page (`internal/web/playground.go`, `ui.CodePlaygroundPage`): sources `?s=<snippet>`, `?component=<slug>` (JS + `Component.Examples`), or a starter. The editor area is a `data-ignore-morph` island; the share strip around it is stream-rendered from `tab_state.PlaygroundShare`.
- `sb-code-editor` highlights with Prism, vendored as an ES module in `components/code-editor/vendor/prism.js` (js-templates plugin, plus `/* css */` templates and `data-*` values as JS). Rebuild it with `go tool task vendor-prism`. Components may import `'datastar'` or files in their own folder; `/c/<slug>/**.js` is served.
- `/playground/run` is the sandbox runner (own CSP, opaque origin via `sandbox="allow-scripts"`). The protocol is documented in `components/code-playground/README.md`. Public assets send `Access-Control-Allow-Origin: *` for it.
- Rocket gotcha: before Datastar is ready, a later `rocket()` call for the same tag replaces the queued one, so the runner imports the edited code first and the parent drops deps whose tags the code defines.
- Snippets: `SaveSnippet` (≤ 64 KB, immutable, `snippets` table). `GET /playground/snippet/{id}` is public JSON. The submission bot imports playground links only from `STARBASE_URL`.

## Pixel board
- `sb-pixel-board` takes `cells` from the server (not preserved: every morph brings the latest). The board section on `/showcase` is **not** an ignore-morph island; Mission Control is.
- `PaintPixels` / `SeedBoard` in `internal/commands/board.go`; the encoded board is cached per version in `web.boardState`; `/cmd/paint` has an in-memory token bucket (20 px/s, burst 60).
- Paint posts use `requestCancellation: 'disabled'`: Datastar cancels in-flight requests to the same URL by default, which would drop pixels mid-stroke.
- `cqrs.Hub` knows each stream's path: `Count(path)` for presence; joins and leaves wake that path's streams.
- `<details>` inside morphed regions need `data-preserve-attr="open"`.
