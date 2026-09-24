# Changelog

All notable changes to this project are documented here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and the project uses
[Semantic Versioning](https://semver.org/).

## [Unreleased]

### Added

- `sb-odometer`: a number that rolls its digit wheels to each new value. Only the digits that change turn, and a wheel turns the way the number moves: over the top when it climbs (9 → 0), back when it falls. A gained digit (9.9 → 10.0) snaps to the new layout, separators follow the locale while the digits stay 0–9, with a linear roll as long as the update interval the wheels move continuously, like a car's, and `drum` draws each wheel as a 3D cylinder whose digits curve away and foreshorten as they turn. Screen readers get the value once, as text. From Libretto, where it runs the live-drive distance readout.
- `sb-button` takes `loading`: an inline pixel spinner (the same one `sb-busy` draws), clicks and Enter blocked against double submits, `aria-busy`, and focus kept. It costs no space until it is loading; bind it to `data-indicator`.
- `sb-echarts`: Apache ECharts driven by an option the server sends as JSON. It colours the chart from the `--sb-*` tokens (a string that is exactly `var(--token)` becomes that colour) and repaints on a theme change without starting again, writes numbers, months and weekdays in the page's language, keeps a wrapping legend clear of the axes, and animates each new option into place. Charts whose shape is code (a custom series' `renderItem`) are named kinds the page defines; a slotted table is the fallback for readers without the chart. ECharts 6.1.0 is vendored unmodified from npm and loaded only when a chart comes near the screen. From Libretto, where it draws every chart.
- `sb-theme-switch` takes a `domain`, so one theme choice can cover every subdomain. A domain the browser refuses falls back to this host instead of dropping the choice silently.
- `sb-theme-change` carries `scheme` (`"light"` or `"dark"`): what the page now paints in, worked out from the theme's own `color-scheme`, so code that draws needs no list of theme names. The theme switch's docs gain a section on following the theme from a canvas.
- `sb-autoloader`: the generic version of the site's autoloader, as a component. It takes a tag map and/or a URL pattern, loads any custom element the first time its tag appears (morphs included), pre-loads dependencies, and can un-cloak the page when the first round is defined.
- `sb-count-up`: counts a number to its value the first time it is on screen, and on from where it stands when the server sends a new one. The server's text inside it is what search engines and people without JavaScript see; screen readers get the final value, never the frames. From libretto.ch, whose version rendered nothing without JavaScript.

### Changed

- Components follow your page's font. Where one uses a font token (`sb-gauge`'s value, `sb-button variant="pixel"`, `sb-echarts`), it now falls back to the page's font instead of a hard-coded monospace, so outside Starbase they match the page without any configuration. `sb-gauge`'s value and label now scale with `--sb-gauge-size`, and its docs have a Styling section.
- Pages load every component as one file (`/c/bundle.js`, 66 kB) instead of one module per component. Measured on production over slow 4G, a visit that includes the homepage is 250–450 ms faster, at the cost of about 265 ms for a visitor who only sees one component page. Once the bundle grows past 100 kB (brotli), pages go back to the autoloader on their own.
- The Themes page remembers the previewed theme and the 8-bit switch for the session: they carry across pages and browser tabs, and render from the first paint. The gallery's sort is a session default too: a sort in the URL still wins (shared links show what was shared); without one, the gallery uses the last sort you chose. Search and category keep the sort and never change the default.
- Installation snippets end with a copyable `usage:` instead of the gallery card's `preview:`. It is a new optional front-matter field: the smallest markup that shows how you'd use the component in your own page (no card sizing, demo signals or timers, forced `open`, or images that only exist on Starbase), with `preview:` as the fallback. 30 components have one; a copied `sb-theme-switch`, for one, now themes the page instead of a demo attribute. A test keeps demo wiring and Starbase-only paths out of every component's installation markup.
- Assets are served precompressed at brotli -11 (gzip -9 as the fallback) instead of being compressed per request at level 5: every component module, the autoloader, the CSS bundles, the vendored Datastar build and the art. The size tables now show exactly the bytes a browser receives.
- Installation on component pages is four tabs instead of one long block: Autoloader (the default), This component (its minified module pinned with its frozen integrity, plus every component it renders), Pinned (today's catalog snapshot with an integrity import map) and Self-host (each file's minified and readable link with its size, and an import map at your own Datastar). Each tab has one sentence and a snippet that is exactly what to paste, with its own copy button. The chosen tab is remembered for the session, across component pages and browser tabs, through a new kind of state: session preferences (`session_prefs`), for choices that should outlive a page.
- Component modules are minified (esbuild) next to the readable source: every `x.js` has an `x.min.js`, and the autoloader and the site load those — about 23% less over the wire across the catalog. The readable file stays the default URL for the docs and the playground. Minified bytes are frozen per version, so an esbuild upgrade can never change a pinned URL. Size tables add a minified column, and gallery cards show the minified size.
- `sb-code-playground` is no longer listed in the gallery: it is the machinery behind `/playground` rather than a community component. It stays served, versioned and documented, through a new `unlisted:` front-matter flag.
- Live demos stand on a pixel grid: in the gallery the preview panel is visibly the component itself (and says "live demo" on hover), which is why it is the one part of a card that isn't the link to the component page. Demo stages in the docs use the same grid.

