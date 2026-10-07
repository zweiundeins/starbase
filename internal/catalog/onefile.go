package catalog

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"path"
	"slices"
	"strings"
	"sync"

	"github.com/evanw/esbuild/pkg/api"
)

// One-file bundles. A page that loads a component's modules makes one request
// per file, in as many rounds as its imports are deep (sb-kanban-board: 13
// files in three rounds). So every component also ships as one file: its main
// module with every module it imports statically inlined, shrunk and minified
// like the .min files. 'datastar' stays external (the page's import map
// provides it, so a page keeps one Datastar), a dynamic import() stays lazy
// and points at the .min file next to the bundle, and the components it
// renders are not inlined: they load as their own bundles. Module identity
// across components doesn't change: each folder has its own copy of what it
// imports (PD rockets' core/), whose shared registry lives on globalThis.
//
// The bundle is one of MinFiles, so it is frozen per version like them
// (SyncCatalog, component_files) and served at
// /c/<slug>@<hash>/<slug>.bundle.min.js. Its banner names the tag and the
// version, nothing else, so a version's bundle has the same bytes wherever
// it is built (cmd/dist).

// BundleName is a component's one-file bundle inside its folder:
// "<slug>.bundle.min.js". No component file may take that name (Load).
func BundleName(slug string) string { return slug + ".bundle.min.js" }

// VersionedBundle is the bundle's versioned URL path:
// "<slug>@<hash>/<slug>.bundle.min.js".
func (c *Component) VersionedBundle() string {
	return c.Slug + "@" + c.Hash + "/" + BundleName(c.Slug)
}

// ComponentBundle is a component's one-file bundle.
type ComponentBundle struct {
	Body []byte
	// Modules are the module files inlined in Body, by name inside the
	// folder, sorted.
	Modules []string
	// Imports are what Body imports itself, in order: "datastar", and its
	// lazy imports of .min files next to it ("./vendor/prism.min.js").
	Imports []BundleImport
	// Lazy are the files a page may load after Body, on demand: its lazy
	// imports and what those import, by name inside the folder, sorted.
	Lazy []string
}

// BundleImport is one import in a bundle.
type BundleImport struct {
	Path string
	Lazy bool // a dynamic import()
}

var componentBundles sync.Map // Component.Hash → *ComponentBundle

// BundleOf returns c's one-file bundle.
func (cat *Catalog) BundleOf(c *Component) (*ComponentBundle, error) {
	if b, ok := componentBundles.Load(c.Hash); ok {
		return b.(*ComponentBundle), nil
	}
	b, err := cat.bundleOf(c)
	if err != nil {
		return nil, fmt.Errorf("%s: one-file bundle: %w", c.Slug, err)
	}
	componentBundles.Store(c.Hash, b)
	return b, nil
}

