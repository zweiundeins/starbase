---
name: Inline Edit
tag: sb-inline-edit
category: forms
summary: Rename a title in place with a double press. The page renders the field, the server saves.
author: derekr
license: Beerware
source: https://github.com/derekr/pd-rockets/tree/4f4111722763d2aa8f5851c397af026fcbe97e4f/rocket/inline-edit
tags: [inline edit, rename, edit in place, title, double click, pd rockets]
since: 2026-10-06
preview: |
  <style>
    .demo-edit-card { display: grid; gap: 6px; inline-size: min(100%, 12rem); }
    .demo-edit-card [data-inline-edit-trigger] { display: block; box-sizing: border-box; inline-size: 100%; margin: 0; padding: 0.4rem 0.7rem; border: 1px dashed var(--sb-border); background: var(--sb-surface-card); color: var(--sb-text-1); font: inherit; text-align: start; cursor: text; }
    .demo-edit-card [data-inline-edit-trigger]:focus-visible { outline: 2px solid var(--sb-brand-light); outline-offset: 2px; }
    @media (forced-colors: active) { .demo-edit-card [data-inline-edit-trigger]:focus-visible { outline-color: Highlight; } }
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

From [PD rockets](https://github.com/derekr/pd-rockets) by derekr, where it is `pd-inline-edit`. This copy carries the patches in `patches/pd-rockets`: Enter or F2 on the title asks to edit it, a held Enter counts once, and a blur waits until the page's update is done and compares the field with the saved text it started from.

A title you rename where it stands. A double press on the title, or Enter or F2 while it has the focus, asks to edit it; Enter or leaving the field commits the new text; Escape cancels. That is all the component does: it turns those gestures into three events. The field, the edit mode, the validation and the save are the page's and the server's. The server answers a request with the field, and a commit with the saved title or with the reason it refused it, so nothing shows as saved before it is.

## Examples

### Nicknames for planets

Double-press a nickname, or focus it and press Enter or F2, then type a new one. The server refuses an empty nickname, one longer than 40 characters and one with control or invisible formatting characters, and keeps what you typed while it says why. While a request is out the nickname fades, and a request that fails says so below it. Each nickname carries its state in `data-state`, sent with every event to `/demo/arrange/inline-edit`, which answers with that nickname rendered again. Nothing is stored: reload and the old names are back.

```html preview
<style>
  .demo-edits { display: grid; gap: 8px; inline-size: min(100%, 24rem); }
  .demo-edit {
    display: grid;
    grid-template-columns: 6.5rem 1fr;
    align-items: center;
    column-gap: 0.75rem;
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
  .demo-edit[aria-busy="true"] :is([data-inline-edit-trigger], [data-inline-edit-input]) { opacity: 0.6; }
  .demo-edit__problem, .demo-edit__failed { grid-column: 2; color: var(--sb-danger); font-size: 0.8125rem; }
  .demo-edit__problem, .demo-edit__failed:not(:empty) { margin-block-start: 0.25rem; }
  @media (forced-colors: active) {
    .demo-edit :focus-visible { outline-color: Highlight; }
    .demo-edit [aria-invalid="true"] { border: 3px double CanvasText; }
  }
</style>
<div class="demo-edits">
<sb-inline-edit id="nick-earth" class="demo-edit" data-context-id="earth" data-state="body=earth name=Blue+Marble"
	data-on:sb-inline-edit-request="@get('/demo/arrange/inline-edit', {requestCancellation: 'disabled', payload: {id: el.id, state: el.dataset.state, move: {type: 'request', contextId: evt.detail.contextId}}})"
	data-on:sb-inline-edit-commit="@get('/demo/arrange/inline-edit', {requestCancellation: 'disabled', payload: {id: el.id, state: el.dataset.state, move: {type: 'commit', contextId: evt.detail.contextId, value: evt.detail.value, key: $_nick_earth_key}}})"
	data-on:sb-inline-edit-cancel="@get('/demo/arrange/inline-edit', {requestCancellation: 'disabled', payload: {id: el.id, state: el.dataset.state, move: {type: 'cancel', contextId: evt.detail.contextId, key: $_nick_earth_key}}})"
	data-indicator="_nick_earth_saving" data-attr:aria-busy="String($_nick_earth_saving)" data-preserve-attr="aria-busy"
	data-on:datastar-fetch="evt.detail.el === el && ['started', 'error', 'retries-failed'].includes(evt.detail.type) && ($_nick_earth_failed = evt.detail.type !== 'started')">
	<span class="demo-edit__body">🪐 Earth</span>
	<button type="button" data-inline-edit-trigger aria-label="Blue Marble, nickname of Earth: press Enter to rename"><span data-inline-edit-value>Blue Marble</span></button>
	<span id="nick-earth-failed" class="demo-edit__failed" role="alert" data-text="$_nick_earth_failed ? 'Nothing changed: the request failed. Try again.' : ''"></span>
</sb-inline-edit>
<sb-inline-edit id="nick-mars" class="demo-edit" data-context-id="mars" data-state="body=mars name=Red+Planet"
	data-on:sb-inline-edit-request="@get('/demo/arrange/inline-edit', {requestCancellation: 'disabled', payload: {id: el.id, state: el.dataset.state, move: {type: 'request', contextId: evt.detail.contextId}}})"
	data-on:sb-inline-edit-commit="@get('/demo/arrange/inline-edit', {requestCancellation: 'disabled', payload: {id: el.id, state: el.dataset.state, move: {type: 'commit', contextId: evt.detail.contextId, value: evt.detail.value, key: $_nick_mars_key}}})"
	data-on:sb-inline-edit-cancel="@get('/demo/arrange/inline-edit', {requestCancellation: 'disabled', payload: {id: el.id, state: el.dataset.state, move: {type: 'cancel', contextId: evt.detail.contextId, key: $_nick_mars_key}}})"
	data-indicator="_nick_mars_saving" data-attr:aria-busy="String($_nick_mars_saving)" data-preserve-attr="aria-busy"
	data-on:datastar-fetch="evt.detail.el === el && ['started', 'error', 'retries-failed'].includes(evt.detail.type) && ($_nick_mars_failed = evt.detail.type !== 'started')">
	<span class="demo-edit__body">🪐 Mars</span>
	<button type="button" data-inline-edit-trigger aria-label="Red Planet, nickname of Mars: press Enter to rename"><span data-inline-edit-value>Red Planet</span></button>
	<span id="nick-mars-failed" class="demo-edit__failed" role="alert" data-text="$_nick_mars_failed ? 'Nothing changed: the request failed. Try again.' : ''"></span>
</sb-inline-edit>
<sb-inline-edit id="nick-jupiter" class="demo-edit" data-context-id="jupiter" data-state="body=jupiter name=Gas+Giant"
	data-on:sb-inline-edit-request="@get('/demo/arrange/inline-edit', {requestCancellation: 'disabled', payload: {id: el.id, state: el.dataset.state, move: {type: 'request', contextId: evt.detail.contextId}}})"
	data-on:sb-inline-edit-commit="@get('/demo/arrange/inline-edit', {requestCancellation: 'disabled', payload: {id: el.id, state: el.dataset.state, move: {type: 'commit', contextId: evt.detail.contextId, value: evt.detail.value, key: $_nick_jupiter_key}}})"
	data-on:sb-inline-edit-cancel="@get('/demo/arrange/inline-edit', {requestCancellation: 'disabled', payload: {id: el.id, state: el.dataset.state, move: {type: 'cancel', contextId: evt.detail.contextId, key: $_nick_jupiter_key}}})"
	data-indicator="_nick_jupiter_saving" data-attr:aria-busy="String($_nick_jupiter_saving)" data-preserve-attr="aria-busy"
	data-on:datastar-fetch="evt.detail.el === el && ['started', 'error', 'retries-failed'].includes(evt.detail.type) && ($_nick_jupiter_failed = evt.detail.type !== 'started')">
	<span class="demo-edit__body">🪐 Jupiter</span>
	<button type="button" data-inline-edit-trigger aria-label="Gas Giant, nickname of Jupiter: press Enter to rename"><span data-inline-edit-value>Gas Giant</span></button>
	<span id="nick-jupiter-failed" class="demo-edit__failed" role="alert" data-text="$_nick_jupiter_failed ? 'Nothing changed: the request failed. Try again.' : ''"></span>
</sb-inline-edit>
</div>
```

## Markup and events

The component listens inside its own element and emits three events. Their `contextId` is the element's `data-context-id` (`""` without one), so one handler can serve many titles.

| Element | Role |
|---|---|
| `[data-inline-edit-trigger]` | The title. A double press on it (two presses of the main button within 500 ms) asks to edit it, and so do Enter and F2 while it has the focus: make it a `<button>` or give it `tabindex="0"`. |
| `[data-inline-edit-value]` | The saved text. The component reads it when the field takes the focus. |
| `[data-inline-edit-input]` | The field, once the page renders it. |

| Event | Detail | When |
|---|---|---|
| `sb-inline-edit-request` | `{ contextId }` | A double press on the trigger, or Enter or F2 on it. |
| `sb-inline-edit-commit` | `{ contextId, value }` | Enter in the field, or the field loses the focus, with a text other than the saved one. |
| `sb-inline-edit-cancel` | `{ contextId }` | Escape in the field, or Enter or a lost focus with the saved text still in it. |

The events bubble. The component never shows or hides the field: render the field on a request and the title again on a commit or a cancel. Render the saved text in a `[data-inline-edit-value]` next to the field (it may be hidden), or every blur commits. The component decides a blur a microtask later, once the page's update is done: a field your page removed sends nothing, and neither does one that has the focus again.

## Keyboard

| Keys | Action |
|---|---|
| Enter or F2 on the title | Ask to edit |
| Enter in the field | Commit, or cancel when the text is unchanged (the field gives up the focus) |
| Escape in the field | Cancel |

A held Enter counts once, so its repeats can't land in the field the page has just opened. Text shortcuts in the field are the browser's.

## On the server

The server owns the text and decides. A handler applies each event and renders the title again; this is the demo's, which keeps each nickname in its markup instead of a database:

```go source=internal/web/demo_arrange_edit.go#arrangeInlineEdit,renderInlineEdit
// arrangeInlineEdit applies an sb-inline-edit event ({type, contextId,
// value, key}) to a nickname: a request opens the field, a commit saves a
// valid name or keeps the text with the reason it was refused, a cancel
// closes it. key is the last key pressed in the field: Enter or Escape
// means the keyboard ended the edit, so the answer moves the focus.
func arrangeInlineEdit(state string, move json.RawMessage) (string, error) {
	n, err := parseNickname(state)
	if err != nil {
		return "", err
	}
	var m struct {
		Type      string `json:"type"`
		ContextID string `json:"contextId"`
		Value     string `json:"value"`
		Key       string `json:"key"`
	}
	if err := json.Unmarshal(move, &m); err != nil {
		return "", err
	}
	if m.ContextID != n.body {
		return "", fmt.Errorf("the event is about %q, the nickname of %q", m.ContextID, n.body)
	}
	if (m.Type == "commit" || m.Type == "cancel") && !n.editing {
		return "", fmt.Errorf("the nickname of %q isn't being edited", n.body)
	}
	keyed := m.Key == "Enter" || m.Key == "Escape"
	switch m.Type {
	case "request":
		if !n.editing { // a second request while the field is open changes nothing
			n = nickname{body: n.body, name: n.name, editing: true, draft: n.name}
		}
	case "commit":
		value := strings.TrimSpace(m.Value)
		if problem := nicknameProblem(value); problem != "" {
			n.draft, n.problem, n.try, n.focus = truncate(m.Value, 80), problem, n.try+1, keyed
		} else {
			n = nickname{body: n.body, name: value, focus: keyed}
		}
	case "cancel":
		n = nickname{body: n.body, name: n.name, focus: keyed}
	default:
		return "", fmt.Errorf("unknown event %q", m.Type)
	}
	return n.String(), nil
}

// renderInlineEdit is one nickname's markup: the host the morph replaces,
// with the state in data-state, showing the name or the field.
func renderInlineEdit(id, state string) string {
	n, _ := parseNickname(state)
	signal := "_" + strings.ReplaceAll(id, "-", "_") // this nickname's signals: _key, _saving, _failed
	on := func(event, move string) string {
		return fmt.Sprintf("\n\tdata-on:%s=\"@get('/demo/arrange/inline-edit', {requestCancellation: 'disabled', payload: {id: el.id, state: el.dataset.state, move: %s}})\"", event, move)
	}
	var b strings.Builder
	fmt.Fprintf(&b, "<sb-inline-edit id=\"%s\" class=\"demo-edit\" data-context-id=\"%s\" data-state=\"%s\"", id, n.body, html.EscapeString(state))
	b.WriteString(on("sb-inline-edit-request", "{type: 'request', contextId: evt.detail.contextId}"))
	b.WriteString(on("sb-inline-edit-commit", "{type: 'commit', contextId: evt.detail.contextId, value: evt.detail.value, key: $"+signal+"_key}"))
	b.WriteString(on("sb-inline-edit-cancel", "{type: 'cancel', contextId: evt.detail.contextId, key: $"+signal+"_key}"))
	// Pending while a request is out; a failed one shows a message until the next answer.
	fmt.Fprintf(&b, "\n\tdata-indicator=\"%s_saving\" data-attr:aria-busy=\"String($%[1]s_saving)\" data-preserve-attr=\"aria-busy\"", signal)
	fmt.Fprintf(&b, "\n\tdata-on:datastar-fetch=\"evt.detail.el === el && ['started', 'error', 'retries-failed'].includes(evt.detail.type) && ($%s_failed = evt.detail.type !== 'started')\"", signal)
	fmt.Fprintf(&b, ">\n\t<span class=\"demo-edit__body\">%s</span>\n", label(n.body))
	// The answer to a request or a key moves the focus, and only from the body, where Enter or the removed
	// name or field left it. A morph of the host re-runs every data-init in it, so only those answers carry one.
	const unfocused = "document.activeElement === document.body"
	name, of := html.EscapeString(n.name), html.EscapeString(bodies()[n.body].Name)
	if !n.editing {
		init := ""
		if n.focus {
			init = " data-init=\"" + unfocused + " && el.focus()\""
		}
		fmt.Fprintf(&b, "\t<button type=\"button\" data-inline-edit-trigger aria-label=\"%s, nickname of %s: press Enter to rename\"%s><span data-inline-edit-value>%s</span></button>\n", name, of, init, name)
	} else {
		// One field for the whole edit: a refusal keeps it, and what the user typed since.
		field := id + "-field"
		described, init := "", " data-init=\""+unfocused+" && el.focus()\""
		if n.problem != "" {
			described, init = fmt.Sprintf(" aria-invalid=\"true\" aria-describedby=\"%s-problem-%d\"", id, n.try), ""
		}
		fmt.Fprintf(&b, "\t<input id=\"%s\" data-inline-edit-input value=\"%s\" maxlength=\"80\" aria-label=\"Nickname of %s\"%s data-preserve-attr=\"value\"\n", field, html.EscapeString(n.draft), of, described)
		fmt.Fprintf(&b, "\t\tdata-on:focus=\"$%s_key = ''\" data-on:keydown=\"$%[1]s_key = evt.key\"%s>\n", signal, init)
		fmt.Fprintf(&b, "\t<span data-inline-edit-value hidden>%s</span>\n", name)
		if n.problem != "" {
			// A new alert for every refusal, so the same reason is announced again.
			init := ""
			if n.focus {
				init = fmt.Sprintf(" data-init=\"%s && document.getElementById('%s').focus()\"", unfocused, field)
			}
			fmt.Fprintf(&b, "\t<span id=\"%s-problem-%d\" class=\"demo-edit__problem\" role=\"alert\"%s>%s</span>\n", id, n.try, init, nicknameProblems[n.problem])
		}
	}
	fmt.Fprintf(&b, "\t<span id=\"%s-failed\" class=\"demo-edit__failed\" role=\"alert\" data-text=\"$%s_failed ? 'Nothing changed: the request failed. Try again.' : ''\"></span>\n", id, signal)
	b.WriteString("</sb-inline-edit>")
	return b.String()
}
```

The field keeps one id for the whole edit and `data-preserve-attr="value"`, so an answer that refuses a commit leaves the field in place with whatever was typed since. Each refusal renders a new `role="alert"` with its own id, so a screen reader announces a repeated reason again.

A new field takes the focus, and after that the server moves it only when a key ended the edit. The field keeps the last key pressed in it in `$_<id>_key` (each nickname has its own signals, named after the host's id with `_` for `-`; the key is cleared when the field takes the focus), and the commit and the cancel send it along. After a refused Enter the new alert puts the focus back into the field; after Enter or Escape closed the field, the title takes it. All of this happens only while the focus is on the page's body, where Enter or the removed title or field left it, so after a click or a Tab elsewhere the focus stays where the user put it. A morph of the host runs every `data-init` in it again, so only the answers that move the focus carry one.

`data-indicator` sets `$_<id>_saving` while a request from that nickname is out, `data-attr` turns it into `aria-busy="true"`, and the stylesheet fades the title or the field; `data-preserve-attr` keeps the attribute through the answer's morph. A request that fails (an error status, or no answer after Datastar's retries) sets `$_<id>_failed`, and the message below the title says that nothing changed. The next request clears it.

The handlers use `requestCancellation: 'disabled'`. Datastar cancels a request still on its way when the same method and URL are called again, and every title here calls the same URL: without it, a commit on one title (its blur) could be cancelled by a double press on the next. Each answer morphs only its own title, so answers for different titles never touch each other. On one title the last answer to arrive wins, and the server refuses a commit or a cancel for a title whose field it has closed.

## Styling

The component adds no styles and sets no attributes: the title, the field and any message are your page's markup, styled by your page's CSS. Style the states the server renders, such as a field with `aria-invalid="true"`, and the ones the page sets, such as `aria-busy="true"` while a request is out. The example above shows one way, with a forced-colours block that keeps the focus ring and a refused field visible.

## Accessibility

The title is yours to make reachable: a `<button>` (as in the example) takes the focus and reads its text, its `aria-label` can say how to rename it, and Enter or F2 on it asks to edit. Give the field a label. The example's server marks a refused field `aria-invalid` and points `aria-describedby` at the reason, a new `role="alert"` for every refusal, so a screen reader announces each one. On the server says when it moves the focus. The component itself announces nothing.

## Licence

The code is PD rockets', under its Beer-Ware licence, in `LICENSE-pd-rockets.txt` next to it. `go run ./cmd/vendorpd` brings in a newer release and applies the patches again.
