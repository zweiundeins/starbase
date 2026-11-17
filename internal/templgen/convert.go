package templgen

import (
	"encoding/json"
	"fmt"
	"go/format"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/zweiundeins/starbase/templ/sb"
)

// PropsLiteral is the Go literal of c's props that renders the attributes
// of n, sb's names qualified by qual ("sb." outside package sb): id and
// class become ID and Class, a prop's attribute its field, anything else
// goes to Attrs.
func (c *Component) PropsLiteral(n *Node, qual string) (string, error) {
	var fields, attrs []string
	for _, a := range n.Attrs {
		switch a.Name {
		case "id":
			fields = append(fields, "ID: "+strconv.Quote(a.Value))
			continue
		case "class":
			fields = append(fields, "Class: "+strconv.Quote(a.Value))
			continue
		}
		f, ok := c.Field(a.Name)
		if !ok {
			v := "true"
			if !a.Bare {
				v = strconv.Quote(a.Value)
			}
			attrs = append(attrs, strconv.Quote(a.Name)+": "+v)
			continue
		}
		var v string
		var err error
		if f.Kind == KindChoice {
			v = choiceLiteral(a.Value, n.has("multiple"))
		} else {
			v, err = f.literal(a, qual)
		}
		if err != nil {
			return "", fmt.Errorf("<%s %s>: %w", c.Tag, a.Name, err)
		}
		if (f.Kind == KindNumber || f.Kind == KindInt) && v == "0" || (f.Kind == KindString || f.Kind == KindURL) && v == `""` {
			continue // the zero value: what an absent attribute means
		}
		fields = append(fields, f.GoName+": "+v)
	}
	if len(attrs) > 0 {
		fields = append(fields, "Attrs: templ.Attributes{"+strings.Join(attrs, ", ")+"}")
	}
	lit := qual + c.GoName + "Props{" + strings.Join(fields, ", ") + "}"
	if len(lit) > 72 && len(fields) > 1 {
		lit = qual + c.GoName + "Props{\n" + strings.Join(fields, ",\n") + ",\n}"
	}
	return gofmtExpr(lit)
}

// gofmtExpr formats a Go expression.
func gofmtExpr(expr string) (string, error) {
	const pre = "package p\n\nvar _ = "
	b, err := format.Source([]byte(pre + expr + "\n"))
	if err != nil {
		return "", fmt.Errorf("%w in %s", err, expr)
	}
	return strings.TrimSuffix(strings.TrimPrefix(string(b), pre), "\n"), nil
}

// literal is the Go value of the field for the attribute a.
func (f *Field) literal(a Attr, qual string) (string, error) {
	switch f.Kind {
	case KindString, KindURL:
		return strconv.Quote(a.Value), nil
	case KindStringPtr:
		return qual + "Str(" + strconv.Quote(a.Value) + ")", nil
	case KindBool:
		return "true", nil // present, whatever its value
	case KindFlag:
		if a.Value == "false" {
			return qual + "Off", nil
		}
		return qual + "On", nil
	case KindEnum:
		for _, e := range f.Enum {
			if e.Value == a.Value {
				return qual + e.GoName, nil
			}
		}
		return "", fmt.Errorf("%q is none of its values", a.Value)
	case KindNumber, KindNumberPtr:
		n, err := strconv.ParseFloat(strings.TrimSpace(a.Value), 64)
		if err != nil {
			return "", err
		}
		s := strconv.FormatFloat(n, 'f', -1, 64)
		if f.Kind == KindNumberPtr {
			return qual + "Num(" + s + ")", nil
		}
		return s, nil
	case KindInt, KindIntPtr:
		n, err := strconv.ParseFloat(strings.TrimSpace(a.Value), 64)
		if err != nil {
			return "", err
		}
		if n != float64(int(n)) {
			return "", fmt.Errorf("%v is not a whole number", n)
		}
		s := strconv.Itoa(int(n))
		if f.Kind == KindIntPtr {
			return qual + "Int(" + s + ")", nil
		}
		return s, nil
	case KindWords:
		return stringsLiteral(strings.Fields(a.Value)), nil
	case KindJSON:
		v := reflect.New(f.JSON)
		dec := json.NewDecoder(strings.NewReader(a.Value))
		dec.DisallowUnknownFields()
		if err := dec.Decode(v.Interface()); err != nil {
			return "", fmt.Errorf("as %s: %w", f.GoType, err)
		}
		// A key the Go type has no field for (a custom UnmarshalJSON lets it
		// through) would be lost: refuse it.
		var orig, back any
		b, err := json.Marshal(v.Interface())
		if err != nil {
			return "", err
		}
		json.Unmarshal([]byte(a.Value), &orig)
		json.Unmarshal(b, &back)
		if err := lostKeys(orig, back, f.Prop.Attribute); err != nil {
			return "", err
		}
		return GoValue(v.Elem(), qual)
	}
	return "", fmt.Errorf("unknown kind %d", f.Kind)
}

