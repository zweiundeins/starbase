package sb

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/a-h/templ"
	templruntime "github.com/a-h/templ/runtime"
)

// Tag names a component, for ImportsProps.Components: TagToggle is
// "sb-toggle".
type Tag string

// Flag is a prop that is on by default, such as sb-meter's show-value. Its
// zero value, Default, leaves the attribute out, so the component's default
// (on) applies; On writes x="true" and Off writes x="false".
type Flag uint8

const (
	Default Flag = iota // the component's default: on
	On                  // x="true"
	Off                 // x="false"
)

// String is the attribute value: "", "true" or "false".
func (f Flag) String() string {
	switch f {
	case On:
		return "true"
	case Off:
		return "false"
	}
	return ""
}

// MarshalJSON writes On as true, Off as false and Default as null. Struct
// fields of type Flag carry omitempty, so Default leaves the key out.
func (f Flag) MarshalJSON() ([]byte, error) {
	switch f {
	case On:
		return []byte("true"), nil
	case Off:
		return []byte("false"), nil
	}
	return []byte("null"), nil
}

// UnmarshalJSON reads true, false and null.
func (f *Flag) UnmarshalJSON(b []byte) error {
	switch string(b) {
	case "true":
		*f = On
	case "false":
		*f = Off
	case "null":
		*f = Default
	default:
		return fmt.Errorf("sb: a Flag is true, false or null, not %s", b)
	}
	return nil
}

// Num returns a pointer to v, for the number fields whose zero value would
// mean something: Max: sb.Num(10).
func Num(v float64) *float64 { return &v }

// Str returns a pointer to s, for the string fields where an empty string is
// a value of its own (the server's word): Value: sb.Str("").
func Str(s string) *string { return &s }

// element collects the attributes of one component's host element, in
// order, and renders it with the templ children around the content.
type element struct {
	tag   string
	attrs []attr
	extra templ.Attributes
	err   error
}

type attr struct {
	name, value string
	bare        bool
}

func newElement(tag, id, class string, extra templ.Attributes) *element {
	e := &element{tag: tag, extra: extra}
	e.str("id", id)
	e.str("class", class)
	return e
}

// str writes a string attribute; empty leaves it out.
func (e *element) str(name, v string) {
	if v != "" {
		e.attrs = append(e.attrs, attr{name: name, value: v})
	}
}

// url writes a URL attribute sanitized like templ's href: a scheme other
// than http, https, mailto, tel, ftp and ftps becomes
// about:invalid#TemplFailedSanitizationURL.
func (e *element) url(name, v string) {
	e.str(name, sanitizeURL(v))
}

func sanitizeURL(v string) string {
	if v == "" {
		return ""
	}
	return string(templ.URL(v))
}

// strPtr writes a string attribute when v is set, even to "".
func (e *element) strPtr(name string, v *string) {
	if v != nil {
		e.attrs = append(e.attrs, attr{name: name, value: *v})
	}
}

// bool writes a boolean attribute by presence: true is the bare name.
func (e *element) bool(name string, v bool) {
	if v {
		e.attrs = append(e.attrs, attr{name: name, bare: true})
	}
}

func (e *element) flag(name string, v Flag) {
	if v == On || v == Off {
		e.attrs = append(e.attrs, attr{name: name, value: v.String()})
	} else if v != Default {
		e.fail(fmt.Errorf("sb: <%s> %s: Flag(%d) is none of Default, On and Off", e.tag, name, v))
	}
}

// num writes a number whose zero value is the component's default.
func (e *element) num(name string, v float64) {
	if v != 0 {
		e.attrs = append(e.attrs, attr{name: name, value: formatNum(v)})
	}
}

func (e *element) numPtr(name string, v *float64) {
	if v != nil {
		e.attrs = append(e.attrs, attr{name: name, value: formatNum(*v)})
	}
}

// enum writes a oneOf value, checked against the values the prop allows.
func (e *element) enum(name, v string, allowed ...string) {
	if v == "" {
		return
	}
	for _, a := range allowed {
		if v == a {
			e.attrs = append(e.attrs, attr{name: name, value: v})
			return
		}
	}
	e.fail(fmt.Errorf("sb: <%s> %s=%q is none of %s", e.tag, name, v, strings.Join(allowed, ", ")))
}

// json writes v as JSON; nil (a nil slice, map or pointer) leaves it out.
func (e *element) json(name string, v any, isNil bool) {
	if isNil {
		return
	}
	b, err := json.Marshal(v)
	if err != nil {
		e.fail(fmt.Errorf("sb: <%s> %s: %w", e.tag, name, err))
		return
	}
	e.attrs = append(e.attrs, attr{name: name, value: string(b)})
}

func (e *element) fail(err error) {
	if e.err == nil {
		e.err = err
	}
}

func formatNum(v float64) string { return strconv.FormatFloat(v, 'f', -1, 64) }

// render writes <tag …>children</tag>. An attribute in Attrs wins over the
// field that writes the same attribute (names compare without case).
func (e *element) render(ctx context.Context, w io.Writer) error {
	if e.err != nil {
		return e.err
	}
	var open, scripts bytes.Buffer
	open.WriteString("<" + e.tag)
	for _, a := range e.attrs {
		if e.overridden(a.name) {
			continue
		}
		open.WriteString(" " + a.name)
		if !a.bare {
			open.WriteString(`="` + templ.EscapeString(a.value) + `"`)
		}
	}
	if len(e.extra) > 0 {
		if err := templruntime.RenderAttributes(ctx, &open, &scripts, e.tag, e.extra); err != nil {
			return err
		}
	}
	open.WriteString(">")
	if _, err := w.Write(scripts.Bytes()); err != nil {
		return err
	}
	if _, err := w.Write(open.Bytes()); err != nil {
		return err
	}
	children := templ.GetChildren(ctx)
	if err := children.Render(templ.ClearChildren(ctx), w); err != nil {
		return err
	}
	_, err := io.WriteString(w, "</"+e.tag+">")
	return err
}

func (e *element) overridden(name string) bool {
	for k := range e.extra {
		if strings.EqualFold(k, name) {
			return true
		}
	}
	return false
}
