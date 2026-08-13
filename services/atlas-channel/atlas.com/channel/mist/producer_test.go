package mist

import (
	"encoding/json"
	"testing"

	mistmsg "atlas-channel/kafka/message/mist"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// TestCreateCommandProvider_KeySetMatchesAtlasMaps pins the CREATE command's
// JSON shape. atlas-maps owns this contract; the channel mirrors it. A key
// that disagrees produces a mist with silently-zero bounds or lifetime.
func TestCreateCommandProvider_KeySetMatchesAtlasMaps(t *testing.T) {
	body := mistmsg.CreateCommandBody{
		WorldId: 0, ChannelId: 0, MapId: 100000000, Instance: uuid.Nil,
		OwnerType: "CHARACTER", OwnerId: 1001,
		TargetKind: mistmsg.TargetKindMonster,
		EffectKind: mistmsg.EffectKindDamageOverTime,
		OriginX:    500, OriginY: 300,
		LtX: -110, LtY: -82, RbX: 110, RbY: 83,
		Disease: "POISON", DiseaseValue: 0, DiseaseDuration: 4000,
		Duration: 4000, TickIntervalMs: 1000,
		SourceSkillId: 2111003, SourceSkillLevel: 1,
	}

	msgs, err := CreateCommandProvider(body)()
	require.NoError(t, err)
	require.Len(t, msgs, 1)

	var envelope struct {
		Type string          `json:"type"`
		Body json.RawMessage `json:"body"`
	}
	require.NoError(t, json.Unmarshal(msgs[0].Value, &envelope))
	require.Equal(t, "CREATE", envelope.Type)

	var got map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(envelope.Body, &got))
	require.ElementsMatch(t, []string{
		"worldId", "channelId", "mapId", "instance",
		"ownerType", "ownerId", "originX", "originY",
		"ltX", "ltY", "rbX", "rbY",
		"disease", "diseaseValue", "diseaseDuration",
		"duration", "tickIntervalMs",
		"sourceSkillId", "sourceSkillLevel",
		"targetKind", "effectKind",
	}, keysOf(got))
}

func keysOf(m map[string]json.RawMessage) []string {
	ks := make([]string, 0, len(m))
	for k := range m {
		ks = append(ks, k)
	}
	return ks
}
