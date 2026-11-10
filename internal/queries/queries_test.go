package queries_test

import (
	"context"
	"io"
	"log/slog"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/zweiundeins/lith/writer"

	"starbase/internal/catalog"
	"starbase/internal/commands"
	"starbase/internal/db"
	"starbase/internal/model"
	"starbase/internal/queries"
)

func component(slug, tag, category, since, summary string) fstest.MapFS {
	readme := "---\nname: " + slug + "\ntag: " + tag + "\ncategory: " + category +
		"\nsummary: " + summary + "\nauthor: octocat\ntags: [space]\nsince: " + since +
		"\npreview: <" + tag + "></" + tag + ">\n---\nDocs.\n"
	return fstest.MapFS{
		slug + "/README.md":       {Data: []byte(readme)},
		slug + "/" + slug + ".js": {Data: []byte("rocket('" + tag + "', {})")},
	}
}

type env struct {
	w *writer.Writer
	q *queries.Queries
}

func setup(t *testing.T) env {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	d, err := db.Open(ctx, filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { d.Close() })
	w, err := writer.New(d, writer.Options{Log: slog.New(slog.NewTextHandler(io.Discard, nil))})
	if err != nil {
		t.Fatal(err)
	}
	go w.Run(ctx)
	t.Cleanup(func() { w.Close() })

	fsys := fstest.MapFS{}
	for _, c := range []fstest.MapFS{
		component("alpha", "sb-alpha", "forms", "2026-01-01", "Launch forms quickly"),
		component("bravo", "sb-bravo", "forms", "2026-03-01", "Orbit selector"),
		component("charlie", "sb-charlie", "feedback", "2026-02-01", "Mission alerts"),
	} {
		for k, v := range c {
			fsys[k] = v
		}
	}
	cat, err := catalog.Load(fsys)
	if err != nil {
		t.Fatal(err)
	}
	if err := w.Do(ctx, commands.SyncCatalog{Catalog: cat}); err != nil {
		t.Fatal(err)
	}
	return env{w: w, q: queries.New(d)}
}

func (e env) signIn(t *testing.T, sid, login string, ghID int64) *model.User {
	t.Helper()
	ctx := context.Background()
	if err := e.w.Do(ctx, commands.SignIn{SID: sid, User: model.User{GitHubID: ghID, Login: login}}); err != nil {
		t.Fatal(err)
	}
	u, _ := e.q.ViewValue(ctx, func(r *queries.Reader) (*model.User, error) { return r.SessionUser(ctx, sid) })
	if u == nil || u.Login != login {
		t.Fatalf("session user = %+v", u)
	}
	return u
}

func (e env) browse(t *testing.T, b model.Browse, uid int64) queries.BrowseResult {
	t.Helper()
	res, err := e.q.ViewValue(context.Background(), func(r *queries.Reader) (queries.BrowseResult, error) {
		return r.Browse(context.Background(), b.Normalize(), uid)
	})
	if err != nil {
		t.Fatal(err)
	}
	return res
}

func slugs(cards []queries.Card) []string {
	var s []string
	for _, c := range cards {
		s = append(s, c.Slug)
	}
	return s
}

func TestStarsAreIdempotentAndCounted(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	ada := e.signIn(t, "sid-ada", "ada", 1)
	bob := e.signIn(t, "sid-bob", "bob", 2)

	for range 3 { // double clicks must not double count
		if err := e.w.Do(ctx, commands.Star{UserID: ada.ID, Slug: "charlie"}); err != nil {
			t.Fatal(err)
		}
	}
	e.w.Do(ctx, commands.Star{UserID: bob.ID, Slug: "charlie"})
	e.w.Do(ctx, commands.Star{UserID: bob.ID, Slug: "alpha"})

	res := e.browse(t, model.Browse{Sort: model.SortPopular}, ada.ID)
	if got := slugs(res.Cards); got[0] != "charlie" || got[1] != "alpha" {
		t.Fatalf("popular order = %v", got)
	}
	if res.Cards[0].Stars != 2 || !res.Cards[0].Starred || res.Cards[1].Starred {
		t.Fatalf("cards = %+v", res.Cards[:2])
	}

	for range 2 {
		e.w.Do(ctx, commands.Unstar{UserID: ada.ID, Slug: "charlie"})
	}
	res = e.browse(t, model.Browse{}, ada.ID)
	for _, c := range res.Cards {
		if c.Slug == "charlie" && (c.Stars != 1 || c.Starred) {
			t.Fatalf("after unstar: %+v", c)
		}
	}

	if err := e.w.Do(ctx, commands.Star{Slug: "alpha"}); err == nil {
		t.Fatal("anonymous star must be rejected")
	}
	if err := e.w.Do(ctx, commands.Star{UserID: ada.ID, Slug: "nope"}); err != nil {
		t.Fatalf("starring a missing component should be a no-op, got %v", err)
	}
}

