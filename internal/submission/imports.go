package submission

import (
	"fmt"
	"path"
	"strings"

	"starbase/internal/catalog"
)

const (
	maxVendorFile  = 2 << 20
	maxVendorTotal = 4 << 20
	maxVendorFiles = 32
)

// vendorFiles follows the relative imports of the component file entry
// (a path inside files) and returns every file it reaches, keyed by its path
// relative to the component's folder (the entry's directory). Imports must
// be 'datastar' or relative and stay inside that folder.
func vendorFiles(files map[string]string, entry string) (map[string]string, error) {
	root := path.Dir(entry)
	out := map[string]string{}
	total := 0
	var walk func(file string) error
	walk = func(file string) error {
		for _, spec := range catalog.Imports(files[file]) {
			if spec == "datastar" {
				continue
			}
			if !strings.HasPrefix(spec, "./") && !strings.HasPrefix(spec, "../") {
				return fmt.Errorf("`%s` imports `%s`. Components may only import `'datastar'` and files in their own folder: vendor the library there (e.g. `./vendor/%s.js`) and import it relatively.", file, spec, path.Base(spec))
			}
			target := path.Join(path.Dir(file), spec)
			rel, inside := within(root, target)
			if !inside {
				return fmt.Errorf("`%s` imports `%s`, which is outside the component's folder (`%s`).", file, spec, root)
			}
			if target == entry {
				continue
			}
			if _, seen := out[rel]; seen {
				continue
			}
			code, ok := files[target]
			if !ok {
				return fmt.Errorf("`%s` imports `%s`, which isn't part of the submission. Vendored files come along when you link a repository (as .js or .mjs files of up to %d MB).", file, spec, maxVendorFile>>20)
			}
			total += len(code)
			if len(out) >= maxVendorFiles || total > maxVendorTotal {
				return fmt.Errorf("the component imports more than %d files or %d MB. Please trim the vendored build.", maxVendorFiles, maxVendorTotal>>20)
			}
			out[rel] = code
			if err := walk(target); err != nil {
				return err
			}
		}
		return nil
	}
	return out, walk(entry)
}

// within reports whether the clean path p is inside dir, and p relative to it.
func within(dir, p string) (string, bool) {
	if dir == "." {
		return p, p != ".." && !strings.HasPrefix(p, "../")
	}
	rel, ok := strings.CutPrefix(p, dir+"/")
	return rel, ok
}
