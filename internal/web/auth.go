package web

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"

	"starbase/internal/commands"
	"starbase/internal/config"
	"starbase/internal/model"
)

type oauthConfig struct{ *oauth2.Config }

func newOAuth(cfg config.Config) *oauthConfig {
	if !cfg.GitHubEnabled() {
		return nil
	}
	return &oauthConfig{&oauth2.Config{
		ClientID:     cfg.GitHubClientID,
		ClientSecret: cfg.GitHubClientSecret,
		Endpoint:     github.Endpoint,
		RedirectURL:  strings.TrimSuffix(cfg.BaseURL, "/") + "/auth/github/callback",
		Scopes:       nil, // public profile only
	}}
}

// localPath only allows same-site relative redirects.
func localPath(next string) string {
	if u, err := url.Parse(next); err == nil && u.IsAbs() {
		next = u.RequestURI()
	}
	if !strings.HasPrefix(next, "/") || strings.HasPrefix(next, "//") || strings.HasPrefix(next, "/\\") {
		return "/"
	}
	return next
}

const stateCookie = "oauth_state"

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	next := localPath(r.URL.Query().Get("next"))
	if s.oauth == nil {
		if s.cfg.Dev {
			http.Redirect(w, r, "/auth/dev?next="+url.QueryEscape(next), http.StatusSeeOther)
			return
		}
		http.Error(w, "GitHub sign-in is not configured", http.StatusServiceUnavailable)
		return
	}
	state := randomToken(18)
	http.SetCookie(w, &http.Cookie{
		Name:     stateCookie,
		Value:    state + "|" + url.QueryEscape(next),
		Path:     "/auth/",
		MaxAge:   600,
		HttpOnly: true,
		Secure:   s.secure,
		SameSite: http.SameSiteLaxMode,
	})
	http.Redirect(w, r, s.oauth.AuthCodeURL(state), http.StatusSeeOther)
}

func (s *Server) githubCallback(w http.ResponseWriter, r *http.Request) {
	if s.oauth == nil {
		http.NotFound(w, r)
		return
	}
	c, err := r.Cookie(stateCookie)
	if err != nil {
		http.Error(w, "sign-in expired, please try again", http.StatusBadRequest)
		return
	}
	http.SetCookie(w, &http.Cookie{Name: stateCookie, Path: "/auth/", MaxAge: -1})
	state, nextEsc, _ := strings.Cut(c.Value, "|")
	if state == "" || r.URL.Query().Get("state") != state {
		http.Error(w, "sign-in state mismatch, please try again", http.StatusBadRequest)
		return
	}
	next, _ := url.QueryUnescape(nextEsc)

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	tok, err := s.oauth.Exchange(ctx, r.URL.Query().Get("code"))
	if err != nil {
		s.log.Warn("oauth exchange", "err", err)
		http.Error(w, "GitHub sign-in failed", http.StatusBadGateway)
		return
	}
	u, err := fetchGitHubUser(ctx, s.oauth.Client(ctx, tok))
	if err != nil {
		s.log.Warn("github user", "err", err)
		http.Error(w, "could not read your GitHub profile", http.StatusBadGateway)
		return
	}
	s.signIn(w, r, u, next)
}

// signIn rotates the session id (preventing fixation) and binds it to u.
func (s *Server) signIn(w http.ResponseWriter, r *http.Request, u model.User, next string) {
	sid := randomToken(20)
	if err := s.bus.Exec(r.Context(), commands.SignIn{SID: sid, User: u}); err != nil {
		s.fail(w, r, err)
		return
	}
	s.setSessionCookie(w, sid)
	http.Redirect(w, r, localPath(next), http.StatusSeeOther)
}

func fetchGitHubUser(ctx context.Context, c *http.Client) (model.User, error) {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com/user", nil)
	req.Header.Set("Accept", "application/vnd.github+json")
	res, err := c.Do(req)
	if err != nil {
		return model.User{}, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return model.User{}, fmt.Errorf("github: %s", res.Status)
	}
	var gh struct {
		ID        int64  `json:"id"`
		Login     string `json:"login"`
		Name      string `json:"name"`
		AvatarURL string `json:"avatar_url"`
	}
	if err := json.NewDecoder(res.Body).Decode(&gh); err != nil {
		return model.User{}, err
	}
	return model.User{GitHubID: gh.ID, Login: gh.Login, Name: gh.Name, AvatarURL: gh.AvatarURL}, nil
}

func (s *Server) logout(w http.ResponseWriter, r *http.Request) {
	if err := s.bus.Exec(r.Context(), commands.SignOut{SID: sessionID(r)}); err != nil {
		s.fail(w, r, err)
		return
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}
