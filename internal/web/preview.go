package web

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"path"
	"regexp"
	"strings"
	"sync"
	"time"

	"starbase/internal/catalog"
)

// Pull request previews: /playground?preview=<commit>/<slug> opens a
// component from a commit of a submission pull request (the bot links one
// in every PR), so maintainers can try it before merging. The code runs in
// the playground's sandbox like any snippet. Only commits of pull requests
// from the repository's own component/* branches qualify: forks share
// objects with the repository, and the preview must not become a way to
// serve anyone's code from this origin.
//
// A commit's files never change, so fetching them once and memoizing the
// result keeps the page a pure function of its URL.

var commitRe = regexp.MustCompile(`^[0-9a-f]{40}$`)

const (
	maxPreviewFile    = 512 << 10
	maxPreviews       = 64
	maxVendorBytes    = 2 << 20 // per vendored file, as in the submission bot
	maxPreviewModules = 64      // files the type check reads beside the module
	maxModuleBytes    = 4 << 20 // their total, as in the submission bot
)

type preview struct {
	Name  string
	Files map[string]string // component.js (component.ts when written in TypeScript) and index.html

	dir     string // its folder at the commit, on the raw file host
	slug    string
	main    string // component.js or component.ts
	mu      sync.Mutex
	modules map[string]string // see previewCache.modules
}

type previewCache struct {
	raw    string // raw file host, e.g. https://raw.githubusercontent.com/<owner>/<repo>
	api    string // e.g. https://api.github.com/repos/<owner>/<repo>
	repo   string // <owner>/<repo>
	token  string // optional
	client *http.Client

	mu      sync.Mutex
	entries map[string]*previewEntry
	commits map[string]commitCheck // commit → is it a submission PR's?
}

type commitCheck struct {
	ok bool
	at time.Time
}

type previewEntry struct {
	once sync.Once
	p    *preview
	err  error
}

var errNoPreview = errors.New("no such component at that commit")

// newPreviewCache serves previews from repoURL (a https://github.com/<owner>/<repo> URL).
func newPreviewCache(repoURL, token string) *previewCache {
	c := &previewCache{token: token, client: &http.Client{Timeout: 10 * time.Second}, entries: map[string]*previewEntry{}, commits: map[string]commitCheck{}}
	if rest, ok := strings.CutPrefix(strings.TrimSuffix(repoURL, "/"), "https://github.com/"); ok {
		c.repo = rest
		c.raw = "https://raw.githubusercontent.com/" + rest
		c.api = "https://api.github.com/repos/" + rest
	}
	return c
}

// submission reports whether commit belongs to a pull request from one of
// the repository's own component/* branches (the bot's). GitHub lists open
// and merged pull requests for a commit, so a submission closed without
// merging loses its preview. Answers are
// cached: yes for good, no for a minute (GitHub links new commits to their
// pull request with a short delay).
func (c *previewCache) submission(ctx context.Context, commit string) (bool, error) {
	c.mu.Lock()
	if v, ok := c.commits[commit]; ok && (v.ok || time.Since(v.at) < time.Minute) {
		c.mu.Unlock()
		return v.ok, nil
	}
	c.mu.Unlock()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.api+"/commits/"+commit+"/pulls", nil)
	if err != nil {
		return false, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	res, err := c.client.Do(req)
	if err != nil {
		return false, err
	}
	defer res.Body.Close()
	ok := false
	switch res.StatusCode {
	case http.StatusOK:
		var pulls []struct {
			Head struct {
				Ref  string `json:"ref"`
				Repo struct {
					FullName string `json:"full_name"`
				} `json:"repo"`
			} `json:"head"`
		}
		if err := json.NewDecoder(io.LimitReader(res.Body, 1<<20)).Decode(&pulls); err != nil {
			return false, err
		}
		for _, p := range pulls {
			if strings.EqualFold(p.Head.Repo.FullName, c.repo) && strings.HasPrefix(p.Head.Ref, "component/") {
				ok = true
			}
		}
	case http.StatusNotFound, http.StatusUnprocessableEntity: // unknown commit
	default:
		return false, fmt.Errorf("checking commit %s: GitHub says %s", commit, res.Status)
	}
	c.mu.Lock()
	if len(c.commits) > 4096 {
		clear(c.commits)
	}
	c.commits[commit] = commitCheck{ok: ok, at: time.Now()}
	c.mu.Unlock()
	return ok, nil
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
		if ok, err := c.submission(ctx, commit); err != nil || !ok {
			e.err = cmp.Or(err, errNoPreview)
			return
		}
		e.p, e.err = c.fetch(ctx, commit, slug)
	})
	if e.err != nil {
		c.mu.Lock()
		delete(c.entries, ref) // only successes are memoized; the commit check has its own cache
		c.mu.Unlock()
	}
	return e.p, e.err
}

