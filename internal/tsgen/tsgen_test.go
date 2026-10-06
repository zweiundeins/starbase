package tsgen

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"starbase/internal/tscheck"
)

// Every TypeScript component compiles without errors, into the .js committed next to it.
func TestComponentsUpToDate(t *testing.T) {
	c := tscheck.New(t.TempDir())
	if c == nil {
		t.Skip("no compiler embedded: go run ./cmd/fetchtsc")
	}
	res, err := Refresh(context.Background(), c, "../..", false)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range res {
		for _, d := range r.Diagnostics {
			t.Errorf("components/%s:%d:%d: TS%d %s", d.File, d.Line, d.Col, d.Code, d.Message)
		}
		if r.Stale {
			t.Errorf("%s: its .js is stale; run go tool task ts", r.Source)
		}
	}
}

func TestRefresh(t *testing.T) {
	c := tscheck.New(t.TempDir())
	if c == nil {
		t.Skip("no compiler embedded: go run ./cmd/fetchtsc")
	}
	root := t.TempDir()
	write := func(name, s string) {
		os.MkdirAll(filepath.Dir(filepath.Join(root, name)), 0o755)
		os.WriteFile(filepath.Join(root, name), []byte(s), 0o644)
	}
	write("components/x/x.ts", "import { pad } from './vendor/pad.js'\nimport { twice } from './lib/twice.ts'\nexport const a: string = twice(pad('a'))\n")
	write("components/x/lib/twice.ts", "export const twice = (s: string) => s + s\n")
	write("components/x/vendor/pad.js", "export const pad = (s) => s\n")
	write("components/x/vendor/pad.d.ts", "export declare const pad: (s: string) => string\n")
	write("components/y/y.js", "export const b = 1\n") // JavaScript components stay as they are
	ctx := context.Background()
	if res, err := Refresh(ctx, c, root, true); err != nil || len(res) != 2 || !res[0].Stale || res[0].Source != "components/x/lib/twice.ts" {
		t.Fatalf("first run: %v %+v", err, res)
	}
	got, _ := os.ReadFile(filepath.Join(root, "components/x/x.js"))
	if !strings.HasPrefix(string(got), "// Generated from x.ts") || !strings.Contains(string(got), "export const a = twice(pad('a'));") || !strings.Contains(string(got), "from './lib/twice.js'") {
		t.Errorf("x.js:\n%s", got)
	}
	if helper, _ := os.ReadFile(filepath.Join(root, "components/x/lib/twice.js")); !strings.HasPrefix(string(helper), "// Generated from twice.ts") {
		t.Errorf("lib/twice.js:\n%s", helper)
	}
	if _, err := os.Stat(filepath.Join(root, "components/x/vendor/pad.d.js")); err == nil {
		t.Error("declarations compiled into JavaScript")
	}
	if res, _ := Refresh(ctx, c, root, false); len(res) != 0 {
		t.Errorf("stale after a write: %+v", res)
	}
	write("components/x/x.ts", "export const f = (n) => n\n")
	if res, _ := Refresh(ctx, c, root, true); len(res) != 1 || len(res[0].Diagnostics) != 1 || res[0].Source != "components/x/x.ts" {
		t.Errorf("an error: %+v", res)
	}
	if got2, _ := os.ReadFile(filepath.Join(root, "components/x/x.js")); string(got2) != string(got) {
		t.Error("a component with errors was written")
	}
}
