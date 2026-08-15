// Command scopeguardvet is the `go vet -vettool=` entry point for the
// scopeguard analyzer. See tools/rediskeyguard/cmd/rediskeyguardvet for why
// this exists alongside a plain singlechecker driver: `go vet -vettool=`
// lets the go command cache facts/diagnostics per package in GOCACHE, so
// unchanged packages are not re-analyzed on a warm run.
package main

import (
	"golang.org/x/tools/go/analysis/unitchecker"

	"github.com/Chronicle20/atlas/tools/scopeguard"
)

func main() {
	unitchecker.Main(scopeguard.Analyzer)
}
