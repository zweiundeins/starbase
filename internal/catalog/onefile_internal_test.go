package catalog

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"starbase/components"
)

// A version's bundle is frozen once stored, and cmd/dist writes it from a
// checkout: building it again gives the same bytes.
func TestBundleDeterministic(t *testing.T) {
	cat, err := Load(components.FS)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range cat.Components {
		cached, err := cat.BundleOf(c)
		if err != nil {
			t.Fatal(err)
		}
		again, err := cat.bundleOf(c)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(cached.Body, again.Body) {
			t.Errorf("%s: two builds of the bundle differ", c.Slug)
		}
	}
}

// A snapshot's autoloader is frozen under the catalog's hash, so the hash
// covers what the autoloader loads (snapshotFormat): the catalog that moved
// the autoloader to bundles, with the same component versions as the one
// before, has a snapshot of its own.
func TestCatalogHashNamesTheAutoloader(t *testing.T) {
	cat, err := Load(components.FS)
	if err != nil {
		t.Fatal(err)
	}
	h := sha256.New()
	for _, c := range cat.Components {
		h.Write([]byte(c.Hash))
	}
	if cat.Hash == hex.EncodeToString(h.Sum(nil))[:12] {
		t.Error("the catalog's hash leaves out snapshotFormat")
	}
}
