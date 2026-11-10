package web

import (
	"bytes"
	"context"
	"testing"

	"starbase/internal/commands"
	"starbase/internal/model"
	"starbase/internal/queries"
	"starbase/internal/ui"
	"starbase/static"
)

// renderWith renders one page as a stream frame and as a document, with the
// given preferences (load would read them from the database).
func (s *Server) renderWith(t *testing.T, path string, fn pageFunc, prefs model.SessionPrefs) (frame, doc []byte) {
	t.Helper()
	r := frameRequest(path)
	var f, d bytes.Buffer
	err := s.q.View(r.Context(), func(rd *queries.Reader) error {
		tab := model.TabState{Browse: model.BrowseFromQuery(r.URL.Query(), prefs.DefaultSort())}
		v, err := fn(&renderCtx{ctx: r.Context(), r: rd, req: r, sid: "benchsession0000", tabID: "benchtab0000", prefs: prefs, tab: tab})
		if err != nil {
			return err
		}
		sh := s.shell(r, nil, v)
		if err := ui.App(sh, v.Body(sh)).Render(r.Context(), &f); err != nil {
			return err
		}
		return ui.Document(sh, v.Body(sh)).Render(r.Context(), &d)
	})
	if err != nil {
		t.Fatalf("%s: %v", path, err)
	}
	return f.Bytes(), d.Bytes()
}

// Every page renders the same bytes from the render cache as without it:
// each variant is rendered with an empty cache, then again from what that
// render stored.
func TestRenderCacheMatchesFresh(t *testing.T) {
	s := newBenchServer(t)
	check := func(name, path string, fn pageFunc, prefs model.SessionPrefs) {
		t.Helper()
		s.cache = newRenderCache(s)
		freshFrame, freshDoc := s.renderWith(t, path, fn, prefs)
		frame, doc := s.renderWith(t, path, fn, prefs)
		if !bytes.Equal(frame, freshFrame) || !bytes.Equal(doc, freshDoc) {
			t.Errorf("%s: the cached render differs from the fresh one", name)
		}
	}
	check("gallery", "/", s.galleryPage, model.SessionPrefs{})
	for _, c := range s.catalog.Components {
		for _, tab := range model.InstallTabs {
			check(c.Slug+" "+tab.Key, "/components/"+c.Slug, s.componentPage, model.SessionPrefs{InstallTab: tab.Key})
		}
	}
	check("playground", "/playground", s.codePlaygroundPage, model.SessionPrefs{})
	check("playground data-table", "/playground?component=data-table", s.codePlaygroundPage, model.SessionPrefs{})
	check("showcase", "/showcase", s.showcasePage, model.SessionPrefs{})
	check("about", "/about", s.contentPage("about", "about"), model.SessionPrefs{})
	check("contribute", "/contribute", s.contentPage("contribute", "contribute"), model.SessionPrefs{})
	for _, th := range model.PreviewThemes {
		for _, smooth := range []bool{false, true} {
			check("themes "+th.Slug, "/themes", s.themesPage, model.SessionPrefs{PreviewTheme: th.Slug, PreviewSmooth: smooth})
		}
	}
}

// An install view is kept only once every integrity it pins is stored:
// before the first sync it pins what it can, and is built again next time.
func TestInstallCacheWaitsForIntegrities(t *testing.T) {
	s := newBenchServer(t)
	c, _ := s.catalog.Get("kanban-board")
	r := frameRequest("/components/kanban-board")
	install := func() (v ui.InstallView, complete bool) {
		err := s.q.View(r.Context(), func(rd *queries.Reader) error {
			var err error
			rc := &renderCtx{ctx: r.Context(), r: rd, req: r}
			if v, err = s.install(rc, c); err != nil {
				return err
			}
			_, complete, err = s.buildInstall(rc, c)
			return err
		})
		if err != nil {
			t.Fatal(err)
		}
		return v, complete
	}
	if _, complete := install(); !complete {
		t.Fatal("after the sync, every integrity should be stored")
	}
	if _, ok := s.cache.install("kanban-board"); !ok {
		t.Error("a complete install view should be kept")
	}

	// A fresh database: nothing stored yet.
	empty := newBenchServerWith(t, false)
	s.q, s.cache = empty.q, newRenderCache(s)
	if v, complete := install(); complete || v.Pinned.Raw != "" {
		t.Fatalf("before the sync nothing is pinned yet: complete %v", complete)
	}
	if _, ok := s.cache.install("kanban-board"); ok {
		t.Error("an install view with missing integrities must not be kept")
	}
	if err := empty.writer.Do(context.Background(), commands.SyncCatalog{Catalog: s.catalog, Datastar: static.Datastar()}); err != nil {
		t.Fatal(err)
	}
	if v, complete := install(); !complete || v.Pinned.Raw == "" {
		t.Error("after the sync the view pins everything")
	}
	if _, ok := s.cache.install("kanban-board"); !ok {
		t.Error("and is kept")
	}
}
