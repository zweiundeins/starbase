package web

import (
	"encoding/json"
	"fmt"
	"html"
	"slices"
	"strings"
)

func init() {
	arrangers["context-menu"] = arranger{arrange: arrangeContextMenu, render: renderContextMenu}
}

// menuLists reads the context menu demo's state: the bodies in their order,
// then the removed ones, each with a "-" before its id. A "*" before a body
// marks the one whose button takes the focus after the answer.
func menuLists(state string) (shown, removed []string, focus string, err error) {
	var all []string
	for _, w := range strings.Fields(state) {
		id, gone := strings.CutPrefix(w, "-")
		if f, ok := strings.CutPrefix(w, "*"); ok && focus == "" {
			id, focus = f, f
		}
		all = append(all, id)
		switch {
		case gone:
			removed = append(removed, id)
		case len(removed) > 0:
			return nil, nil, "", fmt.Errorf("%q: the removed bodies come last", state)
		default:
			shown = append(shown, id)
		}
	}
	if _, err := ids(strings.Join(all, " ")); err != nil || len(shown) == 0 {
		return nil, nil, "", fmt.Errorf("%q is not a list of bodies", state)
	}
	return shown, removed, focus, nil
}

// arrangeContextMenu applies an sb-menu-action ({action, contextId}) to the
// list: the menu's actions, and restore, which the page's button sends. A
// removal or a restore takes away the focused button, so the new state marks
// the body whose button takes the focus instead.
func arrangeContextMenu(state string, move json.RawMessage) (string, error) {
	shown, removed, _, err := menuLists(state)
	if err != nil {
		return "", err
	}
	var m struct {
		Action    string `json:"action"`
		ContextID string `json:"contextId"`
	}
	if err := json.Unmarshal(move, &m); err != nil {
		return "", err
	}
	i := slices.Index(shown, m.ContextID)
	if i < 0 && m.Action != "restore" {
		return "", fmt.Errorf("%q is not in the list", m.ContextID)
	}
	at := func(j int) string { // the body at j, or "" past the end
		if j < len(shown) {
			return shown[j]
		}
		return ""
	}
	focus := ""
	switch m.Action {
	case "top":
		if i > 0 {
			shown, err = moveBefore(shown, m.ContextID, shown[0])
		}
	case "bottom":
		if i < len(shown)-1 {
			shown, err = moveBefore(shown, m.ContextID, "")
		}
	case "up":
		if i > 0 {
			shown, err = moveBefore(shown, m.ContextID, shown[i-1])
		}
	case "down":
		if i < len(shown)-1 {
			shown, err = moveBefore(shown, m.ContextID, at(i+2))
		}
	case "remove":
		if len(shown) == 1 {
			return "", fmt.Errorf("the last body stays")
		}
		shown, removed = slices.Delete(shown, i, i+1), append(removed, m.ContextID)
		focus = shown[min(i, len(shown)-1)] // the body that took its place
	case "restore":
		if len(removed) == 0 {
			return "", fmt.Errorf("no body to bring back")
		}
		shown, removed, focus = append(shown, removed...), nil, removed[0]
	default:
		return "", fmt.Errorf("no action %q", m.Action)
	}
	if err != nil {
		return "", err
	}
	words := make([]string, 0, len(shown)+len(removed))
	for _, id := range shown {
		if id == focus {
			id = "*" + id
		}
		words = append(words, id)
	}
	for _, id := range removed {
		words = append(words, "-"+id)
	}
	return strings.Join(words, " "), nil
}

