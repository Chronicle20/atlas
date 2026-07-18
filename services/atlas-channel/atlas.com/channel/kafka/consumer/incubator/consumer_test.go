package incubator

import (
	incubator2 "atlas-channel/kafka/message/incubator"
	"context"
	"encoding/json"
	"io"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// TestToIncubatorResult_EggIdSurvives proves the sacrificed Pigmy Egg id on a
// ResultEvent reaches NewIncubatorResult's 3rd argument (gachaponItemId)
// unchanged. A field swap or wrong-arg-order regression in toIncubatorResult
// must make this test fail.
func TestToIncubatorResult_EggIdSurvives(t *testing.T) {
	event := incubator2.ResultEvent{
		CharacterId: 12345,
		WorldId:     0,
		ChannelId:   1,
		ItemId:      5000000,
		Count:       1,
		EggId:       4170003,
	}

	result := toIncubatorResult(event)

	require.Equal(t, event.ItemId, result.ItemId())
	require.Equal(t, uint16(event.Count), result.Count())
	require.Equal(t, event.EggId, result.GachaponItemId(), "EggId must reach IncubatorResult as gachaponItemId")
}

// TestToIncubatorResult_EggIdSurvivesJSONRoundTrip proves EggId also survives
// decoding the wire JSON body the saga-orchestrator producer emits (guards a
// struct-tag typo on ResultEvent, not just an in-process mapping bug).
func TestToIncubatorResult_EggIdSurvivesJSONRoundTrip(t *testing.T) {
	body := []byte(`{"characterId":12345,"worldId":0,"channelId":1,"itemId":5000000,"count":1,"eggId":4170007}`)

	var event incubator2.ResultEvent
	require.NoError(t, json.Unmarshal(body, &event))
	require.Equal(t, uint32(4170007), event.EggId)

	result := toIncubatorResult(event)
	require.Equal(t, uint32(4170007), result.GachaponItemId())
}

// TestToIncubatorResult_EncodeCarriesEggIdOnV95 proves the mapped
// IncubatorResult's Encode output actually carries the egg id on the wire for
// a GMS v95 tenant (the client version that reads gachaponItemId), closing
// the loop from ResultEvent through toIncubatorResult to the encoded bytes.
func TestToIncubatorResult_EncodeCarriesEggIdOnV95(t *testing.T) {
	event := incubator2.ResultEvent{
		CharacterId: 12345,
		WorldId:     0,
		ChannelId:   1,
		ItemId:      5000000,
		Count:       1,
		EggId:       4170003,
	}

	tm, err := tenant.Create(uuid.New(), "GMS", 95, 1)
	require.NoError(t, err)
	ctx := tenant.WithContext(context.Background(), tm)

	logger := logrus.New()
	logger.SetOutput(io.Discard)
	b := toIncubatorResult(event).Encode(logger, ctx)(map[string]interface{}{})

	// itemId(4) + count(2) + gachaponItemId(4) + bonusItemId(4) + bonusCount(4) = 18 bytes on v95.
	require.Len(t, b, 18)
	require.Equal(t, uint32(4170003), byteOrderUint32LE(b[6:10]), "encoded gachaponItemId must equal the ResultEvent's EggId")
}

func byteOrderUint32LE(b []byte) uint32 {
	return uint32(b[0]) | uint32(b[1])<<8 | uint32(b[2])<<16 | uint32(b[3])<<24
}
