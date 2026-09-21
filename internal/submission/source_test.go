package submission_test

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"starbase/internal/submission"
)

func tarball(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	for name, body := range files {
		tw.WriteHeader(&tar.Header{Name: "you-repo-abc1234/" + name, Mode: 0o644, Size: int64(len(body)), Typeflag: tar.TypeReg})
		tw.Write([]byte(body))
	}
	tw.Close()
	gz.Close()
	return buf.Bytes()
}

func fakeGitHub(t *testing.T, files map[string]string) *httptest.Server {
	data := tarball(t, files)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/repos/you/repo/tarball") {
			http.NotFound(w, r)
			return
		}
		w.Write(data)
	}))
	t.Cleanup(srv.Close)
	return srv
}

const repoCode = "import { rocket } from 'datastar'\nrocket('sb-orbit-picker', { render: ({ html }) => html`<b>hi</b>` })\n"

func TestFetchFolder(t *testing.T) {
	gh := fakeGitHub(t, map[string]string{
		"README.md":                                 "# repo root",
		"components/orbit-picker/orbit.js":          repoCode,
		"components/orbit-picker/README.md":         "---\nsummary: From the repo\npreview: <sb-orbit-picker></sb-orbit-picker>\n---\nRepo docs.\n",
		"components/other/other.js":                 "rocket('sb-other', {})",
		"components/orbit-picker/node_modules/x.js": "rocket('sb-evil', {})",
	})
	src, err := submission.Fetch(context.Background(), gh.Client(), gh.URL, "", "https://github.com/you/repo/tree/main/components/orbit-picker")
	if err != nil {
		t.Fatal(err)
	}
	if src.URL != "https://github.com/you/repo/tree/abc1234/components/orbit-picker" || src.CodeFile != "components/orbit-picker/orbit.js" {
		t.Fatalf("source = %+v", src)
	}
	sub := submission.Submission{Name: "Orbit Picker", Category: "forms", Licensed: true, Code: "ignored starter", Docs: "ignored template"}
	sub.Apply(src)
	if sub.Summary != "From the repo" || sub.Preview == "" || strings.TrimSpace(sub.Docs) != "Repo docs." || sub.Code != repoCode {
		t.Fatalf("applied = %+v", sub)
	}
	res, err := sub.Build("you", timeNow())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(res.Files["README.md"]), "source: https://github.com/you/repo/tree/abc1234/components/orbit-picker") {
		t.Errorf("README lacks pinned source:\n%s", res.Files["README.md"])
	}
}

func TestFetchNeedsExactlyOneComponent(t *testing.T) {
	gh := fakeGitHub(t, map[string]string{"a/a.js": "rocket('sb-a', {})", "b/b.js": "rocket('sb-b', {})"})
	_, err := submission.Fetch(context.Background(), gh.Client(), gh.URL, "", "https://github.com/you/repo")
	if err == nil || !strings.Contains(err.Error(), "several components") {
		t.Fatalf("err = %v", err)
	}
	_, err = submission.Fetch(context.Background(), gh.Client(), gh.URL, "", "https://gitlab.com/you/repo")
	if err == nil || !strings.Contains(err.Error(), "not a GitHub repository URL") {
		t.Fatalf("err = %v", err)
	}
}

func TestFetchSnippet(t *testing.T) {
	site := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/playground/snippet/AbCd2345" {
			http.NotFound(w, r)
			return
		}
		w.Write([]byte(`{"files": {"component.js": "rocket('sb-orbit-picker', {})", "index.html": "<sb-orbit-picker></sb-orbit-picker>\n\n<sb-orbit-picker big></sb-orbit-picker>"}}`))
	}))
	defer site.Close()
	link := site.URL + "/playground?s=AbCd2345"
	if !submission.IsPlaygroundLink(link) {
		t.Fatal("not recognised as a playground link")
	}
	if _, err := submission.FetchSnippet(context.Background(), site.Client(), "https://starbase.example", link); err == nil {
		t.Fatal("links to other hosts must be refused")
	}
	src, err := submission.FetchSnippet(context.Background(), site.Client(), site.URL, link)
	if err != nil {
		t.Fatal(err)
	}
	sub := submission.Submission{Name: "Orbit Picker", Category: "forms", Summary: "Spins.", Licensed: true, Docs: submission.DocsTemplate}
	sub.Apply(src)
	if sub.Preview != "<sb-orbit-picker></sb-orbit-picker>" || strings.Count(sub.Docs, "```html preview") != 2 || sub.Source != link {
		t.Fatalf("applied = %+v", sub)
	}
	if _, err := sub.Build("you", timeNow()); err != nil {
		t.Fatal(err)
	}
}

func TestDocsTemplateMatchesForm(t *testing.T) {
	b, err := os.ReadFile("../../.github/ISSUE_TEMPLATE/new-component.yml")
	if err != nil {
		t.Fatal(err)
	}
	var form struct {
		Body []struct {
			ID         string `yaml:"id"`
			Attributes struct {
				Value string `yaml:"value"`
			} `yaml:"attributes"`
		} `yaml:"body"`
	}
	if err := yaml.Unmarshal(b, &form); err != nil {
		t.Fatal(err)
	}
	for _, el := range form.Body {
		if el.ID == "docs" && strings.TrimSpace(el.Attributes.Value) != submission.DocsTemplate {
			t.Fatalf("DocsTemplate drifted from the form:\n%q\n%q", strings.TrimSpace(el.Attributes.Value), submission.DocsTemplate)
		}
	}
}