func TestBrowseFilterSortSearch(t *testing.T) {
	e := setup(t)
	res := e.browse(t, model.Browse{Category: "forms", Sort: model.SortNewest}, 0)
	if got := slugs(res.Cards); len(got) != 2 || got[0] != "bravo" {
		t.Fatalf("forms newest = %v", got)
	}
	if res.Total != 3 || res.Counts["forms"] != 2 || res.Counts["feedback"] != 1 {
		t.Fatalf("counts = %v total %d", res.Counts, res.Total)
	}

	res = e.browse(t, model.Browse{Q: "orb"}, 0) // prefix match on summary
	if got := slugs(res.Cards); len(got) != 1 || got[0] != "bravo" {
		t.Fatalf("search orb = %v", got)
	}
	if res.Total != 1 || res.Counts["forms"] != 1 {
		t.Fatalf("counts follow the search: %v", res.Counts)
	}

	res = e.browse(t, model.Browse{Q: `"mission*^`}, 0) // FTS syntax is neutralised
	if got := slugs(res.Cards); len(got) != 1 || got[0] != "charlie" {
		t.Fatalf("hostile query = %v", got)
	}

	res = e.browse(t, model.Browse{Q: "feedback"}, 0) // category is indexed too
	if got := slugs(res.Cards); len(got) != 1 || got[0] != "charlie" {
		t.Fatalf("search by category = %v", got)
	}
}

func TestTabState(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	cmd := commands.SetBrowseFilter{SID: "s", TabID: "tab12345", Browse: model.Browse{Q: "  hi  ", Category: "bogus", Sort: "weird"}}
	if err := e.w.Do(ctx, cmd); err != nil {
		t.Fatal(err)
	}
	var st model.TabState
	var ok bool
	e.q.View(ctx, func(r *queries.Reader) (err error) { st, ok, err = r.Tab(ctx, "s", "tab12345"); return })
	if want := (model.TabState{Browse: model.Browse{Q: "hi", Sort: model.SortPopular}}); !ok || !reflect.DeepEqual(st, want) {
		t.Fatalf("tab = %+v, want %+v", st, want)
	}
	if err := e.w.Do(ctx, commands.SetBrowseFilter{SID: "s", TabID: "X"}); err == nil {
		t.Fatal("bad tab id must be rejected")
	}
}

// The sweep forgets tabs untouched for TabTTL and keeps the session's
// preferences.
func TestPruneTabs(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	for _, cmd := range []writer.Command{
		commands.SetGallerySort{SID: "s", TabID: "tab12345", Browse: model.Browse{Sort: model.SortNewest}},
		commands.SetBrowseFilter{SID: "s", TabID: "tab67890", Browse: model.Browse{Q: "hi"}},
		writer.Func(func(tx *writer.Tx) error {
			_, err := tx.Exec(`UPDATE tab_state SET updated_at = ? WHERE tab_id = 'tab12345'`, time.Now().Add(-commands.TabTTL-time.Minute).Unix())
			return err
		}),
		commands.PruneTabs{OlderThan: commands.TabTTL},
	} {
		if err := e.w.Do(ctx, cmd); err != nil {
			t.Fatalf("%T: %v", cmd, err)
		}
	}
	var old, fresh bool
	var prefs model.SessionPrefs
	e.q.View(ctx, func(r *queries.Reader) (err error) {
		_, old, _ = r.Tab(ctx, "s", "tab12345")
		_, fresh, _ = r.Tab(ctx, "s", "tab67890")
		prefs, err = r.Prefs(ctx, "s")
		return
	})
	if old || !fresh {
		t.Errorf("after the sweep: stale tab kept %v, fresh tab kept %v", old, fresh)
	}
	if prefs.GallerySort != model.SortNewest {
		t.Errorf("the sweep dropped the session's preferences: %+v", prefs)
	}
}

