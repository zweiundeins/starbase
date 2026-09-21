package queries_test

import (
	"context"
	"io"
	"log/slog"
	"path/filepath"
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
	var st model.TabState
	var ok bool
	e.q.View(ctx, func(r *queries.Reader) (err error) { st, ok, err = r.Tab(ctx, "s", "tab12345"); return })
	want := model.TabState{Browse: model.Browse{Q: "hi", Sort: model.SortPopular}, PreviewTheme: "nebula"}
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
