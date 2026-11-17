//go:build !templgen

package sb

import (
	"context"
	"io"
	"slices"
	"sync"

	"github.com/a-h/templ"
)

// Tracked renders its children, then the import map and a module script for
// each component they rendered (and the components those render
// themselves), pinned like Imports. It writes the page's import map, so it
// goes around the content of <body>, with no import map or module script
// before it: the page's own modules come after it.
//
//	<body>
//		@sb.Tracked(sb.ImportsProps{}) {
//			…the page…
//		}
//	</body>
//
// A component a morph adds later is not loaded: name it in
// ImportsProps.Components, or use Imports with the autoloader on pages that
// morph in new kinds of components. A Tracked inside another only renders
// its children.
func Tracked(p ImportsProps) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		children := templ.GetChildren(ctx)
		ctx = templ.ClearChildren(ctx)
		if collectorOf(ctx) != nil {
			return children.Render(ctx, w)
		}
		u := &collector{seen: map[Tag]bool{}}
		if err := children.Render(context.WithValue(ctx, collectorKey{}, u), w); err != nil {
			return err
		}
		return writeImports(ctx, w, p, append(slices.Clone(p.Components), u.tags()...), false)
	})
}

type collectorKey struct{}

// collector collects the components rendered inside a Tracked.
type collector struct {
	mu   sync.Mutex
	seen map[Tag]bool
	list []Tag
}

func collectorOf(ctx context.Context) *collector {
	u, _ := ctx.Value(collectorKey{}).(*collector)
	return u
}

// used notes that a wrapper rendered t, inside a Tracked.
func used(ctx context.Context, t Tag) {
	u := collectorOf(ctx)
	if u == nil {
		return
	}
	u.mu.Lock()
	defer u.mu.Unlock()
	for _, d := range append([]Tag{t}, pins[t].deps...) {
		if !u.seen[d] {
			u.seen[d] = true
			u.list = append(u.list, d)
		}
	}
}

func (u *collector) tags() []Tag {
	u.mu.Lock()
	defer u.mu.Unlock()
	return slices.Clone(u.list)
}
