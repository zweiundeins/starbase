package commands_test

import (
	"strings"
	"testing"

	"starbase/internal/commands"
)

func TestSaveSnippetValidate(t *testing.T) {
	ok := commands.SaveSnippet{SID: "s", TabID: "tab12345", ID: "AbCd2345", Files: map[string]string{"component.js": "x", "index.html": "y"}}
	if err := ok.Validate(); err != nil {
		t.Fatal(err)
	}
	for name, c := range map[string]commands.SaveSnippet{
		"bad id":   {SID: "s", TabID: "tab12345", ID: "x", Files: ok.Files},
		"no files": {SID: "s", TabID: "tab12345", ID: "AbCd2345"},
		"bad name": {SID: "s", TabID: "tab12345", ID: "AbCd2345", Files: map[string]string{"../etc/passwd": "x"}},
		"too big":  {SID: "s", TabID: "tab12345", ID: "AbCd2345", Files: map[string]string{"a.js": strings.Repeat("x", 65<<10)}},
		"bad tab":  {SID: "s", TabID: "?", ID: "AbCd2345", Files: ok.Files},
	} {
		if c.Validate() == nil {
			t.Errorf("%s: expected a validation error", name)
		}
	}
}
