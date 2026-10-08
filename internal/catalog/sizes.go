package catalog

import (
	"io/fs"
	"slices"
	"strings"
	"sync"

	"starbase/internal/precompress"
)

// Size is what a file (or a group of files) weighs over the wire. Min is
// the minified file under brotli: what a page actually downloads.
type Size struct {
	Raw, Gzip, Brotli, Min int
}

func (s Size) Add(o Size) Size {
	return Size{s.Raw + o.Raw, s.Gzip + o.Gzip, s.Brotli + o.Brotli, s.Min + o.Min}
}

// NamedSize is one line of a size table: a file, or a component it renders.
type NamedSize struct {
	Name string
	Size
	Lazy bool // loaded on first use (a dynamic import)
}

// Sizes is a component's download weight, each file compressed on its own
// (the way it is served), at the best levels (gzip -9, brotli -11), as a
// build that precompresses would ship it. Datastar and Rocket are not
// counted: every component shares them. A file loaded on first use
// (code-editor's Prism, ECharts' library) counts: the component loads it as
// soon as it highlights code or draws a chart.
type Sizes struct {
	// What a page loads: the one-file bundles (onefile.go), and what they
	// load on first use. A bundle is minified, so its Min is its Brotli.
	Bundle   NamedSize   // its own bundle (BundleName)
	Lazy     []NamedSize // the files its bundle loads on first use (ComponentBundle.Lazy)
	Bundles  []NamedSize // Bundle, then the bundles of the components it renders (Catalog.Deps)
	UsesLazy []NamedSize // the files those load on first use, as "<slug>/<file>"
	Total    Size        // the sum of Bundles, Lazy and UsesLazy: its download size (gallery cards, its page)

	// The same as module files, for a page that loads them as they are or
	// bundles them itself.
	Files        []NamedSize // the module files a page loads: <slug>.js first, then what it imports
	Uses         []NamedSize // components it renders (transitively), by tag, all their files
	Own          Size        // the sum of Files
	ModulesTotal Size        // Own plus Uses
	ModuleFiles  int         // the requests: the files in Files and those Uses stands for
}

// LazyMin is how much of Total is loaded on first use, minified under brotli.
func (s Sizes) LazyMin() int {
	n := 0
	for _, f := range append(s.Lazy, s.UsesLazy...) {
		n += f.Min
	}
	return n
}

// compressed measures b the way it is served: precompress keeps the brotli
// and gzip bytes it computes here, so the server sends exactly these.
func compressed(b []byte) Size {
	br, gz := precompress.Get(b)
	return Size{Raw: len(b), Gzip: len(gz), Brotli: len(br)}
}

func brotliLen(b []byte) int {
	br, _ := precompress.Get(b)
	return len(br)
}

// Uses returns the catalog components c renders itself (found as <sb-… in
// its module), in order of first appearance. It scans the minified module:
// its comments are gone, so a tag a comment merely mentions (e.g. "a pick on
// <sb-theme-switch> arrives as…") is not mistaken for a dependency, while the
// templates and strings that really render tags are kept as they are.
func (cat *Catalog) Uses(c *Component) []*Component {
	src, _ := fs.ReadFile(cat.FS, c.Script)
	if mins, err := cat.MinFiles(c); err == nil {
		if m, ok := mins[MinOf(strings.TrimPrefix(c.Script, c.Slug+"/"))]; ok {
			src = m
		}
	}
	return cat.rendered(src, map[string]bool{c.Tag: true})
}

// rendered returns the catalog components src renders (<sb-… in it), in
// order of first appearance, except the tags in own.
func (cat *Catalog) rendered(src []byte, own map[string]bool) []*Component {
	var out []*Component
	for _, m := range usesTagRe.FindAllStringSubmatch(string(src), -1) {
		for _, d := range cat.Components {
			if d.Tag == m[1] && !own[d.Tag] && !slices.Contains(out, d) {
				out = append(out, d)
			}
		}
	}
	return out
}

