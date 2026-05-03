package timer

import (
	"testing"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func mkTenant(t *testing.T) tenant.Model {
	t.Helper()
	tt, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	require.NoError(t, err)
	return tt
}

func TestEntry_GettersExposeAllFields(t *testing.T) {
	tt := mkTenant(t)
	f := field.NewBuilder(0, 0, 100000000).SetInstance(uuid.Nil).Build()
	tok := uuid.New()
	expires := time.Now().Add(10 * time.Minute)

	e := NewEntryBuilder().
		SetTenant(tt).
		SetCharacterId(42).
		SetField(f).
		SetForcedReturnMapId(_map.Id(100000201)).
		SetSeconds(600).
		SetToken(tok).
		SetExpiresAt(expires).
		Build()

	require.Equal(t, tt, e.Tenant())
	require.Equal(t, uint32(42), e.CharacterId())
	require.True(t, e.Field().Equals(f))
	require.Equal(t, _map.Id(100000201), e.ForcedReturnMapId())
	require.Equal(t, uint32(600), e.Seconds())
	require.Equal(t, tok, e.Token())
	require.Equal(t, expires, e.ExpiresAt())
}
