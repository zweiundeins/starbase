---
name: Typewriter
tag: sb-typewriter
category: utilities
summary: Types its text out behind a blinking cursor, or cycles through phrases.
author: zweiundeins
tags: [typing, typewriter, text, animation, hero, terminal]
since: 2026-10-06
preview: |
  <sb-typewriter prompt="$ " phrases='["ignition","lift-off","orbit"]' loop style="font-size: 1.5rem; font-family: var(--sb-font-display)">ignition</sb-typewriter>
usage: |
  <sb-typewriter prompt="$ ">Ready for launch.</sb-typewriter>
playground:
  props:
    interval: {min: 5, max: 200, step: 5}
    delay: {min: 0, max: 3000, step: 100}
    hold: {min: 0, max: 5000, step: 100}
  values: {prompt: "$ "}
  content: All systems nominal. Ready for launch.
  style: "font-size: 1.5rem"
---

Types its text out, character by character, behind a blinking cursor: a hero line, a terminal prompt, a tagline that keeps changing. It takes its font, size and colour from where you put it.

Put the text inside it. That is what search engines and readers without JavaScript see, and what screen readers announce, all at once rather than letter by letter. The finished text is laid out from the start, invisibly, so the line breaks never change and nothing around it moves while it types. It starts when it comes into view and pauses while it is out of it; with reduced motion, the text is simply there.

## Examples

### A command line

`prompt` is there from the start, the text after it types out. The cursor stays and blinks when it is done.

```html preview
<p style="font-size: 1.125rem; margin: 0">
  <sb-typewriter prompt="$ " interval="35">starbase deploy --to orbit</sb-typewriter>
</p>
```

### Phrases

`phrases` types each phrase in turn and erases it before the next; `loop` starts over after the last. The element takes the size of the longest phrase from the start, so a heading under it never jumps. Keep the text inside it: it is the one readers get without JavaScript, and the one screen readers announce.

```html preview
<h3 style="font-size: 2rem; margin: 0">
  Next stop:
  <sb-typewriter phrases='["the Moon.","Mars.","Europa.","wherever the data goes."]' loop hold="1400" style="color: var(--sb-brand-light)">the Moon.</sb-typewriter>
</h3>
```

`sb-typed` fires each time a phrase is complete, with its `text` and `index`; with reduced motion, once for the phrase shown.

### Cursors

`cursor` is `block` (the default), `bar`, `underscore` or `none`.

```html preview
<div style="display: grid; gap: 8px; font-size: 1.25rem">
  <sb-typewriter cursor="block">Block cursor</sb-typewriter>
  <sb-typewriter cursor="bar">Bar cursor</sb-typewriter>
  <sb-typewriter cursor="underscore">Underscore cursor</sb-typewriter>
  <sb-typewriter cursor="none">No cursor</sb-typewriter>
</div>
```

### Text from the server

New text types again: a morph that changes the element's text, or `data-text` from a signal. Here the buttons stand in for the server.

```html preview
<div data-signals="{_status: 'Awaiting telemetry…'}" style="display: grid; gap: 12px; justify-items: start">
  <p style="margin: 0"><sb-typewriter prompt="> " interval="25" data-text="$_status">Awaiting telemetry…</sb-typewriter></p>
  <div style="display: flex; gap: 8px">
    <sb-button size="sm" data-on:click="$_status = 'Signal acquired. 4 satellites in view.'">Acquire</sb-button>
    <sb-button size="sm" variant="ghost" data-on:click="$_status = 'Signal lost.'">Lose</sb-button>
  </div>
</div>
```

## Styling

Style it from your page's CSS, without changing the component or importing anything into it. Custom properties, inherited properties and `::part()` all reach into its shadow root.

- **Fonts and size:** the text uses your page's font, size and colour; it sets none of its own.
- **Colours:** `--sb-typewriter-cursor` colours the cursor (by default `--sb-brand-light`).
- **Parts:** `text` (the whole line), `prompt` and `cursor`. The cursor moves with `translate`, so the page never counts a layout shift while it types. Your page's `::part()` rules win over the component's own, without `!important`.

```html preview
<style>
  .my-type { --sb-typewriter-cursor: #FFD166; font: 600 1.5rem Georgia, serif; }
  .my-type::part(prompt) { color: #FFD166; }
</style>
<sb-typewriter class="my-type" prompt="» " cursor="bar">A quieter kind of terminal.</sb-typewriter>
```

## Accessibility

Screen readers get the element's text in one piece, once: the typing, the cursor and the prompt are hidden from them. With phrases they get the element's own text rather than every phrase, so keep it meaningful. With `prefers-reduced-motion`, nothing types, cycles or blinks: the first phrase is there from the start.
