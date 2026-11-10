package web

import (
	"bytes"
	"cmp"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"

	"github.com/a-h/templ"
	"github.com/starfederation/datastar-go/datastar"
	"github.com/zweiundeins/lith/edge"
	"github.com/zweiundeins/lith/live"
	"github.com/zweiundeins/lith/sse"

	"starbase/internal/commands"
	"starbase/internal/model"
	"starbase/internal/queries"
	"starbase/internal/ui"
)

// view is what a page renders from the read model.
type view struct {
	Title       string
	Description string
	Nav         string
	Body        func(ui.Shell) templ.Component
	URL         string         // canonical URL for the current state; "" keeps the address bar
	SearchLive  bool           // header search filters this page in place
	Search      string         // header search value
	Signals     map[string]any // initial page signals (full document only)
	Status      int            // 0 = 200
	Schema      []any          // extra JSON-LD objects (schema.org), after the site's
	LD          func() string  // the page's whole JSON-LD, computed once (the homepage's), in place of Schema
}

// renderCtx is the input of a page render: one read snapshot plus who is
// looking and what their tab state is.
type renderCtx struct {
	ctx    context.Context
	r      *queries.Reader
	req    *http.Request
	sid    string
	tabID  string
	stream bool // for the tab's stream, where a region is a hole (region.go)
	user   *model.User
	tab    model.TabState     // stored state, or seeded from the query string
	prefs  model.SessionPrefs // the session's preferences (known on the first GET too)
}

type pageFunc func(rc *renderCtx) (view, error)

var errNotFound = errors.New("not found")

var tabIDRe = regexp.MustCompile(`^[a-z0-9]{8,32}$`)

// page registers GET (document) and POST (render stream) for a pattern.
// The page's streams also watch the views of its regions, which all its
// viewers share (the Showcase's board), given in the order the page shows
// them.
func (s *Server) page(mux *http.ServeMux, pattern string, fn pageFunc, regions ...*region) {
	mux.HandleFunc("GET "+pattern, s.document(fn))
	mux.HandleFunc("POST "+pattern, s.stream(fn, regions))
}

// load renders a page in rd's snapshot: the document's own read
// transaction, or a render worker's.
func (s *Server) load(ctx context.Context, rd *queries.Reader, req *http.Request, tabID string, fn pageFunc) (v view, user *model.User, err error) {
	sid := edge.SessionID(req.Context())
	if user, err = rd.SessionUser(ctx, sid); err != nil {
		return v, nil, err
	}
	prefs, err := rd.Prefs(ctx, sid)
	if err != nil {
		return v, nil, err
	}
	tab, ok, err := rd.Tab(ctx, sid, tabID)
	if err != nil {
		return v, nil, err
	}
	if !ok { // a new page: its URL, and the session's default sort
		tab = model.TabState{Browse: model.BrowseFromQuery(req.URL.Query(), prefs.DefaultSort())}
	}
	v, err = fn(&renderCtx{ctx: ctx, r: rd, req: req, sid: sid, tabID: tabID, stream: tabID != "", user: user, tab: tab, prefs: prefs})
	return v, user, err
}

func (s *Server) shell(req *http.Request, user *model.User, v view) ui.Shell {
	next := req.URL.RequestURI()
	if v.URL != "" {
		next = v.URL
	}
	return ui.Shell{
		Title:        v.Title,
		Description:  v.Description,
		Nav:          v.Nav,
		User:         user,
		Search:       v.Search,
		SearchLive:   v.SearchLive,
		URL:          v.URL,
		Signals:      ui.PageSignals(s.pageSignals(v.Signals)),
		LoginURL:     s.loginURL(next),
		RepoURL:      s.cfg.RepoURL,
		BaseURL:      s.cfg.BaseURL,
		Assets:       s.assets,
		Nonce:        edge.Nonce(req.Context()),
		Boot:         s.boot,
		Dev:          s.cfg.Dev,
		LoadBundle:   s.assets.useBundle(req),
		ManifestTags: s.manifestTags(),
		Version:      cmp.Or(s.cfg.Version, "dev"),
		Theme:        siteTheme(req),
		Canonical:    strings.TrimSuffix(s.cfg.BaseURL, "/") + req.URL.Path,
		SocialImage:  strings.TrimSuffix(s.cfg.BaseURL, "/") + "/og.png",
		JSONLD:       s.jsonLD(v),
		Heartbeat:    s.cfg.StreamHeartbeat(),
	}
}

// jsonLD describes the site (with its search, for search engines) and
// whatever the page adds (a component is SoftwareSourceCode). Pages that
// add nothing share one string (rendercache.go).
func (s *Server) jsonLD(v view) string {
	if v.LD != nil {
		return v.LD()
	}
	if len(v.Schema) == 0 {
		return s.cache.siteLD()
	}
	return s.buildJSONLD(v.Schema)
}

