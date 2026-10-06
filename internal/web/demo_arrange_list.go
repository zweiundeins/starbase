package web

import (
	"encoding/json"
	"fmt"
	"strings"
)

func init() {
	arrangers["sortable-list"] = arranger{arrange: arrangeSortableList, render: renderSortableList}
}

// arrangeSortableList applies an sb-sortable-move ({itemId, before}) to a
// list of bodies.
func arrangeSortableList(state string, move json.RawMessage) (string, error) {
	list, err := ids(state)
	if err != nil {
		return "", err
	}
	var m struct {
		ItemID string `json:"itemId"`
		Before string `json:"before"`
	}
	if err := json.Unmarshal(move, &m); err != nil {
		return "", err
	}
	if list, err = moveBefore(list, m.ItemID, m.Before); err != nil {
		return "", err
	}
	return strings.Join(list, " "), nil
}

// renderSortableList is the list's markup: the host the morph replaces,
// with the arrangement in data-state and an item per body.
func renderSortableList(id, state string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "<sb-sortable-list id=\"%s\" class=\"demo-sortable\" data-state=\"%s\"\n\tdata-on:sb-sortable-move=\"%s\">\n", id, state, arrangeOn("sortable-list"))
	for _, item := range strings.Fields(state) {
		fmt.Fprintf(&b, "\t<div data-sortable-item=\"%s\" tabindex=\"0\">%s</div>\n", item, label(item))
	}
	b.WriteString("</sb-sortable-list>")
	return b.String()
}
