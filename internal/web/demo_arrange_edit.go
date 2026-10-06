package web

import (
	"encoding/json"
	"fmt"
	"html"
	"net/url"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

func init() {
	arrangers["inline-edit"] = arranger{arrange: arrangeInlineEdit, render: renderInlineEdit}
}

// nickname is one sb-inline-edit of the demo: a body's nickname and, while
// it is edited, the text in the field and why the server refused it.
type nickname struct {
	body, name string
	editing    bool
	draft      string
	problem    string // "", "empty", "long" or "control"
	try        int    // refused commits: each one renders a new field, which takes the focus
	back       bool   // the field just closed: the name takes the focus if nothing has it
}

var nicknameProblems = map[string]string{
	"empty":   "A nickname needs at least one character.",
	"long":    "A nickname has at most 40 characters.",
	"control": "A nickname can't hold control characters.",
}

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

// nicknameProblem is why a nickname can't be saved, or "".
func nicknameProblem(s string) string {
	switch {
	case s == "":
		return "empty"
	case utf8.RuneCountInString(s) > 40:
		return "long"
	case strings.ContainsFunc(s, unicode.IsControl):
		return "control"
	}
	return ""
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

// parseNickname reads a nickname's data-state: words like body=earth and
// name=Blue+Marble, the texts query-escaped; edit=1, draft=…, problem=…
// and try=… while the field is open, back=1 once it closed.
func parseNickname(state string) (nickname, error) {
	var n nickname
	for _, w := range strings.Fields(state) {
		k, v, _ := strings.Cut(w, "=")
		text, err := url.QueryUnescape(v)
		if err != nil {
			return n, fmt.Errorf("%q: %v", w, err)
		}
		switch k {
		case "body":
			n.body = text
		case "name":
			n.name = text
		case "edit":
			n.editing = text == "1"
		case "draft":
			n.draft = text
		case "problem":
			n.problem = text
		case "try":
			n.try, err = strconv.Atoi(text)
		case "back":
			n.back = text == "1"
		default:
			err = fmt.Errorf("unknown field %q", k)
		}
		if err != nil {
			return n, err
		}
	}
	if _, ok := bodies()[n.body]; !ok {
		return n, fmt.Errorf("%q is not a body", n.body)
	}
	if nicknameProblem(n.name) != "" || n.name != strings.TrimSpace(n.name) {
		return n, fmt.Errorf("%q is not a nickname", n.name)
	}
	if _, ok := nicknameProblems[n.problem]; (n.problem != "" && !ok) || utf8.RuneCountInString(n.draft) > 80 || n.try < 0 || n.try > 999 {
		return n, fmt.Errorf("%q is not a field's state", state)
	}
	return n, nil
}

// String is the nickname's data-state.
func (n nickname) String() string {
	s := "body=" + url.QueryEscape(n.body) + " name=" + url.QueryEscape(n.name)
	if n.back {
		s += " back=1"
	}
	if n.editing {
		s += " edit=1 draft=" + url.QueryEscape(n.draft)
		if n.problem != "" {
			s += " problem=" + n.problem + " try=" + strconv.Itoa(n.try)
		}
	}
	return s
}

func truncate(s string, n int) string {
	if r := []rune(s); len(r) > n {
		return string(r[:n])
	}
	return s
}
