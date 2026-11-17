package web

import (
	"strings"
	"sync"

	"github.com/zweiundeins/starbase/templ/sb"

	"starbase/internal/catalog"
	"starbase/internal/templgen"
	"starbase/internal/ui"
)

// templModels holds each catalog's templ wrappers (templgen.Build), made
// once: the templ tab writes a component's usage as calls of them.
var templModels sync.Map // *catalog.Catalog → *templgen.Model or error

func (s *Server) templModel() (*templgen.Model, error) {
	if v, ok := templModels.Load(s.catalog); ok {
		if err, ok := v.(error); ok {
			return nil, err
		}
		return v.(*templgen.Model), nil
	}
	m, err := templgen.Build(s.catalog)
	if err != nil {
		templModels.Store(s.catalog, err)
		return nil, err
	}
	templModels.Store(s.catalog, m)
	return m, nil
}

// templInstall is the templ tab of c's Installation: go get, sb.Imports for
// the components its usage renders, and the usage as wrapper calls.
func (s *Server) templInstall(c *catalog.Component) ui.TemplInstall {
	t := ui.TemplInstall{
		Get:    ui.NewSnippetIn("go get github.com/zweiundeins/starbase/templ@latest", "sh"),
		Module: strings.TrimSuffix(s.cfg.RepoURL, "/") + "/tree/main/templ#readme",
	}
	m, err := s.templModel()
	if err != nil {
		s.log.Warn("templ wrappers", "err", err)
		return t
	}
	uses, err := m.Uses(c.InstallMarkup())
	if err != nil || len(uses) == 0 {
		w, _ := m.Get(c.Tag)
		uses = []*templgen.Component{w}
	}
	var tags []string
	for _, u := range uses {
		if u != nil {
			tags = append(tags, "sb.Tag"+u.GoName)
		}
	}
	props := "Components: []sb.Tag{" + strings.Join(tags, ", ") + "}"
	if base := strings.TrimSuffix(s.cfg.BaseURL, "/"); base != sb.DefaultBase {
		props = "Base: \"" + base + "\", " + props
	}
	t.Imports = ui.NewSnippetIn("@sb.Imports(sb.ImportsProps{"+props+"})", "templ")
	if usage, err := m.Templ(c.InstallMarkup()); err == nil {
		t.Usage = ui.NewSnippetIn(usage, "templ")
	} else {
		s.log.Warn("templ usage", "component", c.Slug, "err", err)
	}
	return t
}
