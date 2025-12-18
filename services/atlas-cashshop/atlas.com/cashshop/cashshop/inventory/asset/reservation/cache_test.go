package reservation

import (
	"testing"
	"time"
)

func TestReservationCache_IsReserved(t *testing.T) {
	cache := &ReservationCache{
		reservations: make(map[uint32]uint32),
		expirations:  make(map[uint32]time.Time),
	}

	assetID := uint32(1)
	characterID := uint32(123)

	// Initially, the asset should not be reserved
	if cache.IsReserved(assetID) {
		t.Errorf("Expected asset to not be reserved initially")
	}

	// Reserve the asset
	if !cache.Reserve(assetID, characterID) {
		t.Errorf("Failed to reserve asset")
	}

	// Now the asset should be reserved
	if !cache.IsReserved(assetID) {
		t.Errorf("Expected asset to be reserved after reservation")
	}

	// Release the reservation
	cache.Release(assetID)

	// After release, the asset should not be reserved
	if cache.IsReserved(assetID) {
		t.Errorf("Expected asset to not be reserved after release")
	}
}

func TestReservationCache_Reserve(t *testing.T) {
	cache := &ReservationCache{
		reservations: make(map[uint32]uint32),
		expirations:  make(map[uint32]time.Time),
	}

	assetID := uint32(1)
	characterID := uint32(123)
	anotherCharacterID := uint32(456)

	// First reservation should succeed
	if !cache.Reserve(assetID, characterID) {
		t.Errorf("Failed to reserve asset")
	}

	// Second reservation for the same asset should fail
	if cache.Reserve(assetID, anotherCharacterID) {
		t.Errorf("Expected second reservation to fail")
	}

	// Release the reservation
	cache.Release(assetID)

	// After release, a new reservation should succeed
	if !cache.Reserve(assetID, anotherCharacterID) {
		t.Errorf("Failed to reserve asset after release")
	}
}

func TestReservationCache_Expiration(t *testing.T) {
	cache := &ReservationCache{
		reservations: make(map[uint32]uint32),
		expirations:  make(map[uint32]time.Time),
	}

	assetID := uint32(1)
	characterID := uint32(123)

	// Reserve the asset with a short expiration time for testing
	cache.mu.Lock()
	cache.reservations[assetID] = characterID
	cache.expirations[assetID] = time.Now().Add(10 * time.Millisecond)
	cache.mu.Unlock()

	// Initially, the asset should be reserved
	if !cache.IsReserved(assetID) {
		t.Errorf("Expected asset to be reserved initially")
	}

	// Wait for the reservation to expire
	time.Sleep(20 * time.Millisecond)

	// After expiration, the asset should not be reserved
	if cache.IsReserved(assetID) {
		t.Errorf("Expected asset to not be reserved after expiration")
	}
}

func TestReservationCache_Singleton(t *testing.T) {
	// Get the singleton instance
	instance1 := GetInstance()

	// Reserve an asset
	assetID := uint32(1)
	characterID := uint32(123)

	if !instance1.Reserve(assetID, characterID) {
		t.Errorf("Failed to reserve asset")
	}

	// Get the singleton instance again
	instance2 := GetInstance()

	// The asset should still be reserved in the second instance
	if !instance2.IsReserved(assetID) {
		t.Errorf("Expected asset to be reserved in second instance")
	}

	// Release the reservation
	instance2.Release(assetID)

	// The asset should not be reserved in either instance
	if instance1.IsReserved(assetID) || instance2.IsReserved(assetID) {
		t.Errorf("Expected asset to not be reserved after release")
	}
}
