// Command scopeguard is a standalone singlechecker driver for the
// scopeguard analyzer — the local/dev interface and single-guard escape
// hatch alongside cmd/scopeguardvet's `go vet -vettool=` driver.
package main

import (
	"golang.org/x/tools/go/analysis/singlechecker"

	"github.com/Chronicle20/atlas/tools/scopeguard"
)

func main() {
	singlechecker.Main(scopeguard.Analyzer)
}
