package web

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"starbase/components"
	"starbase/content"
	"starbase/internal/catalog"
	"starbase/internal/commands"
	"starbase/internal/config"
	"starbase/internal/db"
	"starbase/internal/queries"
	"starbase/internal/ui"
	"starbase/static"

	"github.com/zweiundeins/lith/live"
	"github.com/zweiundeins/lith/writer"
)

// renderPages are the pages the render benchmarks cover: a pattern's
// pageFunc and a path to request it with.
func (s *Server) renderPages() []struct {
	name, path string
	fn         pageFunc
} {
	return []struct {
		name, path string
		fn         pageFunc
	}{
		{"gallery", "/", s.galleryPage},
		{"data-table", "/components/data-table", s.componentPage},
		{"select", "/components/select", s.componentPage},
		{"kanban-board", "/components/kanban-board", s.componentPage},
		{"playground", "/playground", s.codePlaygroundPage},
		{"showcase", "/showcase", s.showcasePage},
		{"themes", "/themes", s.themesPage},
	}
}

func newBenchServer(tb testing.TB) *Server {
	tb.Helper()
	return newBenchServerWith(tb, true)
}

// newBenchServerWith is a server on a new database, with the catalog
// synced (and the demo data seeded) or not.
func newBenchServerWith(tb testing.TB, synced bool) *Server {
	tb.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	tb.Cleanup(cancel)
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	d, err := db.Open(ctx, filepath.Join(tb.TempDir(), "bench.db"))
	if err != nil {
		tb.Fatal(err)
	}
	tb.Cleanup(func() { d.Close() })
	cat, err := catalog.Load(components.FS)
	if err != nil {
		tb.Fatal(err)
	}
	cfg := config.Config{BaseURL: "https://starbase.example", RepoURL: "https://example.com/repo"}
	hub := live.New(live.Options{Read: d.R, Log: log}) // a frame renders the same on any hub
	hubDone := make(chan struct{})
	go func() { hub.Run(ctx); close(hubDone) }()
	tb.Cleanup(func() { cancel(); <-hubDone })
	wr, err := writer.New(d, writer.Options{OnCommit: hub.Changed, Log: log})
	if err != nil {
		tb.Fatal(err)
	}
	go wr.Run(ctx)
	tb.Cleanup(func() { wr.Close() })
	if synced {
		if err := wr.Do(ctx, commands.SyncCatalog{Catalog: cat, Datastar: static.Datastar()}); err != nil {
			tb.Fatal(err)
		}
		if err := wr.Do(ctx, commands.SeedDemo{}); err != nil {
			tb.Fatal(err)
		}
	}
	return New(ctx, Deps{Config: cfg, Log: log, Writer: wr, Hub: hub, Queries: queries.New(d),
		Catalog: cat, StaticFS: static.FS, Content: content.FS})
}

// frameRequest is a render stream's request, without the session and the
// nonce the middleware would add (lith/edge keeps their context keys).
func frameRequest(path string) *http.Request {
	r := httptest.NewRequest("POST", path, nil)
	if slug, ok := strings.CutPrefix(path, "/components/"); ok {
		r.SetPathValue("slug", slug)
	}
	return r
}

// renderFrame is one frame of a page's render stream, without the SSE
// encoding and compression.
func (s *Server) renderFrame(r *http.Request, fn pageFunc, buf *bytes.Buffer) error {
	return s.q.View(r.Context(), func(rd *queries.Reader) error {
		v, user, err := s.load(r.Context(), rd, r, "benchtab0000", fn)
		if err != nil {
			return err
		}
		buf.Reset()
		sh := s.shell(r, user, v)
		return ui.App(sh, v.Body(sh)).Render(r.Context(), buf)
	})
}

// BenchmarkRenderFrame measures one stream frame of each page; with
// -memprofile and -cpuprofile it shows where a render spends.
func BenchmarkRenderFrame(b *testing.B) {
	s := newBenchServer(b)
	for _, p := range s.renderPages() {
		b.Run(p.name, func(b *testing.B) {
			r := frameRequest(p.path)
			var buf bytes.Buffer
			if err := s.renderFrame(r, p.fn, &buf); err != nil {
				b.Fatal(err)
			}
			b.SetBytes(int64(buf.Len()))
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				if err := s.renderFrame(r, p.fn, &buf); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkRenderGET measures each page's document through the handler
// (middleware included), uncompressed.
func BenchmarkRenderGET(b *testing.B) {
	s := newBenchServer(b)
	h := s.Handler()
	for _, p := range s.renderPages() {
		b.Run(p.name, func(b *testing.B) {
			first := httptest.NewRecorder()
			h.ServeHTTP(first, httptest.NewRequest("GET", p.path, nil))
			if first.Code != 200 {
				b.Fatalf("%s: %d", p.path, first.Code)
			}
			cookies := first.Result().Cookies()
			b.SetBytes(int64(first.Body.Len()))
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				req := httptest.NewRequest("GET", p.path, nil)
				for _, c := range cookies {
					req.AddCookie(c)
				}
				h.ServeHTTP(discard{http.Header{}}, req)
			}
		})
	}
}

// discard is a ResponseWriter that drops the body, so the GET benchmark
// doesn't count a recorder's buffer.
type discard struct{ h http.Header }

func (d discard) Header() http.Header         { return d.h }
func (d discard) Write(b []byte) (int, error) { return len(b), nil }
func (d discard) WriteHeader(int)             {}

// BenchmarkStreamFrame measures a stream's frame as the hub renders it:
// the page and its patch event, uncompressed.
func BenchmarkStreamFrame(b *testing.B) {
	s := newBenchServer(b)
	for _, p := range s.renderPages() {
		b.Run(p.name, func(b *testing.B) {
			r := frameRequest(p.path)
			frame := func() int {
				var n int
				if err := s.q.View(r.Context(), func(rd *queries.Reader) error {
					ev, err := s.frame(r.Context(), rd, r, "benchtab0000", p.fn)
					n = len(ev)
					return err
				}); err != nil {
					b.Fatal(err)
				}
				return n
			}
			b.SetBytes(int64(frame()))
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				frame()
			}
		})
	}
}
