package web

import (
	"encoding/json"
	"io/fs"
	"net/http"
	"regexp"
	"starbase/internal/precompress"
	"strings"

	"starbase/internal/catalog"
	"starbase/internal/queries"
)

// Versioned component URLs. Everything here is immutable, public and CORS-open,
// so other sites can pin exactly what they reviewed:
//
//	/c/<slug>@<hash>/<file>.js      one version of a component's file
//	/c/@<catalog>/autoloader.js     a snapshot of the whole catalog's autoloader
//	/c/@<catalog>/importmap.json    SRI hashes for everything that snapshot loads
//
// SyncCatalog stores every version it sees, so old URLs keep working after
// later deploys. The current version is served from the binary.

var versionHashRe = regexp.MustCompile(`^[0-9a-f]{12}$`)

func (s *Server) serveC(w http.ResponseWriter, r *http.Request) {
	first, rest, _ := strings.Cut(r.PathValue("path"), "/")
	switch {
	case strings.HasPrefix(first, "@"):
		s.serveSnapshot(w, r, first[1:], rest)
	case strings.Contains(first, "@"):
		slug, hash, _ := strings.Cut(first, "@")
		s.serveVersioned(w, r, slug, hash, rest)
	default:
		s.assets.serveComponents(w, r)
	}
}

func publicModule(p string) bool {
	return fs.ValidPath(p) && (strings.HasSuffix(p, ".js") || strings.HasSuffix(p, ".mjs"))
}

func (s *Server) serveVersioned(w http.ResponseWriter, r *http.Request, slug, hash, file string) {
	if !catalog.ValidSlug(slug) || !versionHashRe.MatchString(hash) || !publicModule(file) {
		http.NotFound(w, r)
		return
	}
	var body []byte
	c, current := s.catalog.Get(slug)
	current = current && c.Hash == hash
	switch {
	case current && !catalog.IsMinPath(file):
		body, _ = fs.ReadFile(s.catalog.FS, slug+"/"+file) // the current version
	default:
		// Older versions, and every minified file: the stored copy is the one
		// that was first published, so an esbuild upgrade can't change the
		// bytes behind a URL someone pinned.
		s.q.View(r.Context(), func(rd *queries.Reader) (err error) {
			body, _, _, err = rd.ComponentFile(r.Context(), slug, hash, file)
			return
		})
		if body == nil && current { // not synced yet (tests, first start)
			if mins, err := s.catalog.MinFiles(c); err == nil {
				body = mins[file]
			}
		}
	}
	if body == nil {
		http.NotFound(w, r)
		return
	}
	s.immutableModule(w, "text/javascript; charset=utf-8")
	precompress.Write(w, r, body)
}

func (s *Server) serveSnapshot(w http.ResponseWriter, r *http.Request, hash, file string) {
	if !versionHashRe.MatchString(hash) || (file != "autoloader.js" && file != "importmap.json") {
		http.NotFound(w, r)
		return
	}
	var auto, integrity string
	var ok bool
	var files []queries.VersionedFile
	err := s.q.View(r.Context(), func(rd *queries.Reader) (err error) {
		if auto, integrity, ok, err = rd.Snapshot(r.Context(), hash); err != nil || !ok || file != "importmap.json" {
			return err
		}
		files, err = rd.SnapshotFiles(r.Context(), hash)
		return err
	})
	if err != nil {
		s.fail(w, r, err)
		return
	}
	if !ok {
		http.NotFound(w, r)
		return
	}
	if file == "autoloader.js" {
		s.immutableModule(w, "text/javascript; charset=utf-8")
		precompress.Write(w, r, []byte(auto))
		return
	}
	// Merge "integrity" into your import map; browsers then refuse any
	// module whose bytes don't match.
	base := strings.TrimSuffix(s.cfg.BaseURL, "/") + "/c/"
	m := map[string]string{base + "@" + hash + "/autoloader.js": integrity}
	for _, f := range files {
		m[base+f.Slug+"@"+f.Hash+"/"+f.Path] = f.Integrity
	}
	s.immutableModule(w, "application/json")
	body, _ := json.MarshalIndent(map[string]any{"integrity": m}, "", "  ")
	precompress.Write(w, r, append(body, '\n'))
}

func (s *Server) immutableModule(w http.ResponseWriter, contentType string) {
	h := w.Header()
	h.Set("Content-Type", contentType)
	h.Set("Access-Control-Allow-Origin", "*") // for other sites and the playground's sandbox
	s.assets.cacheHeader(w, true)
}
