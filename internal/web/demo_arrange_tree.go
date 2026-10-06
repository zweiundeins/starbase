package web

import (
	"encoding/json"
	"fmt"
	"html"
	"strings"
)

func init() {
	arrangers["sortable-tree"] = arranger{arrange: arrangeSortableTree, render: renderSortableTree}
	arrangers["sortable-tree-card"] = arranger{arrange: arrangeSortableTree, render: renderSortableTreeCard}
}

// A treeNode is a body in the tree: a folder (with children, maybe none)
// or a file.
type treeNode struct {
	id       string
	folder   bool
	children []*treeNode
}

// parseTree reads a tree's state: ids separated by spaces, a folder's
// children in parentheses after it, as in "earth(moon) mars(phobos deimos) ceres()".
func parseTree(state string) ([]*treeNode, error) {
	tokens := strings.Fields(strings.NewReplacer("(", " ( ", ")", " ) ").Replace(state))
	var flat []string
	for _, t := range tokens {
		if t != "(" && t != ")" {
			flat = append(flat, t)
		}
	}
	if _, err := ids(strings.Join(flat, " ")); err != nil {
		return nil, err
	}
	pos := 0
	var list func(depth int) ([]*treeNode, error)
	list = func(depth int) ([]*treeNode, error) {
		var out []*treeNode
		for pos < len(tokens) && tokens[pos] != ")" {
			if tokens[pos] == "(" {
				return nil, fmt.Errorf("%q: a folder needs a name", state)
			}
			n := &treeNode{id: tokens[pos]}
			pos++
			if pos < len(tokens) && tokens[pos] == "(" {
				if depth == 8 {
					return nil, fmt.Errorf("%q is too deep", state)
				}
				pos++
				children, err := list(depth + 1)
				if err != nil {
					return nil, err
				}
				if pos == len(tokens) {
					return nil, fmt.Errorf("%q: a folder isn't closed", state)
				}
				pos++ // ")"
				n.folder, n.children = true, children
			}
			out = append(out, n)
		}
		return out, nil
	}
	root, err := list(0)
	if err == nil && pos < len(tokens) {
		err = fmt.Errorf("%q: a folder is closed twice", state)
	}
	return root, err
}

// formatTree writes a tree back as its state.
func formatTree(list []*treeNode) string {
	words := make([]string, len(list))
	for i, n := range list {
		words[i] = n.id
		if n.folder {
			words[i] += "(" + formatTree(n.children) + ")"
		}
	}
	return strings.Join(words, " ")
}

// arrangeSortableTree applies an sb-tree-move ({itemId, fromParent,
// toParent, before}; "" is the top level) to the tree in a state.
func arrangeSortableTree(state string, move json.RawMessage) (string, error) {
	root, err := parseTree(state)
	if err != nil {
		return "", err
	}
	var m struct {
		ItemID     string `json:"itemId"`
		FromParent string `json:"fromParent"`
		ToParent   string `json:"toParent"`
		Before     string `json:"before"`
	}
	if err := json.Unmarshal(move, &m); err != nil {
		return "", err
	}
	// Every node's list and the folder it is in.
	lists := map[string]*[]*treeNode{"": &root}
	parent := map[string]string{}
	var item *treeNode
	var walk func(folder string, list []*treeNode)
	walk = func(folder string, list []*treeNode) {
		for _, n := range list {
			parent[n.id] = folder
			if n.id == m.ItemID {
				item = n
			}
			if n.folder {
				lists[n.id] = &n.children
				walk(n.id, n.children)
			}
		}
	}
	walk("", root)
	to, ok := lists[m.ToParent]
	switch {
	case item == nil || parent[m.ItemID] != m.FromParent:
		return "", fmt.Errorf("%q is not in %q", m.ItemID, m.FromParent)
	case !ok:
		return "", fmt.Errorf("%q is not a folder", m.ToParent)
	}
	for f := m.ToParent; f != ""; f = parent[f] {
		if f == m.ItemID {
			return "", fmt.Errorf("%q can't go into itself", m.ItemID)
		}
	}
	// Out of its folder, at the end of the other, then before its new neighbour.
	from := lists[m.FromParent]
	*from = remove(*from, item)
	order := make([]string, 0, len(*to)+1)
	byID := map[string]*treeNode{item.id: item}
	for _, n := range *to {
		order = append(order, n.id)
		byID[n.id] = n
	}
	if order, err = moveBefore(append(order, item.id), item.id, m.Before); err != nil {
		return "", err
	}
	*to = (*to)[:0]
	for _, id := range order {
		*to = append(*to, byID[id])
	}
	state = formatTree(root)
	if _, err := parseTree(state); err != nil {
		return "", fmt.Errorf("%q in %q nests the tree too deep", m.ItemID, m.ToParent)
	}
	return state, nil
}

