// Package ui renders every view with templ. Views are pure functions of the
// read model: the same component renders the initial GET and every frame
// pushed over the SSE render stream.
package ui

import (
	"encoding/json"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/a-h/templ"

	"starbase/internal/model"
)

// Assets resolves cache-busted URLs.
type Assets interface {
	Static(name string) string // files under static/
	Art(name string) string    // generated pixel art (internal/pixelart)
	Components() string        // the autoloader: loads each component the page uses
	Bundle() string            // every component in one module (see web.bundleBudget)
	AllComponents() string     // imports every component, listed or not (the dev manifest publisher)
	Datastar() string          // the vendored datastar-rocket bundle
	SiteCSS() string           // all the site's CSS in one file (see web/assets.go)
	// ArtSVG is a pixel art illustration as inline SVG markup, so its
	// theme-aware colours (--sb-art-*) follow the page's theme.
	ArtSVG(name string) string
	ArtSize(name string) (w, h int) // intrinsic size, for width/height attributes
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
	LoadBundle  bool // load every component as one file (default while it fits its budget) instead of the autoloader
	// ManifestTags lists every component tag; in dev the page publishes
	// their Rocket manifests to the server once they are defined.
	ManifestTags []string
	Version      string
	// Theme is the site theme from the sb-theme cookie: "auto" or a theme
	// slug. The server renders it, so there is no flash of another theme.
	Theme string
	// For search engines and link previews.
	Canonical   string // absolute URL of the page, without query
	SocialImage string // absolute URL of the Open Graph image
	JSONLD      string // schema.org JSON-LD
	// Heartbeat is how often the render stream patches $_ping (Watchdog).
	Heartbeat time.Duration
	// Acks is the tab's acked ops as JSON ({key: op id}), which
	// static/js/commands.js reads from #ops after every frame.
	Acks string
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

func jsString(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

func importMap(a Assets) string {
	b, _ := json.Marshal(map[string]any{"imports": map[string]string{"datastar": a.Datastar()}})
	return string(b)
}

// StreamID is the element that opens the tab's render stream, and the
// stream's live.Spec.Reconnect: on a deploy, the stream's last event copies
// the element, and the copy's data-init opens the stream on the next process.
const StreamID = "stream"

// OpsID is the hidden element in #app whose data-acks every frame renders.
const OpsID = "ops"

// streamInit opens this tab's render stream: a long-lived POST to the page's
// own URL. retry 'error' keeps asking through a restart's 502s and 503s, and
// a stream the server ends stays closed; hidden tabs release the connection
// and re-open (with a fresh render) when visible again. Accepting only an
// event stream marks the request, so a refusal for good can answer it with
// 204, which 'error' does not repeat (it repeats every 4xx).
const streamInit = `@post(location.pathname + location.search, {retry: 'error', retryMaxCount: Infinity, retryInterval: 400, retryMaxWait: 4000, openWhenHidden: false, headers: {Accept: 'text/event-stream'}})`

// streamRetrying holds the watchdog off while Datastar retries the stream,
// as it does for a tab the budget refuses: re-creating #stream would restart the backoff.
const streamRetrying = `evt.detail.el === el && evt.detail.type === 'retrying' && ($_quiet = 0)`

// PingSignal is the signal the render stream's heartbeat patches
// (live.Options.HeartbeatSignal), with a number that grows.
const PingSignal = "_ping"

// QuietSignal counts the watchdog's ticks without a new ping; a stream's
// first write resets it, since its first ping comes a heartbeat later.
const QuietSignal = "_quiet"

// watchdogTick runs once a heartbeat and re-creates #stream, as sse.Reconnect
// does, after 3 to 4 heartbeats without a ping (a hidden tab's stream is closed on purpose).
const watchdogTick = `$_quiet = document.hidden || $_ping !== $_seen ? 0 : $_quiet + 1; $_seen = $_ping; ` +
	`$_quiet > 2 && ($_quiet = 0, document.getElementById('` + StreamID + `').replaceWith(document.getElementById('` + StreamID + `').cloneNode(true)))`

// watchdogInterval is the data-on-interval attribute that ticks once a heartbeat.
func watchdogInterval(heartbeat time.Duration) string {
	return "data-on-interval__duration." + strconv.FormatInt(heartbeat.Milliseconds(), 10) + "ms"
}

// TelemetryID is the element that opens /demo/telemetry: the stream names it
// in its last event when the server stops, like StreamID.
const TelemetryID = "telemetry"

// telemetryInit opens Mission Control's telemetry, like the docs examples.
const telemetryInit = `@get('/demo/telemetry', {retry: 'error', retryMaxCount: Infinity, retryInterval: 400, retryMaxWait: 4000})`

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
	// getRandomValues, unlike randomUUID, also exists outside secure contexts (plain HTTP).
	s := `{tabid: Array.from(crypto.getRandomValues(new Uint8Array(8)), (b) => b.toString(16).padStart(2, '0')).join('')`
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

// Art inlines a pixel art illustration: decorative, with the given classes.
// It keeps its proportions, standing on the bottom of its box.
func Art(a Assets, name, class string) templ.Component {
	src := a.ArtSVG(name)
	key := artKey{name, class}
	if e, ok := inlineArt.Load(key); ok && e.(artEntry).src == src {
		return templ.Raw(e.(artEntry).svg)
	}
	svg := strings.Replace(src, `<svg xmlns="http://www.w3.org/2000/svg" class="px-art"`,
		`<svg xmlns="http://www.w3.org/2000/svg" class="px-art `+class+`" aria-hidden="true" focusable="false" preserveAspectRatio="xMidYMax meet"`, 1)
	inlineArt.Store(key, artEntry{src, svg})
	return templ.Raw(svg)
}

// inlineArt keeps Art's markup per illustration and classes (a handful of
// call sites), checked against the source it was made from.
var inlineArt sync.Map // artKey → artEntry

type artKey struct{ name, class string }

type artEntry struct{ src, svg string }

func onOff(b bool) string {
	if b {
		return "on"
	}
	return "off"
}

// command posts a command to path as an op of key (a JavaScript expression,
// the command's key in TabState.Ops) through sbCommand
// (static/js/commands.js): the op id goes in the Op-Id header, and opts
// (JavaScript object members, or "") are the rest of @post's options.
func command(key, path, opts string) string {
	if opts != "" {
		opts = ", " + opts
	}
	return "sbCommand(el, " + key + ", (op) => @post('" + path + "', {headers: {'Op-Id': op}" + opts + "}))"
}

// browseCommand posts the gallery's search and category.
var browseCommand = command("'browse'", "/cmd/browse", "")
