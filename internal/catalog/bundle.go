package catalog

import (
	"fmt"
	"io/fs"
	"path"
	"strings"

	"github.com/evanw/esbuild/pkg/api"
)

// Bundle is every component in one minified module (an experiment: one file
// against the autoloader's many, see /c/bundle.js and ?load=bundle).
// 'datastar' stays external (the page's import map provides it), and dynamic
// imports stay lazy: they point at the component's versioned public files
// instead of being inlined, so a library a component loads on demand (ECharts)
// is not pulled into every page.
func (cat *Catalog) Bundle() ([]byte, error) {
	var entry strings.Builder
	for _, c := range cat.Components {
		fmt.Fprintf(&entry, "import %q\n", "./"+c.Script)
	}
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
					return api.OnResolveResult{Path: "/c/" + slug + "@" + c.Hash + "/" + rest, External: true}, nil
				}
				return api.OnResolveResult{Path: p, Namespace: "cfs"}, nil
			})
			b.OnLoad(api.OnLoadOptions{Filter: `.*`, Namespace: "cfs"}, func(args api.OnLoadArgs) (api.OnLoadResult, error) {
				body, err := fs.ReadFile(cat.FS, args.Path)
				s := string(body)
				return api.OnLoadResult{Contents: &s, Loader: api.LoaderJS}, err
			})
		},
	}
}
