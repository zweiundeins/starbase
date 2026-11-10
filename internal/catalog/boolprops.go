package catalog

import (
	"regexp"
	"strings"
)

// boolDefaultRe finds `bool.default(`, which official Rocket (v1.0.5 and
// later) refuses when the component is defined: a bool prop is read by
// presence, so its absence is always false.
var boolDefaultRe = regexp.MustCompile(`\bbool\s*\.\s*default\s*\(`)

// BoolDefaults returns the lines (1-based) of code that declare a bool prop
// with a default. A flag that is on by default is `oneOf(true, false)`.
func BoolDefaults(code string) []int {
	var lines []int
	for _, m := range boolDefaultRe.FindAllStringIndex(code, -1) {
		lines = append(lines, strings.Count(code[:m[0]], "\n")+1)
	}
	return lines
}
