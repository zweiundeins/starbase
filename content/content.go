// Package content embeds the markdown for the site's editorial pages.
package content

import "embed"

//go:embed *.md
var FS embed.FS
