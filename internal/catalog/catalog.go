// Package catalog loads the component collection from the components/
// directory. Each component is one folder:
//
//	components/<slug>/README.md      front matter + docs with live examples
//	components/<slug>/<slug>.js      the Rocket component definition
//	components/<slug>/manifest.json  props/slots/events, generated in dev
//
// The catalog is immutable after Load. The SyncCatalog command mirrors it into
// SQLite so queries can filter, search, sort and count stars.
package catalog

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	stdhtml "html"
	"io/fs"
	"path"
	"regexp"
	"slices"
	"strings"
	"time"
)

type Meta struct {
	Name     string   `yaml:"name"`
	Tag      string   `yaml:"tag"`
	Category string   `yaml:"category"`
	Summary  string   `yaml:"summary"`
	Author   string   `yaml:"author"`
	Tags     []string `yaml:"tags"`
	Since    string   `yaml:"since"`
	Preview  string   `yaml:"preview"`  // the gallery card's live demo
	Usage    string   `yaml:"usage"`    // optional: the smallest markup to paste into a page; installation snippets fall back to Preview
	Source   string   `yaml:"source"`   // optional: upstream repository (pinned)
	Unlisted bool     `yaml:"unlisted"` // part of the site, not the gallery: served and documented, but not browsed

	Playground PlaygroundMeta `yaml:"playground"`
}

// InstallMarkup is the markup the installation snippets end with: the
// component's usage, or its gallery preview when it has none.
func (m Meta) InstallMarkup() string {
	if u := strings.TrimSpace(m.Usage); u != "" {
		return u
	}
	return strings.TrimSpace(m.Preview)
}

type Component struct {
	Meta
	Slug     string
	Script   string // path inside the components FS, e.g. "button/button.js"
	DocHTML  string
	Manifest *Manifest
	Hash     string // content hash over every file in the folder
	// Integrity is the SRI hash of the component's module (<slug>.js).
	Integrity string
	Headings  []Heading
	Examples  []string // bodies of the README's ```html preview blocks
	Sizes     Sizes    // download weight (computed by Load)
}

// Manifest mirrors one entry of Rocket's generated manifest document.
type Manifest struct {
	Tag    string          `json:"tag"`
	Props  []ManifestProp  `json:"props"`
	Slots  []ManifestSlot  `json:"slots"`
	Events []ManifestEvent `json:"events"`
}

type ManifestProp struct {
	Name      string          `json:"name"`
	Attribute string          `json:"attribute"`
	Type      string          `json:"type"`
	Default   json.RawMessage `json:"default"`
	Values    []any           `json:"values,omitempty"`
	Docs      *struct {
		Description string `json:"description"`
	} `json:"docs,omitempty"`
}

type ManifestSlot struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type ManifestEvent struct {
	Name        string `json:"name"`
	Kind        string `json:"kind"`
	Description string `json:"description"`
}

type Catalog struct {
	FS         fs.FS
	Components []*Component // sorted by slug
	bySlug     map[string]*Component
	Hash       string // hash over all component hashes
}

func (c *Catalog) Get(slug string) (*Component, bool) {
	comp, ok := c.bySlug[slug]
	return comp, ok
}

var (
	slugRe   = regexp.MustCompile(`^[a-z][a-z0-9]*(-[a-z0-9]+)*$`)
	tagRe    = regexp.MustCompile(`^sb-[a-z0-9]+(-[a-z0-9]+)*$`)
	authorRe = regexp.MustCompile(`^[A-Za-z0-9](?:[A-Za-z0-9]|-[A-Za-z0-9]){0,38}$`)
)

// Load reads and validates every component folder in fsys. All problems are
// reported together so a contributor sees everything at once.
func Load(fsys fs.FS) (*Catalog, error) {
	entries, err := fs.ReadDir(fsys, ".")
	if err != nil {
		return nil, err
	}
	cat := &Catalog{FS: fsys, bySlug: map[string]*Component{}}
	var errs []error
	tags := map[string]string{}
	for _, e := range entries {
		if !e.IsDir() || strings.HasPrefix(e.Name(), ".") || strings.HasPrefix(e.Name(), "_") {
			continue
		}
		c, err := loadOne(fsys, e.Name())
		if err != nil {
			errs = append(errs, fmt.Errorf("components/%s: %w", e.Name(), err))
			continue
		}
		if other, dup := tags[c.Tag]; dup {
			errs = append(errs, fmt.Errorf("components/%s: tag %q already used by %s", c.Slug, c.Tag, other))
			continue
		}
		tags[c.Tag] = c.Slug
		cat.Components = append(cat.Components, c)
		cat.bySlug[c.Slug] = c
	}
	if len(errs) > 0 {
		return nil, errors.Join(errs...)
	}
	slices.SortFunc(cat.Components, func(a, b *Component) int { return strings.Compare(a.Slug, b.Slug) })
	h := sha256.New()
	for _, c := range cat.Components {
		h.Write([]byte(c.Hash))
	}
	cat.Hash = hex.EncodeToString(h.Sum(nil))[:12]
	if err := cat.computeSizes(); err != nil {
		return nil, err
	}
	return cat, nil
}

