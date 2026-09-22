package web

import (
	"net/http"

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
			"application/manifest+json",
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
		compressed.ServeHTTP(w, r)
	})
}
