// Package submission turns a "Submit a component" GitHub issue form into a
// component folder. The issue form (.github/ISSUE_TEMPLATE/new-component.yml)
// renders each field as a "### <label>" section in the issue body.
package submission

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"starbase/internal/catalog"
)

// Field labels, exactly as in the issue form. Keep them in sync: the test
// reads the template and checks.
const (
	LabelName       = "Component name"
	LabelCategory   = "Category"
	LabelSummary    = "Summary"
	LabelTags       = "Tags"
	LabelCode       = "Component code"
	LabelPreview    = "Preview"
	LabelDocs       = "Documentation"
	LabelPlayground = "Playground settings"
	LabelSource     = "Source repository"
	LabelLicense    = "License"
)

var Labels = []string{LabelName, LabelCategory, LabelSummary, LabelTags, LabelSource, LabelCode, LabelPreview, LabelDocs, LabelPlayground, LabelLicense}

type Submission struct {
	Name       string
	Category   string
	Summary    string
	Tags       []string
	Code       string
	Preview    string
	Docs       string
	Playground string // YAML, optional
	Source     string // repository URL (as given, or pinned after Apply)
	Licensed   bool
	Vendor     map[string]string // files the code imports relatively (from a linked repository)
}

var fenceRe = regexp.MustCompile("(?s)^```[a-zA-Z]*\\n(.*?)\\n?```$")

// sections splits the body on "### <known label>" headings only, so docs
// may contain their own ### headings.
func sections(body string) map[string]string {
	known := map[string]bool{}
	for _, l := range Labels {
		known[l] = true
	}
	out := map[string]string{}
	current := ""
	var buf []string
	flush := func() {
		if current != "" {
			v := strings.TrimSpace(strings.Join(buf, "\n"))
			if v == "_No response_" {
				v = ""
			}
			if m := fenceRe.FindStringSubmatch(v); m != nil { // render: <lang> fields
				v = strings.TrimSpace(m[1])
			}
			out[current] = v
		}
		buf = nil
	}
	for _, line := range strings.Split(strings.ReplaceAll(body, "\r\n", "\n"), "\n") {
		if h, ok := strings.CutPrefix(line, "### "); ok && known[strings.TrimSpace(h)] {
			flush()
			current = strings.TrimSpace(h)
			continue
		}
		buf = append(buf, line)
	}
	flush()
	return out
}

// Parse reads an issue form body.
func Parse(body string) Submission {
	s := sections(body)
	sub := Submission{
		Name:       s[LabelName],
		Category:   strings.ToLower(s[LabelCategory]),
		Summary:    s[LabelSummary],
		Code:       s[LabelCode],
		Preview:    s[LabelPreview],
		Docs:       s[LabelDocs],
		Playground: s[LabelPlayground],
		Source:     strings.TrimSpace(s[LabelSource]),
		Licensed:   strings.Contains(s[LabelLicense], "[X]") || strings.Contains(s[LabelLicense], "[x]"),
	}
	for _, t := range strings.Split(s[LabelTags], ",") {
		if t = strings.ToLower(strings.TrimSpace(t)); t != "" {
			sub.Tags = append(sub.Tags, t)
		}
	}
	return sub
}

var tagInCodeRe = regexp.MustCompile(`rocket\(\s*['"](sb-[a-z0-9]+(?:-[a-z0-9]+)*)['"]`)

// Result is a validated submission ready to be written to components/<Slug>.
type Result struct {
	Slug  string
	Tag   string
	Files map[string][]byte // file name → contents
}

