package web

import (
	"path"

	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/tdewolff/minify/v2"
	mincss "github.com/tdewolff/minify/v2/css"
	"io/fs"
	"log/slog"
	"net/http"
	"regexp"
	"starbase/internal/precompress"
	"strconv"
	"strings"

	"github.com/benbjohnson/hashfs"

	"starbase/internal/catalog"
	"starbase/internal/pixelart"
)

// assets serves static files under content-hashed names, the generated pixel
// art and the community component modules. It implements ui.Assets.
type assets struct {
	static      *hashfs.FS
	art         map[string]generated
	catalog     *catalog.Catalog
	components  generated            // /c/index.js: imports every component module
	bundle      generated            // /c/bundle.js: every component in one file
	bundleFits  bool                 // the bundle is within bundleBudget: pages load it
	autoloader  generated            // /c/autoloader.js: loads <sb-*> modules on first use
	autoTheme   generated            // /theme/auto.css: daylight's tokens for "auto" on light systems
	datastarSRI string               // of the vendored bundle, identical to the jsDelivr release
	bundles     map[string]generated // /bundle/<name>.css: stylesheets in one file, minified
	dev         bool
}

type generated struct {
	body []byte
	hash string
}

func hashOf(b []byte) string {
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])[:12]
}

func newAssets(staticFS fs.FS, cat *catalog.Catalog, dev bool) *assets {
	a := &assets{static: hashfs.NewFS(staticFS), art: map[string]generated{}, catalog: cat, dev: dev}
	for name, svg := range pixelart.All() {
		a.art[name] = generated{body: []byte(svg), hash: hashOf([]byte(svg))}
	}
	var js strings.Builder
	// The readable modules: the manifest publisher needs their .docs() and
	// manifest: blocks, which the .min files and the bundle leave out.
	js.WriteString("// Generated: loads every community component (readable modules).\n")
	for _, c := range cat.Components {
		fmt.Fprintf(&js, "import %q\n", "/c/"+c.VersionedScript())
	}
	a.components = generated{body: []byte(js.String()), hash: hashOf([]byte(js.String()))}
	if b, err := cat.Bundle(); err == nil {
		a.bundle = generated{body: b, hash: hashOf(b)}
		br, _ := precompress.Get(b)
		a.bundleFits = len(br) <= bundleBudget
		if !a.bundleFits {
			slog.Warn("component bundle over budget: pages load the autoloader", "brotli", len(br), "budget", bundleBudget)
		}
	} else {
		slog.Warn("component bundle", "err", err) // pages load the autoloader
	}
	auto := catalog.AutoloaderJS(cat, "")
	a.autoloader = generated{body: []byte(auto), hash: hashOf([]byte(auto))}
	ds, _ := fs.ReadFile(staticFS, "vendor/datastar-rocket.js")
	a.datastarSRI = catalog.SRI(ds)
	css := autoThemeCSS(staticFS)
	a.autoTheme = generated{body: []byte(css), hash: hashOf([]byte(css))}
	a.bundles = map[string]generated{
		"site.css":   a.bundleCSS(staticFS, siteStylesheets, css),
		"runner.css": a.bundleCSS(staticFS, runnerStylesheets, ""),
	}
	go a.warm() // after every field is set: it reads them concurrently
	return a
}

// autoThemeCSS: with no theme chosen ("auto"), the site is deep-space (the
// :root tokens), and on light systems the daylight tokens. They are copied
// from the daylight block, so the two can't drift apart.
func autoThemeCSS(staticFS fs.FS) string {
	b, _ := fs.ReadFile(staticFS, "css/themes/showcase.css")
	for _, m := range themeBlockRe.FindAllStringSubmatch(string(b), -1) {
		if m[1] == "daylight" {
			return "/* Generated from css/themes/showcase.css: the daylight tokens, for auto on light systems. */\n" +
				"@layer theme {\n\t@media (prefers-color-scheme: light) {\n\t\t:root:not([data-sb-theme]) {" + m[2] + "\n\t\t}\n\t}\n}\n"
		}
	}
	return ""
}

func (a *assets) AutoTheme() string { return "/theme/auto.css?v=" + a.autoTheme.hash }

