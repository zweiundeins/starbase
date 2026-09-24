package web

import (
	"cmp"
	"fmt"

	"github.com/a-h/templ"

	"io/fs"
	"strings"

	"starbase/internal/catalog"
	"starbase/internal/commands"
	"starbase/internal/model"
	"starbase/internal/ui"
)

func (s *Server) galleryPage(rc *renderCtx) (view, error) {
	b := rc.tab.Browse
	var uid int64
	if rc.user != nil {
		uid = rc.user.ID
	}
	res, err := rc.r.Browse(rc.ctx, b, uid)
	if err != nil {
		return view{}, err
	}
	previews := make(map[string]string, len(res.Cards))
	sizes := make(map[string]catalog.Size, len(res.Cards))
	for _, c := range res.Cards {
		if comp, ok := s.catalog.Get(c.Slug); ok {
			previews[c.Slug] = comp.Preview
			sizes[c.Slug] = comp.Sizes.Total
		}
	}
	u := "/"
	if q := b.Query().Encode(); q != "" {
		u += "?" + q
	}
	return view{
		Title:       "Starbase · Community components for Rocket",
		Description: "Community-built Rocket web components for the Datastar ecosystem. Copy, use, remix, and launch something great.",
		Nav:         "components",
		Body: func(sh ui.Shell) templ.Component {
			return ui.Gallery(sh, ui.GalleryView{Browse: b, Result: res, Previews: previews, Sizes: sizes})
		},
		URL:        u,
		SearchLive: true,
		Search:     b.Q,
		Signals:    map[string]any{"q": b.Q, "cat": b.Category, "sort": string(b.Sort)},
	}, nil
}

func (s *Server) componentPage(rc *renderCtx) (view, error) {
	slug := rc.req.PathValue("slug")
	comp, ok := s.catalog.Get(slug)
	if !ok {
		return view{}, errNotFound
	}
	var uid int64
	if rc.user != nil {
		uid = rc.user.ID
	}
	card, err := rc.r.Component(rc.ctx, slug, uid)
	if err != nil {
		return view{}, err
	}
	if card == nil {
		return view{}, errNotFound
	}
	install, err := s.install(rc, comp)
	if err != nil {
		return view{}, err
	}
	base := strings.TrimSuffix(s.cfg.BaseURL, "/")
	repo := strings.TrimSuffix(s.cfg.RepoURL, "/") + "/tree/main/components/" + slug
	schema := map[string]any{
		"@type":               "SoftwareSourceCode",
		"name":                card.Name,
		"description":         card.Summary,
		"url":                 base + "/components/" + slug,
		"codeRepository":      cmp.Or(comp.Source, repo),
		"programmingLanguage": "JavaScript",
		"runtimePlatform":     "Web browser (Datastar Rocket)",
		"license":             "https://opensource.org/licenses/MIT",
		"keywords":            strings.Join(comp.Tags, ", "),
		"dateCreated":         comp.Since,
		"author":              map[string]any{"@type": "Person", "name": comp.Author, "url": "https://github.com/" + comp.Author},
		"isPartOf":            map[string]any{"@id": base + "/#website"},
	}
	return view{
		Schema:      []any{schema},
		Title:       card.Name + " · Starbase",
		Description: card.Summary,
		Nav:         "components",
		Body: func(sh ui.Shell) templ.Component {
			return ui.ComponentPage(sh, ui.ComponentView{
				Card:       *card,
				Component:  comp,
				Playground: comp.Playground(),
				Install:    install,
				EditURL:    strings.TrimSuffix(s.cfg.RepoURL, "/") + "/tree/main/components/" + slug,
			})
		},
	}, nil
}

// datastarCDN is the Datastar + Rocket bundle the snippets load. It is
// byte-identical to static/vendor/datastar-rocket.js (so its SRI is ours).
const datastarCDN = "https://cdn.jsdelivr.net/gh/starfederation/datastar@v1.0.4/bundles/datastar-rocket.js"

