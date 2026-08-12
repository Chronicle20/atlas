package mist

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	mistKafka "atlas-maps/kafka/message/mist"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
)

func mkField(t *testing.T) field.Model {
	t.Helper()
	return field.NewBuilder(0, 0, 100000000).SetInstance(uuid.Nil).Build()
}

func TestMistBuilder_BuildsImmutable(t *testing.T) {
	id := uuid.New()
	f := mkField(t)
	m := NewBuilder(id, f).
		SetOwner("MONSTER", 9001).
		SetOrigin(100, 200).
		SetBounds(-50, -30, 50, 30).
		SetDisease("POISON", 80, 30*time.Second).
		SetDuration(10 * time.Second).
		SetTickInterval(time.Second).
		Build()

	require.Equal(t, id, m.Id())
	require.Equal(t, "MONSTER", m.OwnerType())
	require.Equal(t, uint32(9001), m.OwnerId())
	require.Equal(t, int16(100), m.OriginX())
	require.Equal(t, int16(200), m.OriginY())
	require.Equal(t, int16(-50), m.LtX())
	require.Equal(t, int16(50), m.RbX())
	require.Equal(t, "POISON", m.Disease())
	require.Equal(t, int32(80), m.DiseaseValue())
	require.Equal(t, 30*time.Second, m.DiseaseDuration())
	require.Equal(t, 10*time.Second, m.Duration())
	require.Equal(t, time.Second, m.TickInterval())
}

func TestMist_Contains_InsideAndOutside(t *testing.T) {
	id := uuid.New()
	m := NewBuilder(id, mkField(t)).
		SetOrigin(100, 200).
		SetBounds(-50, -30, 50, 30).
		SetDuration(time.Second).
		Build()

	require.True(t, m.Contains(100, 200), "origin")
	require.True(t, m.Contains(150, 230), "max corner inclusive")
	require.True(t, m.Contains(50, 170), "min corner inclusive")
	require.False(t, m.Contains(151, 200), "outside x")
	require.False(t, m.Contains(100, 231), "outside y")
}

func TestMist_Expired_AfterDuration(t *testing.T) {
	id := uuid.New()
	m := NewBuilder(id, mkField(t)).
		SetOrigin(0, 0).
		SetBounds(-1, -1, 1, 1).
		SetDuration(0).
		Build()
	require.True(t, m.Expired())
}

func TestMist_ShouldTick_RespectsLastTick(t *testing.T) {
	id := uuid.New()
	m := NewBuilder(id, mkField(t)).
		SetOrigin(0, 0).
		SetBounds(-1, -1, 1, 1).
		SetDuration(time.Minute).
		SetTickInterval(time.Second).
		Build()
	require.True(t, m.ShouldTick(), "fresh mist, lastTick = createdAt - tickInterval")

	updated := m.WithLastTick(time.Now())
	require.False(t, updated.ShouldTick())
}

// TestMist_Kinds_RoundTrip asserts the target/effect descriptors survive the
// builder (task-200 FR-2.5). mkField is the file's existing helper (line 13).
func TestMist_Kinds_RoundTrip(t *testing.T) {
	f := mkField(t)
	m := NewBuilder(uuid.New(), f).
		SetKinds(mistKafka.TargetKindMonster, mistKafka.EffectKindDamageOverTime).
		Build()
	if m.TargetKind() != "MONSTER" {
		t.Fatalf("m.TargetKind() = %q, want MONSTER", m.TargetKind())
	}
	if m.EffectKind() != "DAMAGE_OVER_TIME" {
		t.Fatalf("m.EffectKind() = %q, want DAMAGE_OVER_TIME", m.EffectKind())
	}
}

// TestMist_Rect_AgreesWithContains pins Rect() against Contains() on the
// boundary coordinates, so the two rectangle derivations cannot drift.
func TestMist_Rect_AgreesWithContains(t *testing.T) {
	f := mkField(t)
	m := NewBuilder(uuid.New(), f).
		SetOrigin(500, 300).
		SetBounds(-110, -82, 110, 83).
		Build()

	x1, y1, x2, y2 := m.Rect()
	if x1 != 390 || y1 != 218 || x2 != 610 || y2 != 383 {
		t.Fatalf("m.Rect() = (%d,%d,%d,%d), want (390,218,610,383)", x1, y1, x2, y2)
	}
	// Every rect corner is inside (Contains is inclusive of edges).
	for _, p := range [][2]int16{{x1, y1}, {x2, y2}, {x1, y2}, {x2, y1}} {
		if !m.Contains(p[0], p[1]) {
			t.Fatalf("m.Contains(%d,%d) = false, want true", p[0], p[1])
		}
	}
	// One unit outside each edge is outside.
	if m.Contains(x1-1, y1) || m.Contains(x2+1, y2) || m.Contains(x1, y1-1) || m.Contains(x2, y2+1) {
		t.Fatal("Contains returned true outside the Rect bounds")
	}
}

// PartyMemberIds must hand back a copy: the tick fans out across goroutines
// (processTenant), and a shared backing array is exactly the mutable state
// that parallelism punishes.
func TestMist_PartyMemberIds_IsDefensiveCopy(t *testing.T) {
	ids := []uint32{1, 2, 3}
	m := NewBuilder(uuid.New(), mkField(t)).
		SetRecovery(38, ids).
		Build()

	got := m.PartyMemberIds()
	require.Equal(t, []uint32{1, 2, 3}, got)

	got[0] = 99
	ids[1] = 88
	require.Equal(t, []uint32{1, 2, 3}, m.PartyMemberIds())
}

func TestMist_RecoveryMp_RoundTripsThroughBuilder(t *testing.T) {
	m := NewBuilder(uuid.New(), mkField(t)).
		SetRecovery(80, nil).
		Build()

	require.Equal(t, int32(80), m.RecoveryMp())
	require.Empty(t, m.PartyMemberIds())
}
