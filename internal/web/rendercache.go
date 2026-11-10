package web

import (
	"sync"

	"starbase/internal/catalog"
	"starbase/internal/model"
	"starbase/internal/ui"
)

// renderCache keeps the parts of pages that depend only on what the binary
// embeds (the catalog, content/*.md, the stylesheets) and on rows that never
// change once stored, so a stream frame doesn't redo them: the install tabs
// (their snippets highlighted with chroma), a component page's Playground,
// Size and API sections, the Themes page's highlighted CSS, the content
// pages' Markdown, the playground's import list. Every entry is bounded by the
// binary (a component, a preview theme, a content page), never by a session,
// and none holds a nonce or a preference. Each is what the uncached code
// returns, byte for byte (rendercache_internal_test.go).
type renderCache struct {
	mu       sync.Mutex
	installs map[string]ui.InstallView    // by component slug, without the selected tab
	parts    map[string]ui.ComponentParts // by component slug
	content  map[string]contentDoc        // by content page name

	themes map[string]func() themeCode // by preview theme, filled on first use
	deps   func() string               // the playground's tag → script map, as JSON
	siteLD func() string               // the JSON-LD of a page that adds no schema
	homeLD func() string               // the homepage's: the site and the project
}

func newRenderCache(s *Server) *renderCache {
	c := &renderCache{
		installs: map[string]ui.InstallView{},
		parts:    map[string]ui.ComponentParts{},
		content:  map[string]contentDoc{},
		themes:   map[string]func() themeCode{},
		deps:     sync.OnceValue(s.componentDeps),
		siteLD:   sync.OnceValue(func() string { return s.buildJSONLD(nil) }),
		homeLD:   sync.OnceValue(func() string { return s.buildJSONLD([]any{s.softwareLD()}) }),
	}
	for _, t := range model.PreviewThemes {
		c.themes[t.Slug] = sync.OnceValue(func() themeCode { return s.buildThemeCode(t.Slug) })
	}
	return c
}

func (c *renderCache) install(slug string) (ui.InstallView, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	v, ok := c.installs[slug]
	return v, ok
}

func (c *renderCache) setInstall(slug string, v ui.InstallView) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.installs[slug] = v
}

// themeCode is a preview theme's token overrides, as written and highlighted.
type themeCode struct{ CSS, HTML string }

func (s *Server) buildThemeCode(theme string) themeCode {
	css := s.themeCSS(theme)
	return themeCode{CSS: css, HTML: catalog.Highlight(css, "css")}
}

// themeCode is cached for the preview themes, and built for anything else.
func (s *Server) themeCode(theme string) themeCode {
	if f, ok := s.cache.themes[theme]; ok {
		return f()
	}
	return s.buildThemeCode(theme)
}

// contentDoc is a content page rendered from content/<name>.md.
type contentDoc struct {
	Title, Lede, Description string
	HTML                     string
}

// contentDoc renders content/<name>.md once: content.FS is embedded.
func (s *Server) contentDoc(name string) (contentDoc, error) {
	s.cache.mu.Lock()
	d, ok := s.cache.content[name]
	s.cache.mu.Unlock()
	if ok {
		return d, nil
	}
	d, err := s.buildContentDoc(name)
	if err != nil {
		return d, err
	}
	s.cache.mu.Lock()
	s.cache.content[name] = d
	s.cache.mu.Unlock()
	return d, nil
}

// componentParts renders a component page's Playground, Size and API
// reference sections once per component: they depend on the catalog alone.
func (s *Server) componentParts(c *catalog.Component) (ui.ComponentParts, error) {
	s.cache.mu.Lock()
	p, ok := s.cache.parts[c.Slug]
	s.cache.mu.Unlock()
	if ok {
		return p, nil
	}
	p, err := ui.RenderComponentParts(c)
	if err != nil {
		return p, err
	}
	s.cache.mu.Lock()
	s.cache.parts[c.Slug] = p
	s.cache.mu.Unlock()
	return p, nil
}
