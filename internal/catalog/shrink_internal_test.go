package catalog

import (
	"strings"
	"testing"

	"github.com/evanw/esbuild/pkg/api"

	"starbase/components"
)

func TestShrink(t *testing.T) {
	for _, tc := range []struct{ name, in, want string }{
		{"docs", "props: ({ string }) => ({ a: string.trim.docs({ description: 'x (y)' }).default('z') })",
			"props: ({ string }) => ({ a: string.trim.default('z') })"},
		{"docs with a template", "a: string.docs({ description: `a ${b}` }),", "a: string,"},
		{"manifest", "rocket('sb-x', {\n\tmanifest: { events: [{ name: 'sb-y' }] },\n\trender: () => 1,\n})",
			"rocket('sb-x', {\n\t\n\trender: () => 1,\n})"},
		{"manifest last", "rocket('sb-x', { render: 1, manifest: { slots: [] } })", "rocket('sb-x', { render: 1,  })"},
		{"a manifest key elsewhere stays", "const o = { manifest: 1 }; o.manifest", "const o = { manifest: 1 }; o.manifest"},
		{"css", "const s = /* css */ `\n:host {\n\tdisplay: block; /* why */\n\tcolor: red;\n}\n.a .b, .c { margin: 0 auto; }\n`",
			"const s = /* css */ ` :host{display:block;color:red}.a .b,.c{margin:0 auto}`"},
		{"css strings stay", "/* css */ `a::before { content: \"  /* x */  \"; }`", "/* css */ `a::before{content:\"  /* x */  \"}`"},
		{"css keeps a space next to an interpolation", "/* css */ `.a ${sel} { top: ${y}px; }`", "/* css */ `.a ${sel}{top:${y}px}`"},
		{"an untagged template stays", "const s = `\n  a  {  b: c  }\n`", "const s = `\n  a  {  b: c  }\n`"},
		{"html", "html`\n\t<div\n\t\tclass=\"a\"\n\t\tdata-on:click=\"@x()\">\n\t\t<!-- note -->\n\t\t<span>${a}</span>\n\t</div>\n`",
			"html` <div class=\"a\" data-on:click=\"@x()\"> <span>${a}</span> </div> `"},
		{"html keeps attribute values and pre", "html`<p title=\"a\n  b\">x\n  y</p><pre>\n  keep\n</pre>`",
			"html`<p title=\"a\n  b\">x y</p><pre>\n  keep\n</pre>`"},
		{"regex and strings are not code", "const r = /`[.docs(]/g; const s = '.docs(`'", "const r = /`[.docs(]/g; const s = '.docs(`'"},
	} {
		if got := string(shrink("x.js", []byte(tc.in))); got != tc.want {
			t.Errorf("%s:\n got %q\nwant %q", tc.name, got, tc.want)
		}
	}
	// Vendored files and files the scanner can't read ship as they are.
	for name, in := range map[string]string{"vendor/lib.js": "a.docs({})", "x.js": "a.docs({ `unterminated"} {
		if got := string(shrink(name, []byte(in))); got != in {
			t.Errorf("%s: %q became %q", name, in, got)
		}
	}
}

// Every shrunk component module is still valid JavaScript, and shrinking
// changes nothing but what it is meant to: parsing it with esbuild works, and
// no .docs( call survives outside strings.
func TestShrinkComponents(t *testing.T) {
	cat, err := Load(components.FS)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range cat.Components {
		files, err := cat.ModuleFiles(c)
		if err != nil {
			t.Fatal(err)
		}
		for name, src := range files {
			// Shrinking removes any manifest: {…} property, meant for rocket()'s:
			// a module can't use the key for anything else.
			if r, err := shScan(string(src)); err == nil && len(r.manifests) > 1 {
				t.Errorf("%s/%s: %d manifest: {…} properties; only rocket()'s may use that key (shrink.go removes them all)", c.Slug, name, len(r.manifests))
			}
			out := shrink(name, src)
			if strings.HasPrefix(name, "vendor/") {
				if string(out) != string(src) {
					t.Errorf("%s/%s: a vendored file was changed", c.Slug, name)
				}
				continue
			}
			if res := api.Transform(string(out), api.TransformOptions{Loader: api.LoaderJS, Format: api.FormatESModule}); len(res.Errors) > 0 {
				t.Errorf("%s/%s: shrunk source doesn't parse: %s", c.Slug, name, res.Errors[0].Text)
			}
			if r, err := shScan(string(out)); err != nil {
				t.Errorf("%s/%s: %v", c.Slug, name, err)
			} else if len(r.docs) > 0 || len(r.manifests) > 0 {
				t.Errorf("%s/%s: %d docs and %d manifests left", c.Slug, name, len(r.docs), len(r.manifests))
			}
			if name == c.Slug+".js" && string(out) == string(src) {
				t.Errorf("%s: nothing was shrunk (did the scanner give up?)", c.Slug)
			}
		}
	}
}
