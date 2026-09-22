package web

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"regexp"
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
	autoloader generated // /c/autoloader.js: loads <sb-*> modules on first use
	autoTheme  generated // /theme/auto.css: daylight's tokens for "auto" on light systems
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
	auto := autoloaderJS(cat)
	a.autoloader = generated{body: []byte(auto), hash: hashOf([]byte(auto))}
	css := autoThemeCSS(staticFS)
	a.autoTheme = generated{body: []byte(css), hash: hashOf([]byte(css))}
	return a
}

var usesTagRe = regexp.MustCompile(`<(sb-[a-z0-9]+(?:-[a-z0-9]+)*)`)

// autoloaderJS generates the autoloader: a map from every tag to its module
// (relative to the autoloader, so it works from other sites too) and the
// components each one renders (found as <sb-… in its source).
func autoloaderJS(cat *catalog.Catalog) string {
	modules := map[string]string{}
	requires := map[string][]string{}
	for _, c := range cat.Components {
		modules[c.Tag] = c.Script + "?v=" + c.Hash
	}
	for _, c := range cat.Components {
		src, _ := fs.ReadFile(cat.FS, c.Script)
		seen := map[string]bool{c.Tag: true}
		for _, m := range usesTagRe.FindAllStringSubmatch(string(src), -1) {
			if _, ok := modules[m[1]]; ok && !seen[m[1]] {
				seen[m[1]] = true
				requires[c.Tag] = append(requires[c.Tag], m[1])
			}
		}
	}
	mj, _ := json.Marshal(modules)
	rj, _ := json.Marshal(requires)
	return `// Starbase autoloader (generated). Loads each <sb-*> component the first
// time its tag appears, including tags added later (e.g. by a Datastar
// morph). Components import 'datastar': the page needs an import map for it.
//
//   <script type="importmap">{"imports": {"datastar": "…/datastar-rocket.js"}}</script>
//   <script type="module" src="…/c/autoloader.js"></script>
//
// Optional, against the flash of undefined elements: put class="sb-cloak" on
// <html> and add  .sb-cloak :not(:defined) { visibility: hidden }
// The class is removed once the first components are defined (or after 3 s).
const modules = ` + string(mj) + `
const requires = ` + string(rj) + `
const started = new Set()
let pending = 0
let settled = false
let markReady
/** Resolves when the components present at startup are defined. */
export const ready = new Promise((resolve) => (markReady = resolve))

const report = (err) => (typeof reportError === 'function' ? reportError(err) : console.error(err))

const uncloak = () => {
	if (settled) return
	settled = true
	document.documentElement.classList.remove('sb-cloak')
	markReady()
}
// Wait a microtask and a frame, so tags added right after a load count too.
const settle = () =>
	queueMicrotask(() => pending === 0 && requestAnimationFrame(() => pending === 0 && uncloak()))
setTimeout(uncloak, 3000) // never leave a page cloaked

const load = (tag) => {
	if (started.has(tag) || !modules[tag] || customElements.get(tag)) return
	started.add(tag)
	for (const dep of requires[tag] || []) load(dep) // tags it renders itself
	pending++
	import(new URL(modules[tag], import.meta.url).href)
		// Rocket defines elements once Datastar is ready: wait for that too.
		.then(() => customElements.whenDefined(tag))
		.catch((err) => {
			started.delete(tag)
			report(new Error('[starbase] could not load <' + tag + '>', { cause: err }))
		})
		.finally(() => {
			pending--
			settle()
		})
}

/** Load the components used in root (an element, document or shadow root). */
export const discover = (root) => {
	if (root.localName?.startsWith('sb-')) load(root.localName)
	for (const el of root.querySelectorAll?.(':not(:defined)') ?? []) load(el.localName)
}

discover(document.documentElement)
settle() // nothing to load: uncloak right away
new MutationObserver((records) => {
	for (const r of records) for (const n of r.addedNodes) if (n.nodeType === 1) discover(n)
}).observe(document.documentElement, { subtree: true, childList: true })
`
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
