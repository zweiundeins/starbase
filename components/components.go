// Package components embeds the community component collection.
// Every sub-folder is one component; see CONTRIBUTING in /contribute.
package components

import "embed"

//go:embed */*
var FS embed.FS
