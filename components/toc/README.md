---
name: Table of Contents
tag: sb-toc
category: navigation
summary: The sections of a long page, the one being read marked, with reading progress.
author: zweiundeins
tags: [toc, table of contents, navigation, scroll spy, progress, article]
since: 2026-10-06
preview: |
  <div style="display: grid; grid-template-columns: 7.5rem 1fr; gap: 12px; inline-size: 16rem; font-size: 11px">
    <sb-toc label="Log" start-label="Countdown" progress>
      <ol><li><a href="#pv-toc-launch">Launch</a></li><li><a href="#pv-toc-orbit">Orbit</a></li><li><a href="#pv-toc-landing">Landing</a></li></ol>
    </sb-toc>
    <div style="display: grid; gap: 6px; opacity: 0.8"><b id="pv-toc-launch">Launch</b><span>T+0: lift-off.</span><b id="pv-toc-orbit">Orbit</b><span>Burn complete.</span><b id="pv-toc-landing">Landing</b><span>Tranquility.</span></div>
  </div>
usage: |
  <sb-toc content="article" levels="h2 h3" start-label="Intro" compact="(max-width: 64rem)"></sb-toc>
playground:
  values: {startLabel: Top, levels: h2}
  style: "inline-size: min(100%, 16rem)"
---

The sections of a long page: a blog post, documentation, a report. The entry you are reading is marked as you scroll, a link jumps to its section, and a bar shows how far through you are. Put it in a sticky sidebar. On narrow screens, `compact` folds it into a bar with the current section, and the list opens from there.

It lists the headings of `content` (by default the closest `<article>`, else `<main>`), `levels` deep. Headings without an `id` get one from their text. **Or render the list on the server:** put an `<ol>` (or `<ul>`) of `#id` links inside the element and it uses those. That list is also what readers and crawlers see without JavaScript; nested lists become indented levels.

## Examples

### Beside an article

The playground above lists the sections of this page. A sidebar usually sits in a grid next to the text, sticky, and no taller than the viewport: the list scrolls on its own, keeping the current entry in view.

```html preview
<div style="display: grid; grid-template-columns: minmax(0, 11rem) minmax(0, 1fr); gap: 32px; align-items: start; inline-size: 100%">
  <sb-toc content="#mission-log" levels="h3 h4" start-label="Countdown" progress
    style="position: sticky; inset-block-start: 5rem; max-block-size: calc(100vh - 7rem)"></sb-toc>
  <article id="mission-log" style="display: grid; gap: 12px; max-inline-size: 34rem">
    <p>T−10 minutes. Tanks pressurised, guidance aligned, the crew strapped in. Weather is go.</p>
    <h3>Launch</h3>
    <p>Main engines ignite at T−6 seconds. Hold-down arms release, and the stack clears the tower in eight seconds, rolling onto its heading.</p>
    <p>Max-Q at one minute ten: the hardest push of the ascent. Engines throttle down, then back up.</p>
    <h4>Staging</h4>
    <p>The first stage cuts off at two minutes forty and falls away. The second stage lights cleanly; the escape tower is jettisoned.</p>
    <h3>Orbit</h3>
    <p>Insertion at eleven minutes, 190 kilometres up. Two orbits of checks before the burn that leaves Earth behind.</p>
    <h4>Translunar injection</h4>
    <p>A six-minute burn over the Pacific. The planet shrinks in the window to something you can cover with a thumb.</p>
    <h3>Landing</h3>
    <p>Powered descent from fifteen kilometres. Program alarms, a boulder field, the last seconds of fuel, and then contact light.</p>
    <h3>Return</h3>
    <p>Re-entry at eleven kilometres per second behind a heat shield, three parachutes, and a splash in the Pacific.</p>
  </article>
</div>
```

`start-label` adds a first entry that leads back to the start of the content; it is the current one until the first heading scrolls past. The counter shows where you are (`3/7`), and `progress` adds the reading progress under the heading.

### A list from the server

