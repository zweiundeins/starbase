package web

import (
	"io/fs"
	"regexp"
	"strings"

	"starbase/internal/ui"
	"starbase/static"
)

var themeBlockRe = regexp.MustCompile(`(?s)\[data-sb-theme="([a-z-]+)"\]\s*\{(.*?)\n\t\}`)

// themeCSS extracts a theme's token overrides from the stylesheets, so the
// Themes page shows exactly the CSS that is live.
func (s *Server) themeCSS(theme string) string {
	for _, file := range []string{"css/themes/showcase.css", "css/theme.css"} {
		b, err := fs.ReadFile(static.FS, file)
		if err != nil {
			continue
		}
		for _, m := range themeBlockRe.FindAllStringSubmatch(string(b), -1) {
			if m[1] == theme {
				lines := strings.Split(strings.TrimSpace(m[2]), "\n")
				for i, l := range lines {
					lines[i] = "  " + strings.TrimSpace(l)
				}
				return `[data-sb-theme="` + theme + `"] {` + "\n" + strings.Join(lines, "\n") + "\n}"
			}
		}
	}
	return ""
}

// themePreviews picks a representative set of component previews for the
// Themes page stage.
func (s *Server) themePreviews() []ui.ThemePreview {
	var out []ui.ThemePreview
	for _, slug := range []string{"button", "input", "slider", "alert", "toggle", "tabs", "card", "tooltip", "modal"} {
		if c, ok := s.catalog.Get(slug); ok {
			html := c.Preview
			if slug == "tooltip" { // the gallery's forced-open tooltip would cover the cell title
				html = `<sb-tooltip content="More info" placement="right" open><img src="/art/info.svg" alt="Info" width="32" height="32" style="image-rendering: pixelated"></sb-tooltip>`
			}
			out = append(out, ui.ThemePreview{Name: c.Name, HTML: html})
		}
	}
	return out
}
