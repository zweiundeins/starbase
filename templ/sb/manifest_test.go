package sb

import (
	"encoding/json"
	"fmt"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"
	"unicode"

	"github.com/a-h/templ"
)

// These tests hold the wrappers to the manifests on their own terms: the
// rules for each manifest type are written out here again, not taken from
// the generator.

// wantField is the field name of a prop: rowKey is RowKey, resetMs ResetMS.
func wantField(prop string) string {
	var b strings.Builder
	for _, w := range splitCamel(prop) {
		switch strings.ToLower(w) {
		case "id", "url", "ms", "html", "json", "ecc":
			b.WriteString(strings.ToUpper(w))
		default:
			b.WriteString(strings.ToUpper(w[:1]) + w[1:])
		}
	}
	return b.String()
}

func splitCamel(s string) []string {
	var out []string
	start := 0
	for i, r := range s {
		if i > 0 && unicode.IsUpper(r) {
			out = append(out, s[start:i])
			start = i
		}
	}
	return append(out, s[start:])
}

func entry(t *testing.T, tag string) Entry {
	t.Helper()
	for _, e := range Entries {
		if e.Tag == tag {
			return e
		}
	}
	t.Fatalf("no wrapper for <%s>", tag)
	return Entry{}
}

var (
	flagType  = reflect.TypeFor[Flag]()
	attrsType = reflect.TypeFor[templ.Attributes]()
)

// integers are the number props that hold whole numbers, by
// "<tag>.<attribute>", as their docs and code read: counts, indexes,
// digits, and times in ms or s. Every other number is a float64.
var integers = map[string]bool{
	"sb-autoloader.timeout": true, "sb-busy.lines": true, "sb-busy.delay": true, "sb-busy.min": true,
	"sb-code-editor.tab-size": true, "sb-code-playground.delay": true, "sb-copy-button.reset-ms": true,
	"sb-count-up.decimals": true, "sb-count-up.duration": true, "sb-data-table.offset": true, "sb-data-table.total": true,
	"sb-date-picker.step": true, "sb-gauge.decimals": true, "sb-input.minlength": true,
	"sb-meter.segments": true, "sb-meter.decimals": true, "sb-nebula.levels": true, "sb-nebula.seed": true,
	"sb-odometer.decimals": true, "sb-pixel-board.size": true, "sb-pixel-board.color": true, "sb-qr-code.border": true,
	"sb-rating.max": true, "sb-select.delay": true, "sb-select.min-chars": true, "sb-select.max-chips": true, "sb-select.total": true,
	"sb-sparkline.length": true, "sb-sparkline.decimals": true, "sb-starfield.density": true, "sb-tabs.selected": true,
	"sb-toast.max": true, "sb-toast.duration": true, "sb-typewriter.interval": true, "sb-typewriter.delay": true, "sb-typewriter.hold": true,
	"sb-virtual-scroll.total": true, "sb-virtual-scroll.offset": true, "sb-virtual-scroll.columns": true, "sb-virtual-scroll.step": true,
}

// stringLists are the string props that hold a list: ids or heading levels
// separated by spaces, and sb-select's value (one, or with multiple several).
var stringLists = map[string]bool{"sb-select.value": true, "sb-toc.levels": true, "sb-tree.value": true, "sb-tree.expanded": true}

// TestPropsMatchManifests: every wrapper has ID, Class and Attrs, then one
// field per manifest prop, in order, of the Go type its manifest type maps
// to; and every component has a wrapper.
func TestPropsMatchManifests(t *testing.T) {
	all := manifests(t)
	if len(all) != len(Entries) {
		t.Errorf("%d manifests, %d wrappers", len(all), len(Entries))
	}
	props := 0
	for _, m := range all {
		typ := reflect.TypeOf(entry(t, m.Tag).New()).Elem()
		want := []string{"ID string", "Class string", "Attrs " + attrsType.String()}
		for _, p := range m.Props {
			want = append(want, wantField(p.Name)+" "+wantKind(t, m.Tag+"."+p.Attribute, p.Type, p.Values, p.Default))
			props++
		}
		var got []string
		match := typ.NumField() == len(want)
		for i := range typ.NumField() {
			f := typ.Field(i)
			got = append(got, f.Name+" "+kindOf(f.Type))
			if i < len(want) {
				name, kinds, _ := strings.Cut(want[i], " ")
				match = match && name == f.Name && slices.Contains(strings.Split(kinds, "|"), kindOf(f.Type))
			}
		}
		if !match {
			t.Errorf("%s:\n got %s\nwant %s", typ.Name(), strings.Join(got, "; "), strings.Join(want, "; "))
		}
	}
	t.Logf("%d props of %d components", props, len(all))
}

