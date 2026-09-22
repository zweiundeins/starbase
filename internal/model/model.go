// Package model holds types shared by the command and query sides.
package model

import (
	"net/url"
	"strings"
	"unicode/utf8"

	"starbase/internal/catalog"
)

type Sort string

const (
	SortPopular Sort = "popular"
	SortNewest  Sort = "newest"
	SortName    Sort = "name"
)

var Sorts = []struct {
	Value Sort
	Label string
}{
	{SortPopular, "Most popular"},
	{SortNewest, "Newest"},
	{SortName, "Name"},
}

func (s Sort) Valid() bool {
	return s == SortPopular || s == SortNewest || s == SortName
}

// Browse is the gallery's filter state. It lives in tab_state and mirrors
// the page's query string so URLs stay shareable.
type Browse struct {
	Q        string `json:"q"`
	Category string `json:"cat"`
	Sort     Sort   `json:"sort"`
}

const MaxQueryLen = 80

// Normalize clamps every field into a valid value.
func (b Browse) Normalize() Browse {
	b.Q = strings.TrimSpace(b.Q)
	for utf8.RuneCountInString(b.Q) > MaxQueryLen {
		_, size := utf8.DecodeLastRuneInString(b.Q)
		b.Q = b.Q[:len(b.Q)-size]
	}
	if _, ok := catalog.CategoryBySlug(b.Category); !ok {
		b.Category = ""
	}
	if !b.Sort.Valid() {
		b.Sort = SortPopular
	}
	return b
}

func BrowseFromQuery(v url.Values) Browse {
	return Browse{Q: v.Get("q"), Category: v.Get("cat"), Sort: Sort(v.Get("sort"))}.Normalize()
}

// Query encodes the non-default fields as a query string.
func (b Browse) Query() url.Values {
	v := url.Values{}
	if b.Q != "" {
		v.Set("q", b.Q)
	}
	if b.Category != "" {
		v.Set("cat", b.Category)
	}
	if b.Sort != "" && b.Sort != SortPopular {
		v.Set("sort", string(b.Sort))
	}
	return v
}

// TabState is everything the server remembers about one browser tab.
type TabState struct {
	Browse       Browse `json:"browse"`
	PreviewTheme string `json:"previewTheme,omitempty"`
	// PreviewSmooth shows the Themes page previews without 8-bit details.
	PreviewSmooth bool `json:"smooth,omitempty"`
	// PlaygroundShare is the id of the snippet this tab saved last.
	PlaygroundShare string `json:"share,omitempty"`
}

var PreviewThemes = []struct {
	Slug, Label, Blurb string
}{
	{"deep-space", "Deep Space", "The default. Night-sky navy, violet signal, Datastar green."},
	{"nebula", "Nebula", "Warm magenta dust clouds with a peach glow."},
	{"terminal", "Terminal", "Phosphor green on black, like a mission console."},
	{"daylight", "Daylight", "A light theme for documentation-heavy apps."},
}

// ValidSiteTheme reports whether s is a theme the site itself can wear:
// "auto" (the system's dark or light) or one of the preview themes.
func ValidSiteTheme(s string) bool { return s == "auto" || ValidPreviewTheme(s) }

// LightThemes are the themes with a light color scheme.
var LightThemes = map[string]bool{"daylight": true}

func ValidPreviewTheme(s string) bool {
	for _, t := range PreviewThemes {
		if t.Slug == s {
			return true
		}
	}
	return false
}

type User struct {
	ID        int64
	GitHubID  int64
	Login     string
	Name      string
	AvatarURL string
}
