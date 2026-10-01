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
  exclude: [offset, total, rowKey, buffer, confirm, name, hiddenColumns]
---

A table for rows the server keeps: a sticky header, sorting (on the server, or in the table when it has every row), row selection, and a virtual scroll the server drives. Cells can be links, carry a muted suffix or show as badges; a column picker lets the user hide columns, and exports come from the server, which has every row. The table asks for the rows in view (plus a buffer), the server sends that window, and a table of a million rows ships a few hundred of them.

The rows scroll in an [sb-virtual-scroll](/components/virtual-scroll), after the virtual scroll in Anders Murphy's [hyperlith](https://github.com/andersmurphy/hyperlith). Every row has the same height (`row-height`), so the scroll position alone tells which rows are in view. When the view comes halfway into the buffer, the table emits `sb-window` with the rows it wants; until they arrive, the rows it lacks show as placeholders.

## Examples

### 100,000 stars from the server

The first 100,000 stars of the site's made-up star catalog, stored in SQLite so the server can sort them (`/demo/data/rows`). Scroll, drag the scrollbar to the middle, click a header to sort, pick rows, hide columns, click a star's name, and download the table as CSV or JSON. The page never holds more than a few hundred rows:

1. On connect, and whenever the view needs rows it doesn't have, the table emits `sb-window` with `{offset, count}` and the order to send them in (`key`, `dir`).
2. A header click emits `sb-sort` with the order asked for (`key`, `dir`) and the window to answer with (`offset` 0, `count`).
3. Both run `@get('/demo/data/rows?…&into=_stars&cells=rich')`, and the server patches `$_stars` with `{rows, offset, total, sort}`. With `cells=rich`, its rows hold cell objects: the names are links, the distances carry their unit, and stars bright enough to see have a badge on their magnitude.
4. `data-attr` hands them back. A new `sort` scrolls to the top. Answers can land out of order: after a header click, the table takes rows only in the order it asked for, and drops late answers in the order it replaced.
5. A click on a name emits `sb-cell-activate`. The page cancels the link and shows the star instead: this is how a cell runs an action on the page.
6. The CSV and JSON buttons are the page's own, in the `toolbar` slot. They call `requestExport()`, the table emits `sb-export` with its order, the shown columns and the selection, and the page turns that into a download of `/demo/data/rows/export` (see [Exports](#exports)). A download link starts no navigation, so the page's streams stay open.

`data-indicator` sets `loading` while a request is in flight.

```html preview
<div data-signals="{_stars: {rows: [], offset: 0, total: 0, sort: {key: '', dir: ''}}, _loading: false, _picked: [], _star: ''}" style="display: grid; gap: 12px">
  <sb-data-table label="Star catalog" selection="multiple" column-picker style="block-size: 24rem"
    columns='[{"key":"name","label":"Name","sortable":true},{"key":"class","label":"Class","width":"6rem","sortable":true},{"key":"constellation","label":"Constellation","sortable":true},{"key":"distance","label":"Distance","align":"end","sortable":true},{"key":"magnitude","label":"Magnitude","align":"end","width":"7.5rem","sortable":true},{"key":"planets","label":"Planets","align":"end","width":"6rem"}]'
    data-attr="{rows: JSON.stringify($_stars.rows), offset: $_stars.offset, total: $_stars.total, sort: JSON.stringify($_stars.sort), loading: $_loading}"
    data-preserve-attr="rows offset total sort loading"
    data-indicator:_loading
    data-on:sb-window="@get('/demo/data/rows?into=_stars&cells=rich&' + new URLSearchParams(evt.detail))"
    data-on:sb-sort="@get('/demo/data/rows?into=_stars&cells=rich&' + new URLSearchParams(evt.detail))"
    data-on:sb-change="$_picked = evt.detail.value"
    data-on:sb-cell-activate="evt.preventDefault(); $_star = evt.detail.value"
    data-on:sb-export="Object.assign(document.createElement('a'), {href: '/demo/data/rows/export?' + new URLSearchParams({format: evt.detail.format, key: evt.detail.sort.key, dir: evt.detail.sort.dir, columns: evt.detail.columns, selected: evt.detail.selected}), download: ''}).click()">
    <button slot="toolbar" data-on:click="el.closest('sb-data-table').requestExport('csv')">CSV</button>
    <button slot="toolbar" data-on:click="el.closest('sb-data-table').requestExport('json')">JSON</button>
  </sb-data-table>
  <span>Selected: <b data-text="$_picked.join(', ') || 'nothing'"></b></span>
  <span>Star: <b data-text="$_star || 'click a name'"></b></span>
</div>
```