// Deps returns every catalog component c renders, transitively (depth
// first, in order of first appearance): what a page needs besides c itself.
func (cat *Catalog) Deps(c *Component) []*Component {
	seen := map[*Component]bool{c: true}
	var out []*Component
	var walk func(*Component)
	walk = func(d *Component) {
		for _, u := range cat.Uses(d) {
			if !seen[u] {
				seen[u] = true
				out = append(out, u)
				walk(u)
			}
		}
	}
	walk(c)
	return out
}

// ownSizes caches each component version's own sizes by its content hash:
// brotli -11 is slow, and a process (a test binary above all) loads the
// same catalog many times.
var ownSizes sync.Map // Component.Hash → Sizes (Bundle, Lazy, Files, Own, ModuleFiles)

func (cat *Catalog) ownSizes(c *Component) (Sizes, error) {
	if s, ok := ownSizes.Load(c.Hash); ok {
		return s.(Sizes), nil
	}
	files, err := cat.ModuleFiles(c)
	if err != nil {
		return Sizes{}, err
	}
	mins, err := cat.MinFiles(c)
	if err != nil {
		return Sizes{}, err
	}
	bundle, err := cat.BundleOf(c)
	if err != nil {
		return Sizes{}, err
	}
	var s Sizes
	s.Bundle = NamedSize{Name: BundleName(c.Slug), Size: compressed(bundle.Body)}
	s.Bundle.Min = s.Bundle.Brotli
	for _, n := range bundle.Lazy {
		b, ok := mins[n]
		if !ok {
			b = files[n] // an already-minified vendored file
		}
		sz := compressed(b)
		sz.Min = sz.Brotli
		s.Lazy = append(s.Lazy, NamedSize{Name: n, Size: sz, Lazy: true})
	}

	main := strings.TrimPrefix(c.Script, c.Slug+"/")
	names := make([]string, 0, len(files))
	for n := range cat.Loaded(c, files) {
		names = append(names, n)
	}
	slices.SortFunc(names, func(a, b string) int {
		if (a == main) != (b == main) {
			if a == main {
				return -1
			}
			return 1
		}
		return strings.Compare(a, b)
	})
	for _, n := range names {
		sz := compressed(files[n])
		sz.Min = sz.Brotli // an already-minified file ships as it is
		if m, ok := mins[MinPath(n)]; ok {
			sz.Min = brotliLen(m)
		}
		s.Files = append(s.Files, NamedSize{Name: n, Size: sz, Lazy: slices.Contains(bundle.Lazy, MinOf(n))})
		s.Own = s.Own.Add(sz)
		s.ModuleFiles++
	}
	ownSizes.Store(c.Hash, s)
	return s, nil
}

// computeSizes fills every component's Sizes. One component at a time: a
// brotli -11 encoder is memory-hungry, and the service runs under a tight
// memory limit.
func (cat *Catalog) computeSizes() error {
	own := make(map[*Component]Sizes, len(cat.Components))
	for _, c := range cat.Components {
		s, err := cat.ownSizes(c)
		if err != nil {
			return err
		}
		own[c] = s
	}
	for _, c := range cat.Components {
		s := own[c]
		s.Bundles = []NamedSize{s.Bundle}
		s.Total = s.Bundle.Size
		for _, f := range s.Lazy {
			s.Total = s.Total.Add(f.Size)
		}
		s.ModulesTotal = s.Own
		for _, u := range cat.Deps(c) {
			d := own[u]
			s.Bundles = append(s.Bundles, d.Bundle)
			s.Total = s.Total.Add(d.Bundle.Size)
			for _, f := range d.Lazy {
				f.Name = u.Slug + "/" + f.Name
				s.UsesLazy = append(s.UsesLazy, f)
				s.Total = s.Total.Add(f.Size)
			}
			s.Uses = append(s.Uses, NamedSize{Name: u.Tag, Size: d.Own})
			s.ModulesTotal = s.ModulesTotal.Add(d.Own)
			s.ModuleFiles += d.ModuleFiles
		}
		c.Sizes = s
	}
	return nil
}
