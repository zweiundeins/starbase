# Starbase: notes for Claude

Community gallery for Datastar Rocket web components. Go 1.27, templ, SQLite (modernc, pure Go), Datastar v1.0.4 + Rocket (vendored in `static/vendor/datastar-rocket.js`).

## Commands
- `go tool task live`: dev server (air, `-tags dev`, live reload via `/dev/reload`)
- `go tool task test`: `go vet` + `go test ./...`. Run `go tool templ generate` after editing `.templ` (the `_templ.go` files are committed).
- `go tool task new -- <slug> --category <cat>`: scaffold a component
- `go tool task manifests`: regenerate all `manifest.json` in headless Chrome (`cmd/manifests`, runs the app in-process); `--check` in CI. Chrome keeps its sandbox (submitted code runs there); on machines without one, set `STARBASE_CHROME_NO_SANDBOX=1` (never in CI).

## Performance and SEO
- Responses to GET/HEAD are compressed (brotli/zstd/gzip, `web/compress.go`); render streams (POST) compress themselves and bypass it. Set `Vary` with `Add`, never `Set`.
- Hashed static files are `immutable`. `robots.txt`, `sitemap.xml` (pages + active components), `/og.png`, `/apple-touch-icon.png` and `/favicon.ico` (PNG, rendered from the pixel art: `pixelart/png.go`) are in `web/seo.go`. Every page has a canonical URL, Open Graph/Twitter tags and JSON-LD (`WebSite` with a search action; component pages add `SoftwareSourceCode` via `view.Schema`).

## Deploy
- `deploy/`: host files (systemd unit, env, Caddy snippet), `starbase-deploy` (root; verifies, restarts, checks `/healthz`, rolls back) and `setup-host.sh` (the restricted `starbase-deploy` user with a forced-command key). `.github/workflows/deploy.yml` runs after CI on `main`, gated by the repository variable `DEPLOY_HOST`.

## Architecture rules (CQRS)
- State changes are **commands** (`internal/commands`): structs with `Apply(ctx, *sql.Tx)`, optional `Validate()`, and `Scope()` (session id) when only that session's views change. Handlers call `bus.Send` and return 204. Commands never render HTML.
- Only the `cqrs.Bus` writes to the database (single writer, `db.W`). Queries use `db.R` through `queries.Queries.View` (one read tx per render).
- Pages are `pageFunc`s in `internal/web` registered with `s.page(mux, pattern, fn)`, which gives the GET document and the POST render stream. A page must be a pure function of `renderCtx`.
- Per-tab UI state goes in `tab_state` via commands, never in handler memory or client-only signals.
- Exception: the site theme is a browser preference, kept by `sb-theme-switch` in the `sb-theme` cookie. The layout renders it (`Shell.Theme`, `siteTheme`) on `<html data-sb-theme>`, so there is no flash. "auto" means no attribute: `:root` is deep-space, and `/theme/auto.css` (generated from the daylight block in `css/themes/showcase.css`) applies daylight on light systems.

