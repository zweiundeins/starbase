package main

import (
	"bytes"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"starbase/components"
	"starbase/internal/catalog"
)

// dist writes what the site serves for this version: each bundle, the
// components it renders, and the files a bundle loads lazily.
func TestDist(t *testing.T) {
	out := t.TempDir()
	var log bytes.Buffer
	if err := run(out, []string{"kanban-board", "code-playground"}, &log); err != nil {
		t.Fatal(err)
	}
	var got []string
	filepath.WalkDir(out, func(p string, d os.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			rel, _ := filepath.Rel(out, p)
			got = append(got, filepath.ToSlash(rel))
		}
		return err
	})
	slices.Sort(got)
	if want := []string{"code-editor.js", "code-playground.js", "kanban-board.js", "vendor/prism.min.js"}; !slices.Equal(got, want) {
		t.Fatalf("wrote %v, want %v", got, want)
	}
	cat, err := catalog.Load(components.FS)
	if err != nil {
		t.Fatal(err)
	}
	for _, slug := range []string{"kanban-board", "code-playground", "code-editor"} {
		c, _ := cat.Get(slug)
		mins, _ := cat.MinFiles(c) // what SyncCatalog stores and the site serves
		b, _ := os.ReadFile(filepath.Join(out, slug+".js"))
		if !bytes.Equal(b, mins[catalog.BundleName(slug)]) {
			t.Errorf("%s.js differs from %s", slug, c.VersionedBundle())
		}
		if !strings.Contains(log.String(), slug+".js\t") || !strings.Contains(log.String(), catalog.SRI(b)) {
			t.Errorf("the log should list %s.js with its integrity:\n%s", slug, log.String())
		}
	}
	if err := run(t.TempDir(), []string{"nope"}, &log); err == nil {
		t.Error("an unknown slug should fail")
	}
}
