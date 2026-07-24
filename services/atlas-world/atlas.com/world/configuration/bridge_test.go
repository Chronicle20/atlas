package configuration

import (
	"atlas-world/configuration/tenant"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestChangedTenants(t *testing.T) {
	id1 := uuid.New()
	id2 := uuid.New()

	a := tenant.RestModel{Id: id1.String(), Region: "GMS", MajorVersion: 83, MinorVersion: 1}
	aChanged := a
	aChanged.MajorVersion = 84

	prev := map[uuid.UUID]tenant.RestModel{id1: a}
	next := map[uuid.UUID]tenant.RestModel{
		id1: aChanged,                          // changed
		id2: {Id: id2.String(), Region: "GMS"}, // new
	}

	// changed + new are returned.
	require.ElementsMatch(t, []uuid.UUID{id1, id2}, changedTenants(prev, next))

	// Identical maps → nothing changed.
	require.Empty(t, changedTenants(next, next))

	// Removed tenant (in prev, absent from next) is not returned and does
	// not panic — only id1 (which changed back) is reported.
	require.Equal(t, []uuid.UUID{id1}, changedTenants(next, prev))
}