func (s *Server) buildJSONLD(schema []any) string {
	base := strings.TrimSuffix(s.cfg.BaseURL, "/")
	graph := []any{map[string]any{
		"@type":       "WebSite",
		"@id":         base + "/#website",
		"name":        "Starbase",
		"description": "Community Components for Rocket: a gallery of Datastar Rocket web components.",
		"url":         base + "/",
		"potentialAction": map[string]any{
			"@type":       "SearchAction",
			"target":      base + "/?q={search_term_string}",
			"query-input": "required name=search_term_string",
		},
		"publisher": map[string]any{"@id": orgID},
	}}
	graph = append(graph, schema...)
	b, _ := json.Marshal(map[string]any{"@context": "https://schema.org", "@graph": graph}) // escapes <, > and &
	return string(b)
}

// orgID names zwei und eins gmbh in JSON-LD. The company is described only on
// zweiundeins.gmbh; other sites refer to it by this @id.
const orgID = "https://zweiundeins.gmbh/#organization"

// softwareLD is the project itself, for the homepage's JSON-LD.
func (s *Server) softwareLD() map[string]any {
	base := strings.TrimSuffix(s.cfg.BaseURL, "/")
	return map[string]any{
		"@type":               "SoftwareSourceCode",
		"@id":                 base + "/#software",
		"name":                "Starbase",
		"description":         "Community Components for Rocket: a gallery of Datastar Rocket web components.",
		"url":                 base + "/",
		"codeRepository":      s.cfg.RepoURL,
		"programmingLanguage": []string{"Go", "TypeScript"},
		"license":             "https://opensource.org/licenses/MIT",
		"author":              map[string]any{"@type": "Organization", "@id": orgID, "name": "zwei und eins gmbh", "url": "https://zweiundeins.gmbh"},
		"publisher":           map[string]any{"@id": orgID},
		"maintainer":          map[string]any{"@id": orgID},
	}
}

// siteTheme is the theme the sb-theme-switch in the header stored in its
// cookie (a preference of the browser, not tab state), or "auto".
func siteTheme(r *http.Request) string {
	if c, err := r.Cookie("sb-theme"); err == nil && model.ValidSiteTheme(c.Value) {
		return c.Value
	}
	return "auto"
}

// loginURL is empty when nobody can sign in (production without OAuth).
func (s *Server) loginURL(next string) string {
	if s.oauth == nil && !s.cfg.Dev {
		return ""
	}
	return "/auth/login?next=" + url.QueryEscape(next)
}

func (s *Server) document(fn pageFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var v view
		var user *model.User
		err := s.q.View(r.Context(), func(rd *queries.Reader) (err error) {
			v, user, err = s.load(r.Context(), rd, r, "", fn)
			return err
		})
		if errors.Is(err, errNotFound) {
			s.notFound(w, r)
			return
		}
		if err != nil {
			s.fail(w, r, err)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Add("Vary", "Cookie")
		if v.Status != 0 {
			w.WriteHeader(v.Status)
		}
		sh := s.shell(r, user, v)
		ui.Document(sh, v.Body(sh)).Render(r.Context(), w)
	}
}

// MaxStreamsPerClient is how many render streams one client (edge.ClientKey:
// an IPv4 address, an IPv6 /64) may hold open (app.HubOptions), so that no
// one takes all of the hub's. A browser closes the stream of a hidden tab, so
// a person needs one per window; 24 leaves room for the people behind a
// shared address (carrier-grade NAT, an office, conference Wi-Fi).
const MaxStreamsPerClient = 24

