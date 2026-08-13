package saga

import "testing"

// The saga consumer compares e.Body.SagaType against this string copy, not
// against the atlas-saga Type. The two are separate declarations in separate
// modules; if they drift, every meso_sack_use failure silently skips its
// client-notification arm and the player stays input-locked.
func TestSagaTypeMesoSackUseString(t *testing.T) {
	if SagaTypeMesoSackUse != "meso_sack_use" {
		t.Fatalf("SagaTypeMesoSackUse = %q, want %q", SagaTypeMesoSackUse, "meso_sack_use")
	}
}

// atlas-character emits this exact string as StatusEventMesoErrorBody.Error;
// the orchestrator threads it verbatim onto the saga-failed event's errorCode.
func TestErrorCodeMesoOverflowString(t *testing.T) {
	if ErrorCodeMesoOverflow != "MESO_OVERFLOW" {
		t.Fatalf("ErrorCodeMesoOverflow = %q, want %q", ErrorCodeMesoOverflow, "MESO_OVERFLOW")
	}
}
