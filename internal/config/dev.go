//go:build dev

package config

// Dev is true when built with -tags=dev: assets are served from disk,
// the browser live-reloads and the fake GitHub login is available.
const Dev = true
