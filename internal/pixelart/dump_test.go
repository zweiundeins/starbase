package pixelart

import (
	"os"
	"strings"
	"testing"
)

func TestArtRenders(t *testing.T) {
	for name, svg := range All() {
		if !strings.HasPrefix(svg, "<svg") || !strings.Contains(svg, "<rect") {
			t.Errorf("%s: not an SVG with pixels", name)
		}
	}
}

// ART_DUMP=/some/dir go test -run TestDump ./internal/pixelart writes the
// SVGs for eyeballing.
func TestDump(t *testing.T) {
	dir := os.Getenv("ART_DUMP")
	if dir == "" {
		t.Skip("set ART_DUMP to write the SVGs")
	}
	for name, svg := range All() {
		if err := os.WriteFile(dir+"/"+name+".svg", []byte(svg), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}
