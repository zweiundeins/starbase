// Package ui renders every view with templ. Views are pure functions of the
// read model: the same component renders the initial GET and every frame
// pushed over the SSE render stream.
package ui

import (
	"encoding/json"
	"net/url"
	"strconv"

	"github.com/a-h/templ"

	"starbase/internal/model"
)

// Assets resolves cache-busted URLs.
type Assets interface {
	Static(name string) string // files under static/
	Art(name string) string    // generated pixel art (internal/pixelart)
	Components() string        // module that loads every community component
	Datastar() string          // the vendored datastar-rocket bundle
	AutoTheme() string         // CSS: the light theme for "auto" on light systems
}

// Shell is the page chrome around a view.
type Shell struct {
	Title       string
	Description string
	Nav         string // key of the active primary nav item
	User        *model.User
	Search      string // header search value
	SearchLive  bool   // header search filters the gallery in place
	URL         string // canonical URL of the current state (history.replaceState)
	Signals     string // initial page signals (JS object literal)
	LoginURL    string
	RepoURL     string
	BaseURL     string
	Assets      Assets
	Nonce       string
	Boot        string // server boot id, for dev live reload
	Dev         bool
	// ManifestTags lists every component tag; in dev the page publishes
	// their Rocket manifests to the server once they are defined.
	ManifestTags []string
	Version      string
	// Theme is the site theme from the sb-theme cookie: "auto" or a theme
	// slug. The server renders it, so there is no flash of another theme.
	Theme string
}

type navItem struct{ Key, Label, Href string }

var navItems = []navItem{
	{"components", "Components", "/"},
	{"playground", "Playground", "/playground"},
	{"themes", "Themes", "/themes"},
	{"showcase", "Showcase", "/showcase"},
	{"contribute", "Contribute", "/contribute"},
	{"about", "About", "/about"},
}

var stylesheets = []string{
	"css/tokens.css",
	"css/theme.css",
	"css/themes/showcase.css",
	"css/reset.css",
	"css/base.css",
	"css/layout.css",
	"css/site.css",
	"css/code.css",
	"css/utilities.css",
}

func jsString(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

func importMap(a Assets) string {
	b, _ := json.Marshal(map[string]any{"imports": map[string]string{"datastar": a.Datastar()}})
	return string(b)
}

// streamInit opens this tab's render stream: a long-lived POST to the page's
// own URL. retry 'always' reconnects even when the stream ends cleanly (a
// server restart or deploy closes it without an error); hidden tabs release
// the connection and re-open (with a fresh render) when visible again.
const streamInit = `@post(location.pathname + location.search, {retry: 'always', retryMaxCount: Infinity, retryInterval: 400, retryMaxWait: 4000, openWhenHidden: false})`

// searchFromAnywhere sends typing on non-gallery pages to the gallery.
const searchFromAnywhere = `el.value.trim() && (window.location.href = '/?q=' + encodeURIComponent(el.value.trim()))`

func avatar(handle string) string {
	return "https://github.com/" + url.PathEscape(handle) + ".png?size=64"
}

func itoa(n int) string { return strconv.Itoa(n) }

func safe(s string) templ.SafeURL { return templ.SafeURL(s) }

// PageSignals builds the body's initial signals: a per-tab id plus extra
// page-owned signals.
func PageSignals(extra map[string]any) string {
	s := `{tabid: crypto.randomUUID().replaceAll('-', '').slice(0, 16)`
	keys := make([]string, 0, len(extra))
	for k := range extra {
		keys = append(keys, k)
	}
	sortStrings(keys)
	for _, k := range keys {
		b, _ := json.Marshal(extra[k])
		s += ", " + k + ": " + string(b)
	}
	return s + "}"
}

// manifestScript (dev only) waits for every component to be defined, then
// posts Rocket's manifest document to the dev server.
func manifestScript(s Shell) string {
	tags, _ := json.Marshal(s.ManifestTags)
	return `<script type="module" nonce="` + s.Nonce + `">
import { publishRocketManifests } from 'datastar'
const tags = ` + string(tags) + `
await Promise.all(tags.map((t) => customElements.whenDefined(t)))
publishRocketManifests({ endpoint: '/dev/manifests' })
</script>`
}

func lightTheme(t string) bool { return model.LightThemes[t] }

// colorScheme is the <meta name="color-scheme"> for the site theme.
func colorScheme(theme string) string {
	switch {
	case theme == "auto":
		return "dark light"
	case model.LightThemes[theme]:
		return "light"
	}
	return "dark"
}

// siteThemes is the header switch's list: auto, then every theme.
func siteThemes() string {
	names := []string{"auto"}
	for _, t := range model.PreviewThemes {
		names = append(names, t.Slug)
	}
	b, _ := json.Marshal(names)
	return string(b)
}
