// Command templgen generates the typed templ wrappers in templ/sb (module
// github.com/zweiundeins/starbase/templ) from the components' manifests
// (see internal/templgen).
//
//	go run ./cmd/templgen          # rewrite what is stale
//	go run ./cmd/templgen --check  # fail if anything is stale
package main

import (
	"flag"
	"fmt"
	"os"

	"starbase/internal/templgen"
)

func main() {
	check := flag.Bool("check", false, "fail if a generated file is stale")
	flag.Parse()
	stale, err := templgen.Refresh(".", !*check)
	if err != nil {
		fmt.Fprintln(os.Stderr, "templgen:", err)
		os.Exit(1)
	}
	for _, p := range stale {
		if *check {
			fmt.Fprintf(os.Stderr, "%s is stale: run go tool task templ-wrappers\n", p)
		} else {
			fmt.Println("wrote", p)
		}
	}
	if *check && len(stale) > 0 {
		os.Exit(1)
	}
}
