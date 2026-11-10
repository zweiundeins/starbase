// Command tsgen compiles the TypeScript components into the .js files next
// to them (see internal/tsgen). It needs the compiler embedded: run
// `go run ./cmd/fetchtsc` first (`go tool task ts` does).
//
//	go run ./cmd/tsgen          # rewrite stale .js files
//	go run ./cmd/tsgen --check  # fail if one is stale
package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"starbase/internal/tscheck"
	"starbase/internal/tsgen"
)

func main() {
	check := flag.Bool("check", false, "fail if a generated .js is stale")
	flag.Parse()
	dir, err := os.MkdirTemp("", "tsgen-")
	if err != nil {
		fail(err)
	}
	defer os.RemoveAll(dir)
	c := tscheck.New(dir)
	if c == nil {
		fail(fmt.Errorf("no compiler embedded: run go run ./cmd/fetchtsc"))
	}
	res, err := tsgen.Refresh(context.Background(), c, ".", !*check)
	if err != nil {
		fail(err)
	}
	failed := false
	for _, r := range res {
		for _, d := range r.Diagnostics {
			fmt.Fprintf(os.Stderr, "components/%s:%d:%d: TS%d %s\n", d.File, d.Line, d.Col, d.Code, d.Message)
			failed = true
		}
		switch {
		case r.Stale && *check:
			fmt.Fprintf(os.Stderr, "%s: its .js is stale; run go tool task ts\n", r.Source)
			failed = true
		case r.Stale:
			fmt.Printf("compiled %s\n", r.Source)
		}
	}
	if failed {
		os.RemoveAll(dir)
		os.Exit(1)
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "tsgen:", err)
	os.Exit(1)
}
