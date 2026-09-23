package catalog

import (
	"crypto/sha512"
	"encoding/base64"
	"encoding/json"
	"io/fs"
	"regexp"
	"strings"
)

// VersionedScript is the component's module, versioned by its content hash:
// "<slug>@<hash>/<slug>.js", served under /c/ and immutable.
func (c *Component) VersionedScript() string {
	return c.Slug + "@" + c.Hash + "/" + strings.TrimPrefix(c.Script, c.Slug+"/")
}

// ModuleFiles returns the component's .js and .mjs files (its module and
// anything it imports, like vendored libraries), keyed by path inside the
// folder. These are the public, versioned files.
func (cat *Catalog) ModuleFiles(c *Component) (map[string][]byte, error) {
	out := map[string][]byte{}
	err := fs.WalkDir(cat.FS, c.Slug, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !(strings.HasSuffix(p, ".js") || strings.HasSuffix(p, ".mjs")) {
			return err
		}
		b, err := fs.ReadFile(cat.FS, p)
		out[strings.TrimPrefix(p, c.Slug+"/")] = b
		return err
	})
	return out, err
}

// SRI is a Subresource Integrity hash ("sha384-…") for integrity="…" and
// import maps.
func SRI(b []byte) string {
	h := sha512.Sum384(b)
	return "sha384-" + base64.StdEncoding.EncodeToString(h[:])
}

var usesTagRe = regexp.MustCompile(`<(sb-[a-z0-9]+(?:-[a-z0-9]+)*)`)

// AutoloaderJS generates the autoloader: a map from every tag to its module
// and the components each one renders (found as <sb-… in its source).
// Module URLs are versioned (<slug>@<hash>/…, immutable) and relative to the
// autoloader, behind prefix: "" for /c/autoloader.js, "../" for a pinned
// snapshot at /c/@<catalog hash>/autoloader.js.
func AutoloaderJS(cat *Catalog, prefix string) string {
	modules := map[string]string{}
	requires := map[string][]string{}
	for _, c := range cat.Components {
		modules[c.Tag] = prefix + c.VersionedMinScript() // the minified module (see min.go)
	}
	for _, c := range cat.Components {
		for _, u := range cat.Uses(c) {
			requires[c.Tag] = append(requires[c.Tag], u.Tag)
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
// For production, pin a snapshot of the whole catalog instead:
// …/c/@<catalog hash>/autoloader.js, with its integrity hashes from
// …/c/@<catalog hash>/importmap.json.
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
