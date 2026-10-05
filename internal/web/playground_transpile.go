package web

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"starbase/internal/catalog"
	"starbase/internal/commands"
)

// POST /playground/transpile turns the playground's component.ts into the
// JavaScript the runner imports (catalog.Transpile), with the line of the
// TypeScript each of its lines comes from. The runner calls it from its
// sandbox, an opaque origin: the answer is a pure function of the body, so
// it takes cross-site requests (the CSRF check lets it through) from any
// origin, rate-limited per client.
func (s *Server) playgroundTranspile(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	src, err := io.ReadAll(io.LimitReader(r.Body, commands.MaxSnippetBytes+1))
	if err != nil || len(src) > commands.MaxSnippetBytes {
		http.Error(w, "bad body", http.StatusBadRequest)
		return
	}
	if !s.transpileLimit.allow(clientIP(r), 1, time.Now()) {
		http.Error(w, "transpiling too often", http.StatusTooManyRequests)
		return
	}
	key := sha256.Sum256(src)
	s.transpileMu.Lock()
	out, ok := s.transpiled[key]
	s.transpileMu.Unlock()
	if !ok {
		t, err := catalog.Transpile(string(src))
		var te *catalog.TranspileError
		if errors.As(err, &te) {
			out, _ = json.Marshal(te)
		} else {
			out, _ = json.Marshal(t)
		}
		s.transpileMu.Lock()
		if len(s.transpiled) >= 256 {
			clear(s.transpiled)
		}
		s.transpiled[key] = out
		s.transpileMu.Unlock()
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.Write(out)
}
