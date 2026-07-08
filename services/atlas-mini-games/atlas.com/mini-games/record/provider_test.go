package record

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetOrZero_Absent(t *testing.T) {
	db := setupTestDB(t)
	tenantId := uuid.New()

	m, err := GetOrZero(db, tenantId, 999, GameTypeMatchCards)
	require.NoError(t, err)
	assert.Equal(t, uint32(999), m.CharacterId())
	assert.Equal(t, GameTypeMatchCards, m.GameType())
	assert.Equal(t, uuid.Nil, m.Id(), "absent row must not synthesize a persisted id")
	assert.Equal(t, uint32(0), m.Wins())
	assert.Equal(t, uint32(0), m.Ties())
	assert.Equal(t, uint32(0), m.Losses())
}

func TestGetOrZero_Present(t *testing.T) {
	db := setupTestDB(t)
	tenantId := uuid.New()
	characterId := uint32(42)

	require.NoError(t, ApplyResult(db, tenantId, GameTypeOmok, characterId, 43, 0, false))

	m, err := GetOrZero(db, tenantId, characterId, GameTypeOmok)
	require.NoError(t, err)
	assert.Equal(t, uint32(1), m.Wins())
	assert.NotEqual(t, uuid.Nil, m.Id())
}

func TestGetByCharacter_ZeroFilledForBothGameTypes(t *testing.T) {
	db := setupTestDB(t)
	tenantId := uuid.New()
	characterId := uint32(7)

	ms, err := GetByCharacter(db, tenantId, characterId)
	require.NoError(t, err)
	require.Len(t, ms, 2, "must always return one row per game type, even with no persisted rows")

	byType := make(map[GameType]Model, len(ms))
	for _, m := range ms {
		byType[m.GameType()] = m
	}
	for _, gt := range []GameType{GameTypeOmok, GameTypeMatchCards} {
		m, ok := byType[gt]
		require.True(t, ok, "missing zero-filled row for %s", gt)
		assert.Equal(t, uint32(0), m.Wins())
		assert.Equal(t, uint32(0), m.Ties())
		assert.Equal(t, uint32(0), m.Losses())
	}

	// Now record a win in one game type only and confirm the other stays
	// zero-filled rather than disappearing from the result.
	require.NoError(t, ApplyResult(db, tenantId, GameTypeOmok, characterId, 8, 0, false))
	ms, err = GetByCharacter(db, tenantId, characterId)
	require.NoError(t, err)
	require.Len(t, ms, 2)
	for _, m := range ms {
		if m.GameType() == GameTypeOmok {
			assert.Equal(t, uint32(1), m.Wins())
		} else {
			assert.Equal(t, uint32(0), m.Wins())
		}
	}
}