The links are the server's: their text, their order, their nesting. The component reads them, marks the current one and keeps reading them when a morph changes the list.

```html preview
<sb-toc label="On this page" style="inline-size: 14rem">
  <ol>
    <li><a href="#beside-an-article">Beside an article</a></li>
    <li><a href="#a-list-from-the-server">A list from the server</a></li>
    <li><a href="#compact">Compact</a></li>
    <li><a href="#styling">Styling</a>
      <ol><li><a href="#accessibility">Accessibility</a></li></ol>
    </li>
  </ol>
</sb-toc>
```

### Compact

`compact` takes a media query. While it matches, the list folds into a bar: the label, the current section, the counter and the reading progress along its edge. The bar opens the list as a popover; picking a section, Escape or a click outside closes it. `compact="all"` keeps it folded everywhere, as here.

```html preview
<sb-toc compact="all" content="#mission-log" levels="h3" start-label="Countdown" style="inline-size: min(100%, 22rem)"></sb-toc>
```

On a phone, make the bar sticky at the top of the article: the reader always sees where they are and can jump anywhere.

### Following the reader

`sb-section` fires when the reader scrolls into another section, with its `id` (empty before the first heading). Here it drives a line of text:

```html preview
<div data-signals="{_section: ''}" style="display: grid; gap: 8px; inline-size: min(100%, 22rem)">
  <sb-toc compact="all" content="#mission-log" levels="h3" data-on:sb-section="$_section = evt.detail.id"></sb-toc>
  <code data-text="$_section ? 'Reading #' + $_section : 'Scroll the mission log…'"></code>
</div>
```

## Styling

Style it from your page's CSS, without changing the component or importing anything into it. Custom properties, inherited properties and `::part()` all reach into its shadow root.

- **Size and place:** it is as wide as its container. For a sticky sidebar, set `position: sticky`, an `inset-block-start` and a `max-block-size` on the element; the list scrolls inside it.
- **Where "current" starts:** a section is current once its heading passes 30% of the way down the viewport, below the page's `scroll-padding-top`. Set that on `html` for a fixed header; links land below it too.
- **8-bit details:** the label and the counter use `--sb-font-display`, the marker on the rail is square while `--sb-notch` is 1 (round at 0), the progress bar is made of blocks (a plain bar at 0), and the compact bar and its list have a stepped frame `--sb-frame-step` thick. `data-sb-style="smooth"` turns them all off.
- **Colours:** entries are `--sb-text-2`, the current one `--sb-text-1` with a `--sb-brand` marker, on a `--sb-border` rail; hover is `--sb-surface-hover`. The label is `--sb-text-muted`. The compact bar and list are `--sb-surface-card` and cast `--sb-shadow-overlay`; the focus ring is `--sb-focus-ring`.
- **Parts:** `nav`, `head`, `label`, `counter`, `progress`, `list`, `link` (every entry), `bar` and `panel` (the compact list). Your page's `::part()` rules win over the component's own, without `!important`.

```html preview
<style>
  .my-toc { --sb-brand: #FFD166; --sb-notch: 0; --sb-font-display: Georgia, serif; inline-size: 14rem; }
  .my-toc::part(label) { text-transform: none; letter-spacing: 0; font-size: 1rem; }
  .my-toc::part(link) { font-size: 0.9375rem; }
</style>
<sb-toc class="my-toc" label="In this log" content="#mission-log" levels="h3"></sb-toc>
```

## Accessibility

It is a `<nav>` landmark named by `label`. The current entry has `aria-current="location"`, so screen readers announce it on the link. Links are ordinary anchors: they move the focus to the section as any in-page link does, add a history entry, and follow the page's `scroll-behavior` (so `prefers-reduced-motion` can turn smooth scrolling off). The start entry scrolls to the content without adding a `#` to the address.

The compact bar is a button that opens the list as a popover: the browser exposes it as expanded or collapsed, Escape closes the list and returns the focus to the bar. The counter and the progress bar are hidden from screen readers; the current section is in the bar's text.
