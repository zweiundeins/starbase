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

func TestReviewNotes(t *testing.T) {
	res := submission.Result{Slug: "chart", Files: map[string][]byte{
		"README.md":     nil,
		"chart.js":      nil,
		"vendor/lib.js": []byte("/*! lib v1 | MIT @someone */\nexport const x = 1\n"),
	}}
	checks := []submission.VendorCheck{{Path: "vendor/lib.js", Problem: "not listed in vendor.json"}}
	md := submission.ReviewNotes(res, nil, "", checks)
	for _, want := range []string{"### Similar components", "No similar components", "### Vendored files", "- ⚠️ `vendor/lib.js` (48 B) is unverified (not listed in vendor.json): `lib v1 | MIT @someone`"} {
		if !strings.Contains(md, want) {
			t.Errorf("notes lack %q:\n%s", want, md)
		}
	}
	verified := submission.ReviewNotes(res, nil, "", []submission.VendorCheck{{Path: "vendor/lib.js", Verified: true, Package: "lib@1.0.0", File: "dist/lib.js", License: "MIT"}})
	if !strings.Contains(verified, "- ✅ `vendor/lib.js` (48 B) is `dist/lib.js` from `lib@1.0.0`, license `MIT`") {
		t.Errorf("verified notes:\n%s", verified)
	}
	if strings.Contains(submission.ReviewNotes(submission.Result{Slug: "x", Files: map[string][]byte{"x.js": nil}}, nil, "", nil), "Vendored") {
		t.Error("no vendored section without vendored files")
	}
}