// wantKind is how a manifest type is typed in Go, as kindOf writes it.
func wantKind(t *testing.T, key, typ string, values []any, def json.RawMessage) string {
	switch typ {
	case "boolean":
		return "bool"
	case "string":
		if stringLists[key] {
			return "[]string"
		}
		return "string|*string"
	case "number":
		n := "float64"
		if integers[key] {
			n = "int"
		}
		if string(def) != "0" {
			return "*" + n // unset must not override a default that isn't 0
		}
		return n + "|*" + n
	case "oneOf":
		if len(values) == 2 && values[0] == true && values[1] == false {
			return "Flag"
		}
		return "enum"
	case "json", "array", "custom":
		return "nillable|[]string"
	}
	t.Fatalf("%s: unknown manifest type %q", key, typ)
	return ""
}

func kindOf(t reflect.Type) string {
	switch {
	case t == flagType:
		return "Flag"
	case t == attrsType:
		return attrsType.String()
	case t.Kind() == reflect.String && t.Name() == "string":
		return "string"
	case t.Kind() == reflect.String:
		return "enum"
	case t.Kind() == reflect.Bool:
		return "bool"
	case t.Kind() == reflect.Float64:
		return "float64"
	case t.Kind() == reflect.Int:
		return "int"
	case t.Kind() == reflect.Pointer && t.Elem().Kind() == reflect.Float64:
		return "*float64"
	case t.Kind() == reflect.Pointer && t.Elem().Kind() == reflect.Int:
		return "*int"
	case t == reflect.TypeFor[[]string]():
		return "[]string"
	case t.Kind() == reflect.Pointer && t.Elem().Kind() == reflect.String:
		return "*string"
	case t.Kind() == reflect.Slice, t.Kind() == reflect.Map, t.Kind() == reflect.Pointer, t.Kind() == reflect.Interface:
		return "nillable"
	}
	return t.String()
}

// TestEveryFieldRendersItsAttribute sets each field of every wrapper, one
// at a time and all at once, and reads the attribute back: strings exactly
// (escaped and unescaped again), booleans bare, flags "true"/"false", enum
// values as given (and others refused), numbers as JSON numbers, JSON values
// equal after a round trip; and the zero value of each field writes nothing.
func TestEveryFieldRendersItsAttribute(t *testing.T) {
	checked := 0
	for _, m := range manifests(t) {
		e := entry(t, m.Tag)
		all := reflect.ValueOf(e.New()).Elem()
		var wantAll []func(token) error
		for i, p := range m.Props {
			fv := all.Field(3 + i)
			for _, s := range samples(t, m.Tag, p.Type, fv.Type(), p.Attribute, p.Values) {
				one := reflect.ValueOf(e.New())
				one.Elem().Field(3 + i).Set(s.v)
				out := render(t, e.Render(one.Interface()))
				h := host(t, out, m.Tag)
				if err := s.check(h); err != nil {
					t.Errorf("%s %s = %s: %v\n%s", m.Tag, p.Name, s.name, err, out)
				}
				if len(h.attrs) > 1 || len(h.attrs) == 1 && h.attrs[0].name != p.Attribute {
					t.Errorf("%s %s = %s wrote other attributes: %s", m.Tag, p.Name, s.name, out)
				}
				checked++
			}
			last := samples(t, m.Tag, p.Type, fv.Type(), p.Attribute, p.Values)
			s := last[len(last)-1]
			fv.Set(s.v)
			wantAll = append(wantAll, s.check)
		}
		all.Field(0).SetString("the-id")
		all.Field(1).SetString("a b")
		out := render(t, e.Render(all.Addr().Interface()))
		h := host(t, out, m.Tag)
		if a, _ := h.attr("id"); a.value != "the-id" {
			t.Errorf("%s: id %q", m.Tag, a.value)
		}
		if a, _ := h.attr("class"); a.value != "a b" {
			t.Errorf("%s: class %q", m.Tag, a.value)
		}
		for _, check := range wantAll {
			if err := check(h); err != nil {
				t.Errorf("%s, every field set: %v\n%s", m.Tag, err, out)
			}
		}
	}
	t.Logf("%d values checked", checked)
}

