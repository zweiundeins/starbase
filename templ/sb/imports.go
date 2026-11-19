//go:build !templgen

package sb

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"runtime/debug"
	"slices"
	"strings"

	"github.com/a-h/templ"
)

// DefaultBase is the Starbase server Imports loads from when ImportsProps
// names none.
const DefaultBase = "https://starbase.zweiundeins.gmbh"

// ImportsProps configures Imports.
type ImportsProps struct {
	// Base is the Starbase server the files come from (DefaultBase when
	// empty), without a trailing slash.
	Base string
	// Components are the components the page uses: each is loaded as one
	// file pinned to the version these wrappers were generated with, with
	// the components it renders itself. Without any, the page gets the
	// catalog snapshot's autoloader instead, which loads every <sb-*> tag the
	// first time it appears (also tags a morph adds later), with an import
	// map that pins every component (about 15 kB). In Tracked, they are
	// loaded besides the components it renders.
	Components []Tag
	// Datastar is the URL of a Datastar build with Rocket the page loads
	// already. Empty: the build these wrappers were generated with, pinned
	// on Base.
	Datastar string
	// DatastarIntegrity is Datastar's SRI hash ("sha384-…"), if known.
	DatastarIntegrity string
	// ScriptsOnly writes only the components' module scripts (or the
	// autoloader's), for a page with an import map of its own that maps
	// "datastar" and loads Datastar itself. Files a bundle loads on demand
	// (Pin(t).Lazy) are then pinned only if that map pins them.
	ScriptsOnly bool
}

// Imports renders the import map and module scripts that load the
// components, each pinned with its Subresource Integrity hash, so the
// browser refuses a file that changed. Datastar gets a module script of its
// own, so a page with Datastar attributes and no component still loads it.
// Put it in <head>, once per page. The scripts get the CSP nonce of
// templ.WithNonce.
//
//	@sb.Imports(sb.ImportsProps{Components: []sb.Tag{sb.TagToggle, sb.TagModal}})
func Imports(p ImportsProps) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		return writeImports(ctx, w, p, p.Components, len(p.Components) == 0)
	})
}

// writeImports writes the import map and the module scripts for tags (with
// their deps), or for the autoloader.
func writeImports(ctx context.Context, w io.Writer, p ImportsProps, tags []Tag, autoload bool) error {
	base := strings.TrimSuffix(p.Base, "/")
	if base == "" {
		base = DefaultBase
	}
	ds, dsSRI := p.Datastar, p.DatastarIntegrity
	if ds == "" {
		ds, dsSRI = base+datastarPath, datastarIntegrity
	}
	integrity := map[string]string{}
	if dsSRI != "" {
		integrity[ds] = dsSRI
	}
	// Datastar itself, so a page without components still gets it; the
	// components' own imports of it reuse this module.
	scripts := []string{script(ds, dsSRI)}
	if autoload {
		for _, t := range pinOrder {
			pin := pins[t]
			integrity[base+"/c/"+pin.bundle] = pin.integrity
			for _, f := range pin.lazy {
				integrity[base+"/c/"+f.path] = f.integrity
			}
		}
		scripts = append(scripts, script(base+CatalogAutoloader().Path, snapshotIntegrity))
	} else {
		seen := map[Tag]bool{}
		var load []Tag
		for _, t := range tags {
			pin, ok := pins[t]
			if !ok {
				return fmt.Errorf("sb: Imports: no component %q", t)
			}
			for _, d := range append([]Tag{t}, pin.deps...) {
				if !seen[d] {
					seen[d] = true
					load = append(load, d)
				}
			}
		}
		for _, t := range load {
			pin := pins[t]
			for _, f := range pin.lazy {
				integrity[base+"/c/"+f.path] = f.integrity
			}
			scripts = append(scripts, script(base+"/c/"+pin.bundle, pin.integrity))
		}
	}
	m := map[string]any{"imports": map[string]string{"datastar": ds}}
	if len(integrity) > 0 {
		m["integrity"] = integrity
	}
	b, err := json.Marshal(m)
	if err != nil {
		return err
	}
	nonce := ""
	if n := templ.GetNonce(ctx); n != "" {
		nonce = ` nonce="` + templ.EscapeString(n) + `"`
	}
	if p.ScriptsOnly {
		scripts = scripts[1:]
	} else if _, err := io.WriteString(w, `<script type="importmap"`+nonce+`>`+string(b)+"</script>"); err != nil {
		// json.Marshal escapes <, > and &, so the map can't end the script.
		return err
	}
	for _, s := range scripts {
		if _, err := io.WriteString(w, strings.Replace(s, "<script", "<script"+nonce, 1)); err != nil {
			return err
		}
	}
	return nil
}

