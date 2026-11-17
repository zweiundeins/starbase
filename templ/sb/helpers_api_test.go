package sb

import (
	"reflect"
	"regexp"
	"testing"

	"github.com/a-h/templ"
)

func TestMerge(t *testing.T) {
	base := templ.Attributes{"class": "a", "data-on:sb-change": "@post('/a')", "Hidden": true}
	got := Merge(base, nil, templ.Attributes{"class": "b", "hidden": false, "aria-label": "x"})
	want := templ.Attributes{"class": "b", "data-on:sb-change": "@post('/a')", "hidden": false, "aria-label": "x"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
	if base["class"] != "a" || len(base) != 3 {
		t.Errorf("Merge changed its argument: %v", base)
	}
	if got := Merge(); got == nil || len(got) != 0 {
		t.Errorf("Merge() = %#v", got)
	}
	out := render(t, Toggle(ToggleProps{Label: "L", Attrs: Merge(templ.Attributes{"LABEL": "x"}, templ.Attributes{"label": "y"})}))
	if out != `<sb-toggle label="y"></sb-toggle>` {
		t.Errorf("rendered %s", out)
	}
}

// TestTabSlot holds TabSlot to sb-tabs' slug() worked out by hand; the
// browser test TestTemplTabSlots checks it against the component itself.
func TestTabSlot(t *testing.T) {
	for _, tc := range []struct {
		labels []string
		want   []string
	}{
		{[]string{"Home", "Docs", "API"}, []string{"home", "docs", "api"}},
		{[]string{"C++", "C#", "C"}, []string{"c", "c-1", "c-2"}},
		{[]string{"a", "a", "a-1"}, []string{"a", "a-1", "a-1-2"}},
		{[]string{"  Hello, World!  ", "", "--", "Übersicht", "日本語 2", "x²"}, []string{"hello-world", "", "-2", "übersicht", "日本語-2", "x²"}},
		{[]string{"İstanbul", "ΟΔΟΣ", "ΣΑ", "ΟΔΟΣ.", "Σ"}, []string{"i-stanbul", "οδος", "σα", "οδος-3", "σ"}},
	} {
		for i, w := range tc.want {
			if got := TabSlot(tc.labels, i); got != w {
				t.Errorf("TabSlot(%q, %d) = %q, want %q", tc.labels, i, got, w)
			}
		}
	}
	if TabSlot([]string{"a"}, 1) != "" || TabSlot(nil, 0) != "" || TabSlot([]string{"a"}, -1) != "" {
		t.Error("an index outside the labels")
	}
}

// TestTabSlug: one label's slug, without the de-duplication; the browser
// test TestTemplTabSlots checks it against the component itself.
func TestTabSlug(t *testing.T) {
	for in, want := range map[string]string{"Home": "home", "C#": "c", "  Hello, World!  ": "hello-world", "--": "", "": "", "Übersicht": "übersicht", "İstanbul": "i-stanbul", "ΟΔΟΣ.": "οδος", "日本語 2": "日本語-2", "a-1": "a-1"} {
		if got := TabSlug(in); got != want {
			t.Errorf("TabSlug(%q) = %q, want %q", in, got, want)
		}
	}
	labels := []string{"Docs", "Docs"}
	if TabSlug(labels[1]) != "docs" || TabSlot(labels, 1) != "docs-1" {
		t.Error("TabSlug does not de-duplicate, TabSlot does")
	}
}

// TestRelease: the release the wrappers were generated with, as the
// changelog writes it (no "v").
func TestRelease(t *testing.T) {
	if !regexp.MustCompile(`^\d+\.\d+\.\d+$`).MatchString(Release) {
		t.Errorf("Release = %q", Release)
	}
}