func (cat *Catalog) bundleOf(c *Component) (*ComponentBundle, error) {
	files, err := cat.ModuleFiles(c)
	if err != nil {
		return nil, err
	}
	out := BundleName(c.Slug)
	res := api.Build(api.BuildOptions{
		EntryPoints:       []string{c.Script},
		Bundle:            true,
		Format:            api.FormatESModule,
		Target:            api.ESNext,
		MinifyWhitespace:  true,
		MinifyIdentifiers: true,
		MinifySyntax:      true,
		Charset:           api.CharsetUTF8,
		LegalComments:     api.LegalCommentsInline,
		Banner:            map[string]string{"js": fmt.Sprintf("/*! %s, version %s@%s: its modules in one file */", c.Tag, c.Slug, c.Hash)},
		Outfile:           out,
		Metafile:          true,
		Write:             false,
		LogLevel:          api.LogLevelSilent,
		Plugins:           []api.Plugin{cat.folderPlugin(c)},
	})
	if len(res.Errors) > 0 {
		return nil, fmt.Errorf("esbuild: %s", res.Errors[0].Text)
	}
	var meta struct {
		Outputs map[string]struct {
			Imports []struct {
				Path     string `json:"path"`
				Kind     string `json:"kind"`
				External bool   `json:"external"`
			} `json:"imports"`
			Inputs map[string]json.RawMessage `json:"inputs"`
		} `json:"outputs"`
	}
	if err := json.Unmarshal([]byte(res.Metafile), &meta); err != nil {
		return nil, err
	}
	o, ok := meta.Outputs[out]
	if !ok || len(res.OutputFiles) != 1 {
		return nil, fmt.Errorf("esbuild wrote %d files, not %s", len(res.OutputFiles), out)
	}
	b := &ComponentBundle{Body: res.OutputFiles[0].Contents}
	for in := range o.Inputs {
		b.Modules = append(b.Modules, strings.TrimPrefix(strings.TrimPrefix(in, "cfs:"), c.Slug+"/"))
	}
	slices.Sort(b.Modules)
	lazy := map[string]bool{}
	for _, imp := range o.Imports {
		bi := BundleImport{Path: imp.Path, Lazy: imp.Kind == "dynamic-import"}
		if !slices.Contains(b.Imports, bi) {
			b.Imports = append(b.Imports, bi)
		}
		if !bi.Lazy {
			continue
		}
		// Everything the lazily loaded file reaches, as the .min files that ship.
		for name := range files {
			if "./"+MinOf(name) == imp.Path {
				for n := range reach(files, name) {
					lazy[MinOf(n)] = true
				}
			}
		}
	}
	for n := range lazy {
		b.Lazy = append(b.Lazy, n)
	}
	slices.Sort(b.Lazy)
	return b, nil
}

// folderPlugin resolves a component's modules inside its folder, shrunk like
// the .min files: 'datastar' is external, and a dynamic import is too, as
// the .min file next to the bundle (which sits at the folder's top).
func (cat *Catalog) folderPlugin(c *Component) api.Plugin {
	return api.Plugin{
		Name: "component-folder",
		Setup: func(b api.PluginBuild) {
			b.OnResolve(api.OnResolveOptions{Filter: `.*`}, func(args api.OnResolveArgs) (api.OnResolveResult, error) {
				switch {
				case args.Kind == api.ResolveEntryPoint:
					return api.OnResolveResult{Path: args.Path, Namespace: "cfs"}, nil
				case args.Path == "datastar":
					return api.OnResolveResult{Path: "datastar", External: true}, nil
				case !strings.HasPrefix(args.Path, "./") && !strings.HasPrefix(args.Path, "../"):
					return api.OnResolveResult{}, fmt.Errorf("%s imports %q: only 'datastar' and files in the component's folder", args.Importer, args.Path)
				}
				p := path.Clean(path.Join(path.Dir(args.Importer), args.Path))
				rest, inside := strings.CutPrefix(p, c.Slug+"/")
				if !inside {
					return api.OnResolveResult{}, fmt.Errorf("%s imports %q, outside the component's folder", args.Importer, args.Path)
				}
				if args.Kind == api.ResolveJSDynamicImport {
					return api.OnResolveResult{Path: "./" + MinOf(rest), External: true}, nil
				}
				return api.OnResolveResult{Path: p, Namespace: "cfs"}, nil
			})
			b.OnLoad(api.OnLoadOptions{Filter: `.*`, Namespace: "cfs"}, cat.loadShrunk)
		},
	}
}

// loadShrunk loads a component file from the catalog's FS, shrunk (shrink.go).
func (cat *Catalog) loadShrunk(args api.OnLoadArgs) (api.OnLoadResult, error) {
	body, err := fs.ReadFile(cat.FS, args.Path)
	_, name, _ := strings.Cut(args.Path, "/") // inside the component folder
	s := string(shrink(name, body))
	return api.OnLoadResult{Contents: &s, Loader: api.LoaderJS}, err
}
