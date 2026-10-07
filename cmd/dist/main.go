// Command dist writes components as one file each, for a site that vendors
// them from a checkout of Starbase:
//
//	go run ./cmd/dist -out <dir> [slug ...]
//
// <dir>/<slug>.js is the one-file bundle the site serves for this checkout's
// version, /c/<slug>@<hash>/<slug>.bundle.min.js, byte for byte (its banner
// names that version). A file the bundle loads on first use goes next to it,
// in its folder (vendor/prism.min.js for code-editor). The components a
// component renders are written too, since a page needs them. Without slugs,
// every component.
package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"maps"
	"os"
	"path/filepath"
	"slices"

	"starbase/components"
	"starbase/internal/catalog"
)

func main() {
	out := flag.String("out", "", "the directory to write into")
	flag.Usage = func() {
		fmt.Fprintln(flag.CommandLine.Output(), "usage: go run ./cmd/dist -out <dir> [slug ...]")
		flag.PrintDefaults()
	}
	flag.Parse()
	if *out == "" {
		flag.Usage()
		os.Exit(2)
	}
	if err := run(*out, flag.Args(), os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "dist:", err)
		os.Exit(1)
	}
}

// run writes the bundles of slugs (all components when there are none) and
// what they need into out, and lists each file with its integrity on w.
func run(out string, slugs []string, w io.Writer) error {
	cat, err := catalog.Load(components.FS)
	if err != nil {
		return err
	}
	want := cat.Components
	if len(slugs) > 0 {
		want = nil
		for _, s := range slugs {
			c, ok := cat.Get(s)
			if !ok {
				return fmt.Errorf("no component %q in components/", s)
			}
			for _, d := range append([]*catalog.Component{c}, cat.Deps(c)...) {
				if !slices.Contains(want, d) {
					want = append(want, d)
				}
			}
		}
	}
	files := map[string][]byte{}
	put := func(name string, b []byte) error {
		if prev, ok := files[name]; ok && !bytes.Equal(prev, b) {
			return fmt.Errorf("two components need a different %s: write them to separate directories", name)
		}
		files[name] = b
		return nil
	}
	for _, c := range want {
		bundle, err := cat.BundleOf(c)
		if err != nil {
			return err
		}
		mins, err := cat.MinFiles(c)
		if err != nil {
			return err
		}
		modules, err := cat.ModuleFiles(c)
		if err != nil {
			return err
		}
		if err := put(c.Slug+".js", bundle.Body); err != nil {
			return err
		}
		for _, p := range bundle.Lazy {
			b, ok := mins[p]
			if !ok {
				b = modules[p] // already minified, as vendored
			}
			if err := put(p, b); err != nil {
				return err
			}
		}
	}
	for _, name := range slices.Sorted(maps.Keys(files)) {
		p := filepath.Join(out, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(p, files[name], 0o644); err != nil {
			return err
		}
		fmt.Fprintf(w, "%s\t%d bytes\t%s\n", filepath.ToSlash(p), len(files[name]), catalog.SRI(files[name]))
	}
	return nil
}
