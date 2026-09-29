---
name: Date Picker
tag: sb-date-picker
category: forms
summary: A date field with a calendar, for one date or a range, in the reader's language.
author: zweiundeins
tags: [date, calendar, datepicker, range, booking, intl, i18n, form]
since: 2026-09-29
preview: |
  <sb-date-picker inline mode="range" value='{"start":"2026-10-06","end":"2026-10-09"}' month="2026-10" disabled-dates='["2026-10-14","2026-10-15"]' lang="en-GB" style="zoom: 0.7"></sb-date-picker>
usage: |
  <sb-date-picker label="Launch date" value="2026-10-14"></sb-date-picker>
playground:
  values: {label: Launch date}
  attrs: {value: "2026-10-14"}
  exclude: [value, disabledDates, month, open, name, lang, error, confirm]
---

A text field for a date, with a button that opens a calendar. The field shows the date the way the page's language writes it and takes typed dates too. With `inline`, the calendar sits on the page on its own, always visible.

- **`mode="range"`:** pick a start and an end. The range is **one value** (`{start, end}`), committed once, like [`sb-range`](/components/range).
- **`min`, `max`, `disabled-dates`:** days that can't be picked, e.g. booked ones. The server sends them, also one month at a time.
- **`month` and `open`:** view state the server may set. The calendar reports the user paging months with `sb-month`.

The value is an ISO date (`2026-09-29`) in and out, whatever the language. The live value is the `value` property, so `data-bind` works.

## Examples

### A date

Type a date, or open the calendar with the button:

```html preview
<div data-signals="{_launch: '2026-10-14'}" style="display: grid; gap: 12px">
  <sb-date-picker label="Launch date" value="2026-10-14" data-bind:_launch__prop.value></sb-date-picker>
  <span>Value: <code data-text="$_launch || 'none'"></code></span>
</div>
```

### A range

The first pick marks the start, the second the end, and only then does the value change, with one `sb-change`. Picking the end first works too: the two are put in order.

```html preview
<div data-signals="{_stay: ''}" style="display: grid; gap: 12px">
  <sb-date-picker mode="range" name="stay" label="Stay" value='{"start":"2026-10-06","end":"2026-10-09"}'
    data-on:sb-change="$_stay = JSON.stringify(evt.detail)"></sb-date-picker>
  <code data-text="$_stay || 'Pick two dates…'"></code>
</div>
```

`el.value` returns `{ start, end }` (or `null` when there is no range), and setting it takes `{ start, end }`, `[start, end]` or the JSON.

### Inline, with bounds and ruled-out days

`min` and `max` limit the calendar, and `disabled-dates` rules out single days. They can be focused and read, but not picked.

```html preview
<sb-date-picker inline label="Launch window" value="2026-10-14" month="2026-10"
  min="2026-10-05" max="2026-11-20"
  disabled-dates='["2026-10-10","2026-10-11","2026-10-17","2026-10-18"]'></sb-date-picker>
```

### Booked days, one month at a time

When the user moves to another month, the picker emits `sb-month` with `{ name, year, month }` (month from 1). The page asks the server for that month's booked days, and the server answers with `disabled-dates`: a signal patch handed over with `data-attr`, or a morph. Here the page makes the days up itself:

```html preview
<div data-signals="{_booked: ['2026-10-03','2026-10-04','2026-10-12','2026-10-13','2026-10-24']}">
  <sb-date-picker inline mode="range" label="Your nights" month="2026-10"
    data-attr:disabled-dates="JSON.stringify($_booked)" data-preserve-attr="disabled-dates"
    data-on:sb-month="$_booked = [3, 4, 12, 13, 24].map((d) => [evt.detail.year, evt.detail.month, d].map((n) => String(n).padStart(2, '0')).join('-'))"></sb-date-picker>
</div>
```

On a real page, `sb-month` asks the server:

```html
<sb-date-picker mode="range" data-attr:disabled-dates="JSON.stringify($_booked)" data-preserve-attr="disabled-dates"
  data-on:sb-month="@get('/booked?year=' + evt.detail.year + '&month=' + evt.detail.month)"></sb-date-picker>
```

```go
func booked(w http.ResponseWriter, r *http.Request) {
	y, _ := strconv.Atoi(r.URL.Query().Get("year"))
	m, _ := strconv.Atoi(r.URL.Query().Get("month"))
	datastar.NewSSE(w, r).MarshalAndPatchSignals(map[string]any{
		"_booked": bookedDays(y, time.Month(m)), // ["2026-10-03", ...]
	})
}
```

The first month comes with the page, so it needs no request. `disabled-dates` is server data: it only flows in, and a new list never touches the value or a range the user is halfway through.

### Languages

Month and weekday names come from the browser's `Intl`, in the element's `lang` (or the nearest one above it, also outside another component's shadow root). The language also picks the first day of the week (Monday where the browser doesn't know) and the format the field shows and reads. A tag with an underscore (`de_DE`) works; one the browser can't read falls back to its own language.

```html preview
<div style="display: grid; gap: 12px">
  <sb-date-picker lang="de" label="Startdatum" value="2026-10-14"></sb-date-picker>
  <sb-date-picker lang="en-US" label="Start date" value="2026-10-14"></sb-date-picker>
  <sb-date-picker lang="ja" label="開始日" value="2026-10-14"></sb-date-picker>
</div>
```

### Typing

The field takes a date in the language's numeric format (`14.10.2026` in German, `10/14/2026` in US English, `2026/10/14` in Japanese, in the language's own digits) or in ISO (`2026-10-14`), with a four-digit year. The placeholder shows the pattern, e.g. `dd.mm.yyyy`. With `mode="range"`, type both dates with a dash between them (`14.10.2026 - 18.10.2026`).

