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
		// The download size is the bundles: its own, then those of the
		// components it renders, each minified already (Min is Brotli), and
		// what they load on first use.
		var total catalog.Size
		for _, b := range s.Bundles {
			total = total.Add(b.Size)
			if b.Min != b.Brotli || b.Min == 0 || b.Lazy {
				t.Errorf("%s: bundle %+v", c.Slug, b)
			}
		}
		for _, f := range append(s.Lazy, s.UsesLazy...) {
			total = total.Add(f.Size)
		}
		if s.ModuleFiles != len(s.Files)+func() (n int) {
			for _, d := range cat.Deps(c) {
				n += len(d.Sizes.Files)
			}
			return n
		}() {
			t.Errorf("%s: %d module files", c.Slug, s.ModuleFiles)
		}
		if s.Bundle.Name != catalog.BundleName(c.Slug) || len(s.Bundles) != 1+len(cat.Deps(c)) || s.Bundles[0] != s.Bundle || s.Total != total {
			t.Errorf("%s: bundles %+v, total %+v", c.Slug, s.Bundles, s.Total)
		}
	}
	// Vendored files are the component's own; a file loaded on first use is
	// marked, and counts; rendered components are listed separately, with
	// everything they download.
	editor, _ := cat.Get("code-editor")
	if f := editor.Sizes.Files; len(f) != 2 || f[1].Name != "vendor/prism.js" || !f[1].Lazy || f[0].Lazy || editor.Sizes.Own != f[0].Add(f[1].Size) || editor.Sizes.ModuleFiles != 2 {
		t.Errorf("code-editor files: %+v, own %+v", f, editor.Sizes.Own)
	}
	if l := editor.Sizes.Lazy; len(l) != 1 || l[0].Name != "vendor/prism.min.js" || l[0].Min != editor.Sizes.Files[1].Min || editor.Sizes.Total != editor.Sizes.Bundle.Add(l[0].Size) || editor.Sizes.LazyMin() != l[0].Min {
		t.Errorf("code-editor's Prism, on first use: %+v, total %+v", l, editor.Sizes.Total)
	}
	pg, _ := cat.Get("code-playground")
	if pg.Sizes.Total != pg.Sizes.Bundle.Add(editor.Sizes.Total) || len(pg.Sizes.UsesLazy) != 1 || pg.Sizes.UsesLazy[0].Name != "code-editor/vendor/prism.min.js" {
		t.Errorf("code-playground's bundles: %+v, %+v", pg.Sizes.Bundles, pg.Sizes.UsesLazy)
	}
	if len(pg.Sizes.Uses) != 1 || pg.Sizes.Uses[0].Name != "sb-code-editor" || pg.Sizes.Uses[0].Size != editor.Sizes.Own || pg.Sizes.ModulesTotal != pg.Sizes.Own.Add(editor.Sizes.Own) {
		t.Errorf("code-playground uses: %+v", pg.Sizes.Uses)
	}
	// One request where the modules make one per file, and fewer bytes,
	// since brotli sees them together.
	kanban, _ := cat.Get("kanban-board")
	if s := kanban.Sizes; s.ModuleFiles != len(s.Files) || s.ModuleFiles < 10 || s.Total.Min >= s.ModulesTotal.Min {
		t.Errorf("kanban-board: %d module files, %+v as one file, %+v as modules", s.ModuleFiles, s.Total, s.ModulesTotal)
	}
	echarts, _ := cat.Get("echarts")
	if s := echarts.Sizes; len(s.Lazy) != 1 || s.Lazy[0].Name != "vendor/echarts.esm.min.js" || s.LazyMin() < 200_000 || s.Total.Min != s.Bundle.Min+s.LazyMin() || s.ModulesTotal.Min < s.LazyMin() {
		t.Errorf("echarts loads its library on first use: %+v, %+v", s.Total, s.Lazy)
	}
	// A file only the docs' examples import is not part of the download.
	auto, _ := cat.Get("autoloader")
	for _, f := range auto.Sizes.Files {
		if f.Name == "demo-badge.js" {
			t.Errorf("autoloader counts its example-only module: %+v", auto.Sizes.Files)
		}
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
