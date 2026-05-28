package compartment_test

import (
	"atlas-inventory/compartment"
	"testing"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Lock registry tests
// ---------------------------------------------------------------------------

// TestLockUnlock verifies a basic Lock→Unlock cycle: after Unlock, a second
// mutex for the same key can re-acquire immediately (it was released).
func TestLockUnlock(t *testing.T) {
	reg := compartment.LockRegistry()
	require.NotNil(t, reg)

	m := reg.Get(9001, inventory.TypeValueEquip)
	m.Lock()
	m.Unlock()

	// A second mutex for the same (characterId, inventoryType) must be able to
	// acquire after the first holder unlocked. If Unlock did not release, Lock
	// would spin for the full 10 s timeout — too slow for a unit test, so we
	// verify via ReleaseToken semantics: reconstruct a fresh mutex and confirm
	// Lock returns without blocking (Lock sets a fresh token every call, so
	// after m.Unlock() the key is gone and the new mutex's SetNX wins at once).
	m2 := reg.Get(9001, inventory.TypeValueEquip)
	done := make(chan struct{})
	go func() {
		m2.Lock()
		m2.Unlock()
		close(done)
	}()
	select {
	case <-done:
		// passed
	case <-time.After(2 * time.Second):
		t.Fatal("second Lock() blocked for > 2 s after Unlock — key was not released")
	}
}

// TestUnlockNonHolder verifies that Unlock by a non-holder (wrong token)
// does NOT release the key. A holder's Unlock must be a CAS operation.
func TestUnlockNonHolder(t *testing.T) {
	reg := compartment.LockRegistry()
	require.NotNil(t, reg)

	holder := reg.Get(9002, inventory.TypeValueUse)
	holder.Lock()
	defer holder.Unlock()

	// A second mutex pointing at the same key; calling Unlock on it without
	// first calling Lock means its token is empty — ReleaseToken("") should
	// not match the holder's token.
	nonHolder := reg.Get(9002, inventory.TypeValueUse)
	nonHolder.Unlock() // must be a no-op CAS miss, not an unconditional DEL

	// Holder should still hold the lock: a fresh mutex for the same key should
	// block immediately (SetNX fails). We verify by checking that a goroutine
	// waiting for it does NOT finish within 200 ms.
	challenger := reg.Get(9002, inventory.TypeValueUse)
	done := make(chan struct{})
	go func() {
		challenger.Lock()
		challenger.Unlock()
		close(done)
	}()
	select {
	case <-done:
		t.Fatal("challenger acquired lock that should still be held by holder")
	case <-time.After(200 * time.Millisecond):
		// correct: challenger is still waiting
	}
}

// TestTwoMutexesMutualExclusion verifies that two goroutines acquiring the
// same key are mutually exclusive (one blocks while the other holds).
func TestTwoMutexesMutualExclusion(t *testing.T) {
	reg := compartment.LockRegistry()
	require.NotNil(t, reg)

	m1 := reg.Get(9003, inventory.TypeValueSetup)
	m1.Lock()

	acquired := make(chan struct{})
	go func() {
		m2 := reg.Get(9003, inventory.TypeValueSetup)
		m2.Lock()
		close(acquired)
		m2.Unlock()
	}()

	// m2 must not acquire while m1 holds
	select {
	case <-acquired:
		m1.Unlock()
		t.Fatal("m2 acquired lock while m1 was still holding it")
	case <-time.After(150 * time.Millisecond):
		// correct: m2 is blocked
	}

	m1.Unlock()

	// Now m2 should acquire within the retry window
	select {
	case <-acquired:
		// passed
	case <-time.After(2 * time.Second):
		t.Fatal("m2 never acquired lock after m1 released it")
	}
}

// ---------------------------------------------------------------------------
// Reservation registry tests
// ---------------------------------------------------------------------------

func testReservationTenant() tenant.Model {
	tm, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	return tm
}

// TestAddAndGetReservedQuantity verifies the basic add→get round-trip.
func TestAddAndGetReservedQuantity(t *testing.T) {
	reg := compartment.GetReservationRegistry()
	require.NotNil(t, reg)

	tm := testReservationTenant()
	txId := uuid.New()
	characterId := uint32(1001)
	invType := inventory.TypeValueEquip
	slot := int16(1)

	_, err := reg.AddReservation(tm, txId, characterId, invType, slot, 1002000, 5, 10*time.Minute)
	require.NoError(t, err)

	qty := reg.GetReservedQuantity(tm, characterId, invType, slot)
	assert.Equal(t, uint32(5), qty)

	// Cleanup
	_, _ = reg.RemoveReservation(tm, txId, characterId, invType, slot)
}

// TestRemoveReservation verifies that RemoveReservation removes the correct entry.
func TestRemoveReservation(t *testing.T) {
	reg := compartment.GetReservationRegistry()
	require.NotNil(t, reg)

	tm := testReservationTenant()
	characterId := uint32(1002)
	invType := inventory.TypeValueUse
	slot := int16(2)
	tx1 := uuid.New()
	tx2 := uuid.New()

	_, err := reg.AddReservation(tm, tx1, characterId, invType, slot, 2000000, 3, 10*time.Minute)
	require.NoError(t, err)
	_, err = reg.AddReservation(tm, tx2, characterId, invType, slot, 2000000, 7, 10*time.Minute)
	require.NoError(t, err)

	// Remove the first; tx2's qty should remain
	_, err = reg.RemoveReservation(tm, tx1, characterId, invType, slot)
	require.NoError(t, err)

	qty := reg.GetReservedQuantity(tm, characterId, invType, slot)
	assert.Equal(t, uint32(7), qty)

	// Remove non-existent
	_, err = reg.RemoveReservation(tm, tx1, characterId, invType, slot)
	assert.Error(t, err, "expected error when removing non-existent reservation")

	// Cleanup
	_, _ = reg.RemoveReservation(tm, tx2, characterId, invType, slot)
}

// TestSwapReservation verifies that SwapReservation exchanges slot contents.
func TestSwapReservation(t *testing.T) {
	reg := compartment.GetReservationRegistry()
	require.NotNil(t, reg)

	tm := testReservationTenant()
	characterId := uint32(1003)
	invType := inventory.TypeValueSetup
	slotA := int16(3)
	slotB := int16(5)
	txA := uuid.New()
	txB := uuid.New()

	_, err := reg.AddReservation(tm, txA, characterId, invType, slotA, 3000000, 2, 10*time.Minute)
	require.NoError(t, err)
	_, err = reg.AddReservation(tm, txB, characterId, invType, slotB, 4000000, 4, 10*time.Minute)
	require.NoError(t, err)

	reg.SwapReservation(tm, characterId, invType, slotA, slotB)

	assert.Equal(t, uint32(4), reg.GetReservedQuantity(tm, characterId, invType, slotA), "slotA should now have slotB's reservation")
	assert.Equal(t, uint32(2), reg.GetReservedQuantity(tm, characterId, invType, slotB), "slotB should now have slotA's reservation")

	// Cleanup
	_, _ = reg.RemoveReservation(tm, txB, characterId, invType, slotA)
	_, _ = reg.RemoveReservation(tm, txA, characterId, invType, slotB)
}

// TestExpiredReservationFilteredOut verifies that expired reservations are
// silently dropped at read time (in-value expiry filter).
func TestExpiredReservationFilteredOut(t *testing.T) {
	reg := compartment.GetReservationRegistry()
	require.NotNil(t, reg)

	tm := testReservationTenant()
	characterId := uint32(1004)
	invType := inventory.TypeValueETC
	slot := int16(1)
	txExpired := uuid.New()
	txValid := uuid.New()

	// Add an already-expired reservation (1 ns duration — expires immediately)
	_, err := reg.AddReservation(tm, txExpired, characterId, invType, slot, 4000000, 10, 1*time.Nanosecond)
	require.NoError(t, err)

	// Small sleep to ensure expiry time has passed
	time.Sleep(5 * time.Millisecond)

	// Add a valid reservation
	_, err = reg.AddReservation(tm, txValid, characterId, invType, slot, 4000000, 8, 10*time.Minute)
	require.NoError(t, err)

	// Expired one must not count
	qty := reg.GetReservedQuantity(tm, characterId, invType, slot)
	assert.Equal(t, uint32(8), qty, "expired reservation must be filtered out; only valid qty should count")

	// Cleanup
	_, _ = reg.RemoveReservation(tm, txValid, characterId, invType, slot)
}
