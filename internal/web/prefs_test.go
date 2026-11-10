package web_test

import (
	"context"
	"io"
	"net/http"
	"net/http/cookiejar"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/zweiundeins/lith/writer"

	"starbase/internal/commands"
	"starbase/internal/config"
	"starbase/internal/model"
)

// eventually retries check until it holds: commands are applied after the
// handler's 204.
func eventually(t *testing.T, what string, check func() bool) {
	t.Helper()
	for deadline := time.Now().Add(3 * time.Second); !check(); time.Sleep(20 * time.Millisecond) {
		if time.Now().After(deadline) {
			t.Fatalf("never: %s", what)
		}
	}
}

func newSession() *http.Client {
	jar, _ := cookiejar.New(nil)
	return &http.Client{Jar: jar}
}

// The Themes page preview is a session preference: every page of the
// session renders it from the first GET; another session sees the default.
func TestThemePreviewRemembered(t *testing.T) {
	ts, c := newServer(t)
	themes := func(c *http.Client) string { _, page := get(t, c, ts.URL+"/themes"); return page }
	if p := themes(c); !strings.Contains(p, `data-sb-theme="deep-space" data-sb-style="pixel"`) || !strings.Contains(p, `label="8-bit details" checked="true"`) {
		t.Fatal("default preview should be Deep Space with 8-bit details")
	}
	for path, want := range map[string]int{"/cmd/theme/daylight": 204, "/cmd/theme-style/smooth": 204, "/cmd/theme/hacker": 400} {
		if r := post(t, c, ts.URL+path, `{}`, "same-origin"); r.StatusCode != want {
			t.Errorf("%s = %d, want %d", path, r.StatusCode, want)
		}
	}
	eventually(t, "the stored preview renders", func() bool {
		p := themes(c)
		return strings.Contains(p, `data-sb-theme="daylight" data-sb-style="smooth"`) && strings.Contains(p, `label="8-bit details" checked="false"`)
	})
	pressed := regexp.MustCompile(`aria-pressed="true"[^>]*>\s*<span class="swatches"[^<]*(?:<span[^>]*></span>\s*)*</span>\s*<strong>([^<]+)</strong>`)
	if m := pressed.FindStringSubmatch(themes(c)); m == nil || m[1] != "Daylight" {
		t.Errorf("pressed theme = %v, want Daylight", m)
	}
	if p := themes(newSession()); !strings.Contains(p, `data-sb-theme="deep-space" data-sb-style="pixel"`) {
		t.Error("another session sees this session's preview")
	}
}

var selectedSortRe = regexp.MustCompile(`<option value="(\w+)" selected>`)

// The gallery's sort: a sort in the URL wins; without one, the session's
// last chosen sort applies. Search and category keep the sort.
func TestGallerySortDefault(t *testing.T) {
	ts, c := newServer(t)
	gallery := func(c *http.Client, path string) (sort, page string) {
		t.Helper()
		_, page = get(t, c, ts.URL+path)
		m := selectedSortRe.FindStringSubmatch(page)
		if m == nil {
			t.Fatalf("%s: no selected sort", path)
		}
		return m[1], page
	}
	if s, _ := gallery(c, "/"); s != "popular" {
		t.Fatalf("default sort = %s", s)
	}
	for _, tab := range []string{"tab12345", "tab99999"} {
		next, _ := openStream(t, c, ts.URL+"/", tab)
		until(t, next, `id="app"`)
	}
	for body, want := range map[string]int{`{"tabid":"tab12345","sort":"weird"}`: 400, `{"tabid":"tab12345","sort":"newest"}`: 204} {
		if r := post(t, c, ts.URL+"/cmd/sort", body, "same-origin"); r.StatusCode != want {
			t.Errorf("sort %s = %d, want %d", body, r.StatusCode, want)
		}
	}
	// The first GET of / renders the stored sort, and writes it into the URL.
	eventually(t, "/ uses the stored sort", func() bool { s, _ := gallery(c, "/"); return s == "newest" })
	if _, p := gallery(c, "/"); !strings.Contains(p, `replaceState(null, &#39;&#39;, &#34;/?sort=newest&#34;)`) {
		t.Error("the URL should carry the sort shown")
	}
	// A shared link's sort wins.
	if s, _ := gallery(c, "/?sort=name"); s != "name" {
		t.Errorf("/?sort=name shows %s", s)
	}
	// Category and search keep the default sort, and their links carry it.
	s, p := gallery(c, "/?cat=forms")
	if s != "newest" || !strings.Contains(p, `href="/?cat=navigation&amp;sort=newest#components"`) {
		t.Errorf("/?cat=forms: sort %s, or category links lose it", s)
	}
	if s, _ := gallery(c, "/?q=but"); s != "newest" {
		t.Errorf("/?q=but shows %s", s)
	}
	// A search or category change (with another sort in its URL) doesn't
	// change the default; only a chosen sort does.
	if r := post(t, c, ts.URL+"/cmd/browse", `{"tabid":"tab99999","q":"x","sort":"name"}`, "same-origin"); r.StatusCode != 204 {
		t.Fatalf("browse = %d", r.StatusCode)
	}
	time.Sleep(100 * time.Millisecond)
	if s, _ := gallery(c, "/"); s != "newest" {
		t.Errorf("a search changed the default sort to %s", s)
	}
	// "Most popular" in a URL wins over a stored Newest and stays in the URL,
	// so a reload shows the same list.
	if s, p := gallery(c, "/?sort=popular"); s != "popular" || !strings.Contains(p, `&#34;/?sort=popular&#34;`) {
		t.Errorf("/?sort=popular: sort %s, or the URL dropped it", s)
	}
	// Another session has the plain default, and its URLs stay short.
	if s, p := gallery(newSession(), "/"); s != "popular" || strings.Contains(p, "replaceState(null, &#39;&#39;, &#34;/?sort") {
		t.Errorf("another session: sort %s", s)
	}
}