// The site's stylesheets, in cascade order (they declare their @layer).
var siteStylesheets = []string{
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

// The playground runner's: tokens and themes for the components, no chrome.
var runnerStylesheets = []string{"css/tokens.css", "css/theme.css", "css/themes/showcase.css", "css/reset.css"}

var fontURLRe = regexp.MustCompile(`url\(\s*["']?\.\./fonts/([^"')]+)["']?\s*\)`)

// bundleCSS concatenates stylesheets into one minified file (one request,
// nothing render-blocking behind it). Font URLs become their hashed,
// immutable names.
func (a *assets) bundleCSS(staticFS fs.FS, files []string, extra string) generated {
	var b strings.Builder
	for _, f := range files {
		src, err := fs.ReadFile(staticFS, f)
		if err != nil {
			panic("bundle: " + err.Error()) // embedded: a missing file is a build error
		}
		b.Write(src)
		b.WriteByte('\n')
	}
	b.WriteString(extra)
	css := fontURLRe.ReplaceAllStringFunc(b.String(), func(m string) string {
		return `url("/static/` + a.static.HashName("fonts/"+fontURLRe.FindStringSubmatch(m)[1]) + `")`
	})
	m := minify.New()
	m.AddFunc("text/css", mincss.Minify)
	if out, err := m.String("text/css", css); err == nil {
		css = out
	}
	return generated{body: []byte(css), hash: hashOf([]byte(css))}
}

func (a *assets) SiteCSS() string   { return "/bundle/site.css?v=" + a.bundles["site.css"].hash }
func (a *assets) RunnerCSS() string { return "/bundle/runner.css?v=" + a.bundles["runner.css"].hash }

func (a *assets) serveBundle(w http.ResponseWriter, r *http.Request) {
	g, ok := a.bundles[r.PathValue("name")]
	if !ok {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/css; charset=utf-8")
	w.Header().Set("Access-Control-Allow-Origin", "*") // the runner's opaque origin loads it
	a.cacheHeader(w, r.URL.Query().Get("v") == g.hash)
	precompress.Write(w, r, g.body)
}

func (a *assets) serveAutoTheme(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/css; charset=utf-8")
	a.cacheHeader(w, r.URL.Query().Get("v") == a.autoTheme.hash)
	precompress.Write(w, r, a.autoTheme.body)
}

func (a *assets) Static(name string) string { return "/static/" + a.static.HashName(name) }
func (a *assets) Datastar() string          { return a.Static("vendor/datastar-rocket.js") }
func (a *assets) Components() string        { return "/c/autoloader.js?v=" + a.autoloader.hash }

// Bundle is every component in one module.
func (a *assets) Bundle() string { return "/c/bundle.js?v=" + a.bundle.hash }

// bundleBudget is how big (brotli) the one-file bundle may grow before pages
// go back to the autoloader. Measured on production (slow 4G, cold cache):
// at 66 kB the bundle makes a visit that includes the homepage 250-450 ms
// faster, and costs a visitor who lands on one component page about 265 ms;
// that cost grows with the bundle, the autoloader's doesn't. 100 kB keeps it
// under about 400 ms.
const bundleBudget = 100 << 10

// useBundle reports whether a page loads the one-file bundle (the default
// while it fits the budget) or the autoloader. ?load=auto and ?load=bundle
// override it, for measuring.
func (a *assets) useBundle(r *http.Request) bool {
	switch r.URL.Query().Get("load") {
	case "bundle":
		return a.bundle.body != nil
	case "auto":
		return false
	}
	return a.bundleFits
}

// AllComponents imports every component's readable module, listed or not:
// the dev manifest publisher needs them all, with their docs.
func (a *assets) AllComponents() string { return "/c/index.js?v=" + a.components.hash }

func (a *assets) ArtSVG(name string) string { return string(a.art[name].body) }

var svgSizeRe = regexp.MustCompile(`<svg[^>]* width="(\d+)" height="(\d+)"`)

func (a *assets) ArtSize(name string) (w, h int) {
	if m := svgSizeRe.FindSubmatch(a.art[name].body); m != nil {
		w, _ = strconv.Atoi(string(m[1]))
		h, _ = strconv.Atoi(string(m[2]))
	}
	return w, h
}

func (a *assets) Art(name string) string {
	return "/art/" + name + ".svg?v=" + a.art[name].hash
}

// ComponentScript is the component's versioned (immutable) module URL, the
// minified one: what the site itself loads.
func (a *assets) ComponentScript(c *catalog.Component) string {
	return "/c/" + c.VersionedMinScript()
}

const immutable = "public, max-age=31536000, immutable"

func (a *assets) cacheHeader(w http.ResponseWriter, versioned bool) {
	switch {
	case a.dev:
		w.Header().Set("Cache-Control", "no-store")
	case versioned:
		w.Header().Set("Cache-Control", immutable)
	default:
		w.Header().Set("Cache-Control", "public, max-age=300")
	}
}

// serveStatic serves the hashed static files. They are public, and the
// playground's sandboxed runner (opaque origin) loads them cross-origin, so
// they allow any origin.
func (a *assets) serveStatic() http.Handler {
	h := http.StripPrefix("/static/", hashfs.FileServer(a.static))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		switch _, hash := hashfs.ParseName(strings.TrimPrefix(r.URL.Path, "/static/")); {
		case a.dev:
			w = cacheWriter{w, "no-store"}
		case hash != "":
			w = cacheWriter{w, immutable} // the name changes with the content
		}
		// Text assets go out precompressed (brotli -11); fonts and images are
		// compressed formats already and keep the file server (ranges etc.).
		name := strings.TrimPrefix(r.URL.Path, "/static/")
		if ctype, ok := precompressedTypes[path.Ext(name)]; ok && fs.ValidPath(name) {
			if body, err := fs.ReadFile(a.static, name); err == nil {
				w.Header().Set("Content-Type", ctype)
				precompress.Write(w, r, body)
				return
			}
		}
		h.ServeHTTP(w, r)
	})
}

