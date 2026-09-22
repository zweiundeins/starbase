package submission_test

import (
	"context"
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	"starbase/internal/catalog"
	"starbase/internal/submission"
)

func TestImports(t *testing.T) {
	code := `import { rocket } from 'datastar'
import * as echarts from "./vendor/echarts.js"
import './side-effect.js'
export { x } from './re.js'
const lazy = () => import('./lazy.js')
// a string that says import is fine
const s = 'no import here'`
	want := []string{"datastar", "./vendor/echarts.js", "./side-effect.js", "./re.js", "./lazy.js"}
	if got := catalog.Imports(code); !slices.Equal(got, want) {
		t.Fatalf("imports = %q, want %q", got, want)
	}
}

const chartCode = "import { rocket } from 'datastar'\nimport * as lib from './vendor/lib.js'\nrocket('sb-chart', { render: ({ html }) => html`<b>${lib.x}</b>` })\n"

func TestFetchVendoredFiles(t *testing.T) {
	gh := fakeGitHub(t, map[string]string{
		"src/chart.js":           chartCode,
		"src/README.md":          "---\nsummary: A chart\npreview: <sb-chart></sb-chart>\n---\nDocs.\n",
		"src/vendor/lib.js":      "/*! lib v1 | MIT */\nexport * from './chunk.js'\nexport const x = 1\n",
		"src/vendor/chunk.js":    "export const y = 2\n",
		"src/vendor/unused.js":   "export const z = 3\n",
		"src/test/chart.test.js": "import './../chart.js'",
	})
	src, err := submission.Fetch(context.Background(), gh.Client(), gh.URL, "", "https://github.com/you/repo/tree/main/src")
	if err != nil {
		t.Fatal(err)
	}
	var keys []string
	for k := range src.Vendor {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	if !slices.Equal(keys, []string{"vendor/chunk.js", "vendor/lib.js"}) {
		t.Fatalf("vendored = %q: only the files the component reaches", keys)
	}
	sub := submission.Submission{Name: "Chart", Category: "data", Licensed: true}
	sub.Apply(src)
	res, err := sub.Build("you", timeNow())
	if err != nil {
		t.Fatal(err)
	}
	fsys := fstest.MapFS{}
	for name, data := range res.Files {
		fsys["chart/"+name] = &fstest.MapFile{Data: data}
	}
	if _, ok := fsys["chart/vendor/lib.js"]; !ok {
		t.Fatalf("files = %v", fsys)
	}
	if _, err := catalog.Load(fsys); err != nil {
		t.Fatalf("generated folder is invalid: %v", err)
	}
}

func TestImportRules(t *testing.T) {
	base := submission.Submission{Name: "Chart", Category: "data", Summary: "A chart.", Preview: "<sb-chart></sb-chart>", Licensed: true}
	for _, tc := range []struct {
		name, code string
		vendor     map[string]string
		want       string
	}{
		{"bare import", "import * as e from 'echarts'\nrocket('sb-chart', {})", nil, "vendor the library"},
		{"url import", "import * as e from 'https://cdn.example/e.js'\nrocket('sb-chart', {})", nil, "vendor the library"},
		{"outside the folder", "import x from '../x.js'\nrocket('sb-chart', {})", map[string]string{"../x.js": ""}, "outside the component's folder"},
		{"missing file (pasted code)", "import x from './lib.js'\nrocket('sb-chart', {})", nil, "link a repository"},
		{"vendored bare import", "import x from './lib.js'\nrocket('sb-chart', {})", map[string]string{"lib.js": "import 'lodash'"}, "`component/lib.js` imports `lodash`"},
		{"name clash", "import x from './README.md'\nrocket('sb-chart', {})", map[string]string{"README.md": ""}, "clashes"},
	} {
		sub := base
		sub.Code, sub.Vendor = tc.code, tc.vendor
		_, err := sub.Build("you", timeNow())
		if err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%s: err = %v, want %q", tc.name, err, tc.want)
		}
	}
	ok := base
	ok.Code = "import { rocket } from 'datastar'\nrocket('sb-chart', {})"
	if _, err := ok.Build("you", timeNow()); err != nil {
		t.Errorf("datastar-only import: %v", err)
	}
}
