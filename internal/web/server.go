// Package web is the HTTP edge. It maps requests to queries (pages and their
// render streams) and to commands (short POSTs that return 204).
package web

import (
	"context"
	"io/fs"
	"log/slog"
	"net/http"
	"strings"

	"starbase/internal/catalog"
	"starbase/internal/config"
	"starbase/internal/cqrs"
	"starbase/internal/queries"
	"starbase/internal/ui"
)

type Server struct {
	ctx     context.Context // cancelled on shutdown; ends render streams
	cfg     config.Config
	log     *slog.Logger
	bus     *cqrs.Bus
	hub     *cqrs.Hub
	q       *queries.Queries
	catalog *catalog.Catalog
	content fs.FS
	assets  *assets
	oauth   *oauthConfig
	boot    string
	secure  bool
}

type Deps struct {
	Config   config.Config
	Log      *slog.Logger
	Bus      *cqrs.Bus
	Hub      *cqrs.Hub
	Queries  *queries.Queries
	Catalog  *catalog.Catalog
	StaticFS fs.FS
	Content  fs.FS
}

func New(ctx context.Context, d Deps) *Server {
	s := &Server{
		ctx:     ctx,
		cfg:     d.Config,
		log:     d.Log,
		bus:     d.Bus,
		hub:     d.Hub,
		q:       d.Queries,
		catalog: d.Catalog,
		content: d.Content,
		assets:  newAssets(d.StaticFS, d.Catalog, d.Config.Dev),
		boot:    strings.ToLower(randomToken(6)),
		secure:  strings.HasPrefix(d.Config.BaseURL, "https://"),
	}
	s.oauth = newOAuth(d.Config)
	return s
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	// Queries: every page is a GET (full document) plus a POST to the same
	// URL that opens the tab's render stream.
	s.page(mux, "/{$}", s.galleryPage)
	s.page(mux, "/components/{slug}", s.componentPage)
	s.page(mux, "/themes", s.themesPage)
	s.page(mux, "/contribute", s.contentPage("contribute", "contribute"))
	s.page(mux, "/about", s.contentPage("about", "about"))
	s.page(mux, "/showcase", s.contentPage("showcase", "showcase", ui.MissionControl()))
	mux.Handle("GET /components", http.RedirectHandler("/", http.StatusMovedPermanently))
	// The easiest way to contribute: a GitHub issue form, turned into a PR by a bot.
	mux.Handle("GET /submit", http.RedirectHandler(strings.TrimSuffix(s.cfg.RepoURL, "/")+"/issues/new?template=new-component.yml", http.StatusSeeOther))

	// Commands.
	mux.HandleFunc("POST /cmd/browse", s.cmdBrowse)
	mux.HandleFunc("POST /cmd/star/{slug}", s.cmdStar(true))
	mux.HandleFunc("POST /cmd/unstar/{slug}", s.cmdStar(false))
	mux.HandleFunc("POST /cmd/theme/{theme}", s.cmdTheme)

	// Auth.
	mux.HandleFunc("GET /auth/login", s.login)
	mux.HandleFunc("GET /auth/github/callback", s.githubCallback)
	mux.HandleFunc("POST /auth/logout", s.logout)

	// Assets.
	mux.Handle("GET /static/", s.assets.serveStatic())
	mux.HandleFunc("GET /art/{name}", s.assets.serveArt)
	mux.HandleFunc("GET /c/{path...}", s.assets.serveComponents)
	// Demo data: a read-only signal stream for the live examples.
	mux.HandleFunc("GET /demo/telemetry", s.demoTelemetry)

	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("ok")) })

	if s.cfg.Dev {
		s.devRoutes(mux)
	}
	mux.HandleFunc("/", s.notFound)

	var h http.Handler = mux
	h = s.sameOrigin(h)
	h = s.session(h)
	h = s.securityHeaders(h)
	h = s.logRequests(h)
	h = s.recoverer(h)
	return h
}