// precompressedTypes are the static files served precompressed.
var precompressedTypes = map[string]string{
	".js":   "text/javascript; charset=utf-8",
	".mjs":  "text/javascript; charset=utf-8",
	".css":  "text/css; charset=utf-8",
	".svg":  "image/svg+xml",
	".json": "application/json",
	".map":  "application/json",
	".txt":  "text/plain; charset=utf-8",
}

// cacheWriter overrides hashfs's Cache-Control: no-store in development,
// immutable for content-hashed names.
type cacheWriter struct {
	http.ResponseWriter
	value string
}

func (w cacheWriter) WriteHeader(code int) {
	if code == http.StatusOK {
		w.Header().Set("Cache-Control", w.value)
	}
	w.ResponseWriter.WriteHeader(code)
}

func (a *assets) serveArt(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimSuffix(r.PathValue("name"), ".svg")
	g, ok := a.art[name]
	if !ok {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "image/svg+xml")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	a.cacheHeader(w, r.URL.Query().Get("v") == g.hash)
	if !a.dev && r.URL.Query().Get("v") == "" {
		w.Header().Set("Cache-Control", "public, max-age=86400") // e.g. linked from docs; art rarely changes
	}
	precompress.Write(w, r, g.body)
}

// serveComponents serves /c/index.js and /c/<slug>/<file>.js from the
// catalog; nothing else in the component folders is public.
func (a *assets) serveComponents(w http.ResponseWriter, r *http.Request) {
	p := r.PathValue("path")
	if g, ok := map[string]generated{"index.js": a.components, "autoloader.js": a.autoloader, "bundle.js": a.bundle}[p]; ok && g.body != nil {
		w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
		w.Header().Set("Access-Control-Allow-Origin", "*") // usable from any site
		a.cacheHeader(w, r.URL.Query().Get("v") == g.hash)
		precompress.Write(w, r, g.body)
		return
	}
	// A component's own .js and .mjs files are public (its module, plus
	// anything it imports relatively, like vendored libraries); nothing else is.
	slug, file, ok := strings.Cut(p, "/")
	c, found := a.catalog.Get(slug)
	if !ok || !found || !(strings.HasSuffix(file, ".js") || strings.HasSuffix(file, ".mjs")) || !fs.ValidPath(p) {
		http.NotFound(w, r)
		return
	}
	b, err := fs.ReadFile(a.catalog.FS, p)
	// The unversioned route follows the current version's minified file. A
	// vendored file that is already minified (echarts.esm.min.js) has none: it
	// is its own, which the readable module imports from here.
	if catalog.IsMinPath(file) {
		if mins, merr := a.catalog.MinFiles(c); merr != nil || mins[file] != nil {
			b, err = mins[file], merr
		}
	}
	if err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
	w.Header().Set("Access-Control-Allow-Origin", "*") // installable from other sites and the sandbox
	a.cacheHeader(w, r.URL.Query().Get("v") == c.Hash)
	precompress.Write(w, r, b)
}

// warm precompresses the generated and static text assets in the
// background, one at a time (the service runs under a tight memory limit), so
// the first request after a deploy doesn't wait for brotli -11. The component
// modules are warm already: the catalog's size tables compressed them.
func (a *assets) warm() {
	for _, g := range []generated{a.components, a.autoloader, a.bundle, a.autoTheme} {
		if g.body != nil {
			precompress.Get(g.body)
		}
	}
	for _, g := range a.bundles {
		precompress.Get(g.body)
	}
	for _, g := range a.art {
		precompress.Get(g.body)
	}
	fs.WalkDir(a.static, ".", func(p string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			if _, ok := precompressedTypes[path.Ext(p)]; ok {
				if b, err := fs.ReadFile(a.static, p); err == nil {
					precompress.Get(b)
				}
			}
		}
		return nil
	})
}
