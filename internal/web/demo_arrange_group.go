package web

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
)

func init() {
	arrangers["drag-group"] = arranger{arrange: arrangeDragGroup, render: renderDragGroup}
	arrangers["drag-group-card"] = arranger{arrange: arrangeDragGroup, render: renderDragGroupCard}
}

// groupLists are the demo's lists, by the name its state uses.
var groupLists = map[string]string{"planets": "Planets", "dwarfs": "Dwarf planets"}

// groupState is a drag group's arrangement: lists of bodies, in order. Its
// words are name=id,id… (an empty list is name=).
type groupState struct {
	names []string
	items map[string][]string
}

func parseGroup(state string) (groupState, error) {
	g := groupState{items: map[string][]string{}}
	var all []string
	for _, word := range strings.Fields(state) {
		name, list, found := strings.Cut(word, "=")
		if _, ok := groupLists[name]; !ok || !found || slices.Contains(g.names, name) {
			return g, fmt.Errorf("%q is not a list of the demo", name)
		}
		g.names = append(g.names, name)
		if list != "" {
			g.items[name] = strings.Split(list, ",")
			all = append(all, g.items[name]...)
		}
	}
	if _, err := ids(strings.Join(all, " ")); err != nil || len(g.names) == 0 || slices.Contains(all, "") {
		return g, fmt.Errorf("%q is not an arrangement of bodies", state)
	}
	return g, nil
}

func (g groupState) String() string {
	words := make([]string, len(g.names))
	for i, name := range g.names {
		words[i] = name + "=" + strings.Join(g.items[name], ",")
	}
	return strings.Join(words, " ")
}

// arrangeDragGroup applies an sb-drag-group-move ({itemId, fromList,
// toList, before}) to lists of bodies.
func arrangeDragGroup(state string, move json.RawMessage) (string, error) {
	g, err := parseGroup(state)
	if err != nil {
		return "", err
	}
	var m struct {
		ItemID   string `json:"itemId"`
		FromList string `json:"fromList"`
		ToList   string `json:"toList"`
		Before   string `json:"before"`
	}
	if err := json.Unmarshal(move, &m); err != nil {
		return "", err
	}
	from, to := g.items[m.FromList], g.items[m.ToList]
	if !slices.Contains(g.names, m.ToList) || !slices.Contains(from, m.ItemID) {
		return "", fmt.Errorf("%q is not in %q, or %q is not a list", m.ItemID, m.FromList, m.ToList)
	}
	if m.FromList != m.ToList {
		g.items[m.FromList] = slices.DeleteFunc(slices.Clone(from), func(id string) bool { return id == m.ItemID })
		to = append(slices.Clone(to), m.ItemID)
	}
	if g.items[m.ToList], err = moveBefore(to, m.ItemID, m.Before); err != nil {
		return "", err
	}
	return g.String(), nil
}

// renderDragGroup is the group's markup: the host the morph replaces, with
// the arrangement in data-state, and a list per name with an item per body.
// Item ids make the morph move an item, focus and all; tabindex="-1" lets
// the arrow keys focus an empty list.
func renderDragGroup(id, state string) string {
	g, _ := parseGroup(state)
	var b strings.Builder
	fmt.Fprintf(&b, "<sb-drag-group id=\"%s\" data-ignore-morph class=\"demo-group\" data-state=\"%s\"\n\tdata-on:sb-drag-group-move=\"%s\">\n", id, state, arrangeOn("drag-group"))
	fmt.Fprintf(&b, "\t<p id=\"%s-keys\" hidden>To move it, hold Alt and press the arrow keys.</p>\n", id)
	for _, name := range g.names {
		fmt.Fprintf(&b, "\t<section data-drop-list=\"%s\" tabindex=\"-1\" aria-label=\"%s\">\n\t\t<span class=\"demo-group__title\">%s</span>\n", name, groupLists[name], groupLists[name])
		for _, item := range g.items[name] {
			fmt.Fprintf(&b, "\t\t<div id=\"%s-%s\" data-drag-item=\"%s\" tabindex=\"0\" aria-describedby=\"%s-keys\">%s</div>\n", id, item, item, id, label(item))
		}
		b.WriteString("\t</section>\n")
	}
	b.WriteString("</sb-drag-group>")
	return b.String()
}

// renderDragGroupCard is the gallery card's group: the lists without titles,
// and items without tab stops, since the gallery is a page of cards.
func renderDragGroupCard(id, state string) string {
	g, _ := parseGroup(state)
	var b strings.Builder
	fmt.Fprintf(&b, "<sb-drag-group id=\"%s\" data-ignore-morph class=\"demo-group-card\" data-state=\"%s\"\n\tdata-on:sb-drag-group-move=\"%s\">\n", id, state, arrangeOn("drag-group-card"))
	for _, name := range g.names {
		fmt.Fprintf(&b, "\t<section data-drop-list=\"%s\">\n", name)
		for _, item := range g.items[name] {
			fmt.Fprintf(&b, "\t\t<div data-drag-item=\"%s\">%s</div>\n", item, label(item))
		}
		b.WriteString("\t</section>\n")
	}
	b.WriteString("</sb-drag-group>")
	return b.String()
}
