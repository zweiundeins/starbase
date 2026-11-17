package sb

import (
	"context"
	"encoding/json"
	"regexp"
	"strings"
	"testing"

	"github.com/a-h/templ"
)

type importMap struct {
	Imports   map[string]string
	Integrity map[string]string
}

var (
	mapRe    = regexp.MustCompile(`^<script type="importmap"(?: nonce="([^"]*)")?>(.*?)</script>`)
	scriptRe = regexp.MustCompile(`<script(?: nonce="([^"]*)")? type="module" src="([^"]+)" integrity="(sha384-[A-Za-z0-9+/=]{64})"></script>`)
)

func imports(t *testing.T, ctx context.Context, p ImportsProps) (importMap, [][]string) {
	t.Helper()
	var b strings.Builder
	if err := Imports(p).Render(ctx, &b); err != nil {
		t.Fatal(err)
	}
	out := b.String()
	m := mapRe.FindStringSubmatch(out)
	if m == nil {
		t.Fatalf("no import map first: %s", out)
	}
	var im importMap
	if err := json.Unmarshal([]byte(m[2]), &im); err != nil {
		t.Fatal(err)
	}
	scripts := scriptRe.FindAllStringSubmatch(out, -1)
	if rest := scriptRe.ReplaceAllString(mapRe.ReplaceAllString(out, ""), ""); rest != "" {
		t.Errorf("more than the map and module scripts: %q", rest)
	}
	return im, scripts
}

func TestImportsComponents(t *testing.T) {
	im, scripts := imports(t, context.Background(), ImportsProps{Components: []Tag{TagCodePlayground, TagToggle, TagCodeEditor, TagToggle}})
	ds := DefaultBase + datastarPath
	if im.Imports["datastar"] != ds || im.Integrity[ds] != datastarIntegrity {
		t.Errorf("Datastar: %+v", im)
	}
	// The playground, the editor it renders (once), the toggle; Prism, which
	// the editor loads on demand, is pinned in the map.
	var srcs []string
	for _, s := range scripts {
		srcs = append(srcs, strings.TrimPrefix(s[2], DefaultBase+"/c/"))
		if s[3] != pins[Tag("sb-"+strings.Split(strings.TrimPrefix(s[2], DefaultBase+"/c/"), "@")[0])].integrity {
			t.Errorf("%s: integrity %s", s[2], s[3])
		}
	}
	want := []string{pins[TagCodePlayground].bundle, pins[TagCodeEditor].bundle, pins[TagToggle].bundle}
	if strings.Join(srcs, " ") != strings.Join(want, " ") {
		t.Errorf("scripts %v, want %v", srcs, want)
	}
	prism := pins[TagCodeEditor].lazy[0]
	if im.Integrity[DefaultBase+"/c/"+prism.path] != prism.integrity || len(im.Integrity) != 2 {
		t.Errorf("integrity: %v", im.Integrity)
	}
}

func TestImportsAutoloader(t *testing.T) {
	ctx := templ.WithNonce(context.Background(), `n"1`)
	im, scripts := imports(t, ctx, ImportsProps{Base: "https://example.com/sb/"})
	if len(scripts) != 1 || scripts[0][2] != "https://example.com/sb/c/@"+snapshot+"/autoloader.js" || scripts[0][3] != snapshotIntegrity || scripts[0][1] != "n&#34;1" {
		t.Fatalf("scripts: %v", scripts)
	}
	// Every bundle and every file one loads on demand, and Datastar.
	n := 1
	for _, p := range pins {
		n += 1 + len(p.lazy)
		if im.Integrity["https://example.com/sb/c/"+p.bundle] != p.integrity {
			t.Errorf("%s is not pinned", p.bundle)
		}
	}
	if len(im.Integrity) != n || len(pins) != len(pinOrder) {
		t.Errorf("%d pins, want %d", len(im.Integrity), n)
	}
}

func TestImportsOwnDatastar(t *testing.T) {
	im, _ := imports(t, context.Background(), ImportsProps{Components: []Tag{TagToggle}, Datastar: "/js/datastar.js"})
	if im.Imports["datastar"] != "/js/datastar.js" || len(im.Integrity) != 0 {
		t.Errorf("%+v", im)
	}
	im, _ = imports(t, context.Background(), ImportsProps{Components: []Tag{TagToggle}, Datastar: "/js/datastar.js", DatastarIntegrity: "sha384-x"})
	if im.Integrity["/js/datastar.js"] != "sha384-x" {
		t.Errorf("%+v", im)
	}
}

func TestImportsUnknownComponent(t *testing.T) {
	var b strings.Builder
	err := Imports(ImportsProps{Components: []Tag{"sb-nope"}}).Render(context.Background(), &b)
	if err == nil || !strings.Contains(err.Error(), "sb-nope") || b.Len() != 0 {
		t.Errorf("%v, wrote %q", err, b.String())
	}
}

// Every tag has a pin, and a pin's dependencies are tags.
func TestPins(t *testing.T) {
	for _, e := range Entries {
		p, ok := pins[Tag(e.Tag)]
		if !ok || !strings.HasPrefix(p.bundle, strings.TrimPrefix(e.Tag, "sb-")+"@") {
			t.Errorf("%s: pin %+v", e.Tag, p)
		}
		for _, d := range p.deps {
			if _, ok := pins[d]; !ok {
				t.Errorf("%s depends on %s, which has no pin", e.Tag, d)
			}
		}
	}
}
