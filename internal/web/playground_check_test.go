package web_test

import (
	"io"
	"io/fs"
	"strconv"
	"strings"
	"testing"

	"starbase/internal/tscheck"
)

// The playground's type check answers with $_diag for component.js.
func TestPlaygroundCheck(t *testing.T) {
	ts, c, _, cat := newServerBus(t)
	check := func(slug, code string) string {
		t.Helper()
		body := `{"component":` + strconv.Quote(slug) + `,"code":` + strconv.Quote(code) + `}`
		res := post(t, c, ts.URL+"/playground/check", body, "same-origin")
		b, _ := io.ReadAll(res.Body)
		if res.StatusCode != 200 {
			t.Fatalf("%d %s", res.StatusCode, b)
		}
		return string(b)
	}
	typo := "import { rocket } from 'datastar'\nrocket('sb-x', { props: ({ string }) => ({ label: string }), setup: ({ props }) => props.lable })\n"
	if got := check("", typo); !strings.Contains(got, `"_diag":"{\"component.js\":[]}"`) {
		t.Errorf("JavaScript without // @ts-check is not checked:\n%s", got)
	}
	if tscheck.New(t.TempDir()) == nil {
		t.Skip("no compiler embedded: go run ./cmd/fetchtsc")
	}
	if got := check("", "// @ts-check\n"+typo); !strings.Contains(got, `\"line\":3,\"col\":90,\"length\":5,\"text\":\"lable\",\"code\":2551`) {
		t.Errorf("a typo:\n%s", got)
	}
	// TypeScript is always checked, and strictly.
	body := `{"name":"component.ts","code":"export const f = (x) => x\n"}`
	res := post(t, c, ts.URL+"/playground/check", body, "same-origin")
	if b, _ := io.ReadAll(res.Body); !strings.Contains(string(b), `{\"component.ts\":[{\"line\":1,\"col\":19,\"length\":1,\"text\":\"x\",\"code\":7006`) {
		t.Errorf("TypeScript:\n%s", b)
	}
	// A component's own files answer its relative imports (code-editor's vendored Prism).
	editor, _ := cat.Get("code-editor")
	src, _ := fs.ReadFile(cat.FS, editor.Script)
	if got := check("code-editor", "// @ts-check\n"+string(src)); strings.Contains(got, `\"code\":2307`) {
		t.Errorf("an import of the component's own files is not found:\n%s", got)
	}
	// A component written in TypeScript opens as component.ts and checks clean.
	table, _ := cat.Get("data-table")
	if _, page := get(t, c, ts.URL+"/playground?component=data-table"); !strings.Contains(page, "component.ts&#34;") {
		t.Error("the playground doesn't open data-table's TypeScript")
	}
	tsSrc, _ := fs.ReadFile(cat.FS, table.SourceFile)
	body = `{"component":"data-table","name":"component.ts","code":` + strconv.Quote(string(tsSrc)) + `}`
	if b, _ := io.ReadAll(post(t, c, ts.URL+"/playground/check", body, "same-origin").Body); !strings.Contains(string(b), `{\"component.ts\":[]}`) {
		t.Errorf("data-table.ts in the playground:\n%s", b)
	}
	if got := check("", "// @ts-check\nimport { x } from '/etc/x.js'\n"); !strings.Contains(got, `\"col\":19,\"length\":11,\"text\":\"'/etc/x.js'\",\"code\":2307`) {
		t.Errorf("a missing file outside the folder:\n%s", got)
	}
}
