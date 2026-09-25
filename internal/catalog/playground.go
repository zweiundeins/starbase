package catalog

import (
	"encoding/json"
	"fmt"
	"html"
	"maps"
	"math"
	"slices"
	"strconv"
	"strings"
)

// PlaygroundMeta tunes the generated Playground from README front matter:
//
//	playground:
//	  props: { yaw: {min: -180, max: 180}, zoom: {min: 0.5, max: 3, step: 0.1} }
//	  values: { yaw: 30 }        # initial values (default: the prop defaults)
//	  content: Blast off         # slotted content of the live element
//	  style: "inline-size: 18rem" # inline style of the live element
//	  attrs: {values: "[1,2,3]"}   # static attributes (e.g. props the controls skip)
//	  exclude: [href]
//	  sync: {sb-orbit: [yaw, pitch]} # an event whose detail moves these controls
type PlaygroundMeta struct {
	Props   map[string]PlaygroundRange `yaml:"props"`
	Values  map[string]any             `yaml:"values"`
	Content string                     `yaml:"content"`
	Style   string                     `yaml:"style"`
	Attrs   map[string]string          `yaml:"attrs"`
	Exclude []string                   `yaml:"exclude"`
	Sync    map[string][]string        `yaml:"sync"`
}

type PlaygroundRange struct {
	Min  *float64 `yaml:"min"`
	Max  *float64 `yaml:"max"`
	Step *float64 `yaml:"step"`
}

// Control kinds.
const (
	ControlNumber = "number"
	ControlBool   = "bool"
	ControlSelect = "select"
	ControlText   = "text"
)

// Control is one input of the Playground, bound to the signal $_pg.<Prop>.
type Control struct {
	Prop        string // camelCase prop name = signal key
	Attr        string // kebab-case attribute
	Kind        string
	Description string
	Min         float64
	Max         float64
	Step        float64
	Options     []string
	Initial     any
	Default     any // the component's own default, omitted from the markup
}

// Playground is everything needed to render a component's live controls.
type Playground struct {
	Tag      string
	Controls []Control
	Content  string
	Style    string
	Attrs    map[string]string
	Sync     map[string][]string // event → props its detail reports (only on the live element)
}

// Playground derives controls from the manifest; nil if nothing is tweakable.
func (c *Component) Playground() *Playground {
	if c.Manifest == nil {
		return nil
	}
	pg := &Playground{Tag: c.Tag, Content: c.Meta.Playground.Content, Style: c.Meta.Playground.Style, Attrs: c.Meta.Playground.Attrs, Sync: c.Meta.Playground.Sync}
	excluded := map[string]bool{}
	for _, e := range c.Meta.Playground.Exclude {
		excluded[e] = true
	}
	for _, p := range c.Manifest.Props {
		if excluded[p.Name] || excluded[p.Attribute] {
			continue
		}
		var def any
		_ = json.Unmarshal(p.Default, &def)
		ctl := Control{Prop: p.Name, Attr: p.Attribute, Default: def, Initial: def}
		if p.Docs != nil {
			ctl.Description = p.Docs.Description
		}
		switch p.Type {
		case "number":
			ctl.Kind = ControlNumber
			ctl.Min, ctl.Max, ctl.Step = 0, 100, 1
			if f, ok := def.(float64); ok {
				// Make sure the default is reachable.
				ctl.Min = math.Min(ctl.Min, math.Floor(f))
				ctl.Max = math.Max(ctl.Max, math.Ceil(f))
			}
			if r, ok := c.Meta.Playground.Props[p.Name]; ok {
				if r.Min != nil {
					ctl.Min = *r.Min
				}
				if r.Max != nil {
					ctl.Max = *r.Max
				}
				if r.Step != nil {
					ctl.Step = *r.Step
				}
			}
		case "boolean":
			ctl.Kind = ControlBool
		case "oneOf":
			ctl.Kind = ControlSelect
			for _, v := range p.Values {
				if s, ok := v.(string); ok {
					ctl.Options = append(ctl.Options, s)
				}
			}
			if len(ctl.Options) == 0 {
				continue
			}
		case "string":
			ctl.Kind = ControlText
			if def == nil {
				ctl.Default, ctl.Initial = "", ""
			}
		default: // json, arrays, objects, dates: not playground material
			continue
		}
		if v, ok := c.Meta.Playground.Values[p.Name]; ok {
			ctl.Initial = v
			if ctl.Kind == ControlNumber {
				if f, ok := toFloat(v); ok {
					ctl.Initial = f
				}
			}
		}
		pg.Controls = append(pg.Controls, ctl)
	}
	if len(pg.Controls) == 0 {
		return nil
	}
	return pg
}

func toFloat(v any) (float64, bool) {
	switch n := v.(type) {
	case int:
		return float64(n), true
	case float64:
		return n, true
	}
	return 0, false
}

// Signals is the initial value of $_pg as a JS object literal.
func (pg *Playground) Signals() string {
	m := map[string]any{}
	for _, c := range pg.Controls {
		m[c.Prop] = c.Initial
	}
	b, _ := json.Marshal(m)
	return string(b)
}

