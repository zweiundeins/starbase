package web

import (
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
	for _, c := range res.Cards {
		if comp, ok := s.catalog.Get(c.Slug); ok {
			previews[c.Slug] = comp.Preview
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
			return ui.Gallery(sh, ui.GalleryView{Browse: b, Result: res, Previews: previews})
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
	install := s.installSnippet(comp)
	return view{
		Title:       card.Name + " · Starbase",
		Description: card.Summary,
		Nav:         "components",
		Body: func(sh ui.Shell) templ.Component {
			return ui.ComponentPage(sh, ui.ComponentView{
				Card:       *card,
				Component:  comp,
				Playground: comp.Playground(),
				Install:    catalog.Highlight(install, "html"),
				InstallRaw: install,
				EditURL:    strings.TrimSuffix(s.cfg.RepoURL, "/") + "/tree/main/components/" + slug,
			})
		},
	}, nil
}

func (s *Server) installSnippet(c *catalog.Component) string {
	base := strings.TrimSuffix(s.cfg.BaseURL, "/")
	return fmt.Sprintf(`<script type="importmap">
  { "imports": { "datastar": "https://cdn.jsdelivr.net/gh/starfederation/datastar@v1.0.4/bundles/datastar-rocket.js" } }
</script>
<script type="module" src="%s/c/%s"></script>

%s`, base, c.Script, strings.TrimSpace(c.Preview))
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
			return ui.ThemesPage(ui.ThemesView{Theme: theme, CSS: s.themeCSS(theme), CSSHTML: catalog.Highlight(s.themeCSS(theme), "css"), Previews: s.themePreviews()})
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
		return ui.ContentPage(meta.Title, meta.Lede, html, ui.Showcase(board))
	}
	return v, nil
}