// fetch reads the component's module and examples. A component written in
// TypeScript opens as its <slug>.ts, like ?component= (catalog.Component.SourceFile).
func (c *previewCache) fetch(ctx context.Context, commit, slug string) (*preview, error) {
	p := &preview{dir: c.raw + "/" + commit + "/components/" + slug + "/", slug: slug, main: "component.ts"}
	src, err := c.file(ctx, p.dir+slug+".ts", maxPreviewFile)
	if errors.Is(err, errNoPreview) {
		p.main = "component.js"
		src, err = c.file(ctx, p.dir+slug+".js", maxPreviewFile)
	}
	if err != nil {
		return nil, err
	}
	readme, err := c.file(ctx, p.dir+"README.md", maxPreviewFile)
	if err != nil {
		return nil, err
	}
	var meta catalog.Meta
	if _, err := catalog.RenderMarkdown(readme, &meta); err != nil {
		return nil, fmt.Errorf("README.md: %w", err)
	}
	p.Name = meta.Name
	p.Files = map[string]string{p.main: string(src), "index.html": strings.Join(catalog.Examples(readme), "\n\n") + "\n"}
	return p, nil
}

// moduleName matches the names the type check takes (tscheck.File).
var moduleName = regexp.MustCompile(`^(?:[\w-][\w.-]*/)*[\w-][\w.-]*\.m?[jt]s$`)

// modules returns the files of the folder of preview ref that its module,
// as committed, reaches through relative imports: what the type check reads
// beside it, as it reads a catalog component's folder. They are fetched on
// the first check, a round of imports at a time, and kept with the preview.
// A file that is missing or too large is left out, so the check reports it.
// A ref that is no preview has none.
func (c *previewCache) modules(ctx context.Context, ref string) (map[string]string, error) {
	p, err := c.get(ctx, ref)
	if errors.Is(err, errNoPreview) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.modules != nil {
		return p.modules, nil
	}
	seen := map[string]bool{p.slug + ".ts": true, p.slug + ".js": true} // the module itself, checked as p.main
	out := map[string]string{}
	var next []string
	follow := func(name, code string) {
		for _, spec := range catalog.Imports(code) {
			to := path.Join(path.Dir(name), spec)
			if (strings.HasPrefix(spec, "./") || strings.HasPrefix(spec, "../")) && moduleName.MatchString(to) && !seen[to] && len(seen) <= maxPreviewModules {
				seen[to] = true
				next = append(next, to)
			}
		}
	}
	follow(p.main, p.Files[p.main])
	total := 0
	for len(next) > 0 {
		round := next
		next = nil
		bodies := make([][]byte, len(round))
		errs := make([]error, len(round))
		var wg sync.WaitGroup
		for i, name := range round {
			wg.Go(func() { bodies[i], errs[i] = c.file(ctx, p.dir+name, maxVendorBytes) })
		}
		wg.Wait()
		for i, name := range round {
			switch err := errs[i]; {
			case errors.Is(err, errNoPreview), errors.Is(err, errTooLarge):
			case err != nil:
				return nil, err // not kept: the next check tries again
			case total+len(bodies[i]) <= maxModuleBytes:
				total += len(bodies[i])
				out[name] = string(bodies[i])
				follow(name, out[name])
			}
		}
	}
	p.modules = out
	return out, nil
}

var errTooLarge = errors.New("file too large")

func (c *previewCache) file(ctx context.Context, url string, max int) ([]byte, error) {
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
	b, err := io.ReadAll(io.LimitReader(res.Body, int64(max)+1))
	if err == nil && len(b) > max {
		err = fmt.Errorf("%s is larger than %d KB: %w", url, max>>10, errTooLarge)
	}
	return b, err
}

// servePreviewFile serves a preview's other .js files (its modules and
// vendored libraries; the runner points a TypeScript module's imports of
// ./x.ts at the committed x.js) from the pinned commit, so relative imports
// work in the playground. It
// streams them; a commit's files never change, so browsers cache them for good.
func (s *Server) servePreviewFile(w http.ResponseWriter, r *http.Request) {
	commit, slug, file := r.PathValue("commit"), r.PathValue("slug"), r.PathValue("file")
	c := s.previews
	if !commitRe.MatchString(commit) || !catalog.ValidSlug(slug) || !fs.ValidPath(file) || c.raw == "" ||
		!(strings.HasSuffix(file, ".js") || strings.HasSuffix(file, ".mjs")) {
		http.NotFound(w, r)
		return
	}
	if ok, err := c.submission(r.Context(), commit); err != nil || !ok {
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
