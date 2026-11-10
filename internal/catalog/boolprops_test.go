package catalog_test

import (
	"io/fs"
	"path"
	"slices"
	"testing"

	"starbase/components"
	"starbase/internal/catalog"
)

func TestBoolDefaults(t *testing.T) {
	code := "props: {\n\topen: bool,\n\tshown: bool.default(true),\n\tx: bool .default( false ),\n\ty: oneOf(true, false),\n}"
	if got := catalog.BoolDefaults(code); !slices.Equal(got, []int{3, 4}) {
		t.Errorf("BoolDefaults = %v, want [3 4]", got)
	}
}

// knownBoolDefaults still declare one, until they move to oneOf(true, false).
var knownBoolDefaults = map[string]bool{"alert": true, "code-editor": true, "code-playground": true, "count-up": true, "drawer": true, "meter": true, "modal": true, "nebula": true, "odometer": true, "range": true, "relative-time": true, "slider": true}

// No component declares `bool.default(…)`: official Rocket throws on it when
// the component is defined, and every component queued after it stays undefined.
func TestNoBoolDefaults(t *testing.T) {
	err := fs.WalkDir(components.FS, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == "vendor" {
				return fs.SkipDir
			}
			return nil
		}
		if ext := path.Ext(p); ext != ".js" && ext != ".ts" {
			return nil
		}
		b, err := fs.ReadFile(components.FS, p)
		if err != nil {
			return err
		}
		for _, l := range catalog.BoolDefaults(string(b)) {
			if knownBoolDefaults[path.Dir(p)] {
				continue
			}
			t.Errorf("%s:%d declares bool.default(…); a flag that is on by default is oneOf(true, false)", p, l)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
