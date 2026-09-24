---
name: Busy
tag: sb-busy
category: feedback
summary: Spinner, bar or skeleton that follows Datastar's in-flight requests.
author: zweiundeins
tags: [loading, spinner, skeleton, progress, indicator, fetch]
since: 2026-09-23
preview: |
  <sb-busy busy show-label label="Plotting course"></sb-busy>
usage: |
  <button data-on:click="@post('/save')">Save</button>
  <sb-busy label="Saving" show-label></sb-busy>
playground:
  props:
    lines: {min: 1, max: 8}
    delay: {min: 0, max: 2000, step: 50}
    min: {min: 0, max: 2000, step: 50}
  values: {busy: true, variant: spinner, label: Plotting course, showLabel: true}
---

One loading indicator in three shapes, and it knows when the page is waiting. `sb-busy` listens for Datastar's `datastar-fetch` events and shows itself while a request it watches is in flight, so a button plus an indicator needs no signal, no `data-indicator` and no server round trip to appear.

Two things decide whether it is up, and they are kept strictly apart:

- **The server owns `busy`.** A long job the server knows about sets `busy` on the element, and it wins: the indicator stays up until the server takes the attribute away (`busy="false"`).
- **The component owns the rest locally.** Which watched requests are in flight, the `delay`/`min` timers and the animation state live in `$$` signals and closure variables, never in attributes: a morph resets attributes, and the indicator must not blink every time the server re-renders the page.

The visible state is `busy` **or** a watched request in flight, put through `delay` and `min`. Every flip emits `sb-busy-change` with `{ busy }`.

## Examples

### The three variants

```html preview
<div style="display: grid; gap: 24px; inline-size: min(100%, 24rem)">
  <sb-busy busy show-label label="Docking"></sb-busy>
  <sb-busy busy variant="bar" label="Uploading"></sb-busy>
  <sb-busy busy variant="bar" value="64" label="Uploading" show-label></sb-busy>
  <sb-busy busy variant="skeleton" lines="3" label="Loading crew"></sb-busy>
</div>
```

A `bar` with no `value` sweeps (indeterminate); with `value` between 0 and 100 it becomes a determinate `role="progressbar"`. Sizes are `sm`, `md` and `lg`:

```html preview
<div style="display: flex; align-items: center; gap: 24px">
  <sb-busy busy size="sm"></sb-busy>
  <sb-busy busy size="md"></sb-busy>
  <sb-busy busy size="lg"></sb-busy>
</div>
```

### Server-driven

`busy` is an ordinary attribute, so anything that can set an attribute can drive it: the server's markup, or a signal on the page.

```html preview
<div data-signals:_working="false" style="display: flex; align-items: center; gap: 16px">
  <sb-toggle label="Long job running" data-bind:_working__prop.checked></sb-toggle>
  <sb-busy data-attr:busy="$_working" data-preserve-attr="busy" show-label label="Working"></sb-busy>
</div>
```

### Following a real request

This is the point of the component. The button fires a request at a demo endpoint that answers after 900 ms; the `sb-busy` next to it shows itself for exactly as long as the request is in flight. Nothing is wired between them: with no `for`, the indicator watches every request coming from inside its own parent element.

```html preview
<div data-signals="{_crew: []}" style="display: grid; gap: 16px; justify-items: start">
  <div style="display: flex; align-items: center; gap: 16px">
    <sb-button data-on:click="@get('/demo/data/children?into=_crew&delay=900')">Load the catalog</sb-button>
    <sb-busy label="Loading the catalog" show-label></sb-busy>
  </div>
  <p style="margin: 0; color: var(--sb-text-2)" data-text="$_crew.length ? $_crew.length + ' galaxies' : 'nothing loaded yet'"></p>
</div>
```

`datastar-fetch` is dispatched on the document and reaches *every* listener, so `sb-busy` checks `event.detail.el` when a request starts. Requests from elsewhere on the page leave it alone. From then on the request belongs to the element that fired it, and its end counts even if that element is gone by then: a Delete button whose row the response removes still takes the indicator down.

### Which requests to watch

| `for` | watches |
| --- | --- |
| *empty* | any request triggered from inside the host's parent element (the host itself included) |
| a CSS selector | requests whose triggering element matches it, or has an ancestor that does (`for="#crew-form"` covers every control in that form) |
| `*` | every request on the page |

