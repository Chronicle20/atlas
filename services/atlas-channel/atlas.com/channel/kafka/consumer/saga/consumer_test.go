package saga

import (
	"encoding/json"
	"testing"

	"atlas-channel/kafka/message/saga"
)

// TestResultDecoders_TolerateJSONFloat64 proves the COMPLETED Results decoders
// read the marker + characterId off a map that has been through a JSON round-trip
// (numeric values become float64), matching what the orchestrator emits for a
// take-home saga. A wrong decode here would drop the take-home notice.
func TestResultDecoders_TolerateJSONFloat64(t *testing.T) {
	raw := []byte(`{"kind":"mts_take_home","characterId":1001,"templateId":1402001}`)
	var results map[string]any
	if err := json.Unmarshal(raw, &results); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if got := resultKind(results); got != saga.MtsTakeHomeResultKind {
		t.Fatalf("resultKind = %q, want %q", got, saga.MtsTakeHomeResultKind)
	}
	if got := resultUint32(results, "characterId"); got != 1001 {
		t.Fatalf("resultUint32(characterId) = %d, want 1001", got)
	}
}

// TestResultDecoders_MissingAndNil proves the decoders are safe on a nil map and a
// missing/typed-wrong key (returns zero values, not a panic) so a non-take-home
// COMPLETED event is a clean no-op.
func TestResultDecoders_MissingAndNil(t *testing.T) {
	if got := resultKind(nil); got != "" {
		t.Fatalf("resultKind(nil) = %q, want empty", got)
	}
	if got := resultUint32(nil, "characterId"); got != 0 {
		t.Fatalf("resultUint32(nil) = %d, want 0", got)
	}
	if got := resultUint32(map[string]any{"characterId": "not-a-number"}, "characterId"); got != 0 {
		t.Fatalf("resultUint32 of non-numeric = %d, want 0", got)
	}
}

// TestMtsFailureArm_KnownKinds proves each MTS failure kind maps to a non-nil
// clientbound *Failed body so handleFailedEvent unhangs the matching dialog.
func TestMtsFailureArm_KnownKinds(t *testing.T) {
	for _, kind := range []string{saga.MtsFailureKindBuy, saga.MtsFailureKindList, saga.MtsFailureKindTakeHome} {
		body, ok := mtsFailureArm(kind)
		if !ok {
			t.Fatalf("mtsFailureArm(%q) ok = false, want true", kind)
		}
		if body == nil {
			t.Fatalf("mtsFailureArm(%q) body = nil, want non-nil", kind)
		}
	}
}

// TestMtsFailureArm_UnknownKind proves an unknown/empty kind returns ok=false so
// the handler skips notifying rather than sending the wrong dialog arm.
func TestMtsFailureArm_UnknownKind(t *testing.T) {
	for _, kind := range []string{"", "mts_bid", "garbage"} {
		if _, ok := mtsFailureArm(kind); ok {
			t.Fatalf("mtsFailureArm(%q) ok = true, want false", kind)
		}
	}
}
