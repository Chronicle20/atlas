// Command atlasguards-libs-vet is the `go vet -vettool=` entry point carrying
// every analyzer that applies to modules under libs/.
//
// It is a strict subset of atlasguards-services-vet: rediskeyguard and
// outboxguard have always scanned services/ only, and merging the CI jobs must
// not widen them. See package atlasguards.
package main

import (
	"golang.org/x/tools/go/analysis/unitchecker"

	"github.com/Chronicle20/atlas/tools/atlasguards"
)

func main() {
	unitchecker.Main(atlasguards.Libraries()...)
}
