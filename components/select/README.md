---
name: Select
tag: sb-select
category: forms
summary: A select that can filter, pick several, or autocomplete from the server.
author: zweiundeins
tags: [select, dropdown, combobox, autocomplete, search, form]
since: 2026-09-22
preview: |
  <sb-select label="Destination" placeholder="Pick a planet" value="Mars" options='["Mercury","Venus","Earth","Mars","Jupiter","Saturn"]' style="inline-size: 14rem"></sb-select>
usage: |
  <sb-select label="Destination" placeholder="Pick a planet" options='["Mercury","Venus","Earth","Mars"]'></sb-select>
playground:
  attrs:
    options: '["Mercury","Venus","Earth","Mars","Jupiter","Saturn","Uranus","Neptune"]'
  values: {placeholder: "Pick a planet", searchable: true}
  exclude: [options, value, delay, minChars, loading, name, remote]
---

A select for one or several values:

- **Plain:** pick from a list.
- **`searchable`:** type to filter the options in the browser.
- **`remote`:** type to search on the server. It follows the same pattern as the tree: the select asks with an event, and the server answers with `results`.

The live value is the `value` property (a string, or an array with `multiple`), so `data-bind` works.

## Examples

### Autocomplete from the server

Type a star, planet or moon ("or", "sat", "eu"…), from the site's example dataset:

1. After a short pause, the select emits `sb-search` with the query.
2. `@get('/demo/data/search?q=…&into=_found')` asks the server.
3. The server patches `$_found` with the results.
4. `data-attr:results` hands them back.

`data-indicator` shows the spinner while the request is in flight. Without `loading`, the spinner only covers the pause before `sb-search`.

```html preview
<div data-signals="{_found: [], _body: '', _searching: false}" style="display: grid; gap: 12px">
  <sb-select remote clearable label="Find a star, planet or moon" placeholder="Type a name…"
    data-attr:results="JSON.stringify($_found)"
    data-attr:loading="$_searching"
    data-preserve-attr="results loading"
    data-indicator:_searching
    data-on:sb-search="@get('/demo/data/search?kind=star,planet,dwarf,moon&into=_found&delay=150&q=' + encodeURIComponent(evt.detail.query))"
    data-bind:_body__prop.value></sb-select>
  <span>Picked: <b data-text="$_body || 'nothing yet'"></b></span>
</div>
```

The server side is a plain Datastar handler (Go here; any language works):

```go
func search(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	datastar.NewSSE(w, r).MarshalAndPatchSignals(map[string]any{
		"_found": find(q), // [{value, label, description?}], at most a few
	})
}
```

Or the server re-renders the element with a new `results` attribute: a changed attribute always wins.

Until the query has `min-chars` characters, the list shows no results. With `min-chars="0"`, opening the list searches too, with an empty query (e.g. for recent or popular picks).

### Searchable

```html preview
<sb-select searchable label="Planet" placeholder="Type to filter"
  options='[{"value":"mercury","label":"Mercury","description":"0.39 AU"},{"value":"venus","label":"Venus","description":"0.72 AU"},{"value":"earth","label":"Earth","description":"1 AU"},{"value":"mars","label":"Mars","description":"1.52 AU"},{"value":"jupiter","label":"Jupiter","description":"5.2 AU"},{"value":"saturn","label":"Saturn","description":"9.5 AU"},{"value":"pluto","label":"Pluto","description":"No longer a planet","disabled":true}]'></sb-select>
```

### Several values

With `multiple`, the value is an array. Keep it in a signal with `sb-change`: `data-bind` treats array signals as checkbox groups, which is a different thing.

```html preview
<div data-signals="{_crew: ['Ada', 'Yuri']}" style="display: grid; gap: 12px">
  <sb-select multiple searchable clearable label="Crew" placeholder="Add crew"
    options='["Ada","Buzz","Chris","Mae","Sally","Valentina","Yuri"]' value='["Ada","Yuri"]'
    data-on:sb-change="$_crew = evt.detail.value"></sb-select>
  <span>Crew: <b data-text="$_crew.join(', ') || 'nobody'"></b></span>
</div>
```

## With commands

Give it a `name`, and it emits `sb-change` with `{ name, value }` when the value changes: ready to post as a command. With `confirm`, it sets `:state(pending)` until the server's re-rendered `value` matches, and `revert()` goes back to the server's value when a command is rejected. See [Commands and components](/contribute#commands-and-components).

A new `value` from the server always wins, and `value=""` clears it. Markup re-sent with the same `value` leaves the user's pick alone. Typing a search fires no `input` event on the element: `change` and `sb-change` come when the value changes.

## Options

`options` (and `results`, for remote searches) is a JSON array of strings, or of `{value, label?, description?, disabled?}`. The server can change either at any time. A selected value keeps its label even after the options it came from are gone.

## Forms

`sb-select` is not a form-associated element: a `<form>` doesn't submit it, `FormData` and Datastar's `contentType: 'form'` don't see it, and a form reset doesn't reset it. Send its value as a command instead: `sb-change` carries `{ name, value }` (see [With commands](#with-commands)).

## Styling

Style it from your page's CSS — no need to change the component or import anything into it. Custom properties, inherited properties and `::part()` all reach into its shadow root.

- **Size:** it fills the width it is given, up to `26rem`; set `max-inline-size` on the element to change that. The control is at least `2.75rem` tall and grows as chips wrap; the list is at most `18rem` tall (`max-block-size` on `::part(listbox)`).
- **Fonts:** the label, the text you type, the chips and the options use your page's font.
- **Colours:** the control is `--sb-control-bg` with a `--sb-control-border` edge (`--sb-control-border-hover` on hover) and `--sb-control-text`; the placeholder and the arrow are `--sb-control-placeholder`, the label `--sb-text-2`. Focus draws a `--sb-brand-light` edge with a `--sb-brand-subtle` glow, and chips are `--sb-brand-subtle`. The list is `--sb-surface-raised`; the active option is `--sb-surface-hover` with a `--sb-brand` edge, a selected one `--sb-brand-light`, descriptions `--sb-text-muted`. Corners are `--sb-control-radius`.
- **Parts:** `label`, `control` (the box), `input`, `chip` (each chip, with `multiple`), `clear` and `listbox` (the list). Your page's `::part()` rules win over the component's own, without `!important`.

```html preview
<style>
  .my-select { max-inline-size: 16rem; --sb-control-radius: 0; }
  .my-select::part(control) { border-width: 2px; }
  .my-select::part(listbox) { max-block-size: 10rem; }
</style>
<sb-select class="my-select" label="Destination" placeholder="Pick a planet" options='["Mercury","Venus","Earth","Mars"]'></sb-select>
```

## Accessibility

It follows the ARIA combobox pattern:

- **Structure:** the input is a `combobox` controlling a `listbox`, with the highlighted option in `aria-activedescendant`. It is `aria-autocomplete="list"` only when you can type (`searchable` or `remote`).
- **Keys:**
  - Down and Up open the list and move through it; Home and End jump.
  - Enter picks, and Escape closes.
  - Without `searchable` or `remote`, Space opens and picks like Enter, and typing jumps to the next option that starts with the letters (the same letter again cycles).
  - Backspace in an empty input removes the last chip, with `multiple`.
- **Loading:** the input is `aria-busy` while results are on their way, and "Searching…", "No results" and "Type to search" are announced (a status region).
- **Disabled:** `disabled` takes it out of the tab order, and neither keys nor the pointer can change it.

The list is a native popover, so it is never clipped by a scrolling container.
