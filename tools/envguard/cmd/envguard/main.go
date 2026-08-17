// Command envguard is the standalone singlechecker driver for the
// envguard analyzer, useful for local `go run ./tools/envguard/cmd/envguard
// <packages...>` iteration. tools/env-domain-guard.sh drives cmd/envguardvet
// instead, through the shared `go vet -vettool=` cache.
package main

import (
	"golang.org/x/tools/go/analysis/singlechecker"

	"github.com/Chronicle20/atlas/tools/envguard"
)

func main() {
	singlechecker.Main(envguard.Analyzer)
}
