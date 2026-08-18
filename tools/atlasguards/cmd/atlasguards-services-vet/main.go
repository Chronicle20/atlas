// Command atlasguards-services-vet is the `go vet -vettool=` entry point
// carrying every analyzer that applies to modules under services/.
//
// See package atlasguards for why the guards share one binary and why the
// services/ and libs/ sets differ.
package main

import (
	"golang.org/x/tools/go/analysis/unitchecker"

	"github.com/Chronicle20/atlas/tools/atlasguards"
)

func main() {
	unitchecker.Main(atlasguards.Services()...)
}