### Fixed

- Installation tabs: "This component" now pins everything it loads through the import map's `integrity` (a script tag covers only its own file, so code-editor's Prism was unchecked), and already-minified vendored files are no longer renamed to a non-existent `.min.min.js` — which had dropped ECharts from the Pinned tab's integrity and broken its Self-host link.
- `sb-echarts` wrote a missing value as “undefined” in tooltips and labels (its number formatter stringified whatever it was given); it now writes “-”, as ECharts does. Axis tooltips also leave out the series that have no value at the hovered point – a projection before it starts, costs in a month not yet lived – instead of listing them as empty rows; an option with its own formatter keeps it.
- `sb-odometer` `drum`: a digit sits high in its line box, so on a drum each rode above its face's centre and the neighbour above slid out of the window while the one below slid in. Digits and separators are trimmed to the cap height (`text-box`) and centred, so both neighbours show and the decimal mark keeps the baseline.
- `sb-sparkline` and `sb-gauge` kept the old theme's colours after a pick on the theme switch (and after the system flipped under "auto") until their data next changed: they read colours at paint time but were never asked to paint. They now repaint on `sb-theme-change` and on `prefers-color-scheme` changes.
- `sb-tree` lost the keyboard when the server dropped the focused row: the morph parks a row before removing it, so the focusout looked like the user leaving. Focus now moves to the neighbouring row, and is only given up when it really went somewhere else.
- Two console errors when the gallery's search morph removed a card: `sb-nebula` freed its GPU context in cleanup, which fired `webglcontextlost` after Rocket had torn the element down (the context events are wired with `addEventListener` now), and `sb-select`'s computed label read its chips signal while it was already gone.

### Security

- Dependencies updated (brotli 1.2.4, x/text 0.42), the container base image moved to distroless Debian 13, and Dependabot now watches the Dockerfile as well as the actions and Go modules.

## [0.2.0] - 2026-09-22

Eight new components (26 in total), a command contract that makes every value component CQRS-ready, continuous deployment, and a security, performance and SEO pass.

### Added

- **8 new components** (18 → 26):
  - `sb-nebula`: a drifting WebGL nebula in dithered pixels. The first component that came in through the submission bot.
  - `sb-theme-switch`: auto, dark or light (or any themes), as radio buttons, a select or a header menu. It remembers the choice in a cookie, so the server renders the theme before the first paint (no flash). The Starbase header uses it for the site's own themes, with "auto" following the system (Deep Space or Daylight).
  - `sb-rating`: pixel hearts or stars for a score, with half steps, hover preview and keys.
  - `sb-relative-time`: "3 minutes ago" that stays current, in any language, with a server fallback.
  - `sb-qr-code`: pixel QR codes for any text or URL, with optional brand-coloured corners (uqr, vendored and verified).
  - `sb-tree`: a keyboard-friendly tree whose branches load lazily from the server; `expanded` and `loaded` are props the server can morph.
  - `sb-select`: filter, pick several, or autocomplete from the server through a `results` prop.
  - `sb-range`: a two-thumb slider whose `{start, end}` is one value, committed as one command.
