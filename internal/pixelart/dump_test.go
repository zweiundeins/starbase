package pixelart

import (
	"bytes"
	"image/png"
	"os"
	"strings"
	"testing"
)

func TestArtRenders(t *testing.T) {
	for name, svg := range All() {
		if !strings.HasPrefix(svg, "<svg") || !strings.Contains(svg, "<path") {
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

func TestRasters(t *testing.T) {
	for name, b := range map[string][]byte{"social": SocialImage(), "icon180": Icon(180), "icon32": Icon(32)} {
		img, err := png.Decode(bytes.NewReader(b))
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if name == "social" && (img.Bounds().Dx() != 1200 || img.Bounds().Dy() != 630) {
			t.Errorf("social image is %v", img.Bounds())
		}
		// Some pixels must be art, not background.
		if r, g, b, _ := img.At(img.Bounds().Dx()/2, img.Bounds().Dy()/2).RGBA(); r>>8 == 0x08 && g>>8 == 0x0D && b>>8 == 0x1D && name != "social" {
			t.Errorf("%s: the centre is empty", name)
		}
	}
}
