// Package static embeds the site's CSS, fonts, images and vendored JS.
package static

import "embed"

//go:embed css fonts vendor
var FS embed.FS

// Datastar is the patched Datastar + Rocket build (scripts/vendor-rocket.sh):
// pages load it, and install snippets pin it as a versioned file.
func Datastar() []byte {
	b, _ := FS.ReadFile("vendor/datastar-rocket.js")
	return b
}