func js(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}

// Element renders the live element: every tweakable attribute is driven
// by its signal and also rendered statically, so the first paint matches.
func (pg *Playground) Element() string {
	var b strings.Builder
	b.WriteString("<" + pg.Tag)
	if pg.Style != "" {
		fmt.Fprintf(&b, ` style="%s"`, html.EscapeString(pg.Style))
	}
	for _, k := range slices.Sorted(maps.Keys(pg.Attrs)) {
		fmt.Fprintf(&b, ` %s="%s"`, k, html.EscapeString(pg.Attrs[k]))
	}
	for _, c := range pg.Controls {
		switch c.Kind {
		case ControlBool:
			if c.Default == true {
				// Absent means true, so "off" must be spelled out.
				if c.Initial == false {
					fmt.Fprintf(&b, ` %s="false"`, c.Attr)
				}
				fmt.Fprintf(&b, ` data-attr:%s="$_pg.%s ? null : 'false'"`, c.Attr, c.Prop)
				continue
			}
			if c.Initial == true {
				b.WriteString(" " + c.Attr)
			}
			// Off removes the attribute and sets the property: a component that
			// ignores a removed attribute (sb-toggle's checked, which the server
			// clears with "false") still follows. Not "false": :host([x]) rules
			// would match it.
			fmt.Fprintf(&b, ` data-attr:%s="$_pg.%s || (el.%s = false)"`, c.Attr, c.Prop, c.Prop)
		case ControlText:
			if s, _ := c.Initial.(string); s != "" {
				fmt.Fprintf(&b, ` %s="%s"`, c.Attr, html.EscapeString(s))
			}
			// Empty strings remove the attribute so the component default applies.
			fmt.Fprintf(&b, ` data-attr:%s="$_pg.%s || null"`, c.Attr, c.Prop)
		default:
			fmt.Fprintf(&b, ` %s="%s"`, c.Attr, html.EscapeString(fmt.Sprint(c.Initial)))
			fmt.Fprintf(&b, ` data-attr:%s="$_pg.%s"`, c.Attr, c.Prop)
		}
	}
	// When the user changes a synced prop on the element itself (a drag), the
	// event's detail moves its controls, snapped to their steps. Only here, not
	// in Markup: the copied snippet has no controls to move.
	for _, ev := range slices.Sorted(maps.Keys(pg.Sync)) {
		var sets []string
		for _, prop := range pg.Sync[ev] {
			for _, c := range pg.Controls {
				if c.Prop != prop || c.Kind != ControlNumber {
					continue
				}
				v := "evt.detail." + prop
				if c.Step == 1 {
					v = "Math.round(" + v + ")"
				} else if c.Step > 0 {
					v = fmt.Sprintf("+(Math.round(%s / %s) * %s).toFixed(%d)", v, fmtFloat(c.Step), fmtFloat(c.Step), decimals(c.Step))
				}
				sets = append(sets, "$_pg."+prop+" = "+v)
			}
		}
		if len(sets) > 0 {
			fmt.Fprintf(&b, ` data-on:%s="%s"`, ev, html.EscapeString(strings.Join(sets, "; ")))
		}
	}
	b.WriteString(">" + pg.Content + "</" + pg.Tag + ">")
	return b.String()
}

func fmtFloat(f float64) string { return strconv.FormatFloat(f, 'f', -1, 64) }

// decimals is how many decimals a step has (0.05 → 2).
func decimals(step float64) int {
	s := fmtFloat(step)
	if i := strings.IndexByte(s, '.'); i >= 0 {
		return len(s) - i - 1
	}
	return 0
}

// Markup is a Datastar expression that evaluates to the element's HTML for
// the current signals, omitting attributes that equal the defaults.
func (pg *Playground) Markup() string {
	open := "<" + pg.Tag
	for _, k := range slices.Sorted(maps.Keys(pg.Attrs)) {
		open += fmt.Sprintf(` %s='%s'`, k, pg.Attrs[k])
	}
	parts := []string{js(open)}
	for _, c := range pg.Controls {
		sig := "$_pg." + c.Prop
		switch c.Kind {
		case ControlBool:
			if c.Default == true {
				parts = append(parts, fmt.Sprintf("(%s ? '' : %s)", sig, js(" "+c.Attr+`="false"`)))
				continue
			}
			parts = append(parts, fmt.Sprintf("(%s ? %s : '')", sig, js(" "+c.Attr)))
		case ControlText:
			parts = append(parts, fmt.Sprintf("(%s && %s !== %s ? %s + String(%s).replaceAll('\"', '&quot;') + '\"' : '')", sig, sig, js(c.Default), js(" "+c.Attr+`="`), sig))
		default:
			parts = append(parts, fmt.Sprintf("(%s !== %s ? %s + %s + '\"' : '')", sig, js(c.Default), js(" "+c.Attr+`="`), sig))
		}
	}
	parts = append(parts, js(">"+pg.Content+"</"+pg.Tag+">"))
	return strings.Join(parts, " + ")
}
