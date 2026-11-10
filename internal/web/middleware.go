package web

import (
	"crypto/rand"
	"encoding/base64"
	"log/slog"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/zweiundeins/lith/edge"
)

func randomToken(n int) string {
	b := make([]byte, n)
	rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

// legacySessionID matches the 160-bit session ids starbase made before it
// took edge.Sessions, which makes 256-bit ones: they stay valid.
var legacySessionID = regexp.MustCompile(`^[A-Za-z0-9_-]{27}$`)

// sameOrigin rejects cross-site state-changing requests (CSRF) and caps
// request bodies. Browsers send Sec-Fetch-Site on every request; for older
// clients the Origin header must match.
func (s *Server) sameOrigin(next http.Handler) http.Handler {
	origin := s.cfg.BaseURL
	if u, err := url.Parse(origin); err == nil {
		origin = u.Scheme + "://" + u.Host
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// OPTIONS is a CORS preflight: it never changes state.
		if r.Method != http.MethodGet && r.Method != http.MethodHead && r.Method != http.MethodOptions {
			site := r.Header.Get("Sec-Fetch-Site")
			ok := site == "same-origin" || (site == "" && r.Header.Get("Origin") == origin)
			// The playground's runner, an opaque origin, transpiles there: a pure function, no state.
			if !ok && r.URL.Path != "/playground/transpile" {
				if streamRequest(r) {
					refuseStream(w)
					return
				}
				http.Error(w, "cross-site request rejected", http.StatusForbidden)
				return
			}
			// Large bodies may outlast the server's ReadTimeout on a slow link.
			upload := int64(0)
			switch {
			case strings.HasPrefix(r.URL.Path, "/dev/"):
				upload = 4 << 20
			case r.URL.Path == "/cmd/snippet", r.URL.Path == "/playground/size", r.URL.Path == "/playground/check":
				upload = 512 << 10 // 64 KB of code, JSON-escaped
			}
			if upload == 0 {
				r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
			} else {
				body, err := edge.UploadBody(w, r, edge.UploadMinRate, upload)
				if err != nil {
					s.log.Warn("upload without a deadline", "path", r.URL.Path, "err", err)
				}
				r.Body = body
			}
		}
		next.ServeHTTP(w, r)
	})
}

var hostRe = regexp.MustCompile(`^[A-Za-z0-9.-]+(:[0-9]+)?$|^\[[0-9A-Fa-f:.]+\](:[0-9]+)?$`)

// csp is a request's Content-Security-Policy. Datastar evaluates
// expressions, which needs 'unsafe-eval'; Rocket renders <style> into shadow
// roots, which needs inline styles.
func (s *Server) csp(r *http.Request) edge.CSP {
	scheme := "http"
	if s.secure {
		scheme = "https"
	}
	origin := scheme + "://" + r.Host
	if !hostRe.MatchString(r.Host) {
		origin = strings.TrimSuffix(s.cfg.BaseURL, "/")
	}
	return edge.CSP{
		"default-src": {"'self'"},
		// Scripts only from the two folders that hold them, not the whole
		// origin: other paths serve content that isn't ours to trust
		// (e.g. pull request previews for the playground's sandbox).
		"script-src":      {origin + "/static/", origin + "/c/", edge.NonceSource, "'unsafe-eval'"},
		"style-src":       {"'self'", "'unsafe-inline'"},
		"img-src":         {"'self'", "data:", "https://github.com", "https://avatars.githubusercontent.com"},
		"font-src":        {"'self'"},
		"connect-src":     {"'self'"},
		"object-src":      {"'none'"},
		"base-uri":        {"'self'"},
		"form-action":     {"'self'"},
		"frame-ancestors": {"'none'"},
	}
}

func (s *Server) logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		if d := time.Since(start); d > 500*time.Millisecond && r.Header.Get("Accept") != "text/event-stream" {
			s.log.Debug("slow request", slog.String("method", r.Method), slog.String("path", r.URL.Path), slog.Duration("took", d))
		}
	})
}
