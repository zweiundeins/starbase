package catalog

import (
	"io/fs"
	"slices"
	"strings"
	"sync"

	"starbase/internal/precompress"
)

// Size is what a file (or a group of files) weighs over the wire. Min is
// the minified file under brotli: what the autoloader actually ships.
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
}

// Sizes is a component's download weight, each file compressed on its own
// (the way it is served), at the best levels (gzip -9, brotli -11), as a
// build that precompresses would ship it. Datastar and Rocket are not
// counted: every component shares them.
type Sizes struct {
	Files []NamedSize // its own module files: <slug>.js first, then vendored files
	Uses  []NamedSize // components it renders (transitively), by tag, all their files
	Own   Size        // sum of Files
	Total Size        // Own plus Uses
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
// its module), in order of first appearance.
func (cat *Catalog) Uses(c *Component) []*Component {
	src, _ := fs.ReadFile(cat.FS, c.Script)
	var out []*Component
	for _, m := range usesTagRe.FindAllStringSubmatch(string(src), -1) {
		for _, d := range cat.Components {
			if d.Tag == m[1] && d != c && !slices.Contains(out, d) {
				out = append(out, d)
			}
		}
	}
	return out
}

// ownSizes caches each component version's own Sizes by its content hash:
// brotli -11 is slow, and a process (a test binary above all) loads the
// same catalog many times.
var ownSizes sync.Map // Component.Hash → Sizes (Files and Own)

func (cat *Catalog) ownSizes(c *Component) (Sizes, error) {
	if s, ok := ownSizes.Load(c.Hash); ok {
		return s.(Sizes), nil
	}
	files, err := cat.ModuleFiles(c)
	if err != nil {
		return Sizes{}, err
	}
	main := strings.TrimPrefix(c.Script, c.Slug+"/")
	names := make([]string, 0, len(files))
	for n := range files {
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
	mins, err := cat.MinFiles(c)
	if err != nil {
		return Sizes{}, err
	}
	var s Sizes
	for _, n := range names {
		sz := compressed(files[n])
		sz.Min = sz.Brotli // an already-minified file ships as it is
		if m, ok := mins[MinPath(n)]; ok {
			sz.Min = brotliLen(m)
		}
		s.Files = append(s.Files, NamedSize{n, sz})
		s.Own = s.Own.Add(sz)
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
		s.Total = s.Own
		seen := map[*Component]bool{c: true}
		var walk func(*Component)
		walk = func(d *Component) {
			for _, u := range cat.Uses(d) {
				if !seen[u] {
					seen[u] = true
					s.Uses = append(s.Uses, NamedSize{u.Tag, own[u].Own})
					s.Total = s.Total.Add(own[u].Own)
					walk(u)
				}
			}
		}
		walk(c)
		c.Sizes = s
	}
	return nil
}
