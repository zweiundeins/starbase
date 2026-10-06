---
name: Inline Edit
tag: sb-inline-edit
category: forms
summary: Rename a title in place with a double press. The page renders the field, the server saves.
author: derekr
source: https://github.com/derekr/pd-rockets/tree/4f4111722763d2aa8f5851c397af026fcbe97e4f/rocket/inline-edit
tags: [inline edit, rename, edit in place, title, double click, pd rockets]
since: 2026-10-06
preview: |
  <style>
    .demo-edit-card { display: grid; gap: 6px; inline-size: 12rem; }
    .demo-edit-card [data-inline-edit-trigger] { all: unset; display: block; padding: 0.4rem 0.7rem; border: 1px dashed var(--sb-border); background: var(--sb-surface-card); color: var(--sb-text-1); cursor: text; }
  </style>
  <div class="demo-edit-card">
    <sb-inline-edit data-context-id="earth"><button type="button" data-inline-edit-trigger><span data-inline-edit-value>Blue Marble</span></button></sb-inline-edit>
    <sb-inline-edit data-context-id="mars"><button type="button" data-inline-edit-trigger><span data-inline-edit-value>Red Planet</span></button></sb-inline-edit>
  </div>
usage: |
  <sb-inline-edit data-context-id="42"
    data-on:sb-inline-edit-request="@get('/titles/42/edit')"
    data-on:sb-inline-edit-commit="@post('/titles/42', {payload: {title: evt.detail.value}})"
    data-on:sb-inline-edit-cancel="@get('/titles/42')">
    <button type="button" data-inline-edit-trigger><span data-inline-edit-value>Launch plan</span></button>
  </sb-inline-edit>
---

