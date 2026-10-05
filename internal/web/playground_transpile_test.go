package web_test

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"testing"
)

// The runner's TypeScript: types out, lines mapped, from its opaque origin.
func TestPlaygroundTranspile(t *testing.T) {
	ts, c, _, _ := newServerBus(t)
	transpile := func(src string) (map[string]any, *http.Response) {
		t.Helper()
		req, _ := http.NewRequest(http.MethodPost, ts.URL+"/playground/transpile", strings.NewReader(src))
		req.Header.Set("Content-Type", "text/plain;charset=UTF-8")
		req.Header.Set("Sec-Fetch-Site", "cross-site")
		req.Header.Set("Origin", "null")
		res, err := c.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		var out map[string]any
		b, _ := io.ReadAll(res.Body)
		if res.StatusCode != 200 || json.Unmarshal(b, &out) != nil {
			t.Fatalf("%d %s", res.StatusCode, b)
		}
		return out, res
	}
	out, res := transpile("interface A { x: number }\n\nconst a: A = { x: 1 }\nthrow new Error(String(a.x))\n")
	if res.Header.Get("Access-Control-Allow-Origin") != "*" {
		t.Error("the runner's opaque origin can't read the answer")
	}
	code, _ := out["code"].(string)
	lines, _ := out["lines"].([]any)
	if strings.Contains(code, "interface") || len(lines) != strings.Count(code, "\n")+1 {
		t.Fatalf("got %v", out)
	}
	for i, l := range strings.Split(code, "\n") {
		if strings.Contains(l, "throw") && lines[i] != 4.0 {
			t.Errorf("the throw maps to %v, want line 4", lines[i])
		}
	}
	if out, _ := transpile("const a = 1\nconst b: = 2\n"); out["error"] == nil || out["line"] != 2.0 {
		t.Errorf("a syntax error: %v", out)
	}
	// The size line measures TypeScript as the JavaScript it becomes.
	body := `{"name":"component.ts","code":` + strconv.Quote("const a: number = 1\nexport { a }\n") + `}`
	if b, _ := io.ReadAll(post(t, c, ts.URL+"/playground/size", body, "same-origin").Body); !strings.Contains(string(b), `"title":"component.ts: `) {
		t.Errorf("size of TypeScript: %s", b)
	}
}