Without Datastar signals, the server can also render the element with the first window in its attributes (`rows`, `offset`, `total`, `sort`) and morph in the next ones. A table that arrives with the rows in view asks for nothing on connect.

### A small table

With all rows given and no `total`, the table has everything and never asks the server. A click on a sortable header sorts the rows right here, and the table still emits `sb-sort` for a page that wants to keep the choice.

```html preview
<sb-data-table label="Moons of Jupiter" selection="single" style="inline-size: min(100%, 30rem)"
  columns='[{"key":"name","label":"Moon","sortable":true},{"key":"radius","label":"Radius (km)","align":"end","sortable":true},{"key":"found","label":"Found","align":"end","width":"6rem","sortable":true}]'
  rows='[{"id":"io","name":"Io","radius":1821.6,"found":1610},{"id":"europa","name":"Europa","radius":1560.8,"found":1610},{"id":"ganymede","name":"Ganymede","radius":2634.1,"found":1610},{"id":"callisto","name":"Callisto","radius":2410.3,"found":1610},{"id":"amalthea","name":"Amalthea","radius":83.5,"found":1892}]'></sb-data-table>
```

### Rich cells

A value can be a cell object instead of a plain value: `{"value": 2634.1, "suffix": "km"}` shows the number with a muted unit after it and still sorts by the number, `href` makes the text a link, and `tone` shows it as a badge. The names link to Wikipedia, and the last row shows that a cell's text is only ever text, never markup:

```html preview
<sb-data-table label="Moons of Jupiter" selection="single" style="inline-size: min(100%, 34rem)"
  columns='[{"key":"name","label":"Moon","sortable":true},{"key":"radius","label":"Radius","align":"end","sortable":true},{"key":"group","label":"Group","width":"7rem","sortable":true}]'
  rows='[{"id":"io","name":{"value":"Io","href":"https://en.wikipedia.org/wiki/Io_(moon)"},"radius":{"value":1821.6,"suffix":"km"},"group":{"value":"Galilean","tone":"info"}},{"id":"europa","name":{"value":"Europa","href":"https://en.wikipedia.org/wiki/Europa_(moon)"},"radius":{"value":1560.8,"suffix":"km"},"group":{"value":"Galilean","tone":"info"}},{"id":"ganymede","name":{"value":"Ganymede","href":"https://en.wikipedia.org/wiki/Ganymede_(moon)"},"radius":{"value":2634.1,"suffix":"km"},"group":{"value":"Galilean","tone":"info"}},{"id":"callisto","name":{"value":"Callisto","href":"https://en.wikipedia.org/wiki/Callisto_(moon)"},"radius":{"value":2410.3,"suffix":"km"},"group":{"value":"Galilean","tone":"info"}},{"id":"amalthea","name":{"value":"Amalthea","href":"https://en.wikipedia.org/wiki/Amalthea_(moon)"},"radius":{"value":83.5,"suffix":"km"},"group":{"value":"Inner","tone":"neutral"}},{"id":"test","name":"<b>not bold</b>","radius":{"value":0.5,"suffix":"km"},"group":{"value":"Made up","tone":"warning"}}]'></sb-data-table>
```

### Columns the user picks

With `column-picker`, a Columns button above the table opens a menu of every column, each with a checkbox. `hidden-columns` holds the keys of the columns not shown. It is view state, like a tree's expanded branches: the table works without anyone keeping it, and when the user changes it, the table emits `sb-columns` with the whole list. Here a page signal keeps it:

