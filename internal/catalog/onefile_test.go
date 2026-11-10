package catalog_test

import (
	"maps"
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	"starbase/components"
	"starbase/internal/catalog"
)

// Every component's one-file bundle: the banner first, its own modules
// inlined, and nothing imported but 'datastar' and, lazily, the .min files
// next to it.
func TestComponentBundles(t *testing.T) {
	cat, err := catalog.Load(components.FS)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range cat.Components {
		b, err := cat.BundleOf(c)
		if err != nil {
			t.Fatal(err)
		}
		banner := "/*! " + c.Tag + ", version " + c.Slug + "@" + c.Hash + ": its modules in one file */\n"
		if !strings.HasPrefix(string(b.Body), banner) {
			t.Errorf("%s: the bundle should start with %q: %.120s", c.Slug, banner, b.Body)
		}
		mins, err := cat.MinFiles(c)
		if err != nil {
			t.Fatal(err)
		}
		if string(mins[catalog.BundleName(c.Slug)]) != string(b.Body) {
			t.Errorf("%s: MinFiles (stored and served per version) lacks the bundle", c.Slug)
		}
		files, _ := cat.ModuleFiles(c)
		for _, imp := range b.Imports {
			name, relative := strings.CutPrefix(imp.Path, "./")
			switch {
			case imp.Path == "datastar" && !imp.Lazy:
			case imp.Lazy && relative && catalog.IsMinPath(name) && (mins[name] != nil || files[name] != nil):
			default:
				t.Errorf("%s: the bundle imports %+v: only 'datastar', and lazily a .min file next to it", c.Slug, imp)
			}
		}
		// What the code says, too: no relative import but the lazy ones.
		for _, spec := range catalog.Imports(string(b.Body)) {
			if (strings.HasPrefix(spec, "./") || strings.HasPrefix(spec, "../")) && !slices.Contains(b.Imports, catalog.BundleImport{Path: spec, Lazy: true}) {
				t.Errorf("%s: the bundle still imports %q", c.Slug, spec)
			}
		}
		// Every module a page loads is inlined or loaded lazily, never both
		// (a module in both would run twice, with two copies of its state).
		for name := range cat.Loaded(c, files) {
			inlined, lazy := slices.Contains(b.Modules, name), slices.Contains(b.Lazy, catalog.MinOf(name))
			if inlined == lazy {
				t.Errorf("%s: %s inlined %v, loaded lazily %v", c.Slug, name, inlined, lazy)
			}
		}
		if c.License == "Beerware" && !strings.Contains(string(b.Body), "THE BEER-WARE LICENSE") {
			t.Errorf("%s: the bundle lost its licence notice", c.Slug)
		}
	}
	kanban, _ := cat.Get("kanban-board")
	if b, _ := cat.BundleOf(kanban); len(b.Modules) != len(kanban.Sizes.Files) || len(b.Modules) < 10 {
		t.Errorf("kanban-board inlines %v, loads %d files", b.Modules, len(kanban.Sizes.Files))
	}
	editor, _ := cat.Get("code-editor")
	if b, _ := cat.BundleOf(editor); !slices.Equal(b.Lazy, []string{"vendor/prism.min.js"}) || strings.Contains(string(b.Body), "Prism.languages") {
		t.Errorf("code-editor loads Prism lazily, got %v", b.Lazy)
	}
}

// A file can't take the bundle's name, and the bundle takes only files
// from the component's own folder.
func TestBundleRules(t *testing.T) {
	for _, tc := range []struct {
		files fstest.MapFS
		want  string
	}{
		{fstest.MapFS{"widget/widget.bundle.js": {Data: []byte("export {}")}}, "widget.bundle.js is reserved"},
		{fstest.MapFS{"widget/widget.bundle.min.js": {Data: []byte("export {}")}}, "widget.bundle.min.js is reserved"},
		{fstest.MapFS{
			"widget/widget.js": {Data: []byte("import './core/a.js'\nrocket('sb-widget', {})")},
			"widget/core/a.js": {Data: []byte("import '../../_shared/b.js'")},
			"_shared/b.js":     {Data: []byte("export {}")},
		}, "outside the component's folder"},
		{fstest.MapFS{
			"widget/widget.js": {Data: []byte("import './core/a.js'\nrocket('sb-widget', {})")},
			"widget/core/a.js": {Data: []byte("import 'lit'")},
		}, `imports "lit": only 'datastar'`},
	} {
		fsys := fstest.MapFS{"widget/README.md": {Data: []byte(validReadme)}, "widget/widget.js": {Data: []byte("rocket('sb-widget', {})")}}
		for p, f := range tc.files {
			fsys[p] = f
		}
		if _, err := catalog.Load(fsys); err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%v: err = %v, want %q", slices.Sorted(maps.Keys(tc.files)), err, tc.want)
		}
	}
}
