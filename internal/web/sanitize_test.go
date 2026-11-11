package web_test

import (
	"encoding/xml"
	"net/url"
	"strings"
	"testing"

	"starbase/internal/catalog"
	"starbase/internal/model"
)

// TestNoFailedSanitization renders every page and fails if templ replaced an
// attribute value it took for a URL with about:invalid#TemplFailedSanitizationURL.
func TestNoFailedSanitization(t *testing.T) {
	ts, c, _, cat := newServerWriter(t)
	_, sm := get(t, c, ts.URL+"/sitemap.xml")
	var set struct {
		URLs []struct {
			Loc string `xml:"loc"`
		} `xml:"url"`
	}
	if err := xml.Unmarshal([]byte(sm), &set); err != nil || len(set.URLs) == 0 {
		t.Fatalf("sitemap: %v", err)
	}
	pages := []string{"/?q=button", "/no-such-page", "/components/no-such-component"}
	for _, u := range set.URLs {
		pages = append(pages, strings.TrimPrefix(u.Loc, ts.URL))
	}
	for _, comp := range cat.Components { // unlisted ones too, and each in the playground
		pages = append(pages, "/components/"+comp.Slug, "/playground?component="+url.QueryEscape(comp.Slug))
	}
	for _, cg := range catalog.Categories {
		for _, s := range model.Sorts {
			pages = append(pages, "/?cat="+cg.Slug+"&sort="+string(s.Value))
		}
	}
	for _, p := range pages {
		res, body := get(t, c, ts.URL+p)
		if res.StatusCode >= 500 {
			t.Errorf("%s: %d", p, res.StatusCode)
		}
		if i := strings.Index(body, "TemplFailedSanitization"); i >= 0 {
			t.Errorf("%s: %s", p, body[max(0, i-200):i])
		}
	}
}
