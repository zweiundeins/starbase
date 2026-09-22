# Changelog

All notable changes to this project are documented here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and the project uses
[Semantic Versioning](https://semver.org/).

## [Unreleased]

### Fixed

- Playground: the preview theme picker applied the previous choice (the run started before the binding updated).

### Changed

- A command contract for value components (input, slider, toggle, code editor, tabs): a `name` prop and `sb-change {name, value}` on commit, ready to post as a command; with `confirm`, `:state(pending)` marks an edit the server hasn't confirmed, and `revert()` returns to the server's value after a rejected command. The Showcase demonstrates the loop (pending, a server-normalized value, a rejected command), and the Contribute page documents it for community components.
- The Contribute page covers components with several values: one structured `value` per decision (one event, one command), view state and server data as separate props, and `sb-<verb>` operations with per-item pending.
- The server can set every component's value: a `value`/`checked` attribute the server changes now wins over local edits (input, slider, toggle, code editor), while re-sent identical markup still leaves edits alone. Removed attributes are ignored (morphs also strip reflected ones), so to clear, send `value=""` / `checked="false"`. The manual pre-upgrade property adoption and the dirty flags are gone (Rocket replays those writes itself).
- `sb-alert` has an `open` prop, so the server can hide and re-show it. `sb-theme-switch` draws its icons as CSS masks instead of injecting SVG markup.
- The logo and the hero scene follow the site theme: they are inlined as SVG, and their themeable colours (brand stripes, planet, stars, smoke) are `--sb-art-*` tokens. As standalone files (favicon) they keep the original colours.
- GitHub Actions updated to their Node 24 (LTS) majors.

### Security

- Versioned, immutable component URLs (`/c/<slug>@<hash>/…`) and catalog snapshots (`/c/@<catalog>/autoloader.js`), with SHA-384 integrity for every file (`/c/@<catalog>/importmap.json`). Sites can pin exactly what they reviewed; browsers refuse any file that changed. Old versions keep working after later deploys.
- Vendored libraries are verified: `vendor.json` names each file's npm release, and the bot checks it byte for byte against the registry-verified tarball. Minified code that can't be verified is refused.
- The Starbase service listens on a Unix socket and is denied all of localhost (other services on a shared host), with a fully hardened systemd unit. `starbase-deploy` asks a new binary its version inside a network-less sandbox with a timeout.
- Pull request previews are limited to the repository's own submission branches; `script-src` is limited to `/static/` and `/c/`.
- Rate limits for snippet saves and painting (per session and per IP), and a cap on total snippet storage.
- CI: actions pinned to commit SHAs (Dependabot keeps them current), job timeouts, Chrome's sandbox kept on for submitted code, the bot's write-capable job re-validates what the build job produced, CODEOWNERS for vendored code, CI/CD and deploy files, and tests that keep the production environment to `deploy.yml`.

### Added

- `sb-theme-switch`: auto, dark or light (or any themes), as radio buttons, a select or a header menu. It remembers the choice in a cookie, so the server can render the theme before the first paint (no flash). The Starbase header uses it for the site's own themes, with "auto" following the system (Deep Space or Daylight).
- Continuous deployment: after CI passes on `main`, the Deploy workflow ships the binary over a single-purpose SSH key to `starbase-deploy`, which verifies it, checks `/healthz` and rolls back if the new version isn't healthy. The unit, env and Caddy files are in `deploy/`.
- Vendored libraries: a submission from a repository brings along every file the component imports relatively (up to 2 MB each), listed with its license banner in the PR. Imports are now checked strictly: `'datastar'` or files in the component's folder, never URLs or bare packages.
- `sb-nebula`, a WebGL nebula in dithered pixels: the first component that came in through the submission bot.
- `--sb-notch` and `data-sb-style="smooth"`: opt out of the 8-bit details (pixel corners, notched frames, pixel display font). The slider and toggle follow it, and the Themes page has a switch.
- Submission pull requests get a playground preview link for every revision (`/playground?preview=<commit>/<slug>`) and a list of similar existing components, to spot duplicates before merging.

### Fixed

- The playground can run components that import their own files (e.g. `sb-code-editor` and its vendored Prism): relative imports resolve against the component's folder.
- Submissions from a playground link failed validation: the link was recorded as the component's `source:`, which must be a GitHub repository.

## [0.1.0] - 2026-09-21

The first release: a community gallery of Rocket web components for Datastar.

### Added

- **Gallery.** Categories with counts, realtime full-text search (SQLite FTS5), sorting, and live previews. Per-tab filter state is kept on the server and mirrored in the URL.
- **Component pages.** Rendered docs with live `html preview` examples and copyable source, API tables generated from Rocket manifests, an auto-generated props **Playground**, installation snippets and "Edit on GitHub".
- **18 components:** button, input, slider, toggle, alert, modal, tooltip, meter, card, tabs, copy-button, code-editor (Prism with Rocket-aware highlighting), code-playground, voxel (a software 3D renderer), starfield, gauge, sparkline and pixel-board.
- **Code playground.** Edit a component's JavaScript and HTML with a sandboxed live preview (opaque-origin iframe), a console, theme switching, share links, and "Submit as component".
- **Showcase.** Mission Control, a dashboard driven by server-pushed signals (`/demo/telemetry`), and a **multiplayer pixel board** with live presence.
- **Themes.** An Open-Props-style token system (`--sb-*`), with four themes that restyle every component through shadow DOM.
- **Submitting components without tools.** A GitHub issue form (paste code, link a repository, or paste a playground link) and a bot that validates the component, generates its manifest in headless Chrome, and opens a pull request.
- **Autoloader.** `/c/autoloader.js` loads `<sb-*>` components on first use, follows morphs, knows dependencies between components, and offers `ready` and `sb-cloak` against the flash of undefined elements.
- **Architecture.** CQRS throughout: commands answered with 204, a single SQLite writer that batches commands, one render stream per tab that re-renders the whole page, and Brotli across frames. GitHub sign-in for stars.
- **Tooling.** Live reload (air), `task manifests`, a component scaffolder, CI (tests, templ check, manifest freshness, govulncheck, cross-platform builds, container image) and a release workflow.

### Known issues

- Datastar's morph is not re-entrant with Rocket components that are reordered by id. See `docs/repro/rocket-morph-reentrancy/` for a minimal reproduction and a proposed upstream fix. Starbase avoids the pattern.

[Unreleased]: https://github.com/zweiundeins/starbase/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/zweiundeins/starbase/releases/tag/v0.1.0