type sample struct {
	name  string
	v     reflect.Value
	check func(token) error
}

func present(name, want string) func(token) error {
	return func(h token) error {
		a, ok := h.attr(name)
		switch {
		case !ok:
			return fmt.Errorf("no %s", name)
		case a.bare:
			return fmt.Errorf("%s is bare, want %q", name, want)
		case a.value != want:
			return fmt.Errorf("%s=%q, want %q", name, a.value, want)
		}
		return nil
	}
}

func absent(name string) func(token) error {
	return func(h token) error {
		if _, ok := h.attr(name); ok {
			return fmt.Errorf("%s is written", name)
		}
		return nil
	}
}

func samples(t *testing.T, tag, ptype string, typ reflect.Type, attr string, values []any) []sample {
	t.Helper()
	if ptype == "string" && kindOf(typ) == "[]string" {
		if tag == "sb-select" { // with multiple a JSON array, else the one value
			return []sample{
				{"nil", reflect.Zero(typ), absent(attr)},
				{"empty", reflect.ValueOf([]string{}), present(attr, "")},
				{"one", reflect.ValueOf([]string{nasty}), func(h token) error {
					a, ok := h.attr(attr)
					var list []string
					if ok && json.Unmarshal([]byte(a.value), &list) == nil && len(list) == 1 && list[0] == nasty {
						return nil
					}
					return present(attr, nasty)(h)
				}},
			}
		}
		return []sample{
			{"nil", reflect.Zero(typ), absent(attr)},
			{"empty", reflect.ValueOf([]string{}), present(attr, "")},
			{"two", reflect.ValueOf([]string{"a<&>", "b\"é"}), present(attr, `a<&> b"é`)},
		}
	}
	val := func(v any) reflect.Value { return reflect.ValueOf(v).Convert(typ) }
	switch kindOf(typ) {
	case "bool":
		return []sample{
			{"false", val(false), absent(attr)},
			{"true", val(true), func(h token) error {
				if a, ok := h.attr(attr); !ok || !a.bare {
					return fmt.Errorf("%s is not bare: %+v", attr, a)
				}
				return nil
			}},
		}
	case "Flag":
		return []sample{{"Default", val(Default), absent(attr)}, {"On", val(On), present(attr, "true")}, {"Off", val(Off), present(attr, "false")}}
	case "enum":
		out := []sample{{`""`, val(""), absent(attr)}}
		for _, v := range values {
			s := v.(string)
			out = append(out, sample{strconv.Quote(s), val(s), present(attr, s)})
		}
		return out
	case "string", "*string":
		text := nasty
		if attr == "href" {
			text = "https://example.com/?a=1&b=<x>"
		}
		if typ.Kind() == reflect.Pointer {
			return []sample{
				{"nil", reflect.Zero(typ), absent(attr)},
				{`Str("")`, reflect.ValueOf(Str("")), present(attr, "")},
				{"Str(text)", reflect.ValueOf(Str(text)), present(attr, text)},
			}
		}
		return []sample{{`""`, val(""), absent(attr)}, {"text", val(text), present(attr, text)}}
	case "int", "*int":
		if typ.Kind() == reflect.Pointer {
			return []sample{
				{"nil", reflect.Zero(typ), absent(attr)},
				{"Int(0)", reflect.ValueOf(Int(0)), present(attr, "0")},
				{"Int(-2)", reflect.ValueOf(Int(-2)), present(attr, "-2")},
			}
		}
		return []sample{{"0", val(0), absent(attr)}, {"7", val(7), present(attr, "7")}, {"-3", val(-3), present(attr, "-3")}}
	case "float64", "*float64":
		if typ.Kind() == reflect.Pointer {
			return []sample{
				{"nil", reflect.Zero(typ), absent(attr)},
				{"Num(0)", reflect.ValueOf(Num(0)), present(attr, "0")},
				{"Num(-2.5)", reflect.ValueOf(Num(-2.5)), present(attr, "-2.5")},
			}
		}
		return []sample{{"0", val(0.0), absent(attr)}, {"0.25", val(0.25), present(attr, "0.25")}, {"-3", val(-3.0), present(attr, "-3")}}
	case "nillable", "[]string":
		full := fill(typ, 0)
		check := func(h token) error {
			a, ok := h.attr(attr)
			if !ok {
				return fmt.Errorf("no %s", attr)
			}
			back := reflect.New(typ)
			if err := json.Unmarshal([]byte(a.value), back.Interface()); err != nil {
				return fmt.Errorf("%s=%s: %v", attr, a.value, err)
			}
			if !reflect.DeepEqual(back.Elem().Interface(), full.Interface()) {
				return fmt.Errorf("%s=%s came back as %#v, want %#v", attr, a.value, back.Elem().Interface(), full.Interface())
			}
			return nil
		}
		out := []sample{{"nil", reflect.Zero(typ), absent(attr)}}
		if typ.Kind() == reflect.Slice {
			out = append(out, sample{"empty", reflect.MakeSlice(typ, 0, 0), present(attr, "[]")})
		}
		if typ.Kind() == reflect.Map {
			out = append(out, sample{"empty", reflect.MakeMap(typ), present(attr, "{}")})
		}
		return append(out, sample{"full", full, check})
	}
	t.Fatalf("no samples for %s", typ)
	return nil
}

