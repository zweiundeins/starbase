---
title: About
lede: A community shelf of small, hypermedia-friendly web components for Datastar.
description: About Starbase, the Rocket community component collection.
---

## Why

[Datastar](https://data-star.dev) keeps state on the server and streams HTML to the browser. Some UI still wants a little local behavior: a switch, a tab bar, a dialog. [Rocket](https://data-star.dev/reference/rocket) is Datastar's own way to write that as a web component, with typed props, scoped signals and zero build step.

This site collects Rocket components the community has already built, so you can copy one, drop it into a Datastar app and move on.

## How this site works

The site is itself a Datastar app, built the way Datastar recommends: **CQRS** over a single SQLite database.

- **Commands:** starring a component, filtering the gallery or picking a theme is a short `POST`. It is validated, queued, and answers `204 No Content`.
- **One writer:** a single goroutine drains the queue, applies every queued command in one transaction, commits, and then marks only the affected pages for a new render.
- **Queries:** every open tab holds one long-lived SSE stream. After a relevant commit the server re-renders the whole page from one read snapshot, at most once every 50 ms however many commits came in, and sends it. Datastar morphs the difference, and zstd (or Brotli), sharing one window across frames, keeps resends tiny. A part that every viewer sees alike, such as the Showcase's pixel board, has a view of its own, which renders once for all of them. When the server stops for a deploy, the stream's last event has the page open it again on the new process, and `retry: 'error'` keeps asking until that process answers.
- **Server-owned UI state:** your filters and sort order live in SQLite per tab, not in client JavaScript. The URL stays in sync, so links are shareable.

Go, [templ](https://templ.guide), SQLite (pure Go, no cgo), Datastar and Rocket. No Node, no bundler.

## Credits

Built with inspiration from Anders Murphy's [hyperlith](https://github.com/andersmurphy/hyperlith) and zangster300's [northstar](https://github.com/zangster300/northstar). Pixel Art is generated in Go. Type is set in [Pixelify Sans](https://fonts.google.com/specimen/Pixelify+Sans) and [JetBrains Mono](https://www.jetbrains.com/lp/mono/), both under the SIL Open Font License. Interface icons are from [Lucide](https://lucide.dev) (ISC).
