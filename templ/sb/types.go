package sb

import "encoding/json"

// Option is a choice of sb-select, sb-radio-group or sb-checkbox-group.
// One with only a Value is written as a plain string ("Mars"), which the
// components read as that value and label.
type Option struct {
	Value       string `json:"value"`
	Label       string `json:"label,omitempty"`
	Description string `json:"description,omitempty"`
	Disabled    bool   `json:"disabled,omitempty"`
}

// MarshalJSON writes an Option with only a Value as a string.
func (o Option) MarshalJSON() ([]byte, error) {
	if o.Label == "" && o.Description == "" && !o.Disabled {
		return json.Marshal(o.Value)
	}
	type plain Option
	return json.Marshal(plain(o))
}

// UnmarshalJSON reads both forms: "Mars" and {"value": "Mars", …}.
func (o *Option) UnmarshalJSON(b []byte) error {
	if s, ok := scalar(b); ok {
		*o = Option{Value: s}
		return nil
	}
	type plain Option
	return json.Unmarshal(b, (*plain)(o))
}

// DropdownItem is one entry of sb-dropdown's items: an action (Value), a
// link (Href), a divider, a group heading (Group: the items after it, up to
// the next heading or divider, are its group) or a submenu (Label and
// Children, nested up to 5 levels deep). A submenu with Type
// DropdownItemTypeRadio is the dropdown's one radio group: its children are
// the choices, and the dropdown's Value is the checked one.
type DropdownItem struct {
	Value       string `json:"value,omitempty"`
	Label       string `json:"label,omitempty"`
	Description string `json:"description,omitempty"`
	// Icon is text, such as an emoji (rich icons are the markup form's).
	Icon string `json:"icon,omitempty"`
	// Href makes the item a link the browser follows (no sb-select),
	// sanitized like templ's href.
	Href     string         `json:"href,omitempty"`
	Disabled bool           `json:"disabled,omitempty"`
	Danger   bool           `json:"danger,omitempty"`
	Divider  bool           `json:"divider,omitempty"`
	Group    string         `json:"group,omitempty"`
	Children []DropdownItem `json:"children,omitempty"`
	// Type makes a submenu's children a radio group.
	Type DropdownItemType `json:"type,omitempty"`
}

// DropdownItemType is a submenu's kind; the zero value is a menu of
// actions.
type DropdownItemType string

const DropdownItemTypeRadio DropdownItemType = "radio"

// MarshalJSON writes the item with its Href sanitized like templ's href: a
// scheme other than http, https, mailto, tel, ftp and ftps becomes
// about:invalid#TemplFailedSanitizationURL.
func (d DropdownItem) MarshalJSON() ([]byte, error) {
	type plain DropdownItem
	d.Href = sanitizeURL(d.Href)
	return json.Marshal(plain(d))
}

// UnmarshalJSON also reads the string forms: "Rename" is {Value: "Rename"},
// "-" and "---" are dividers.
func (d *DropdownItem) UnmarshalJSON(b []byte) error {
	if s, ok := scalar(b); ok {
		if s == "-" || s == "---" {
			*d = DropdownItem{Divider: true}
		} else {
			*d = DropdownItem{Value: s}
		}
		return nil
	}
	type plain DropdownItem
	return json.Unmarshal(b, (*plain)(d))
}

// scalar reads a JSON string or number as a string.
func scalar(b []byte) (string, bool) {
	var s string
	if json.Unmarshal(b, &s) == nil {
		return s, true
	}
	var n json.Number
	if json.Unmarshal(b, &n) == nil {
		return n.String(), true
	}
	return "", false
}

// ColumnAlign aligns a column of sb-data-table.
type ColumnAlign string

const (
	ColumnAlignStart  ColumnAlign = "start"
	ColumnAlignCenter ColumnAlign = "center"
	ColumnAlignEnd    ColumnAlign = "end"
)

