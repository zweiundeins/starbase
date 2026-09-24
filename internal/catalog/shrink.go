package catalog

import (
	"fmt"
	"slices"
	"strings"
)

// Shrinking: what esbuild can't take out of a component module, because it
// never looks inside template literals or at what a call is for. Applied to
// the source before minifying, for the .min files and the bundle, never to
// vendored files:
//
//   - .docs({…}) calls and the rocket() manifest: {…} property. They only
//     describe props and events for manifest.json, which is generated from
//     the readable modules (see web.assets.AllComponents).
//   - Comments and whitespace in /* css */ templates, one quasi at a time,
//     so an interpolation is never touched.
//   - Newline runs in html`` and svg`` templates, collapsed to one space
//     (never dropped, so text between tags keeps its separation), and
//     <!-- --> comments. <pre>, <textarea>, <script> and <style> content and
//     attribute values are left as they are.
//
// When the scanner doesn't understand a file, the file ships unshrunk.
//
// The shrunk bytes are part of each version's frozen .min files, so a
// change to what shrinking does must bump minFormat (catalog.go), which
// gives every component a new version.

// shrink returns src with the above removed, or src itself when the file is
// vendored or can't be scanned.
func shrink(name string, src []byte) []byte {
	if strings.HasPrefix(name, "vendor/") || strings.Contains(name, "/vendor/") {
		return src
	}
	s := string(src)
	r, err := shScan(s)
	if err != nil {
		return src
	}
	var edits []shEdit
	for _, d := range r.docs {
		edits = append(edits, shEdit{d[0], d[1], ""})
	}
	for _, m := range r.manifests {
		edits = append(edits, shEdit{m[0], m[1], ""})
	}
	for _, t := range r.templates {
		switch {
		case t.css:
			edits = append(edits, shCSS(s, t)...)
		case t.tag == "html" || t.tag == "svg":
			edits = append(edits, shHTML(s, t)...)
		}
	}
	return []byte(shApply(s, edits))
}

type shEdit struct {
	start, end int
	repl       string
}

func shApply(src string, edits []shEdit) string {
	slices.SortFunc(edits, func(a, b shEdit) int { return a.start - b.start })
	var b strings.Builder
	pos := 0
	for _, e := range edits {
		if e.start < pos { // inside an earlier deletion (a template inside a removed .docs())
			continue
		}
		b.WriteString(src[pos:e.start])
		b.WriteString(e.repl)
		pos = e.end
	}
	b.WriteString(src[pos:])
	return b.String()
}

// --- scanner -----------------------------------------------------------------

// A deliberately small JavaScript scanner: enough to find template literals
// (with their tag, or a /* css */ comment right before them), .docs(…) calls
// and manifest: {…} properties, while skipping strings, comments and regular
// expressions.

type shTemplate struct {
	tag    string   // identifier right before the backtick ("html", "svg")
	css    bool     // preceded by /* css */
	quasis [][2]int // the literal text between interpolations, [start, end)
}

type shResult struct {
	templates []*shTemplate
	docs      [][2]int // from the '.' to after the ')'
	manifests [][2]int // from the key to after the '}' (and a trailing comma)
}

const (
	shNone = iota
	shDocs
	shManifest
)

type shFrame struct {
	kind   byte // '(', '[', '{', or '$' for a template interpolation
	t      *shTemplate
	mark   int // shDocs or shManifest: this bracket closes a match
	mstart int
}

var shRegexKeywords = map[string]bool{"return": true, "typeof": true, "case": true, "do": true, "else": true, "in": true, "of": true, "new": true, "delete": true, "void": true, "throw": true, "instanceof": true, "yield": true, "await": true}

func shIdentStart(c byte) bool {
	return c == '_' || c == '$' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= 0x80
}
func shIdent(c byte) bool { return shIdentStart(c) || c >= '0' && c <= '9' }
func shSpace(c byte) bool { return c == ' ' || c == '\t' || c == '\n' || c == '\r' }

func shSkipSpace(s string, i int) int {
	for i < len(s) && shSpace(s[i]) {
		i++
	}
	return i
}