```html preview
<div data-signals="{_a: [], _b: []}" style="display: grid; gap: 16px">
  <div id="sb-busy-left" style="display: flex; align-items: center; gap: 16px">
    <sb-button variant="outline" data-on:click="@get('/demo/data/children?into=_a&delay=900')">Left request</sb-button>
    <span style="color: var(--sb-text-2); font-size: 0.8125rem">watched</span>
  </div>
  <div style="display: flex; align-items: center; gap: 16px">
    <sb-button variant="outline" data-on:click="@get('/demo/data/search?q=star&into=_b&delay=900')">Right request</sb-button>
    <span style="color: var(--sb-text-2); font-size: 0.8125rem">ignored</span>
  </div>
  <sb-busy for="#sb-busy-left" variant="bar" label="Loading the left panel" show-label></sb-busy>
</div>
```

A skeleton usually stands in for the content it is waiting for, so give it the shape of that content and hide the content while it is up:

```html preview
<div data-signals="{_moons: []}" style="display: grid; gap: 16px">
  <sb-button data-on:click="@get('/demo/data/search?q=moon&into=_moons&delay=1200')">Scan for moons</sb-button>
  <sb-busy variant="skeleton" lines="4" label="Scanning"></sb-busy>
  <ul data-show="$_moons.length" style="margin: 0; color: var(--sb-text-2)">
    <template data-for="m in $_moons"><li data-text="m?.label"></li></template>
  </ul>
</div>
```

### A spinner inside the button

The most common loading state there is. For a button, reach for [`sb-button`](/components/button)'s own `loading` prop first: it draws the same eight blinking dots as `sb-busy`, sized from the button's own text and colour, and it blocks clicks (mouse, Enter or Space) in the capture phase — which is what stops a double submit, and which a spinner sitting inside the button cannot do.

```html preview
<div data-signals="{_synced: []}" style="display: flex; align-items: center; gap: 16px">
  <sb-button
    data-indicator:_syncing
    data-attr:loading="$_syncing"
    data-preserve-attr="loading"
    data-on:click="@get('/demo/data/children?into=_synced&delay=900')"
  >Sync catalog</sb-button>
  <span style="color: var(--sb-text-2); font-size: 0.8125rem" data-text="$_synced.length ? 'synced ' + $_synced.length + ' galaxies' : 'not synced yet'"></span>
</div>
```

`data-indicator` sets the signal while that element's request is in flight, and `data-preserve-attr` keeps the attribute through a server morph. That is one signal, and in exchange the button cannot be clicked twice.

**Or with no signal at all.** Drop an `sb-busy` into the button instead and it wires itself: it watches the request the button fires and shows itself for exactly as long as it is in flight.

```html preview
<div data-signals="{_counted: []}" style="display: flex; align-items: center; gap: 16px">
  <sb-button variant="outline" data-on:click="@get('/demo/data/children?into=_counted&delay=900')">
    <sb-busy size="sm" label="Counting" style="display: contents; --sb-brand: currentColor"></sb-busy>
    Count galaxies
  </sb-button>
  <span style="color: var(--sb-text-2); font-size: 0.8125rem" data-text="$_counted.length ? 'counted ' + $_counted.length + ' galaxies' : 'not counted yet'"></span>
</div>
```

Which one: `sb-button loading` when a click starts a command, because only the button itself can swallow the second click; the composed `sb-busy` when you want no page state at all, or a shape `sb-button` does not draw — a bar, a skeleton — or the `delay` and `min` timings the button prop deliberately leaves out.

The composed version needs no `for`: the default rule watches any request from inside the host's parent element, and here that parent *is* the button it sits in. Point `for` at the button (`for="#save"`) only when the spinner lives somewhere else on the page. The same wiring works for `@post('/cmd/…')` — `sb-busy` watches the element, not the method.

Two details make the composed spinner behave inside a button:

- **`display: contents`** so the idle spinner costs nothing. The host is an `inline-block`, and a zero-width child still takes the button's flex `gap`, which would leave a blank notch in an idle button forever. With `display: contents` the host generates no box at all: while idle the button is exactly as wide as a button without it, and when the request starts the spinner and its gap appear and the button grows. Nothing is reserved for a state the button is not in. It arrives at once, undecorated; `sb-busy::part(spinner)` is there if you want to animate it in.
- **`--sb-brand: currentColor`** so the dots take the button's own text colour instead of the brand purple they would be invisible in on a filled button.

### When a plain indicator is enough

`sb-busy` earns its place when you want the spinner, the bar or the skeleton, with `delay` and `min` around them. If all the page needs is a control that shows it is working while a request is in flight, `data-indicator` already does that on its own, and it is lighter:

```html
<sb-button data-indicator:_saving data-attr:loading="$_saving" data-preserve-attr="loading"
  data-on:click="@post('/cmd/save')">Save</sb-button>
```

