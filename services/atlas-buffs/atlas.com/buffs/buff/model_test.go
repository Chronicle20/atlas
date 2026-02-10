package buff

import (
	"atlas-buffs/buff/stat"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func setupTestChanges() []stat.Model {
	return []stat.Model{
		stat.NewStat("STR", 10),
		stat.NewStat("DEX", 5),
	}
}

func TestNewBuff(t *testing.T) {
	sourceId := int32(2001001)
	duration := int32(60)
	changes := setupTestChanges()

	b, err := NewBuff(sourceId, byte(5), duration, changes)

	assert.NoError(t, err)
	assert.Equal(t, sourceId, b.SourceId())
	assert.Equal(t, duration, b.Duration())
	assert.Len(t, b.Changes(), 2)
	assert.NotEmpty(t, b.id)
}

func TestBuff_Timestamps(t *testing.T) {
	sourceId := int32(2001001)
	duration := int32(60)
	changes := setupTestChanges()

	before := time.Now().Add(-time.Millisecond) // Small buffer for timing
	b, err := NewBuff(sourceId, byte(5), duration, changes)
	assert.NoError(t, err)
	after := time.Now().Add(time.Millisecond) // Small buffer for timing

	// CreatedAt should be within the test window
	assert.True(t, !b.CreatedAt().Before(before), "CreatedAt should be after or equal to before")
	assert.True(t, !b.CreatedAt().After(after), "CreatedAt should be before or equal to after")

	// ExpiresAt should be approximately duration seconds after CreatedAt
	expectedExpiry := b.CreatedAt().Add(time.Duration(duration) * time.Second)
	diff := b.ExpiresAt().Sub(expectedExpiry)
	assert.True(t, diff >= -time.Millisecond && diff <= time.Millisecond,
		"ExpiresAt should be within 1ms of expected expiry")
}

func TestBuff_Expired_NotExpired(t *testing.T) {
	sourceId := int32(2001001)
	duration := int32(60) // 60 seconds - should not be expired
	changes := setupTestChanges()

	b, err := NewBuff(sourceId, byte(5), duration, changes)
	assert.NoError(t, err)

	assert.False(t, b.Expired())
}

func TestBuff_Expired_ZeroDuration(t *testing.T) {
	sourceId := int32(2001001)
	duration := int32(0) // 0 seconds - should be rejected
	changes := setupTestChanges()

	_, err := NewBuff(sourceId, byte(5), duration, changes)

	assert.ErrorIs(t, err, ErrInvalidDuration)
}

func TestBuff_Expired_NegativeDuration(t *testing.T) {
	sourceId := int32(2001001)
	duration := int32(-1) // Negative duration - should be rejected
	changes := setupTestChanges()

	_, err := NewBuff(sourceId, byte(5), duration, changes)

	assert.ErrorIs(t, err, ErrInvalidDuration)
}

func TestBuff_Changes(t *testing.T) {
	sourceId := int32(2001001)
	duration := int32(60)
	changes := []stat.Model{
		stat.NewStat("STR", 10),
		stat.NewStat("DEX", 5),
		stat.NewStat("INT", 15),
	}

	b, err := NewBuff(sourceId, byte(5), duration, changes)
	assert.NoError(t, err)

	resultChanges := b.Changes()
	assert.Len(t, resultChanges, 3)

	// Verify changes are preserved
	assert.Equal(t, "STR", resultChanges[0].Type())
	assert.Equal(t, int32(10), resultChanges[0].Amount())
	assert.Equal(t, "DEX", resultChanges[1].Type())
	assert.Equal(t, int32(5), resultChanges[1].Amount())
	assert.Equal(t, "INT", resultChanges[2].Type())
	assert.Equal(t, int32(15), resultChanges[2].Amount())
}

func TestBuff_UniqueIds(t *testing.T) {
	sourceId := int32(2001001)
	duration := int32(60)
	changes := setupTestChanges()

	b1, err1 := NewBuff(sourceId, byte(5), duration, changes)
	assert.NoError(t, err1)
	b2, err2 := NewBuff(sourceId, byte(5), duration, changes)
	assert.NoError(t, err2)

	// Each buff should have a unique ID
	assert.NotEqual(t, b1.id, b2.id)
}

func TestBuff_EmptyChanges(t *testing.T) {
	sourceId := int32(2001001)
	duration := int32(60)
	changes := []stat.Model{}

	_, err := NewBuff(sourceId, byte(5), duration, changes)

	assert.ErrorIs(t, err, ErrEmptyChanges)
}

func TestBuff_Accessors(t *testing.T) {
	sourceId := int32(2001001)
	duration := int32(60)
	changes := setupTestChanges()

	b, err := NewBuff(sourceId, byte(5), duration, changes)
	assert.NoError(t, err)

	// Test all accessors return expected values
	assert.Equal(t, sourceId, b.SourceId())
	assert.Equal(t, duration, b.Duration())
	assert.NotNil(t, b.Changes())
	assert.NotZero(t, b.CreatedAt())
	assert.NotZero(t, b.ExpiresAt())
}
