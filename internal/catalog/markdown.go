package catalog

import (
	"bytes"
	"html"
	"strings"

	"github.com/alecthomas/chroma/v2"
	chromahtml "github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	gmhtml "github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/util"
	"go.abhg.dev/goldmark/frontmatter"
)

// Markdown renders component docs and content pages.
//
// Fenced code blocks tagged "html preview" render twice: once live, inside a
// demo stage, and once as highlighted, copyable source. Every other fenced
// block is highlighted with chroma using CSS classes (styled in code.css).
var Markdown = goldmark.New(
	goldmark.WithExtensions(
		extension.GFM,
		&frontmatter.Extender{},
	),
	goldmark.WithParserOptions(parser.WithAutoHeadingID()),
	goldmark.WithRendererOptions(
		gmhtml.WithUnsafe(), // docs come from reviewed PRs and contain live demos
		renderer.WithNodeRenderers(util.Prioritized(&codeRenderer{}, 100)),
	),
)

type codeRenderer struct{}

func (r *codeRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(ast.KindFencedCodeBlock, r.render)
}

func (r *codeRenderer) render(w util.BufWriter, src []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}
	n := node.(*ast.FencedCodeBlock)
	info := ""
	if n.Info != nil {
		info = string(n.Info.Segment.Value(src))
	}
	fields := strings.Fields(info)
	lang := ""
	if len(fields) > 0 {
		lang = fields[0]
	}
	var code bytes.Buffer
	for i := 0; i < n.Lines().Len(); i++ {
		seg := n.Lines().At(i)
		code.Write(seg.Value(src))
	}
	source := code.String()

	preview := lang == "html" && len(fields) > 1 && fields[1] == "preview"
	if preview {
		w.WriteString(`<div class="demo"><div class="demo-stage">`)
		w.WriteString(source)
		w.WriteString(`</div><div class="demo-code">`)
	} else {
		w.WriteString(`<div class="code-block">`)
	}
	w.WriteString(`<sb-copy-button class="code-copy" value="`)
	w.WriteString(html.EscapeString(strings.TrimRight(source, "\n")))
	w.WriteString(`"></sb-copy-button>`)
	w.WriteString(Highlight(source, lang))
	w.WriteString(`</div>`)
	if preview {
		w.WriteString(`</div>`)
	}
	return ast.WalkSkipChildren, nil
}

var formatter = chromahtml.New(chromahtml.WithClasses(true), chromahtml.PreventSurroundingPre(false))

// Highlight returns source as highlighted HTML (<pre class="chroma">).
func Highlight(source, lang string) string {
	lexer := lexers.Get(lang)
	if lexer == nil {
		lexer = lexers.Fallback
	}
	lexer = chroma.Coalesce(lexer)
	it, err := lexer.Tokenise(nil, source)
	if err != nil {
		return "<pre class=\"chroma\"><code>" + html.EscapeString(source) + "</code></pre>"
	}
	var b bytes.Buffer
	if err := formatter.Format(&b, styles.Fallback, it); err != nil {
		return "<pre class=\"chroma\"><code>" + html.EscapeString(source) + "</code></pre>"
	}
	return b.String()
}

// RenderMarkdown converts markdown (with optional front matter) to HTML and
// decodes the front matter into meta when non-nil.
func RenderMarkdown(src []byte, meta any) (string, error) {
	ctx := parser.NewContext()
	var b bytes.Buffer
	if err := Markdown.Convert(src, &b, parser.WithContext(ctx)); err != nil {
		return "", err
	}
	if meta != nil {
		if fm := frontmatter.Get(ctx); fm != nil {
			if err := fm.Decode(meta); err != nil {
				return "", err
			}
		}
	}
	return b.String(), nil
}
