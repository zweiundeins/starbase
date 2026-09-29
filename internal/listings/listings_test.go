package listings

import (
	"os"
	"path/filepath"
	"testing"
)

// Every Go listing in the docs matches the code it names.
func TestListingsUpToDate(t *testing.T) {
	stale, err := Refresh("../..", false)
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range stale {
		if s.Err != nil {
			t.Errorf("%s:%d: %v", s.Doc, s.Line, s.Err)
		} else {
			t.Errorf("%s:%d: the listing doesn't match its source; run go tool task listings", s.Doc, s.Line)
		}
	}
}

func TestExtractAndRefresh(t *testing.T) {
	dir := t.TempDir()
	write := func(name, s string) {
		os.MkdirAll(filepath.Dir(filepath.Join(dir, name)), 0o755)
		if err := os.WriteFile(filepath.Join(dir, name), []byte(s), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("x/x.go", "package x\n\n// T is a thing.\ntype T struct{ A int }\n\n// Do does it.\nfunc (t *T) Do() int { return t.A }\n\nfunc other() {}\n")
	write("content/doc.md", "# Doc\n\n```go source=x/x.go#T,T.Do\nold\n```\n\n```go\nuntouched\n```\n\n````md\n```go source=x/x.go#T\n```\n````\n")
	stale, err := Refresh(dir, true)
	if err != nil || len(stale) != 1 {
		t.Fatalf("refresh: %v, %+v", err, stale)
	}
	got, _ := os.ReadFile(filepath.Join(dir, "content/doc.md"))
	want := "# Doc\n\n```go source=x/x.go#T,T.Do\n// T is a thing.\ntype T struct{ A int }\n\n// Do does it.\nfunc (t *T) Do() int { return t.A }\n```\n\n```go\nuntouched\n```\n\n````md\n```go source=x/x.go#T\n```\n````\n"
	if string(got) != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
	if stale, _ := Refresh(dir, false); len(stale) != 0 {
		t.Errorf("still stale after a rewrite: %+v", stale)
	}
	write("content/doc.md", "```go source=x/x.go#Missing\n```\n")
	if stale, _ := Refresh(dir, false); len(stale) != 1 || stale[0].Err == nil {
		t.Errorf("a missing declaration must be reported: %+v", stale)
	}
}
