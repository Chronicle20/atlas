package timer

import (
	"encoding/json"
	"testing"

	characterKafka "atlas-maps/kafka/message/character"
	mapKafka "atlas-maps/kafka/message/map"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestMapTimerStartedProvider_BuildsCorrectEvent(t *testing.T) {
	txn := uuid.MustParse("11111111-2222-3333-4444-555555555555")
	f := field.NewBuilder(world.Id(1), channel.Id(2), _map.Id(100000000)).SetInstance(uuid.Nil).Build()
	prov := mapTimerStartedProvider(txn, f, uint32(42), uint32(600))
	msgs, err := prov()
	require.NoError(t, err)
	require.Len(t, msgs, 1)

	var ev mapKafka.StatusEvent[mapKafka.MapTimerStarted]
	require.NoError(t, json.Unmarshal(msgs[0].Value, &ev))
	require.Equal(t, mapKafka.EventTopicMapStatusTypeMapTimerStarted, ev.Type)
	require.Equal(t, txn, ev.TransactionId)
	require.Equal(t, world.Id(1), ev.WorldId)
	require.Equal(t, channel.Id(2), ev.ChannelId)
	require.Equal(t, _map.Id(100000000), ev.MapId)
	require.Equal(t, uint32(42), ev.Body.CharacterId)
	require.Equal(t, uint32(600), ev.Body.Seconds)
}

func TestChangeMapProvider_BuildsCorrectCommand(t *testing.T) {
	txn := uuid.MustParse("11111111-2222-3333-4444-555555555555")
	prov := changeMapProvider(txn, uint32(42), world.Id(1), channel.Id(2), _map.Id(100000201))
	msgs, err := prov()
	require.NoError(t, err)
	require.Len(t, msgs, 1)

	var cmd characterKafka.Command[characterKafka.ChangeMapBody]
	require.NoError(t, json.Unmarshal(msgs[0].Value, &cmd))
	require.Equal(t, characterKafka.CommandChangeMap, cmd.Type)
	require.Equal(t, txn, cmd.TransactionId)
	require.Equal(t, world.Id(1), cmd.WorldId)
	require.Equal(t, uint32(42), cmd.CharacterId)
	require.Equal(t, channel.Id(2), cmd.Body.ChannelId)
	require.Equal(t, _map.Id(100000201), cmd.Body.MapId)
	require.Equal(t, uuid.Nil, cmd.Body.Instance, "forced-return goes to non-instanced field")
	require.Equal(t, uint32(0), cmd.Body.PortalId, "default spawn portal")
}
