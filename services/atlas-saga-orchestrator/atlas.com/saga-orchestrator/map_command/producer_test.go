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
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
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

func TestSetEnvironmentStateCommandProvider(t *testing.T) {
	transactionId := uuid.New()
	f := field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(910010000)).SetInstance(uuid.Nil).Build()

	msgs, err := SetEnvironmentStateCommandProvider(transactionId, f, field.ObjectKindObstacle, "obs3", uint32(2))()
	require.NoError(t, err)
	require.Len(t, msgs, 1)
	assert.Equal(t, string(producer.CreateKey(int(f.MapId()))), string(msgs[0].Key))

	var c mapKafka.Command[mapKafka.SetEnvironmentStateCommandBody]
	require.NoError(t, json.Unmarshal(msgs[0].Value, &c))

	assert.Equal(t, transactionId, c.TransactionId)
	assert.Equal(t, world.Id(0), c.WorldId)
	assert.Equal(t, channel.Id(1), c.ChannelId)
	assert.Equal(t, _map.Id(910010000), c.MapId)
	assert.Equal(t, uuid.Nil, c.Instance)
	assert.Equal(t, mapKafka.CommandTypeSetEnvironmentState, c.Type)
	assert.Equal(t, "OBSTACLE", c.Body.Kind)
	assert.Equal(t, "obs3", c.Body.Name)
	assert.Equal(t, uint32(2), c.Body.State)
}

func TestResetEnvironmentCommandProvider(t *testing.T) {
	transactionId := uuid.New()
	f := field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(910010000)).SetInstance(uuid.Nil).Build()

	msgs, err := ResetEnvironmentCommandProvider(transactionId, f)()
	require.NoError(t, err)
	require.Len(t, msgs, 1)
	assert.Equal(t, string(producer.CreateKey(int(f.MapId()))), string(msgs[0].Key))

	var c mapKafka.Command[mapKafka.ResetEnvironmentCommandBody]
	require.NoError(t, json.Unmarshal(msgs[0].Value, &c))

	assert.Equal(t, transactionId, c.TransactionId)
	assert.Equal(t, world.Id(0), c.WorldId)
	assert.Equal(t, channel.Id(1), c.ChannelId)
	assert.Equal(t, _map.Id(910010000), c.MapId)
	assert.Equal(t, uuid.Nil, c.Instance)
	assert.Equal(t, mapKafka.CommandTypeResetEnvironment, c.Type)
}

func TestSetEnvironmentStateCommandProvider_EnvironmentKind(t *testing.T) {
	transactionId := uuid.New()
	f := field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(910010000)).SetInstance(uuid.Nil).Build()

	msgs, err := SetEnvironmentStateCommandProvider(transactionId, f, field.ObjectKindEnvironment, "obs3", uint32(2))()
	require.NoError(t, err)
	require.Len(t, msgs, 1)

	var c mapKafka.Command[mapKafka.SetEnvironmentStateCommandBody]
	require.NoError(t, json.Unmarshal(msgs[0].Value, &c))

	assert.Equal(t, "ENVIRONMENT", c.Body.Kind)
}

func TestSetBackEffectCommandProvider(t *testing.T) {
	transactionId := uuid.New()
	instanceId := uuid.New()
	f := field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(100000000)).SetInstance(instanceId).Build()

	msgs, err := SetBackEffectCommandProvider(transactionId, f, 0, 100000000, 1, 1000)()
	require.NoError(t, err)
	require.Len(t, msgs, 1)
	assert.Equal(t, producer.CreateKey(100000000), msgs[0].Key)

	var c mapKafka.Command[mapKafka.SetBackEffectCommandBody]
	require.NoError(t, json.Unmarshal(msgs[0].Value, &c))

	assert.Equal(t, mapKafka.CommandTypeSetBackEffect, c.Type)
	assert.Equal(t, transactionId, c.TransactionId)
	assert.Equal(t, world.Id(0), c.WorldId)
	assert.Equal(t, channel.Id(1), c.ChannelId)
	assert.Equal(t, _map.Id(100000000), c.MapId)
	assert.Equal(t, instanceId, c.Instance)
	assert.Equal(t, uint8(0), c.Body.Effect)
	assert.Equal(t, uint32(100000000), c.Body.FieldId)
	assert.Equal(t, uint8(1), c.Body.PageId)
	assert.Equal(t, uint32(1000), c.Body.Duration)
}

func TestClearBackEffectCommandProvider(t *testing.T) {
	transactionId := uuid.New()
	instanceId := uuid.New()
	f := field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(100000000)).SetInstance(instanceId).Build()

	msgs, err := ClearBackEffectCommandProvider(transactionId, f)()
	require.NoError(t, err)
	require.Len(t, msgs, 1)
	assert.Equal(t, producer.CreateKey(100000000), msgs[0].Key)

	var c mapKafka.Command[mapKafka.ClearBackEffectCommandBody]
	require.NoError(t, json.Unmarshal(msgs[0].Value, &c))

	assert.Equal(t, mapKafka.CommandTypeClearBackEffect, c.Type)
	assert.Equal(t, transactionId, c.TransactionId)
	assert.Equal(t, world.Id(0), c.WorldId)
	assert.Equal(t, channel.Id(1), c.ChannelId)
	assert.Equal(t, _map.Id(100000000), c.MapId)
	assert.Equal(t, instanceId, c.Instance)
	assert.Equal(t, mapKafka.ClearBackEffectCommandBody{}, c.Body)
}