func remove(list []*treeNode, n *treeNode) []*treeNode {
	out := make([]*treeNode, 0, len(list))
	for _, c := range list {
		if c != n {
			out = append(out, c)
		}
	}
	return out
}

// renderSortableTree is the tree's markup: the host the morph replaces,
// with the tree in data-state, folders open, and a row per body. Each node
// has an id, so the morph moves it rather than rewriting the node in its
// place, and the component sees the move (it closes its closed folders
// again when the tree's children change).
func renderSortableTree(id, state string) string {
	root, _ := parseTree(state)
	var b strings.Builder
	fmt.Fprintf(&b, "<sb-sortable-tree id=\"%s\" class=\"demo-tree\" data-state=\"%s\"\n\tdata-on:sb-tree-move=\"%s\">\n", id, state, arrangeOn("sortable-tree"))
	b.WriteString("\t<div data-tree-children data-tree-parent=\"\" aria-label=\"Solar System\">\n")
	var nodes func(list []*treeNode, indent string)
	nodes = func(list []*treeNode, indent string) {
		for _, n := range list {
			kind, expanded := "file", ""
			if n.folder {
				kind, expanded = "folder", ` aria-expanded="true"`
			}
			name := html.EscapeString(bodies()[n.id].Name)
			fmt.Fprintf(&b, "%s<div id=\"%s-%s\" data-tree-node=\"%s\" data-tree-kind=\"%s\">\n", indent, id, n.id, n.id, kind)
			fmt.Fprintf(&b, "%s\t<div data-tree-row tabindex=\"0\" aria-label=\"%s: %s\"%s>%s</div>\n", indent, kind, name, expanded, label(n.id))
			if n.folder {
				fmt.Fprintf(&b, "%s\t<div data-tree-children data-tree-parent=\"%s\">\n", indent, n.id)
				nodes(n.children, indent+"\t\t")
				fmt.Fprintf(&b, "%s\t</div>\n", indent)
			}
			fmt.Fprintf(&b, "%s</div>\n", indent)
		}
	}
	nodes(root, "\t\t")
	b.WriteString("\t</div>\n</sb-sortable-tree>")
	return b.String()
}

// renderSortableTreeCard is the gallery card's tree: without labels, and
// with one tab stop (the arrow keys reach the other rows), since the gallery
// is a page of cards.
func renderSortableTreeCard(id, state string) string {
	root, _ := parseTree(state)
	var b strings.Builder
	fmt.Fprintf(&b, "<sb-sortable-tree id=\"%s\" class=\"demo-tree-card\" data-state=\"%s\"\n\tdata-on:sb-tree-move=\"%s\">\n", id, state, arrangeOn("sortable-tree-card"))
	tabindex := "0"
	var nodes func(list []*treeNode, parent, indent string)
	nodes = func(list []*treeNode, parent, indent string) {
		fmt.Fprintf(&b, "%s<div data-tree-children data-tree-parent=\"%s\">\n", indent, parent)
		for _, n := range list {
			kind, expanded := "file", ""
			if n.folder {
				kind, expanded = "folder", ` aria-expanded="true"`
			}
			row := fmt.Sprintf("<div data-tree-row tabindex=\"%s\"%s>%s</div>", tabindex, expanded, label(n.id))
			tabindex = "-1"
			fmt.Fprintf(&b, "%s\t<div id=\"%s-%s\" data-tree-node=\"%s\" data-tree-kind=\"%s\">", indent, id, n.id, n.id, kind)
			if !n.folder {
				b.WriteString(row + "</div>\n")
				continue
			}
			fmt.Fprintf(&b, "\n%s\t\t%s\n", indent, row)
			nodes(n.children, n.id, indent+"\t\t")
			fmt.Fprintf(&b, "%s\t</div>\n", indent)
		}
		fmt.Fprintf(&b, "%s</div>\n", indent)
	}
	nodes(root, "", "\t")
	b.WriteString("</sb-sortable-tree>")
	return b.String()
}
