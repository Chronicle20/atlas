package field

import (
	"atlas-maps/map/character"
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	cfield "github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// TestProcessorGetFieldsFilters exercises the processor's filter behavior
// directly, independent of the HTTP handler.
func TestProcessorGetFieldsFilters(t *testing.T) {
	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	require.NoError(t, err)
	ctx := tenant.WithContext(context.Background(), ten)

	f1 := cfield.NewBuilder(world.Id(0), channel.Id(1), _map.Id(100000000)).SetInstance(uuid.Nil).Build()
	f2 := cfield.NewBuilder(world.Id(0), channel.Id(2), _map.Id(100000000)).SetInstance(uuid.Nil).Build()
	f3 := cfield.NewBuilder(world.Id(1), channel.Id(1), _map.Id(200000000)).SetInstance(uuid.Nil).Build()

	cp := character.NewProcessor(logrus.New(), ctx)
	cp.Enter(uuid.New(), f1, 1)
	cp.Enter(uuid.New(), f2, 2)
	cp.Enter(uuid.New(), f3, 3)

	p := NewProcessor(logrus.New(), ctx)

	t.Run("no filter returns all, sorted", func(t *testing.T) {
		result := p.GetFields(ten, nil, nil, nil)
		require.Len(t, result, 3)
		assert.Equal(t, world.Id(0), result[0].Field.WorldId())
		assert.Equal(t, channel.Id(1), result[0].Field.ChannelId())
		assert.Equal(t, world.Id(0), result[1].Field.WorldId())
		assert.Equal(t, channel.Id(2), result[1].Field.ChannelId())
		assert.Equal(t, world.Id(1), result[2].Field.WorldId())
	})

	t.Run("world filter", func(t *testing.T) {
		wid := world.Id(0)
		result := p.GetFields(ten, &wid, nil, nil)
		require.Len(t, result, 2)
		for _, r := range result {
			assert.Equal(t, world.Id(0), r.Field.WorldId())
		}
	})

	t.Run("channel filter", func(t *testing.T) {
		cid := channel.Id(1)
		result := p.GetFields(ten, nil, &cid, nil)
		require.Len(t, result, 2)
		for _, r := range result {
			assert.Equal(t, channel.Id(1), r.Field.ChannelId())
		}
	})

	t.Run("map filter", func(t *testing.T) {
		mid := _map.Id(200000000)
		result := p.GetFields(ten, nil, nil, &mid)
		require.Len(t, result, 1)
		assert.Equal(t, _map.Id(200000000), result[0].Field.MapId())
	})

	t.Run("all three filters", func(t *testing.T) {
		wid := world.Id(0)
		cid := channel.Id(2)
		mid := _map.Id(100000000)
		result := p.GetFields(ten, &wid, &cid, &mid)
		require.Len(t, result, 1)
		assert.Equal(t, channel.Id(2), result[0].Field.ChannelId())
	})

	t.Run("no match", func(t *testing.T) {
		wid := world.Id(9)
		result := p.GetFields(ten, &wid, nil, nil)
		require.NotNil(t, result)
		assert.Len(t, result, 0)
	})
}

// TestProcessorGetFieldsSortOrder confirms the deterministic sort order:
// world, then channel, then map, then instance-id string.
func TestProcessorGetFieldsSortOrder(t *testing.T) {
	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	require.NoError(t, err)
	ctx := tenant.WithContext(context.Background(), ten)

	worldId := world.Id(0)
	channelId := channel.Id(1)

	cp := character.NewProcessor(logrus.New(), ctx)
	shuffledMapIds := []uint32{400000000, 100000000, 300000000, 200000000}
	for i, mapId := range shuffledMapIds {
		f := cfield.NewBuilder(worldId, channelId, _map.Id(mapId)).SetInstance(uuid.Nil).Build()
		cp.Enter(uuid.New(), f, uint32(i+1))
	}

	p := NewProcessor(logrus.New(), ctx)
	result := p.GetFields(ten, nil, nil, nil)
	require.Len(t, result, 4)

	expected := []_map.Id{100000000, 200000000, 300000000, 400000000}
	for i, want := range expected {
		assert.Equal(t, want, result[i].Field.MapId())
	}
}