func script(src, integrity string) string {
	if integrity == "" {
		return `<script type="module" src="` + templ.EscapeString(src) + `"></script>`
	}
	return `<script type="module" src="` + templ.EscapeString(src) + `" integrity="` + templ.EscapeString(integrity) + `"></script>`
}

// Asset is a file on a Starbase server: its path (from the server's root)
// and its Subresource Integrity hash.
type Asset struct {
	Path      string // "/c/toggle@0123456789ab/toggle.bundle.min.js"
	Integrity string // "sha384-…"
}

// Pinned is what Imports loads for a component.
type Pinned struct {
	// Bundle is the component's module in one file.
	Bundle Asset
	// Lazy are the files the bundle imports on demand (sb-code-editor's
	// Prism), by a URL relative to the bundle's.
	Lazy []Asset
	// Deps are the components it renders itself, which a page loads too.
	Deps []Tag
}

// Pin returns what Imports loads for t, for a page that builds its own
// script tags or serves the files from its own paths; false for a tag that
// isn't one of the components.
func Pin(t Tag) (Pinned, bool) {
	p, ok := pins[t]
	if !ok {
		return Pinned{}, false
	}
	out := Pinned{Bundle: Asset{"/c/" + p.bundle, p.integrity}, Deps: slices.Clone(p.deps)}
	for _, f := range p.lazy {
		out.Lazy = append(out.Lazy, Asset{"/c/" + f.path, f.integrity})
	}
	return out, true
}

// Tags returns every component, in the order of their slugs.
func Tags() []Tag { return slices.Clone(pinOrder) }

// DatastarBuild is the Datastar build with Rocket the components were
// generated against, which Imports loads unless ImportsProps.Datastar names
// another.
func DatastarBuild() Asset { return Asset{datastarPath, datastarIntegrity} }

// CatalogAutoloader is the catalog snapshot's autoloader, which Imports loads
// without Components.
func CatalogAutoloader() Asset { return Asset{"/c/@" + snapshot + "/autoloader.js", snapshotIntegrity} }

// Snapshot is the hash of the catalog snapshot the wrappers were generated
// from: every component's version and the Datastar build. Starbase serves
// its autoloader and import map under /c/@<Snapshot>/.
const Snapshot = snapshot

// Release is the newest Starbase release when the wrappers were generated,
// from its CHANGELOG, without the tag's "v" ("0.11.1" for the tag v0.11.1).
// It changes only with a release, whose
// commit regenerates the wrappers. A host that vendors Starbase's files
// with a lock naming the release compares it with Release, so the files
// and the wrappers come from the same one.
const Release = release

// Version is the version of this module in the program's build info: the
// release tag ("v0.10.0"), a pseudo-version naming the commit, or "" when
// the build doesn't know it (a replace with a directory, go test inside the
// module). Snapshot names the components themselves.
func Version() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return ""
	}
	for _, m := range append([]*debug.Module{&info.Main}, info.Deps...) {
		if m.Path == modulePath {
			if m.Replace != nil {
				return m.Replace.Version // "" for a directory
			}
			if m.Version == "(devel)" {
				return ""
			}
			return m.Version
		}
	}
	return ""
}

const modulePath = "github.com/zweiundeins/starbase/templ"

// pin is a component's one-file bundle at the version the wrappers were
// generated with.
type pin struct {
	bundle    string // "<slug>@<hash>/<slug>.bundle.min.js", under /c/
	integrity string
	lazy      []pinnedFile // files the bundle imports on demand
	deps      []Tag        // the components it renders, transitively
}

type pinnedFile struct{ path, integrity string }
