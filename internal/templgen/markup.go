package templgen

import (
	"fmt"
	"html"
	"strings"
)

// Node is a piece of usage markup: an element, text, or a comment.
type Node struct {
	Tag      string // "" for text and comments
	Attrs    []Attr
	Children []*Node
	Text     string // text (raw, entities kept), a comment's whole source, or a raw-text element's content
	Comment  bool
	SelfEnd  bool // written <x … /> (SVG)
}

// Attr is an attribute as written: Value is unescaped, Bare has no "=".
type Attr struct {
	Name, Value string
	Bare        bool
}

var voidTags = map[string]bool{"area": true, "base": true, "br": true, "col": true, "embed": true, "hr": true, "img": true, "input": true, "link": true, "meta": true, "source": true, "track": true, "wbr": true}

var rawTags = map[string]bool{"script": true, "style": true, "textarea": true, "title": true}

// ParseMarkup parses the hand-written markup of a component's usage: well
// formed, every element closed except void ones.
func ParseMarkup(src string) ([]*Node, error) {
	p := &parser{src: src}
	root := &Node{Tag: "#root"}
	if err := p.children(root); err != nil {
		return nil, err
	}
	return root.Children, nil
}

type parser struct {
	src string
	i   int
}

func (p *parser) errf(format string, args ...any) error {
	line := strings.Count(p.src[:p.i], "\n") + 1
	return fmt.Errorf("markup line %d: %s", line, fmt.Sprintf(format, args...))
}

func (p *parser) children(parent *Node) error {
	for p.i < len(p.src) {
		rest := p.src[p.i:]
		switch {
		case strings.HasPrefix(rest, "<!--"):
			end := strings.Index(rest, "-->")
			if end < 0 {
				return p.errf("unclosed comment")
			}
			parent.Children = append(parent.Children, &Node{Comment: true, Text: rest[:end+3]})
			p.i += end + 3
		case strings.HasPrefix(rest, "</"):
			end := strings.IndexByte(rest, '>')
			if end < 0 {
				return p.errf("unclosed end tag")
			}
			name := strings.ToLower(strings.TrimSpace(rest[2:end]))
			if name != parent.Tag {
				return p.errf("</%s> closes <%s>", name, parent.Tag)
			}
			p.i += end + 1
			return nil
		case strings.HasPrefix(rest, "<") && len(rest) > 1 && isNameStart(rest[1]):
			n, err := p.element()
			if err != nil {
				return err
			}
			parent.Children = append(parent.Children, n)
		default:
			end := strings.IndexByte(rest[1:], '<')
			if end < 0 {
				end = len(rest)
			} else {
				end++
			}
			parent.Children = append(parent.Children, &Node{Text: rest[:end]})
			p.i += end
		}
	}
	if parent.Tag != "#root" {
		return p.errf("<%s> is not closed", parent.Tag)
	}
	return nil
}

func isNameStart(c byte) bool { return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' }

func (p *parser) element() (*Node, error) {
	p.i++ // <
	start := p.i
	for p.i < len(p.src) && !isSpace(p.src[p.i]) && p.src[p.i] != '>' && p.src[p.i] != '/' {
		p.i++
	}
	n := &Node{Tag: strings.ToLower(p.src[start:p.i])}
	for {
		p.skipSpace()
		if p.i >= len(p.src) {
			return nil, p.errf("<%s> is not closed", n.Tag)
		}
		if p.src[p.i] == '>' {
			p.i++
			break
		}
		if strings.HasPrefix(p.src[p.i:], "/>") {
			p.i += 2
			n.SelfEnd = true
			return n, nil
		}
		a, err := p.attr()
		if err != nil {
			return nil, err
		}
		n.Attrs = append(n.Attrs, a)
	}
	if voidTags[n.Tag] {
		return n, nil
	}
	if rawTags[n.Tag] {
		end := strings.Index(strings.ToLower(p.src[p.i:]), "</"+n.Tag)
		if end < 0 {
			return nil, p.errf("<%s> is not closed", n.Tag)
		}
		n.Text = p.src[p.i : p.i+end]
		p.i += end
		close := strings.IndexByte(p.src[p.i:], '>')
		p.i += close + 1
		return n, nil
	}
	return n, p.children(n)
}

func (p *parser) attr() (Attr, error) {
	start := p.i
	for p.i < len(p.src) && !isSpace(p.src[p.i]) && p.src[p.i] != '=' && p.src[p.i] != '>' && !strings.HasPrefix(p.src[p.i:], "/>") {
		p.i++
	}
	a := Attr{Name: p.src[start:p.i]}
	if a.Name == "" {
		return a, p.errf("bad attribute")
	}
	p.skipSpace()
	if p.i >= len(p.src) || p.src[p.i] != '=' {
		a.Bare = true
		return a, nil
	}
	p.i++
	p.skipSpace()
	if p.i < len(p.src) && (p.src[p.i] == '"' || p.src[p.i] == '\'') {
		q := p.src[p.i]
		end := strings.IndexByte(p.src[p.i+1:], q)
		if end < 0 {
			return a, p.errf("unclosed value of %s", a.Name)
		}
		a.Value = html.UnescapeString(p.src[p.i+1 : p.i+1+end])
		p.i += end + 2
		return a, nil
	}
	start = p.i
	for p.i < len(p.src) && !isSpace(p.src[p.i]) && p.src[p.i] != '>' {
		p.i++
	}
	a.Value = html.UnescapeString(p.src[start:p.i])
	return a, nil
}

func (p *parser) skipSpace() {
	for p.i < len(p.src) && isSpace(p.src[p.i]) {
		p.i++
	}
}

func isSpace(c byte) bool { return c == ' ' || c == '\t' || c == '\n' || c == '\r' || c == '\f' }

// HTML writes a node back as HTML.
func (n *Node) HTML() string {
	var b strings.Builder
	n.writeHTML(&b)
	return b.String()
}

func (n *Node) writeHTML(b *strings.Builder) {
	switch {
	case n.Comment || n.Tag == "":
		b.WriteString(n.Text)
		return
	}
	b.WriteString(n.openTag(true))
	if n.SelfEnd || voidTags[n.Tag] {
		return
	}
	if rawTags[n.Tag] {
		b.WriteString(n.Text)
	}
	for _, c := range n.Children {
		c.writeHTML(b)
	}
	b.WriteString("</" + n.Tag + ">")
}

// openTag writes <tag attrs> (or <tag attrs/>), each value in double
// quotes, or in single quotes when it has double ones and no single ones;
// amp escapes & (HTML), templ's constant attributes are written as they are.
func (n *Node) openTag(amp bool) string {
	var b strings.Builder
	b.WriteString("<" + n.Tag)
	for _, a := range n.Attrs {
		b.WriteString(" " + a.Name)
		if !a.Bare {
			b.WriteString("=" + quoteAttr(a.Value, amp))
		}
	}
	if n.SelfEnd {
		b.WriteString("/")
	}
	b.WriteString(">")
	return b.String()
}

func quoteAttr(v string, amp bool) string {
	if amp {
		v = strings.ReplaceAll(v, "&", "&amp;")
	}
	if strings.Contains(v, `"`) && !strings.Contains(v, "'") {
		return "'" + v + "'"
	}
	return `"` + strings.ReplaceAll(v, `"`, "&quot;") + `"`
}
