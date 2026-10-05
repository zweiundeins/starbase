package catalog

import (
	"errors"
	"strings"
	"testing"
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
	var te *TranspileError
	if _, err := Transpile("const a = 1\nconst b: = 2\n"); !errors.As(err, &te) || te.Line != 2 {
		t.Errorf("a syntax error: %v", err)
	}
}
