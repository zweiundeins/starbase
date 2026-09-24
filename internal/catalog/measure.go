package catalog

import (
	"path"
	"regexp"
	"strings"

	"starbase/internal/precompress"
)

// Measured is what code written in the playground would weigh as a
// component module, measured like the catalog's own files (Sizes): Raw is
// the source, Min the minified module and MinBrotli that under brotli -11
// (what a page downloads). Extra is what it pulls in that the catalog has
// already measured: the files of c it imports and the components it renders
// (with theirs), each as its Size.Min.
type Measured struct {
	Raw, Min, MinBrotli int
	Extra               int
	Uses                []string // tags of the components it renders, transitively
}

// Total is MinBrotli plus Extra: what a page using this code downloads.
func (m Measured) Total() int { return m.MinBrotli + m.Extra }

var definesRe = regexp.MustCompile(`rocket\(\s*['"](sb-[a-z0-9-]+)['"]`)

// Measure minifies and compresses code afresh on every call (nothing is
// kept: the playground measures every edit). c is the component whose
// folder relative imports resolve against, or nil. A syntax error esbuild
// can't minify past is returned as the error.
func (cat *Catalog) Measure(code []byte, c *Component) (Measured, error) {
	m, err := minify("component.js", code)
	if err != nil {
		return Measured{}, err
	}
	if len(m) >= len(code) {
		m = code
	}
	if c != nil { // its imports point at the .min files, as the published module's do
		if files, err := cat.ModuleFiles(c); err == nil {
			m = rewriteImports(m, strings.TrimPrefix(c.Script, c.Slug+"/"), files)
		}
	}
	out := Measured{Raw: len(code), Min: len(m), MinBrotli: precompress.BrotliLen(m)}

	// Files of c it imports (e.g. code-editor's ./vendor/prism.js).
	if c != nil {
		for _, spec := range Imports(string(m)) {
			if !strings.HasPrefix(spec, "./") && !strings.HasPrefix(spec, "../") {
				continue
			}
			name := path.Clean(spec)
			for _, f := range c.Sizes.Files {
				if (f.Name == name || MinOf(f.Name) == name) && f.Name != strings.TrimPrefix(c.Script, c.Slug+"/") {
					out.Extra += f.Min
				}
			}
		}
	}

	// Components it renders, except the ones it defines itself. The source,
	// not the minified code: minifying renames the rocket import.
	own := map[string]bool{}
	for _, d := range definesRe.FindAllStringSubmatch(string(code), -1) {
		own[d[1]] = true
	}
	seen := map[*Component]bool{}
	for _, u := range cat.rendered(m, own) {
		for _, d := range append([]*Component{u}, cat.Deps(u)...) {
			if !seen[d] && !own[d.Tag] {
				seen[d] = true
				out.Uses = append(out.Uses, d.Tag)
				out.Extra += d.Sizes.Own.Min
			}
		}
	}
	return out, nil
}
