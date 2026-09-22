package web

import (
	"encoding/xml"
	"fmt"
	"net/http"
	"strings"

	"starbase/internal/pixelart"
	"starbase/internal/queries"
)

// robots allows everything but commands, sign-in and the playground's
// sandbox and previews; user snippets (?s=) and PR previews aren't content.
func (s *Server) robots(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	fmt.Fprintf(w, `User-agent: *
Disallow: /cmd/
Disallow: /auth/
Disallow: /dev/
Disallow: /playground/run
Disallow: /playground/preview/
Disallow: /playground?s=
Disallow: /playground?preview=
Disallow: /submit

Sitemap: %s/sitemap.xml
`, strings.TrimSuffix(s.cfg.BaseURL, "/"))
}

type sitemapURL struct {
	Loc     string `xml:"loc"`
	LastMod string `xml:"lastmod,omitempty"`
}

// sitemap lists the pages and every component page.
func (s *Server) sitemap(w http.ResponseWriter, r *http.Request) {
	var comps []queries.SitemapEntry
	if err := s.q.View(r.Context(), func(rd *queries.Reader) (err error) {
		comps, err = rd.SitemapComponents(r.Context())
		return
	}); err != nil {
		s.fail(w, r, err)
		return
	}
	base := strings.TrimSuffix(s.cfg.BaseURL, "/")
	urls := []sitemapURL{}
	for _, p := range []string{"/", "/playground", "/themes", "/showcase", "/contribute", "/about"} {
		urls = append(urls, sitemapURL{Loc: base + p})
	}
	for _, c := range comps {
		urls = append(urls, sitemapURL{Loc: base + "/components/" + c.Slug, LastMod: c.Updated.Format("2006-01-02")})
	}
	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	w.Write([]byte(xml.Header))
	enc := xml.NewEncoder(w)
	enc.Indent("", "  ")
	enc.Encode(struct {
		XMLName xml.Name     `xml:"urlset"`
		NS      string       `xml:"xmlns,attr"`
		URLs    []sitemapURL `xml:"url"`
	}{NS: "http://www.sitemaps.org/schemas/sitemap/0.9", URLs: urls})
}

// Raster images, rendered once from the pixel art (see pixelart/png.go).
var (
	socialPNG  = pixelart.SocialImage()
	iconPNG    = pixelart.Icon(180)
	faviconPNG = pixelart.Icon(32)
)

func servePNG(b []byte) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.Header().Set("Cache-Control", "public, max-age=86400")
		w.Write(b)
	}
}
