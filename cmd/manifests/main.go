// Command manifests regenerates components/*/manifest.json.
//
// Rocket only knows a component's props, slots and events at runtime, in a
// browser. This tool runs the app in-process (dev build), opens the gallery
// in headless Chrome, and lets the page publish every manifest to the dev
// endpoint, which writes the files.
//
//	go run -tags dev ./cmd/manifests          # write/refresh manifests
//	go run -tags dev ./cmd/manifests --check  # fail if any was missing or stale (CI)
package main

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"time"

	"starbase/internal/app"
	"starbase/internal/config"
)

func main() {
	check := flag.Bool("check", false, "fail if a manifest was missing or out of date")
	chrome := flag.String("chrome", os.Getenv("CHROME"), "Chrome/Chromium binary (default: autodetect)")
	flag.Parse()
	if err := run(*check, *chrome); err != nil {
		fmt.Fprintln(os.Stderr, "manifests:", err)
		os.Exit(1)
	}
}

func findChrome() (string, error) {
	for _, name := range []string{"google-chrome", "google-chrome-stable", "chromium", "chromium-browser", "chrome"} {
		if p, err := exec.LookPath(name); err == nil {
			return p, nil
		}
	}
	return "", errors.New("no Chrome/Chromium found; pass --chrome or set CHROME")
}

func run(check bool, chrome string) error {
	if !config.Dev {
		return errors.New("build with -tags dev: go run -tags dev ./cmd/manifests")
	}
	if chrome == "" {
		var err error
		if chrome, err = findChrome(); err != nil {
			return err
		}
	}

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return err
	}
	base := "http://" + ln.Addr().String()
	tmp, err := os.MkdirTemp("", "starbase-manifests-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)

	cfg := config.Load()
	cfg.BaseURL = base
	cfg.DBPath = filepath.Join(tmp, "db.sqlite")
	cfg.GitHubClientID, cfg.GitHubClientSecret = "", ""
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	a, err := app.New(ctx, cfg, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		return err
	}
	defer a.Close()
	srv := &http.Server{Handler: a.Handler}
	go srv.Serve(ln)
	defer srv.Close()

	before := map[string][]byte{}
	var paths []string
	for _, c := range a.Catalog.Components {
		p := filepath.Join("components", c.Slug, "manifest.json")
		paths = append(paths, p)
		before[p], _ = os.ReadFile(p) // nil if missing
	}

	cmd := exec.Command(chrome, "--headless=new", "--no-sandbox", "--disable-gpu",
		"--virtual-time-budget=10000", "--dump-dom", base+"/")
	cmd.Stdout = io.Discard
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	done := make(chan error, 1)
	go func() { done <- cmd.Run() }()
	select {
	case err := <-done:
		if err != nil {
			return fmt.Errorf("chrome: %w\n%s", err, stderr.String())
		}
	case <-time.After(90 * time.Second):
		cmd.Process.Kill()
		return errors.New("chrome timed out")
	}

	var missing, changed []string
	for _, p := range paths {
		after, err := os.ReadFile(p)
		switch {
		case err != nil:
			missing = append(missing, p)
		case !bytes.Equal(before[p], after):
			changed = append(changed, p)
		}
	}
	slices.Sort(changed)
	for _, p := range changed {
		fmt.Println("updated", p)
	}
	if len(missing) > 0 {
		return fmt.Errorf("no manifest published for: %v (does the component load without errors?)", missing)
	}
	if check && len(changed) > 0 {
		return fmt.Errorf("%d manifest(s) were out of date; run `go tool task manifests` and commit them", len(changed))
	}
	fmt.Printf("%d manifests up to date\n", len(paths)-len(changed))
	return nil
}