```html preview
<div data-signals="{_hidden: ['mass']}" style="display: grid; gap: 12px">
  <sb-data-table label="Planets" column-picker style="inline-size: min(100%, 40rem)"
    columns='[{"key":"name","label":"Planet"},{"key":"moons","label":"Moons","align":"end","width":"5rem"},{"key":"radius","label":"Radius","align":"end"},{"key":"mass","label":"Mass (Earths)","align":"end"},{"key":"day","label":"Day","align":"end"}]'
    rows='[{"id":"mercury","name":"Mercury","moons":0,"radius":{"value":2439.7,"suffix":"km"},"mass":0.055,"day":{"value":4222.6,"suffix":"h"}},{"id":"venus","name":"Venus","moons":0,"radius":{"value":6051.8,"suffix":"km"},"mass":0.815,"day":{"value":2802,"suffix":"h"}},{"id":"earth","name":"Earth","moons":1,"radius":{"value":6371,"suffix":"km"},"mass":1,"day":{"value":24,"suffix":"h"}},{"id":"mars","name":"Mars","moons":2,"radius":{"value":3389.5,"suffix":"km"},"mass":0.107,"day":{"value":24.7,"suffix":"h"}},{"id":"jupiter","name":"Jupiter","moons":95,"radius":{"value":69911,"suffix":"km"},"mass":317.8,"day":{"value":9.9,"suffix":"h"}}]'
    data-attr:hidden-columns="JSON.stringify($_hidden)" data-preserve-attr="hidden-columns"
    data-on:sb-columns="$_hidden = evt.detail.hidden"></sb-data-table>
  <span>Hidden: <b data-text="$_hidden.join(', ') || 'nothing'"></b></span>
</div>
```

A server that should remember the choice posts `sb-columns` as a command and keeps the list as a session preference, so the next page renders it into `hidden-columns`. A new list from the server wins over the user's, the same list again leaves it alone, and `el.hiddenColumns` reads or sets the list on the page without an event. At least one column always shows: the menu won't hide the last one, and a list that hides every column still shows the first.

## Server side