// renderContextMenu is the demo's markup: the list, a row per body that the
// menu opens for (a right click on the row, or its button), with its name
// for the menu's labels and the actions that don't apply to it, the menu's
// template, and a button that brings removed bodies back. The menu ignores
// morphs and the buttons keep what it sets on them (their anchor and
// aria-expanded), so an answer that arrives while it is open leaves it alone.
func renderContextMenu(id, state string) string {
	shown, removed, focus, _ := menuLists(state)
	menu := id + "-menu"
	var b strings.Builder
	fmt.Fprintf(&b, "<div id=\"%s\" class=\"demo-menu\" data-state=\"%s\"\n\tdata-on:sb-menu-action=\"%s\">\n\t<ul aria-label=\"Planets\">\n", id, state, arrangeOn("context-menu"))
	for k, item := range shown {
		name := html.EscapeString(bodies()[item].Name)
		var off []string
		if k == 0 {
			off = append(off, "top", "up")
		}
		if k == len(shown)-1 {
			off = append(off, "bottom", "down")
		}
		if len(shown) == 1 {
			off = append(off, "remove")
		}
		fmt.Fprintf(&b, "\t\t<li id=\"%s-%s\" tabindex=\"-1\" data-context-id=\"%s\" data-menu-for=\"%s\" data-menu-param-name=\"%s\"", id, item, item, menu, name)
		if len(off) > 0 {
			fmt.Fprintf(&b, " data-menu-disabled=\"%s\"", strings.Join(off, " "))
		}
		fmt.Fprintf(&b, ">\n\t\t\t<span>%s</span>\n", label(item))
		fmt.Fprintf(&b, "\t\t\t<button type=\"button\" data-menu-for=\"%s\" aria-haspopup=\"menu\" aria-controls=\"%s\" aria-label=\"Actions for %s\" data-preserve-attr=\"style aria-expanded\"", menu, menu, name)
		if item == focus { // only where the answer took the focus away
			gone := "" // naming the removed row makes each removal's data-init new, so it runs on the same button again
			if len(removed) > 0 {
				gone = fmt.Sprintf("!document.getElementById('%s-%s') && ", id, removed[len(removed)-1])
			}
			fmt.Fprintf(&b, "\n\t\t\t\tdata-init=\"%sdocument.activeElement === document.body && el.focus()\"", gone)
		}
		b.WriteString(">…</button>\n\t\t</li>\n")
	}
	b.WriteString("\t</ul>\n")
	if len(removed) > 0 {
		names := make([]string, len(removed))
		for i, r := range removed {
			names[i] = html.EscapeString(bodies()[r].Name)
		}
		fmt.Fprintf(&b, "\t<button type=\"button\" class=\"demo-menu__restore\"\n\t\tdata-on:click=\"@get('/demo/arrange/context-menu', {payload: {id: '%s', state: el.parentElement.dataset.state, move: {action: 'restore', contextId: ''}}})\">Bring back %s</button>\n", id, strings.Join(names, ", "))
	}
	fmt.Fprintf(&b, "\t<sb-context-menu id=\"%s\" aria-label=\"Planet actions\" data-ignore-morph>\n\t\t<template data-sb-menu>\n", menu)
	for _, a := range [][2]string{{"top", "Move to the top"}, {"bottom", "Move to the bottom"}} {
		fmt.Fprintf(&b, "\t\t\t<button type=\"button\" role=\"menuitem\" data-action=\"%s\">%s</button>\n", a[0], a[1])
	}
	fmt.Fprintf(&b, "\t\t\t<button type=\"button\" role=\"menuitem\" data-submenu=\"%s-step\" aria-haspopup=\"menu\" aria-expanded=\"false\">Move one place</button>\n", id)
	fmt.Fprintf(&b, "\t\t\t<div id=\"%s-step\" role=\"menu\" aria-label=\"Move one place\">\n", id)
	b.WriteString("\t\t\t\t<button type=\"button\" role=\"menuitem\" data-action=\"up\">Up</button>\n")
	b.WriteString("\t\t\t\t<button type=\"button\" role=\"menuitem\" data-action=\"down\">Down</button>\n\t\t\t</div>\n")
	b.WriteString("\t\t\t<div role=\"separator\"></div>\n")
	b.WriteString("\t\t\t<button type=\"button\" role=\"menuitem\" data-action=\"remove\">Remove {name}</button>\n")
	b.WriteString("\t\t</template>\n\t</sb-context-menu>\n</div>")
	return b.String()
}
