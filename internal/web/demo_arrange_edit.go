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
	problem    string // "", "empty", "long", "control" or "format"
	try        int    // refused commits: each one renders a new alert
	focus      bool   // a key ended the last edit: the answer takes the focus
}

var nicknameProblems = map[string]string{
	"empty":   "A nickname needs at least one character.",
	"long":    "A nickname has at most 40 characters.",
	"control": "A nickname can't hold control characters.",
	"format":  "A nickname can't hold invisible formatting characters.",
}

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

// nicknameProblem is why a nickname can't be saved, or "".
func nicknameProblem(s string) string {
	joiner := func(r rune) bool { return r == '\u200c' || r == '\u200d' } // emoji sequences and some scripts need them
	switch {
	case strings.TrimFunc(s, func(r rune) bool { return unicode.IsSpace(r) || joiner(r) }) == "":
		return "empty"
	case utf8.RuneCountInString(s) > 40:
		return "long"
	case strings.ContainsFunc(s, unicode.IsControl):
		return "control"
	case strings.ContainsFunc(s, func(r rune) bool { return unicode.Is(unicode.Cf, r) && !joiner(r) }):
		return "format"
	}
	return ""
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

// parseNickname reads a nickname's data-state: words like body=earth and
// name=Blue+Marble, the texts query-escaped; edit=1, draft=…, problem=…
// and try=… while the field is open; focus=1 when a key ended the edit.
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
		case "focus":
			n.focus = text == "1"
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
	if n.editing {
		s += " edit=1 draft=" + url.QueryEscape(n.draft)
		if n.problem != "" {
			s += " problem=" + n.problem + " try=" + strconv.Itoa(n.try)
		}
	}
	if n.focus {
		s += " focus=1"
	}
	return s
}

func truncate(s string, n int) string {
	if r := []rune(s); len(r) > n {
		return string(r[:n])
	}
	return s
}
