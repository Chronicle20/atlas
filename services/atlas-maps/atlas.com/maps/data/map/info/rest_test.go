package info

import (
	"testing"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/stretchr/testify/require"
)

func TestExtract_PopulatesAllFields(t *testing.T) {
	rm := RestModel{
		Id:                _map.Id(100000000),
		TimeLimit:         600,
		ForcedReturnMapId: _map.Id(100000201),
	}
	m, err := Extract(rm)
	require.NoError(t, err)
	require.Equal(t, _map.Id(100000000), m.Id())
	require.Equal(t, int32(600), m.TimeLimit())
	require.Equal(t, _map.Id(100000201), m.ForcedReturnMapId())
}

func TestRestModel_ImplementsJSONApiResource(t *testing.T) {
	rm := RestModel{Id: _map.Id(100000000)}
	require.Equal(t, "maps", rm.GetName())
	require.Equal(t, "100000000", rm.GetID())
	require.NoError(t, rm.SetID("200000000"))
	require.Equal(t, _map.Id(200000000), rm.Id)
}
