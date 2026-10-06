package web

import (
	"io/fs"
	"regexp"
	"strings"
	"testing"

	"starbase/components"
)

var arrangeHostRe = regexp.MustCompile(`id="([^"]+)"[^>]*data-state="([^"]*)"\s+data-on:[a-z-]+="@get\('/demo/arrange/([a-z-]+)'`)

// Every arrange demo in a README is markup the server renders for its state,
// so the first move doesn't change anything but the order.
func TestArrangeDemosMatchTheServer(t *testing.T) {
	readmes, _ := fs.Glob(components.FS, "*/README.md")
	found := 0
	for _, p := range readmes {
		b, _ := fs.ReadFile(components.FS, p)
		for _, m := range arrangeHostRe.FindAllStringSubmatch(string(b), -1) {
			a, ok := arrangers[m[3]]
			if !ok {
				t.Errorf("%s: no arranger %q", p, m[3])
				continue
			}
			found++
			if want := a.render(m[1], m[2]); !strings.Contains(string(b), want) {
				t.Errorf("%s: the %s demo isn't what the server renders; want:\n%s", p, m[1], want)
			}
		}
	}
	if found == 0 {
		t.Skip("no arrange demos yet")
	}
}

func TestMoveBefore(t *testing.T) {
	for _, tc := range []struct {
		list         string
		item, before string
		want         string
	}{
		{"a b c d", "a", "", "b c d a"},
		{"a b c d", "d", "a", "d a b c"},
		{"a b c d", "b", "d", "a c b d"},
		{"a b c d", "c", "b", "a c b d"},
	} {
		got, err := moveBefore(strings.Fields(tc.list), tc.item, tc.before)
		if err != nil || strings.Join(got, " ") != tc.want {
			t.Errorf("%s: %s before %q = %v %v, want %s", tc.list, tc.item, tc.before, got, err, tc.want)
		}
	}
	for _, bad := range [][2]string{{"x", ""}, {"a", "a"}, {"a", "x"}} {
		if _, err := moveBefore([]string{"a", "b"}, bad[0], bad[1]); err == nil {
			t.Errorf("%q before %q: no error", bad[0], bad[1])
		}
	}
}
