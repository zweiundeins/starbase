// Package templgen generates the typed templ wrappers of the module
// github.com/zweiundeins/starbase/templ (package sb) from the components'
// manifests, and turns a component's usage markup into the templ call that
// renders it (the component pages' templ tab).
package templgen

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"unicode"

	"starbase/internal/catalog"
)

// Kind is how a field is written to its attribute.
type Kind int

const (
	KindString    Kind = iota // string: "" leaves the attribute out
	KindURL                   // string, sanitized like templ's href
	KindStringPtr             // *string: nil leaves it out, "" is written
	KindBool                  // bool: true is the bare attribute, false leaves it out
	KindFlag                  // Flag: Default leaves it out, On "true", Off "false"
	KindEnum                  // a named string type with a constant per value
	KindNumber                // float64: 0 (the default) leaves it out
	KindNumberPtr             // *float64: nil leaves it out, 0 is written
	KindJSON                  // a Go value written as JSON: nil leaves it out
)

// Component is one component's wrapper.
type Component struct {
	*catalog.Component
	GoName string // Toggle: the function; ToggleProps is its props
	Fields []*Field
}

// Field is a prop's field in the Props struct.
type Field struct {
	Prop   catalog.ManifestProp
	GoName string
	Kind   Kind
	// GoType is the field's type as written inside package sb.
	GoType string
	// JSON is the field's type, for KindJSON.
	JSON reflect.Type
	// Enum are the constants of a KindEnum's type, in the manifest's order.
	Enum []EnumConst
	// Default is the prop's default as decoded (null when none).
	Default any

	tag string
}

// EnumConst is a constant of a oneOf prop's type.
type EnumConst struct{ GoName, Value string }

// Model is every component's wrapper, sorted by slug.
type Model struct {
	Catalog    *catalog.Catalog
	Components []*Component
}

// Get returns the wrapper of the component with tag.
func (m *Model) Get(tag string) (*Component, bool) {
	for _, c := range m.Components {
		if c.Tag == tag {
			return c, true
		}
	}
	return nil, false
}

// Field returns the field that writes attribute name.
func (c *Component) Field(name string) (*Field, bool) {
	for _, f := range c.Fields {
		if f.Prop.Attribute == name {
			return f, true
		}
	}
	return nil, false
}

var identRe = regexp.MustCompile(`^[A-Z][A-Za-z0-9]*$`)

// Build makes the model of every component in cat. Each needs a manifest,
// and every prop a Go type: a json prop without one in jsonTypes, or a
// oneOf value that makes no Go name, is an error naming what to add.
func Build(cat *catalog.Catalog) (*Model, error) {
	m := &Model{Catalog: cat}
	var errs []string
	names := map[string]string{}
	for _, cc := range cat.Components {
		if cc.Manifest == nil {
			errs = append(errs, cc.Slug+": no manifest.json (go tool task manifests)")
			continue
		}
		c := &Component{Component: cc, GoName: goName(cc.Name)}
		if !identRe.MatchString(c.GoName) {
			errs = append(errs, fmt.Sprintf("%s: name %q makes no Go name", cc.Slug, cc.Name))
			continue
		}
		if other, dup := names[c.GoName]; dup {
			errs = append(errs, fmt.Sprintf("%s: Go name %s is %s's too", cc.Slug, c.GoName, other))
			continue
		}
		names[c.GoName] = cc.Slug
		fieldNames := map[string]bool{"ID": true, "Class": true, "Attrs": true}
		for _, p := range cc.Manifest.Props {
			f, err := buildField(c, p)
			if err != nil {
				errs = append(errs, fmt.Sprintf("%s: prop %s: %v", cc.Slug, p.Name, err))
				continue
			}
			if fieldNames[f.GoName] {
				errs = append(errs, fmt.Sprintf("%s: prop %s: field %s is taken", cc.Slug, p.Name, f.GoName))
				continue
			}
			fieldNames[f.GoName] = true
			c.Fields = append(c.Fields, f)
		}
		m.Components = append(m.Components, c)
	}
	for key := range jsonTypes {
		tag, attr, _ := strings.Cut(key, ".")
		if c, ok := m.Get(tag); !ok || func() bool { _, ok := c.Field(attr); return !ok }() {
			errs = append(errs, "jsonTypes names "+key+", which no manifest has")
		}
	}
	if len(errs) > 0 {
		return nil, fmt.Errorf("templ wrappers:\n  %s", strings.Join(errs, "\n  "))
	}
	return m, nil
}

