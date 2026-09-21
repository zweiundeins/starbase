// Command newcomponent scaffolds a component folder:
//
//	go run ./cmd/newcomponent my-widget --category forms --author octocat
package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"
	"time"

	"starbase/internal/catalog"
)

var slugRe = regexp.MustCompile(`^[a-z][a-z0-9]*(-[a-z0-9]+)*$`)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "newcomponent:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	fs := flag.NewFlagSet("newcomponent", flag.ContinueOnError)
	category := fs.String("category", "experimental", "category slug")
	author := fs.String("author", gitHubUser(), "your GitHub handle")
	dir := fs.String("dir", "components", "components directory")
	// Allow the slug before or after flags.
	var slug string
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		slug, args = args[0], args[1:]
	}
	if err := fs.Parse(args); err != nil {
		return err
	}
	if slug == "" {
		slug = fs.Arg(0)
	}
	slug = strings.TrimPrefix(slug, "sb-")
	if !slugRe.MatchString(slug) {
		return fmt.Errorf("usage: newcomponent <kebab-case-name> [--category forms] [--author you]")
	}
	if _, ok := catalog.CategoryBySlug(*category); !ok {
		var names []string
		for _, c := range catalog.Categories {
			names = append(names, c.Slug)
		}
		return fmt.Errorf("unknown category %q (one of: %s)", *category, strings.Join(names, ", "))
	}
	if *author == "" {
		*author = "your-github-handle"
	}
	target := filepath.Join(*dir, slug)
	if _, err := os.Stat(target); err == nil {
		return fmt.Errorf("%s already exists", target)
	}
	data := map[string]string{
		"Slug":     slug,
		"Tag":      "sb-" + slug,
		"Name":     title(slug),
		"Category": *category,
		"Author":   *author,
		"Today":    time.Now().Format(time.DateOnly),
	}
	if err := os.MkdirAll(target, 0o755); err != nil {
		return err
	}
	for name, tpl := range map[string]string{"README.md": readmeTpl, slug + ".js": jsTpl} {
		var b bytes.Buffer
		if err := template.Must(template.New(name).Delims("[[", "]]").Parse(tpl)).Execute(&b, data); err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(target, name), b.Bytes(), 0o644); err != nil {
			return err
		}
	}
	fmt.Printf("Created %s/\n  README.md\n  %s.js\n\nNext: go tool task live, open /components/%s\n", target, slug, slug)
	return nil
}

func title(slug string) string {
	parts := strings.Split(slug, "-")
	for i, p := range parts {
		parts[i] = strings.ToUpper(p[:1]) + p[1:]
	}
	return strings.Join(parts, " ")
}

func gitHubUser() string {
	out, err := exec.Command("git", "config", "github.user").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

const readmeTpl = `---
name: [[.Name]]
tag: [[.Tag]]
category: [[.Category]]
summary: One line about what [[.Name]] does.
author: [[.Author]]
tags: []
since: [[.Today]]
preview: |
  <[[.Tag]] label="Hello"></[[.Tag]]>
---

Describe the component in a sentence or two: what it is for and when to reach for it.

## Examples

### Basic

` + "```html preview" + `
<[[.Tag]] label="Hello, Starbase"></[[.Tag]]>
` + "```" + `

### With Datastar

` + "```html preview" + `
<div data-signals:_clicks="0">
  <[[.Tag]] label="Click me" data-on:sb-press="$_clicks++"></[[.Tag]]>
  <span data-text="$_clicks"></span>
</div>
` + "```" + `

## Accessibility

Explain keyboard support, roles and labels.
`

const jsTpl = `import { rocket } from 'datastar'

// Read semantic tokens, each with a fallback so the component also works
// outside Starbase.
const styles = /* css */ ` + "`" + `
:host {
	--_bg: var(--sb-surface-card, #141D32);
	--_border: var(--sb-border, #283552);
	--_text: var(--sb-text-1, #F3F4FA);
	--_brand: var(--sb-brand, #8C6BFF);
	--_radius: var(--sb-radius, 8px);
	display: inline-block;
}
button {
	all: unset;
	padding: 0.5rem 1rem;
	border: 1px solid var(--_border);
	border-radius: var(--_radius);
	background: var(--_bg);
	color: var(--_text);
	cursor: pointer;
}
button:hover { border-color: var(--_brand); }
button:focus-visible { outline: 2px solid var(--_brand); outline-offset: 2px; }
` + "`" + `

rocket('[[.Tag]]', {
	props: ({ string }) => ({
		label: string.trim.default('[[.Name]]').docs({ description: 'Text on the button.' }),
	}),
	manifest: {
		events: [{ name: 'sb-press', kind: 'custom-event', bubbles: true, composed: true, description: 'When pressed. detail: { presses }.' }],
	},
	setup: ({ $$, action, adoptStyles, emit, host }) => {
		adoptStyles(host, styles)
		// Interaction state lives in $$ signals, never in attributes.
		$$.presses = 0
		action('press', () => {
			$$.presses++
			emit('sb-press', { presses: $$.presses })
		})
	},
	render: ({ html, props: { label } }) => html` + "`" + `
		<button type="button" part="button" data-on:click="@press()">${label}</button>
	` + "`" + `,
})
`