// Session preferences carry across the session's pages; another session
// has none.
func TestSessionPrefs(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	for _, cmd := range []writer.Command{
		commands.SetPreviewTheme{SID: "s", Theme: "nebula"},
		commands.SetPreviewStyle{SID: "s", Smooth: true},
		commands.SetInstallTab{SID: "s", Tab: "pinned"},
		commands.SetGallerySort{SID: "s", TabID: "tab12345", Browse: model.Browse{Q: "hi", Sort: model.SortNewest}},
	} {
		if err := e.w.Do(ctx, cmd); err != nil {
			t.Fatalf("%T: %v", cmd, err)
		}
	}
	prefs := func(sid string) model.SessionPrefs {
		p, _ := e.q.ViewValue(ctx, func(r *queries.Reader) (model.SessionPrefs, error) { return r.Prefs(ctx, sid) })
		return p
	}
	want := model.SessionPrefs{InstallTab: "pinned", PreviewTheme: "nebula", PreviewSmooth: true, GallerySort: model.SortNewest}
	if got := prefs("s"); got != want {
		t.Fatalf("prefs = %+v, want %+v", got, want)
	}
	// The sort is also the tab's (the page shows what was chosen).
	var st model.TabState
	e.q.View(ctx, func(r *queries.Reader) (err error) { st, _, err = r.Tab(ctx, "s", "tab12345"); return })
	if st.Browse != (model.Browse{Q: "hi", Sort: model.SortNewest}) {
		t.Errorf("tab browse = %+v", st.Browse)
	}
	// A search or category change leaves the default sort alone.
	e.w.Do(ctx, commands.SetBrowseFilter{SID: "s", TabID: "tab12345", Browse: model.Browse{Sort: model.SortName}})
	if got := prefs("s").GallerySort; got != model.SortNewest {
		t.Errorf("a browse filter changed the default sort to %q", got)
	}
	if got := prefs("other"); got != (model.SessionPrefs{}) || got.PreviewThemeOrDefault() != "deep-space" || got.DefaultSort() != model.SortPopular {
		t.Errorf("another session: %+v", got)
	}
	for _, bad := range []writer.Command{
		commands.SetPreviewTheme{SID: "s", Theme: "hacker"},
		commands.SetPreviewTheme{SID: "", Theme: "nebula"},
		commands.SetPreviewStyle{SID: ""},
		commands.SetGallerySort{SID: "s", TabID: "tab12345", Browse: model.Browse{Sort: "weird"}},
		commands.SetGallerySort{SID: "s", TabID: "X", Browse: model.Browse{Sort: model.SortName}},
	} {
		if err := e.w.Do(ctx, bad); err == nil {
			t.Errorf("%+v must be rejected", bad)
		}
	}
}

func TestSyncDeactivatesRemovedComponents(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	cat, _ := catalog.Load(component("alpha", "sb-alpha", "forms", "2026-01-01", "Launch forms quickly"))
	if err := e.w.Do(ctx, commands.SyncCatalog{Catalog: cat}); err != nil {
		t.Fatal(err)
	}
	if got := slugs(e.browse(t, model.Browse{}, 0).Cards); len(got) != 1 || got[0] != "alpha" {
		t.Fatalf("after sync = %v", got)
	}
	if got := slugs(e.browse(t, model.Browse{Q: "orbit"}, 0).Cards); len(got) != 0 {
		t.Fatalf("removed component still searchable: %v", got)
	}
}

