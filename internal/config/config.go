// Package config loads runtime configuration from the environment.
package config

import (
	"bufio"
	"cmp"
	"os"
	"strings"
	"time"
)

type Config struct {
	Addr    string // listen address, e.g. ":7331"
	BaseURL string // public origin, used for OAuth redirects
	DBPath  string
	// MetricsAddr is where the metrics for Prometheus listen, a loopback
	// host:port; empty is none. It is never the public listener.
	MetricsAddr string

	GitHubClientID     string
	GitHubClientSecret string

	// RepoURL is where the component sources live; used for "Edit on GitHub" links.
	RepoURL string
	// GitHubToken (optional, read-only) raises the GitHub API rate limit for
	// pull request preview checks.
	GitHubToken string

	Dev bool

	// Heartbeat is how often a render stream pings the page's watchdog
	// (StreamHeartbeat); zero is the default.
	Heartbeat time.Duration

	Version string // set by main from build flags
}

// StreamHeartbeat is Heartbeat, 5 s unless set. The page reopens a stream
// that misses three.
func (c Config) StreamHeartbeat() time.Duration { return cmp.Or(c.Heartbeat, 5*time.Second) }

// Load reads configuration from the environment, after filling unset
// variables from a .env file in the working directory, if present.
func Load() Config {
	loadDotEnv(".env")
	return Config{
		Addr:               cmp.Or(os.Getenv("ADDR"), ":7331"),
		BaseURL:            cmp.Or(os.Getenv("BASE_URL"), "http://localhost:7331"),
		DBPath:             cmp.Or(os.Getenv("DB_PATH"), "data/starbase.db"),
		MetricsAddr:        os.Getenv("METRICS_ADDR"),
		GitHubClientID:     os.Getenv("GITHUB_CLIENT_ID"),
		GitHubClientSecret: os.Getenv("GITHUB_CLIENT_SECRET"),
		RepoURL:            cmp.Or(os.Getenv("REPO_URL"), "https://github.com/zweiundeins/starbase"),
		GitHubToken:        os.Getenv("GITHUB_TOKEN"),
		Dev:                Dev,
	}
}

// GitHubEnabled reports whether real OAuth credentials are configured.
func (c Config) GitHubEnabled() bool {
	return c.GitHubClientID != "" && c.GitHubClientSecret != ""
}

func loadDotEnv(path string) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		k, v = strings.TrimSpace(k), strings.Trim(strings.TrimSpace(v), `"'`)
		if _, set := os.LookupEnv(k); !set {
			os.Setenv(k, v)
		}
	}
}
