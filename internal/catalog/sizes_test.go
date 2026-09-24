package catalog_test

import (
	"strings"
	"testing"

	"starbase/components"
	"starbase/internal/catalog"
)

func TestSizes(t *testing.T) {
	cat, err := catalog.Load(components.FS)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range cat.Components {
		s := c.Sizes
		if len(s.Files) == 0 || s.Files[0].Name != c.Slug+".js" {
			t.Errorf("%s: the module comes first: %+v", c.Slug, s.Files)
		}
		for _, f := range append(s.Files, s.Uses...) {
			if f.Raw == 0 || f.Brotli == 0 || f.Brotli > f.Raw || f.Gzip > f.Raw {
				t.Errorf("%s: %s: %+v", c.Slug, f.Name, f.Size)
			}
		}
	}
	// Vendored files are the component's own; rendered components are listed
	// separately, with everything they download.
	editor, _ := cat.Get("code-editor")
	if len(editor.Sizes.Files) != 2 || editor.Sizes.Files[1].Name != "vendor/prism.js" {
		t.Errorf("code-editor files: %+v", editor.Sizes.Files)
	}
	pg, _ := cat.Get("code-playground")
	if len(pg.Sizes.Uses) != 1 || pg.Sizes.Uses[0].Name != "sb-code-editor" || pg.Sizes.Uses[0].Size != editor.Sizes.Own {
		t.Errorf("code-playground uses: %+v", pg.Sizes.Uses)
	}
	if pg.Sizes.Total != pg.Sizes.Own.Add(editor.Sizes.Own) {
		t.Errorf("total: %+v", pg.Sizes.Total)
	}
}

// The single-file bundle (loading experiment): every component in one module,
// 'datastar' left to the import map, and lazily loaded libraries left lazy.
func TestBundle(t *testing.T) {
	cat, err := catalog.Load(components.FS)
	if err != nil {
		t.Fatal(err)
	}
	b, err := cat.Bundle()
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	for _, c := range cat.Components {
		if !strings.Contains(s, c.Tag) {
			t.Errorf("the bundle misses %s", c.Tag)
		}
	}
	if !strings.Contains(s, `from"datastar"`) {
		t.Error("datastar must stay external")
	}
	if strings.Contains(s, "Apache ECharts") || len(b) > 600_000 {
		t.Errorf("a lazily imported library was inlined (%d bytes)", len(b))
	}
}

// A tag a comment mentions is not a dependency: gauge, sparkline and
// echarts explain in comments that they repaint on <sb-theme-switch>'s
// events, and must not pull it in. Real uses still count.
func TestUsesIgnoresComments(t *testing.T) {
	cat, err := catalog.Load(components.FS)
	if err != nil {
		t.Fatal(err)
	}
	for _, slug := range []string{"gauge", "sparkline", "echarts"} {
		c, _ := cat.Get(slug)
		for _, u := range cat.Uses(c) {
			t.Errorf("%s: %s is only mentioned in a comment", slug, u.Tag)
		}
	}
	pg, _ := cat.Get("code-playground")
	if uses := cat.Uses(pg); len(uses) != 1 || uses[0].Tag != "sb-code-editor" {
		t.Errorf("code-playground renders sb-code-editor, got %v", uses)
	}
}