// stream is the read side of CQRS for one tab: a view of its own in the
// render loop (lith/live), rendered again in the next tick after a commit
// touches the tab's session, the tab, or everything. Every frame is the
// whole #app, with the page's regions filled in by the stream; Datastar
// morphs the difference, and the stream's compressor (lith/sse), keeping
// its window across frames, makes resending cheap.
func (s *Server) stream(fn pageFunc, regions []*region) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var sig struct {
			TabID string `json:"tabid"`
			Boot  string `json:"boot"` // dev builds only
		}
		if err := datastar.ReadSignals(r, &sig); err != nil || !tabIDRe.MatchString(sig.TabID) {
			refuseStream(w)
			return
		}
		// Dev live reload rides on the render stream (one connection per tab):
		// after a restart the stream reconnects with the old boot id.
		if s.cfg.Dev && sig.Boot != "" && sig.Boot != s.boot {
			datastar.NewSSE(w, r).PatchElements(`<div hidden data-init="window.location.reload()"></div>`,
				datastar.WithSelector("body"), datastar.WithModeAppend())
			return
		}
		s.warmPreview(r)
		// The view renders on a render worker, also after this request has
		// ended: it keeps the request's values, not its cancellation.
		req := r.Clone(context.WithoutCancel(r.Context()))
		sid := edge.SessionID(r.Context())
		tab := live.View{
			Key:    "tab:" + sig.TabID,
			Topics: []string{commands.SessionTopic(sid), commands.TabTopic(sig.TabID), commands.TopicAll},
			Render: func(ctx context.Context, tx *sql.Tx) (any, error) {
				return s.frame(ctx, queries.Read(tx), req, sig.TabID, fn)
			},
		}
		spec := live.Spec{Client: edge.ClientKey(r), ID: streamID(sid, sig.TabID), Views: []live.View{tab}, Reconnect: ui.StreamID}
		for _, rg := range regions {
			spec.Views = append(spec.Views, rg.view)
		}
		spec.WriteTick = resetWatchdog(fillHoles(tab.Key, regions))
		k := commands.TabKey{SID: sid, Tab: sig.TabID}
		s.tabs.add(k)
		err := s.hub.Serve(w, r, spec)
		s.tabs.remove(k)
		switch {
		case errors.Is(err, live.ErrClientFull):
			// The hub answered 429, which retry 'error' asks again, as after its 503.
			s.capWarn.warn(s.log, "render stream refused: a client at its cap", "client", spec.Client, "cap", MaxStreamsPerClient)
		case errors.Is(err, sse.ErrNoRoom):
			st := s.hub.Budget().Stats()
			s.roomWarn.warn(s.log, "render stream refused: the streams' budget is full", "path", r.URL.Path,
				"streams", s.hub.Stats().Streams, "reserved_mib", st.Reserved>>20, "limit_mib", st.Limit>>20, "refused", st.Refused)
		case errors.Is(err, live.ErrFull):
			s.fullWarn.warn(s.log, "render stream refused: too many open", "path", r.URL.Path)
		case err != nil && !errors.Is(err, live.ErrStopped):
			s.log.Debug("render stream ended", "path", r.URL.Path, "err", err)
		}
	}
}

// streamRequest reports whether r opens a page's render stream: streamInit
// accepts nothing but an event stream, where commands take Datastar's list.
func streamRequest(r *http.Request) bool {
	return r.Method == http.MethodPost && r.Header.Get("Accept") == "text/event-stream"
}

// refuseStream refuses a stream for good. retry 'error' repeats every 4xx
// and 5xx forever, and stops at a 204.
func refuseStream(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

// frame renders a tab's #app as a patch. A page that is gone (a component
// a deploy removed) renders the not-found page, as its document would.
func (s *Server) frame(ctx context.Context, rd *queries.Reader, req *http.Request, tabID string, fn pageFunc) ([]byte, error) {
	v, user, err := s.load(ctx, rd, req, tabID, fn)
	if errors.Is(err, errNotFound) {
		v, user, err = s.load(ctx, rd, req, tabID, notFoundPage)
	}
	if err != nil {
		return nil, err
	}
	buf := frameBufs.Get().(*bytes.Buffer)
	defer putFrameBuf(buf)
	sh := s.shell(req, user, v)
	if err := ui.App(sh, v.Body(sh)).Render(ctx, buf); err != nil {
		return nil, err
	}
	return sse.PatchElements(buf.Bytes()), nil
}

// frameBufs keep render buffers between frames: PatchElements copies the
// HTML, and a buffer grown afresh made twice a frame's size in garbage.
var frameBufs = sync.Pool{New: func() any { return new(bytes.Buffer) }}

func putFrameBuf(b *bytes.Buffer) {
	if b.Cap() > 1<<20 {
		return
	}
	b.Reset()
	frameBufs.Put(b)
}

func (s *Server) fail(w http.ResponseWriter, r *http.Request, err error) {
	s.log.Error("request failed", "path", r.URL.Path, "err", err)
	http.Error(w, "Something went wrong. Please try again.", http.StatusInternalServerError)
}

func (s *Server) notFound(w http.ResponseWriter, r *http.Request) {
	if streamRequest(r) { // a 404 page's stream, or a route a deploy removed
		refuseStream(w)
		return
	}
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.NotFound(w, r)
		return
	}
	s.document(notFoundPage)(w, r)
}

func notFoundPage(*renderCtx) (view, error) {
	return view{
		Title:  "Lost in space · Starbase",
		Body:   func(ui.Shell) templ.Component { return ui.NotFound() },
		Status: http.StatusNotFound,
	}, nil
}

// pageSignals adds the dev boot id (used for live reload) to a page's signals.
func (s *Server) pageSignals(extra map[string]any) map[string]any {
	if !s.cfg.Dev {
		return extra
	}
	out := map[string]any{"boot": s.boot}
	for k, v := range extra {
		out[k] = v
	}
	return out
}
