package templgen

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"slices"

	"starbase/internal/catalog"
)

// Load builds the model of the components in root/components.
func Load(root string) (*Model, error) {
	cat, err := catalog.Load(os.DirFS(filepath.Join(root, "components")))
	if err != nil {
		return nil, err
	}
	return Build(cat)
}

// Refresh generates every file under root and returns the paths whose
// content differs from what is there; with write, it writes them.
func Refresh(root string, write bool) (stale []string, err error) {
	m, err := Load(root)
	if err != nil {
		return nil, err
	}
	datastar, err := os.ReadFile(filepath.Join(root, "static", "vendor", "datastar-rocket.js"))
	if err != nil {
		return nil, err
	}
	changelog, err := os.ReadFile(filepath.Join(root, "CHANGELOG.md"))
	if err != nil {
		return nil, err
	}
	release, err := Release(changelog)
	if err != nil {
		return nil, err
	}
	files, err := Files(m, datastar, release, SiteURL)
	if err != nil {
		return nil, err
	}
	for p, b := range files {
		path := filepath.Join(root, filepath.FromSlash(p))
		if old, err := os.ReadFile(path); err == nil && bytes.Equal(old, b) {
			continue
		}
		stale = append(stale, p)
		if write {
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				return nil, err
			}
			if err := os.WriteFile(path, b, 0o644); err != nil {
				return nil, err
			}
		}
	}
	slices.Sort(stale)
	return stale, nil
}

// Release is the newest released version in a changelog: the first
// "## [x.y.z]" heading, which skips [Unreleased].
func Release(changelog []byte) (string, error) {
	m := releaseHeading.FindSubmatch(changelog)
	if m == nil {
		return "", errors.New("CHANGELOG.md: no released version (## [x.y.z])")
	}
	return string(m[1]), nil
}

var releaseHeading = regexp.MustCompile(`(?m)^## \[(\d+\.\d+\.\d+)\]`)

// SiteURL is where the generated docs link the components' pages.
const SiteURL = "https://starbase.zweiundeins.gmbh"
