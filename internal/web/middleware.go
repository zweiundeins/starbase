package web

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

type ctxKey int

const (
	sidKey ctxKey = iota
	nonceKey
)

func randomToken(n int) string {
	b := make([]byte, n)
	rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

var sidRe = regexp.MustCompile(`^[A-Za-z0-9_-]{27}$`)

func sessionID(r *http.Request) string {
	sid, _ := r.Context().Value(sidKey).(string)
	return sid
}

func nonce(r *http.Request) string {
	n, _ := r.Context().Value(nonceKey).(string)
	return n
}

// cookieName is __Host- prefixed when served over HTTPS.
func (s *Server) cookieName() string {
	if s.secure {
		return "__Host-sid"
	}
	return "sid"
}

func (s *Server) setSessionCookie(w http.ResponseWriter, sid string) {
	http.SetCookie(w, &http.Cookie{
		Name:     s.cookieName(),
		Value:    sid,
		Path:     "/",
		MaxAge:   int((400 * 24 * time.Hour).Seconds()),
		HttpOnly: true,
		Secure:   s.secure,
		SameSite: http.SameSiteLaxMode,
	})
}

// session gives every browser an opaque random id. Nothing is written to
// the database until the browser signs in or changes UI state.
func (s *Server) session(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var sid string
		if c, err := r.Cookie(s.cookieName()); err == nil && sidRe.MatchString(c.Value) {
			sid = c.Value
		} else {
			sid = randomToken(20)
			s.setSessionCookie(w, sid)
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), sidKey, sid)))
	})
}

// sameOrigin rejects cross-site state-changing requests (CSRF) and caps
// request bodies. Browsers send Sec-Fetch-Site on every request; for older
// clients the Origin header must match.
func (s *Server) sameOrigin(next http.Handler) http.Handler {
	origin := s.cfg.BaseURL
	if u, err := url.Parse(origin); err == nil {
		origin = u.Scheme + "://" + u.Host
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			site := r.Header.Get("Sec-Fetch-Site")
			ok := site == "same-origin" || (site == "" && r.Header.Get("Origin") == origin)
			if !ok {
				http.Error(w, "cross-site request rejected", http.StatusForbidden)
				return
			}
			limit := int64(64 << 10)
			switch {
			case strings.HasPrefix(r.URL.Path, "/dev/"):
				limit = 4 << 20
			case r.URL.Path == "/cmd/snippet":
				limit = 512 << 10 // 64 KB of code, JSON-escaped
			}
			r.Body = http.MaxBytesReader(w, r.Body, limit)
		}
		next.ServeHTTP(w, r)
	})
}

// securityHeaders sets a strict CSP with a per-request nonce (for the
// import map). Datastar evaluates expressions, which needs 'unsafe-eval';
// Rocket renders <style> into shadow roots, which needs inline styles.
func (s *Server) securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := randomToken(16)
		h := w.Header()
		h.Set("Content-Security-Policy", fmt.Sprintf("default-src 'self'; "+
			"script-src 'self' 'nonce-%s' 'unsafe-eval'; "+
			"style-src 'self' 'unsafe-inline'; "+
			"img-src 'self' data: https://github.com https://avatars.githubusercontent.com; "+
			"font-src 'self'; connect-src 'self'; object-src 'none'; base-uri 'self'; "+
			"form-action 'self'; frame-ancestors 'none'", n))
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Cross-Origin-Opener-Policy", "same-origin")
		if s.secure {
			h.Set("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), nonceKey, n)))
	})
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

func (s *Server) recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if v := recover(); v != nil {
				if v == http.ErrAbortHandler {
					panic(v)
				}
				s.log.Error("panic", "path", r.URL.Path, "err", v)
				http.Error(w, "internal error", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}
