package buff

import (
	"atlas-buffs/buff/stat"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewNoExpiryBuff(t *testing.T) {
	b, err := NewNoExpiryBuff(int32(5211006), byte(1), setupTestChanges(), "")

	assert.NoError(t, err)
	assert.True(t, b.NoExpiry())
	assert.Equal(t, int32(0), b.Duration())
	assert.True(t, b.ExpiresAt().IsZero())
	assert.False(t, b.Expired(), "no-expiry buff must never report expired despite zero expiresAt")
	assert.Len(t, b.Changes(), 2)
}

func TestNewNoExpiryBuff_EmptyChanges(t *testing.T) {
	_, err := NewNoExpiryBuff(int32(5211006), byte(1), []stat.Model{}, "")
	assert.ErrorIs(t, err, ErrEmptyChanges)
}

func TestNewBuff_StillRejectsNonPositiveDuration(t *testing.T) {
	_, err := NewBuff(int32(2001001), byte(5), 0, setupTestChanges(), "")
	assert.ErrorIs(t, err, ErrInvalidDuration)
	_, err = NewBuff(int32(2001001), byte(5), -1, setupTestChanges(), "")
	assert.ErrorIs(t, err, ErrInvalidDuration)
}

func TestNoExpiryBuff_JSONRoundTrip(t *testing.T) {
	b, err := NewNoExpiryBuff(int32(5220011), byte(10), setupTestChanges(), "")
	assert.NoError(t, err)

	data, err := json.Marshal(b)
	assert.NoError(t, err)

	var out Model
	assert.NoError(t, json.Unmarshal(data, &out))
	assert.True(t, out.NoExpiry())
	assert.False(t, out.Expired())
}

// A finite buff marshalled before this change has no noExpiry field; it must
// unmarshal to noExpiry=false so previously Redis-persisted buffs are unaffected.
func TestFiniteBuff_JSONAbsentNoExpiryDefaultsFalse(t *testing.T) {
	b, err := NewBuff(int32(2001001), byte(5), 60000, setupTestChanges(), "")
	assert.NoError(t, err)

	data, err := json.Marshal(b)
	assert.NoError(t, err)
	assert.NotContains(t, string(data), "noExpiry", "omitempty must keep finite-buff JSON unchanged")

	var out Model
	assert.NoError(t, json.Unmarshal(data, &out))
	assert.False(t, out.NoExpiry())
}

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

	b, err := NewBuff(sourceId, byte(5), duration, changes, "")

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
	b, err := NewBuff(sourceId, byte(5), duration, changes, "")
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

	b, err := NewBuff(sourceId, byte(5), duration, changes, "")
	assert.NoError(t, err)

	assert.False(t, b.Expired())
}

func TestBuff_Expired_ZeroDuration(t *testing.T) {
	sourceId := int32(2001001)
	duration := int32(0) // 0 seconds - should be rejected
	changes := setupTestChanges()

	_, err := NewBuff(sourceId, byte(5), duration, changes, "")

	assert.ErrorIs(t, err, ErrInvalidDuration)
}

func TestBuff_Expired_NegativeDuration(t *testing.T) {
	sourceId := int32(2001001)
	duration := int32(-1) // Negative duration - should be rejected
	changes := setupTestChanges()

	_, err := NewBuff(sourceId, byte(5), duration, changes, "")

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

	b, err := NewBuff(sourceId, byte(5), duration, changes, "")
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

	b1, err1 := NewBuff(sourceId, byte(5), duration, changes, "")
	assert.NoError(t, err1)
	b2, err2 := NewBuff(sourceId, byte(5), duration, changes, "")
	assert.NoError(t, err2)

	// Each buff should have a unique ID
	assert.NotEqual(t, b1.id, b2.id)
}

func TestBuff_EmptyChanges(t *testing.T) {
	sourceId := int32(2001001)
	duration := int32(60)
	changes := []stat.Model{}

	_, err := NewBuff(sourceId, byte(5), duration, changes, "")

	assert.ErrorIs(t, err, ErrEmptyChanges)
}

