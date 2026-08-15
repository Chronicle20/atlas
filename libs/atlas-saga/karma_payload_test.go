package saga

import (
	"encoding/json"
	"testing"
)

// TestApplyAssetKarmaPayloadUnmarshal proves the action's arm exists in
// Step.UnmarshalJSON: without it the payload decodes to nil and the orchestrator
// fails the step with "invalid payload".
func TestApplyAssetKarmaPayloadUnmarshal(t *testing.T) {
	raw := []byte(`{
		"stepId": "apply_asset_karma",
		"status": "pending",
		"action": "apply_asset_karma",
		"payload": {"characterId": 42, "inventoryType": 1, "slot": 3, "scissorsKarma": 2}
	}`)

	var st Step[any]
	if err := json.Unmarshal(raw, &st); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	p, ok := st.Payload.(ApplyAssetKarmaPayload)
	if !ok {
		t.Fatalf("payload type = %T, want ApplyAssetKarmaPayload", st.Payload)
	}
	if p.CharacterId != 42 || p.InventoryType != 1 || p.Slot != 3 {
		t.Fatalf("payload = %+v, want {42 1 3 ...}", p)
	}
	// ScissorsKarma is what lets atlas-inventory re-run the EQUALITY half of the
	// eligibility predicate. Dropping it would silently weaken the v87+ model to
	// the v83 non-zero model at the owning service.
	if p.ScissorsKarma != 2 {
		t.Fatalf("ScissorsKarma = %d, want 2", p.ScissorsKarma)
	}
}
