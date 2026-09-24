package web

import (
	"crypto/sha256"
	"net/http"
	"strings"
	"time"

	"github.com/starfederation/datastar-go/datastar"

	"starbase/internal/commands"
	"starbase/internal/ui"
)

// POST /playground/size measures the playground's component.js the way the
// catalog measures its modules (esbuild, brotli -11): the size line in the
// playground's top bar. It is a query (nothing is stored) that answers with
// a patch of the page-local $_size signal. The code travels in the request
// body, since the editor is a client island.
func (s *Server) playgroundSize(w http.ResponseWriter, r *http.Request) {
	var p struct {
		Component string `json:"component"`
		Code      string `json:"code"`
	}
	if err := datastar.ReadSignals(r, &p); err != nil || len(p.Code) > commands.MaxSnippetBytes {
		http.Error(w, "bad payload", http.StatusBadRequest)
		return
	}
	now := time.Now()
	if !s.sizeLimit.allow(sessionID(r), 1, now) || !s.sizeLimitIP.allow(clientIP(r), 1, now) {
		http.Error(w, "measuring too often", http.StatusTooManyRequests)
		return
	}
	datastar.NewSSE(w, r).MarshalAndPatchSignals(map[string]any{"_size": s.measure(p.Component, p.Code)})
}

// measure returns the $_size patch: the size line's texts, formatted here.
// own is component.js minified under brotli (what a page downloads), min
// minified, raw as written, total with what it imports and renders ("" when
// nothing), title the whole breakdown. When the code doesn't parse, only
// stale is set: the patch merges, so the last good numbers stay.
//
// It is memoized by content (a page render measures its initial code,
// and pages re-render on every broadcast), in a small map that is simply
// dropped when full.
func (s *Server) measure(slug, code string) map[string]any {
	key := sha256.Sum256([]byte(slug + "\x00" + code))
	s.sizeMu.Lock()
	v, ok := s.sizes[key]
	s.sizeMu.Unlock()
	if ok {
		return v
	}
	c, _ := s.catalog.Get(slug)
	m, err := s.catalog.Measure([]byte(code), c)
	if err != nil {
		v = map[string]any{"stale": true, "title": "The code doesn't parse yet: these are the last sizes that did. " + firstLine(err.Error())}
	} else {
		own, min, raw, total := ui.FmtBytes(m.MinBrotli), ui.FmtBytes(m.Min), ui.FmtBytes(m.Raw), ""
		title := "component.js: " + own + " minified with brotli (what a page downloads), " + min + " minified, " + raw + " as written."
		if m.Extra > 0 {
			total = ui.FmtBytes(m.Total())
			with := "the files it imports"
			if len(m.Uses) > 0 {
				with = strings.Join(m.Uses, ", ")
			}
			title += " With " + with + ": " + total + "."
		}
		v = map[string]any{"own": own, "min": min, "raw": raw, "total": total, "title": title, "stale": false}
	}
	s.sizeMu.Lock()
	if len(s.sizes) >= 256 {
		clear(s.sizes)
	}
	s.sizes[key] = v
	s.sizeMu.Unlock()
	return v
}

func firstLine(s string) string {
	s, _, _ = strings.Cut(s, "\n")
	return s
}
