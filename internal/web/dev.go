package web

import (
	"bytes"
	"encoding/json"
	"hash/fnv"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/starfederation/datastar-go/datastar"

	"starbase/internal/model"
)

// devRoutes exist only in -tags=dev builds.
func (s *Server) devRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /dev/reload", s.devReload)
	mux.HandleFunc("GET /auth/dev", s.devLogin)
	mux.HandleFunc("POST /dev/manifests", s.devManifests)
}

// devReload holds an SSE connection open. When the server restarts (air),
// the browser reconnects with the old boot id and is told to reload.
func (s *Server) devReload(w http.ResponseWriter, r *http.Request) {
	sse := datastar.NewSSE(w, r)
	if r.URL.Query().Get("boot") != s.boot {
		sse.PatchElements(`<div hidden data-init="window.location.reload()"></div>`,
			datastar.WithSelector("body"), datastar.WithModeAppend())
		return
	}
	select {
	case <-r.Context().Done():
	case <-s.ctx.Done():
	}
}

var devLoginRe = regexp.MustCompile(`^[a-z0-9-]{1,39}$`)

// devLogin signs in as a fake GitHub user, for working without OAuth.
func (s *Server) devLogin(w http.ResponseWriter, r *http.Request) {
	login := r.URL.Query().Get("login")
	if !devLoginRe.MatchString(login) {
		login = "dev-astronaut"
	}
	h := fnv.New32a()
	h.Write([]byte(login))
	s.signIn(w, r, model.User{
		GitHubID:  -int64(h.Sum32()), // negative: never collides with real GitHub ids
		Login:     login,
		Name:      "Dev " + login,
		AvatarURL: "/art/moon.svg",
	}, r.URL.Query().Get("next"))
}

// devManifests receives Rocket's manifest document (published by the page
// in dev) and writes components/<slug>/manifest.json for every component
// that changed. Commit the files.
func (s *Server) devManifests(w http.ResponseWriter, r *http.Request) {
	var doc struct {
		Components []json.RawMessage `json:"components"`
	}
	if err := json.NewDecoder(r.Body).Decode(&doc); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	byTag := map[string]string{}
	for _, c := range s.catalog.Components {
		byTag[c.Tag] = c.Slug
	}
	written := 0
	for _, raw := range doc.Components {
		var head struct {
			Tag string `json:"tag"`
		}
		json.Unmarshal(raw, &head)
		slug, ok := byTag[head.Tag]
		if !ok {
			continue
		}
		var pretty bytes.Buffer
		if err := json.Indent(&pretty, raw, "", "  "); err != nil {
			continue
		}
		pretty.WriteByte('\n')
		path := filepath.Join("components", slug, "manifest.json")
		if old, err := os.ReadFile(path); err == nil && bytes.Equal(old, pretty.Bytes()) {
			continue
		}
		if err := os.WriteFile(path, pretty.Bytes(), 0o644); err != nil {
			s.log.Warn("write manifest", "path", path, "err", err)
			continue
		}
		written++
		s.log.Info("manifest updated", "component", slug)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"written": written, "at": time.Now()})
}

func (s *Server) manifestTags() []string {
	if !s.cfg.Dev {
		return nil
	}
	tags := make([]string, 0, len(s.catalog.Components))
	for _, c := range s.catalog.Components {
		tags = append(tags, c.Tag)
	}
	return tags
}
