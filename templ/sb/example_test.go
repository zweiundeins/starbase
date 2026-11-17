package sb_test

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/a-h/templ"

	"github.com/zweiundeins/starbase/templ/sb"
)

// In a .templ file this is @sb.Toggle(sb.ToggleProps{…}); a boolean is
// written by presence, an enum is a constant, events go in Attrs.
func ExampleToggle() {
	c := sb.Toggle(sb.ToggleProps{
		Name:    "notify",
		Label:   "Notifications",
		Checked: true,
		Size:    sb.ToggleSizeLg,
		Attrs:   templ.Attributes{"data-on:sb-change": "@post('/cmd/notify')"},
	})
	c.Render(context.Background(), os.Stdout)
	// Output: <sb-toggle checked label="Notifications" size="lg" name="notify" data-on:sb-change="@post(&#39;/cmd/notify&#39;)"></sb-toggle>
}

// Content goes in as templ children: @sb.Modal(…) { … }. A named slot is an
// element with slot="…".
func ExampleModal() {
	body := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		_, err := io.WriteString(w, `This cannot be undone.<button slot="footer" data-sb-close>Cancel</button>`)
		return err
	})
	c := sb.Modal(sb.ModalProps{Heading: "Delete this file?", Closable: sb.Off})
	c.Render(templ.WithChildren(context.Background(), body), os.Stdout)
	// Output: <sb-modal heading="Delete this file?" closable="false">This cannot be undone.<button slot="footer" data-sb-close>Cancel</button></sb-modal>
}

// JSON props take Go values.
func ExampleDropdown() {
	c := sb.Dropdown(sb.DropdownProps{
		Label: "Ship",
		Items: []sb.DropdownItem{
			{Value: "refuel", Label: "Refuel", Icon: "⛽"},
			{Divider: true},
			{Value: "scuttle", Label: "Scuttle", Danger: true},
		},
	})
	c.Render(context.Background(), os.Stdout)
	// Output: <sb-dropdown items="[{&#34;value&#34;:&#34;refuel&#34;,&#34;label&#34;:&#34;Refuel&#34;,&#34;icon&#34;:&#34;⛽&#34;},{&#34;divider&#34;:true},{&#34;value&#34;:&#34;scuttle&#34;,&#34;label&#34;:&#34;Scuttle&#34;,&#34;danger&#34;:true}]" label="Ship"></sb-dropdown>
}

// A value the server owns is a pointer: nil leaves the attribute out, and
// sb.Num(0) sends a 0 that wins over what the user dragged. A number whose
// default isn't 0 is a pointer too, so the zero value keeps the default.
func ExampleSlider() {
	sb.Slider(sb.SliderProps{Label: "Thrust", Value: sb.Num(0), Max: sb.Num(10), ShowValue: sb.Off}).
		Render(context.Background(), os.Stdout)
	fmt.Println()
	sb.Slider(sb.SliderProps{Label: "Thrust"}).Render(context.Background(), os.Stdout)
	// Output:
	// <sb-slider value="0" max="10" label="Thrust" show-value="false"></sb-slider>
	// <sb-slider label="Thrust"></sb-slider>
}

// Imports goes in <head>: Datastar and the components, pinned with their
// integrity hashes to the versions these wrappers were generated with.
func ExampleImports() {
	head := sb.Imports(sb.ImportsProps{Components: []sb.Tag{sb.TagToggle, sb.TagModal}})
	ctx := templ.WithNonce(context.Background(), "r4nd0m")
	if err := head.Render(ctx, io.Discard); err != nil {
		fmt.Println(err)
	}
	// Output:
}

// A whole number is an int: sb.Int(0) selects the first tab (the server's
// word), while a nil Selected leaves the user's choice alone. An empty
// Labels is no tabs; a nil one would be the default Home, Docs, API.
func ExampleTabs() {
	labels := []string{"Overview", "C++", "C#"}
	panels := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		for i := range labels {
			fmt.Fprintf(w, `<div slot="%s">…</div>`, sb.TabSlot(labels, i))
		}
		return nil
	})
	sb.Tabs(sb.TabsProps{Labels: labels, Selected: sb.Int(0)}).Render(templ.WithChildren(context.Background(), panels), os.Stdout)
	// Output: <sb-tabs labels="[&#34;Overview&#34;,&#34;C++&#34;,&#34;C#&#34;]" selected="0"><div slot="overview">…</div><div slot="c">…</div><div slot="c-2">…</div></sb-tabs>
}

// Collect tells a page with its own loader which components rendered, and
// Pin where each one's files are.
func ExampleCollect() {
	ctx, rendered := sb.Collect(context.Background())
	sb.DataTable(sb.DataTableProps{}).Render(ctx, io.Discard)
	for _, t := range rendered() {
		p, _ := sb.Pin(t)
		fmt.Println(t, strings.HasPrefix(p.Bundle.Path, "/c/"), strings.HasPrefix(p.Bundle.Integrity, "sha384-"))
	}
	// Output:
	// sb-data-table true true
	// sb-virtual-scroll true true
}
