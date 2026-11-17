// Package sb renders the Starbase components (https://starbase.zweiundeins.gmbh),
// Rocket web components for Datastar, from templ with typed props.
//
// Each component is a function and a props struct, generated from its
// manifest:
//
//	@sb.Toggle(sb.ToggleProps{Name: "notify", Label: "Notifications", Checked: true})
//
//	@sb.Modal(sb.ModalProps{Heading: "Delete this file?"}) {
//		This cannot be undone.
//		<button slot="footer" data-sb-close>Cancel</button>
//	}
//
// Imports, once in <head>, loads Datastar and the components, pinned with
// their integrity hashes to the versions the wrappers were generated with.
// Tracked, around the content of <body>, loads exactly the components
// rendered inside it.
//
// # Props
//
// Every props struct has ID, Class and Attrs, then one field per prop of
// the component, in its manifest's order. A field writes its attribute as
// follows:
//
//   - A string writes its text; "" leaves the attribute out, and the
//     component uses its default (the doc comment says which: labels carry
//     their English text there).
//   - A boolean is written by presence: true is the bare attribute, false
//     leaves it out. It is never written as "false".
//   - A Flag is a prop that is on by default (sb-meter's ShowValue): the
//     zero value Default leaves it out, Off writes x="false", On x="true".
//   - A prop with a fixed set of values has a named string type with a
//     constant per value (ToggleSizeLg); "" leaves it out.
//   - A number is a float64 when its default is 0, which leaves it out. When
//     its default is something else (sb-slider's Max), it is a *float64 set
//     with Num, so the zero value can't override the default.
//   - A JSON prop takes a Go value (DropdownItem, Option, Column, …), written
//     with encoding/json; nil leaves it out, and an empty non-nil slice or
//     map is written as [] or {}.
//   - An href is sanitized like templ's own: a javascript: URL becomes
//     about:invalid#TemplFailedSanitizationURL. Other values are written as
//     they are, escaped for HTML: templ's URL rules for names such as icon
//     or src don't apply to the props.
//
// # The server's word
//
// A component's value (sb-select's Value, sb-slider's Value, sb-tabs'
// Selected, …) is the server's word: a new value in the attribute wins over
// what the user did, and markup that sends the same value again keeps the
// user's edit. A morph that drops the attribute is ignored, so such fields
// are pointers: nil leaves the attribute out, Str("") or Num(0) sends an
// empty value or 0 that clears the user's. Booleans of this kind (checked,
// open) are written by presence, and removing them is the server's word too.
// Props whose presence means something on its own (sb-virtual-scroll's
// Total) are pointers for the same reason.
//
// # Attrs, children and slots
//
// Attrs are written after the props and win over a prop that writes the
// same attribute (names compare without case); they go through templ's own
// attribute rules. Datastar's attributes go there: "data-on:sb-change",
// "data-bind:x", "data-attr:value", "data-preserve-attr", and aria-*, slot,
// style. The events each component sends are listed in its doc comment.
//
// Templ children render inside the element. A named slot is the caller's:
// give an element slot="footer". Each component's doc comment lists its
// slots.
package sb
