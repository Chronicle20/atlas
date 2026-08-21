package map_command

import (
	"encoding/json"
	"testing"

	mapKafka "atlas-saga-orchestrator/kafka/message/map"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

func TestPlayJukeboxCommandProvider(t *testing.T) {
	transactionId := uuid.New()
	instanceId := uuid.New()
	f := field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(100000000)).SetInstance(instanceId).Build()

	msgs, err := PlayJukeboxCommandProvider(transactionId, f, 5100000, "Chronicle", 45000)()
	require.NoError(t, err)
	require.Len(t, msgs, 1)

	var c mapKafka.Command[mapKafka.PlayJukeboxCommandBody]
	require.NoError(t, json.Unmarshal(msgs[0].Value, &c))

	assert.Equal(t, mapKafka.CommandTypePlayJukebox, c.Type)
	assert.Equal(t, transactionId, c.TransactionId)
	assert.Equal(t, world.Id(0), c.WorldId)
	assert.Equal(t, channel.Id(1), c.ChannelId)
	assert.Equal(t, _map.Id(100000000), c.MapId)
	assert.Equal(t, instanceId, c.Instance)
	assert.Equal(t, uint32(5100000), c.Body.ItemId)
	assert.Equal(t, "Chronicle", c.Body.PlayerName)
	assert.Equal(t, uint32(45000), c.Body.DurationMs)
}