// install builds a component page's Installation tabs (model.InstallTabs).
// Every snippet is exactly what to paste: the explanations are the page's.
func (s *Server) install(rc *renderCtx, c *catalog.Component) (ui.InstallView, error) {
	base := strings.TrimSuffix(s.cfg.BaseURL, "/")
	preview := strings.TrimSpace(c.Preview)
	deps := s.catalog.Deps(c)
	all := append([]*catalog.Component{c}, deps...)
	v := ui.InstallView{Tab: rc.prefs.InstallTabOrDefault(), Datastar: datastarCDN}
	for _, d := range deps {
		v.Deps = append(v.Deps, d.Tag)
	}
	snippet := func(raw string) ui.Snippet { return ui.Snippet{Raw: raw, HTML: catalog.Highlight(raw, "html")} }
	importMap := func(datastar string) string {
		return fmt.Sprintf("<script type=\"importmap\">\n  { \"imports\": { \"datastar\": %q } }\n</script>\n", datastar)
	}

	v.Autoloader = snippet(importMap(datastarCDN) +
		fmt.Sprintf("<script type=\"module\" src=\"%s/c/autoloader.js\"></script>\n\n%s", base, preview))

	// Integrity for everything the page will load: Datastar, and every file
	// these components ship (the module, plus what it imports itself, like
	// code-editor's Prism). A script tag's integrity covers only that file, so
	// the imported ones are pinned through the import map.
	var entries strings.Builder
	fmt.Fprintf(&entries, "      %q: %q", datastarCDN, s.assets.datastarSRI)
	for _, d := range all {
		for _, f := range d.Sizes.Files {
			p := catalog.MinOf(f.Name)
			sri, err := rc.r.FileIntegrity(rc.ctx, d.Slug, d.Hash, p)
			if err != nil {
				return v, err
			}
			if sri != "" {
				fmt.Fprintf(&entries, ",\n      %q: %q", base+"/c/"+d.Slug+"@"+d.Hash+"/"+p, sri)
			}
		}
	}
	pinnedMap := fmt.Sprintf(`<script type="importmap">
  {
    "imports": { "datastar": %q },
    "integrity": {
%s
    }
  }
</script>
`, datastarCDN, entries.String())

	// This component (and what it renders), pinned to this version.
	var scripts strings.Builder
	for _, d := range all {
		script, sri, err := s.pinnedScript(rc, d)
		if err != nil {
			return v, err
		}
		fmt.Fprintf(&scripts, "<script type=\"module\" src=\"%s/c/%s\" integrity=\"%s\"></script>\n", base, script, sri)
	}
	v.Component = snippet(pinnedMap + scripts.String() + "\n" + preview)

	// Today's catalog snapshot: its autoloader, and integrity for Datastar
	// and every file this component loads (importmap.json has them all).
	_, snapshotSRI, _, err := rc.r.Snapshot(rc.ctx, s.catalog.Hash)
	if err != nil {
		return v, err
	}
	v.ImportMap = fmt.Sprintf("%s/c/@%s/importmap.json", base, s.catalog.Hash)
	if snapshotSRI != "" {
		v.Pinned = snippet(pinnedMap + fmt.Sprintf(`<script type="module" src="%s/c/@%s/autoloader.js" integrity="%s"></script>

%s`, base, s.catalog.Hash, snapshotSRI, preview))
	}

	// Self-host: the files, and an import map at your own Datastar.
	var own strings.Builder
	for _, d := range all {
		g := ui.SelfHostGroup{Tag: d.Tag}
		for _, f := range d.Sizes.Files {
			u := base + "/c/" + d.Slug + "@" + d.Hash + "/"
			g.Files = append(g.Files, ui.SelfHostFile{Name: f.Name, Min: u + catalog.MinOf(f.Name), Readable: u + f.Name, Size: f.Size})
		}
		v.Files = append(v.Files, g)
		fmt.Fprintf(&own, "<script type=\"module\" src=\"/js/%s/%s\"></script>\n", d.Slug, catalog.MinPath(strings.TrimPrefix(d.Script, d.Slug+"/")))
	}
	v.SelfHost = snippet(importMap("/js/datastar-rocket.js") + own.String() + "\n" + preview)
	return v, nil
}

// pinnedScript is a component's module pinned to its version: the minified
// file with its frozen integrity (from when this version was first
// published, see catalog/min.go), or the readable file while this version
// isn't stored yet.
func (s *Server) pinnedScript(rc *renderCtx, c *catalog.Component) (script, sri string, err error) {
	sri, err = rc.r.FileIntegrity(rc.ctx, c.Slug, c.Hash, catalog.MinPath(strings.TrimPrefix(c.Script, c.Slug+"/")))
	if err != nil || sri == "" {
		return c.VersionedScript(), c.Integrity, err
	}
	return c.VersionedMinScript(), sri, nil
}

func (s *Server) themesPage(rc *renderCtx) (view, error) {
	theme := rc.tab.PreviewTheme
	if !model.ValidPreviewTheme(theme) {
		theme = "deep-space"
	}
	return view{
		Title:       "Themes · Starbase",
		Description: "Every component is styled with semantic design tokens. Swap the token set, restyle everything.",
		Nav:         "themes",
		Body: func(ui.Shell) templ.Component {
			return ui.ThemesPage(ui.ThemesView{Theme: theme, Smooth: rc.tab.PreviewSmooth, CSS: s.themeCSS(theme), CSSHTML: catalog.Highlight(s.themeCSS(theme), "css"), Previews: s.themePreviews()})
		},
	}, nil
}

// contentPage renders content/<name>.md, optionally after an extra block.
func (s *Server) contentPage(name, nav string, extra ...templ.Component) pageFunc {
	return func(rc *renderCtx) (view, error) {
		src, err := fs.ReadFile(s.content, name+".md")
		if err != nil {
			return view{}, errNotFound
		}
		var meta struct {
			Title       string `yaml:"title"`
			Lede        string `yaml:"lede"`
			Description string `yaml:"description"`
		}
		html, err := catalog.RenderMarkdown(src, &meta)
		if err != nil {
			return view{}, err
		}
		return view{
			Title:       meta.Title + " · Starbase",
			Description: meta.Description,
			Nav:         nav,
			Body: func(ui.Shell) templ.Component {
				var x templ.Component
				if len(extra) > 0 {
					x = extra[0]
				}
				return ui.ContentPage(meta.Title, meta.Lede, html, x)
			},
		}, nil
	}
}

// showcasePage: Mission Control (a client island fed by /demo/telemetry)
// and the multiplayer pixel board (server-owned, re-rendered every frame).
func (s *Server) showcasePage(rc *renderCtx) (view, error) {
	v, err := s.contentPage("showcase", "showcase")(rc)
	if err != nil {
		return view{}, err
	}
	cells, pixels, err := s.boardState(rc.ctx, rc.r)
	if err != nil {
		return view{}, err
	}
	src, _ := fs.ReadFile(s.content, "showcase.md")
	var meta struct {
		Title string `yaml:"title"`
		Lede  string `yaml:"lede"`
	}
	html, err := catalog.RenderMarkdown(src, &meta)
	if err != nil {
		return view{}, err
	}
	board := ui.BoardView{Cells: cells, Size: commands.BoardSize, Pixels: pixels, Viewers: max(1, s.hub.Count("/showcase"))}
	v.Body = func(ui.Shell) templ.Component {
		return ui.ContentPage(meta.Title, meta.Lede, html, ui.Showcase(board, rc.tab.Flight.OrDefault()))
	}
	return v, nil
}
