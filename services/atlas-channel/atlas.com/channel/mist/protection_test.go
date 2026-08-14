package mist_test

import (
	"atlas-channel/mist"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func testTenant(t *testing.T) tenant.Model {
	t.Helper()
	tn, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	require.NoError(t, err)
	return tn
}

// protectionAt builds a live protection covering (0,0)..(200,200).
func protectionAt(f field.Model, ownerId uint32, ttl time.Duration) mist.Protection {
	return mist.NewProtectionBuilder(uuid.New(), f).
		SetOwnerId(ownerId).
		SetRect(0, 0, 200, 200).
		SetExpiresAt(time.Now().Add(ttl)).
		Build()
}

func TestCovering_ReturnsProtectionContainingThePoint(t *testing.T) {
	r := mist.NewTestProtectionRegistry()
	tn := testTenant(t)
	f := field.NewBuilder(0, 0, 100000000).Build()
	p := protectionAt(f, 1001, time.Minute)
	r.Add(tn, p)

	require.Len(t, r.Covering(tn, f, 100, 100, time.Now()), 1)
	// Inclusive edges, matching atlas-maps' Mist.Contains.
	require.Len(t, r.Covering(tn, f, 0, 0, time.Now()), 1)
	require.Len(t, r.Covering(tn, f, 200, 200, time.Now()), 1)
	require.Empty(t, r.Covering(tn, f, 201, 100, time.Now()))
	require.Empty(t, r.Covering(tn, f, 100, -1, time.Now()))
}

func TestCovering_IgnoresOtherFieldsAndTenants(t *testing.T) {
	r := mist.NewTestProtectionRegistry()
	tn := testTenant(t)
	other := testTenant(t)
	f := field.NewBuilder(0, 0, 100000000).Build()
	elsewhere := field.NewBuilder(0, 0, 100000001).Build()
	r.Add(tn, protectionAt(f, 1001, time.Minute))

	require.Empty(t, r.Covering(tn, elsewhere, 100, 100, time.Now()))
	require.Empty(t, r.Covering(other, f, 100, 100, time.Now()))
}

// A dropped MIST_DESTROYED must not leave a permanently protective
// rectangle: expiry is evaluated on read.
func TestCovering_TreatsExpiredAsAbsent(t *testing.T) {
	r := mist.NewTestProtectionRegistry()
	tn := testTenant(t)
	f := field.NewBuilder(0, 0, 100000000).Build()
	r.Add(tn, protectionAt(f, 1001, -time.Second))

	require.Empty(t, r.Covering(tn, f, 100, 100, time.Now()))
}

func TestRemove_DropsTheProtection(t *testing.T) {
	r := mist.NewTestProtectionRegistry()
	tn := testTenant(t)
	f := field.NewBuilder(0, 0, 100000000).Build()
	p := protectionAt(f, 1001, time.Minute)
	r.Add(tn, p)
	require.Len(t, r.Covering(tn, f, 100, 100, time.Now()), 1)

	r.Remove(tn, p.Id())

	require.Empty(t, r.Covering(tn, f, 100, 100, time.Now()))
}

// Add prunes expired entries lazily, so a channel that never sees a
// MIST_DESTROYED does not accumulate them for the process's lifetime.
func TestAdd_PrunesExpiredEntries(t *testing.T) {
	r := mist.NewTestProtectionRegistry()
	tn := testTenant(t)
	f := field.NewBuilder(0, 0, 100000000).Build()
	r.Add(tn, protectionAt(f, 1001, -time.Second))
	r.Add(tn, protectionAt(f, 1002, time.Minute))

	require.Equal(t, 1, r.Len(tn))
}

func TestGetProtectionRegistry_IsASingleton(t *testing.T) {
	require.Same(t, mist.GetProtectionRegistry(), mist.GetProtectionRegistry())
}

// TestConcurrentAccess exercises Add, Remove, Covering, and Len from many
// goroutines simultaneously against the same registry. The registry is read
// on the damage hot path and written from a Kafka consumer concurrently, so
// this must be race-clean under `go test -race`.
func TestConcurrentAccess(t *testing.T) {
	r := mist.NewTestProtectionRegistry()
	tn := testTenant(t)
	f := field.NewBuilder(0, 0, 100000000).Build()

	const workers = 20
	const iterations = 50

	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(worker int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				p := protectionAt(f, uint32(1000+worker), time.Minute)
				r.Add(tn, p)
				_ = r.Covering(tn, f, 100, 100, time.Now())
				_ = r.Len(tn)
				r.Remove(tn, p.Id())
			}
		}(i)
	}
	wg.Wait()
}
