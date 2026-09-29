// Command listings rewrites the Go listings in the docs from their source
// (see internal/listings).
//
//	go run ./cmd/listings          # rewrite stale listings
//	go run ./cmd/listings --check  # fail if any listing is stale
package main

import (
	"flag"
	"fmt"
	"os"

	"starbase/internal/listings"
)

func main() {
	check := flag.Bool("check", false, "fail if a listing doesn't match its source")
	flag.Parse()
	stale, err := listings.Refresh(".", !*check)
	if err != nil {
		fmt.Fprintln(os.Stderr, "listings:", err)
		os.Exit(1)
	}
	failed := false
	for _, s := range stale {
		switch {
		case s.Err != nil:
			fmt.Fprintf(os.Stderr, "%s:%d: %v\n", s.Doc, s.Line, s.Err)
			failed = true
		case *check:
			fmt.Fprintf(os.Stderr, "%s:%d: stale; run go tool task listings\n", s.Doc, s.Line)
			failed = true
		default:
			fmt.Printf("updated %s:%d\n", s.Doc, s.Line)
		}
	}
	if failed {
		os.Exit(1)
	}
}
