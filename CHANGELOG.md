# Changelog

All notable changes to this project are documented here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and the project uses
[Semantic Versioning](https://semver.org/).

## [Unreleased]

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
