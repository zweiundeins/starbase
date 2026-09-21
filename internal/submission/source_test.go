package submission_test

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

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