`data-indicator` sets `$_saving` while that element's request is in flight; the same signal drives `data-attr:disabled` or a CSS class on anything that has no `loading` of its own. Reach for `sb-busy` when the wait needs a shape, a delay, a minimum, or a place of its own.

### No flash, no flicker

`delay` is how long a request may take before anything appears, and `min` is how long the indicator stays once it did appear. A request that finishes inside `delay` never shows an indicator at all; one that finishes right after it keeps it up for `min`. The failure of a watched request skips `min` and drops the indicator at once.

```html preview
<div data-signals="{_fast: [], _slow: []}" style="display: grid; gap: 16px; justify-items: start">
  <div style="display: flex; align-items: center; gap: 16px">
    <sb-button variant="outline" data-on:click="@get('/demo/data/children?into=_fast&delay=100')">100 ms request</sb-button>
    <sb-button variant="outline" data-on:click="@get('/demo/data/children?into=_slow&delay=1200')">1200 ms request</sb-button>
  </div>
  <sb-busy delay="300" min="600" label="Loading" show-label></sb-busy>
</div>
```

### Disabling a form while it waits

`sb-busy-change` fires on every visible flip, so the page can do more than show the indicator.

```html preview
<div data-signals="{_busy: false, _found: []}" style="display: grid; gap: 16px; justify-items: start">
  <div style="display: flex; align-items: center; gap: 16px">
    <sb-button data-attr:disabled="$_busy" data-on:click="@get('/demo/data/search?q=a&into=_found&delay=900')">Search</sb-button>
    <sb-busy variant="bar" label="Searching" data-on:sb-busy-change="$_busy = evt.detail.busy" style="inline-size: 12rem"></sb-busy>
  </div>
  <p style="margin: 0; color: var(--sb-text-2)" data-text="$_busy ? 'searching…' : $_found.length + ' results'"></p>
</div>
```

An `sb-busy` removed while it is up emits a last `{ busy: false }`, so whatever the page switched on with it is switched off again. Its own `data-on` listener still hears it; listeners further up do not, since the element has left the document.

The indicator never shows a *result* before the server produced one: it only says that the page is waiting. The list, the count and the message come from the server's render.

## Styling

Parts: `base` (the indicator, shown while it is up), `spinner`, `bar`, `fill`, `skeleton` and `label`.

| Token | Used for |
| --- | --- |
| `--sb-brand`, `--sb-brand-light` | the dots, the bar fill and the shimmer |
| `--sb-surface-inset`, `--sb-border` | the bar track and the skeleton blocks |
| `--sb-text-2` | the label |
| `--sb-notch` | pixel corners on the bar and the skeleton (`0` falls back to a border radius) |

`inline-size` on the host sizes the bar and the skeleton. The host also carries `:state(busy)` while the indicator is up — a CSS custom state, so a server morph cannot reset it:

```css
form:has(sb-busy:state(busy)) { opacity: 0.6; }
```

The live visible state is on the element as a read-only `visible` property; `busy` stays the server's attribute.

A watched request that was already in flight before the element was upgraded is not counted: the component starts watching when it connects. A morph that moves the element keeps what it counted; an element removed and inserted again starts afresh. Long-lived streams (`@get` on an SSE endpoint) stay in flight until the stream ends, so point `for` at elements that make one-shot requests.

## Accessibility

- A visually hidden `role="status"` region (polite) is always there, and it gets the `label` text when the wait starts, so "Loading systems" is announced then; it is emptied when the wait ends. A region that only appears with its text already in it is not reliably read out. The visible label (`show-label`) is `aria-hidden`, so it is not read twice.
- `sb-busy` sets no `aria-busy`. It belongs on the content that is loading, not on the indicator, and on an ancestor of the status region it would hold the announcement back. Set it on your own region from `sb-busy-change` (`data-attr:aria-busy="$_busy && 'true'"` with the form example above) or from `data-indicator`.
- A determinate bar is a `role="progressbar"` with `aria-valuemin`, `aria-valuemax`, `aria-valuenow` and a percentage `aria-valuetext`, labelled by the same label. An indeterminate bar has no `aria-valuenow`, which is how "unknown progress" is expressed. In a right-to-left page the bar fills and sweeps from the right.
- The spinner dots and the skeleton lines are `aria-hidden` decoration: the text carries the meaning, never the animation.
- With `prefers-reduced-motion: reduce` nothing spins, sweeps or shimmers. Each variant keeps a static state instead: one lit dot, a dimmed full bar, plain blocks.
- In forced colors (Windows High Contrast) the dots and the fill are drawn in `CanvasText`, and the bar and the skeleton blocks get an outline: forced colors drop the backgrounds and shadows they are otherwise drawn with.
