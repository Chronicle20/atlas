package backeffect

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func TestSetThenGetActive(t *testing.T) {
	l, _ := test.NewNullLogger()
	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	require.NoError(t, err)
	f := field.NewBuilder(0, 1, 100000000).Build()
	ctx := tenant.WithContext(context.Background(), ten)

	p := NewProcessor(l, ctx)
	entry := BackEffectEntry{Effect: 0, FieldId: 100000000, PageId: 1, Duration: 1000}
	p.Set(f, entry)

	active := p.GetActive(f)
	require.Len(t, active, 1)
	require.Equal(t, entry, active[0])
}

func TestSetReplacesSamePageInPlace(t *testing.T) {
	l, _ := test.NewNullLogger()
	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	require.NoError(t, err)
	f := field.NewBuilder(0, 1, 100000000).Build()
	ctx := tenant.WithContext(context.Background(), ten)

	p := NewProcessor(l, ctx)
	p.Set(f, BackEffectEntry{Effect: 0, FieldId: 100000000, PageId: 1, Duration: 1000})
	p.Set(f, BackEffectEntry{Effect: 0, FieldId: 100000000, PageId: 2, Duration: 1000})
	p.Set(f, BackEffectEntry{Effect: 1, FieldId: 100000000, PageId: 1, Duration: 250})

	active := p.GetActive(f)
	require.Len(t, active, 2)
	require.Equal(t, byte(1), active[0].PageId)
	require.Equal(t, byte(1), active[0].Effect)
	require.Equal(t, uint32(250), active[0].Duration)
	require.Equal(t, byte(2), active[1].PageId)
}

func TestClearRemovesEveryPage(t *testing.T) {
	l, _ := test.NewNullLogger()
	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	require.NoError(t, err)
	f := field.NewBuilder(0, 1, 100000000).Build()
	ctx := tenant.WithContext(context.Background(), ten)

	p := NewProcessor(l, ctx)
	p.Set(f, BackEffectEntry{Effect: 0, FieldId: 100000000, PageId: 1, Duration: 1000})
	p.Set(f, BackEffectEntry{Effect: 0, FieldId: 100000000, PageId: 2, Duration: 1000})
	p.Set(f, BackEffectEntry{Effect: 0, FieldId: 100000000, PageId: 3, Duration: 1000})

	require.True(t, p.Clear(f))
	require.Len(t, p.GetActive(f), 0)
	require.False(t, p.Clear(f))
}

func TestBackEffectIsTenantIsolated(t *testing.T) {
	l, _ := test.NewNullLogger()
	tenA, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	require.NoError(t, err)
	tenB, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	require.NoError(t, err)
	f := field.NewBuilder(0, 1, 100000000).Build()

	ctxA := tenant.WithContext(context.Background(), tenA)
	ctxB := tenant.WithContext(context.Background(), tenB)

	pA := NewProcessor(l, ctxA)
	pB := NewProcessor(l, ctxB)

	pA.Set(f, BackEffectEntry{Effect: 0, FieldId: 100000000, PageId: 1, Duration: 1000})

	require.Len(t, pA.GetActive(f), 1)
	require.Len(t, pB.GetActive(f), 0)
}