// Build validates the submission and renders the component folder.
// author is the GitHub login of the issue's author.
func (s Submission) Build(author string, since time.Time) (Result, error) {
	var problems []string
	add := func(format string, args ...any) { problems = append(problems, fmt.Sprintf(format, args...)) }

	matches := tagInCodeRe.FindAllStringSubmatch(s.Code, -1)
	var tag string
	switch {
	case len(matches) == 0:
		add("The component code must define exactly one component with `rocket('sb-your-name', { … })`. The tag must start with `sb-`.")
	case len(matches) > 1:
		add("The component code defines %d components. Submit one per issue.", len(matches))
	default:
		tag = matches[0][1]
	}
	if strings.TrimSpace(s.Name) == "" {
		add("Please give the component a name.")
	}
	if _, ok := catalog.CategoryBySlug(s.Category); !ok {
		add("Unknown category %q.", s.Category)
	}
	if s.Summary == "" || len(s.Summary) > 90 {
		add("The summary must be one line of at most 90 characters (it is %d).", len(s.Summary))
	}
	if strings.TrimSpace(s.Preview) == "" {
		add("Please add a preview snippet: the HTML shown on the gallery card.")
	}
	if !s.Licensed {
		add("Please confirm the license checkbox.")
	}
	var vendor map[string]string
	if tag != "" {
		// Check the imports exactly as the folder will be laid out.
		entry := strings.TrimPrefix(tag, "sb-") + ".js"
		files := map[string]string{"component/" + entry: s.Code}
		for rel, code := range s.Vendor {
			if rel == entry || rel == "README.md" || rel == "manifest.json" {
				add("The vendored file `%s` clashes with a generated file. Please rename it.", rel)
			}
			files["component/"+rel] = code
		}
		var err error
		if vendor, err = vendorFiles(files, "component/"+entry); err != nil {
			add("%s", err.Error())
		}
	}
	var pg map[string]any
	if strings.TrimSpace(s.Playground) != "" {
		if err := yaml.Unmarshal([]byte(s.Playground), &pg); err != nil {
			add("The playground settings are not valid YAML: %v", err)
		}
	}
	if len(problems) > 0 {
		return Result{}, errors.New("- " + strings.Join(problems, "\n- "))
	}

	slug := strings.TrimPrefix(tag, "sb-")
	front := map[string]any{
		"name":     strings.TrimSpace(s.Name),
		"tag":      tag,
		"category": s.Category,
		"summary":  s.Summary,
		"author":   author,
		"tags":     s.Tags,
		"since":    since.UTC().Format(time.DateOnly),
		"preview":  strings.TrimSpace(s.Preview) + "\n",
	}
	if pg != nil {
		front["playground"] = pg
	}
	if s.Source != "" {
		front["source"] = s.Source
	}
	fm, err := marshalOrdered(front, []string{"name", "tag", "category", "summary", "author", "tags", "since", "source", "preview", "playground"})
	if err != nil {
		return Result{}, err
	}
	docs := strings.TrimSpace(s.Docs)
	if docs == "" {
		docs = "## Examples\n\n```html preview\n" + strings.TrimSpace(s.Preview) + "\n```"
	}
	readme := "---\n" + fm + "---\n\n" + docs + "\n"
	code := strings.TrimSpace(s.Code) + "\n"
	out := map[string][]byte{"README.md": []byte(readme), slug + ".js": []byte(code)}
	for rel, b := range vendor {
		out[rel] = []byte(b)
	}
	return Result{Slug: slug, Tag: tag, Files: out}, nil
}

// marshalOrdered writes YAML keys in a fixed, human-friendly order.
func marshalOrdered(m map[string]any, order []string) (string, error) {
	node := &yaml.Node{Kind: yaml.MappingNode}
	for _, k := range order {
		v, ok := m[k]
		if !ok {
			continue
		}
		var val yaml.Node
		if err := val.Encode(v); err != nil {
			return "", err
		}
		if k == "tags" {
			val.Style = yaml.FlowStyle
		}
		if s, ok := v.(string); ok && strings.Contains(s, "\n") {
			val.Style = yaml.LiteralStyle
		}
		node.Content = append(node.Content, &yaml.Node{Kind: yaml.ScalarNode, Value: k}, &val)
	}
	b, err := yaml.Marshal(node)
	return string(b), err
}

func yamlUnmarshal(s string, v any) error { return yaml.Unmarshal([]byte(s), v) }

func yamlMarshal(v any) string {
	b, _ := yaml.Marshal(v)
	return string(b)
}