func buildField(c *Component, p catalog.ManifestProp) (*Field, error) {
	f := &Field{Prop: p, GoName: fieldName(p.Name), tag: c.Tag}
	if !identRe.MatchString(f.GoName) {
		return nil, fmt.Errorf("makes no Go name (%q)", f.GoName)
	}
	if len(p.Default) > 0 {
		if err := json.Unmarshal(p.Default, &f.Default); err != nil {
			return nil, fmt.Errorf("default: %w", err)
		}
	}
	key := c.Tag + "." + p.Attribute
	switch p.Type {
	case "string":
		f.Kind, f.GoType = KindString, "string"
		if isURLAttr(p.Attribute) {
			f.Kind = KindURL
		}
		if serverWord[key] || presence[key] {
			f.Kind, f.GoType = KindStringPtr, "*string"
		}
	case "boolean":
		if f.Default != false {
			return nil, fmt.Errorf("a boolean is false by default, not %v", f.Default)
		}
		f.Kind, f.GoType = KindBool, "bool"
	case "number":
		f.Kind, f.GoType = KindNumber, "float64"
		if f.Default != 0.0 || serverWord[key] || presence[key] {
			f.Kind, f.GoType = KindNumberPtr, "*float64"
		}
	case "oneOf":
		if isFlag(p) {
			f.Kind, f.GoType = KindFlag, "Flag"
			break
		}
		f.Kind, f.GoType = KindEnum, c.GoName+f.GoName
		for _, v := range p.Values {
			s, ok := v.(string)
			if !ok {
				return nil, fmt.Errorf("oneOf value %v is not a string", v)
			}
			name := enumNames[key+"="+s]
			if name == "" {
				name = goName(s)
			}
			if !identRe.MatchString(name) {
				return nil, fmt.Errorf("oneOf value %q makes no Go name: add it to enumNames", s)
			}
			f.Enum = append(f.Enum, EnumConst{GoName: f.GoType + name, Value: s})
		}
	case "json", "array", "custom":
		t, ok := jsonTypes[key]
		if !ok {
			return nil, fmt.Errorf("a %s prop needs its Go type in jsonTypes (internal/templgen/types.go)", p.Type)
		}
		f.Kind, f.JSON, f.GoType = KindJSON, t, GoType(t, "")
	default:
		return nil, fmt.Errorf("unknown type %q", p.Type)
	}
	return f, nil
}

// isURLAttr reports whether a prop is a link the component renders as an
// <a href>, which the wrapper sanitizes like templ's href. Attributes templ
// would also treat as URLs by name (icon, anything with src or url) are
// text or data here and are written as they are.
func isURLAttr(name string) bool { return name == "href" }

// isFlag reports whether p is oneOf(true, false): a flag that is on by default.
func isFlag(p catalog.ManifestProp) bool {
	return p.Type == "oneOf" && len(p.Values) == 2 && p.Values[0] == true && p.Values[1] == false
}

// initialisms are written in capitals in Go names.
var initialisms = map[string]string{"id": "ID", "url": "URL", "qr": "QR", "html": "HTML", "json": "JSON", "ecc": "ECC", "ms": "MS"}

// goName turns a name ("QR Code", "bottom-start", "Table of Contents") into
// an exported Go name: QRCode, BottomStart, TableOfContents.
func goName(s string) string {
	var b strings.Builder
	for _, w := range strings.FieldsFunc(s, func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) }) {
		if up, ok := initialisms[strings.ToLower(w)]; ok {
			b.WriteString(up)
			continue
		}
		b.WriteString(strings.ToUpper(w[:1]) + w[1:])
	}
	return b.String()
}

// fieldName turns a prop name (camelCase) into a field name: rowKey is
// RowKey, resetMs is ResetMS.
func fieldName(prop string) string {
	var words []string
	start := 0
	for i, r := range prop {
		if i > 0 && unicode.IsUpper(r) {
			words = append(words, prop[start:i])
			start = i
		}
	}
	words = append(words, prop[start:])
	for i, w := range words {
		if up, ok := initialisms[strings.ToLower(w)]; ok {
			words[i] = up
		} else {
			words[i] = strings.ToUpper(w[:1]) + w[1:]
		}
	}
	return strings.Join(words, "")
}

// GoType writes t as Go source; qual qualifies the types of package sb
// ("sb." outside it).
func GoType(t reflect.Type, qual string) string {
	switch t.Kind() {
	case reflect.Pointer:
		return "*" + GoType(t.Elem(), qual)
	case reflect.Slice:
		return "[]" + GoType(t.Elem(), qual)
	case reflect.Map:
		return "map[" + GoType(t.Key(), qual) + "]" + GoType(t.Elem(), qual)
	case reflect.Interface:
		if t.NumMethod() == 0 {
			return "any"
		}
	}
	if t.PkgPath() == sbPath {
		return qual + t.Name()
	}
	return t.String()
}

// prettyJSON is a default for a doc comment.
func prettyJSON(raw json.RawMessage) string {
	var b bytes.Buffer
	if json.Compact(&b, raw) != nil {
		return string(raw)
	}
	return b.String()
}
