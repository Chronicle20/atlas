// Command producerseamguardvet is the `go vet -vettool=` entry point for the
// producerseamguard analyzer. See tools/outboxguard/cmd/outboxguardvet for
// why this driver exists alongside cmd/producerseamguard — the analyzer is
// identical either way, only the driver (and its caching behavior) differs.
package main

import (
	"golang.org/x/tools/go/analysis/unitchecker"

	"github.com/Chronicle20/atlas/tools/producerseamguard"
)

func main() {
	unitchecker.Main(producerseamguard.Analyzer)
}
