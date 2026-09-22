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
playground:
  attrs:
    options: '["Mercury","Venus","Earth","Mars","Jupiter","Saturn","Uranus","Neptune"]'
  values: {placeholder: "Pick a planet", searchable: true}
  exclude: [options, value, delay, minChars, loading, name, remote]
---

A select for one or several values:

- **Plain:** pick from a list.
- **`searchable`:** type to filter the options in the browser.
- **`remote`:** type to search on the server. It follows the same pattern as the tree: the select asks with an event, and the server answers by patching a signal.

The live value is the `value` property (a string, or an array with `multiple`), so `data-bind` works.

## Examples

### Autocomplete from the server

Type a star or a constellation ("or", "cru", "lyra"…):

1. After a short pause, the select emits `sb-search` with the query.
2. `@get('/demo/search?q=…')` asks the server.
3. The server patches `$_found` with the results.
4. `data-attr:options` hands them back.

`data-indicator` shows the spinner while the request is in flight.

```html preview
<div data-signals="{_found: [], _star: '', _searching: false}" style="display: grid; gap: 12px">
  <sb-select remote clearable label="Find a star" placeholder="Type a star or constellation…"
    data-attr:options="JSON.stringify($_found)"
    data-attr:loading="$_searching"
    data-preserve-attr="options loading"
    data-indicator:_searching
    data-on:sb-search="@get('/demo/search?q=' + encodeURIComponent(evt.detail.query))"
    data-bind:_star__prop.value></sb-select>
  <span>Picked: <b data-text="$_star || 'nothing yet'"></b></span>
</div>
```

The server side is a plain Datastar handler (Go here; any language works):

```go
func search(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	datastar.NewSSE(w, r).MarshalAndPatchSignals(map[string]any{
		"_found": findStars(q), // [{value, label, description?}], at most a few
	})
}
```

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

## Options

`options` is a JSON array of strings, or of `{value, label?, description?, disabled?}`. Bind it to a signal to change it, e.g. from the server. A selected value keeps its label even after the options it came from are gone.

## Accessibility

It follows the ARIA combobox pattern:

- **Structure:** the input is a `combobox` controlling a `listbox`, with the highlighted option in `aria-activedescendant`.
- **Keys:**
  - Down and Up open the list and move through it; Home and End jump.
  - Enter picks, and Escape closes.
  - Backspace in an empty input removes the last chip, with `multiple`.
- **Loading:** the input is `aria-busy` while results are on their way.

The list is a native popover, so it is never clipped by a scrolling container.