The handler behind the first example is Go with the [Datastar SDK](https://data-star.dev/reference/sdks); any SDK works the same way. It reads the window and the order from the query, and `demoAnswer` patches the signal named by `into` (it also serves plain JSON and the demo's `delay`). The answer carries the order it used, so the header shows what the rows are, not what was clicked. With `cells=rich`, each star goes through `demoStarCells` (below, with the export), which turns it into cell objects:

```go source=internal/web/demo_data.go#Server.demoRows,Server.demoAnswer
// demoRows answers a table's window and sort requests (sb-data-table's
// sb-window and sb-sort) with ?count= stars from ?offset= (at most 500) in the
// order asked for (?key=, ?dir=), and says which order it used. With
// &cells=rich, the rows hold cell objects (demoStarCells).
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
	if q.Get("cells") == "rich" {
		rows := make([]map[string]any, len(stars))
		for i, st := range stars {
			rows[i] = demoStarCells(st)
		}
		window["rows"] = rows
	}
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

```go source=internal/queries/demo_stars.go#starOrder,starOrderBy,Reader.DemoStars
// starOrder maps a sort key of the star catalog to its column: the class
// sorts by temperature.
var starOrder = map[string]string{
	"name": "name", "class": "temp", "constellation": "constellation",
	"distance": "distance", "magnitude": "magnitude",
}

// starOrderBy is the ORDER BY for sort (a key of starOrder, else the
// catalog's own order). Ties go in id order (the indexes hold it), so every
// window of one order fits the next.
func starOrderBy(sort string, desc bool) string {
	col, dir := starOrder[sort], " ASC"
	if col == "" {
		col = "id"
	}
	if desc {
		dir = " DESC"
	}
	return col + dir + ", id" + dir
}

// DemoStars returns up to count stars from offset, sorted by sort (a key of
// starOrder, else the catalog's own order), and how many stars there are.
func (r *Reader) DemoStars(ctx context.Context, sort string, desc bool, offset, count int) ([]demo.Star, int, error) {
	var total int
	if err := r.tx.QueryRowContext(ctx, `SELECT count(*) FROM demo_stars`).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := r.tx.QueryContext(ctx, `SELECT id, name, class, temp, constellation, distance, magnitude, planets FROM demo_stars
		ORDER BY `+starOrderBy(sort, desc)+` LIMIT ? OFFSET ?`, count, offset)
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

### Exports

A page puts its own export buttons in the `toolbar` slot, and they call the table's `requestExport(format)` (`csv` by default), which emits `sb-export` with `{name, format, sort, columns, selected}`: the order on screen, the keys of the shown columns in their order, and the user's selection. The table never builds a file and never asks for the rows: the server has them. The page turns the event into a download of `/demo/data/rows/export`, which checks every parameter, is rate limited (per session, three downloads in a row, then one every five seconds; the whole server streams at most two at a time), and streams the file. The whole catalog is 100,000 rows; it goes out compressed, and the browser never holds more than the window it shows. `demoStarCells` is the rich rows' shape from the first example:

```go source=internal/web/demo_data.go#Server.demoRowsExport,demoStarCells
// demoRowsExport answers sb-data-table's sb-export with a download of the star
// catalog: ?format=csv or json, in the order asked for (?key=, ?dir=), the
// ?columns= in their order (default: all), and with ?selected= only those
// stars (at most 1000 ids). The rows stream from one read transaction, so the
// whole catalog never sits in memory, and go out compressed.
func (s *Server) demoRowsExport(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	format := q.Get("format")
	if format != "csv" && format != "json" {
		http.Error(w, "format must be csv or json", http.StatusBadRequest)
		return
	}
	key, dir := q.Get("key"), q.Get("dir")
	if !queries.DemoStarSortable(key) {
		key, dir = "", ""
	} else if dir != "desc" {
		dir = "asc"
	}
	columns := demoStarColumns
	if c := q.Get("columns"); c != "" {
		columns = strings.Split(c, ",")
		for i, k := range columns {
			if !slices.Contains(demoStarColumns, k) {
				http.Error(w, "unknown column "+strconv.Quote(k), http.StatusBadRequest)
				return
			}
			if slices.Contains(columns[:i], k) {
				http.Error(w, "column "+strconv.Quote(k)+" twice", http.StatusBadRequest)
				return
			}
		}
	}
	var ids []int
	if sel := q.Get("selected"); sel != "" {
		list := strings.Split(sel, ",")
		if len(list) > 1000 {
			http.Error(w, "at most 1000 selected ids", http.StatusBadRequest)
			return
		}
		for _, v := range list {
			id, err := strconv.Atoi(v)
			if err != nil {
				http.Error(w, "selected must be ids", http.StatusBadRequest)
				return
			}
			ids = append(ids, id)
		}
	}
	now := time.Now()
	if !s.exportLimit.allow(sessionID(r), 1, now) || !s.exportLimitIP.allow(clientIP(r), 1, now) {
		http.Error(w, "exporting too often", http.StatusTooManyRequests)
		return
	}
	// A download holds a read connection until it ends: slow ones must not
	// take them all from the pages, and one that stalls for a minute ends.
	select {
	case s.exports <- struct{}{}:
		defer func() { <-s.exports }()
	default:
		w.Header().Set("Retry-After", "5")
		http.Error(w, "too many downloads at once", http.StatusServiceUnavailable)
		return
	}
	setWriteDeadline(w, r, time.Now().Add(time.Minute))

	h := w.Header()
	h.Set("Access-Control-Allow-Origin", "*") // public; used from the playground sandbox
	h.Set("Cache-Control", "public, max-age=300")
	h.Set("Content-Disposition", `attachment; filename="stars.`+format+`"`)
	// Each star is written as it comes from the cursor.
	var write func(demo.Star) error
	var end func() error
	if format == "csv" {
		h.Set("Content-Type", "text/csv; charset=utf-8")
		cw := csv.NewWriter(w)
		cw.Write(columns)
		row := make([]string, len(columns))
		write = func(st demo.Star) error {
			for i, k := range columns {
				switch v := demoStarField(st, k).(type) {
				case float64:
					row[i] = strconv.FormatFloat(v, 'f', -1, 64) // 1234567, never 1.234567e+06
				default:
					row[i] = fmt.Sprint(v)
				}
			}
			return cw.Write(row)
		}
		end = func() error { cw.Flush(); return cw.Error() }
	} else {
		h.Set("Content-Type", "application/json")
		bw := bufio.NewWriter(w)
		bw.WriteString("[")
		sep := ""
		write = func(st demo.Star) error {
			obj := []byte(sep + "{")
			for i, k := range columns {
				if i > 0 {
					obj = append(obj, ',')
				}
				v, err := json.Marshal(demoStarField(st, k))
				if err != nil {
					return err
				}
				obj = append(append(strconv.AppendQuote(obj, k), ':'), v...)
			}
			sep = ",\n"
			_, err := bw.Write(append(obj, '}'))
			return err
		}
		end = func() error { bw.WriteString("]\n"); return bw.Flush() }
	}
	n := 0
	err := s.q.View(r.Context(), func(rd *queries.Reader) error {
		return rd.EachDemoStar(r.Context(), key, dir == "desc", ids, func(st demo.Star) error {
			if n++; n%1000 == 0 {
				setWriteDeadline(w, r, time.Now().Add(time.Minute))
			}
			return write(st)
		})
	})
	if err == nil {
		err = end()
	}
	if err != nil {
		// A file cut short must not look complete: the download fails instead.
		if r.Context().Err() == nil {
			s.log.Error("export failed", "err", err)
		}
		panic(http.ErrAbortHandler)
	}
}

// demoStarCells is a star as sb-data-table's cell objects: the name links to
// the star (an in-page link the demo cancels, to run an action instead), the
// distance carries its unit, and the magnitude of a star the naked eye can see
// (6 or brighter) shows as a badge.
func demoStarCells(st demo.Star) map[string]any {
	var magnitude any = st.Magnitude
	if st.Magnitude <= 6 {
		magnitude = map[string]any{"value": st.Magnitude, "tone": "info"}
	}
	return map[string]any{
		"id":            st.ID,
		"name":          map[string]any{"value": st.Name, "href": "#star-" + strconv.Itoa(st.ID)},
		"class":         st.Class,
		"temp":          st.Temp,
		"constellation": st.Constellation,
		"distance":      map[string]any{"value": st.Distance, "suffix": "ly"},
		"magnitude":     magnitude,
		"planets":       st.Planets,
	}
}
```

It reads the stars from a cursor in the same order as the windows, so the file has the order the table shows:

```go source=internal/queries/demo_stars.go#Reader.EachDemoStar
// EachDemoStar calls fn with every star in the order DemoStars gives them, or
// only with the stars whose id is in ids, and stops at fn's first error. The
// rows stream from the cursor: a whole export never sits in memory.
func (r *Reader) EachDemoStar(ctx context.Context, sort string, desc bool, ids []int, fn func(demo.Star) error) error {
	where, args := "", make([]any, len(ids))
	if len(ids) > 0 {
		where = ` WHERE id IN (?` + strings.Repeat(", ?", len(ids)-1) + `)`
		for i, id := range ids {
			args[i] = id
		}
	}
	rows, err := r.tx.QueryContext(ctx, `SELECT id, name, class, temp, constellation, distance, magnitude, planets FROM demo_stars`+where+`
		ORDER BY `+starOrderBy(sort, desc), args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var s demo.Star
		if err := rows.Scan(&s.ID, &s.Name, &s.Class, &s.Temp, &s.Constellation, &s.Distance, &s.Magnitude, &s.Planets); err != nil {
			return err
		}
		if err := fn(s); err != nil {
			return err
		}
	}
	return rows.Err()
}
```

A real server should also neutralise text cells that start with `=`, `+`, `-` or `@` (for example with a leading `'`), or a spreadsheet that opens the CSV may run them as formulas. The star catalog has no text like that.

## With commands

Give it a `name` and a `selection`, and it emits `sb-change` with `{ name, value }` (an array of row keys) when the selection changes: ready to post as a command. With `confirm`, it sets `:state(pending)` until the server's re-rendered `selected` matches, and `revert()` goes back to the server's selection when a command is rejected. See [Commands and components](/contribute#commands-and-components).

A new `selected` from the server always wins, and `selected="[]"` clears it. Markup re-sent with the same `selected` leaves the user's selection alone. The order (`sort`) is not part of the value: it only ever comes from the server. Neither is `hidden-columns`: it is view state with its own event (`sb-columns`), never pending, and `revert()` leaves it as it is.

## Columns and rows

- `columns`: `[{key, label?, width?, align?, sortable?}]`. `width` is a CSS grid track (`"8rem"`, `"minmax(4rem, 2fr)"`, a number for px; default `minmax(6rem, 1fr)`). Widths whose minimum is a length keep the columns steady while rows come and go; a table wider than its box scrolls sideways. `align` is `start`, `center` or `end`.
- `rows`: objects keyed by column. Numbers show in the reader's format (`4,242.5`), everything else as text. The field named by `row-key` (default `id`) identifies a row.
- A value can also be a cell object, `{value, text?, suffix?, href?, tone?}`:
  - `value` is the raw value: the sort key, what `sb-cell-activate` reports, and the text when there is no `text` (formatted like a plain value).
  - `text` is shown instead of the formatted `value`, and `suffix` after it in a muted colour (the table adds the space).
  - `href` makes the text a link, but only when it resolves (against the page's URL) to `http:`, `https:` or `mailto:`. Anything else, such as `javascript:` or `data:`, shows plain text.
  - `tone` shows the text as a badge: `info`, `success`, `warning`, `danger` or `neutral`. Any other tone is ignored.
  - Nothing in a cell is ever read as HTML: `<b>` stays the text `<b>`. Other fields are ignored, and a row's key may be a cell object too (its `value` is the key).
- A click or middle click on a link, or Enter on its cell, emits `sb-cell-activate` with `{key, column, value, href}`. It is cancelable: a listener that calls `preventDefault()` stops the navigation (a middle click then opens no tab) and can run an action instead, such as `data-on:sb-cell-activate="evt.detail.column === 'ip' && (evt.preventDefault(), @post('/ip-info?ip=' + encodeURIComponent(evt.detail.value)))"`. A click on a link doesn't select the row.
- `offset` is the index of the first row in `rows`, and `total` the number of rows there are. Without `total`, the table has what it was given.
- Sorting: a table that holds every row (`offset` 0, and no `total` or one no larger than the rows given) sorts them itself when a sortable header is clicked. Numbers sort by value, text in the reader's order (with numbers inside it by value, so "Io 2" comes before "Io 10"), and missing values last. A cell object sorts by its `value`, else its `text`. It still emits `sb-sort`, a new `sort` from the server wins, and rows the server sends in are sorted the same way, so their order doesn't jump. A table that holds only a window asks, and the server sends the rows in the new order.
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
- **Shadow:** `--sb-shadow-overlay` sets the column picker's drop shadow: one shadow without spread, such as `0 8px 16px rgb(0 0 0 / 0.3)`, or `none`.
- **Toolbar:** it shows above the table with `column-picker`, or while a control you put in its slot shows (one with `hidden`, or hidden by `data-show`, doesn't count). The Columns button and its menu are `--sb-surface-raised` with a `--sb-border` frame and `--sb-text-2` text; checkboxes take `--sb-brand`. A `<button>` you put in the slot looks like the Columns button until your page styles it.
- **Rich cells:** links are `--sb-brand-light` and underlined, suffixes `--sb-text-muted`. Badges take their tone from `--sb-info`, `--sb-ok`, `--sb-warn` and `--sb-danger` (`neutral` from `--sb-text-muted`), with a tinted background and border.
- **Parts:** `grid` (the sb-virtual-scroll that scrolls), `header` (the header row), `column` (a header cell), `row` and `cell`. Selected rows are also `selected`, so `::part(row selected)` styles only those. In rows with cell objects, a cell holds a `text` (also `link` for a link, `badge` for a badge, so `::part(text badge)` styles only badges) and a `suffix`. Above the table: `toolbar`, `columns-button`, `columns-menu` (the popover) and `column-option` (a label with its checkbox). Your page's `::part()` rules win over the component's own, without `!important`.

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
- **Links in cells:** a link is not a tab stop of its own; the cell is, and Enter on it follows the link (after `sb-cell-activate`). Space still selects the row.
- **Column menu:** the Columns button has `aria-haspopup="dialog"` and `aria-expanded`, and opens a dialog named by `columns-label` with a native checkbox per column. The focus goes to the first checkbox; Tab and Shift+Tab cycle through them, Space toggles one (the focus stays on it), and Escape closes only the menu, also inside a drawer, and returns the focus to the button.
- **Badges:** a tone is colour only, so the badge's text has to carry the meaning ("Failed", not a red dot). In forced colours, links use the system's link colour and badges keep a border.
