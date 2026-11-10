package commands_test

import (
	"context"
	"io"
	"log/slog"
	"path/filepath"
	"slices"
	"testing"
	"testing/fstest"
	"time"

	"github.com/zweiundeins/lith/writer"

	"starbase/internal/catalog"
	"starbase/internal/commands"
	"starbase/internal/db"
	"starbase/internal/model"
)

// TestTopics: every command touches the topics of the pages that show what
// it changed, and nothing when it changed nothing a page shows. A command
// that touches nothing leaves pages stale.
func TestTopics(t *testing.T) {
	ctx := context.Background()
	d, err := db.Open(ctx, filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { d.Close() })
	var touched []string
	w, err := writer.New(d, writer.Options{
		OnCommit: func(topics []string) { touched = topics },
		Log:      slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	if err != nil {
		t.Fatal(err)
	}
	go w.Run(ctx)
	t.Cleanup(func() { w.Close() })
	cat, err := catalog.Load(fstest.MapFS{
		"alpha/README.md": {Data: []byte("---\nname: Alpha\ntag: sb-alpha\ncategory: forms\nsummary: A.\nauthor: octocat\nsince: 2026-01-01\npreview: <sb-alpha></sb-alpha>\n---\nDocs.\n")},
		"alpha/alpha.js":  {Data: []byte("rocket('sb-alpha', {})")},
	})
	if err != nil {
		t.Fatal(err)
	}

	sid, tab := "session1", "tab12345"
	all, session, page := []string{commands.TopicAll}, []string{commands.SessionTopic(sid)}, []string{commands.TabTopic(tab)}
	board := []string{commands.TopicBoard}
	for i, tc := range []struct {
		cmd  writer.Command
		want []string
	}{
		{commands.SyncCatalog{Catalog: cat}, all},
		{commands.SeedBoard{}, nil},
		{commands.SeedDemo{}, nil},
		{commands.SignIn{SID: sid, User: model.User{GitHubID: 7, Login: "ada"}}, session},
		{commands.Star{UserID: 1, Slug: "alpha"}, all},
		{commands.Star{UserID: 1, Slug: "alpha"}, nil}, // starred already
		{commands.Unstar{UserID: 1, Slug: "alpha"}, all},
		{commands.PaintPixels{Board: commands.BoardName, Color: 3, Cells: []int{0}}, board},
		{commands.SetBrowseFilter{SID: sid, TabID: tab, Browse: model.Browse{Q: "x"}}, page},
		{commands.SetFlight{SID: sid, TabID: tab, Name: "thrust", Value: "40"}, page},
		{commands.SaveSnippet{SID: sid, TabID: tab, ID: "AbCd2345", Files: map[string]string{"component.js": "x"}}, page},
		{commands.SetGallerySort{SID: sid, TabID: tab, Browse: model.Browse{Sort: model.SortNewest}}, session},
		{commands.SetInstallTab{SID: sid, Tab: "pinned"}, session},
		{commands.SetPreviewTheme{SID: sid, Theme: "nebula"}, session},
		{commands.SetPreviewStyle{SID: sid, Smooth: true}, session},
		{commands.SignOut{SID: sid}, session},
		{commands.PruneTabs{OlderThan: time.Hour}, nil},
		{commands.PruneSessionPrefs{OlderThan: time.Hour}, nil},
	} {
		touched = []string{"nothing committed"}
		if err := w.Do(ctx, tc.cmd); err != nil {
			t.Fatalf("%d %T: %v", i, tc.cmd, err)
		}
		if !slices.Equal(touched, tc.want) {
			t.Errorf("%d %T touched %q, want %q", i, tc.cmd, touched, tc.want)
		}
	}
}
