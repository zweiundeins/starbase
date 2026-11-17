# Starbase for templ

Typed [templ](https://templ.guide) components for the [Starbase](https://starbase.zweiundeins.gmbh) catalog: 52 Rocket web components for [Datastar](https://data-star.dev) (forms, menus, dialogs, a data table, a tree, charts), each a Go function with a props struct generated from the component's manifest.

```templ
@sb.Slider(sb.SliderProps{Name: "thrust", Label: "Thrust", Value: sb.Num(72), Unit: "%"})
```

The module depends on templ and the standard library only.

## Install

```sh
go get github.com/zweiundeins/starbase/templ@latest
```

```go
import "github.com/zweiundeins/starbase/templ/sb"
```

A version of the module (`templ/vX.Y.Z`) is tagged from the same commit as the Starbase release `vX.Y.Z`, and pins the components and the Datastar build of that release.

## A page

```templ
package page

import "github.com/zweiundeins/starbase/templ/sb"

templ Mission(thrust float64, shields bool) {
	<!DOCTYPE html>
	<html lang="en">
		<head>
			<meta charset="utf-8"/>
			<title>Mission control</title>
			@sb.Imports(sb.ImportsProps{Components: []sb.Tag{sb.TagSlider, sb.TagToggle, sb.TagModal, sb.TagDropdown}})
		</head>
		<body>
			@sb.Slider(sb.SliderProps{
				Name:  "thrust",
				Label: "Thrust",
				Value: sb.Num(thrust),
				Unit:  "%",
				Attrs: templ.Attributes{"data-on:sb-change": "@post('/cmd/flight', {payload: evt.detail})"},
			})
			@sb.Toggle(sb.ToggleProps{Name: "shields", Label: "Shields", Checked: shields})
			@sb.Dropdown(sb.DropdownProps{
				Label: "Ship",
				Items: []sb.DropdownItem{
					{Value: "refuel", Label: "Refuel", Icon: "⛽"},
					{Divider: true},
					{Value: "scuttle", Label: "Scuttle", Danger: true},
				},
				Attrs: templ.Attributes{"data-on:sb-select": "@post('/cmd/' + evt.detail.value)"},
			})
			<button data-on:click="$_confirm.show()">Abort…</button>
			@sb.Modal(sb.ModalProps{Heading: "Abort the mission?", Attrs: templ.Attributes{"data-ref:_confirm": true}}) {
				The crew comes home.
				<button slot="footer" data-sb-close="cancel">Cancel</button>
				<button slot="footer" data-sb-close="abort">Abort</button>
			}
		</body>
	</html>
}
```

`sb.Imports` writes an import map for Datastar and a module script for each component (and the components it renders itself), every file pinned with its integrity hash, so the browser refuses one that changed. It takes the CSP nonce from `templ.WithNonce`. Without `Components`, it loads the catalog's autoloader instead, which loads each `<sb-*>` tag the first time it appears (also tags a morph adds later). `Base` points it at another Starbase server, `Datastar` at a Datastar build with Rocket the page loads already.

Every component page on [the site](https://starbase.zweiundeins.gmbh) has a templ tab under Installation with its usage as wrapper calls.

## Props

Every props struct has `ID`, `Class` and `Attrs`, then one field per prop of the component, in the manifest's order, with the prop's docs as its doc comment. The field writes its attribute like this:

| Manifest type | Go type | Attribute |
| --- | --- | --- |
| `string` | `string` | the text; `""` leaves it out, for the component's default (the doc comment carries it, e.g. the English label) |
| `boolean` | `bool` | `true` is the bare attribute, `false` leaves it out; never `"false"` |
| `oneOf(true, false)`, on by default | `sb.Flag` | `sb.Default` leaves it out, `sb.Off` writes `x="false"`, `sb.On` `x="true"` |
| `oneOf("sm", "md", "lg")` | a named type, e.g. `sb.ToggleSize` | a constant per value (`sb.ToggleSizeLg`); `""` leaves it out; another value is a render error |
| `number`, default 0 | `float64` | the number; 0 leaves it out |
| `number`, another default | `*float64` | `nil` leaves it out, `sb.Num(x)` writes x, also 0 |
| `json`, `array` | a Go type: `[]sb.DropdownItem`, `[]sb.Option`, `[]sb.Column`, `*sb.RangeValue`, `[]string`, … | JSON (`encoding/json`); `nil` leaves it out, an empty slice or map writes `[]` or `{}` |
| the server's word (`value` and the like) | `*string`, `*float64` | `nil` leaves it out, `sb.Str("")` or `sb.Num(0)` sends an empty value or 0 |

The server's word: a component's value (`sb-select`'s `Value`, `sb-slider`'s `Value`, `sb-tabs`' `Selected`, `sb-tree`'s `Value` and `Expanded`, …) wins over what the user did whenever the server sends a new one, while markup that sends the same value again keeps the user's edit. A morph that drops such an attribute is ignored, so these fields are pointers, and the server clears with `sb.Str("")` or `sb.Num(0)`. Booleans of this kind (`Checked`, `Open`) are written by presence, and dropping them is the server's word too. Each field's doc comment says which rule it follows.

The JSON shapes are typed where the component's docs describe them: `sb.Option` (written as a plain string when it has only a value), `sb.DropdownItem`, `sb.Column` and `sb.Cell`, `sb.TableSort`, `sb.RangeValue`, `sb.Diagnostic`, `sb.TreeItem`, `sb.ToastMessage`. Three places take any value: `sb-echarts`' `Option` (an ECharts option), a `sb-data-table` row (`map[string]any`, keyed by your columns) and a cell's `Value`.

An `href` (`sb-button`, `sb-card`, a dropdown item's or a cell's) is sanitized like templ's own: a `javascript:` URL becomes `about:invalid#TemplFailedSanitizationURL`. Other props are written as they are, escaped for HTML: templ's URL rules for attribute names like `icon` or `src` don't apply to them.

## Events, Datastar and slots

`Attrs` are written after the props and go through templ's attribute rules. They win over a prop that writes the same attribute. Events and Datastar bindings go there:

```templ
@sb.Select(sb.SelectProps{
	Name:    "planet",
	Label:   "Destination",
	Options: []sb.Option{{Value: "Mars"}, {Value: "Venus"}},
	Attrs: templ.Attributes{
		"data-on:sb-change": "@post('/cmd/planet', {payload: evt.detail})",
		"data-on:datastar-fetch": "evt.detail.el === el && evt.detail.type === 'error' && el.revert()",
	},
})
```

Each component's doc comment lists the events it sends, and its slots. Templ children render inside the element; a named slot is an element with `slot="…"`.

## Contributing

The wrappers are generated: `go tool task templ-wrappers` in the Starbase repository writes `sb/components_gen.go` and `sb/pins_gen.go` from the components' manifests and files (`cmd/templgen`, `internal/templgen`), and CI fails when they are stale. The JSON shapes are in `sb/types.go`. Tests render every wrapper with every field set and compare the attributes with the manifests, render each component's README usage through the wrappers, and load all of them in headless Chrome, where every decoded prop must equal its Go value.

MIT licensed, © zwei und eins gmbh and Starbase contributors.
