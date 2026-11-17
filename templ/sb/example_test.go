package sb_test

import (
	"context"
	"fmt"
	"io"
	"os"

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
