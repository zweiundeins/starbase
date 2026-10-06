// Package tsgen compiles the components written in TypeScript. The source is
// components/<slug>/<slug>.ts, type-checked strictly against the patched
// Datastar build, and the <slug>.js next to it is what the compiler emits
// for it, committed and served like any other component's module.
// `go tool task ts` rewrites the .js files; a test fails when one is stale.
package tsgen

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"starbase/internal/tscheck"
)

// Header starts every generated file.
const Header = "// Generated from %s by `go tool task ts`: edit the TypeScript, not this file.\n"

// Result is a TypeScript component that has errors, or whose .js is (or,
// with write, was) stale.
type Result struct {
	Source      string // the .ts, relative to the root
	Stale       bool
	Diagnostics []tscheck.Diagnostic // File is relative to components/
}

// Sources returns the TypeScript components under root, as
// components/<slug>/<slug>.ts.
func Sources(root string) ([]string, error) {
	all, err := filepath.Glob(filepath.Join(root, "components", "*", "*.ts"))
	var out []string
	for _, f := range all {
		slug := filepath.Base(filepath.Dir(f))
		if filepath.Base(f) == slug+".ts" {
			rel, _ := filepath.Rel(root, f)
			out = append(out, filepath.ToSlash(rel))
		}
	}
	return out, err
}

// Refresh compiles every TypeScript component under root in one program and,
// with write, rewrites the .js files that differ. Nothing is written while
// any component has errors.
func Refresh(ctx context.Context, c *tscheck.Checker, root string, write bool) ([]Result, error) {
	sources, err := Sources(root)
	if err != nil || len(sources) == 0 {
		return nil, err
	}
	var files []tscheck.File
	for _, src := range sources {
		slug := path.Base(path.Dir(src))
		dir := filepath.Join(root, "components", slug)
		// The component and the modules it may import (vendored files), not the .js it becomes.
		err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
			rel, _ := filepath.Rel(dir, p)
			rel = filepath.ToSlash(rel)
			if err != nil || d.IsDir() || rel == slug+".js" || !(rel == slug+".ts" || strings.HasSuffix(rel, ".js") || strings.HasSuffix(rel, ".mjs")) {
				return err
			}
			b, err := os.ReadFile(p)
			files = append(files, tscheck.File{Name: slug + "/" + rel, Source: string(b)})
			return err
		})
		if err != nil {
			return nil, err
		}
	}
	js, ds, err := c.Compile(ctx, files)
	if err != nil {
		return nil, err
	}
	var out []Result
	if len(ds) > 0 {
		by := map[string][]tscheck.Diagnostic{}
		for _, d := range ds {
			src := "components/" + path.Dir(d.File) + "/" + path.Base(path.Dir(d.File)) + ".ts"
			by[src] = append(by[src], d)
		}
		for src, ds := range by {
			out = append(out, Result{Source: src, Diagnostics: ds})
		}
		sort.Slice(out, func(i, j int) bool { return out[i].Source < out[j].Source })
		return out, nil
	}
	for _, src := range sources {
		slug := path.Base(path.Dir(src))
		want := fmt.Sprintf(Header, slug+".ts") + js[slug+"/"+slug+".js"]
		file := filepath.Join(root, "components", slug, slug+".js")
		if got, _ := os.ReadFile(file); string(got) == want {
			continue
		}
		out = append(out, Result{Source: src, Stale: true})
		if write {
			if err := os.WriteFile(file, []byte(want), 0o644); err != nil {
				return nil, err
			}
		}
	}
	return out, nil
}
