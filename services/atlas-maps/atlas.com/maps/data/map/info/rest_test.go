package info

import (
	"testing"

	"github.com/stretchr/testify/require"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
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

// TestTransformRoundTrip confirms Transform is the faithful inverse of
// Extract: every field set by Extract survives a Transform -> Extract round
// trip. The codemod's SKIP reason was a fixture-generation defect (Build()
// called as if it returned two values); Builder.Build() here returns a
// single Model.
func TestTransformRoundTrip(t *testing.T) {
	m := NewBuilder().
		SetId(_map.Id(100000000)).
		SetTimeLimit(600).
		SetForcedReturnMapId(_map.Id(100000001)).
		Build()

	rm, err := Transform(m)
	require.NoError(t, err)
	require.Equal(t, m.Id(), rm.Id)

	m2, err := Extract(rm)
	require.NoError(t, err)
	require.Equal(t, m, m2)
}