func shScan(src string) (*shResult, error) {
	res := &shResult{}
	var stack []shFrame
	prevKind, prevVal, prevEnd := "", "", -1 // the previous significant token
	lastComment, lastCommentEnd := "", -1
	lastDot := -1
	pendMark, pendPos, pendStart := shNone, -1, -1
	n := len(src)

	// template scans literal text from i up to the closing backtick or ${.
	template := func(t *shTemplate, i int) (int, error) {
		start := i
		for i < n {
			switch src[i] {
			case '\\':
				i += 2
				continue
			case '`':
				t.quasis = append(t.quasis, [2]int{start, i})
				prevKind, prevVal, prevEnd = "tmpl", "", i+1
				return i + 1, nil
			case '$':
				if i+1 < n && src[i+1] == '{' {
					t.quasis = append(t.quasis, [2]int{start, i})
					stack = append(stack, shFrame{kind: '$', t: t})
					prevKind, prevVal, prevEnd = "punct", "{", i+2
					return i + 2, nil
				}
			}
			i++
		}
		return i, fmt.Errorf("unterminated template")
	}

	for i := 0; i < n; {
		c := src[i]
		switch {
		case shSpace(c):
			i++
		case c == '/' && i+1 < n && src[i+1] == '/':
			for i < n && src[i] != '\n' {
				i++
			}
		case c == '/' && i+1 < n && src[i+1] == '*':
			e := strings.Index(src[i+2:], "*/")
			if e < 0 {
				return nil, fmt.Errorf("unterminated comment at %d", i)
			}
			lastComment, lastCommentEnd = src[i:i+e+4], i+e+4
			i = lastCommentEnd
		case c == '\'' || c == '"':
			j := i + 1
			for j < n && src[j] != c {
				if src[j] == '\\' {
					j++
				}
				j++
			}
			prevKind, prevVal, prevEnd = "str", "", j+1
			i = j + 1
		case c == '`':
			t := &shTemplate{}
			if prevKind == "id" && strings.TrimSpace(src[prevEnd:i]) == "" {
				t.tag = prevVal
			}
			if lastCommentEnd >= 0 && strings.TrimSpace(src[lastCommentEnd:i]) == "" && strings.TrimSpace(strings.Trim(lastComment, "/*")) == "css" {
				t.css = true
			}
			res.templates = append(res.templates, t)
			var err error
			if i, err = template(t, i+1); err != nil {
				return nil, err
			}
		case c == '/':
			regex := prevKind == "" || prevKind == "punct" && !strings.Contains(")]}", prevVal) || prevKind == "id" && shRegexKeywords[prevVal]
			if !regex {
				prevKind, prevVal, prevEnd = "punct", "/", i+1
				i++
				continue
			}
			j, class := i+1, false
			for j < n {
				switch {
				case src[j] == '\\':
					j++
				case src[j] == '[':
					class = true
				case src[j] == ']':
					class = false
				case src[j] == '/' && !class:
					goto end
				case src[j] == '\n':
					return nil, fmt.Errorf("bad regex at %d", i)
				}
				j++
			}
		end:
			j++
			for j < n && shIdent(src[j]) {
				j++
			}
			prevKind, prevVal, prevEnd = "re", "", j
			i = j
		case shIdentStart(c):
			j := i
			for j < n && shIdent(src[j]) {
				j++
			}
			word := src[i:j]
			k := shSkipSpace(src, j)
			if word == "docs" && prevKind == "punct" && prevVal == "." && k < n && src[k] == '(' {
				pendMark, pendPos, pendStart = shDocs, k, lastDot
			}
			if word == "manifest" && prevKind == "punct" && (prevVal == "{" || prevVal == ",") && k < n && src[k] == ':' {
				if b := shSkipSpace(src, k+1); b < n && src[b] == '{' {
					pendMark, pendPos, pendStart = shManifest, b, i
				}
			}
			prevKind, prevVal, prevEnd = "id", word, j
			i = j
		case c >= '0' && c <= '9':
			j := i
			for j < n && (shIdent(src[j]) || src[j] == '.') {
				j++
			}
			prevKind, prevVal, prevEnd = "num", "", j
			i = j
		case c == '(' || c == '[' || c == '{':
			f := shFrame{kind: c}
			if pendPos == i {
				f.mark, f.mstart = pendMark, pendStart
				pendMark, pendPos = shNone, -1
			}
			stack = append(stack, f)
			prevKind, prevVal, prevEnd = "punct", string(c), i+1
			i++
		case c == ')' || c == ']' || c == '}':
			if len(stack) == 0 {
				return nil, fmt.Errorf("unbalanced %c at %d", c, i)
			}
			f := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			if f.kind == '$' {
				if c != '}' {
					return nil, fmt.Errorf("bad close %c at %d", c, i)
				}
				var err error
				if i, err = template(f.t, i+1); err != nil {
					return nil, err
				}
				continue
			}
			if want := map[byte]byte{'(': ')', '[': ']', '{': '}'}[f.kind]; c != want {
				return nil, fmt.Errorf("mismatched %c at %d", c, i)
			}
			switch f.mark {
			case shDocs:
				res.docs = append(res.docs, [2]int{f.mstart, i + 1})
			case shManifest:
				end := i + 1
				if k := shSkipSpace(src, end); k < n && src[k] == ',' {
					end = k + 1
				}
				res.manifests = append(res.manifests, [2]int{f.mstart, end})
			}
			prevKind, prevVal, prevEnd = "punct", string(c), i+1
			i++
		default:
			if c == '.' {
				lastDot = i
			}
			prevKind, prevVal, prevEnd = "punct", string(c), i+1
			i++
		}
	}
	if len(stack) != 0 {
		return nil, fmt.Errorf("unclosed brackets")
	}
	return res, nil
}

// --- CSS ---------------------------------------------------------------------

