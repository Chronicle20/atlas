package searchcount

import (
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-database/databasetest"
)

func newTestProcessor(t *testing.T) (Processor, Processor) {
	t.Helper()
	db := databasetest.NewInMemoryTenantDB(t, Migration)
	l := logrus.New()
	tidA, tidB := uuid.New(), uuid.New()
	pA := NewProcessor(l, databasetest.TenantContext(tidA), db)
	pB := NewProcessor(l, databasetest.TenantContext(tidB), db)
	return pA, pB
}

func TestRecordSearch_IncrementsAndIsolatesTenants(t *testing.T) {
	pA, pB := newTestProcessor(t)

	require.NoError(t, pA.RecordSearch(0, 2060000))
	require.NoError(t, pA.RecordSearch(0, 2060000))
	require.NoError(t, pA.RecordSearch(0, 1302000))
	require.NoError(t, pB.RecordSearch(0, 2060000))

	top, err := pA.GetTop(0, 10)
	require.NoError(t, err)
	require.Len(t, top, 2)
	require.Equal(t, uint32(2060000), top[0].ItemId())
	require.Equal(t, uint64(2), top[0].Count())
	require.Equal(t, uint32(1302000), top[1].ItemId())
	require.Equal(t, uint64(1), top[1].Count())

	topB, err := pB.GetTop(0, 10)
	require.NoError(t, err)
	require.Len(t, topB, 1)
	require.Equal(t, uint64(1), topB[0].Count())
}

func TestRecordSearch_WorldScoped(t *testing.T) {
	pA, _ := newTestProcessor(t)
	require.NoError(t, pA.RecordSearch(0, 2060000))
	require.NoError(t, pA.RecordSearch(1, 2060000))
	require.NoError(t, pA.RecordSearch(1, 2060000))

	top0, err := pA.GetTop(0, 10)
	require.NoError(t, err)
	require.Len(t, top0, 1)
	require.Equal(t, uint64(1), top0[0].Count())

	top1, err := pA.GetTop(1, 10)
	require.NoError(t, err)
	require.Len(t, top1, 1)
	require.Equal(t, uint64(2), top1[0].Count())
}

func TestGetTop_LimitsToTen(t *testing.T) {
	pA, _ := newTestProcessor(t)
	for i := uint32(0); i < 15; i++ {
		itemId := 2060000 + i
		for j := uint32(0); j <= i; j++ {
			require.NoError(t, pA.RecordSearch(0, itemId))
		}
	}
	top, err := pA.GetTop(0, 10)
	require.NoError(t, err)
	require.Len(t, top, 10)
	// highest count first
	require.Equal(t, uint64(15), top[0].Count())
	require.Equal(t, uint32(2060014), top[0].ItemId())
}

// TestRecordSearch_ConcurrentIncrements: parallel increments must sum
// correctly (atomic upsert, no lost updates). sqlite serializes writers;
// if the driver returns SQLITE_BUSY under -race, bound the concurrency
// but keep total increments at 20.
func TestRecordSearch_ConcurrentIncrements(t *testing.T) {
	pA, _ := newTestProcessor(t)
	var wg sync.WaitGroup
	errs := make(chan error, 20)
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			errs <- pA.RecordSearch(world.Id(0), 2060000)
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		require.NoError(t, err)
	}
	top, err := pA.GetTop(0, 10)
	require.NoError(t, err)
	require.Len(t, top, 1)
	require.Equal(t, uint64(20), top[0].Count())
}
