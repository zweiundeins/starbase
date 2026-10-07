package catalog

import (
	"bytes"
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
