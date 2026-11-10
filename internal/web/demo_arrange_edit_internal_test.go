package web

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestArrangeInlineEdit(t *testing.T) {
	const closed = "body=earth name=Blue+Marble"
	const open = "body=earth name=Blue+Marble edit=1 draft=Blue+Marble"
	const refused = "body=earth name=Blue+Marble edit=1 draft=+ problem=empty try=1 focus=1"
	for _, tc := range []struct {
		name, state, move string
		want              string // "" for a refusal
	}{
		{"a request opens the field", closed, `{"type":"request","contextId":"earth"}`, open},
		{"a request after a key closed it", closed + " focus=1", `{"type":"request","contextId":"earth"}`, open},
		{"a second request changes nothing", refused, `{"type":"request","contextId":"earth"}`, refused},
		{"Enter saves and moves the focus", open, `{"type":"commit","contextId":"earth","value":" Pale Blue Dot ","key":"Enter"}`, "body=earth name=Pale+Blue+Dot focus=1"},
		{"a blur saves and leaves the focus", open, `{"type":"commit","contextId":"earth","value":"Pale Blue Dot","key":"t"}`, "body=earth name=Pale+Blue+Dot"},
		{"Enter on an empty name is refused, the focus goes back", open, `{"type":"commit","contextId":"earth","value":" ","key":"Enter"}`, refused},
		{"a blur on an empty name is refused, the focus stays", open, `{"type":"commit","contextId":"earth","value":""}`, "body=earth name=Blue+Marble edit=1 draft= problem=empty try=1"},
		{"each refusal counts", refused, `{"type":"commit","contextId":"earth","value":"\u200b","key":"Enter"}`, "body=earth name=Blue+Marble edit=1 draft=%E2%80%8B problem=format try=2 focus=1"},
		{"a long name is refused", open, `{"type":"commit","contextId":"earth","value":"` + strings.Repeat("x", 41) + `"}`, "body=earth name=Blue+Marble edit=1 draft=" + strings.Repeat("x", 41) + " problem=long try=1"},
		{"the draft is cut at 80", open, `{"type":"commit","contextId":"earth","value":"` + strings.Repeat("y", 90) + `"}`, "body=earth name=Blue+Marble edit=1 draft=" + strings.Repeat("y", 80) + " problem=long try=1"},
		{"a control character is refused", open, `{"type":"commit","contextId":"earth","value":"Blue\u0007"}`, "body=earth name=Blue+Marble edit=1 draft=Blue%07 problem=control try=1"},
		{"a bidi override is refused", open, `{"type":"commit","contextId":"earth","value":"Blue\u202eMarble"}`, "body=earth name=Blue+Marble edit=1 draft=Blue%E2%80%AEMarble problem=format try=1"},
		{"Escape closes and moves the focus", refused, `{"type":"cancel","contextId":"earth","key":"Escape"}`, closed + " focus=1"},
		{"a blur with the name unchanged closes", open, `{"type":"cancel","contextId":"earth"}`, closed},
		{"a commit on a closed field", closed, `{"type":"commit","contextId":"earth","value":"Pale Blue"}`, ""},
		{"a cancel on a closed field", closed, `{"type":"cancel","contextId":"earth"}`, ""},
		{"another body's event", open, `{"type":"commit","contextId":"mars","value":"Rust"}`, ""},
		{"an unknown event", open, `{"type":"rename","contextId":"earth"}`, ""},
		{"no event", open, `[]`, ""},
		{"an unknown body", "body=vulcan name=Home", `{"type":"request","contextId":"vulcan"}`, ""},
		{"no body", "name=Home", `{"type":"request","contextId":""}`, ""},
		{"an unknown word", closed + " colour=blue", `{"type":"request","contextId":"earth"}`, ""},
		{"a bad escape", "body=earth name=Blue%ZZ", `{"type":"request","contextId":"earth"}`, ""},
		{"an empty name", "body=earth name=", `{"type":"request","contextId":"earth"}`, ""},
		{"a name with spaces around it", "body=earth name=+Blue", `{"type":"request","contextId":"earth"}`, ""},
		{"a name with a format character", "body=earth name=Blue%E2%80%8B", `{"type":"request","contextId":"earth"}`, ""},
		{"an unknown problem", open + " problem=rude try=1", `{"type":"cancel","contextId":"earth"}`, ""},
		{"a draft over 80", "body=earth name=Blue+Marble edit=1 draft=" + strings.Repeat("z", 81), `{"type":"cancel","contextId":"earth"}`, ""},
		{"a try out of range", open + " problem=empty try=1000", `{"type":"cancel","contextId":"earth"}`, ""},
		{"a try that is no number", open + " problem=empty try=x", `{"type":"cancel","contextId":"earth"}`, ""},
	} {
		got, err := arrangeInlineEdit(tc.state, json.RawMessage(tc.move))
		if tc.want == "" {
			if err == nil {
				t.Errorf("%s: %q → %q, want a refusal", tc.name, tc.state, got)
			}
		} else if err != nil || got != tc.want {
			t.Errorf("%s: %q → %q %v, want %q", tc.name, tc.state, got, err, tc.want)
		}
	}
}

