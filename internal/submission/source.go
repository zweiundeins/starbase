package submission

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path"
	"regexp"
	"strings"
)

// Source is a component fetched from the contributor's own repository.
type Source struct {
	URL      string // pinned: https://github.com/<owner>/<repo>/tree/<sha>/<dir>
	CodeFile string // path of the component file inside the repo
	Code     string
	Readme   string // README.md next to the component file, if any
}

var repoURLRe = regexp.MustCompile(`^https://github\.com/([A-Za-z0-9_.-]+)/([A-Za-z0-9_.-]+?)(?:\.git)?(?:/tree/([^/]+)(?:/(.*?))?)?/?$`)

const (
	maxTarball = 10 << 20
	maxFile    = 256 << 10
)

// Fetch downloads a public GitHub repository (or a folder of it) through the
// GitHub API and picks out the single Rocket component in it. apiBase is
// "https://api.github.com" in production; token may be empty.
func Fetch(ctx context.Context, client *http.Client, apiBase, token, repoURL string) (Source, error) {
	m := repoURLRe.FindStringSubmatch(strings.TrimSpace(repoURL))
	if m == nil {
		return Source{}, fmt.Errorf("%q is not a GitHub repository URL like https://github.com/you/repo or https://github.com/you/repo/tree/main/path/to/component", repoURL)
	}
	owner, repo, ref, dir := m[1], m[2], m[3], strings.Trim(m[4], "/")

	u := fmt.Sprintf("%s/repos/%s/%s/tarball", strings.TrimSuffix(apiBase, "/"), owner, repo)
	if ref != "" {
		u += "/" + ref
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return Source{}, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	res, err := client.Do(req)
	if err != nil {
		return Source{}, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return Source{}, fmt.Errorf("could not download %s/%s (GitHub says %s). Is the repository public?", owner, repo, res.Status)
	}

	gz, err := gzip.NewReader(io.LimitReader(res.Body, maxTarball))
	if err != nil {
		return Source{}, fmt.Errorf("unexpected archive: %w", err)
	}
	tr := tar.NewReader(gz)
	files := map[string]string{} // repo-relative path → contents (.js and README.md only)
	sha := ""
	for {
		h, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return Source{}, fmt.Errorf("reading archive: %w", err)
		}
		// Entries look like "<owner>-<repo>-<sha>/path/to/file".
		top, rel, _ := strings.Cut(h.Name, "/")
		if sha == "" {
			sha = top[strings.LastIndex(top, "-")+1:]
		}
		if h.Typeflag != tar.TypeReg || rel == "" {
			continue
		}
		if dir != "" && rel != dir && !strings.HasPrefix(rel, dir+"/") {
			continue
		}
		if !strings.HasSuffix(rel, ".js") && !strings.HasSuffix(rel, ".mjs") && path.Base(rel) != "README.md" {
			continue
		}
		if strings.Contains(rel, "node_modules/") || h.Size > maxFile {
			continue
		}
		b, err := io.ReadAll(io.LimitReader(tr, maxFile))
		if err != nil {
			return Source{}, err
		}
		files[rel] = string(b)
	}

	var candidates []string
	for p, c := range files {
		if !strings.HasSuffix(p, "README.md") && len(tagInCodeRe.FindAllString(c, -1)) > 0 {
			candidates = append(candidates, p)
		}
	}
	where := owner + "/" + repo
	if dir != "" {
		where += "/" + dir
	}
	switch len(candidates) {
	case 0:
		return Source{}, fmt.Errorf("no file in %s calls `rocket('sb-…', …)`", where)
	case 1:
	default:
		return Source{}, fmt.Errorf("%s contains several components (%s). Link the folder of the one to submit, e.g. …/tree/main/path/to/it", where, strings.Join(candidates, ", "))
	}
	file := candidates[0]
	srcDir := path.Dir(file)
	readme := files[path.Join(srcDir, "README.md")]
	if readme == "" && dir != "" {
		readme = files[path.Join(dir, "README.md")]
	}
	pinned := fmt.Sprintf("https://github.com/%s/%s/tree/%s", owner, repo, sha)
	if srcDir != "." {
		pinned += "/" + srcDir
	}
	return Source{URL: pinned, CodeFile: file, Code: files[file], Readme: readme}, nil
}

// Apply merges a fetched source into the submission. The repository wins
// for code and docs. Its README front matter fills fields the form left
// empty (preview, playground, tags, summary).
func (s *Submission) Apply(src Source) {
	s.Source = src.URL
	s.Code = src.Code
	fm, body := splitFrontMatter(src.Readme)
	if strings.TrimSpace(body) != "" {
		s.Docs = body
	}
	var meta struct {
		Summary    string         `yaml:"summary"`
		Tags       []string       `yaml:"tags"`
		Preview    string         `yaml:"preview"`
		Playground map[string]any `yaml:"playground"`
	}
	if fm != "" {
		_ = yamlUnmarshal(fm, &meta)
	}
	if s.Summary == "" {
		s.Summary = meta.Summary
	}
	if len(s.Tags) == 0 {
		s.Tags = meta.Tags
	}
	if strings.TrimSpace(s.Preview) == "" {
		s.Preview = meta.Preview
	}
	if strings.TrimSpace(s.Playground) == "" && meta.Playground != nil {
		s.Playground = yamlMarshal(meta.Playground)
	}
}

func splitFrontMatter(md string) (fm, body string) {
	md = strings.ReplaceAll(md, "\r\n", "\n")
	if rest, ok := strings.CutPrefix(md, "---\n"); ok {
		if i := strings.Index(rest, "\n---\n"); i >= 0 {
			return rest[:i], rest[i+5:]
		}
	}
	return "", md
}
