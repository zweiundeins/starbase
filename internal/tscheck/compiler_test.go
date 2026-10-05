package tscheck

import "testing"

// Every target the release builds has a pinned compiler.
func TestPlatformCoversReleaseTargets(t *testing.T) {
	for _, target := range [][2]string{{"linux", "amd64"}, {"linux", "arm64"}, {"darwin", "amd64"}, {"darwin", "arm64"}, {"windows", "amd64"}} {
		if Platform(target[0], target[1]) == "" {
			t.Errorf("no TypeScript compiler pinned for %s/%s", target[0], target[1])
		}
	}
	if p := Platform("freebsd", "amd64"); p != "" {
		t.Errorf("freebsd/amd64: %q, want none", p)
	}
}
