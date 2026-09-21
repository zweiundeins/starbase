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
	static     *hashfs.FS
	art        map[string]generated
	catalog    *catalog.Catalog
	components generated // /c/index.js: imports every component module
	dev        bool
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
	return a
}

func (a *assets) Static(name string) string { return "/static/" + a.static.HashName(name) }
func (a *assets) Datastar() string          { return a.Static("vendor/datastar-rocket.js") }
func (a *assets) Components() string        { return "/c/index.js?v=" + a.components.hash }

func (a *assets) Art(name string) string {
	return "/art/" + name + ".svg?v=" + a.art[name].hash
}

func (a *assets) ComponentScript(c *catalog.Component) string {
	return "/c/" + c.Script + "?v=" + c.Hash
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

func (a *assets) serveStatic() http.Handler {
	h := http.StripPrefix("/static/", hashfs.FileServer(a.static))
	if !a.dev {
		return h
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h.ServeHTTP(noStoreWriter{w}, r)
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
	a.cacheHeader(w, r.URL.Query().Get("v") == g.hash)
	w.Write(g.body)
}

// serveComponents serves /c/index.js and /c/<slug>/<file>.js from the
// catalog; nothing else in the component folders is public.
func (a *assets) serveComponents(w http.ResponseWriter, r *http.Request) {
	p := r.PathValue("path")
	if p == "index.js" {
		w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
		a.cacheHeader(w, r.URL.Query().Get("v") == a.components.hash)
		w.Write(a.components.body)
		return
	}
	slug, file, ok := strings.Cut(p, "/")
	c, found := a.catalog.Get(slug)
	if !ok || !found || slug+"/"+file != c.Script {
		http.NotFound(w, r)
		return
	}
	b, err := fs.ReadFile(a.catalog.FS, c.Script)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
	w.Header().Set("Access-Control-Allow-Origin", "*") // installable from other sites
	a.cacheHeader(w, r.URL.Query().Get("v") == c.Hash)
	w.Write(b)
}
