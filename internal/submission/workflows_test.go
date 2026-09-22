package submission_test

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// The deploy key is a secret of the `production` environment, and GitHub
// hands it to any job that names that environment and runs on main. The
// submission workflow also runs on main (issue events use the default
// branch) and processes untrusted input, so only deploy.yml may name it.
func TestOnlyDeployUsesProduction(t *testing.T) {
	files, _ := filepath.Glob("../../.github/workflows/*.yml")
	if len(files) == 0 {
		t.Fatal("no workflows found")
	}
	env := regexp.MustCompile(`(?m)^\s*(environment:\s*production|name:\s*production)\s*$`)
	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		if env.Match(b) && filepath.Base(f) != "deploy.yml" {
			t.Errorf("%s uses the production environment; only deploy.yml may", filepath.Base(f))
		}
		if strings.Contains(string(b), "pull_request_target") {
			t.Errorf("%s uses pull_request_target, which runs untrusted code with secrets", filepath.Base(f))
		}
	}
}

// Untrusted submission code runs only in the read-only build job.
func TestSubmissionBuildIsReadOnly(t *testing.T) {
	b, err := os.ReadFile("../../.github/workflows/component-from-issue.yml")
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	build := s[strings.Index(s, "\n  build:"):strings.Index(s, "\n  pull-request:")]
	if !strings.Contains(build, "contents: read") || strings.Contains(build, "write") {
		t.Error("the build job must only have read permissions")
	}
	if strings.Contains(s[strings.Index(s, "\n  pull-request:"):], "go run") {
		t.Error("the write-capable jobs must not run repository code")
	}
}
