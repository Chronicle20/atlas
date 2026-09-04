package character

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func TestGetFieldsWithCharacters(t *testing.T) {
	t.Run("empty registry", func(t *testing.T) {
		ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
		require.NoError(t, err)

		result := getRegistry().GetFieldsWithCharacters(ten)

		assert.NotNil(t, result)
		assert.Equal(t, 0, len(result))
	})

	t.Run("single occupied field", func(t *testing.T) {
		ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
		require.NoError(t, err)

		f := field.NewBuilder(0, 1, 910340000).SetInstance(uuid.Nil).Build()
		getRegistry().AddCharacter(MapKey{Tenant: ten, Field: f}, 100)

		result := getRegistry().GetFieldsWithCharacters(ten)

		require.Equal(t, 1, len(result))
		assert.Equal(t, world.Id(0), result[0].Field.WorldId())
		assert.Equal(t, channel.Id(1), result[0].Field.ChannelId())
		assert.Equal(t, _map.Id(910340000), result[0].Field.MapId())
		assert.Equal(t, uuid.Nil, result[0].Field.Instance())
		assert.Equal(t, uint32(1), result[0].CharacterCount)
	})

	t.Run("drained field excluded", func(t *testing.T) {
		ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
		require.NoError(t, err)

		key := MapKey{Tenant: ten, Field: field.NewBuilder(0, 1, 100000000).SetInstance(uuid.Nil).Build()}
		getRegistry().AddCharacter(key, 100)
		getRegistry().RemoveCharacter(key, 100)

		result := getRegistry().GetFieldsWithCharacters(ten)

		assert.Equal(t, 0, len(result))
	})

	t.Run("drained plus live", func(t *testing.T) {
		ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
		require.NoError(t, err)

		drainedKey := MapKey{Tenant: ten, Field: field.NewBuilder(0, 1, 100000000).SetInstance(uuid.Nil).Build()}
		getRegistry().AddCharacter(drainedKey, 100)
		getRegistry().RemoveCharacter(drainedKey, 100)

		liveKey := MapKey{Tenant: ten, Field: field.NewBuilder(0, 2, 200000000).SetInstance(uuid.Nil).Build()}
		getRegistry().AddCharacter(liveKey, 200)
		getRegistry().AddCharacter(liveKey, 201)

		result := getRegistry().GetFieldsWithCharacters(ten)

		require.Equal(t, 1, len(result))
		assert.True(t, result[0].Field.Equals(liveKey.Field))
		assert.Equal(t, uint32(2), result[0].CharacterCount)
	})

	t.Run("tenant isolation", func(t *testing.T) {
		tenA, err := tenant.Create(uuid.New(), "GMS", 83, 1)
		require.NoError(t, err)
		tenB, err := tenant.Create(uuid.New(), "GMS", 83, 1)
		require.NoError(t, err)

		keyA := MapKey{Tenant: tenA, Field: field.NewBuilder(0, 1, 300000000).SetInstance(uuid.Nil).Build()}
		getRegistry().AddCharacter(keyA, 1)

		keyB := MapKey{Tenant: tenB, Field: field.NewBuilder(0, 1, 400000000).SetInstance(uuid.Nil).Build()}
		getRegistry().AddCharacter(keyB, 2)

		resultA := getRegistry().GetFieldsWithCharacters(tenA)
		require.Equal(t, 1, len(resultA))
		assert.Equal(t, _map.Id(300000000), resultA[0].Field.MapId())

		resultB := getRegistry().GetFieldsWithCharacters(tenB)
		require.Equal(t, 1, len(resultB))
		assert.Equal(t, _map.Id(400000000), resultB[0].Field.MapId())
	})

	t.Run("two instances same map", func(t *testing.T) {
		ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
		require.NoError(t, err)

		instance2 := uuid.New()

		key1 := MapKey{Tenant: ten, Field: field.NewBuilder(0, 1, 910340000).SetInstance(uuid.Nil).Build()}
		getRegistry().AddCharacter(key1, 1)

		key2 := MapKey{Tenant: ten, Field: field.NewBuilder(0, 1, 910340000).SetInstance(instance2).Build()}
		getRegistry().AddCharacter(key2, 2)

		result := getRegistry().GetFieldsWithCharacters(ten)

		require.Equal(t, 2, len(result))
		for _, occupancy := range result {
			assert.Equal(t, _map.Id(910340000), occupancy.Field.MapId())
			assert.Equal(t, uint32(1), occupancy.CharacterCount)
		}
		assert.NotEqual(t, result[0].Field.Instance(), result[1].Field.Instance())
	})
}
