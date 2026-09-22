package web

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"starbase/internal/catalog"
)

// Pull request previews: /playground?preview=<commit>/<slug> opens a
// component from any commit of the site's repository (the submission bot
// links one in every PR), so maintainers can try it before merging. The
// code runs in the playground's sandbox like any snippet.
//
// A commit's files never change, so fetching them once and memoizing the
// result keeps the page a pure function of its URL.

var commitRe = regexp.MustCompile(`^[0-9a-f]{40}$`)

const (
	maxPreviewFile = 512 << 10
	maxPreviews    = 64
	maxVendorBytes = 2 << 20 // per vendored file, as in the submission bot
)

type preview struct {
	Name  string
	Files map[string]string
}

type previewCache struct {
	raw    string // raw file host, e.g. https://raw.githubusercontent.com/<owner>/<repo>
	client *http.Client

	mu      sync.Mutex
	entries map[string]*previewEntry
}

type previewEntry struct {
	once sync.Once
	p    *preview
	err  error
}

var errNoPreview = errors.New("no such component at that commit")

// newPreviewCache serves previews from repoURL (a https://github.com/<owner>/<repo> URL).
func newPreviewCache(repoURL string) *previewCache {
	raw := ""
	if rest, ok := strings.CutPrefix(strings.TrimSuffix(repoURL, "/"), "https://github.com/"); ok {
		raw = "https://raw.githubusercontent.com/" + rest
	}
	return &previewCache{raw: raw, client: &http.Client{Timeout: 10 * time.Second}, entries: map[string]*previewEntry{}}
}

// get returns the component <slug> at <commit>, from "<commit>/<slug>".
func (c *previewCache) get(ctx context.Context, ref string) (*preview, error) {
	commit, slug, ok := strings.Cut(ref, "/")
	if !ok || !commitRe.MatchString(commit) || !catalog.ValidSlug(slug) || c.raw == "" {
		return nil, errNoPreview
	}
	c.mu.Lock()
	e := c.entries[ref]
	if e == nil {
		if len(c.entries) >= maxPreviews {
			clear(c.entries) // rare; they refetch
		}
		e = &previewEntry{}
		c.entries[ref] = e
	}
	c.mu.Unlock()
	e.once.Do(func() {
		// Not the request's context: a cancelled request must not cache a failure.
		ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 15*time.Second)
		defer cancel()
		e.p, e.err = c.fetch(ctx, commit, slug)
	})
	if e.err != nil && !errors.Is(e.err, errNoPreview) {
		c.mu.Lock()
		delete(c.entries, ref) // retry transient failures next time
		c.mu.Unlock()
	}
	return e.p, e.err
}

func (c *previewCache) fetch(ctx context.Context, commit, slug string) (*preview, error) {
	dir := c.raw + "/" + commit + "/components/" + slug + "/"
	js, err := c.file(ctx, dir+slug+".js")
	if err != nil {
		return nil, err
	}
	readme, err := c.file(ctx, dir+"README.md")
	if err != nil {
		return nil, err
	}
	var meta catalog.Meta
	if _, err := catalog.RenderMarkdown(readme, &meta); err != nil {
		return nil, fmt.Errorf("README.md: %w", err)
	}
	return &preview{
		Name:  meta.Name,
		Files: map[string]string{"component.js": string(js), "index.html": strings.Join(catalog.Examples(readme), "\n\n") + "\n"},
	}, nil
}

func (c *previewCache) file(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	res, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	switch {
	case res.StatusCode == http.StatusNotFound:
		return nil, errNoPreview
	case res.StatusCode != http.StatusOK:
		return nil, fmt.Errorf("fetching %s: %s", url, res.Status)
	}
	b, err := io.ReadAll(io.LimitReader(res.Body, maxPreviewFile+1))
	if err == nil && len(b) > maxPreviewFile {
		err = fmt.Errorf("%s is larger than %d KB", url, maxPreviewFile>>10)
	}
	return b, err
}

// servePreviewFile serves a preview's other .js files (vendored libraries)
// from the pinned commit, so relative imports work in the playground. It
// streams them; a commit's files never change, so browsers cache them for good.
func (s *Server) servePreviewFile(w http.ResponseWriter, r *http.Request) {
	commit, slug, file := r.PathValue("commit"), r.PathValue("slug"), r.PathValue("file")
	c := s.previews
	if !commitRe.MatchString(commit) || !catalog.ValidSlug(slug) || !fs.ValidPath(file) || c.raw == "" ||
		!(strings.HasSuffix(file, ".js") || strings.HasSuffix(file, ".mjs")) {
		http.NotFound(w, r)
		return
	}
	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, c.raw+"/"+commit+"/components/"+slug+"/"+file, nil)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	res, err := c.client.Do(req)
	if err != nil {
		http.Error(w, "upstream error", http.StatusBadGateway)
		return
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK || res.ContentLength > maxVendorBytes {
		http.NotFound(w, r)
		return
	}
	h := w.Header()
	h.Set("Content-Type", "text/javascript; charset=utf-8")
	h.Set("Access-Control-Allow-Origin", "*") // the sandboxed runner has an opaque origin
	h.Set("Cache-Control", immutable)
	io.Copy(w, io.LimitReader(res.Body, maxVendorBytes))
}