// Column is a column of sb-data-table.
type Column struct {
	// Key names the column's value in each row.
	Key   string `json:"key"`
	Label string `json:"label,omitempty"`
	// Width is a CSS grid track, such as "12rem" or "180px" (default
	// minmax(6rem, 1fr)).
	Width    string      `json:"width,omitempty"`
	Align    ColumnAlign `json:"align,omitempty"`
	Sortable bool        `json:"sortable,omitempty"`
	// Resizable is on by default (with the table's resizable): Off keeps
	// the column's width.
	Resizable Flag `json:"resizable,omitempty"`
	// Min is the narrowest the user can make it, in px (default 40).
	Min float64 `json:"min,omitempty"`
	// Fit is its width for the whole result, in px.
	Fit float64 `json:"fit,omitempty"`
}

// CellTone shows a sb-data-table cell as a badge.
type CellTone string

const (
	CellToneInfo    CellTone = "info"
	CellToneSuccess CellTone = "success"
	CellToneWarning CellTone = "warning"
	CellToneDanger  CellTone = "danger"
	CellToneNeutral CellTone = "neutral"
)

// Cell is a sb-data-table value with more than a scalar: a row's value for
// a key is a scalar (string, number, bool) or a Cell. Nothing is rendered
// as HTML.
type Cell struct {
	// Value is the sort key, and what is shown without Text.
	Value  any    `json:"value"`
	Text   string `json:"text,omitempty"`
	Suffix string `json:"suffix,omitempty"`
	// Href (http, https or mailto) makes the cell a link; sanitized like
	// templ's href.
	Href string   `json:"href,omitempty"`
	Tone CellTone `json:"tone,omitempty"`
}

// MarshalJSON writes the cell with its Href sanitized.
func (c Cell) MarshalJSON() ([]byte, error) {
	type plain Cell
	c.Href = sanitizeURL(c.Href)
	return json.Marshal(plain(c))
}

// SortDir is the direction of a TableSort.
type SortDir string

const (
	SortDirAsc  SortDir = "asc"
	SortDirDesc SortDir = "desc"
)

// TableSort is the order of sb-data-table's rows.
type TableSort struct {
	Key string  `json:"key"`
	Dir SortDir `json:"dir"`
}

// RangeValue is sb-range's value.
type RangeValue struct {
	Start float64 `json:"start"`
	End   float64 `json:"end"`
}

// Diagnostic is a problem sb-code-editor underlines. Line and Col are
// 1-based; Col and Length count UTF-16 code units (a JavaScript string
// index).
type Diagnostic struct {
	Line    int    `json:"line"`
	Col     int    `json:"col"`
	Length  int    `json:"length"`
	Message string `json:"message"`
	// Text, when set, shows the problem only while the code there still
	// reads Text, so a list about an older version fades as the code changes.
	Text string `json:"text,omitempty"`
}

// TreeItem is an item of sb-tree's items.
type TreeItem struct {
	ID       string     `json:"id"`
	Label    string     `json:"label,omitempty"`
	Icon     string     `json:"icon,omitempty"`
	Children []TreeItem `json:"children,omitempty"`
	// Lazy: an item without Children asks for them with sb-load when opened.
	Lazy bool `json:"lazy,omitempty"`
}

// ToastVariant is a toast's tone.
type ToastVariant string

const (
	ToastVariantInfo   ToastVariant = "info"
	ToastVariantOK     ToastVariant = "ok"
	ToastVariantWarn   ToastVariant = "warn"
	ToastVariantDanger ToastVariant = "danger"
)

// ToastMessage is one notification of sb-toast.
type ToastMessage struct {
	ID      string       `json:"id"`
	Title   string       `json:"title,omitempty"`
	Text    string       `json:"text"`
	Variant ToastVariant `json:"variant,omitempty"`
	// Duration before it dismisses itself, in ms, over the element's
	// duration; Int(0) makes it sticky.
	Duration *int `json:"duration,omitempty"`
}
