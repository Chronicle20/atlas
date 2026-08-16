package saga

import (
	"atlas-channel/kafka/message/saga"
	"encoding/json"
	"testing"
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

// TestExtractResultCharacterId proves the note_send completion branch reads
// Results["characterId"] safely: a nil map, a missing key, and a wrong-typed
// value all resolve to 0 (no panic, no bogus announce), while a JSON-decoded
// float64 (what the orchestrator's Results map actually contains) resolves.
func TestExtractResultCharacterId(t *testing.T) {
	cases := []struct {
		name    string
		results map[string]any
		want    uint32
	}{
		{"nil results", nil, 0},
		{"missing key", map[string]any{"other": float64(5)}, 0},
		{"json float64", map[string]any{"characterId": float64(100)}, 100},
		{"wrong type", map[string]any{"characterId": "100"}, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := extractResultCharacterId(tc.results); got != tc.want {
				t.Errorf("got %d, want %d", got, tc.want)
			}
		})
	}
}

// A meso_sack_use saga can fail three ways: the ceiling rejection, a destroy
// failure, and a timeout. Only the first may claim the meso limit as the reason
// — saying "you cannot hold any more mesos" after a timeout would be a lie.
func TestMesoSackFailureMessage(t *testing.T) {
	cases := []struct {
		code string
		want string
	}{
		{saga.ErrorCodeMesoOverflow, "You cannot hold any more mesos."},
		{saga.ErrorCodeUnknown, "You are unable to use this item right now."},
		{"SAGA_TIMEOUT", "You are unable to use this item right now."},
		{"", "You are unable to use this item right now."},
	}
	for _, tc := range cases {
		if got := mesoSackFailureMessage(tc.code); got != tc.want {
			t.Errorf("mesoSackFailureMessage(%q) = %q, want %q", tc.code, got, tc.want)
		}
	}
}

// A pet_name_tag_use saga can fail on the rename step, the consume step, or by
// timeout, and no atlas-pets error code names a player-actionable cause today,
// so every errorCode maps to the same generic message.
func TestPetNameTagFailureMessage(t *testing.T) {
	cases := []string{
		saga.ErrorCodeUnknown,
		"SAGA_TIMEOUT",
		"",
	}
	want := "You are unable to rename your pet right now."
	for _, code := range cases {
		if got := petNameTagFailureMessage(code); got != want {
			t.Errorf("petNameTagFailureMessage(%q) = %q, want %q", code, got, want)
		}
	}
}
