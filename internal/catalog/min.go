package catalog

import (
	"fmt"
	"path"
	"strings"
	"sync"

	"github.com/evanw/esbuild/pkg/api"
)

// Minified module files. Every module file x.js (or x.mjs) gets a sibling
// x.min.js (x.min.mjs), whose relative imports point at the minified files
// too. The readable file stays the default URL (the docs, the playground and
// "view source" show the real code); the autoloader and the site load .min.
//
// Minified bytes depend on the esbuild version, and versioned URLs are
// immutable, so a version's .min files are frozen the first time they are
// stored (SyncCatalog, component_files) and served from there afterwards.

// MinPath is the minified sibling of a module file: "x.js" → "x.min.js".
func MinPath(p string) string {
	ext := path.Ext(p)
	return strings.TrimSuffix(p, ext) + ".min" + ext
}

// IsMinPath reports whether p names a minified module file.
func IsMinPath(p string) bool {
	return strings.HasSuffix(p, ".min.js") || strings.HasSuffix(p, ".min.mjs")
}

// VersionedMinScript is the minified module's versioned URL path:
// "<slug>@<hash>/<slug>.min.js".
func (c *Component) VersionedMinScript() string {
	return c.Slug + "@" + c.Hash + "/" + MinPath(strings.TrimPrefix(c.Script, c.Slug+"/"))
}

var minCache sync.Map // Component.Hash → map[string][]byte (min path → bytes)

// MinFiles returns the component's minified module files, keyed by their
// .min path inside the folder.
func (cat *Catalog) MinFiles(c *Component) (map[string][]byte, error) {
	if m, ok := minCache.Load(c.Hash); ok {
		return m.(map[string][]byte), nil
	}
	files, err := cat.ModuleFiles(c)
	if err != nil {
		return nil, err
	}
	out := make(map[string][]byte, len(files))
	for p, src := range files {
		// A file that is already minified (a vendored "*.min.js") is its own
		// minified version: re-minifying it only costs time and bytes.
		if IsMinPath(p) {
			continue
		}
		b, err := minify(src)
		if err != nil {
			return nil, fmt.Errorf("%s/%s: %w", c.Slug, p, err)
		}
		if len(b) >= len(src) { // nothing to gain: keep the original bytes
			b = src
		}
		out[MinPath(p)] = rewriteImports(b, p, files)
	}
	minCache.Store(c.Hash, out)
	return out, nil
}

func minify(src []byte) ([]byte, error) {
	res := api.Transform(string(src), api.TransformOptions{
		Loader:            api.LoaderJS,
		Format:            api.FormatESModule,
		Target:            api.ESNext, // minify only; never lower the syntax
		MinifyWhitespace:  true,
		MinifyIdentifiers: true,
		MinifySyntax:      true,
		Charset:           api.CharsetUTF8, // keep emoji and ✓ as they are, not \u escapes
		LegalComments:     api.LegalCommentsInline,
	})
	if len(res.Errors) > 0 {
		return nil, fmt.Errorf("esbuild: %s", res.Errors[0].Text)
	}
	return res.Code, nil
}

// rewriteImports points a minified file's relative imports at the minified
// siblings: "./vendor/prism.js" → "./vendor/prism.min.js". Only specifiers
// that name one of the component's own module files are touched.
func rewriteImports(code []byte, file string, files map[string][]byte) []byte {
	s := string(code)
	for _, spec := range Imports(s) {
		if !strings.HasPrefix(spec, "./") && !strings.HasPrefix(spec, "../") {
			continue
		}
		target := path.Join(path.Dir(file), spec)
		if _, ok := files[target]; !ok || IsMinPath(target) { // already minified: it has no sibling
			continue
		}
		min := MinPath(spec)
		for _, q := range []string{`"`, `'`} {
			s = strings.ReplaceAll(s, q+spec+q, q+min+q)
		}
	}
	return []byte(s)
}
