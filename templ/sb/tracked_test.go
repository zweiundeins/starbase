package sb

import (
	"context"
	"encoding/json"
	"io"
	"strings"
	"sync"
	"testing"

	"github.com/a-h/templ"
)

func children(cs ...templ.Component) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		for _, c := range cs {
			if err := c.Render(ctx, w); err != nil {
				return err
			}
		}
		return nil
	})
}

// tracked renders Tracked(p) around body and splits its output into what
// the body wrote, the import map and the module scripts (src → integrity).
func tracked(t *testing.T, ctx context.Context, p ImportsProps, body templ.Component) (string, importMap, [][]string) {
	t.Helper()
	var b strings.Builder
	if err := Tracked(p).Render(templ.WithChildren(ctx, body), &b); err != nil {
		t.Fatal(err)
	}
	out := b.String()
	i := strings.Index(out, `<script type="importmap"`)
	if i < 0 {
		t.Fatalf("no import map: %s", out)
	}
	if strings.Contains(out[:i], "<script") {
		t.Errorf("a script before the import map: %s", out)
	}
	m := mapRe.FindStringSubmatch(out[i:])
	var im importMap
	if err := json.Unmarshal([]byte(m[2]), &im); err != nil {
		t.Fatal(err)
	}
	nonce := ""
	if m[1] != "" {
		nonce = ` nonce="` + m[1] + `"`
	}
	after := datastarFirst(t, out[i+len(m[0]):], im, nonce)
	scripts := scriptRe.FindAllStringSubmatch(after, -1)
	if rest := scriptRe.ReplaceAllString(after, ""); rest != "" {
		t.Errorf("after the map, more than module scripts: %q", rest)
	}
	for _, s := range scripts {
		if s[1] != m[1] {
			t.Errorf("%s: nonce %q, the map's %q", s[2], s[1], m[1])
		}
	}
	return out[:i], im, scripts
}

func srcs(t *testing.T, scripts [][]string) []string {
	t.Helper()
	var out []string
	for _, s := range scripts {
		src := strings.TrimPrefix(s[2], DefaultBase+"/c/")
		out = append(out, src)
		if pin := pins[Tag("sb-"+strings.Split(src, "@")[0])]; s[3] != pin.integrity {
			t.Errorf("%s: integrity %s", s[2], s[3])
		}
	}
	return out
}

func bundles(tags ...Tag) []string {
	var out []string
	for _, t := range tags {
		out = append(out, pins[t].bundle)
	}
	return out
}

func TestTrackedLoadsWhatRendered(t *testing.T) {
	// Nested (a toggle in a modal), repeated, and a component with a dep
	// (the playground renders the editor, which loads Prism on demand).
	body := children(
		Toggle(ToggleProps{Label: "a"}),
		templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
			return Modal(ModalProps{}).Render(templ.WithChildren(ctx, children(Toggle(ToggleProps{}), CodePlayground(CodePlaygroundProps{}))), w)
		}),
		Toggle(ToggleProps{Label: "c"}),
	)
	ctx := templ.WithNonce(context.Background(), "n0")
	html, im, scripts := tracked(t, ctx, ImportsProps{}, body)
	if !strings.HasPrefix(html, `<sb-toggle label="a"></sb-toggle><sb-modal><sb-toggle></sb-toggle><sb-code-playground>`) {
		t.Errorf("body: %s", html)
	}
	want := bundles(TagToggle, TagModal, TagCodePlayground, TagCodeEditor)
	if got := srcs(t, scripts); strings.Join(got, " ") != strings.Join(want, " ") {
		t.Errorf("scripts %v, want %v", got, want)
	}
	if scripts[0][1] != "n0" {
		t.Errorf("nonce %q", scripts[0][1])
	}
	prism := pins[TagCodeEditor].lazy[0]
	if ds := DefaultBase + datastarPath; im.Imports["datastar"] != ds || im.Integrity[ds] != datastarIntegrity ||
		im.Integrity[DefaultBase+"/c/"+prism.path] != prism.integrity || len(im.Integrity) != 2 {
		t.Errorf("map: %+v", im)
	}
}

func TestTrackedRecordsOnlyInside(t *testing.T) {
	// A wrapper rendered outside a Tracked records nothing, and a Tracked
	// inside another leaves the imports to the outer one.
	if err := Slider(SliderProps{}).Render(context.Background(), io.Discard); err != nil {
		t.Fatal(err)
	}
	inner := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		return Tracked(ImportsProps{Components: []Tag{TagRating}}).Render(templ.WithChildren(ctx, Tabs(TabsProps{})), w)
	})
	html, _, scripts := tracked(t, context.Background(), ImportsProps{Components: []Tag{TagToast}}, inner)
	if html != "<sb-tabs></sb-tabs>" {
		t.Errorf("body: %s", html)
	}
	if got, want := srcs(t, scripts), bundles(TagToast, TagTabs); strings.Join(got, " ") != strings.Join(want, " ") {
		t.Errorf("scripts %v, want %v", got, want)
	}
}

func TestTrackedNothingRendered(t *testing.T) {
	_, im, scripts := tracked(t, context.Background(), ImportsProps{Base: "https://example.com/"}, children())
	if len(scripts) != 0 || im.Imports["datastar"] != "https://example.com"+datastarPath {
		t.Errorf("%v %+v", scripts, im)
	}
}

func TestTrackedConcurrent(t *testing.T) {
	all := []Tag{TagToggle, TagSlider, TagRating, TagDataTable}
	body := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		var wg sync.WaitGroup
		for i := range 64 {
			wg.Go(func() {
				var c templ.Component
				switch all[i%len(all)] {
				case TagToggle:
					c = Toggle(ToggleProps{})
				case TagSlider:
					c = Slider(SliderProps{})
				case TagRating:
					c = Rating(RatingProps{})
				default:
					c = DataTable(DataTableProps{})
				}
				c.Render(ctx, io.Discard)
			})
		}
		wg.Wait()
		return nil
	})
	_, _, scripts := tracked(t, context.Background(), ImportsProps{}, body)
	got := srcs(t, scripts)
	if len(got) != 5 { // and the table's virtual scroll
		t.Errorf("scripts %v", got)
	}
}

func TestTrackedUnknownComponent(t *testing.T) {
	err := Tracked(ImportsProps{Components: []Tag{"sb-nope"}}).Render(templ.WithChildren(context.Background(), children()), io.Discard)
	if err == nil || !strings.Contains(err.Error(), "sb-nope") {
		t.Errorf("%v", err)
	}
}
