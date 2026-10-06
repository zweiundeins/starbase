package web

import (
	"strings"
	"testing"
)

func TestArrangeContextMenu(t *testing.T) {
	for _, tc := range []struct {
		state, move, want string
	}{
		{"mercury venus earth mars", `{"action":"top","contextId":"earth"}`, "earth mercury venus mars"},
		{"mercury venus earth mars", `{"action":"bottom","contextId":"venus"}`, "mercury earth mars venus"},
		{"mercury venus earth mars", `{"action":"up","contextId":"mars"}`, "mercury venus mars earth"},
		{"mercury venus earth mars", `{"action":"down","contextId":"mercury"}`, "venus mercury earth mars"},
		// Moves that don't apply leave the list as it is.
		{"mercury venus earth mars", `{"action":"top","contextId":"mercury"}`, "mercury venus earth mars"},
		{"mercury venus earth mars", `{"action":"up","contextId":"mercury"}`, "mercury venus earth mars"},
		{"mercury venus earth mars", `{"action":"bottom","contextId":"mars"}`, "mercury venus earth mars"},
		{"mercury venus earth mars", `{"action":"down","contextId":"mars"}`, "mercury venus earth mars"},
		// A removal marks the body that took its place, or the new last one.
		{"mercury venus earth mars", `{"action":"remove","contextId":"venus"}`, "mercury *earth mars -venus"},
		{"mercury venus earth mars", `{"action":"remove","contextId":"mars"}`, "mercury venus *earth -mars"},
		{"mercury venus *earth -mars", `{"action":"remove","contextId":"mercury"}`, "*venus earth -mars -mercury"},
		// A restore marks the first body it brings back; the old mark goes.
		{"*venus earth -mars -mercury", `{"action":"restore","contextId":""}`, "venus earth *mars mercury"},
		{"mercury *venus earth", `{"action":"top","contextId":"earth"}`, "earth mercury venus"},
	} {
		got, err := arrangeContextMenu(tc.state, []byte(tc.move))
		if err != nil || got != tc.want {
			t.Errorf("%s with %s = %q, %v; want %q", tc.state, tc.move, got, err, tc.want)
		}
	}
}

func TestArrangeContextMenuRefuses(t *testing.T) {
	for _, tc := range []struct{ state, move string }{
		// Bad states.
		{"", `{"action":"top","contextId":"earth"}`},
		{"earth vulcan", `{"action":"top","contextId":"earth"}`},
		{"earth earth", `{"action":"top","contextId":"earth"}`},
		{"earth -earth", `{"action":"top","contextId":"earth"}`},
		{"-mars earth", `{"action":"top","contextId":"earth"}`},
		{"-mars", `{"action":"restore","contextId":""}`},
		{"*earth *mars", `{"action":"top","contextId":"earth"}`},
		{"earth -*mars", `{"action":"top","contextId":"earth"}`},
		{"earth *-mars", `{"action":"top","contextId":"earth"}`},
		{strings.Repeat("earth ", 41), `{"action":"top","contextId":"earth"}`},
		// Bodies that aren't in the list.
		{"mercury venus", `{"action":"top","contextId":"vulcan"}`},
		{"mercury venus -mars", `{"action":"top","contextId":"mars"}`},
		{"mercury venus", `{"action":"remove","contextId":""}`},
		// Actions the demo doesn't have, or that break its rules.
		{"mercury venus", `{"action":"rename","contextId":"venus"}`},
		{"earth -mars", `{"action":"remove","contextId":"earth"}`},
		{"mercury venus", `{"action":"restore","contextId":""}`},
		{"mercury venus", `[1, 2]`},
		{"mercury venus", `{"action":`},
	} {
		if got, err := arrangeContextMenu(tc.state, []byte(tc.move)); err == nil {
			t.Errorf("%q with %s = %q, want an error", tc.state, tc.move, got)
		}
	}
}

// The rows tell the menu what doesn't apply to them, and only the marked
// body's button takes the focus.
func TestRenderContextMenu(t *testing.T) {
	html := renderContextMenu("m", "mercury *venus mars -earth")
	for _, want := range []string{
		`<li id="m-mercury" tabindex="-1" data-context-id="mercury" data-menu-for="m-menu" data-menu-param-name="Mercury" data-menu-disabled="top up">`,
		`<li id="m-venus" tabindex="-1" data-context-id="venus" data-menu-for="m-menu" data-menu-param-name="Venus">`,
		`<li id="m-mars" tabindex="-1" data-context-id="mars" data-menu-for="m-menu" data-menu-param-name="Mars" data-menu-disabled="bottom down">`,
		`aria-label="Actions for Venus" data-preserve-attr="style aria-expanded"` + "\n\t\t\t\t" + `data-init="!document.getElementById('m-earth') && document.activeElement === document.body && el.focus()">`,
		`>Bring back Earth</button>`,
		`<sb-context-menu id="m-menu" aria-label="Planet actions" data-ignore-morph>`,
	} {
		if !strings.Contains(html, want) {
			t.Errorf("no %s in\n%s", want, html)
		}
	}
	if n := strings.Count(html, "data-init="); n != 1 {
		t.Errorf("%d buttons take the focus, want 1", n)
	}
	if one := renderContextMenu("m", "earth -mars"); !strings.Contains(one, `data-menu-disabled="top up bottom down remove"`) {
		t.Errorf("the last body can be removed:\n%s", one)
	}
}
