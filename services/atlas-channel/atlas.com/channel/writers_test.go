package main

import (
	"testing"

	fieldcb "github.com/Chronicle20/atlas/libs/atlas-packet/field/clientbound"
)

// TestProduceWriters_RegistersMtsWriters guards against the silent gap that broke
// the MTS "Charge" button: a handler announced MtsChargeParamResult, the tenant
// config mapped its opcode, but the writer name was missing from produceWriters()
// (the code-side availableWriters list). BuildWriterProducer only registers a
// writer present in BOTH config AND that list — and warns only for the opposite
// mismatch — so the omission surfaced only as a runtime "writer not found" and a
// frozen client. Every MTS clientbound writer a handler can announce must be here.
func TestProduceWriters_RegistersMtsWriters(t *testing.T) {
	registered := make(map[string]bool)
	for _, w := range produceWriters() {
		registered[w] = true
	}

	for _, name := range []string{
		fieldcb.MtsOperationWriter,
		fieldcb.MtsOperation2Writer,
		fieldcb.MtsChargeParamResultWriter,
	} {
		if !registered[name] {
			t.Errorf("produceWriters() must register writer [%s] or Announce fails with 'writer not found'", name)
		}
	}
}
