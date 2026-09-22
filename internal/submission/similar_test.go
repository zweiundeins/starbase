package submission_test

import (
	"strings"
	"testing"

	"starbase/internal/catalog"
	"starbase/internal/submission"
)

func TestSimilar(t *testing.T) {
	comp := func(slug, category, summary string, tags ...string) *catalog.Component {
		return &catalog.Component{Slug: slug, Meta: catalog.Meta{Name: slug, Tag: "sb-" + slug, Category: category, Summary: summary, Tags: tags}}
	}
	cat := &catalog.Catalog{Components: []*catalog.Component{
		comp("starfield", "media", "A warp-speed pixel starfield.", "canvas", "animation", "background", "space"),
		comp("gauge", "data", "A dial for one value.", "chart", "svg"),
		comp("voxel", "media", "A spinning voxel model.", "3d", "canvas"),
		comp("nebula", "media", "A drifting WebGL nebula in dithered pixels.", "webgl", "space", "background"),
	}}
	got := submission.Similar(cat, cat.Components[3])
	if len(got) != 1 || got[0].Slug != "starfield" {
		t.Fatalf("similar = %v, want only starfield (voxel shares just the category)", got)
	}
	md := submission.SimilarMarkdown(got, "https://starbase.example/")
	if !strings.Contains(md, "[`<sb-starfield>` starfield](https://starbase.example/components/starfield)") {
		t.Errorf("markdown = %s", md)
	}
	if submission.SimilarMarkdown(nil, "") != "No similar components in the catalog." {
		t.Error("empty list")
	}
}