// shCSS drops comments and needless whitespace from each quasi of a /* css */
// template: no space around { } ; or after : and , and no ; before }.
// Strings are copied as they are. At a quasi's edge (next to an
// interpolation) one space stays, since the interpolation may be a token.
func shCSS(src string, t *shTemplate) []shEdit {
	var edits []shEdit
	for _, q := range t.quasis {
		in := src[q[0]:q[1]]
		var b strings.Builder
		last := func() byte {
			if b.Len() == 0 {
				return 0
			}
			return b.String()[b.Len()-1]
		}
		for i := 0; i < len(in); {
			c := in[i]
			switch {
			case c == '/' && i+1 < len(in) && in[i+1] == '*':
				e := strings.Index(in[i+2:], "*/")
				if e < 0 {
					b.WriteString(in[i:])
					i = len(in)
					continue
				}
				i += e + 4
				// A comment separates tokens: keep a space unless one isn't needed.
				if l := last(); l != 0 && !strings.ContainsRune(" {};:,", rune(l)) {
					b.WriteByte(' ')
				}
			case c == '"' || c == '\'':
				j := i + 1
				for j < len(in) && in[j] != c {
					if in[j] == '\\' {
						j++
					}
					j++
				}
				b.WriteString(in[i:min(j+1, len(in))])
				i = j + 1
			case shSpace(c):
				j := i
				for j < len(in) && shSpace(in[j]) {
					j++
				}
				// One space stays unless punctuation makes it needless. At a quasi's
				// edge (l or next is 0) that keeps it next to the interpolation.
				l, next := last(), byte(0)
				if j < len(in) {
					next = in[j]
				}
				needless := l != 0 && strings.IndexByte("{};:, ", l) >= 0 || next != 0 && strings.IndexByte("{};", next) >= 0
				if !needless {
					b.WriteByte(' ')
				}
				i = j
			case c == ';':
				j := shSkipSpace(in, i+1)
				if j < len(in) && in[j] == '}' { // ;} → }
					i++
					continue
				}
				b.WriteByte(c)
				i++
			default:
				b.WriteByte(c)
				i++
			}
		}
		if out := b.String(); out != in {
			edits = append(edits, shEdit{q[0], q[1], out})
		}
	}
	return edits
}

// --- HTML --------------------------------------------------------------------

type shHTMLState int

const (
	shText shHTMLState = iota
	shTag
	shAttrDQ
	shAttrSQ
	shComment
	shRaw
)

var shRawTags = map[string]bool{"pre": true, "textarea": true, "script": true, "style": true}

// shHTML collapses newline runs in an html or svg template's text and tags to
// one space (whitespace before a tag's '>' goes), and drops <!-- --> comments.
func shHTML(src string, t *shTemplate) []shEdit {
	var edits []shEdit
	st, raw, pendingRaw := shText, "", ""
	for _, q := range t.quasis {
		i, end := q[0], q[1]
		for i < end {
			c := src[i]
			switch st {
			case shText:
				switch {
				case strings.HasPrefix(src[i:end], "<!--"):
					if e := strings.Index(src[i:end], "-->"); e >= 0 {
						j := i + e + 3
						// After whitespace, the whitespace that follows goes too, so the
						// two runs around the comment don't both stay as spaces.
						if i > q[0] && shSpace(src[i-1]) {
							for j < end && shSpace(src[j]) {
								j++
							}
						}
						edits = append(edits, shEdit{i, j, ""})
						i = j
					} else {
						st = shComment
						i += 4
					}
				case c == '<' && i+1 < end && (shIdentStart(src[i+1]) || src[i+1] == '/'):
					j := i + 1
					closing := src[j] == '/'
					if closing {
						j++
					}
					k := j
					for k < end && (shIdent(src[k]) || src[k] == '-') {
						k++
					}
					pendingRaw = ""
					if name := strings.ToLower(src[j:k]); !closing && shRawTags[name] {
						pendingRaw = name
					}
					st, i = shTag, k
				case shSpace(c):
					j := i
					for j < end && shSpace(src[j]) {
						j++
					}
					if run := src[i:j]; strings.Contains(run, "\n") && run != " " {
						edits = append(edits, shEdit{i, j, " "})
					}
					i = j
				default:
					i++
				}
			case shTag:
				switch {
				case shSpace(c):
					j := i
					for j < end && shSpace(src[j]) {
						j++
					}
					if j < end && src[j] == '>' {
						edits = append(edits, shEdit{i, j, ""})
					} else if src[i:j] != " " {
						edits = append(edits, shEdit{i, j, " "})
					}
					i = j
				case c == '"':
					st, i = shAttrDQ, i+1
				case c == '\'':
					st, i = shAttrSQ, i+1
				case c == '>':
					if pendingRaw != "" {
						st, raw = shRaw, pendingRaw
					} else {
						st = shText
					}
					i++
				default:
					i++
				}
			case shAttrDQ, shAttrSQ:
				if (st == shAttrDQ && c == '"') || (st == shAttrSQ && c == '\'') {
					st = shTag
				}
				i++
			case shComment:
				if e := strings.Index(src[i:end], "-->"); e >= 0 {
					st, i = shText, i+e+3
				} else {
					i = end
				}
			case shRaw:
				if strings.HasPrefix(strings.ToLower(src[i:min(end, i+2+len(raw))]), "</"+raw) {
					st = shText // the closing tag is read as text next
					continue
				}
				i++
			}
		}
	}
	return edits
}
