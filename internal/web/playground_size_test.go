package web_test

import (
	"io"
	"io/fs"
	"strconv"
	"strings"
	"testing"

	"starbase/internal/ui"
)

// The playground's size line measures like the catalog: a component's own
// source gets the numbers of its size table.
func TestPlaygroundSize(t *testing.T) {
	ts, c, _, cat := newServerBus(t)
	measure := func(slug, code string) string {
		t.Helper()
		body := `{"component":` + strconv.Quote(slug) + `,"code":` + strconv.Quote(code) + `}`
		res := post(t, c, ts.URL+"/playground/size", body, "same-origin")
		b, _ := io.ReadAll(res.Body)
		if res.StatusCode != 200 {
			t.Fatalf("%s: %d %s", slug, res.StatusCode, b)
		}
		return string(b)
	}
	for _, slug := range []string{"code-editor", "code-playground", "button"} {
		comp, _ := cat.Get(slug)
		src, _ := fs.ReadFile(cat.FS, comp.Script)
		got := measure(slug, string(src))
		want := []string{`"own":"` + ui.FmtBytes(comp.Sizes.Files[0].Min) + `"`, `"raw":"` + ui.FmtBytes(comp.Sizes.Files[0].Raw) + `"`}
		if comp.Sizes.Total.Min != comp.Sizes.Files[0].Min {
			want = append(want, `"total":"`+ui.FmtBytes(comp.Sizes.Total.Min)+`"`)
		} else {
			want = append(want, `"total":""`)
		}
		for _, w := range want {
			if !strings.Contains(got, w) {
				t.Errorf("%s: want %s in\n%s", slug, w, got)
			}
		}
	}
	// Code that doesn't parse only marks the last numbers stale.
	if got := measure("", "rocket('sb-x', {"); !strings.Contains(got, `"stale":true`) || strings.Contains(got, `"own"`) {
		t.Errorf("unparseable code:\n%s", got)
	}
	if res := post(t, c, ts.URL+"/playground/size", `{"code":"1"}`, "cross-site"); res.StatusCode != 403 {
		t.Errorf("cross-site: %d", res.StatusCode)
	}
	// The page renders the first numbers into the playground's bar.
	button, _ := cat.Get("button")
	_, page := get(t, c, ts.URL+"/playground?component=button")
	if !strings.Contains(page, `slot="bar" class="pg-size"`) || !strings.Contains(page, ">"+ui.FmtBytes(button.Sizes.Files[0].Min)+"</strong>") {
		t.Error("the playground page lacks the size line with the first numbers")
	}
}
