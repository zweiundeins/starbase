//go:build !templgen

package sb

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
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
	// map that pins every component (about 15 kB).
	Components []Tag
	// Datastar is the URL of a Datastar build with Rocket the page loads
	// already. Empty: the build these wrappers were generated with, pinned
	// on Base.
	Datastar string
	// DatastarIntegrity is Datastar's SRI hash ("sha384-…"), if known.
	DatastarIntegrity string
}

// Imports renders the import map and module scripts that load the
// components, each pinned with its Subresource Integrity hash, so the
// browser refuses a file that changed. Put it in <head>, once per page. The
// scripts get the CSP nonce of templ.WithNonce.
//
//	@sb.Imports(sb.ImportsProps{Components: []sb.Tag{sb.TagToggle, sb.TagModal}})
func Imports(p ImportsProps) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
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
		var scripts []string
		if len(p.Components) == 0 {
			for _, t := range pinOrder {
				pin := pins[t]
				integrity[base+"/c/"+pin.bundle] = pin.integrity
				for _, f := range pin.lazy {
					integrity[base+"/c/"+f.path] = f.integrity
				}
			}
			scripts = append(scripts, script(base+"/c/@"+snapshot+"/autoloader.js", snapshotIntegrity))
		} else {
			seen := map[Tag]bool{}
			var load []Tag
			for _, t := range p.Components {
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
		// json.Marshal escapes <, > and &, so the map can't end the script.
		if _, err := io.WriteString(w, `<script type="importmap"`+nonce+`>`+string(b)+"</script>"); err != nil {
			return err
		}
		for _, s := range scripts {
			if _, err := io.WriteString(w, strings.Replace(s, "<script", "<script"+nonce, 1)); err != nil {
				return err
			}
		}
		return nil
	})
}

func script(src, integrity string) string {
	return `<script type="module" src="` + templ.EscapeString(src) + `" integrity="` + integrity + `"></script>`
}

// pin is a component's one-file bundle at the version the wrappers were
// generated with.
type pin struct {
	bundle    string // "<slug>@<hash>/<slug>.bundle.min.js", under /c/
	integrity string
	lazy      []pinnedFile // files the bundle imports on demand
	deps      []Tag        // the components it renders, transitively
}

type pinnedFile struct{ path, integrity string }
