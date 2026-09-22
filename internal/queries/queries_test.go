package queries_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"starbase/internal/catalog"
	"starbase/internal/commands"
	"starbase/internal/cqrs"
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
	bus *cqrs.Bus
	q   *queries.Queries
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
	bus := cqrs.NewBus(d.W, cqrs.NewHub(), slog.New(slog.NewTextHandler(io.Discard, nil)))
	go bus.Run(ctx)

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
	if err := bus.Exec(ctx, commands.SyncCatalog{Catalog: cat}); err != nil {
		t.Fatal(err)
	}
	return env{bus: bus, q: queries.New(d.R)}
}

func (e env) signIn(t *testing.T, sid, login string, ghID int64) *model.User {
	t.Helper()
	ctx := context.Background()
	if err := e.bus.Exec(ctx, commands.SignIn{SID: sid, User: model.User{GitHubID: ghID, Login: login}}); err != nil {
		t.Fatal(err)
	}
	var u *model.User
	e.q.View(ctx, func(r *queries.Reader) (err error) { u, err = r.SessionUser(ctx, sid); return })
	if u == nil || u.Login != login {
		t.Fatalf("session user = %+v", u)
	}
	return u
}

func (e env) browse(t *testing.T, b model.Browse, uid int64) queries.BrowseResult {
	t.Helper()
	var res queries.BrowseResult
	err := e.q.View(context.Background(), func(r *queries.Reader) (err error) {
		res, err = r.Browse(context.Background(), b.Normalize(), uid)
		return
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
		if err := e.bus.Exec(ctx, commands.Star{UserID: ada.ID, Slug: "charlie"}); err != nil {
			t.Fatal(err)
		}
	}
	e.bus.Exec(ctx, commands.Star{UserID: bob.ID, Slug: "charlie"})
	e.bus.Exec(ctx, commands.Star{UserID: bob.ID, Slug: "alpha"})

	res := e.browse(t, model.Browse{Sort: model.SortPopular}, ada.ID)
	if got := slugs(res.Cards); got[0] != "charlie" || got[1] != "alpha" {
		t.Fatalf("popular order = %v", got)
	}
	if res.Cards[0].Stars != 2 || !res.Cards[0].Starred || res.Cards[1].Starred {
		t.Fatalf("cards = %+v", res.Cards[:2])
	}

	for range 2 {
		e.bus.Exec(ctx, commands.Unstar{UserID: ada.ID, Slug: "charlie"})
	}
	res = e.browse(t, model.Browse{}, ada.ID)
	for _, c := range res.Cards {
		if c.Slug == "charlie" && (c.Stars != 1 || c.Starred) {
			t.Fatalf("after unstar: %+v", c)
		}
	}

	if err := e.bus.Exec(ctx, commands.Star{Slug: "alpha"}); err == nil {
		t.Fatal("anonymous star must be rejected")
	}
	if err := e.bus.Exec(ctx, commands.Star{UserID: ada.ID, Slug: "nope"}); err != nil {
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
	if err := e.bus.Exec(ctx, cmd); err != nil {
		t.Fatal(err)
	}
	e.bus.Exec(ctx, commands.SetPreviewTheme{SID: "s", TabID: "tab12345", Theme: "nebula"})
	e.bus.Exec(ctx, commands.SetPreviewStyle{SID: "s", TabID: "tab12345", Smooth: true})
	var st model.TabState
	var ok bool
	e.q.View(ctx, func(r *queries.Reader) (err error) { st, ok, err = r.Tab(ctx, "s", "tab12345"); return })
	want := model.TabState{Browse: model.Browse{Q: "hi", Sort: model.SortPopular}, PreviewTheme: "nebula", PreviewSmooth: true}
	if !ok || st != want {
		t.Fatalf("tab = %+v, want %+v", st, want)
	}
	if err := e.bus.Exec(ctx, commands.SetPreviewTheme{SID: "s", TabID: "tab12345", Theme: "hacker"}); err == nil {
		t.Fatal("unknown theme must be rejected")
	}
	if err := e.bus.Exec(ctx, commands.SetBrowseFilter{SID: "s", TabID: "X"}); err == nil {
		t.Fatal("bad tab id must be rejected")
	}
}

func TestSyncDeactivatesRemovedComponents(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	cat, _ := catalog.Load(component("alpha", "sb-alpha", "forms", "2026-01-01", "Launch forms quickly"))
	if err := e.bus.Exec(ctx, commands.SyncCatalog{Catalog: cat}); err != nil {
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
	if err := e.bus.Exec(ctx, cmd); err != nil {
		t.Fatal(err)
	}
	if err := e.bus.Exec(ctx, cmd); err == nil {
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
	if err := e.bus.Exec(ctx, commands.SeedBoard{}); err != nil {
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
	e.bus.Exec(ctx, commands.SeedBoard{})
	if again, v, _ := read(); again != seeded || v != v1 {
		t.Fatal("second seed changed the board")
	}
	if err := e.bus.Exec(ctx, commands.PaintPixels{Board: "main", Color: 13, Cells: []int{0, 1, 47}}); err != nil {
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
		return e.bus.Exec(ctx, commands.SaveSnippet{SID: "s", TabID: "tab12345", ID: id, Files: map[string]string{"component.js": strings.Repeat("x", size)}})
	}
	if err := save("AAAAAAAA", 100); err != nil {
		t.Fatal(err)
	}
	var n int64
	e.q.View(ctx, func(r *queries.Reader) (err error) { n, err = r.SnippetBytes(ctx); return })
	if n < 100 || n > 200 {
		t.Fatalf("stored bytes = %d, want the JSON size of one ~100 byte snippet", n)
	}
	defer func(old int64) { commands.MaxSnippetStore = old }(commands.MaxSnippetStore)
	commands.MaxSnippetStore = 2*n + 10           // room for exactly one more of the same size
	if err := save("BBBBBBBB", 100); err != nil { // fits
		t.Fatal(err)
	}
	if err := save("CCCCCCCC", 100); !errors.Is(err, commands.ErrSnippetStoreFull) {
		t.Fatalf("err = %v, want ErrSnippetStoreFull", err)
	}
	var sn *queries.Snippet
	e.q.View(ctx, func(r *queries.Reader) (err error) { sn, err = r.Snippet(ctx, "CCCCCCCC"); return })
	if sn != nil {
		t.Fatal("the refused snippet must not be stored")
	}
}

func TestDemoData(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	for range 2 { // idempotent
		if err := e.bus.Exec(ctx, commands.SeedDemo{}); err != nil {
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
		return e.bus.Exec(ctx, commands.SetFlight{SID: "s", TabID: "tab12345", Name: name, Value: value})
	}
	for _, c := range [][2]string{{"thrust", "70"}, {"shields", "false"}, {"callsign", " apollo-11 "}} {
		if err := send(c[0], c[1]); err != nil {
			t.Fatalf("%s=%s: %v", c[0], c[1], err)
		}
	}
	if err := send("thrust", "95"); !errors.Is(err, commands.ErrThrust) {
		t.Errorf("thrust 95: err = %v, want the business rule", err)
	}
	for _, bad := range [][2]string{{"thrust", "abc"}, {"shields", "maybe"}, {"callsign", "no spaces"}, {"warp", "9"}} {
		if err := send(bad[0], bad[1]); err == nil {
			t.Errorf("%s=%q must be rejected", bad[0], bad[1])
		}
	}
	var st model.TabState
	e.q.View(ctx, func(r *queries.Reader) (err error) { st, _, err = r.Tab(ctx, "s", "tab12345"); return })
	want := model.FlightPlan{Set: true, Thrust: 70, Shields: false, Callsign: "APOLLO-11"}
	if st.Flight != want {
		t.Errorf("flight = %+v, want %+v (normalized, the rejected thrust not applied)", st.Flight, want)
	}
	if (model.FlightPlan{}).OrDefault().Thrust != 40 {
		t.Error("defaults")
	}
}
