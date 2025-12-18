package reservation

import (
	"sync"
	"time"
)

// ReservationCache is a thread-safe cache for asset reservations
type ReservationCache struct {
	mu           sync.Mutex
	reservations map[uint32]uint32 // Maps item ID to character ID
	expirations  map[uint32]time.Time
}

var (
	instance *ReservationCache
	once     sync.Once
)

// GetInstance returns the singleton instance of the ReservationCache
func GetInstance() *ReservationCache {
	once.Do(func() {
		instance = &ReservationCache{
			reservations: make(map[uint32]uint32),
			expirations:  make(map[uint32]time.Time),
		}
		// Start a goroutine to clean up expired reservations
		go instance.cleanupExpired()
	})
	return instance
}

// IsReserved checks if an item is already reserved
func (c *ReservationCache) IsReserved(itemID uint32) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	_, reserved := c.reservations[itemID]
	if reserved {
		// Check if the reservation has expired
		expiration, ok := c.expirations[itemID]
		if ok && time.Now().After(expiration) {
			// Reservation has expired, remove it
			delete(c.reservations, itemID)
			delete(c.expirations, itemID)
			return false
		}
	}
	return reserved
}

// Reserve attempts to reserve an item for a character
// Returns true if the reservation was successful, false if the item is already reserved
func (c *ReservationCache) Reserve(itemID uint32, characterID uint32) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Check if the item is already reserved
	_, reserved := c.reservations[itemID]
	if reserved {
		// Check if the reservation has expired
		expiration, ok := c.expirations[itemID]
		if ok && time.Now().After(expiration) {
			// Reservation has expired, remove it
			delete(c.reservations, itemID)
			delete(c.expirations, itemID)
		} else {
			// Item is still reserved
			return false
		}
	}

	// Reserve the item
	c.reservations[itemID] = characterID
	// Set expiration time (5 minutes from now)
	c.expirations[itemID] = time.Now().Add(5 * time.Minute)
	return true
}

// Release releases a reservation for an item
func (c *ReservationCache) Release(itemID uint32) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.reservations, itemID)
	delete(c.expirations, itemID)
}

// cleanupExpired periodically removes expired reservations
func (c *ReservationCache) cleanupExpired() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		c.mu.Lock()
		now := time.Now()
		for itemID, expiration := range c.expirations {
			if now.After(expiration) {
				delete(c.reservations, itemID)
				delete(c.expirations, itemID)
			}
		}
		c.mu.Unlock()
	}
}
