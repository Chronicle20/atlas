package info

import (
	"testing"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/stretchr/testify/require"
)

func TestModel_Getters(t *testing.T) {
	m := Model{
		id:                _map.Id(100000000),
		timeLimit:         600,
		forcedReturnMapId: _map.Id(100000201),
	}
	require.Equal(t, _map.Id(100000000), m.Id())
	require.Equal(t, int32(600), m.TimeLimit())
	require.Equal(t, _map.Id(100000201), m.ForcedReturnMapId())
}

func TestModel_IsTimeLimited_BothFieldsPresent(t *testing.T) {
	m := Model{timeLimit: 600, forcedReturnMapId: _map.Id(100000201)}
	require.True(t, m.IsTimeLimited())
}

func TestModel_IsTimeLimited_TimeLimitZero(t *testing.T) {
	m := Model{timeLimit: 0, forcedReturnMapId: _map.Id(100000201)}
	require.False(t, m.IsTimeLimited(), "timeLimit=0 must disable")
}

func TestModel_IsTimeLimited_TimeLimitNegative(t *testing.T) {
	m := Model{timeLimit: -1, forcedReturnMapId: _map.Id(100000201)}
	require.False(t, m.IsTimeLimited(), "negative timeLimit treated as disabled")
}

func TestModel_IsTimeLimited_ForcedReturnSentinel(t *testing.T) {
	m := Model{timeLimit: 600, forcedReturnMapId: _map.Id(999999999)}
	require.False(t, m.IsTimeLimited(), "999999999 sentinel must disable")
}

func TestModel_IsTimeLimited_BothMissing(t *testing.T) {
	m := Model{timeLimit: 0, forcedReturnMapId: _map.Id(999999999)}
	require.False(t, m.IsTimeLimited())
}
