package tscheck

import (
	"context"
	"strings"
	"testing"
)

// A compiler that runs out of memory is an error, not a program without problems.
func TestCheckOutOfMemory(t *testing.T) {
	c := New(t.TempDir())
	if c == nil {
		t.Skip("no compiler embedded: go run ./cmd/fetchtsc")
	}
	if err := c.Warm(); err != nil {
		t.Fatal(err)
	}
	defer func(m uint64) { maxData = m }(maxData)
	maxData = 16 << 20
	ds, err := c.Check(context.Background(), []File{{"component.ts", "export const a: number = 1\n"}})
	t.Log(err)
	if err == nil || !strings.Contains(err.Error(), "tscheck:") {
		t.Errorf("got %+v, %v; want an error", ds, err)
	}
}
