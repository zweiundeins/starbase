package catalog

import (
	"encoding/json"
	"strings"

	"github.com/evanw/esbuild/pkg/api"
)

// Transpiled is TypeScript turned into JavaScript. Lines[i] is the line of
// the TypeScript that line i+1 of Code comes from: esbuild prints the code
// anew, so errors and stack lines need it to point at the code as written.
type Transpiled struct {
	Code  string `json:"code"`
	Lines []int  `json:"lines"`
}

// TranspileError is a syntax error in the TypeScript, at its line.
type TranspileError struct {
	Message string `json:"error"`
	Line    int    `json:"line"`
}

func (e *TranspileError) Error() string { return e.Message }

// Transpile strips the types from a TypeScript module (esbuild, no
// bundling, the syntax kept as it is).
func Transpile(src string) (Transpiled, error) {
	res := api.Transform(src, api.TransformOptions{
		Loader:    api.LoaderTS,
		Format:    api.FormatESModule,
		Target:    api.ESNext,
		Charset:   api.CharsetUTF8,
		Sourcemap: api.SourceMapExternal,
	})
	if len(res.Errors) > 0 {
		e := &TranspileError{Message: res.Errors[0].Text}
		if l := res.Errors[0].Location; l != nil {
			e.Line = l.Line
		}
		return Transpiled{}, e
	}
	code := tsImports(string(res.Code))
	return Transpiled{Code: code, Lines: sourceLines(string(res.Map), strings.Count(code, "\n")+1)}, nil
}

// tsImports points relative imports of TypeScript modules at the JavaScript
// they are compiled to, as tsc does (./util.ts becomes ./util.js).
func tsImports(code string) string {
	var at []int // where each specifier's ".ts" starts
	for _, re := range importRes {
		for _, m := range re.FindAllStringSubmatchIndex(code, -1) {
			spec := code[m[4]:m[5]]
			if (strings.HasPrefix(spec, "./") || strings.HasPrefix(spec, "../")) && strings.HasSuffix(spec, ".ts") && !strings.HasSuffix(spec, ".d.ts") {
				at = append(at, m[5]-len(".ts"))
			}
		}
	}
	b := []byte(code)
	for _, i := range at {
		copy(b[i:], ".js")
	}
	return string(b)
}

// sourceLines reads a source map's mappings: for each of n generated lines,
// the 1-based source line of its first segment (or of the line before).
func sourceLines(sourceMap string, n int) []int {
	var m struct{ Mappings string }
	json.Unmarshal([]byte(sourceMap), &m)
	mappings := m.Mappings
	lines := make([]int, n)
	src, last := 0, 1
	for i, line := range strings.Split(mappings, ";") {
		first := true
		for _, seg := range strings.Split(line, ",") {
			if v := vlq(seg); len(v) >= 4 {
				src += v[2] // relative to the previous segment, across lines
				if first {
					last, first = src+1, false
				}
			}
		}
		if i < n {
			lines[i] = last
		}
	}
	for i := len(strings.Split(mappings, ";")); i < n; i++ {
		lines[i] = last
	}
	return lines
}

// vlq decodes one segment of a source map's mappings.
func vlq(s string) []int {
	const digits = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"
	var out []int
	v, shift := 0, 0
	for _, c := range s {
		d := strings.IndexRune(digits, c)
		if d < 0 {
			return nil
		}
		v += (d & 31) << shift
		if d&32 != 0 {
			shift += 5
			continue
		}
		if v&1 != 0 {
			v = -(v >> 1)
		} else {
			v >>= 1
		}
		out = append(out, v)
		v, shift = 0, 0
	}
	return out
}
