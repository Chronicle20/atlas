package jukebox

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func TestJukeboxStartThenGetActive(t *testing.T) {
	l, _ := test.NewNullLogger()
	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	require.NoError(t, err)
	f := field.NewBuilder(0, 1, 100000000).Build()
	ctx := tenant.WithContext(context.Background(), ten)

	p := NewProcessor(l, ctx)
	p.Start(f, 5100000, "Chronicle", time.Minute)

	entry, ok := p.GetActive(f)
	require.True(t, ok)
	require.Equal(t, uint32(5100000), entry.ItemId)
	require.Equal(t, "Chronicle", entry.PlayerName)
	require.True(t, entry.ExpiresAt.After(time.Now()))
}

func TestJukeboxStartReplacesActiveEntry(t *testing.T) {
	l, _ := test.NewNullLogger()
	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	require.NoError(t, err)
	f := field.NewBuilder(0, 1, 100000000).Build()
	ctx := tenant.WithContext(context.Background(), ten)

	p := NewProcessor(l, ctx)
	p.Start(f, 5100000, "Chronicle", time.Hour)
	p.Start(f, 5100001, "Other", time.Minute)

	entry, ok := p.GetActive(f)
	require.True(t, ok)
	require.Equal(t, uint32(5100001), entry.ItemId)
	require.Equal(t, "Other", entry.PlayerName)
	require.True(t, entry.ExpiresAt.Before(time.Now().Add(2*time.Minute)))
}

func TestJukeboxGetExpiredReturnsOnlyExpired(t *testing.T) {
	l, _ := test.NewNullLogger()
	tenA, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	require.NoError(t, err)
	tenB, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	require.NoError(t, err)
	f := field.NewBuilder(0, 1, 100000000).Build()

	ctxA := tenant.WithContext(context.Background(), tenA)
	ctxB := tenant.WithContext(context.Background(), tenB)

	pA := NewProcessor(l, ctxA)
	pA.Start(f, 5100000, "Chronicle", -time.Second)

	pB := NewProcessor(l, ctxB)
	pB.Start(f, 5100001, "Other", time.Hour)

	keyA := FieldKey{Tenant: tenA, Field: f}
	keyB := FieldKey{Tenant: tenB, Field: f}

	expired := GetExpired()

	var foundA, foundB bool
	for _, e := range expired {
		if e.Key == keyA {
			foundA = true
			require.Equal(t, uint32(5100000), e.Entry.ItemId)
		}
		if e.Key == keyB {
			foundB = true
		}
	}
	require.True(t, foundA)
	require.False(t, foundB)
}

func TestJukeboxIsTenantIsolated(t *testing.T) {
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

	_, ok := pB.GetActive(f)
	require.False(t, ok)

	pA.Start(f, 5100000, "Chronicle", time.Minute)

	_, ok = pB.GetActive(f)
	require.False(t, ok)
}
