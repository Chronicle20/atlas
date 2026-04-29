package monster

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestStatusEffect_ReflectFields_DefaultZero(t *testing.T) {
	se := NewStatusEffect(SourceTypeMonsterSkill, 0, 100, 1,
		map[string]int32{"FREEZE": 1}, 5*time.Second, 0)
	require.Equal(t, "", se.ReflectKind())
	require.Equal(t, int32(0), se.ReflectPercent())
	require.Equal(t, int16(0), se.ReflectLtX())
	require.Equal(t, int16(0), se.ReflectLtY())
	require.Equal(t, int16(0), se.ReflectRbX())
	require.Equal(t, int16(0), se.ReflectRbY())
	require.Equal(t, int32(0), se.ReflectMaxDamage())
	require.False(t, se.IsReflect())
}

func TestNewReflectStatusEffect_PopulatesAllFields(t *testing.T) {
	se := NewReflectStatusEffect(
		SourceTypeMonsterSkill, 0, 143, 1,
		map[string]int32{"WEAPON_COUNTER": 30}, 60*time.Second,
		"PHYSICAL", 30, -50, -30, 50, 30, 32767,
	)
	require.Equal(t, "PHYSICAL", se.ReflectKind())
	require.Equal(t, int32(30), se.ReflectPercent())
	require.Equal(t, int16(-50), se.ReflectLtX())
	require.Equal(t, int16(-30), se.ReflectLtY())
	require.Equal(t, int16(50), se.ReflectRbX())
	require.Equal(t, int16(30), se.ReflectRbY())
	require.Equal(t, int32(32767), se.ReflectMaxDamage())
	require.True(t, se.IsReflect())
}

func TestStatusEffect_WithLastTick_PreservesReflectFields(t *testing.T) {
	se := NewReflectStatusEffect(
		SourceTypeMonsterSkill, 0, 143, 1,
		map[string]int32{"WEAPON_COUNTER": 30}, 60*time.Second,
		"PHYSICAL", 30, -10, -10, 10, 10, 32767,
	)
	updated := se.WithLastTick(time.Now())
	require.Equal(t, "PHYSICAL", updated.ReflectKind())
	require.Equal(t, int32(30), updated.ReflectPercent())
	require.Equal(t, int16(-10), updated.ReflectLtX())
}
