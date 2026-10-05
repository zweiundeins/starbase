// Command fetchtsc puts the pinned TypeScript compiler for a build target into
// internal/tscheck/dist, where the binary embeds it (see internal/tscheck).
// Run it before `go build`; without it the binary has no type check.
//
//	go run ./cmd/fetchtsc                        # this machine's target
//	go run ./cmd/fetchtsc -os linux -arch arm64  # another target
//	go run ./cmd/fetchtsc -unpack DIR            # also unpack this machine's compiler into DIR
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"starbase/internal/tscheck"
)

const dist = "internal/tscheck/dist"

func main() {
	goos := flag.String("os", runtime.GOOS, "target GOOS")
	goarch := flag.String("arch", runtime.GOARCH, "target GOARCH")
	unpack := flag.String("unpack", "", "directory to unpack the compiler into (this machine's target only)")
	flag.Parse()
	if err := run(*goos, *goarch, *unpack); err != nil {
		fmt.Fprintln(os.Stderr, "fetchtsc:", err)
		os.Exit(1)
	}
}

func run(goos, goarch, unpack string) error {
	p := tscheck.Platform(goos, goarch)
	if unpack != "" && (goos != runtime.GOOS || goarch != runtime.GOARCH) {
		return fmt.Errorf("-unpack needs this machine's target, not %s/%s", goos, goarch)
	}
	// Only one target's compiler may be embedded.
	old, _ := filepath.Glob(filepath.Join(dist, "*.tgz"))
	for _, f := range old {
		if filepath.Base(f) != tscheck.Tarball(p) {
			os.Remove(f)
		}
	}
	if p == "" {
		fmt.Printf("TypeScript has no compiler for %s/%s: the binary will have no type check\n", goos, goarch)
		return nil
	}
	file := filepath.Join(dist, tscheck.Tarball(p))
	tgz, err := os.ReadFile(file)
	if err != nil || tscheck.Verify(p, tgz) != nil {
		if tgz, err = tscheck.Fetch(context.Background(), p); err != nil {
			return err
		}
		if err := os.WriteFile(file, tgz, 0o644); err != nil {
			return err
		}
		fmt.Printf("fetched %s (%d bytes)\n", file, len(tgz))
	}
	if unpack == "" {
		return nil
	}
	if err := os.MkdirAll(unpack, 0o755); err != nil {
		return err
	}
	return tscheck.Unpack(tgz, unpack)
}
