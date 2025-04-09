package compartment

import (
	"errors"
	"github.com/Chronicle20/atlas-constants/inventory"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"sync"
	"time"
)

type ReservationRequest struct {
	Slot     int16
	ItemId   uint32
	Quantity int16
}

type ReservationKey struct {
	tenant        tenant.Model
	characterId   uint32
	inventoryType inventory.Type
	slot          int16
}

func (k ReservationKey) CharacterId() uint32 {
	return k.characterId
}

func (k ReservationKey) Tenant() tenant.Model {
	return k.tenant
}

func NewReservationKey(tenant tenant.Model, characterId uint32, inventoryType inventory.Type, slot int16) ReservationKey {
	return ReservationKey{tenant: tenant, characterId: characterId, inventoryType: inventoryType, slot: slot}
}

type Reservation struct {
	id       uuid.UUID
	itemId   uint32
	quantity uint32
	expiry   time.Time
}

func (r Reservation) Id() uuid.UUID { return r.id }

func (r Reservation) ItemId() uint32 { return r.itemId }

func (r Reservation) Quantity() uint32 { return r.quantity }

func (r Reservation) Expiry() time.Time { return r.expiry }

type ReservationRegistry struct {
	lock         sync.RWMutex
	reservations map[ReservationKey][]Reservation
}

var (
	instance        *ReservationRegistry
	reservationOnce sync.Once
)

func GetReservationRegistry() *ReservationRegistry {
	reservationOnce.Do(func() {
		instance = &ReservationRegistry{
			reservations: make(map[ReservationKey][]Reservation),
		}
	})
	return instance
}

func (r *ReservationRegistry) AddReservation(t tenant.Model, transactionId uuid.UUID, characterId uint32, inventoryType inventory.Type, slot int16, itemId uint32, quantity uint32, expiry time.Duration) (Reservation, error) {
	key := ReservationKey{t, characterId, inventoryType, slot}

	r.lock.Lock()
	defer r.lock.Unlock()

	res := Reservation{
		id:       transactionId,
		itemId:   itemId,
		quantity: quantity,
		expiry:   time.Now().Add(expiry),
	}

	r.reservations[key] = append(r.reservations[key], res)

	return res, nil
}

func (r *ReservationRegistry) RemoveReservation(t tenant.Model, transactionId uuid.UUID, characterId uint32, inventoryType inventory.Type, slot int16) (Reservation, error) {
	key := ReservationKey{t, characterId, inventoryType, slot}

	r.lock.Lock()
	defer r.lock.Unlock()

	reservations, exists := r.reservations[key]
	if !exists {
		return Reservation{}, errors.New("does not exist")
	}

	var removed Reservation
	newReservations := make([]Reservation, 0, len(reservations))
	for _, res := range reservations {
		if res.Id() != transactionId {
			newReservations = append(newReservations, res)
		} else {
			removed = res
		}
	}

	if len(newReservations) > 0 {
		r.reservations[key] = newReservations
	} else {
		delete(r.reservations, key)
	}
	return removed, nil
}

func (r *ReservationRegistry) SwapReservation(t tenant.Model, characterId uint32, inventoryType inventory.Type, oldSlot int16, newSlot int16) {
	oldKey := NewReservationKey(t, characterId, inventoryType, oldSlot)
	newKey := NewReservationKey(t, characterId, inventoryType, newSlot)

	r.lock.Lock()
	defer r.lock.Unlock()

	reservations1 := r.reservations[oldKey]
	reservations2 := r.reservations[newKey]

	if len(reservations1) > 0 || len(reservations2) > 0 {
		r.reservations[oldKey] = reservations2
		r.reservations[newKey] = reservations1
	}
}

func (r *ReservationRegistry) ExpireReservations() {
	r.lock.Lock()
	defer r.lock.Unlock()

	now := time.Now()
	for key, reservations := range r.reservations {
		newReservations := reservations[:0]
		for _, res := range reservations {
			if res.Expiry().After(now) {
				newReservations = append(newReservations, res)
			}
		}

		if len(newReservations) > 0 {
			r.reservations[key] = newReservations
		} else {
			delete(r.reservations, key)
		}
	}
}

func (r *ReservationRegistry) GetReservedQuantity(t tenant.Model, characterId uint32, inventoryType inventory.Type, slot int16) uint32 {
	key := NewReservationKey(t, characterId, inventoryType, slot)

	r.lock.RLock()
	defer r.lock.RUnlock()

	reservations, exists := r.reservations[key]
	if !exists {
		return 0
	}

	var totalQuantity uint32
	for _, res := range reservations {
		totalQuantity += res.Quantity()
	}

	return totalQuantity
}

func (r *ReservationRegistry) RemoveAllReservationsForCharacter(tenant tenant.Model, characterId uint32) {
	r.lock.Lock()
	defer r.lock.Unlock()

	for key := range r.reservations {
		if key.Tenant() == tenant && key.CharacterId() == characterId {
			delete(r.reservations, key)
		}
	}
}
