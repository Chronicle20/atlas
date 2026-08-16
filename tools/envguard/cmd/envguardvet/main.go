// Command envguardvet is the `go vet -vettool=` entry point for the
// envguard analyzer, driven by tools/lib/analyzer-guard.sh's shared cache
// (see tools/rediskeyguard/cmd/rediskeyguardvet for the identical pattern).
package main

import (
	"golang.org/x/tools/go/analysis/unitchecker"

	"github.com/Chronicle20/atlas/tools/envguard"
)

func main() {
	unitchecker.Main(envguard.Analyzer)
}