func TestNicknameProblem(t *testing.T) {
	for s, want := range map[string]string{
		"Blue Marble":           "",
		"👩\u200d🚀 Crew":         "", // a joiner inside an emoji sequence
		"Ruz\u200cbeh":          "", // and a non-joiner in a word
		"":                      "empty",
		"\u200b":                "format",
		"\u200d\u200c":          "empty",
		"\u00a0":                "empty",
		strings.Repeat("a", 40): "",
		strings.Repeat("a", 41): "long",
		"tab\there":             "control",
		"\u202eevil":            "format",
		"soft\u00adhyphen":      "format",
	} {
		if got := nicknameProblem(s); got != want {
			t.Errorf("nicknameProblem(%q) = %q, want %q", s, got, want)
		}
	}
}

// A state survives its own round trip, whatever the texts hold, and the
// markup escapes them.
func TestNicknameStateRoundTrip(t *testing.T) {
	for _, n := range []nickname{
		{body: "earth", name: `<b>"Home" & co</b>`},
		{body: "earth", name: "50% + 1 = more", focus: true},
		{body: "mars", name: "Red", editing: true, draft: `a=b c+d "e" <f>`, problem: "long", try: 3, focus: true},
	} {
		got, err := parseNickname(n.String())
		if err != nil || got != n {
			t.Errorf("%+v → %q → %+v %v", n, n.String(), got, err)
		}
		markup := renderInlineEdit("nick", n.String())
		if strings.Contains(markup, "<b>") || strings.Contains(markup, "<f>") || strings.Contains(markup, `"Home"`) {
			t.Errorf("%+v: the markup doesn't escape the texts:\n%s", n, markup)
		}
	}
}

// A refusal keeps the field (one id for the whole edit, the typed value
// preserved) and renders a new alert. Only a fresh field, a refusal of
// Enter and a field a key closed carry a data-init that moves the focus:
// a morph of the host re-runs every data-init in it.
func TestRenderInlineEditFocus(t *testing.T) {
	for _, tc := range []struct {
		state      string
		has, hasnt []string
	}{
		{"body=earth name=Blue+Marble", []string{`<button type="button" data-inline-edit-trigger aria-label="Blue Marble, nickname of Earth: press Enter to rename">`}, []string{"data-init"}},
		{"body=earth name=Blue+Marble focus=1", []string{`aria-label="Blue Marble, nickname of Earth: press Enter to rename" data-init="document.activeElement === document.body && el.focus()"`}, nil},
		{"body=earth name=Blue+Marble edit=1 draft=Blue+Marble", []string{`<input id="nick-earth-field" data-inline-edit-input value="Blue Marble"`, `data-preserve-attr="value"`, `<span data-inline-edit-value hidden>Blue Marble</span>`, `evt.key" data-init="document.activeElement === document.body && el.focus()">`}, []string{"aria-invalid", `role="alert">A`}},
		{"body=earth name=Blue+Marble edit=1 draft= problem=empty try=2 focus=1", []string{`<input id="nick-earth-field"`, `aria-describedby="nick-earth-problem-2"`, `<span id="nick-earth-problem-2" class="demo-edit__problem" role="alert" data-init="document.activeElement === document.body && document.getElementById('nick-earth-field').focus()">`}, []string{`evt.key" data-init`}},
		{"body=earth name=Blue+Marble edit=1 draft= problem=empty try=3", []string{`<span id="nick-earth-problem-3" class="demo-edit__problem" role="alert">`}, []string{"data-init"}},
	} {
		markup := renderInlineEdit("nick-earth", tc.state)
		for _, s := range tc.has {
			if !strings.Contains(markup, s) {
				t.Errorf("%s: no %s in\n%s", tc.state, s, markup)
			}
		}
		for _, s := range tc.hasnt {
			if strings.Contains(markup, s) {
				t.Errorf("%s: %s in\n%s", tc.state, s, markup)
			}
		}
	}
}
