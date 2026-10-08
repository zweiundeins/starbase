package catalog

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/evanw/esbuild/pkg/api"

	"starbase/internal/precompress"
)

// Measured is what code written in the playground would weigh as a
// component, measured like the catalog's bundles (Sizes): Raw is the
// source, Min its one-file bundle (the code with every file of c it imports
// inlined, minified) and MinBrotli that under brotli -11. Extra is what a
// page downloads besides, each file as its Size.Min: the files of c it loads
// on first use (Lazy of it), and the bundles of the components it renders,
// with theirs and their first-use files.
type Measured struct {
	Raw, Min, MinBrotli int
	Extra, Lazy         int
	Uses                []string // tags of the components it renders, transitively
}

// Total is MinBrotli plus Extra: what a page using this code downloads.
func (m Measured) Total() int { return m.MinBrotli + m.Extra }

var definesRe = regexp.MustCompile(`rocket\(\s*['"](sb-[a-z0-9-]+)['"]`)

// Measure bundles and compresses code afresh on every call (nothing is
// kept: the playground measures every edit). name is component.js, or
// component.ts for TypeScript (measured as the JavaScript it compiles to). c
// is the component whose folder relative imports resolve against, or nil.
// A syntax error esbuild can't bundle past is returned as the error.
func (cat *Catalog) Measure(name string, code []byte, c *Component) (Measured, error) {
	js := code
	if strings.HasSuffix(name, ".ts") { // shrink reads JavaScript
		t, err := Transpile(string(code))
		if err != nil {
			return Measured{}, fmt.Errorf("esbuild: %w", err)
		}
		name, js = strings.TrimSuffix(name, ".ts")+".js", []byte(t.Code)
	}
	opts := bundleOptions(c)
	opts.Stdin = &api.StdinOptions{Contents: string(shrink(name, js)), Sourcefile: name, Loader: api.LoaderJS}
	opts.Outfile = "measure.js"
	opts.Metafile = true
	opts.Plugins = []api.Plugin{cat.folderPlugin(c, true)}
	res := api.Build(opts)
	if len(res.Errors) > 0 {
		return Measured{}, fmt.Errorf("esbuild: %s", res.Errors[0].Text)
	}
	b := res.OutputFiles[0].Contents
	out := Measured{Raw: len(code), Min: len(b), MinBrotli: precompress.BrotliLen(b)}

	// The files of c it loads on first use, and what those import.
	if c != nil {
		var meta struct {
			Outputs map[string]struct {
				Imports []struct{ Path, Kind string } `json:"imports"`
			} `json:"outputs"`
		}
		json.Unmarshal([]byte(res.Metafile), &meta)
		files, _ := cat.ModuleFiles(c)
		lazy := map[string]bool{}
		for _, imp := range meta.Outputs["measure.js"].Imports {
			for name := range files {
				if imp.Kind == "dynamic-import" && "./"+MinOf(name) == imp.Path {
					for n := range reach(files, name) {
						lazy[MinOf(n)] = true
					}
				}
			}
		}
		for _, f := range c.Sizes.Lazy {
			if lazy[f.Name] {
				out.Lazy += f.Min
			}
		}
		out.Extra += out.Lazy
	}

	// Components it renders, except the ones it defines itself. The source
	// tells which those are: bundling renames the rocket import.
	own := map[string]bool{}
	for _, d := range definesRe.FindAllStringSubmatch(string(code), -1) {
		own[d[1]] = true
	}
	seen := map[*Component]bool{}
	for _, u := range cat.rendered(b, own) {
		for _, d := range append([]*Component{u}, cat.Deps(u)...) {
			if !seen[d] && !own[d.Tag] {
				seen[d] = true
				out.Uses = append(out.Uses, d.Tag)
				out.Extra += d.Sizes.Bundle.Min
				for _, f := range d.Sizes.Lazy {
					out.Extra += f.Min
				}
			}
		}
	}
	return out, nil
}
