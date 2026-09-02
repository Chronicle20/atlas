package environment

import (
	"encoding/json"
	"testing"

	mapKafka "atlas-maps/kafka/message/map"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
)

func TestEnvironmentStateChangedEventProvider(t *testing.T) {
	transactionId := uuid.New()
	f := field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(910010000)).SetInstance(uuid.Nil).Build()
	entry := ObjectEntry{Kind: field.ObjectKindObstacle, Name: "obs3", State: 2}

	msgs, err := EnvironmentStateChangedEventProvider(transactionId, f, entry)()
	require.NoError(t, err)
	require.Len(t, msgs, 1)

	assert.Equal(t, string(producer.CreateKey(int(f.MapId()))), string(msgs[0].Key))

	var e mapKafka.StatusEvent[mapKafka.EnvironmentStateChanged]
	require.NoError(t, json.Unmarshal(msgs[0].Value, &e))

	assert.Equal(t, mapKafka.EventTopicMapStatusTypeEnvironmentStateChanged, e.Type)
	assert.Equal(t, world.Id(0), e.WorldId)
	assert.Equal(t, channel.Id(1), e.ChannelId)
	assert.Equal(t, _map.Id(910010000), e.MapId)
	assert.Equal(t, uuid.Nil, e.Instance)
	assert.Equal(t, "OBSTACLE", e.Body.Kind)
	assert.Equal(t, "obs3", e.Body.Name)
	assert.Equal(t, uint32(2), e.Body.State)
	assert.Equal(t, transactionId, e.TransactionId)
}

func TestEnvironmentResetEventProvider(t *testing.T) {
	transactionId := uuid.New()
	f := field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(910010000)).SetInstance(uuid.Nil).Build()
	cleared := []ObjectEntry{
		{Kind: field.ObjectKindObstacle, Name: "a", State: 1},
		{Kind: field.ObjectKindEnvironment, Name: "b", State: 2, DefaultState: 1},
	}

	msgs, err := EnvironmentResetEventProvider(transactionId, f, cleared)()
	require.NoError(t, err)
	require.Len(t, msgs, 1)

	var e mapKafka.StatusEvent[mapKafka.EnvironmentReset]
	require.NoError(t, json.Unmarshal(msgs[0].Value, &e))

	assert.Equal(t, mapKafka.EventTopicMapStatusTypeEnvironmentReset, e.Type)
	require.Len(t, e.Body.Cleared, 2)
	// The event carries each object's DECLARED DEFAULT, not the state it was
	// cleared from: that is the value the channel restores.
	assert.Equal(t, mapKafka.EnvironmentObject{Kind: "OBSTACLE", Name: "a", State: 0}, e.Body.Cleared[0])
	assert.Equal(t, mapKafka.EnvironmentObject{Kind: "ENVIRONMENT", Name: "b", State: 1}, e.Body.Cleared[1])
}

func TestEnvironmentResetEventProvider_EmptyCleared(t *testing.T) {
	transactionId := uuid.New()
	f := field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(910010000)).SetInstance(uuid.Nil).Build()

	msgs, err := EnvironmentResetEventProvider(transactionId, f, []ObjectEntry{})()
	require.NoError(t, err)
	require.Len(t, msgs, 1)

	var raw map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(msgs[0].Value, &raw))

	var body map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(raw["body"], &body))

	assert.JSONEq(t, "[]", string(body["cleared"]))
}
