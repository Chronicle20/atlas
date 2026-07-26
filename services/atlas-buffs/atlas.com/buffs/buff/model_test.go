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

	// ExpiresAt should be approximately duration milliseconds after CreatedAt
	expectedExpiry := b.CreatedAt().Add(time.Duration(duration) * time.Millisecond)
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

// TestBuff_DurationInMilliseconds pins the unit contract for atlas-buffs:
// Duration is interpreted as time.Millisecond (NOT time.Second). Aligned
// with atlas-data's reader emitting ms after task-054.
func TestBuff_DurationInMilliseconds(t *testing.T) {
	sourceId := int32(2001001)
	duration := int32(60000) // 60 seconds expressed in ms
	changes := setupTestChanges()

	b, err := NewBuff(sourceId, byte(5), duration, changes)
	assert.NoError(t, err)

	gap := b.ExpiresAt().Sub(b.CreatedAt())
	expected := 60 * time.Second
	tolerance := 50 * time.Millisecond
	diff := gap - expected
	if diff < 0 {
		diff = -diff
	}
	assert.True(t, diff <= tolerance,
		"expected ExpiresAt-CreatedAt within %v of %v, got %v (diff %v)", tolerance, expected, gap, diff)
}

func TestModel_WithStatAmount_ReplacesTargetStat(t *testing.T) {
	changes := []stat.Model{stat.NewStat("COMBO", 1), stat.NewStat("WATK", 20)}
	m, err := NewBuff(1111002, 20, 150000, changes)
	if err != nil {
		t.Fatalf("NewBuff: %v", err)
	}

	updated, ok := m.WithStatAmount("COMBO", 3)
	if !ok {
		t.Fatal("expected ok=true for present stat type")
	}

	var combo, watk int32
	for _, c := range updated.Changes() {
		switch c.Type() {
		case "COMBO":
			combo = c.Amount()
		case "WATK":
			watk = c.Amount()
		}
	}
	if combo != 3 {
		t.Fatalf("COMBO amount = %d, want 3", combo)
	}
	if watk != 20 {
		t.Fatalf("WATK amount = %d, want 20 (other stats preserved)", watk)
	}
}

func TestModel_WithStatAmount_PreservesIdentityAndExpiry(t *testing.T) {
	m, err := NewBuff(1111002, 20, 150000, []stat.Model{stat.NewStat("COMBO", 1)})
	if err != nil {
		t.Fatalf("NewBuff: %v", err)
	}

	updated, ok := m.WithStatAmount("COMBO", 2)
	if !ok {
		t.Fatal("expected ok=true")
	}
	if updated.SourceId() != m.SourceId() || updated.Level() != m.Level() || updated.Duration() != m.Duration() {
		t.Fatal("identity fields must be preserved")
	}
	if !updated.CreatedAt().Equal(m.CreatedAt()) || !updated.ExpiresAt().Equal(m.ExpiresAt()) {
		t.Fatal("createdAt/expiresAt must be preserved (remaining-duration contract)")
	}
	// original untouched (immutability)
	if m.Changes()[0].Amount() != 1 {
		t.Fatalf("original buff mutated: COMBO = %d, want 1", m.Changes()[0].Amount())
	}
}

func TestModel_WithStatAmount_MissingStatType(t *testing.T) {
	m, err := NewBuff(1111002, 20, 150000, []stat.Model{stat.NewStat("WATK", 20)})
	if err != nil {
		t.Fatalf("NewBuff: %v", err)
	}
	if _, ok := m.WithStatAmount("COMBO", 3); ok {
		t.Fatal("expected ok=false for absent stat type")
	}
}
