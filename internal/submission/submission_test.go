package submission_test

import (
	"os"
	"regexp"
	"slices"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"starbase/internal/catalog"
	"starbase/internal/submission"
)

// TestLabelsMatchTemplate keeps the parser and the issue form in sync.
func TestLabelsMatchTemplate(t *testing.T) {
	b, err := os.ReadFile("../../.github/ISSUE_TEMPLATE/new-component.yml")
	if err != nil {
		t.Fatal(err)
	}
	var labels []string
	for _, m := range regexp.MustCompile(`(?m)^      label: (.+)$`).FindAllStringSubmatch(string(b), -1) {
		labels = append(labels, m[1])
	}
	if !slices.Equal(labels, submission.Labels) {
		t.Fatalf("template labels %q\nparser labels  %q", labels, submission.Labels)
	}
}

// body mimics how GitHub renders an issue form.
const body = "### Component name\n\nOrbit Picker\n\n" +
	"### Category\n\nforms\n\n" +
	"### Summary\n\nPick a planet from a spinning pixel carousel.\n\n" +
	"### Tags\n\nPicker, Carousel , space\n\n" +
	"### Component code\n\n```javascript\nimport { rocket } from 'datastar'\nrocket('sb-orbit-picker', {\n  render: ({ html }) => html`<b>hi</b>`,\n})\n```\n\n" +
	"### Preview\n\n```html\n<sb-orbit-picker></sb-orbit-picker>\n```\n\n" +
	"### Documentation\n\nSpins.\n\n## Examples\n\n### Basic\n\n```html preview\n<sb-orbit-picker></sb-orbit-picker>\n```\n\n" +
	"### Playground settings\n\n_No response_\n\n" +
	"### License\n\n- [X] I wrote this component (or have the right to share it), and I license it under the MIT license."

func TestParseAndBuild(t *testing.T) {
	sub := submission.Parse(body)
	if sub.Name != "Orbit Picker" || sub.Category != "forms" || !sub.Licensed || sub.Playground != "" {
		t.Fatalf("parsed %+v", sub)
	}
	if !slices.Equal(sub.Tags, []string{"picker", "carousel", "space"}) {
		t.Errorf("tags = %q", sub.Tags)
	}
	if !strings.Contains(sub.Docs, "### Basic") {
		t.Error("docs lost their own ### heading")
	}
	res, err := sub.Build("astroalex", time.Date(2026, 9, 21, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if res.Slug != "orbit-picker" || res.Tag != "sb-orbit-picker" {
		t.Fatalf("slug/tag = %s/%s", res.Slug, res.Tag)
	}
	// The generated folder must pass the same validation as a hand-made PR.
	fsys := fstest.MapFS{}
	for name, data := range res.Files {
		fsys["orbit-picker/"+name] = &fstest.MapFile{Data: data}
	}
	cat, err := catalog.Load(fsys)
	if err != nil {
		t.Fatalf("generated folder is invalid: %v\n%s", err, res.Files["README.md"])
	}
	c, _ := cat.Get("orbit-picker")
	if c.Author != "astroalex" || c.Since != "2026-09-21" || !strings.Contains(c.DocHTML, `class="demo"`) {
		t.Errorf("component = %+v", c.Meta)
	}
}

func TestBuildReportsProblems(t *testing.T) {
	bad := strings.NewReplacer(
		"rocket('sb-orbit-picker'", "rocket('orbit-picker'",
		"### Category\n\nforms", "### Category\n\nspaceships",
		"- [X]", "- [ ]",
	).Replace(body)
	_, err := submission.Parse(bad).Build("astroalex", time.Now())
	if err == nil {
		t.Fatal("expected problems")
	}
	for _, want := range []string{"rocket('sb-your-name'", "Unknown category", "license"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("report lacks %q:\n%v", want, err)
		}
	}
}

func TestPlaygroundYAML(t *testing.T) {
	b := strings.Replace(body, "### Playground settings\n\n_No response_", "### Playground settings\n\n```yaml\nvalues: {label: Hi}\n```", 1)
	res, err := submission.Parse(b).Build("astroalex", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(res.Files["README.md"]), "playground:") {
		t.Errorf("README lacks playground:\n%s", res.Files["README.md"])
	}
}

func timeNow() time.Time { return time.Date(2026, 9, 21, 0, 0, 0, 0, time.UTC) }
