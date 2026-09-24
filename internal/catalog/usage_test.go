package catalog_test

import (
	"strings"
	"testing"

	"starbase/components"
	"starbase/internal/catalog"
)

// TestInstallMarkup: the installation snippets end with markup people paste
// into their own page, so it must use the component and carry none of the
// gallery card's demo wiring or Starbase-only assets.
func TestInstallMarkup(t *testing.T) {
	cat, err := catalog.Load(components.FS)
	if err != nil {
		t.Fatal(err)
	}
	banned := []string{
		"sb-theme-demo",    // theme-switch's demo cookie
		"data-demo-theme",  // … and attribute
		"/art/",            // Starbase's pixel art
		"data-signals",     // demo state
		"data-on-interval", // demo timers
		"inline-size",      // card sizing
	}
	for _, c := range cat.Components {
		m := c.InstallMarkup()
		if !strings.Contains(m, "<"+c.Tag) {
			t.Errorf("%s: installation markup does not use <%s>:\n%s", c.Slug, c.Tag, m)
		}
		for _, b := range banned {
			if strings.Contains(m, b) {
				t.Errorf("%s: installation markup contains %q; give it a usage: in the front matter:\n%s", c.Slug, b, m)
			}
		}
	}
}

func TestInstallMarkupFallback(t *testing.T) {
	m := catalog.Meta{Preview: " <sb-x demo></sb-x>\n"}
	if got := m.InstallMarkup(); got != "<sb-x demo></sb-x>" {
		t.Errorf("without usage: %q", got)
	}
	m.Usage = "<sb-x></sb-x>\n"
	if got := m.InstallMarkup(); got != "<sb-x></sb-x>" {
		t.Errorf("with usage: %q", got)
	}
}