// A session id from before edge.Sessions (160 bits, 27 characters) stays
// valid: its browser stays signed in and keeps its preferences. A browser
// without a valid id gets a new 256-bit one.
func TestOldSessionIDsStayValid(t *testing.T) {
	ts, _, wr, _ := newServerWriter(t)
	old := "c3RhcmJhc2Ugc2Vzc2lvbiAxNjA" // 20 bytes in base64url, as starbase made them
	for _, cmd := range []writer.Command{
		commands.SignIn{SID: old, User: model.User{GitHubID: 42, Login: "ada"}},
		commands.SetPreviewTheme{SID: old, Theme: "daylight"},
	} {
		if err := wr.Do(context.Background(), cmd); err != nil {
			t.Fatal(err)
		}
	}
	themes := func(cookie string) (setCookie []*http.Cookie, page string) {
		t.Helper()
		req, _ := http.NewRequest("GET", ts.URL+"/themes", nil)
		if cookie != "" {
			req.AddCookie(&http.Cookie{Name: "sid", Value: cookie})
		}
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		b, _ := io.ReadAll(res.Body)
		return res.Cookies(), string(b)
	}
	set, page := themes(old)
	if len(set) != 0 {
		t.Errorf("the old id was replaced: %v", set)
	}
	if !strings.Contains(page, "Signed in as <strong>ada</strong>") {
		t.Error("the old id's browser was signed out")
	}
	if !strings.Contains(page, `data-sb-theme="daylight"`) {
		t.Error("the old id's preferences were lost")
	}
	for _, cookie := range []string{"", old[:26], old + "A"} {
		if set, _ := themes(cookie); len(set) != 1 || set[0].Name != "sid" || len(set[0].Value) != 43 {
			t.Errorf("cookie %q: got %v, want a new 256-bit id", cookie, set)
		}
	}
}

// A tab whose stream is open keeps its row through the sweep; a closed
// tab's row goes.
func TestOpenTabsOutliveTheSweep(t *testing.T) {
	s := startServerKeeping(t, config.Config{}, 50*time.Millisecond)
	get(t, s.c, s.ts.URL+"/showcase")
	open, _ := openStream(t, s.c, s.ts.URL+"/showcase", "keepa000")
	until(t, open, `id="app"`)
	closed, stop := openStream(t, s.c, s.ts.URL+"/showcase", "keepb000")
	until(t, closed, `id="app"`)
	for _, tab := range []string{"keepa000", "keepb000"} {
		if r := post(t, s.c, s.ts.URL+"/cmd/flight", `{"tabid":"`+tab+`","name":"thrust","value":70}`, "same-origin"); r.StatusCode != http.StatusNoContent {
			t.Fatalf("%s: flight = %d", tab, r.StatusCode)
		}
	}
	ctx := context.Background()
	rows := func() (n int) {
		s.wr.Do(ctx, writer.Func(func(tx *writer.Tx) error {
			return tx.QueryRow(`SELECT count(*) FROM tab_state WHERE tab_id IN ('keepa000', 'keepb000')`).Scan(&n)
		}))
		return n
	}
	eventually(t, "both tabs have rows", func() bool { return rows() == 2 })
	stop()
	eventually(t, "the closed tab's stream ends", func() bool { return s.hub.Stats().Streams == 1 })
	old := time.Now().Add(-2 * time.Hour).Unix()
	if err := s.wr.Do(ctx, writer.Func(func(tx *writer.Tx) error {
		_, err := tx.Exec(`UPDATE tab_state SET updated_at = ?`, old)
		return err
	})); err != nil {
		t.Fatal(err)
	}
	eventually(t, "the open tab's row is touched", func() bool {
		var at int64
		s.wr.Do(ctx, writer.Func(func(tx *writer.Tx) error {
			return tx.QueryRow(`SELECT updated_at FROM tab_state WHERE tab_id = 'keepa000'`).Scan(&at)
		}))
		return at > old
	})
	// Keeping a tab renders nothing.
	stable(t, s.hub)
	renders := s.hub.Stats().Renders
	time.Sleep(300 * time.Millisecond)
	if n := s.hub.Stats().Renders - renders; n != 0 {
		t.Errorf("keeping the open tab rendered %d views", n)
	}
	if err := s.wr.Do(ctx, commands.PruneTabs{OlderThan: time.Hour}); err != nil {
		t.Fatal(err)
	}
	var kept []string
	s.wr.Do(ctx, writer.Func(func(tx *writer.Tx) error {
		r, err := tx.Query(`SELECT tab_id FROM tab_state WHERE tab_id IN ('keepa000', 'keepb000')`)
		if err != nil {
			return err
		}
		defer r.Close()
		for r.Next() {
			var id string
			r.Scan(&id)
			kept = append(kept, id)
		}
		return r.Err()
	}))
	if len(kept) != 1 || kept[0] != "keepa000" {
		t.Errorf("after the sweep: %v, want the open tab's row alone", kept)
	}
}
