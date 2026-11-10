package app_test

import (
	"bufio"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"starbase/components"
	"starbase/internal/catalog"
)

// officialBundle is the official Datastar + Rocket release that the vendored
// build patches (https://cdn.jsdelivr.net/gh/starfederation/datastar@<version>/bundles/datastar-rocket.js).
const officialBundle = "testdata/datastar-rocket-official.js"

// TestOfficialBundleMatchesVendored keeps officialBundle at the release the
// vendored build was made from: their banners name the same version.
func TestOfficialBundleMatchesVendored(t *testing.T) {
	first := func(p string) string {
		f, err := os.Open(p)
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		s := bufio.NewScanner(f)
		s.Scan()
		return s.Text()
	}
	official, vendored := first(officialBundle), first("../../static/vendor/datastar-rocket.js")
	if want := strings.TrimSuffix(vendored, " (patched: patches/rocket)"); official != want {
		t.Errorf("%s is %q, the vendored build %q: download the release it was made from", officialBundle, official, want)
	}
}

// TestComponentsDefineOnOfficialBundle loads the homepage, which runs every
// component from the bundle, on the official release (or on
// STARBASE_DATASTAR_BUNDLE): every tag is defined, with the props its
// manifest.json lists. Official Rocket throws on a declaration it refuses
// (bool.default(…)), and then no component queued after it is defined.
func TestComponentsDefineOnOfficialBundle(t *testing.T) {
	if os.Getenv("STARBASE_DATASTAR_BUNDLE") == "" {
		t.Setenv("STARBASE_DATASTAR_BUNDLE", officialBundle)
	}
	cat, err := catalog.Load(components.FS)
	if err != nil {
		t.Fatal(err)
	}
	var tags []string
	for _, c := range cat.Components {
		tags = append(tags, c.Tag)
	}
	tj, _ := json.Marshal(tags)
	body := probeBundle(t, "/", strings.Replace(definedPropsJS, "%TAGS%", string(tj), 1), nil)
	var got definedProps
	if err := json.Unmarshal(body, &got); err != nil || got.Error != "" {
		t.Fatalf("%v: %s", err, body)
	}
	checkProps(t, cat, got.Manifests, "the official bundle")
}
