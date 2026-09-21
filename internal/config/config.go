// Package config loads runtime configuration from the environment.
package config

import (
	"bufio"
	"cmp"
	"os"
	"strings"
)

type Config struct {
	Addr    string // listen address, e.g. ":8080"
	BaseURL string // public origin, used for OAuth redirects
	DBPath  string

	GitHubClientID     string
	GitHubClientSecret string

	// RepoURL is where the component sources live; used for "Edit on GitHub" links.
	RepoURL string

	Dev bool
}

// Load reads configuration from the environment, after filling unset
// variables from a .env file in the working directory, if present.
func Load() Config {
	loadDotEnv(".env")
	return Config{
		Addr:               cmp.Or(os.Getenv("ADDR"), ":8080"),
		BaseURL:            cmp.Or(os.Getenv("BASE_URL"), "http://localhost:8080"),
		DBPath:             cmp.Or(os.Getenv("DB_PATH"), "data/starbase.db"),
		GitHubClientID:     os.Getenv("GITHUB_CLIENT_ID"),
		GitHubClientSecret: os.Getenv("GITHUB_CLIENT_SECRET"),
		RepoURL:            cmp.Or(os.Getenv("REPO_URL"), "https://github.com/zweiundeins/starbase"),
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
