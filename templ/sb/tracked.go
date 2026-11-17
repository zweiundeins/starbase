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
// its children. It is Collect around its children, then Imports.
func Tracked(p ImportsProps) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		children := templ.GetChildren(ctx)
		ctx = templ.ClearChildren(ctx)
		for c := collectorOf(ctx); c != nil; c = c.parent {
			if c.tracked {
				return children.Render(ctx, w)
			}
		}
		inner, rendered := collect(ctx, true)
		if err := children.Render(inner, w); err != nil {
			return err
		}
		return writeImports(ctx, w, p, append(slices.Clone(p.Components), rendered()...), false)
	})
}

// Collect returns a context in which every wrapper notes the component it
// renders, and a function that returns those components so far, with the
// components they render themselves, each once, in the order they were
// first rendered. For a page with a loader of its own (see Pin):
//
//	ctx, rendered := sb.Collect(ctx)
//	page.Render(ctx, &body)
//	for _, t := range rendered() { … sb.Pin(t) … }
//
// Collects nest, and a Collect sees what a Tracked inside it rendered.
func Collect(ctx context.Context) (context.Context, func() []Tag) {
	return collect(ctx, false)
}

func collect(ctx context.Context, tracked bool) (context.Context, func() []Tag) {
	c := &collector{parent: collectorOf(ctx), tracked: tracked, seen: map[Tag]bool{}}
	return context.WithValue(ctx, collectorKey{}, c), c.tags
}

type collectorKey struct{}

// collector collects the components rendered inside a Collect or Tracked.
type collector struct {
	parent  *collector
	tracked bool
	mu      sync.Mutex
	seen    map[Tag]bool
	list    []Tag
}

func collectorOf(ctx context.Context) *collector {
	c, _ := ctx.Value(collectorKey{}).(*collector)
	return c
}

// used notes that a wrapper rendered t, in every Collect around it.
func used(ctx context.Context, t Tag) {
	for c := collectorOf(ctx); c != nil; c = c.parent {
		c.add(t)
	}
}

func (c *collector) add(t Tag) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, d := range append([]Tag{t}, pins[t].deps...) {
		if !c.seen[d] {
			c.seen[d] = true
			c.list = append(c.list, d)
		}
	}
}

func (c *collector) tags() []Tag {
	c.mu.Lock()
	defer c.mu.Unlock()
	return slices.Clone(c.list)
}
