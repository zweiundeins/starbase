package web

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/starfederation/datastar-go/datastar"

	"starbase/internal/commands"
	"starbase/internal/cqrs"
	"starbase/internal/model"
	"starbase/internal/queries"
)

// send enqueues a command and answers 204. The tab's render stream shows
// the outcome; commands never return HTML.
func (s *Server) send(w http.ResponseWriter, r *http.Request, cmd cqrs.Command) {
	err := s.bus.Send(cmd)
	switch {
	case err == nil:
		w.WriteHeader(http.StatusNoContent)
	case errors.Is(err, cqrs.ErrInvalid):
		http.Error(w, err.Error(), http.StatusBadRequest)
	default:
		s.log.Warn("command rejected", "err", err)
		http.Error(w, "busy, try again", http.StatusServiceUnavailable)
	}
}

func (s *Server) cmdBrowse(w http.ResponseWriter, r *http.Request) {
	var sig struct {
		TabID string `json:"tabid"`
		Q     string `json:"q"`
		Cat   string `json:"cat"`
		Sort  string `json:"sort"`
	}
	if err := datastar.ReadSignals(r, &sig); err != nil {
		http.Error(w, "bad signals", http.StatusBadRequest)
		return
	}
	s.send(w, r, commands.SetBrowseFilter{
		SID:    sessionID(r),
		TabID:  sig.TabID,
		Browse: model.Browse{Q: sig.Q, Category: sig.Cat, Sort: model.Sort(sig.Sort)},
	})
}

func (s *Server) cmdStar(star bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var user *model.User
		err := s.q.View(r.Context(), func(rd *queries.Reader) error {
			var err error
			user, err = rd.SessionUser(r.Context(), sessionID(r))
			return err
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

func (s *Server) cmdTheme(w http.ResponseWriter, r *http.Request) {
	var sig struct {
		TabID string `json:"tabid"`
	}
	if err := datastar.ReadSignals(r, &sig); err != nil {
		http.Error(w, "bad signals", http.StatusBadRequest)
		return
	}
	s.send(w, r, commands.SetPreviewTheme{SID: sessionID(r), TabID: sig.TabID, Theme: r.PathValue("theme")})
}

func (s *Server) cmdThemeStyle(w http.ResponseWriter, r *http.Request) {
	var sig struct {
		TabID string `json:"tabid"`
	}
	if err := datastar.ReadSignals(r, &sig); err != nil {
		http.Error(w, "bad signals", http.StatusBadRequest)
		return
	}
	s.send(w, r, commands.SetPreviewStyle{SID: sessionID(r), TabID: sig.TabID, Smooth: r.PathValue("style") == "smooth"})
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
	s.send(w, r, commands.SetInstallTab{SID: sessionID(r), Tab: p.Tab})
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
	s.send(w, r, commands.SetFlight{SID: sessionID(r), TabID: p.TabID, Name: p.Name, Value: fmt.Sprint(p.Value)})
}
