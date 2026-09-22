// Command fromissue turns a "Submit a component" issue into a component
// folder. It runs in GitHub Actions and reads the event payload from a file,
// so untrusted issue text never touches a shell.
//
//	go run ./cmd/fromissue --event "$GITHUB_EVENT_PATH" --report report.md
//
// On success it writes components/<slug>/ and, when $GITHUB_OUTPUT is set,
// the outputs slug, tag and name. On failure it writes the problems to the
// report (markdown, posted as an issue comment) and exits 1.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"starbase/internal/catalog"
	"starbase/internal/submission"
)

type event struct {
	Issue struct {
		Number    int       `json:"number"`
		Body      string    `json:"body"`
		CreatedAt time.Time `json:"created_at"`
		User      struct {
			Login string `json:"login"`
		} `json:"user"`
	} `json:"issue"`
}

func main() {
	eventPath := flag.String("event", os.Getenv("GITHUB_EVENT_PATH"), "GitHub event payload (JSON)")
	dir := flag.String("dir", "components", "components directory")
	report := flag.String("report", "report.md", "where to write the markdown report")
	notes := flag.String("notes", "notes.md", "where to write the reviewer notes for the PR body (markdown)")
	api := flag.String("api", "https://api.github.com", "GitHub API base URL")
	site := flag.String("site", os.Getenv("STARBASE_URL"), "public Starbase URL; playground links must point here")
	flag.Parse()

	res, name, like, err := run(*eventPath, *dir, *api, *site)
	if err != nil {
		msg := "🛑 **This submission can't be turned into a pull request yet.**\n\n" + err.Error() +
			"\n\nEdit the issue to fix it, and I'll try again automatically."
		os.WriteFile(*report, []byte(msg+"\n"), 0o644)
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	os.WriteFile(*report, []byte(fmt.Sprintf("✅ `<%s>` is valid. Opening a pull request…\n", res.Tag)), 0o644)
	os.WriteFile(*notes, []byte(like+"\n"), 0o644)
	if out := os.Getenv("GITHUB_OUTPUT"); out != "" {
		f, err := os.OpenFile(out, os.O_APPEND|os.O_WRONLY, 0o644)
		if err == nil {
			// Values are validated (slug/tag regexes); name is single-line.
			fmt.Fprintf(f, "slug=%s\ntag=%s\nname=%s\n", res.Slug, res.Tag, strings.ReplaceAll(name, "\n", " "))
			f.Close()
		}
	}
	fmt.Printf("wrote %s/%s\n", *dir, res.Slug)
}

func run(eventPath, dir, api, site string) (submission.Result, string, string, error) {
	raw, err := os.ReadFile(eventPath)
	if err != nil {
		return submission.Result{}, "", "", err
	}
	var ev event
	if err := json.Unmarshal(raw, &ev); err != nil {
		return submission.Result{}, "", "", err
	}
	author := ev.Issue.User.Login
	sub := submission.Parse(ev.Issue.Body)

	if sub.Source != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		var src submission.Source
		var err error
		if submission.IsPlaygroundLink(sub.Source) {
			src, err = submission.FetchSnippet(ctx, http.DefaultClient, site, sub.Source)
		} else {
			src, err = submission.Fetch(ctx, http.DefaultClient, api, os.Getenv("GITHUB_TOKEN"), sub.Source)
		}
		if err != nil {
			return submission.Result{}, "", "", fmt.Errorf("- %v", err)
		}
		sub.Apply(src)
	}
	since := ev.Issue.CreatedAt
	if since.IsZero() {
		since = time.Now()
	}
	res, err := sub.Build(author, since)
	if err != nil {
		return submission.Result{}, "", "", err
	}

	// Updating is allowed for the same author; taking over a slug is not.
	target := filepath.Join(dir, res.Slug)
	if b, err := os.ReadFile(filepath.Join(target, "README.md")); err == nil {
		var meta catalog.Meta
		if _, err := catalog.RenderMarkdown(b, &meta); err == nil && !strings.EqualFold(meta.Author, author) {
			return submission.Result{}, "", "", fmt.Errorf("- `<%s>` already exists, by @%s. Please pick another tag.", res.Tag, meta.Author)
		}
		if err := os.RemoveAll(target); err != nil {
			return submission.Result{}, "", "", err
		}
	}
	if err := os.MkdirAll(target, 0o755); err != nil {
		return submission.Result{}, "", "", err
	}
	for name, data := range res.Files {
		if err := os.MkdirAll(filepath.Dir(filepath.Join(target, name)), 0o755); err != nil {
			return submission.Result{}, "", "", err
		}
		if err := os.WriteFile(filepath.Join(target, name), data, 0o644); err != nil {
			return submission.Result{}, "", "", err
		}
	}
	// Validate exactly like a hand-made pull request would be.
	cat, err := catalog.Load(os.DirFS(dir))
	if err != nil {
		os.RemoveAll(target)
		var lines []string
		for _, l := range strings.Split(err.Error(), "\n") {
			if strings.Contains(l, res.Slug) {
				lines = append(lines, "- "+l)
			}
		}
		if len(lines) == 0 {
			lines = []string{"- " + err.Error()}
		}
		return submission.Result{}, "", "", errors.New(strings.Join(lines, "\n"))
	}
	c, _ := cat.Get(res.Slug)
	return res, sub.Name, submission.ReviewNotes(res, submission.Similar(cat, c), site), nil
}
