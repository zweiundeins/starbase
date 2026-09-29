---
name: Data Table
tag: sb-data-table
category: data
summary: A table whose server sends only the rows in view, sorts them, and keeps the selection.
author: zweiundeins
tags: [table, grid, data grid, virtual scroll, sort, selection, server]
since: 2026-09-29
preview: |
  <sb-data-table label="Planets" style="inline-size: 100%" columns='[{"key":"name","label":"Planet"},{"key":"moons","label":"Moons","align":"end","width":"5rem"}]' rows='[{"id":"mercury","name":"Mercury","moons":0},{"id":"earth","name":"Earth","moons":1},{"id":"jupiter","name":"Jupiter","moons":95}]'></sb-data-table>
usage: |
  <sb-data-table label="Planets" columns='[{"key":"name","label":"Planet"},{"key":"moons","label":"Moons","align":"end"}]' rows='[{"id":"earth","name":"Earth","moons":1},{"id":"mars","name":"Mars","moons":2}]'></sb-data-table>
playground:
  attrs:
    columns: '[{"key":"name","label":"Name","sortable":true},{"key":"class","label":"Class","width":"6rem","sortable":true},{"key":"constellation","label":"Constellation","sortable":true},{"key":"magnitude","label":"Magnitude","align":"end","width":"7.5rem","sortable":true}]'
    "data-signals:_pgstars": '{rows: [], offset: 0, total: 0, sort: {key: "", dir: ""}}'
    data-attr: '{rows: JSON.stringify($_pgstars.rows), offset: $_pgstars.offset, total: $_pgstars.total, sort: JSON.stringify($_pgstars.sort)}'
    "data-on:sb-window": '@get(`/demo/data/rows?into=_pgstars&${new URLSearchParams(evt.detail)}`)'
    "data-on:sb-sort": '@get(`/demo/data/rows?into=_pgstars&${new URLSearchParams(evt.detail)}`)'
  style: "block-size: 18rem"
  props: {rowHeight: {min: 24, max: 64}}
  values: {selection: single, label: Star catalog}
  exclude: [offset, total, rowKey, buffer, confirm, name]
---

A table for rows the server keeps: a sticky header, sorting on the server, row selection, and a virtual scroll the server drives. The table asks for the rows in view (plus a buffer), the server sends that window, and a table of a million rows ships a few hundred of them.

The rows scroll in an [sb-virtual-scroll](/components/virtual-scroll), after the virtual scroll in Anders Murphy's [hyperlith](https://github.com/andersmurphy/hyperlith). Every row has the same height (`row-height`), so the scroll position alone tells which rows are in view. When the view comes halfway into the buffer, the table emits `sb-window` with the rows it wants; until they arrive, the rows it lacks show as placeholders.

## Examples

### 100,000 stars from the server

The first 100,000 stars of the site's made-up star catalog, stored in SQLite so the server can sort them (`/demo/data/rows`). Scroll, drag the scrollbar to the middle, click a header to sort, and pick rows. The page never holds more than a few hundred rows:

1. On connect, and whenever the view needs rows it doesn't have, the table emits `sb-window` with `{offset, count}` and the order to send them in (`key`, `dir`).
2. A header click emits `sb-sort` with the order asked for (`key`, `dir`) and the window to answer with (`offset` 0, `count`).
3. Both run `@get('/demo/data/rows?…&into=_stars')`, and the server patches `$_stars` with `{rows, offset, total, sort}`.
4. `data-attr` hands them back. A new `sort` scrolls to the top. Answers can land out of order: after a header click, the table takes rows only in the order it asked for, and drops late answers in the order it replaced.

`data-indicator` sets `loading` while a request is in flight.

```html preview
<div data-signals="{_stars: {rows: [], offset: 0, total: 0, sort: {key: '', dir: ''}}, _loading: false, _picked: []}" style="display: grid; gap: 12px">
  <sb-data-table label="Star catalog" selection="multiple" style="block-size: 22rem"
    columns='[{"key":"name","label":"Name","sortable":true},{"key":"class","label":"Class","width":"6rem","sortable":true},{"key":"constellation","label":"Constellation","sortable":true},{"key":"distance","label":"Distance (ly)","align":"end","sortable":true},{"key":"magnitude","label":"Magnitude","align":"end","width":"7.5rem","sortable":true},{"key":"planets","label":"Planets","align":"end","width":"6rem"}]'
    data-attr="{rows: JSON.stringify($_stars.rows), offset: $_stars.offset, total: $_stars.total, sort: JSON.stringify($_stars.sort), loading: $_loading}"
    data-preserve-attr="rows offset total sort loading"
    data-indicator:_loading
    data-on:sb-window="@get('/demo/data/rows?into=_stars&' + new URLSearchParams(evt.detail))"
    data-on:sb-sort="@get('/demo/data/rows?into=_stars&' + new URLSearchParams(evt.detail))"
    data-on:sb-change="$_picked = evt.detail.value"></sb-data-table>
  <span>Selected: <b data-text="$_picked.join(', ') || 'nothing'"></b></span>
</div>
```

