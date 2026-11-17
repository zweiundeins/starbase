package templgen

import (
	"fmt"
	"strconv"
	"strings"
)

// Templ turns usage markup into templ: each <sb-*> element becomes a call of
// its wrapper (qualified "sb."), with its content as children; the rest
// stays markup.
func (m *Model) Templ(markup string) (string, error) {
	nodes, err := ParseMarkup(markup)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	if err := m.writeTempl(&b, nodes, ""); err != nil {
		return "", err
	}
	return strings.TrimSuffix(b.String(), "\n"), nil
}

func (m *Model) writeTempl(b *strings.Builder, nodes []*Node, indent string) error {
	for _, n := range nodes {
		switch {
		case n.Comment:
			b.WriteString(indent + n.Text + "\n")
		case n.Tag == "":
			if t := templText(n.Text); t != "" {
				for _, line := range strings.Split(t, "\n") {
					b.WriteString(indent + line + "\n")
				}
			}
		case m.isComponent(n.Tag):
			c, _ := m.Get(n.Tag)
			lit, err := c.PropsLiteral(n, "sb.")
			if err != nil {
				return err
			}
			call := "@sb." + c.GoName + "(" + strings.ReplaceAll(lit, "\n", "\n"+indent) + ")"
			if !hasContent(n.Children) {
				b.WriteString(indent + call + "\n")
				continue
			}
			b.WriteString(indent + call + " {\n")
			if err := m.writeTempl(b, n.Children, indent+"\t"); err != nil {
				return err
			}
			b.WriteString(indent + "}\n")
		default:
			open := n.openTag(false)
			switch {
			case n.SelfEnd || voidTags[n.Tag]:
				b.WriteString(indent + open + "\n")
			case rawTags[n.Tag]:
				b.WriteString(indent + open + n.Text + "</" + n.Tag + ">\n")
			case !hasContent(n.Children):
				b.WriteString(indent + open + "</" + n.Tag + ">\n")
			case len(n.Children) == 1 && n.Children[0].Tag == "" && !n.Children[0].Comment && !strings.Contains(strings.TrimSpace(n.Children[0].Text), "\n"):
				b.WriteString(indent + open + templText(n.Children[0].Text) + "</" + n.Tag + ">\n")
			default:
				b.WriteString(indent + open + "\n")
				if err := m.writeTempl(b, n.Children, indent+"\t"); err != nil {
					return err
				}
				b.WriteString(indent + "</" + n.Tag + ">\n")
			}
		}
	}
	return nil
}

func (m *Model) isComponent(tag string) bool {
	_, ok := m.Get(tag)
	return ok
}

func hasContent(nodes []*Node) bool {
	for _, n := range nodes {
		if n.Tag != "" || n.Comment || strings.TrimSpace(n.Text) != "" {
			return true
		}
	}
	return false
}

// templText is text as templ markup: its lines trimmed, and text with
// braces as a Go string (templ reads { as the start of an expression).
func templText(s string) string {
	var lines []string
	for _, l := range strings.Split(s, "\n") {
		if l = strings.TrimSpace(l); l != "" {
			lines = append(lines, l)
		}
	}
	t := strings.Join(lines, "\n")
	if strings.ContainsAny(t, "{}") {
		return "{ " + strconv.Quote(strings.Join(lines, " ")) + " }"
	}
	return t
}

// GoExpr turns usage markup into a Go expression of type templ.Component
// that renders it (for package sb's own tests): wrappers for the <sb-*>
// elements, given their content with withChildren, and templ.Raw for the
// rest, kept byte for byte.
func (m *Model) GoExpr(markup string) (string, error) {
	nodes, err := ParseMarkup(markup)
	if err != nil {
		return "", err
	}
	parts, err := m.goParts(nodes)
	if err != nil {
		return "", err
	}
	return join(parts), nil
}

func join(parts []string) string {
	if len(parts) == 1 {
		return parts[0]
	}
	return "templ.Join(\n" + strings.Join(parts, ",\n") + ",\n)"
}

func (m *Model) goParts(nodes []*Node) ([]string, error) {
	var parts []string
	var raw strings.Builder
	flush := func() {
		if raw.Len() > 0 {
			parts = append(parts, "templ.Raw("+strconv.Quote(raw.String())+")")
			raw.Reset()
		}
	}
	for _, n := range nodes {
		switch {
		case n.Tag == "" || n.Comment || !m.hasComponent(n):
			raw.WriteString(n.HTML())
		case m.isComponent(n.Tag):
			flush()
			c, _ := m.Get(n.Tag)
			lit, err := c.PropsLiteral(n, "")
			if err != nil {
				return nil, err
			}
			call := c.GoName + "(" + lit + ")"
			if len(n.Children) > 0 {
				kids, err := m.goParts(n.Children)
				if err != nil {
					return nil, err
				}
				call = "withChildren(" + call + ",\n" + strings.Join(kids, ",\n") + ",\n)"
			}
			parts = append(parts, call)
		default: // markup with a component inside
			raw.WriteString(n.openTag(true))
			flush()
			kids, err := m.goParts(n.Children)
			if err != nil {
				return nil, err
			}
			parts = append(parts, kids...)
			raw.WriteString("</" + n.Tag + ">")
		}
	}
	flush()
	return parts, nil
}

// hasComponent reports whether n is or holds an <sb-*> component.
func (m *Model) hasComponent(n *Node) bool {
	if m.isComponent(n.Tag) {
		return true
	}
	for _, c := range n.Children {
		if m.hasComponent(c) {
			return true
		}
	}
	return false
}

// usageTests writes a test case per component: its install markup (the
// README's usage, or its preview) as HTML and as the Go that renders it
// through the wrappers, which TestUsageMatchesMarkup compares.
func (m *Model) usageTests() ([]byte, error) {
	var b strings.Builder
	b.WriteString(Header + "package sb\n\nimport \"github.com/a-h/templ\"\n\n")
	b.WriteString("// usages are every component's install markup, as HTML and through the\n// wrappers.\nvar usages = []usage{\n")
	for _, c := range m.Components {
		markup := c.InstallMarkup()
		expr, err := m.GoExpr(markup)
		if err != nil {
			return nil, fmt.Errorf("%s usage: %w", c.Slug, err)
		}
		fmt.Fprintf(&b, "{\nslug: %q,\nhtml: %s,\nrender: %s,\n},\n", c.Slug, goString(markup), expr)
	}
	b.WriteString("}\n")
	return gofmt([]byte(b.String()))
}

// goString writes s as a raw string literal when it can be one.
func goString(s string) string {
	if !strings.Contains(s, "`") && !strings.Contains(s, "\r") {
		return "`" + s + "`"
	}
	return strconv.Quote(s)
}

// Uses returns the components markup uses, in order of first appearance.
func (m *Model) Uses(markup string) ([]*Component, error) {
	nodes, err := ParseMarkup(markup)
	if err != nil {
		return nil, err
	}
	var out []*Component
	seen := map[string]bool{}
	var walk func([]*Node)
	walk = func(nodes []*Node) {
		for _, n := range nodes {
			if c, ok := m.Get(n.Tag); ok && !seen[n.Tag] {
				seen[n.Tag] = true
				out = append(out, c)
			}
			walk(n.Children)
		}
	}
	walk(nodes)
	return out, nil
}
