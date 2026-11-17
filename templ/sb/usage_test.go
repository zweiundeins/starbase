package sb

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

// TestUsageMatchesMarkup renders each component's install markup (the
// README's usage, or its preview) through the wrappers, as the docs' templ
// tab writes it, and compares the result with the markup: the same
// elements, text and attributes, where an attribute of a component counts
// as equal when it decodes to the same prop value (an absent one is its
// default; JSON compares as the field's Go value).
func TestUsageMatchesMarkup(t *testing.T) {
	byTag := map[string]manifest{}
	for _, m := range manifests(t) {
		byTag[m.Tag] = m
	}
	if len(usages) != len(byTag) {
		t.Errorf("%d usages for %d components", len(usages), len(byTag))
	}
	for _, u := range usages {
		got, want := significant(tokens(t, render(t, u.render))), significant(tokens(t, u.html))
		if len(got) != len(want) {
			t.Errorf("%s: %d tokens, want %d\n got %s\nwant %s", u.slug, len(got), len(want), render(t, u.render), u.html)
			continue
		}
		for i := range want {
			if err := sameToken(got[i], want[i], byTag); err != nil {
				t.Errorf("%s: %v\n got %s\nwant %s", u.slug, err, render(t, u.render), u.html)
			}
		}
	}
}

// significant drops whitespace-only text and collapses the rest.
func significant(toks []token) []token {
	var out []token
	for _, tok := range toks {
		if tok.kind == "text" {
			tok.text = strings.Join(strings.Fields(tok.text), " ")
			if tok.text == "" {
				continue
			}
		}
		out = append(out, tok)
	}
	return out
}

func sameToken(got, want token, byTag map[string]manifest) error {
	if got.kind != want.kind || got.tag != want.tag || got.text != want.text {
		return fmt.Errorf("%+v, want %+v", got, want)
	}
	if got.kind != "start" {
		return nil
	}
	m, isComponent := byTag[got.tag]
	props := map[string]int{}
	if isComponent {
		for i, p := range m.Props {
			props[p.Attribute] = i
		}
	}
	names := map[string]bool{}
	for _, a := range append(got.attrs, want.attrs...) {
		names[a.name] = true
	}
	for name := range names {
		g, gok := got.attr(name)
		w, wok := want.attr(name)
		i, isProp := props[name]
		if !isProp {
			if gok != wok || g != w {
				return fmt.Errorf("<%s> %s: %+v (%v), want %+v (%v)", got.tag, name, g, gok, w, wok)
			}
			continue
		}
		gv, err := decode(m, i, g, gok)
		if err != nil {
			return err
		}
		wv, err := decode(m, i, w, wok)
		if err != nil {
			return err
		}
		if !reflect.DeepEqual(gv, wv) {
			return fmt.Errorf("<%s> %s: %#v, want %#v", got.tag, name, gv, wv)
		}
	}
	return nil
}

// decode is the value a prop takes from its attribute, the way the
// component's codec reads it: presence for a boolean, "false" or not for a
// flag, the default when absent, JSON as the field's Go type.
func decode(m manifest, i int, a tokAttr, ok bool) (any, error) {
	p := m.Props[i]
	raw := string(p.Default)
	if ok {
		raw = a.value
	}
	switch p.Type {
	case "boolean":
		return ok, nil
	case "oneOf":
		if len(p.Values) == 2 && p.Values[0] == true {
			return !ok || a.value != "false", nil
		}
		if !ok {
			var s string
			err := json.Unmarshal(p.Default, &s)
			return s, err
		}
		return a.value, nil
	case "string":
		if !ok {
			var s string
			err := json.Unmarshal(p.Default, &s)
			return s, err
		}
		return a.value, nil
	case "number":
		return strconv.ParseFloat(strings.TrimSpace(raw), 64)
	}
	typ := reflect.TypeOf(entryByTag(m.Tag).New()).Elem().Field(3 + i).Type
	v := reflect.New(typ)
	if err := json.Unmarshal([]byte(raw), v.Interface()); err != nil {
		return nil, fmt.Errorf("<%s> %s=%s: %v", m.Tag, p.Attribute, raw, err)
	}
	return normalize(v.Elem()), nil
}

// normalize makes nil and empty slices and maps equal, as the components
// read them.
func normalize(v reflect.Value) any {
	switch v.Kind() {
	case reflect.Slice, reflect.Map:
		if v.Len() == 0 {
			return nil
		}
	case reflect.Pointer:
		if v.IsNil() {
			return nil
		}
		if v.Elem().IsZero() {
			return nil // {} for a struct is its zero value
		}
	}
	return v.Interface()
}

func entryByTag(tag string) Entry {
	for _, e := range Entries {
		if e.Tag == tag {
			return e
		}
	}
	panic("no wrapper for " + tag)
}