Without Datastar signals, the server can also render the element with the first window in its attributes (`rows`, `offset`, `total`, `sort`) and morph in the next ones. A table that arrives with the rows in view asks for nothing on connect.

### A small table

With all rows given and no `total`, the table has everything and never asks the server. Its columns aren't sortable: a sortable header only emits `sb-sort`, and the order comes from the server.

```html preview
<sb-data-table label="Moons of Jupiter" selection="single" style="inline-size: min(100%, 30rem)"
  columns='[{"key":"name","label":"Moon"},{"key":"radius","label":"Radius (km)","align":"end"},{"key":"found","label":"Found","align":"end","width":"5rem"}]'
  rows='[{"id":"io","name":"Io","radius":1821.6,"found":1610},{"id":"europa","name":"Europa","radius":1560.8,"found":1610},{"id":"ganymede","name":"Ganymede","radius":2634.1,"found":1610},{"id":"callisto","name":"Callisto","radius":2410.3,"found":1610},{"id":"amalthea","name":"Amalthea","radius":83.5,"found":1892}]'></sb-data-table>
```

## Server side

The handler behind the first example is Go with the [Datastar SDK](https://data-star.dev/reference/sdks); any SDK works the same way. It reads the window and the order from the query, and `demoAnswer` patches the signal named by `into` (it also serves plain JSON and the demo's `delay`). The answer carries the order it used, so the header shows what the rows are, not what was clicked:

```go source=internal/web/demo_data.go#Server.demoRows,Server.demoAnswer
// demoRows answers a table's window and sort requests (sb-data-table's
// sb-window and sb-sort) with ?count= stars from ?offset= (at most 500) in the
// order asked for (?key=, ?dir=), and says which order it used.
func (s *Server) demoRows(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	offset, _ := strconv.Atoi(q.Get("offset"))
	count, err := strconv.Atoi(q.Get("count"))
	if err != nil {
		count = 100
	}
	offset, count = max(offset, 0), min(max(count, 1), 500)
	key, dir := q.Get("key"), q.Get("dir")
	if !queries.DemoStarSortable(key) {
		key, dir = "", ""
	} else if dir != "desc" {
		dir = "asc"
	}
	var stars []demo.Star
	var total int
	if err := s.q.View(r.Context(), func(rd *queries.Reader) (err error) {
		stars, total, err = rd.DemoStars(r.Context(), key, dir == "desc", offset, count)
		return
	}); err != nil {
		s.fail(w, r, err)
		return
	}
	window := map[string]any{"rows": stars, "offset": offset, "total": total, "sort": map[string]string{"key": key, "dir": dir}}
	s.demoAnswer(w, r, "_rows", window, window)
}

// demoAnswer writes the list as JSON, or patches it into the requested signal.
func (s *Server) demoAnswer(w http.ResponseWriter, r *http.Request, defaultSignal string, list, patch any) {
	w.Header().Set("Access-Control-Allow-Origin", "*") // public; used from the playground sandbox
	if ms, _ := strconv.Atoi(r.URL.Query().Get("delay")); ms > 0 {
		select {
		case <-time.After(time.Duration(min(ms, 1500)) * time.Millisecond):
		case <-r.Context().Done():
			return
		}
	}
	// Datastar's own requests accept JSON too: they always get the patch.
	if r.Header.Get("Datastar-Request") == "" && strings.Contains(r.Header.Get("Accept"), "application/json") {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "public, max-age=300")
		json.NewEncoder(w).Encode(list)
		return
	}
	into := r.URL.Query().Get("into")
	if into == "" {
		into = defaultSignal
	}
	if !signalNameRe.MatchString(into) {
		http.Error(w, "into must be a signal name", http.StatusBadRequest)
		return
	}
	datastar.NewSSE(w, r).MarshalAndPatchSignals(map[string]any{into: patch})
}
```

The query asks SQLite for one window in one order. Every sortable column has an index, and the id breaks ties, so the windows of one order fit together. `OFFSET` walks the index: the last window of 100,000 rows takes about a millisecond.

```go source=internal/queries/demo_stars.go#starOrder,Reader.DemoStars
// starOrder maps a sort key of the star catalog to its column: the class
// sorts by temperature.
var starOrder = map[string]string{
	"name": "name", "class": "temp", "constellation": "constellation",
	"distance": "distance", "magnitude": "magnitude",
}

// DemoStars returns up to count stars from offset, sorted by sort (a key of
// starOrder, else the catalog's own order), and how many stars there are.
func (r *Reader) DemoStars(ctx context.Context, sort string, desc bool, offset, count int) ([]demo.Star, int, error) {
	var total int
	if err := r.tx.QueryRowContext(ctx, `SELECT count(*) FROM demo_stars`).Scan(&total); err != nil {
		return nil, 0, err
	}
	col, dir := starOrder[sort], " ASC"
	if col == "" {
		col = "id"
	}
	if desc {
		dir = " DESC"
	}
	// Ties in id order (the indexes hold it), so every window of one order fits the next.
	rows, err := r.tx.QueryContext(ctx, `SELECT id, name, class, temp, constellation, distance, magnitude, planets FROM demo_stars
		ORDER BY `+col+dir+`, id`+dir+` LIMIT ? OFFSET ?`, count, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := []demo.Star{}
	for rows.Next() {
		var s demo.Star
		if err := rows.Scan(&s.ID, &s.Name, &s.Class, &s.Temp, &s.Constellation, &s.Distance, &s.Magnitude, &s.Planets); err != nil {
			return nil, 0, err
		}
		out = append(out, s)
	}
	return out, total, rows.Err()
}
```

## With commands

Give it a `name` and a `selection`, and it emits `sb-change` with `{ name, value }` (an array of row keys) when the selection changes: ready to post as a command. With `confirm`, it sets `:state(pending)` until the server's re-rendered `selected` matches, and `revert()` goes back to the server's selection when a command is rejected. See [Commands and components](/contribute#commands-and-components).

A new `selected` from the server always wins, and `selected="[]"` clears it. Markup re-sent with the same `selected` leaves the user's selection alone. The order (`sort`) is not part of the value: it only ever comes from the server.

## Columns and rows

- `columns`: `[{key, label?, width?, align?, sortable?}]`. `width` is a CSS grid track (`"8rem"`, `"minmax(4rem, 2fr)"`, a number for px; default `minmax(6rem, 1fr)`). Widths whose minimum is a length keep the columns steady while rows come and go; a table wider than its box scrolls sideways. `align` is `start`, `center` or `end`.
- `rows`: objects keyed by column. Numbers show in the reader's format (`4,242.5`), everything else as text. The field named by `row-key` (default `id`) identifies a row.
- `offset` is the index of the first row in `rows`, and `total` the number of rows there are. Without `total`, the table has what it was given.
- `row-height` is fixed (default `36` px), and `buffer` (default `4000` px of rows) is how much the table keeps ready above and below the view.
- A server that sends fewer rows than asked for (a cap) still fills the view: the table then asks for windows that fit the cap.
- Browsers cap how tall an element can be, Firefox at about 17.9 million px (Chrome and Safari at about 33.5 million). Keep rows × `row-height` under that: 100,000 rows of 36 px are 3.6 million px.

## Forms

`sb-data-table` is not a form-associated element: a `<form>` doesn't submit its selection, and `FormData` and Datastar's `contentType: 'form'` don't see it. Send the selection as a command instead: `sb-change` carries `{ name, value }` (see [With commands](#with-commands)).

## Styling

Style it from your page's CSS, without changing the component or importing anything into it. Custom properties, inherited properties and `::part()` all reach into its shadow root.

- **Size:** it fills the width it is given and is at most `24rem` tall; set `block-size` or `max-block-size` on the element to change that. Rows are `row-height` tall. Set `font-size` on the element (default `0.875rem`) to scale the text.
- **Fonts:** the header and the cells use your page's font.
- **Colours:** the table is `--sb-surface-card` with a `--sb-border` frame, rows are separated by `--sb-border-subtle` lines, and placeholders are sb-virtual-scroll's `--sb-surface-hover` bars. The header is `--sb-surface-raised` with `--sb-text-2` labels; cells are `--sb-text-1`. A row turns `--sb-surface-hover` on hover; a selected row is `--sb-brand-subtle` with a `--sb-brand` edge. The focus ring and the sort arrow are `--sb-brand-light`. Corners are `--sb-radius`, or pixel notches while `--sb-notch` is 1.
- **Parts:** `grid` (the sb-virtual-scroll that scrolls), `header` (the header row), `column` (a header cell), `row` and `cell`. Selected rows are also `selected`, so `::part(row selected)` styles only those. Your page's `::part()` rules win over the component's own, without `!important`.

```html preview
<style>
  .my-table { --sb-brand: var(--sb-accent); }
  .my-table::part(header) { text-transform: uppercase; font-size: 0.75rem; }
  .my-table::part(row selected) { font-weight: 700; }
</style>
<sb-data-table class="my-table" label="Planets" selection="single" selected='["earth"]' row-height="40"
  columns='[{"key":"name","label":"Planet"},{"key":"moons","label":"Moons","align":"end"}]'
  rows='[{"id":"venus","name":"Venus","moons":0},{"id":"earth","name":"Earth","moons":1},{"id":"mars","name":"Mars","moons":2}]'></sb-data-table>
```

## Accessibility

It follows the WAI-ARIA grid pattern:

- **Structure:** a `grid` with `aria-rowcount` for every row there is (plus the header), and `aria-rowindex` on each rendered row, so a screen reader knows where it is in the whole table, not in the few rows the page holds. Sorted headers have `aria-sort`; with a `selection`, rows have `aria-selected`. The grid is `aria-busy` while `loading` is set.
- **Focus:** one cell at a time is in the tab order: the first header, then the cell focused last, or the header again once that cell's row is gone. Sortable headers are buttons.
- **Keys:** the arrows move between cells, Home and End to the first and last cell of the row, Ctrl+Home to the first header, Ctrl+End to the last cell of the last row, Page Up and Page Down by a view. Moving past the rendered rows scrolls there, and the focus lands when the rows arrive. Space selects or unselects the row, Enter emits `sb-row-activate` (so does a double click, which selects only once). On a sortable header, Enter or Space sorts.
