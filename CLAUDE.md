# Starbase: notes for Claude

Community gallery for Datastar Rocket web components. Go 1.26, templ, SQLite (modernc, pure Go), Datastar v1.0.4 + Rocket (vendored in `static/vendor/datastar-rocket.js`).

## Commands
- `go tool task live`: dev server (air, `-tags dev`, live reload via `/dev/reload`)
- `go tool task test`: `go vet` + `go test ./...`. Run `go tool templ generate` after editing `.templ` (the `_templ.go` files are committed).
- `go tool task new -- <slug> --category <cat>`: scaffold a component

## Architecture rules (CQRS)
- State changes are **commands** (`internal/commands`): structs with `Apply(ctx, *sql.Tx)`, optional `Validate()`, and `Scope()` (session id) when only that session's views change. Handlers call `bus.Send` and return 204. Commands never render HTML.
- Only the `cqrs.Bus` writes to the database (single writer, `db.W`). Queries use `db.R` through `queries.Queries.View` (one read tx per render).
- Pages are `pageFunc`s in `internal/web` registered with `s.page(mux, pattern, fn)`, which gives the GET document and the POST render stream. A page must be a pure function of `renderCtx`.
- Per-tab UI state goes in `tab_state` via commands, never in handler memory or client-only signals.

## Frontend rules
- CSS: `@layer reset, tokens, theme, base, layout, components, utilities`. Use semantic `--sb-*` tokens (theme.css) in components, not primitives. Prefer container queries over media queries.
- Rocket components (`components/<slug>/`): shadow DOM (default mode), `adoptStyles(host, css)`, `--_x: var(--sb-x, fallback)` locals, interaction state in `$$` signals (never reflected to attributes, because server morphs reset attributes), `emit()` for events, `.docs()` on every prop.
- **Known upstream issue:** Datastar's morph is not re-entrant, and Rocket renders synchronously in `connectedCallback`. Never put `id`s on repeated or reordered elements that contain Rocket components (e.g. gallery cards). The morph would park and move them and crash (`Maximum call stack` / `moveBefore` HierarchyRequestError). Minimal repro and a proven fix are in `docs/repro/rocket-morph-reentrancy/`.
- `data-bind` on a custom element only writes to an existing signal. Declare it with `data-signals` first.
- The CSP uses a nonce (import map) plus `'unsafe-eval'` (Datastar). No inline `<script>` without `s.Nonce`.