- **Command contract** for value components (input, slider, toggle, code editor, tabs, rating, tree, select, range): a `name` prop and `sb-change {name, value}` on commit, ready to post as a command. With `confirm`, `:state(pending)` marks an edit the server hasn't confirmed, and `revert()` returns to the server's value after a rejected command. The Showcase demonstrates the loop (pending, a server-normalized value, a rejected command).
- **Example dataset** (a seeded universe of galaxies, systems, planets and moons) with generic demo endpoints, `/demo/data/children` and `/demo/data/search`, for docs and playground demos (`into=` names the signal, `delay=` simulates latency).
- **Component sizes:** gallery cards show the download size (brotli, including the components it renders), and component pages have a Size table with original, gzip and brotli sizes for each file and each rendered component, plus a total. Datastar and Rocket are not counted.
- **Continuous deployment:** after CI passes on `main`, the Deploy workflow ships the binary over a single-purpose SSH key to `starbase-deploy`, which verifies it, checks `/healthz` and rolls back if the new version isn't healthy. The unit, env and Caddy files are in `deploy/`.
- **Vendored libraries:** a submission from a repository brings along every file the component imports relatively (up to 2 MB each), listed with its license banner in the PR. Imports are checked strictly: `'datastar'` or files in the component's folder, never URLs or bare packages.
- **Submission pull requests** get a playground preview link for every revision (`/playground?preview=<commit>/<slug>`) and a list of similar existing components, to spot duplicates before merging. When the pull request step fails, the bot tells the author on the issue.
- **Opt out of the 8-bit look:** `--sb-notch` and `data-sb-style="smooth"` turn off pixel corners, notched frames and the pixel display font. The Themes page has a switch.
- **SEO:** `robots.txt`, a sitemap, canonical URLs, Open Graph and Twitter cards, JSON-LD (`WebSite` with search, `SoftwareSourceCode` per component) and PNG icons rendered from the pixel art.
- **Contribute guidelines** for component authors: the command contract, components with several values (one structured `value` per decision, view state and server data as separate props, `sb-<verb>` operations), and no optimistic updates (server results only come from the server's render; the user's own input stays, marked pending).

### Changed

- The server can set every component's value: a `value`/`checked` attribute the server changes wins over local edits, while re-sent identical markup leaves edits alone. Removed attributes are ignored (morphs also strip reflected ones), so to clear, send `value=""` / `checked="false"`. The manual pre-upgrade property adoption and the dirty flags are gone (Rocket replays those writes itself).
- `sb-alert` has an `open` prop, so the server can hide and re-show it. `sb-theme-switch` draws its icons as CSS masks.
- The logo and the hero scene follow the site theme: they are inlined as SVG, and their themeable colours are `--sb-art-*` tokens. As standalone files (favicon) they keep the original colours.
- Performance: brotli, zstd or gzip for pages and text assets; one minified, hashed CSS bundle; hashed, preloaded fonts; immutable static files; pixel art drawn as one path per colour; lazy playground preview frames, and no layout shift on `/playground`.
- GitHub Actions updated to their Node 24 (LTS) majors.

### Fixed

- `sb-copy-button`: a refused clipboard write (no secure context, no permission, unfocused document) was silently ignored; it now shows "Copy failed" (`failed-label`) and emits `sb-copy-error {value, error}`. The result is announced through a `role="status"` region outside the button, and the tip is readable on light themes and no longer cut off on code blocks.
- Demo endpoints answer CORS preflights, so components in the playground sandbox (opaque origin) can load demo data with `@get`, e.g. the lazy tree.
- Playground: the preview theme picker applied the previous choice.
- The playground can run components that import their own files (e.g. `sb-code-editor` and its vendored Prism): relative imports resolve against the component's folder.
- Submissions from a playground link failed validation: the link was recorded as the component's `source:`, which must be a GitHub repository.
- The hero scene keeps its proportions at every width; `sb-theme-switch` icons line up with their labels; light-theme contrast fixes.

### Security

- Versioned, immutable component URLs (`/c/<slug>@<hash>/…`) and catalog snapshots (`/c/@<catalog>/autoloader.js`), with SHA-384 integrity for every file (`/c/@<catalog>/importmap.json`). Sites can pin exactly what they reviewed; browsers refuse any file that changed. Old versions keep working after later deploys.
- Vendored libraries are verified: `vendor.json` names each file's npm release, and the bot checks it byte for byte against the registry-verified tarball. Minified code that can't be verified is refused.
- The Starbase service listens on a Unix socket and is denied all of localhost (other services on a shared host), with a fully hardened systemd unit. `starbase-deploy` asks a new binary its version inside a network-less sandbox with a timeout.
- Pull request previews are limited to the repository's own submission branches; `script-src` is limited to `/static/` and `/c/`.
- Rate limits for snippet saves and painting (per session and per IP), and a cap on total snippet storage.
- CI: actions pinned to commit SHAs (Dependabot keeps them current), job timeouts, Chrome's sandbox kept on for submitted code, the bot's write-capable job re-validates what the build job produced, CODEOWNERS for vendored code, CI/CD and deploy files, and tests that keep the production environment to `deploy.yml`.

### Known issues

- Datastar's morph is not re-entrant with Rocket components that are reordered by id: reported upstream as [starfederation/datastar#1209](https://github.com/starfederation/datastar/issues/1209), with a minimal reproduction and a fix in `docs/repro/rocket-morph-reentrancy/`. Starbase avoids the pattern.

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

[Unreleased]: https://github.com/zweiundeins/starbase/compare/v0.2.0...HEAD
[0.2.0]: https://github.com/zweiundeins/starbase/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/zweiundeins/starbase/releases/tag/v0.1.0
