package web

import (
	"cmp"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"maps"
	"net/http"
	"strings"
	"time"

	"github.com/starfederation/datastar-go/datastar"

	"starbase/internal/commands"
	"starbase/internal/tscheck"
)

// POST /playground/check type-checks the playground's component.ts, or its
// component.js when it opts in with // @ts-check, against the patched
// Datastar build and the files of the component it was opened from
// (internal/tscheck). It is a query, like /playground/size, and answers with
// a patch of $_diag: the diagnostics per file as JSON, the playground's
// diagnostics attribute. A string, because a patched object would merge
// into the last one.
func (s *Server) playgroundCheck(w http.ResponseWriter, r *http.Request) {
	var p struct {
		Component string `json:"component"`
		Name      string `json:"name"`
		Code      string `json:"code"`
	}
	if err := datastar.ReadSignals(r, &p); err != nil || len(p.Code) > commands.MaxSnippetBytes || !mainFile(p.Name) {
		http.Error(w, "bad payload", http.StatusBadRequest)
		return
	}
	now := time.Now()
	if !s.checkLimit.allow(sessionID(r), 1, now) || !s.checkLimitIP.allow(clientIP(r), 1, now) {
		http.Error(w, "checking too often", http.StatusTooManyRequests)
		return
	}
	name := cmp.Or(p.Name, "component.js")
	ds, _ := json.Marshal(map[string]any{name: s.typecheck(r.Context(), p.Component, name, p.Code)})
	datastar.NewSSE(w, r).MarshalAndPatchSignals(map[string]any{"_diag": string(ds)})
}

// typecheck returns the diagnostics of code, as file name in the folder of
// the component slug: none when there is no compiler, or when JavaScript
// doesn't ask for a check. Results are memoized by content, like sizes.
func (s *Server) typecheck(ctx context.Context, slug, name, code string) []tscheck.Diagnostic {
	if s.checker == nil || (strings.HasSuffix(name, ".js") && !strings.Contains(code, "@ts-check")) {
		return []tscheck.Diagnostic{}
	}
	key := sha256.Sum256([]byte(slug + "\x00" + name + "\x00" + code))
	s.checkMu.Lock()
	ds, ok := s.checks[key]
	s.checkMu.Unlock()
	if ok {
		return ds
	}
	files := []tscheck.File{{Name: name, Source: code}}
	if c, ok := s.catalog.Get(slug); ok {
		// The code stands in for the component's own module: its other modules,
		// and their TypeScript (helpers, declarations) when it has any.
		mods, _ := s.catalog.ModuleFiles(c)
		srcs, _ := s.catalog.TypeScriptFiles(c)
		maps.Copy(mods, srcs)
		for n, b := range mods {
			if n != strings.TrimPrefix(c.Script, c.Slug+"/") && n != strings.TrimPrefix(c.SourceFile, c.Slug+"/") {
				files = append(files, tscheck.File{Name: n, Source: string(b)})
			}
		}
	}
	ds, err := s.checker.Check(ctx, files)
	switch {
	case errors.Is(err, tscheck.ErrRefused):
		ds = []tscheck.Diagnostic{{Line: 1, Col: 1, Length: 1, Message: "The type check reads only this code, the files of its component and 'datastar'."}}
	case err != nil:
		s.log.Warn("type check", "component", slug, "err", err)
		return []tscheck.Diagnostic{}
	case ds == nil:
		ds = []tscheck.Diagnostic{}
	}
	s.checkMu.Lock()
	if len(s.checks) >= 256 {
		clear(s.checks)
	}
	s.checks[key] = ds
	s.checkMu.Unlock()
	return ds
}
