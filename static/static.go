// Package static embeds the site's CSS, fonts, images and vendored JS.
package static

import "embed"

//go:embed css fonts vendor
var FS embed.FS
