package data

import (
	"testing"

	"atlas-data/data/workers"
)

func containsWorker(ws []workers.Worker, name string) bool {
	for _, w := range ws {
		if w.Name() == name {
			return true
		}
	}
	return false
}

// The String worker populates the item-name registry that the Item worker
// resolves names from during ingest. Running them concurrently races and leaves
// item/pet names empty, so String must be split out as a prerequisite that runs
// to completion before the parallel fan-out.
func TestSplitPrerequisites(t *testing.T) {
	prereq, rest := splitPrerequisites(workers.Registered)

	if len(prereq)+len(rest) != len(workers.Registered) {
		t.Fatalf("partition dropped/duplicated workers: %d + %d != %d", len(prereq), len(rest), len(workers.Registered))
	}
	if !containsWorker(prereq, "STRING") {
		t.Fatal("STRING worker must be a prerequisite")
	}
	if containsWorker(rest, "STRING") {
		t.Fatal("STRING worker must not be in the parallel phase")
	}
	// Every other registered worker (including Item, which consumes the
	// string registry) runs in the parallel phase.
	for _, w := range workers.Registered {
		if w.Name() == "STRING" {
			continue
		}
		if !containsWorker(rest, w.Name()) {
			t.Fatalf("worker %s missing from the parallel phase", w.Name())
		}
	}
}
