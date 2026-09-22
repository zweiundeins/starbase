package catalog_test

import (
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