The date is read when the field is committed (Enter, or leaving it). Text that isn't a date, or is a date that can't be picked, stays in the field, marked invalid, and nothing is sent: the picker never guesses. `error` sets the message shown below the field. An empty field clears the value.

```html preview
<sb-date-picker label="Return date" min="2026-10-01" error="Enter a date from 1 October 2026, like 14.10.2026." lang="de"></sb-date-picker>
```

## With commands

Give it a `name`, and it emits `sb-change` with `{ name, value }` when the value changes: ready to post as a command. With `confirm`, it sets `:state(pending)` until the server's re-rendered `value` matches, and `revert()` goes back to the server's value when a command is rejected. See [Commands and components](/contribute#commands-and-components).

```html
<sb-date-picker name="launch" confirm value="2026-10-14"
  data-on:sb-change="@post('/cmd/launch', {payload: {tabid: $tabid, ...evt.detail}})"
  data-on:datastar-fetch="evt.detail.el === el && evt.detail.type === 'error' && el.revert()"></sb-date-picker>
```

A new `value` from the server always wins, and `value=""` clears it. Markup re-sent with the same `value` leaves the user's pick alone. A range is one command: the server accepts or rejects both ends together, and validates what lies between them (a booked night inside the range, say).

`open` and `month` are view state, not part of the value: never pending, and not touched by `revert()`. The server may set them, and a changed attribute wins (`open="false"` closes). The picker reports the user's changes with `sb-toggle` (`{ name, open }`) and `sb-month`, never for a change the server made.

## Forms

Inside a `<form>`, `sb-date-picker` submits its value under its `name`: the ISO date (`launch=2026-10-14`), with `mode="range"` the JSON of its `value` attribute, and an empty string when there is no date, like `<input type="date">`. A `disabled` picker submits nothing. `new FormData(form)` and Datastar's `contentType: 'form'` include it, and a form reset brings back the server's value and clears invalid typed text. It is not a form-associated element yet (Rocket can't declare one), so `required` and validity, `<fieldset disabled>`, `<label for>` and the `form` attribute don't reach it. With commands, `sb-change` carries `{ name, value }` (see [With commands](#with-commands)).

## Styling

Style it from your page's CSS, without changing the component or importing anything into it. Custom properties, inherited properties and `::part()` all reach into its shadow root.

- **Size:** the field fills the width it is given, up to `20rem`; set `max-inline-size` on the element to change that. The field is `2.75rem` tall, and each day `2.25rem` square, so the calendar is about `17rem` wide.
- **Fonts:** the label, the field, the month and the days use your page's font.
- **Colours:** the field is `--sb-control-bg` with a `--sb-control-border` edge (`--sb-control-border-hover` on hover) and `--sb-control-text`; the placeholder is `--sb-control-placeholder`, the label `--sb-text-2`, the button and the weekdays `--sb-text-muted`. Focus draws a `--sb-brand-light` edge with a `--sb-brand-subtle` glow, and invalid text a `--sb-danger` edge and message. The calendar is `--sb-surface-raised`; a picked day is `--sb-brand` with `--sb-text-on-brand` text, the days of a range `--sb-brand-subtle`, the day under the pointer `--sb-surface-hover`, and today `--sb-brand-light`. Corners are `--sb-control-radius`, and `--sb-notch: 0` rounds the calendar and the days instead of notching them.
- **Parts:** `label`, `control` (the field's box), `input`, `button` (opens the calendar), `error`, `calendar`, `title` (the month), `nav` (the four paging buttons) and `grid`. Every date is `day`, plus `today`, `selected`, `range`, `start`, `end` or `disabled` as they apply: `::part(day today)`. Your page's `::part()` rules win over the component's own, without `!important`.

```html preview
<style>
  .my-dates { --sb-brand: #F97316; --sb-brand-light: #FDBA74; --sb-brand-subtle: rgb(249 115 22 / 0.18); --sb-notch: 0; }
  .my-dates::part(day disabled) { text-decoration: none; opacity: 0.3; }
</style>
<sb-date-picker class="my-dates" inline mode="range" month="2026-10" value='{"start":"2026-10-12","end":"2026-10-16"}' disabled-dates='["2026-10-20","2026-10-21"]'></sb-date-picker>
```

## Accessibility

It follows the ARIA date picker dialog pattern:

- **Structure:** the field is a text input with its label. The button (`aria-haspopup="dialog"`, `aria-expanded`) opens the calendar, a dialog named by the label (or "Choose date"). The days are a `grid` named by the month, whose name is announced when it changes. Each day is named by its full date; the picked ones are `aria-selected`, today is `aria-current="date"`, and a day that can't be picked is `aria-disabled`.
- **Keys in the calendar:**
  - Left and Right move a day (mirrored right to left), Up and Down a week.
  - Page Up and Page Down move a month, with Shift a year.
  - Home and End go to the first and last day of the week.
  - Enter or Space picks the day; a day that can't be picked is skipped by picking, but can still be focused and read.
  - Escape closes the calendar, also from the field, and puts the focus back on the button (inline, it drops a half-picked range). In a drawer, a modal or a popover, the first Escape closes only the calendar. Tab moves between the paging buttons and the grid, and stays in the calendar while it is open.
- **Opening:** the focus goes to the picked day, else to today. Picking a date closes the calendar and returns the focus to the button.
- **Invalid text:** the field is `aria-invalid`, and the `error` message is announced (a live region).
- **Disabled:** `disabled` takes the field, the button and the calendar out of the tab order.
- **Forced colours:** focus rings, the arrows and today's mark use system colours, and picked days `Highlight`.

The calendar is a native popover (`popover="auto"`), so it is never clipped by a scrolling container, and a click outside it closes it.
