package job

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetSkillsForJob_KnownJob(t *testing.T) {
	got, ok := GetSkillsForJob(112) // Hero
	require.True(t, ok)
	require.Equal(t, uint32(112), got.Id)
	require.NotEmpty(t, got.Skills)
	for _, s := range got.Skills {
		require.True(t, s >= 1100000 && s < 1200000, "Hero skills should be in 11xxxxx range, got %d", s)
	}
}

func TestGetSkillsForJob_UnknownJob(t *testing.T) {
	got, ok := GetSkillsForJob(99999)
	require.False(t, ok)
	require.Equal(t, uint32(99999), got.Id)
	require.Empty(t, got.Skills)
}
