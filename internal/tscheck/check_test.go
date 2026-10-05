package tscheck

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf16"
)

const component = `import { rocket } from 'datastar'
rocket('sb-demo', {
	props: ({ string, bool }) => ({ label: string, open: bool }),
	setup: ({ host, props }) => {
		const n = props.lable.length
	},
})
`

func TestCheck(t *testing.T) {
	c := New(t.TempDir())
	if c == nil {
		t.Skip("no compiler embedded: go run ./cmd/fetchtsc")
	}
	ctx := context.Background()
	check := func(files ...File) []Diagnostic {
		t.Helper()
		ds, err := c.Check(ctx, files)
		if err != nil {
			t.Fatal(err)
		}
		return ds
	}

	ds := check(File{"component.ts", component})
	if len(ds) != 1 || ds[0].Code != 2551 || ds[0].Line != 5 || ds[0].Col != 19 || ds[0].Length != 5 || ds[0].Text != "lable" || !strings.Contains(ds[0].Message, "Did you mean 'label'?") {
		t.Errorf("a typo in a prop: %+v", ds)
	}
	if ds := check(File{"component.js", component}); len(ds) != 0 {
		t.Errorf("JavaScript without // @ts-check is not checked: %+v", ds)
	}
	if ds := check(File{"component.js", "// @ts-check\n" + component}); len(ds) != 1 || ds[0].Line != 6 {
		t.Errorf("JavaScript with // @ts-check: %+v", ds)
	}

	// Relative imports reach the other files; messages keep their continuation lines.
	ds = check(
		File{"component.ts", "import { f } from './lib/f.ts'\nconst a: { x: number } = { x: f() }\n"},
		File{"lib/f.ts", "export const f = (): string => ''\n"},
	)
	if len(ds) != 1 || ds[0].Code != 2322 || ds[0].Line != 2 {
		t.Errorf("an imported module: %+v", ds)
	}

	// Nothing outside the folder, even where it exists.
	secret := filepath.Join(t.TempDir(), "secret.ts")
	os.WriteFile(secret, []byte("export const s: number = 'x'\n"), 0o644)
	for _, src := range []string{
		"import { s } from '" + secret + "'\n",
		"/// <reference path=\"" + secret + "\" />\n",
		"const s: typeof import('" + secret + "') = null\n",
	} {
		if _, err := c.Check(ctx, []File{{"component.ts", src}}); !errors.Is(err, ErrRefused) {
			t.Errorf("%q: %v, want ErrRefused", src, err)
		}
	}
	if _, err := c.Check(ctx, []File{{"../component.ts", ""}}); err == nil {
		t.Error("a file name outside the folder is accepted")
	}
}

func TestWordAt(t *testing.T) {
	lines := []string{"const héllo = 1", "  a.$b_2 + 1", "😀 x", `import 'a\'b' from "c"`}
	for _, tc := range []struct {
		line, col int
		want      string
	}{
		{1, 7, "héllo"}, {1, 13, "="}, {2, 5, "$b_2"}, {3, 4, "x"}, {3, 3, " "}, {4, 8, `'a\'b'`}, {4, 20, `"c"`}, {5, 1, ""}, {2, 99, ""},
	} {
		if got := string(utf16.Decode(wordAt(lines, tc.line, tc.col))); got != tc.want {
			t.Errorf("wordAt(%d, %d) = %q, want %q", tc.line, tc.col, got, tc.want)
		}
	}
}

func TestValidName(t *testing.T) {
	for name, want := range map[string]bool{
		"component.ts": true, "component.js": true, "vendor/prism.js": true, "a.mts": true,
		"../x.ts": false, "/x.ts": false, "./x.ts": false, "a//b.ts": false, ".x.ts": false,
		"tsconfig.json": false, "x.ts/": false, "a\\b.ts": false, "": false,
	} {
		if got := validName(name); got != want {
			t.Errorf("validName(%q) = %v, want %v", name, got, want)
		}
	}
}
