package web

import (
	"bytes"
	"cmp"
	"context"
	"errors"
	"hash/fnv"
	"net/http"
	"net/url"
	"regexp"

	"github.com/a-h/templ"
	"github.com/starfederation/datastar-go/datastar"

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
}

// renderCtx is the input of a page render: one read snapshot plus who is
// looking and what their tab state is.
type renderCtx struct {
	ctx   context.Context
	r     *queries.Reader
	req   *http.Request
	sid   string
	tabID string
	user  *model.User
	tab   model.TabState // stored state, or seeded from the query string
}

type pageFunc func(rc *renderCtx) (view, error)

var errNotFound = errors.New("not found")

var tabIDRe = regexp.MustCompile(`^[a-z0-9]{8,32}$`)

// page registers GET (document) and POST (render stream) for a pattern.
func (s *Server) page(mux *http.ServeMux, pattern string, fn pageFunc) {
	mux.HandleFunc("GET "+pattern, s.document(fn))
	mux.HandleFunc("POST "+pattern, s.stream(fn))
}

// load runs one render against a fresh read snapshot.
func (s *Server) load(ctx context.Context, req *http.Request, tabID string, fn pageFunc) (v view, user *model.User, err error) {
	sid := sessionID(req)
	err = s.q.View(ctx, func(r *queries.Reader) error {
		var err error
		if user, err = r.SessionUser(ctx, sid); err != nil {
			return err
		}
		tab, ok, err := r.Tab(ctx, sid, tabID)
		if err != nil {
			return err
		}
		if !ok {
			tab = model.TabState{Browse: model.BrowseFromQuery(req.URL.Query())}
		}
		v, err = fn(&renderCtx{ctx: ctx, r: r, req: req, sid: sid, tabID: tabID, user: user, tab: tab})
		return err
	})
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
		Nonce:        nonce(req),
		Boot:         s.boot,
		Dev:          s.cfg.Dev,
		ManifestTags: s.manifestTags(),
		Version:      cmp.Or(s.cfg.Version, "dev"),
	}
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
		v, user, err := s.load(r.Context(), r, "", fn)
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
		w.Header().Set("Vary", "Cookie")
		if v.Status != 0 {
			w.WriteHeader(v.Status)
		}
		sh := s.shell(r, user, v)
		ui.Document(sh, v.Body(sh)).Render(r.Context(), w)
	}
}

// stream is the read side of CQRS for one tab: it renders the page, then
// waits for the hub to signal a commit that concerns this tab, and renders
// again. Every frame is the whole #app; Datastar morphs the difference and
// Brotli, sharing one window across frames, makes resending cheap.
func (s *Server) stream(fn pageFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var sig struct {
			TabID string `json:"tabid"`
			Boot  string `json:"boot"` // dev builds only
		}
		if err := datastar.ReadSignals(r, &sig); err != nil || !tabIDRe.MatchString(sig.TabID) {
			http.Error(w, "bad stream request", http.StatusBadRequest)
			return
		}
		// Dev live reload rides on the render stream (one connection per tab):
		// after a restart the stream reconnects with the old boot id.
		if s.cfg.Dev && sig.Boot != "" && sig.Boot != s.boot {
			datastar.NewSSE(w, r).PatchElements(`<div hidden data-init="window.location.reload()"></div>`,
				datastar.WithSelector("body"), datastar.WithModeAppend())
			return
		}
		sub := s.hub.Subscribe(sessionID(r), sig.TabID, r.URL.Path)
		defer s.hub.Unsubscribe(sub)

		sse := datastar.NewSSE(w, r, datastar.WithCompression(
			datastar.WithBrotli(datastar.WithBrotliLevel(5)),
			datastar.WithGzip(),
		))
		var last uint64
		var buf bytes.Buffer
		for {
			v, user, err := s.load(r.Context(), r, sig.TabID, fn)
			if err != nil {
				if r.Context().Err() == nil {
					s.log.Error("stream render", "path", r.URL.Path, "err", err)
				}
				return
			}
			buf.Reset()
			sh := s.shell(r, user, v)
			if err := ui.App(sh, v.Body(sh)).Render(r.Context(), &buf); err != nil {
				return
			}
			// Skip frames identical to the last one sent.
			h := fnv.New64a()
			h.Write(buf.Bytes())
			if sum := h.Sum64(); sum != last {
				last = sum
				if err := sse.PatchElements(buf.String()); err != nil {
					return
				}
			}
			select {
			case <-sub.C:
			case <-r.Context().Done():
				return
			case <-s.ctx.Done():
				return
			}
		}
	}
}

func (s *Server) fail(w http.ResponseWriter, r *http.Request, err error) {
	s.log.Error("request failed", "path", r.URL.Path, "err", err)
	http.Error(w, "Something went wrong. Please try again.", http.StatusInternalServerError)
}

func (s *Server) notFound(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.NotFound(w, r)
		return
	}
	s.document(func(rc *renderCtx) (view, error) {
		return view{
			Title:  "Lost in space · Starbase",
			Body:   func(ui.Shell) templ.Component { return ui.NotFound() },
			Status: http.StatusNotFound,
		}, nil
	})(w, r)
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
