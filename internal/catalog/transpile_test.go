package catalog

import (
	"errors"
	"strings"
	"testing"
	"testing/fstest"
)

// The JavaScript runs; its lines lead back to the TypeScript as written.
func TestTranspile(t *testing.T) {
	src := "import { rocket } from 'datastar'\n\n// A comment.\ninterface Props {\n\tlabel: string\n}\n\nconst styles = /* css */ `\n:host { display: block; }\n`\n\nrocket('sb-x', {\n\tprops: ({ string }) => ({ label: string }),\n\n\tsetup: ({ host }: { host: HTMLElement }) => {\n\t\tconst n: number = 1\n\t\tthrow new Error('line 17')\n\t},\n})\n"
	got, err := Transpile(src)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(got.Code, "interface") || strings.Contains(got.Code, ": number") {
		t.Errorf("types left in:\n%s", got.Code)
	}
	js := strings.Split(got.Code, "\n")
	if len(got.Lines) != len(js) {
		t.Fatalf("%d lines mapped for %d", len(got.Lines), len(js))
	}
	for i, l := range js {
		if strings.Contains(l, "line 17") && got.Lines[i] != 17 {
			t.Errorf("the throw on line %d maps to %d, want 17", i+1, got.Lines[i])
		}
		if strings.Contains(l, "rocket(") && got.Lines[i] != 12 {
			t.Errorf("rocket( on line %d maps to %d, want 12", i+1, got.Lines[i])
		}
	}
	if got, _ := Transpile("import { a } from './lib/a.ts'\nimport './b.ts'\nconst c = await import('../c.ts')\nexport const s: string = './d.ts' + a\n"); !strings.Contains(got.Code, `from "./lib/a.js"`) || !strings.Contains(got.Code, `import "./b.js"`) || !strings.Contains(got.Code, `import("../c.js")`) || !strings.Contains(got.Code, `"./d.ts"`) {
		t.Errorf("relative TypeScript imports, and only those:\n%s", got.Code)
	}
	var te *TranspileError
	if _, err := Transpile("const a = 1\nconst b: = 2\n"); !errors.As(err, &te) || te.Line != 2 {
		t.Errorf("a syntax error: %v", err)
	}
}

func TestTypeScriptFiles(t *testing.T) {
	cat := &Catalog{FS: fstest.MapFS{
		"x/x.ts":          {Data: []byte("a")},
		"x/x.js":          {Data: []byte("b")},
		"x/lib/util.ts":   {Data: []byte("c")},
		"x/vendor/v.d.ts": {Data: []byte("d")},
		"x/README.md":     {Data: []byte("e")},
		"other/other.ts":  {Data: []byte("f")},
	}}
	got, err := cat.TypeScriptFiles(&Component{Slug: "x"})
	if err != nil || len(got) != 3 || string(got["x.ts"]) != "a" || string(got["lib/util.ts"]) != "c" || string(got["vendor/v.d.ts"]) != "d" {
		t.Errorf("%v %v", err, got)
	}
}