From [PD rockets](https://github.com/derekr/pd-rockets) by derekr, where it is `pd-inline-edit`: the same code, with its names in Starbase's `sb-` prefix.

A title you rename where it stands. A double press on the title asks to edit it; Enter or leaving the field commits the new text; Escape cancels. That is all the component does: it turns those gestures into three events. The field, the edit mode, the validation and the save are the page's and the server's. The server answers a request with the field, and a commit with the saved title or with the reason it refused it, so nothing shows as saved before it is.

## Examples

### Nicknames for planets

Double-press a nickname, or focus it and press Enter or F2, then type a new one. The server refuses an empty nickname and one longer than 40 characters, and keeps what you typed while it says why. Each nickname carries its state in `data-state`, sent with every event to `/demo/arrange/inline-edit`, which answers with that nickname rendered again. Nothing is stored: reload and the old names are back.

```html preview
<style>
  .demo-edits { display: grid; gap: 8px; inline-size: min(100%, 24rem); }
  .demo-edit {
    display: grid;
    grid-template-columns: 6.5rem 1fr;
    align-items: center;
    gap: 0.25rem 0.75rem;
    color: var(--sb-text-1);
  }
  .demo-edit__body { color: var(--sb-text-2); }
  .demo-edit [data-inline-edit-trigger], .demo-edit [data-inline-edit-input] {
    box-sizing: border-box;
    inline-size: 100%;
    padding: 0.45rem 0.7rem;
    border: 1px solid var(--sb-border);
    background: var(--sb-surface-card);
    color: inherit;
    font: inherit;
    text-align: start;
  }
  .demo-edit [data-inline-edit-trigger] { border-style: dashed; cursor: text; }
  .demo-edit [data-inline-edit-trigger]:hover { border-color: var(--sb-border-strong); }
  .demo-edit [data-inline-edit-input] { border-color: var(--sb-brand); background: var(--sb-surface-inset); }
  .demo-edit :focus-visible { outline: 2px solid var(--sb-brand-light); outline-offset: 2px; }
  .demo-edit [aria-invalid="true"] { border-color: var(--sb-danger); }
  .demo-edit__problem { grid-column: 2; color: var(--sb-danger); font-size: 0.8125rem; }
</style>
<div class="demo-edits">
<sb-inline-edit id="nick-earth" class="demo-edit" data-context-id="earth" data-state="body=earth name=Blue+Marble"
	data-on:sb-inline-edit-request="@get('/demo/arrange/inline-edit', {requestCancellation: 'disabled', payload: {id: el.id, state: el.dataset.state, move: {type: 'request', contextId: evt.detail.contextId}}})"
	data-on:sb-inline-edit-commit="@get('/demo/arrange/inline-edit', {requestCancellation: 'disabled', payload: {id: el.id, state: el.dataset.state, move: {type: 'commit', contextId: evt.detail.contextId, value: evt.detail.value}}})"
	data-on:sb-inline-edit-cancel="@get('/demo/arrange/inline-edit', {requestCancellation: 'disabled', payload: {id: el.id, state: el.dataset.state, move: {type: 'cancel', contextId: evt.detail.contextId}}})"
	data-on:keydown="(evt.key === 'Enter' || evt.key === 'F2') && evt.target.matches('[data-inline-edit-trigger]') && @get('/demo/arrange/inline-edit', {requestCancellation: 'disabled', payload: {id: el.id, state: el.dataset.state, move: {type: 'request', contextId: el.dataset.contextId}}})">
	<span class="demo-edit__body">🪐 Earth</span>
	<button type="button" data-inline-edit-trigger aria-label="Blue Marble, nickname of Earth: press Enter to rename"><span data-inline-edit-value>Blue Marble</span></button>
</sb-inline-edit>
<sb-inline-edit id="nick-mars" class="demo-edit" data-context-id="mars" data-state="body=mars name=Red+Planet"
	data-on:sb-inline-edit-request="@get('/demo/arrange/inline-edit', {requestCancellation: 'disabled', payload: {id: el.id, state: el.dataset.state, move: {type: 'request', contextId: evt.detail.contextId}}})"
	data-on:sb-inline-edit-commit="@get('/demo/arrange/inline-edit', {requestCancellation: 'disabled', payload: {id: el.id, state: el.dataset.state, move: {type: 'commit', contextId: evt.detail.contextId, value: evt.detail.value}}})"
	data-on:sb-inline-edit-cancel="@get('/demo/arrange/inline-edit', {requestCancellation: 'disabled', payload: {id: el.id, state: el.dataset.state, move: {type: 'cancel', contextId: evt.detail.contextId}}})"
	data-on:keydown="(evt.key === 'Enter' || evt.key === 'F2') && evt.target.matches('[data-inline-edit-trigger]') && @get('/demo/arrange/inline-edit', {requestCancellation: 'disabled', payload: {id: el.id, state: el.dataset.state, move: {type: 'request', contextId: el.dataset.contextId}}})">
	<span class="demo-edit__body">🪐 Mars</span>
	<button type="button" data-inline-edit-trigger aria-label="Red Planet, nickname of Mars: press Enter to rename"><span data-inline-edit-value>Red Planet</span></button>
</sb-inline-edit>
<sb-inline-edit id="nick-jupiter" class="demo-edit" data-context-id="jupiter" data-state="body=jupiter name=Gas+Giant"
	data-on:sb-inline-edit-request="@get('/demo/arrange/inline-edit', {requestCancellation: 'disabled', payload: {id: el.id, state: el.dataset.state, move: {type: 'request', contextId: evt.detail.contextId}}})"
	data-on:sb-inline-edit-commit="@get('/demo/arrange/inline-edit', {requestCancellation: 'disabled', payload: {id: el.id, state: el.dataset.state, move: {type: 'commit', contextId: evt.detail.contextId, value: evt.detail.value}}})"
	data-on:sb-inline-edit-cancel="@get('/demo/arrange/inline-edit', {requestCancellation: 'disabled', payload: {id: el.id, state: el.dataset.state, move: {type: 'cancel', contextId: evt.detail.contextId}}})"
	data-on:keydown="(evt.key === 'Enter' || evt.key === 'F2') && evt.target.matches('[data-inline-edit-trigger]') && @get('/demo/arrange/inline-edit', {requestCancellation: 'disabled', payload: {id: el.id, state: el.dataset.state, move: {type: 'request', contextId: el.dataset.contextId}}})">
	<span class="demo-edit__body">🪐 Jupiter</span>
	<button type="button" data-inline-edit-trigger aria-label="Gas Giant, nickname of Jupiter: press Enter to rename"><span data-inline-edit-value>Gas Giant</span></button>
</sb-inline-edit>
</div>
```

## Markup and events

The component listens inside its own element and emits three events. Their `contextId` is the element's `data-context-id` (`""` without one), so one handler can serve many titles.

| Element | Role |
|---|---|
| `[data-inline-edit-trigger]` | What a double press on asks to edit: two presses of the main button within 500 ms. |
| `[data-inline-edit-value]` | The saved text. A commit that leaves it unchanged is a cancel. |
| `[data-inline-edit-input]` | The field, once the page renders it. |

| Event | Detail | When |
|---|---|---|
| `sb-inline-edit-request` | `{ contextId }` | A double press on the trigger. |
| `sb-inline-edit-commit` | `{ contextId, value }` | Enter in the field, or the field loses the focus, with a value other than the saved text. |
| `sb-inline-edit-cancel` | `{ contextId }` | Escape in the field, or the field loses the focus with the saved text still in it. |

The events bubble. The component never shows or hides the field: render the field on a request and the title again on a commit or a cancel. Keep the saved text in a `[data-inline-edit-value]` while the field shows (it may be hidden), or every blur commits.

## Keyboard

| Keys | Action |
|---|---|
| Enter in the field | Commit (the field gives up the focus) |
| Escape in the field | Cancel |

Text shortcuts in the field are the browser's. Starting an edit takes a double press, so the page gives keyboard users their own way in. The example listens for Enter and F2 on the title, a focusable button:

```html
data-on:keydown="(evt.key === 'Enter' || evt.key === 'F2') && evt.target.matches('[data-inline-edit-trigger]') && @get(…)"
```

## On the server

The server owns the text and decides. A handler applies each event and renders the title again; this is the demo's, which keeps each nickname in its markup instead of a database:

```go source=internal/web/demo_arrange_edit.go#arrangeInlineEdit,renderInlineEdit
// arrangeInlineEdit applies an sb-inline-edit event ({type, contextId,
// value}) to a nickname: a request opens the field, a commit saves a valid
// name or keeps the text with the reason it was refused, a cancel closes it.
func arrangeInlineEdit(state string, move json.RawMessage) (string, error) {
	n, err := parseNickname(state)
	if err != nil {
		return "", err
	}
	var m struct {
		Type      string `json:"type"`
		ContextID string `json:"contextId"`
		Value     string `json:"value"`
	}
	if err := json.Unmarshal(move, &m); err != nil {
		return "", err
	}
	if m.ContextID != n.body {
		return "", fmt.Errorf("the event is about %q, the nickname of %q", m.ContextID, n.body)
	}
	switch m.Type {
	case "request":
		if !n.editing { // a second request while the field is open changes nothing
			n = nickname{body: n.body, name: n.name, editing: true, draft: n.name}
		}
	case "commit":
		value := strings.TrimSpace(m.Value)
		if problem := nicknameProblem(value); problem != "" {
			n.editing, n.draft, n.problem, n.try = true, truncate(m.Value, 80), problem, n.try+1
		} else {
			n = nickname{body: n.body, name: value, back: true}
		}
	case "cancel":
		n = nickname{body: n.body, name: n.name, back: true}
	default:
		return "", fmt.Errorf("unknown event %q", m.Type)
	}
	return n.String(), nil
}

// renderInlineEdit is one nickname's markup: the host the morph replaces,
// with the state in data-state, showing the name or the field.
func renderInlineEdit(id, state string) string {
	n, _ := parseNickname(state)
	on := func(event, move string) string {
		return fmt.Sprintf("\n\tdata-on:%s=\"@get('/demo/arrange/inline-edit', {requestCancellation: 'disabled', payload: {id: el.id, state: el.dataset.state, move: %s}})\"", event, move)
	}
	var b strings.Builder
	fmt.Fprintf(&b, "<sb-inline-edit id=\"%s\" class=\"demo-edit\" data-context-id=\"%s\" data-state=\"%s\"", id, n.body, html.EscapeString(state))
	b.WriteString(on("sb-inline-edit-request", "{type: 'request', contextId: evt.detail.contextId}"))
	b.WriteString(on("sb-inline-edit-commit", "{type: 'commit', contextId: evt.detail.contextId, value: evt.detail.value}"))
	b.WriteString(on("sb-inline-edit-cancel", "{type: 'cancel', contextId: evt.detail.contextId}"))
	// The keyboard's way in, which the component leaves to the page: Enter or F2 on the name.
	b.WriteString("\n\tdata-on:keydown=\"(evt.key === 'Enter' || evt.key === 'F2') && evt.target.matches('[data-inline-edit-trigger]') && @get('/demo/arrange/inline-edit', {requestCancellation: 'disabled', payload: {id: el.id, state: el.dataset.state, move: {type: 'request', contextId: el.dataset.contextId}}})\"")
	fmt.Fprintf(&b, ">\n\t<span class=\"demo-edit__body\">%s</span>\n", label(n.body))
	// A new field, or the name after the field closed, takes the focus when
	// nothing else has it (a morph removed what had it): never on page load.
	const focus = " data-init=\"document.activeElement === document.body && el.focus()\""
	name := html.EscapeString(n.name)
	if !n.editing {
		init := ""
		if n.back {
			init = focus
		}
		fmt.Fprintf(&b, "\t<button type=\"button\" data-inline-edit-trigger aria-label=\"%s, nickname of %s: press Enter to rename\"%s><span data-inline-edit-value>%s</span></button>\n", name, html.EscapeString(bodies()[n.body].Name), init, name)
	} else {
		described := ""
		if n.problem != "" {
			described = fmt.Sprintf(" aria-invalid=\"true\" aria-describedby=\"%s-problem\"", id)
		}
		fmt.Fprintf(&b, "\t<input id=\"%s-field-%d\" data-inline-edit-input value=\"%s\" maxlength=\"80\" aria-label=\"Nickname of %s\"%s%s>\n", id, n.try, html.EscapeString(n.draft), html.EscapeString(bodies()[n.body].Name), described, focus)
		fmt.Fprintf(&b, "\t<span data-inline-edit-value hidden>%s</span>\n", name)
		if n.problem != "" {
			fmt.Fprintf(&b, "\t<span id=\"%s-problem\" class=\"demo-edit__problem\" role=\"alert\">%s</span>\n", id, nicknameProblems[n.problem])
		}
	}
	b.WriteString("</sb-inline-edit>")
	return b.String()
}
```

The handlers use `requestCancellation: 'disabled'`. Datastar cancels a request still on its way when the same method and URL are called again, and every title here calls the same URL: without it, a commit on one title (its blur) could be cancelled by a double press on the next. Each answer morphs only its own title, so they can't overwrite each other.

## Styling

The component adds no styles and sets no attributes: the title, the field and any message are your page's markup, styled by your page's CSS. Style the states the server renders, such as a field with `aria-invalid="true"`. The example above shows one way.

## Accessibility

The title is yours to make reachable: a `<button>` (as in the example) takes the focus and reads its text, and its `aria-label` can say how to rename it. Give the field a label. The example's server marks a refused field `aria-invalid` and points `aria-describedby` at the reason, which is a `role="alert"` so a screen reader announces it, and it moves the focus into a new field and back to the title when the field closes, but only when nothing else holds it. The component itself announces nothing.

## Licence

The code is PD rockets', under its Beer-Ware licence, in `LICENSE-pd-rockets.txt` next to it. `go run ./cmd/vendorpd` brings in a newer release.
