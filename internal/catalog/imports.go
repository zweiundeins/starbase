package catalog

import (
	"regexp"
	"slices"
)

// Module specifiers in static imports (`import x from '…'`, `import '…'`),
// re-exports (`export … from '…'`) and dynamic imports (`import('…')`).
var importRes = []*regexp.Regexp{
	regexp.MustCompile(`(?m)(?:^|[;\s}])import\s*(?:[\w$*{}\s,]+?\s*from\s*)?(['"])([^'"\n]+)(['"])`),
	regexp.MustCompile(`(?m)(?:^|[;\s}])export\s*(?:\*(?:\s*as\s+[\w$]+)?|\{[^}]*\})\s*from\s*(['"])([^'"\n]+)(['"])`),
	regexp.MustCompile(`\bimport\s*\(\s*(['"])([^'"\n]+)(['"])\s*\)`),
}

// Imports lists the module specifiers code imports, in order, without duplicates.
func Imports(code string) []string {
	var out []string
	for _, re := range importRes {
		for _, m := range re.FindAllStringSubmatch(code, -1) {
			if m[1] == m[3] && !slices.Contains(out, m[2]) {
				out = append(out, m[2])
			}
		}
	}
	return out
}