## Frontend rules
- 8-bit details are opt-out: pixel-corner clip-paths scale with `--sb-notch` (1 or 0; at 0 use a border radius), frames with `--sb-frame-step`, display text uses `--sb-font-display`. `[data-sb-style="smooth"]` (theme.css) sets all three; the Themes page switch is `tab_state.PreviewSmooth`.
- CSS: `@layer reset, tokens, theme, base, layout, components, utilities`. Pages load one minified bundle (`/bundle/site.css`, built at startup from `siteStylesheets` in `web/assets.go`, font URLs rewritten to hashed names); a new stylesheet must be added there. The playground runner gets `/bundle/runner.css`.
- Syntax colours are semantic tokens (`--sb-code-keyword|function|tag|string|number`), with light values in the daylight block; don't use palette primitives (`--sb-violet-3`…) for anything that must work on light themes. Use semantic `--sb-*` tokens (theme.css) in components, not primitives. Prefer container queries over media queries.
- **Wire components declaratively:** events use `data-on:*` with local `action()`s (`@name()`), state is `$$` signals with `data-bind`/`data-show`/`data-class`/`data-attr`/`data-text`/`data-effect`, lists use `<template data-for>`, and elements come from `data-ref:x` → `onFirstRender({ refs })` (refs are not `$$` signals). `data-bind:x` in rendered markup binds the local `$$x`. Plain `addEventListener` is only for things without an attribute form (`matchMedia`, Intersection/ResizeObserver).
- Rocket components (`components/<slug>/`): shadow DOM (default mode), `adoptStyles(host, css)`, `--_x: var(--sb-x, fallback)` locals, interaction state in `$$` signals (never reflected to attributes, because server morphs reset attributes), `emit()` for events, `.docs()` on every prop.
- **Known upstream issue:** Datastar's morph is not re-entrant, and Rocket renders synchronously in `connectedCallback`. Never put `id`s on repeated or reordered elements that contain Rocket components (e.g. gallery cards). The morph would park and move them and crash (`Maximum call stack` / `moveBefore` HierarchyRequestError). Minimal repro and a proven fix are in `docs/repro/rocket-morph-reentrancy/` (upstream: starfederation/datastar#1209).
- `data-bind` on a custom element only writes to an existing signal. Declare it with `data-signals` first, and use `__prop.value` / `__prop.checked`: Datastar binds before Rocket upgrades the element, and otherwise falls back to the attribute, which morphs strip.
- Component host getters and setters over `$$` (`overrideProp`) must use `startPeeking()`/`stopPeeking()`, or `data-bind`'s effect subscribes to the internal signal and writes stale values back.
- The server can set every value: a form component's `value`/`checked` attribute **wins when the server sends a new one** (`observeProps` on that prop). Re-sent identical markup changes nothing, so user edits survive morphs. A *removed* attribute is ignored (`host.hasAttribute` check): morphs also strip attributes that were only reflected, e.g. by Rocket's upgrade replay of a `data-bind` write made before the element was defined. To clear, the server sends `value=""` / `checked="false"`. Property writes go through `overrideProp` into `$$` and never reflect. Rocket replays pre-upgrade property writes into `props` itself: no manual adoption needed.
- Command contract for value components (input, slider, toggle, code-editor, tabs; rating, select, tree): a `name` prop; `sb-change {name, value}` on commit; `confirm` sets `:state(pending)` (ElementInternals custom state, `internalsOf(host)` caches the internals since `setup` reruns on re-attach) while `$$` differs from the server's attribute; `host.revert()` restores the server's value. Pages: `data-on:datastar-fetch="evt.detail.el === el && evt.detail.type === 'error' && el.revert()"` (fetch events reach every listener). Demo: `SetFlight` / `POST /cmd/flight` / `ui.FlightDemo` on the Showcase.
- Several values: parts of one decision are one structured `value` (JSON attribute, one `sb-change`, one command; e.g. `sb-range` `{start, end}`); view state (`expanded`) is a separate prop and event, never pending; server data (`results`, `options`) only flows in; operations (paint, move) emit `sb-<verb>` with per-item pending until the echo.
- Server data for components are props the server can morph (e.g. `sb-select` `results`, `sb-tree` `loaded`/`expanded`), not only signals.
- Hosts whose attributes are driven by signals (`data-attr:x=...`) need `data-preserve-attr="x ..."`: the morph resets every attribute to the server markup.
- Pixel art (`internal/pixelart`) is generated SVG. Theme-aware colours are `var(--sb-art-*, #fallback)` (set per theme in `css/themes/showcase.css`; Deep Space uses the fallbacks), which only apply where the SVG is inlined with `ui.Art` (logo, hero). As `<img>`/favicon the fallbacks show. Its CSS is scoped under `.px-art`.
- `tab_state` is for server-owned UI state (filters, sort, theme). High-frequency, ephemeral demo state (sliders, playgrounds) stays in local `_`-prefixed signals.
- Component pages render an auto Playground from the manifest (`internal/catalog/playground.go`, `ui.Playground`). It is a `data-ignore-morph` island driven by local `$_pg` signals, tuned by `playground:` front matter.
- `GET /demo/telemetry` (`internal/web/demo.go`) is a stateless query stream of `$_tm` signal patches for live demos.
- Demo data for components' docs comes from the example dataset (`internal/demo`, seeded by `SeedDemo`): use the generic `GET /demo/data/children` and `/demo/data/search` (`internal/web/demo_data.go`, `&into=` names the signal, `&delay=` for loading states) instead of adding component-specific endpoints.
- Canvas components: `renderOnPropChange: false`, repaint on `observeProps`, read theme tokens at paint time (via a 1×1 canvas probe), pause offscreen (IntersectionObserver) and honour `prefers-reduced-motion`.
- The CSP uses a nonce (import map) plus `'unsafe-eval'` (Datastar). No inline `<script>` without `s.Nonce`. `script-src` allows only `<origin>/static/` and `<origin>/c/`, not `'self'`: new script locations must be added there.
- Rate limits (`limiter`, per session and per `clientIP`, which trusts `X-Real-IP` only from loopback/Unix-socket peers): painting, snippet saves. Snippet storage is capped (`MaxSnippetStore`, tracked in `snippet_stats`).

## Submissions
- `.github/ISSUE_TEMPLATE/new-component.yml` is the "Submit a component" form (`/submit` redirects to it). Its labels must match `internal/submission` (a test enforces it).
- `.github/workflows/component-from-issue.yml`: `build` (read-only, runs untrusted code only there: `cmd/fromissue` then `cmd/manifests`) → `pull-request` (write, never executes submitted code) / `report-failure` (comments on the issue).
- Submissions may link a public GitHub repo; `submission.Fetch` pins the commit, recorded as `source:` front matter (shown as a Source link). Playground links set no `source:`. Imports are checked for every submission (`submission.Imports`): only `'datastar'` and relative files inside the component folder; with a repo link the bot vendors every file the component reaches (≤ 2 MB each, 4 MB total). Vendored files are proven via `vendor.json` (`submission.VerifyVendor`: byte-for-byte against the npm tarball, which must match the registry's integrity hash); unverified minified code is refused, unverified readable code is flagged in the PR notes. CODEOWNERS covers `components/**/vendor/`.
- The playground resolves relative imports in `component.js` against `base` (`/c/<slug>/`, or `/playground/preview/<commit>/<slug>/` which proxies the pinned commit's `.js` files).
- Every submission PR lists similar catalog components (`submission.Similar`, in the PR body) and gets a preview comment per revision: `/playground?preview=<commit>/<slug>` (`internal/web/preview.go`) fetches that commit's files from the repo on raw.githubusercontent.com, memoized per commit (immutable).

## Versioned URLs
- `internal/web/versions.go`: `/c/<slug>@<hash>/<file>` (a component version: current from the binary, older from `component_files`), `/c/@<catalog hash>/autoloader.js` and `/importmap.json` (SRI map). All immutable, `ACAO *`. `SyncCatalog` stores every version and catalog snapshot it sees (migration 005), so pinned URLs outlive deploys.
- The site uses versioned module URLs too (`assets.ComponentScript`, `catalog.AutoloaderJS`); the old `/c/<slug>/<file>` route stays for compatibility.

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
