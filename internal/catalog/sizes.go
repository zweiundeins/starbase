package catalog

import (
	"bytes"
	"compress/gzip"
	"errors"
	"io/fs"
	"slices"
	"strings"
	"sync"

	"github.com/andybalholm/brotli"
)

// Size is what a file (or a group of files) weighs over the wire.
type Size struct {
	Raw, Gzip, Brotli int
}

func (s Size) Add(o Size) Size {
	return Size{s.Raw + o.Raw, s.Gzip + o.Gzip, s.Brotli + o.Brotli}
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

func compressed(b []byte) Size {
	var gz, br bytes.Buffer
	zw, _ := gzip.NewWriterLevel(&gz, gzip.BestCompression)
	zw.Write(b)
	zw.Close()
	bw := brotli.NewWriterLevel(&br, brotli.BestCompression)
	bw.Write(b)
	bw.Close()
	return Size{Raw: len(b), Gzip: gz.Len(), Brotli: br.Len()}
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
	var s Sizes
	for _, n := range names {
		sz := compressed(files[n])
		s.Files = append(s.Files, NamedSize{n, sz})
		s.Own = s.Own.Add(sz)
	}
	ownSizes.Store(c.Hash, s)
	return s, nil
}

// computeSizes fills every component's Sizes (components in parallel).
func (cat *Catalog) computeSizes() error {
	own := make(map[*Component]Sizes, len(cat.Components))
	var mu sync.Mutex
	var wg sync.WaitGroup
	errs := make([]error, len(cat.Components))
	for i, c := range cat.Components {
		wg.Go(func() {
			s, err := cat.ownSizes(c)
			mu.Lock()
			own[c], errs[i] = s, err
			mu.Unlock()
		})
	}
	wg.Wait()
	if err := errors.Join(errs...); err != nil {
		return err
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
