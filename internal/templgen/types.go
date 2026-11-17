package templgen

import (
	"reflect"

	"github.com/zweiundeins/starbase/templ/sb"
)

// sbPath is the import path of package sb.
const sbPath = "github.com/zweiundeins/starbase/templ/sb"

// jsonTypes are the Go types of the json, array and custom props, by
// "<tag>.<attribute>": the shapes their docs describe. A new json prop needs
// its type here (and a new shape its type in templ/sb/types.go).
var jsonTypes = map[string]reflect.Type{
	"sb-autoloader.modules":          reflect.TypeFor[map[string]string](),
	"sb-autoloader.requires":         reflect.TypeFor[map[string][]string](),
	"sb-checkbox-group.value":        reflect.TypeFor[[]string](),
	"sb-checkbox-group.options":      reflect.TypeFor[[]sb.Option](),
	"sb-code-editor.diagnostics":     reflect.TypeFor[[]sb.Diagnostic](),
	"sb-code-playground.deps":        reflect.TypeFor[map[string]string](),
	"sb-code-playground.themes":      reflect.TypeFor[[]string](),
	"sb-code-playground.initial":     reflect.TypeFor[map[string]string](),
	"sb-code-playground.diagnostics": reflect.TypeFor[map[string][]sb.Diagnostic](),
	"sb-data-table.columns":          reflect.TypeFor[[]sb.Column](),
	"sb-data-table.rows":             reflect.TypeFor[[]map[string]any](), // keyed by the columns' keys; a value is a scalar or a Cell
	"sb-data-table.sort":             reflect.TypeFor[*sb.TableSort](),
	"sb-data-table.selected":         reflect.TypeFor[[]string](),
	"sb-data-table.hidden-columns":   reflect.TypeFor[[]string](),
	"sb-data-table.widths":           reflect.TypeFor[map[string]float64](),
	"sb-date-picker.disabled-dates":  reflect.TypeFor[[]string](),
	"sb-dropdown.items":              reflect.TypeFor[[]sb.DropdownItem](),
	"sb-echarts.option":              reflect.TypeFor[any](), // an ECharts option: any shape ECharts takes
	"sb-pixel-board.palette":         reflect.TypeFor[[]string](),
	"sb-radio-group.options":         reflect.TypeFor[[]sb.Option](),
	"sb-range.value":                 reflect.TypeFor[*sb.RangeValue](),
	"sb-select.options":              reflect.TypeFor[[]sb.Option](),
	"sb-select.results":              reflect.TypeFor[[]sb.Option](),
	"sb-sparkline.values":            reflect.TypeFor[[]float64](),
	"sb-tabs.labels":                 reflect.TypeFor[[]string](),
	"sb-theme-switch.themes":         reflect.TypeFor[[]string](),
	"sb-theme-switch.labels":         reflect.TypeFor[[]string](),
	"sb-toast.toasts":                reflect.TypeFor[[]sb.ToastMessage](),
	"sb-tree.items":                  reflect.TypeFor[[]sb.TreeItem](),
	"sb-tree.loaded":                 reflect.TypeFor[map[string][]sb.TreeItem](),
	"sb-typewriter.phrases":          reflect.TypeFor[[]string](),
}

// serverWord are the string and number props whose attribute is the
// server's word over a local edit, and whose removal the component ignores
// (a morph strips attributes that were only reflected): to clear one, the
// server sends value="" or value="0". Their fields are pointers, so an
// explicit "" or 0 is written while nil leaves the attribute out.
var serverWord = map[string]bool{
	"sb-code-editor.value":      true,
	"sb-date-picker.value":      true,
	"sb-date-picker.month":      true,
	"sb-dropdown.value":         true,
	"sb-image-compare.position": true,
	"sb-input.value":            true,
	"sb-input.rev":              true,
	"sb-radio-group.value":      true,
	"sb-rating.value":           true,
	"sb-select.value":           true,
	"sb-slider.value":           true,
	"sb-sparkline.value":        true,
	"sb-tabs.selected":          true,
	"sb-tree.value":             true,
	"sb-tree.expanded":          true,
}

// presence are string and number props whose attribute means something
// by being there, even empty or 0: their fields are pointers too.
var presence = map[string]bool{
	"sb-checkbox-group.select-all": true, // present and empty: "All"
	"sb-starfield.label":           true, // "" makes it decorative
	"sb-virtual-scroll.total":      true, // absent: not known yet; 0: an empty list
}

// enumNames name oneOf values that make no Go name of their own, by
// "<tag>.<attribute>=<value>".
var enumNames = map[string]string{
	"sb-rating.precision=1":   "Whole",
	"sb-rating.precision=0.5": "Half",
}

// constNames are the constants of the named string types in package sb's
// JSON shapes, for writing their values as Go.
var constNames = map[reflect.Type]map[string]string{
	reflect.TypeFor[sb.ColumnAlign]():  {"start": "ColumnAlignStart", "center": "ColumnAlignCenter", "end": "ColumnAlignEnd"},
	reflect.TypeFor[sb.CellTone]():     {"info": "CellToneInfo", "success": "CellToneSuccess", "warning": "CellToneWarning", "danger": "CellToneDanger", "neutral": "CellToneNeutral"},
	reflect.TypeFor[sb.SortDir]():      {"asc": "SortAsc", "desc": "SortDesc"},
	reflect.TypeFor[sb.ToastVariant](): {"info": "ToastInfo", "ok": "ToastOK", "warn": "ToastWarn", "danger": "ToastDanger"},
}
