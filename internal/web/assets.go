package web

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"net/http"
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
	components  generated // /c/index.js: imports every component module
	autoloader  generated // /c/autoloader.js: loads <sb-*> modules on first use
	autoTheme   generated // /theme/auto.css: daylight's tokens for "auto" on light systems
	datastarSRI string    // of the vendored bundle, identical to the jsDelivr release
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
	js.WriteString("// Generated: loads every community component.\n")
	for _, c := range cat.Components {
		fmt.Fprintf(&js, "import %q\n", a.ComponentScript(c))
	}
	a.components = generated{body: []byte(js.String()), hash: hashOf([]byte(js.String()))}
	auto := catalog.AutoloaderJS(cat, "")
	a.autoloader = generated{body: []byte(auto), hash: hashOf([]byte(auto))}
	ds, _ := fs.ReadFile(staticFS, "vendor/datastar-rocket.js")
	a.datastarSRI = catalog.SRI(ds)
	css := autoThemeCSS(staticFS)
	a.autoTheme = generated{body: []byte(css), hash: hashOf([]byte(css))}
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

func (a *assets) serveAutoTheme(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/css; charset=utf-8")
	a.cacheHeader(w, r.URL.Query().Get("v") == a.autoTheme.hash)
	w.Write(a.autoTheme.body)
}

func (a *assets) Static(name string) string { return "/static/" + a.static.HashName(name) }
func (a *assets) Datastar() string          { return a.Static("vendor/datastar-rocket.js") }
func (a *assets) Components() string        { return "/c/autoloader.js?v=" + a.autoloader.hash }

// AllComponents is the module that imports every component (the gallery
// and the dev manifest publisher need them all at once).
func (a *assets) AllComponents() string { return "/c/index.js?v=" + a.components.hash }

func (a *assets) Art(name string) string {
	return "/art/" + name + ".svg?v=" + a.art[name].hash
}

// ComponentScript is the component's versioned (immutable) module URL.
func (a *assets) ComponentScript(c *catalog.Component) string {
	return "/c/" + c.VersionedScript()
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
		if a.dev {
			w = noStoreWriter{w}
		}
		h.ServeHTTP(w, r)
	})
}

// noStoreWriter overrides hashfs's immutable caching in development.
type noStoreWriter struct{ http.ResponseWriter }

func (w noStoreWriter) WriteHeader(code int) {
	w.Header().Set("Cache-Control", "no-store")
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
	w.Write(g.body)
}

// serveComponents serves /c/index.js and /c/<slug>/<file>.js from the
// catalog; nothing else in the component folders is public.
func (a *assets) serveComponents(w http.ResponseWriter, r *http.Request) {
	p := r.PathValue("path")
	if g, ok := map[string]generated{"index.js": a.components, "autoloader.js": a.autoloader}[p]; ok {
		w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
		w.Header().Set("Access-Control-Allow-Origin", "*") // usable from any site
		a.cacheHeader(w, r.URL.Query().Get("v") == g.hash)
		w.Write(g.body)
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
	if err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
	w.Header().Set("Access-Control-Allow-Origin", "*") // installable from other sites and the sandbox
	a.cacheHeader(w, r.URL.Query().Get("v") == c.Hash)
	w.Write(b)
}