// choiceLiteral is sb-select's value as the component reads it: a JSON
// array, with multiple a list separated by commas, else one value.
func choiceLiteral(v string, multiple bool) string {
	s := strings.TrimSpace(v)
	var list []string
	switch {
	case s == "":
		list = []string{}
	case strings.HasPrefix(s, "[") && json.Unmarshal([]byte(s), &list) == nil:
	case multiple:
		list = []string{}
		for _, x := range strings.Split(s, ",") {
			if x = strings.TrimSpace(x); x != "" {
				list = append(list, x)
			}
		}
	default:
		list = []string{s}
	}
	return stringsLiteral(list)
}

func stringsLiteral(list []string) string {
	q := make([]string, len(list))
	for i, s := range list {
		q[i] = strconv.Quote(s)
	}
	return "[]string{" + strings.Join(q, ", ") + "}"
}

// lostKeys reports a key of orig that back lacks, unless its value is one
// omitempty leaves out.
func lostKeys(orig, back any, path string) error {
	switch o := orig.(type) {
	case map[string]any:
		b, ok := back.(map[string]any)
		if !ok {
			return nil
		}
		for k, v := range o {
			bv, ok := b[k]
			if !ok {
				if v == nil || v == false || v == "" || v == 0.0 {
					continue
				}
				return fmt.Errorf("%s.%s: no Go field for it", path, k)
			}
			if err := lostKeys(v, bv, path+"."+k); err != nil {
				return err
			}
		}
	case []any:
		if b, ok := back.([]any); ok && len(b) == len(o) {
			for i := range o {
				if err := lostKeys(o[i], b[i], fmt.Sprintf("%s[%d]", path, i)); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// GoValue writes v as a Go expression, sb's names qualified by qual.
func GoValue(v reflect.Value, qual string) (string, error) {
	return goValue(v, qual, true)
}

func goValue(v reflect.Value, qual string, typed bool) (string, error) {
	t := v.Type()
	if t == reflect.TypeFor[sb.Flag]() {
		switch v.Interface().(sb.Flag) {
		case sb.On:
			return qual + "On", nil
		case sb.Off:
			return qual + "Off", nil
		}
		return qual + "Default", nil
	}
	switch t.Kind() {
	case reflect.Interface:
		if v.IsNil() {
			return "nil", nil
		}
		return goValue(v.Elem(), qual, true)
	case reflect.Pointer:
		if v.IsNil() {
			return "nil", nil
		}
		if t.Elem().Kind() == reflect.Float64 {
			return qual + "Num(" + strconv.FormatFloat(v.Elem().Float(), 'f', -1, 64) + ")", nil
		}
		if t.Elem().Kind() == reflect.Int {
			return qual + "Int(" + strconv.FormatInt(v.Elem().Int(), 10) + ")", nil
		}
		s, err := goValue(v.Elem(), qual, true)
		return "&" + s, err
	case reflect.String:
		if names, ok := constNames[t]; ok {
			if n, ok := names[v.String()]; ok {
				return qual + n, nil
			}
		}
		if t.PkgPath() != "" {
			return GoType(t, qual) + "(" + strconv.Quote(v.String()) + ")", nil
		}
		return strconv.Quote(v.String()), nil
	case reflect.Bool:
		return strconv.FormatBool(v.Bool()), nil
	case reflect.Int, reflect.Int64, reflect.Int32:
		return strconv.FormatInt(v.Int(), 10), nil
	case reflect.Float64:
		return strconv.FormatFloat(v.Float(), 'f', -1, 64), nil
	case reflect.Slice:
		if v.IsNil() {
			return "nil", nil
		}
		var items []string
		for i := range v.Len() {
			s, err := goValue(v.Index(i), qual, t.Elem().Kind() == reflect.Interface)
			if err != nil {
				return "", err
			}
			items = append(items, s)
		}
		return composite(GoType(t, qual), items, typed, t.Elem()), nil
	case reflect.Map:
		if v.IsNil() {
			return "nil", nil
		}
		keys := v.MapKeys()
		sort.Slice(keys, func(i, j int) bool { return keys[i].String() < keys[j].String() })
		var items []string
		for _, k := range keys {
			s, err := goValue(v.MapIndex(k), qual, t.Elem().Kind() == reflect.Interface)
			if err != nil {
				return "", err
			}
			items = append(items, strconv.Quote(k.String())+": "+s)
		}
		return composite(GoType(t, qual), items, typed, t.Elem()), nil
	case reflect.Struct:
		var items []string
		for i := range t.NumField() {
			fv := v.Field(i)
			if fv.IsZero() {
				continue
			}
			s, err := goValue(fv, qual, true)
			if err != nil {
				return "", err
			}
			items = append(items, t.Field(i).Name+": "+s)
		}
		return composite(GoType(t, qual), items, typed, nil), nil
	}
	return "", fmt.Errorf("no Go literal for %s", t)
}

// composite writes Type{items}, without the type where Go elides it, one
// item per line when they are composites themselves.
func composite(typ string, items []string, typed bool, elem reflect.Type) string {
	if !typed {
		typ = ""
	}
	multi := len(items) > 1 && elem != nil && (elem.Kind() == reflect.Struct || elem.Kind() == reflect.Map || elem.Kind() == reflect.Slice) && len(strings.Join(items, ", ")) > 60
	if multi {
		return typ + "{\n" + strings.Join(items, ",\n") + ",\n}"
	}
	return typ + "{" + strings.Join(items, ", ") + "}"
}
