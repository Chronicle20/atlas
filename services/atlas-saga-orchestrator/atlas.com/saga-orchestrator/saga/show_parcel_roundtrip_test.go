package saga

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestStep_ShowParcel_JSONRoundTrip asserts a Step[ShowParcelPayload] carrying
// the ShowParcel action survives a full json.Marshal -> json.Unmarshal
// round trip with its payload intact.
//
// This is the regression test for the missing `case ShowParcel:` arm in
// Step[T].UnmarshalJSON (saga/model.go): before the fix, every ShowParcel
// step failed to decode with "unknown action: show_parcel", which meant
// both producers of this step — atlas-channel's Quick Delivery Ticket
// handler and atlas-npc-conversations' open_duey operation — broke at
// runtime before handleShowParcel ever ran.
func TestStep_ShowParcel_JSONRoundTrip(t *testing.T) {
	original := NewStep("step-show-parcel", Pending, ShowParcel, ShowParcelPayload{
		CharacterId: 1001,
		NpcId:       2030,
		WorldId:     0,
		ChannelId:   1,
		Quick:       true,
	})

	marshaled, err := json.Marshal(original)
	assert.NoError(t, err)

	var decoded Step[ShowParcelPayload]
	err = decoded.UnmarshalJSON(marshaled)
	assert.NoError(t, err)

	assert.Equal(t, ShowParcel, decoded.Action())
	assert.Equal(t, original.Payload(), decoded.Payload())
}
