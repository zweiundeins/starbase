package catalog

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"path"
	"strings"
	"sync"

	"github.com/evanw/esbuild/pkg/api"
)

// Bundle is every component in one minified module (/c/bundle.js). Pages
// load it instead of the autoloader while it fits web.bundleBudget.
// 'datastar' stays external (the page's import map provides it), and dynamic
// imports stay lazy: they point at the component's versioned public files
// instead of being inlined, so a library a component loads on demand (ECharts)
// is not pulled into every page.
func (cat *Catalog) Bundle() ([]byte, error) {
	if b, ok := bundleCache.Load(cat.Hash); ok {
		return b.([]byte), nil
	}
	b, err := cat.bundle()
	if err == nil {
		bundleCache.Store(cat.Hash, b)
	}
	return b, err
}

var bundleCache sync.Map // Catalog.Hash → bundle bytes (a process loads the same catalog many times)

func (cat *Catalog) bundle() ([]byte, error) {
	var entry strings.Builder
	tags := make([]string, 0, len(cat.Components))
	for _, c := range cat.Components {
		fmt.Fprintf(&entry, "import %q\n", "./"+c.Script)
		tags = append(tags, c.Tag)
	}
	// Like the autoloader: take the sb-cloak class off <html> once the
	// components on the page are defined (Rocket defines them when Datastar
	// is ready), waiting only for tags this bundle defines, and never longer
	// than 3 s.
	tj, _ := json.Marshal(tags)
	fmt.Fprintf(&entry, `const tags = new Set(%s)
const uncloak = () => document.documentElement.classList.remove('sb-cloak')
const present = [...new Set([...document.querySelectorAll(':not(:defined)')].map((e) => e.localName))].filter((t) => tags.has(t))
Promise.all(present.map((t) => customElements.whenDefined(t))).then(uncloak)
setTimeout(uncloak, 3000)
`, tj)
	src := entry.String()
	res := api.Build(api.BuildOptions{
		Stdin:             &api.StdinOptions{Contents: src, Sourcefile: "bundle-entry.js", Loader: api.LoaderJS},
		Bundle:            true,
		Format:            api.FormatESModule,
		Target:            api.ESNext,
		MinifyWhitespace:  true,
		MinifyIdentifiers: true,
		MinifySyntax:      true,
		Charset:           api.CharsetUTF8,
		LegalComments:     api.LegalCommentsInline,
		Outfile:           "bundle.js",
		Write:             false,
		Plugins:           []api.Plugin{cat.fsPlugin()},
	})
	if len(res.Errors) > 0 {
		return nil, fmt.Errorf("esbuild: %s", res.Errors[0].Text)
	}
	return res.OutputFiles[0].Contents, nil
}

// fsPlugin resolves and loads component files from the catalog's FS.
func (cat *Catalog) fsPlugin() api.Plugin {
	return api.Plugin{
		Name: "components-fs",
		Setup: func(b api.PluginBuild) {
			b.OnResolve(api.OnResolveOptions{Filter: `.*`}, func(args api.OnResolveArgs) (api.OnResolveResult, error) {
				if args.Path == "datastar" {
					return api.OnResolveResult{Path: "datastar", External: true}, nil
				}
				base := "."
				if args.Namespace == "cfs" {
					base = path.Dir(args.Importer)
				}
				p := path.Clean(path.Join(base, args.Path))
				if args.Kind == api.ResolveJSDynamicImport {
					slug, rest, _ := strings.Cut(p, "/")
					c, ok := cat.Get(slug)
					if !ok {
						return api.OnResolveResult{}, fmt.Errorf("dynamic import outside a component: %s", p)
					}
					// The minified file (prism.js → prism.min.js), as the .min modules' own imports do.
					return api.OnResolveResult{Path: "/c/" + slug + "@" + c.Hash + "/" + MinOf(rest), External: true}, nil
				}
				return api.OnResolveResult{Path: p, Namespace: "cfs"}, nil
			})
			b.OnLoad(api.OnLoadOptions{Filter: `.*`, Namespace: "cfs"}, func(args api.OnLoadArgs) (api.OnLoadResult, error) {
				body, err := fs.ReadFile(cat.FS, args.Path)
				_, name, _ := strings.Cut(args.Path, "/") // inside the component folder
				s := string(shrink(name, body))
				return api.OnLoadResult{Contents: &s, Loader: api.LoaderJS}, err
			})
		},
	}
}
