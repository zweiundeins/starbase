package web

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/starfederation/datastar-go/datastar"
	"github.com/zweiundeins/lith/edge"
	"github.com/zweiundeins/lith/writer"

	"starbase/internal/commands"
	"starbase/internal/model"
	"starbase/internal/queries"
)

// send enqueues a command and answers 204. The tab's render stream shows
// the outcome; commands never return HTML.
func (s *Server) send(w http.ResponseWriter, r *http.Request, cmd writer.Command) {
	err := s.writer.Send(cmd)
	switch {
	case err == nil:
		w.WriteHeader(http.StatusNoContent)
	case errors.Is(err, writer.ErrInvalid):
		http.Error(w, err.Error(), http.StatusBadRequest)
	default:
		s.log.Warn("command rejected", "err", err)
		w.WriteHeader(http.StatusServiceUnavailable)
	}
}

// browseSignals reads the gallery's signals: the tab and its Browse.
func browseSignals(r *http.Request) (tabID string, b model.Browse, err error) {
	var sig struct {
		TabID string `json:"tabid"`
		Q     string `json:"q"`
		Cat   string `json:"cat"`
		Sort  string `json:"sort"`
	}
	err = datastar.ReadSignals(r, &sig)
	return sig.TabID, model.Browse{Q: sig.Q, Category: sig.Cat, Sort: model.Sort(sig.Sort)}, err
}

// cmdBrowse is a search or a category: this page only.
func (s *Server) cmdBrowse(w http.ResponseWriter, r *http.Request) {
	tabID, b, err := browseSignals(r)
	if err != nil {
		http.Error(w, "bad signals", http.StatusBadRequest)
		return
	}
	s.send(w, r, commands.SetBrowseFilter{SID: edge.SessionID(r.Context()), TabID: tabID, Browse: b})
}

// cmdSort is a chosen sort: this page, and the session's default sort.
func (s *Server) cmdSort(w http.ResponseWriter, r *http.Request) {
	tabID, b, err := browseSignals(r)
	if err != nil {
		http.Error(w, "bad signals", http.StatusBadRequest)
		return
	}
	s.send(w, r, commands.SetGallerySort{SID: edge.SessionID(r.Context()), TabID: tabID, Browse: b})
}

func (s *Server) cmdStar(star bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, err := s.q.ViewValue(r.Context(), func(rd *queries.Reader) (*model.User, error) {
			return rd.SessionUser(r.Context(), edge.SessionID(r.Context()))
		})
		if err != nil {
			s.fail(w, r, err)
			return
		}
		if user == nil {
			// Not signed in: send the browser to GitHub and back.
			if u := s.loginURL(localPath(r.Referer())); u != "" {
				datastar.NewSSE(w, r).Redirect(u)
				return
			}
			http.Error(w, "sign-in is not available", http.StatusForbidden)
			return
		}
		slug := r.PathValue("slug")
		if star {
			s.send(w, r, commands.Star{UserID: user.ID, Slug: slug})
		} else {
			s.send(w, r, commands.Unstar{UserID: user.ID, Slug: slug})
		}
	}
}

// cmdTheme and cmdThemeStyle set the Themes page preview, a session
// preference: it carries across pages and browser tabs.
func (s *Server) cmdTheme(w http.ResponseWriter, r *http.Request) {
	s.send(w, r, commands.SetPreviewTheme{SID: edge.SessionID(r.Context()), Theme: r.PathValue("theme")})
}

func (s *Server) cmdThemeStyle(w http.ResponseWriter, r *http.Request) {
	s.send(w, r, commands.SetPreviewStyle{SID: edge.SessionID(r.Context()), Smooth: r.PathValue("style") == "smooth"})
}

// cmdInstallTab remembers the component pages' installation tab for the
// session (a session preference: it carries across pages and browser tabs).
func (s *Server) cmdInstallTab(w http.ResponseWriter, r *http.Request) {
	var p struct {
		Tab string `json:"tab"`
	}
	if err := datastar.ReadSignals(r, &p); err != nil {
		http.Error(w, "bad payload", http.StatusBadRequest)
		return
	}
	s.send(w, r, commands.SetInstallTab{SID: edge.SessionID(r.Context()), Tab: p.Tab})
}

// cmdFlight is the Showcase's commands demo: one field per request, from a
// component's sb-change ({name, value}). A short pause keeps the controls'
// pending state visible.
func (s *Server) cmdFlight(w http.ResponseWriter, r *http.Request) {
	var p struct {
		TabID string `json:"tabid"`
		Name  string `json:"name"`
		Value any    `json:"value"`
	}
	if err := datastar.ReadSignals(r, &p); err != nil {
		http.Error(w, "bad payload", http.StatusBadRequest)
		return
	}
	select {
	case <-time.After(400 * time.Millisecond):
	case <-r.Context().Done():
		return
	}
	s.send(w, r, commands.SetFlight{SID: edge.SessionID(r.Context()), TabID: p.TabID, Name: p.Name, Value: fmt.Sprint(p.Value)})
}
