// Package listings keeps Go listings in the docs true to the code. A fenced
// block whose info string names a source,
//
//	```go source=internal/web/demo_data.go#Server.demoChildren,Server.demoAnswer
//
// holds exactly those declarations, doc comments included, as they are in
// that file. `go tool task listings` rewrites the blocks; a test fails when
// one no longer matches its source.
package listings

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

// Docs are the Markdown files that may hold listings, relative to the repo root.
var Docs = []string{"components/*/README.md", "content/*.md"}

// Block is one listing in a Markdown file: its source and the body between the fences.
type Block struct {
	Line       int // of the opening fence, 1-based
	Source     string
	Names      []string
	Body       string
	start, end int // byte range of the body
}

// Find returns the listings in md, in order.
func Find(md []byte) []Block {
	var out []Block
	lines := bytes.SplitAfter(md, []byte("\n"))
	pos := 0
	for i := 0; i < len(lines); i++ {
		line := bytes.TrimRight(lines[i], "\n")
		pos += len(lines[i])
		fence := len(line) - len(bytes.TrimLeft(line, "`"))
		if fence < 3 {
			continue
		}
		src := ""
		for _, f := range strings.Fields(string(line[fence:])) {
			src = strings.TrimPrefix(f, "source=")
			if src != f {
				break
			}
			src = ""
		}
		start := pos
		j := i + 1
		for ; j < len(lines) && !closes(lines[j], fence); j++ {
			pos += len(lines[j])
		}
		if src != "" {
			file, names, _ := strings.Cut(src, "#")
			out = append(out, Block{Line: i + 1, Source: file, Names: strings.Split(names, ","), Body: string(md[start:pos]), start: start, end: pos})
		}
		if j < len(lines) {
			pos += len(lines[j]) // the closing fence
		}
		i = j
	}
	return out
}

// closes reports whether line closes a fence of n backticks.
func closes(line []byte, n int) bool {
	line = bytes.TrimRight(line, " \n")
	return len(line) >= n && len(bytes.Trim(line, "`")) == 0
}

// Extract returns the named declarations of the Go file at root/file, each
// with its doc comment, separated by a blank line. A name is a function or a
// type, const or var ("SetFlight"), or a method as Type.Method.
func Extract(root, file string, names []string) (string, error) {
	src, err := os.ReadFile(filepath.Join(root, file))
	if err != nil {
		return "", err
	}
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, file, src, parser.ParseComments)
	if err != nil {
		return "", err
	}
	decls := map[string]ast.Node{}
	docs := map[string]*ast.CommentGroup{}
	for _, d := range f.Decls {
		switch d := d.(type) {
		case *ast.FuncDecl:
			name := d.Name.Name
			if d.Recv != nil && len(d.Recv.List) > 0 {
				t := d.Recv.List[0].Type
				if s, ok := t.(*ast.StarExpr); ok {
					t = s.X
				}
				if id, ok := t.(*ast.Ident); ok {
					name = id.Name + "." + name
				}
			}
			decls[name], docs[name] = d, d.Doc
		case *ast.GenDecl:
			for _, s := range d.Specs {
				switch s := s.(type) {
				case *ast.TypeSpec:
					decls[s.Name.Name], docs[s.Name.Name] = d, d.Doc
				case *ast.ValueSpec:
					for _, n := range s.Names {
						decls[n.Name], docs[n.Name] = d, d.Doc
					}
				}
			}
		}
	}
	var parts []string
	for _, name := range names {
		d, ok := decls[name]
		if !ok {
			return "", fmt.Errorf("%s has no declaration %s", file, name)
		}
		start := d.Pos()
		if doc := docs[name]; doc != nil {
			start = doc.Pos()
		}
		parts = append(parts, string(src[fset.Position(start).Offset:fset.Position(d.End()).Offset]))
	}
	return strings.Join(parts, "\n\n") + "\n", nil
}

// Stale is a listing that doesn't match its source.
type Stale struct {
	Doc  string
	Line int
	Want string
	Err  error
}

// Refresh checks every listing under root and, with write, rewrites the stale
// ones. It returns the listings that were (or, without write, are) stale.
func Refresh(root string, write bool) ([]Stale, error) {
	var stale []Stale
	for _, pattern := range Docs {
		docs, err := filepath.Glob(filepath.Join(root, pattern))
		if err != nil {
			return nil, err
		}
		for _, doc := range docs {
			md, err := os.ReadFile(doc)
			if err != nil {
				return nil, err
			}
			rel, _ := filepath.Rel(root, doc)
			var out bytes.Buffer
			pos := 0
			for _, b := range Find(md) {
				want, err := Extract(root, b.Source, b.Names)
				if err != nil {
					stale = append(stale, Stale{Doc: rel, Line: b.Line, Err: err})
					continue
				}
				if want == b.Body {
					continue
				}
				stale = append(stale, Stale{Doc: rel, Line: b.Line, Want: want})
				out.Write(md[pos:b.start])
				out.WriteString(want)
				pos = b.end
			}
			if write && pos > 0 {
				out.Write(md[pos:])
				if err := os.WriteFile(doc, out.Bytes(), 0o644); err != nil {
					return nil, err
				}
			}
		}
	}
	return stale, nil
}
