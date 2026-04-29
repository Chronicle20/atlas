package mist

import (
	"testing"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
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