func TestFTSQuery(t *testing.T) {
	for in, want := range map[string]string{
		"Toggle":           `"toggle"*`,
		"  modal  dialog ": `"modal"* "dialog"*`,
		`a"b OR c*`:        `"a"* "b"* "or"* "c"*`,
		"---":              "",
	} {
		if got := queries.FTSQuery(in); got != want {
			t.Errorf("FTSQuery(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestSnippetSaveAndLoad(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	u := e.signIn(t, "sid-snip", "coder", 9)
	cmd := commands.SaveSnippet{SID: "sid-snip", TabID: "tab12345", ID: "AbCd2345", Files: map[string]string{"component.js": "rocket('sb-x', {})"}, Component: "alpha", UserID: u.ID}
	if err := e.w.Do(ctx, cmd); err != nil {
		t.Fatal(err)
	}
	if err := e.w.Do(ctx, cmd); err == nil {
		t.Fatal("ids are unique; saving twice must fail")
	}
	var sn *queries.Snippet
	var st model.TabState
	e.q.View(ctx, func(r *queries.Reader) (err error) {
		if sn, err = r.Snippet(ctx, "AbCd2345"); err != nil {
			return err
		}
		st, _, err = r.Tab(ctx, "sid-snip", "tab12345")
		return err
	})
	if sn == nil || sn.Files["component.js"] != "rocket('sb-x', {})" || sn.Component != "alpha" || sn.Author != "coder" {
		t.Fatalf("snippet = %+v", sn)
	}
	if st.PlaygroundShare != "AbCd2345" {
		t.Fatalf("tab share = %q", st.PlaygroundShare)
	}
}

func TestBoard(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	if err := e.w.Do(ctx, commands.SeedBoard{}); err != nil {
		t.Fatal(err)
	}
	read := func() (cells string, version, pixels int64) {
		e.q.View(ctx, func(r *queries.Reader) (err error) {
			if version, pixels, err = r.BoardMeta(ctx, commands.BoardName); err != nil {
				return err
			}
			cells, err = r.BoardCells(ctx, commands.BoardName, commands.BoardSize)
			return err
		})
		return
	}
	seeded, v1, _ := read()
	if len(seeded) != commands.BoardSize*commands.BoardSize || strings.Count(seeded, "0") == len(seeded) {
		t.Fatalf("seeded board has %d cells, all empty: %v", len(seeded), strings.Count(seeded, "0") == len(seeded))
	}
	// Seeding twice is a no-op.
	e.w.Do(ctx, commands.SeedBoard{})
	if again, v, _ := read(); again != seeded || v != v1 {
		t.Fatal("second seed changed the board")
	}
	if err := e.w.Do(ctx, commands.PaintPixels{Board: "main", Color: 13, Cells: []int{0, 1, 47}}); err != nil {
		t.Fatal(err)
	}
	cells, v2, pixels := read()
	if cells[0] != 'd' || cells[1] != 'd' || cells[47] != 'd' || v2 != v1+1 || pixels != 3 {
		t.Fatalf("after paint: cells[0..1]=%q cells[47]=%q v=%d pixels=%d", cells[:2], cells[47], v2, pixels)
	}
	for name, bad := range map[string]commands.PaintPixels{
		"colour":   {Board: "main", Color: 16, Cells: []int{0}},
		"range":    {Board: "main", Color: 1, Cells: []int{48 * 48}},
		"too many": {Board: "main", Color: 1, Cells: make([]int, 65)},
		"board":    {Board: "other", Color: 1, Cells: []int{0}},
		"empty":    {Board: "main", Color: 1},
	} {
		if bad.Validate() == nil {
			t.Errorf("%s: expected a validation error", name)
		}
	}
}

func TestSnippetStoreCap(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	save := func(id string, size int) error {
		return e.w.Do(ctx, commands.SaveSnippet{SID: "s", TabID: "tab12345", ID: id, Files: map[string]string{"component.js": strings.Repeat("x", size)}})
	}
	if err := save("AAAAAAAA", 100); err != nil {
		t.Fatal(err)
	}
	n, _ := e.q.ViewValue(ctx, func(r *queries.Reader) (int64, error) { return r.SnippetBytes(ctx) })
	if n < 100 || n > 200 {
		t.Fatalf("stored bytes = %d, want the JSON size of one ~100 byte snippet", n)
	}
	defer func(old int64) { commands.MaxSnippetStore = old }(commands.MaxSnippetStore)
	commands.MaxSnippetStore = 2*n + 10           // room for exactly one more of the same size
	if err := save("BBBBBBBB", 100); err != nil { // fits
		t.Fatal(err)
	}
	if err := save("CCCCCCCC", 100); err != nil {
		t.Fatal(err)
	}
	sn, _ := e.q.ViewValue(ctx, func(r *queries.Reader) (*queries.Snippet, error) { return r.Snippet(ctx, "CCCCCCCC") })
	if sn != nil {
		t.Fatal("the refused snippet must not be stored")
	}
	var st model.TabState
	e.q.View(ctx, func(r *queries.Reader) (err error) { st, _, err = r.Tab(ctx, "s", "tab12345"); return })
	if st.ShareRefusal != commands.RefusedStoreFull || st.PlaygroundShare != "BBBBBBBB" {
		t.Fatalf("tab = %+v, want the full store's refusal after the second save", st)
	}
	// Code over MaxSnippetBytes is refused in the tab's row too.
	commands.MaxSnippetStore = 1 << 30
	if err := save("DDDDDDDD", commands.MaxSnippetBytes+1); err != nil {
		t.Fatal(err)
	}
	e.q.View(ctx, func(r *queries.Reader) (err error) { st, _, err = r.Tab(ctx, "s", "tab12345"); return })
	if st.ShareRefusal != commands.RefusedSnippetSize {
		t.Fatalf("tab = %+v, want the size refusal", st)
	}
	if err := save("EEEEEEEE", 10); err != nil {
		t.Fatal(err)
	}
	e.q.View(ctx, func(r *queries.Reader) (err error) { st, _, err = r.Tab(ctx, "s", "tab12345"); return })
	if st.ShareRefusal != "" || st.PlaygroundShare != "EEEEEEEE" {
		t.Fatalf("tab = %+v, want the new share and no refusal", st)
	}
}

func TestDemoData(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	for range 2 { // idempotent
		if err := e.w.Do(ctx, commands.SeedDemo{}); err != nil {
			t.Fatal(err)
		}
	}
	var top, moons, found, stars []queries.DemoBody
	e.q.View(ctx, func(r *queries.Reader) (err error) {
		top, _ = r.DemoChildren(ctx, "")
		moons, _ = r.DemoChildren(ctx, "jupiter")
		found, _ = r.DemoSearch(ctx, "bootes", nil, 10)
		stars, err = r.DemoSearch(ctx, "an", []string{"star"}, 50)
		return
	})
	if len(top) != 5 || top[0].ID != "milkyway" || top[0].Children == 0 {
		t.Errorf("top level = %+v", top)
	}
	if len(moons) != 4 || moons[0].Name != "Io" || moons[0].Children != 0 {
		t.Errorf("jupiter = %+v", moons)
	}
	if len(found) == 0 { // accents fold: Boötes
		t.Error("no match for bootes")
	}
	for i, s := range stars {
		if s.Kind != "star" {
			t.Errorf("kind filter let %s through", s.Kind)
		}
		prefix := strings.HasPrefix(strings.ToLower(s.Name), "an")
		if i > 0 && prefix && !strings.HasPrefix(strings.ToLower(stars[i-1].Name), "an") {
			t.Errorf("prefix matches must come first: %v", stars)
		}
	}
}

func TestSetFlight(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	send := func(name, value string) error {
		return e.w.Do(ctx, commands.SetFlight{SID: "s", TabID: "tab12345", Name: name, Value: value})
	}
	for _, c := range [][2]string{{"thrust", "70"}, {"shields", "false"}, {"callsign", " apollo-11 "}} {
		if err := send(c[0], c[1]); err != nil {
			t.Fatalf("%s=%s: %v", c[0], c[1], err)
		}
	}
	// What no control sends is invalid.
	for _, bad := range [][2]string{{"thrust", "abc"}, {"thrust", "101"}, {"shields", "maybe"}, {"warp", "9"}} {
		if err := send(bad[0], bad[1]); err == nil {
			t.Errorf("%s=%q must be rejected", bad[0], bad[1])
		}
	}
	flight := func() model.FlightPlan {
		var st model.TabState
		e.q.View(ctx, func(r *queries.Reader) (err error) { st, _, err = r.Tab(ctx, "s", "tab12345"); return })
		return st.Flight
	}
	want := model.FlightPlan{Set: true, Thrust: 70, Shields: false, Callsign: "APOLLO-11"}
	if got := flight(); got != want {
		t.Errorf("flight = %+v, want %+v (normalized)", got, want)
	}
	// What a person can choose but the rules forbid is a refusal in the
	// tab's row: the value stays, and the row acks the op under its key.
	for _, c := range []struct {
		name, value, op, refusal string
	}{{"thrust", "95", "op1", commands.RefusedThrust}, {"callsign", "no spaces", "op2", commands.RefusedCallsign}} {
		if err := e.w.Do(ctx, commands.SetFlight{SID: "s", TabID: "tab12345", Op: c.op, Name: c.name, Value: c.value}); err != nil {
			t.Fatalf("%s=%q: %v", c.name, c.value, err)
		}
		if got := flight().Refusal; got != c.refusal {
			t.Errorf("%s=%q: refusal %q, want %q", c.name, c.value, got, c.refusal)
		}
	}
	var st model.TabState
	e.q.View(ctx, func(r *queries.Reader) (err error) { st, _, err = r.Tab(ctx, "s", "tab12345"); return })
	if st.Ops["flight:thrust"] != "op1" || st.Ops["flight:callsign"] != "op2" {
		t.Errorf("acks = %v", st.Ops)
	}
	if err := e.w.Do(ctx, commands.SetFlight{SID: "s", TabID: "tab12345", Op: "Not An Op", Name: "shields", Value: "true"}); err == nil {
		t.Error("a malformed op id must be rejected")
	}
	want.Refusal = commands.RefusedCallsign
	if got := flight(); got != want {
		t.Errorf("flight = %+v, want %+v (the refused values not applied)", got, want)
	}
	if err := send("shields", "true"); err != nil {
		t.Fatal(err)
	}
	if got := flight(); got.Refusal != "" || !got.Shields {
		t.Errorf("an applied command keeps the refusal: %+v", got)
	}
	if (model.FlightPlan{}).OrDefault().Thrust != 40 {
		t.Error("defaults")
	}
}
