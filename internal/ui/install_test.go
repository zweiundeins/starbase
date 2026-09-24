package ui

import "testing"

// TestTabSlot pins the Go twin of sb-tabs' slug() (components/tabs/tabs.js)
// to what the component gives these labels in the browser.
func TestTabSlot(t *testing.T) {
	for label, want := range map[string]string{
		"Self-host":      "self-host",
		"This component": "this-component",
		"Übersicht":      "übersicht",
		"日本語":            "日本語",
		"Русский":        "русский",
		"C++":            "c",
		" v2 (beta) ":    "v2-beta",
		"★":              "",
	} {
		if got := tabSlot(label); got != want {
			t.Errorf("tabSlot(%q) = %q, want %q", label, got, want)
		}
	}
}
