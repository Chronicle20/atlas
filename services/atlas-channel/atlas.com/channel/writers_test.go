package main

import (
	"testing"

	fieldcb "github.com/Chronicle20/atlas/libs/atlas-packet/field/clientbound"
	parcelcb "github.com/Chronicle20/atlas/libs/atlas-packet/parcel/clientbound"
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

// TestProduceWriters_RegistersParcelWriter guards against the bug where clicking
// Duey opened no dialog: the SHOW_PARCEL handler announced parcelcb.ParcelWriter,
// the tenant config mapped its opcode, but the writer name was missing from
// produceWriters() (the code-side availableWriters list). BuildWriterProducer only
// registers a writer present in BOTH config AND that list — and warns only for the
// opposite mismatch — so the omission surfaced only as a runtime "writer not found"
// and a silently swallowed dialog.
func TestProduceWriters_RegistersParcelWriter(t *testing.T) {
	registered := make(map[string]bool)
	for _, w := range produceWriters() {
		registered[w] = true
	}

	if !registered[parcelcb.ParcelWriter] {
		t.Errorf("produceWriters() must register writer [%s] or Announce fails with 'writer not found'", parcelcb.ParcelWriter)
	}
}
