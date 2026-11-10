// Package tsgen compiles the components written in TypeScript. The source is
// components/<slug>/<slug>.ts, with any helper .ts modules and .d.ts
// declarations in its folder, type-checked strictly against the patched
// Datastar build. Each .ts gets the .js next to it that the compiler emits,
// committed and served like any other module of the component.
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

// Result is a TypeScript module that has errors, or whose .js is (or, with
// write, was) stale.
type Result struct {
	Source      string // the .ts, relative to the root
	Stale       bool
	Diagnostics []tscheck.Diagnostic // File is relative to components/
}

// Refresh compiles the TypeScript of every component under root in one
// program and, with write, rewrites the .js files that differ. Nothing is
// written while any module has errors.
func Refresh(ctx context.Context, c *tscheck.Checker, root string, write bool) ([]Result, error) {
	dirs, err := filepath.Glob(filepath.Join(root, "components", "*"))
	if err != nil {
		return nil, err
	}
	var files []tscheck.File
	var sources []string // relative to components/
	for _, dir := range dirs {
		var mods []tscheck.File
		ts := map[string]bool{}
		err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return err
			}
			rel, _ := filepath.Rel(filepath.Join(root, "components"), p)
			rel = filepath.ToSlash(rel)
			if !strings.HasSuffix(rel, ".ts") && !strings.HasSuffix(rel, ".js") && !strings.HasSuffix(rel, ".mjs") {
				return nil
			}
			b, err := os.ReadFile(p)
			mods = append(mods, tscheck.File{Name: rel, Source: string(b)})
			if strings.HasSuffix(rel, ".ts") && !strings.HasSuffix(rel, ".d.ts") {
				ts[strings.TrimSuffix(rel, ".ts")] = true
			}
			return err
		})
		if err != nil {
			return nil, err
		}
		if len(ts) == 0 {
			continue // a JavaScript component
		}
		// The .ts files, and the modules they may import: not the .js they become.
		for _, f := range mods {
			if !strings.HasSuffix(f.Name, ".js") || !ts[strings.TrimSuffix(f.Name, ".js")] {
				files = append(files, f)
			}
			if strings.HasSuffix(f.Name, ".ts") && !strings.HasSuffix(f.Name, ".d.ts") {
				sources = append(sources, f.Name)
			}
		}
	}
	if len(sources) == 0 {
		return nil, nil
	}
	js, ds, err := c.Compile(ctx, files)
	if err != nil {
		return nil, err
	}
	var out []Result
	if len(ds) > 0 {
		by := map[string][]tscheck.Diagnostic{}
		for _, d := range ds {
			by[d.File] = append(by[d.File], d)
		}
		for src, ds := range by {
			out = append(out, Result{Source: "components/" + src, Diagnostics: ds})
		}
		sort.Slice(out, func(i, j int) bool { return out[i].Source < out[j].Source })
		return out, nil
	}
	sort.Strings(sources)
	for _, src := range sources {
		name := strings.TrimSuffix(src, ".ts") + ".js"
		want := fmt.Sprintf(Header, path.Base(src)) + js[name]
		file := filepath.Join(root, "components", filepath.FromSlash(name))
		if got, _ := os.ReadFile(file); string(got) == want {
			continue
		}
		out = append(out, Result{Source: "components/" + src, Stale: true})
		if write {
			if err := os.WriteFile(file, []byte(want), 0o644); err != nil {
				return nil, err
			}
		}
	}
	return out, nil
}