// minFormat is part of every component's version hash. A version's .min
// files are frozen the first time they are stored (min.go), so a change to
// how they are made (esbuild options, shrink.go) must change it: every
// component then gets a new version, whose .min files are made the new way.
const minFormat = "min2"

func loadOne(fsys fs.FS, slug string) (*Component, error) {
	if !slugRe.MatchString(slug) {
		return nil, fmt.Errorf("folder name must be kebab-case")
	}
	readme, err := fs.ReadFile(fsys, path.Join(slug, "README.md"))
	if err != nil {
		return nil, fmt.Errorf("missing README.md")
	}
	c := &Component{Slug: slug, Script: path.Join(slug, slug+".js")}
	c.DocHTML, err = RenderMarkdown(readme, &c.Meta)
	if err != nil {
		return nil, fmt.Errorf("README.md: %w", err)
	}
	c.Headings = headings(c.DocHTML)
	c.Examples = Examples(readme)
	js, err := fs.ReadFile(fsys, c.Script)
	if err != nil {
		return nil, fmt.Errorf("missing %s.js", slug)
	}

	var problems []string
	// Imports: 'datastar', or files inside the folder (vendored libraries).
	for _, spec := range Imports(string(js)) {
		target := path.Join(slug, spec)
		switch {
		case spec == "datastar":
		case !strings.HasPrefix(spec, "./") && !strings.HasPrefix(spec, "../"):
			problems = append(problems, fmt.Sprintf("%s.js imports %q: only 'datastar' and files in the component's folder", slug, spec))
		case !strings.HasPrefix(target, slug+"/"):
			problems = append(problems, fmt.Sprintf("%s.js imports %q, outside the component's folder", slug, spec))
		default:
			if _, err := fs.Stat(fsys, target); err != nil {
				problems = append(problems, fmt.Sprintf("%s.js imports %q, which does not exist", slug, spec))
			}
		}
	}
	req := func(v, field string) {
		if strings.TrimSpace(v) == "" {
			problems = append(problems, "front matter: "+field+" is required")
		}
	}
	req(c.Name, "name")
	req(c.Summary, "summary")
	req(c.Preview, "preview")
	if !tagRe.MatchString(c.Tag) {
		problems = append(problems, fmt.Sprintf("front matter: tag %q must look like sb-my-widget", c.Tag))
	} else if !strings.Contains(string(js), "rocket('"+c.Tag+"'") && !strings.Contains(string(js), `rocket("`+c.Tag+`"`) {
		problems = append(problems, fmt.Sprintf("%s.js must define rocket('%s', ...)", slug, c.Tag))
	}
	if _, ok := CategoryBySlug(c.Category); !ok {
		problems = append(problems, fmt.Sprintf("front matter: unknown category %q", c.Category))
	}
	if !authorRe.MatchString(c.Author) {
		problems = append(problems, fmt.Sprintf("front matter: author %q must be a GitHub handle", c.Author))
	}
	if _, err := time.Parse(time.DateOnly, c.Since); err != nil {
		problems = append(problems, fmt.Sprintf("front matter: since %q must be YYYY-MM-DD", c.Since))
	}
	if c.Source != "" && !strings.HasPrefix(c.Source, "https://github.com/") {
		problems = append(problems, "front matter: source must be a https://github.com/ URL")
	}
	if len(c.Summary) > 90 {
		problems = append(problems, "front matter: summary must be at most 90 characters")
	}

	if b, err := fs.ReadFile(fsys, path.Join(slug, "manifest.json")); err == nil {
		c.Manifest = &Manifest{}
		if err := json.Unmarshal(b, c.Manifest); err != nil {
			problems = append(problems, "manifest.json: "+err.Error())
		}
	}
	if len(problems) > 0 {
		return nil, errors.New(strings.Join(problems, "; "))
	}

	h := sha256.New()
	h.Write([]byte(minFormat))
	err = fs.WalkDir(fsys, slug, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		b, err := fs.ReadFile(fsys, p)
		h.Write([]byte(p))
		h.Write(b)
		return err
	})
	if err != nil {
		return nil, err
	}
	c.Hash = hex.EncodeToString(h.Sum(nil))[:12]
	c.Integrity = SRI(js)
	return c, nil
}

// Heading is an h2 in a component's docs, for the "On this page" nav.
type Heading struct{ ID, Text string }

var previewRe = regexp.MustCompile("(?s)```html preview\n(.*?)```")

// Examples returns the bodies of a README's ```html preview blocks.
func Examples(readme []byte) []string {
	var out []string
	for _, m := range previewRe.FindAllStringSubmatch(string(readme), -1) {
		out = append(out, strings.TrimRight(m[1], "\n"))
	}
	return out
}

var h2Re = regexp.MustCompile(`<h2 id="([^"]+)">(.*?)</h2>`)
var tagStrip = regexp.MustCompile(`<[^>]+>`)

func headings(html string) []Heading {
	var hs []Heading
	for _, m := range h2Re.FindAllStringSubmatch(html, -1) {
		hs = append(hs, Heading{ID: m[1], Text: stdhtml.UnescapeString(tagStrip.ReplaceAllString(m[2], ""))})
	}
	return hs
}

// ValidSlug reports whether s is a valid component folder name.
func ValidSlug(s string) bool { return slugRe.MatchString(s) }
