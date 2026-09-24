package web

import (
	"net/http/httptest"
	"testing"
)

// Over its budget the bundle stops being the default, but stays reachable
// with ?load=bundle for measuring.
func TestUseBundleBudget(t *testing.T) {
	over := &assets{bundle: generated{body: []byte("x")}} // bundleFits false
	if over.useBundle(httptest.NewRequest("GET", "/about", nil)) {
		t.Error("an over-budget bundle must fall back to the autoloader")
	}
	if !over.useBundle(httptest.NewRequest("GET", "/about?load=bundle", nil)) {
		t.Error("?load=bundle should still force it")
	}
	fits := &assets{bundle: generated{body: []byte("x")}, bundleFits: true}
	if fits.useBundle(httptest.NewRequest("GET", "/about?load=auto", nil)) {
		t.Error("?load=auto should force the autoloader")
	}
}
