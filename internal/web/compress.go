package web

import (
	"context"
	"net/http"
	"time"

	"github.com/CAFxX/httpcompression"
)

// compress negotiates brotli, zstd or gzip for documents and text assets.
// Only GET and HEAD: the render streams (POST) compress themselves (brotli
// across frames) and must never be buffered by a middleware.
func compress(next http.Handler) http.Handler {
	adapter, err := httpcompression.DefaultAdapter(
		httpcompression.ContentTypes([]string{
			"text/html", "text/css", "text/javascript", "application/javascript",
			"application/json", "image/svg+xml", "text/plain", "application/xml",
			"application/manifest+json", "text/csv",
		}, false),
		httpcompression.BrotliCompressionLevel(5), // fast enough per request, close to max for text
	)
	if err != nil {
		panic(err) // only with invalid options
	}
	compressed := adapter(next)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			next.ServeHTTP(w, r)
			return
		}
		compressed.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), rawWriterKey{}, w)))
	})
}

// rawWriterKey holds the ResponseWriter under the compressor's, which has no
// Unwrap for http.ResponseController.
type rawWriterKey struct{}

// setWriteDeadline sets the write deadline of the connection under w.
func setWriteDeadline(w http.ResponseWriter, r *http.Request, t time.Time) error {
	if raw, ok := r.Context().Value(rawWriterKey{}).(http.ResponseWriter); ok {
		w = raw
	}
	return http.NewResponseController(w).SetWriteDeadline(t)
}