func TestBuff_Accessors(t *testing.T) {
	sourceId := int32(2001001)
	duration := int32(60)
	changes := setupTestChanges()

	b, err := NewBuff(sourceId, byte(5), duration, changes, "")
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

	b, err := NewBuff(sourceId, byte(5), duration, changes, "")
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
	m, err := NewBuff(1111002, 20, 150000, changes, "")
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
	m, err := NewBuff(1111002, 20, 150000, []stat.Model{stat.NewStat("COMBO", 1)}, "")
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
	m, err := NewBuff(1111002, 20, 150000, []stat.Model{stat.NewStat("WATK", 20)}, "")
	if err != nil {
		t.Fatalf("NewBuff: %v", err)
	}
	if _, ok := m.WithStatAmount("COMBO", 3); ok {
		t.Fatal("expected ok=false for absent stat type")
	}
}

// TestModel_WithStatAmount_PreservesNoExpiry is the regression test for
// task-167 FR-2.4: WithStatAmount must preserve the noExpiry flag on
// no-expiry buffs (e.g. HOMING_BEACON locks) so they are never reaped by
// the expiration ticker.
func TestModel_WithStatAmount_PreservesNoExpiry(t *testing.T) {
	m, err := NewNoExpiryBuff(5211006, 1, []stat.Model{stat.NewStat("LOCK", 1)}, "")
	if err != nil {
		t.Fatalf("NewNoExpiryBuff: %v", err)
	}

	updated, ok := m.WithStatAmount("LOCK", 2)
	if !ok {
		t.Fatal("expected ok=true for present stat type")
	}

	// Stat amount must have changed.
	if updated.Changes()[0].Amount() != 2 {
		t.Fatalf("LOCK amount = %d, want 2", updated.Changes()[0].Amount())
	}

	// Critical: noExpiry flag must be preserved.
	if !updated.NoExpiry() {
		t.Fatal("noExpiry flag lost after WithStatAmount — buff will be reaped")
	}

	// Expired() must short-circuit and return false.
	if updated.Expired() {
		t.Fatal("no-expiry buff must never report expired")
	}
}

func TestCorrelationSurvivesTheRedisRoundTrip(t *testing.T) {
	b, err := NewBuff(9000, 1, 60000, []stat.Model{stat.NewStat("EXP_BUFF_RATE", 200)}, "occ-1")
	if err != nil {
		t.Fatalf("NewBuff: %v", err)
	}

	raw, err := json.Marshal(b)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var back Model
	if err := json.Unmarshal(raw, &back); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if back.CorrelationId() != "occ-1" {
		t.Fatalf("correlation lost: %q", back.CorrelationId())
	}
}

// An existing buff in Redis has no correlationId key. It must unmarshal to
// empty — no migration, no backfill.
func TestLegacyBuffPayloadUnmarshalsWithEmptyCorrelation(t *testing.T) {
	raw := `{"id":"11111111-1111-1111-1111-111111111111","sourceId":9000,"level":1,"duration":60000,"changes":[],"createdAt":"2026-08-15T00:00:00Z","expiresAt":"2026-08-15T00:01:00Z"}`
	var m Model
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if m.CorrelationId() != "" {
		t.Fatalf("correlation = %q, want empty", m.CorrelationId())
	}
}

// An ordinary buff (no correlation) must marshal byte-identically to today.
func TestUncorrelatedBuffMarshalsWithoutTheKey(t *testing.T) {
	b, _ := NewBuff(9000, 1, 60000, []stat.Model{stat.NewStat("EXP_BUFF_RATE", 200)}, "")
	raw, _ := json.Marshal(b)
	if strings.Contains(string(raw), "correlationId") {
		t.Fatalf("uncorrelated buff carries the key: %s", raw)
	}
}