// fill makes a value of typ with every field set, to compare after a round
// trip through JSON.
func fill(typ reflect.Type, depth int) reflect.Value {
	v := reflect.New(typ).Elem()
	switch typ.Kind() {
	case reflect.String:
		if typ.Name() != "string" { // a named type: its constants are lowercase words
			v.SetString("end")
			if typ == reflect.TypeFor[CellTone]() || typ == reflect.TypeFor[ToastVariant]() {
				v.SetString("danger")
			} else if typ == reflect.TypeFor[SortDir]() {
				v.SetString("desc")
			} else if typ == reflect.TypeFor[DropdownItemType]() {
				v.SetString("radio")
			}
		} else {
			v.SetString(nasty)
		}
	case reflect.Bool:
		v.SetBool(true)
	case reflect.Int:
		v.SetInt(7)
	case reflect.Float64:
		v.SetFloat(-12.5)
	case reflect.Uint8: // Flag
		v.Set(reflect.ValueOf(Off))
	case reflect.Pointer:
		v.Set(fill(typ.Elem(), depth).Addr())
	case reflect.Slice:
		if depth > 1 {
			return v
		}
		v = reflect.MakeSlice(typ, 0, 2)
		v = reflect.Append(v, fill(typ.Elem(), depth+1), fill(typ.Elem(), depth+1))
	case reflect.Map:
		v = reflect.MakeMap(typ)
		v.SetMapIndex(reflect.ValueOf("k <&>"), fill(typ.Elem(), depth+1))
		v.SetMapIndex(reflect.ValueOf("ünï"), fill(typ.Elem(), depth+1))
	case reflect.Interface: // free JSON: what encoding/json decodes it to
		v.Set(reflect.ValueOf(map[string]any{"s": nasty, "n": 1.5, "b": true, "l": []any{"a", 2.0}, "nil": nil}))
	case reflect.Struct:
		for i := range typ.NumField() {
			f := typ.Field(i)
			if f.Name == "Href" {
				v.Field(i).SetString("https://example.com/?a=1&b=2")
				continue
			}
			v.Field(i).Set(fill(f.Type, depth))
		}
		if typ == reflect.TypeFor[DropdownItem]() && depth > 0 { // a submenu holds no grandchildren here
			v.FieldByName("Children").Set(reflect.Zero(v.FieldByName("Children").Type()))
		}
	}
	return v
}
