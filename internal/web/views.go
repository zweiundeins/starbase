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
	_, snapshotSRI, _, err := rc.r.Snapshot(rc.ctx, s.catalog.Hash)
	if err != nil {
		return view{}, err
	}
	install := s.installSnippet(comp, snapshotSRI)
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
				Install:    catalog.Highlight(install, "html"),
				InstallRaw: install,
				EditURL:    strings.TrimSuffix(s.cfg.RepoURL, "/") + "/tree/main/components/" + slug,
			})
		},
	}, nil
}

// datastarCDN is the Datastar + Rocket bundle the snippets load. It is
// byte-identical to static/vendor/datastar-rocket.js (so its SRI is ours).
const datastarCDN = "https://cdn.jsdelivr.net/gh/starfederation/datastar@v1.0.4/bundles/datastar-rocket.js"

func (s *Server) installSnippet(c *catalog.Component, snapshotSRI string) string {
	base := strings.TrimSuffix(s.cfg.BaseURL, "/")
	pinned := ""
	if snapshotSRI != "" {
		pinned = fmt.Sprintf(`

<!-- In production, pin today's catalog instead of the latest: the browser then
     refuses any file that changed. Add "integrity" to the import map above: the
     hashes from %[1]s/c/@%[2]s/importmap.json and Datastar's, below. -->
<!--
<script type="importmap">
  { "imports": { "datastar": "%[4]s" },
    "integrity": { "%[4]s": "%[5]s", "…": "…from importmap.json" } }
</script>
<script type="module" src="%[1]s/c/@%[2]s/autoloader.js" integrity="%[3]s"></script>
-->`, base, s.catalog.Hash, snapshotSRI, datastarCDN, s.assets.datastarSRI)
	}
	return fmt.Sprintf(`<!-- Once per page: Datastar with Rocket, and the Starbase autoloader.
     It loads every <sb-…> component the first time its tag appears. -->
<script type="importmap">
  { "imports": { "datastar": "%[5]s" } }
</script>
<script type="module" src="%[1]s/c/autoloader.js"></script>
<!-- Optional, no flash of undefined elements: class="sb-cloak" on <html>, and -->
<style>.sb-cloak :not(:defined) { visibility: hidden }</style>

%[2]s%[6]s

<!-- Or load just this component, pinned to this version: -->
<!-- <script type="module" src="%[1]s/c/%[3]s" integrity="%[4]s"></script> -->`,
		base, strings.TrimSpace(c.Preview), c.VersionedScript(), c.Integrity, datastarCDN, pinned)
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
		return ui.ContentPage(meta.Title, meta.Lede, html, ui.Showcase(board))
	}
	return v, nil
}
