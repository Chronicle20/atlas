// Command goroutineguardvet is the `go vet -vettool=` entry point for the
// goroutineguard analyzer.
//
// It exists alongside cmd/goroutineguard (a singlechecker driver) purely for
// speed. A standalone singlechecker re-parses and re-type-checks every package
// from source on each run, so a warm run costs exactly what a cold one does.
// Driven through `go vet -vettool=`, the go command caches the analyzer's
// facts and diagnostics per package in GOCACHE, so unchanged packages are not
// re-analyzed at all.
//
// The analyzer is identical either way — only the driver differs.
package main

import (
	"github.com/Chronicle20/atlas/tools/goroutineguard"
	"golang.org/x/tools/go/analysis/unitchecker"
)

func main() {
	unitchecker.Main(goroutineguard.Analyzer)
}
