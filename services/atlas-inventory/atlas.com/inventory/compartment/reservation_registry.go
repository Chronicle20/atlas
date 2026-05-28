package compartment

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	atlas "github.com/Chronicle20/atlas/libs/atlas-redis"
	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
)

type ReservationRequest struct {
	Slot     int16
	ItemId   uint32
	Quantity int16
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

type reservationJSON struct {
	Id       uuid.UUID `json:"id"`
	ItemId   uint32    `json:"itemId"`
	Quantity uint32    `json:"quantity"`
	Expiry   time.Time `json:"expiry"`
}

type reservationKey struct {
	characterId   uint32
	inventoryType inventory.Type
	slot          int16
}

type ReservationRegistry struct {
	reg *atlas.TenantRegistry[reservationKey, []reservationJSON]
}

var resReg *ReservationRegistry

func InitReservationRegistry(client *goredis.Client) {
	resReg = &ReservationRegistry{
		reg: atlas.NewTenantRegistry[reservationKey, []reservationJSON](
			client,
			"reservation",
			func(k reservationKey) string {
				return fmt.Sprintf("%d:%d:%d", k.characterId, k.inventoryType, k.slot)
			},
		),
	}
}

func GetReservationRegistry() *ReservationRegistry {
	return resReg
}

func (r *ReservationRegistry) loadReservations(t tenant.Model, key reservationKey) []Reservation {
	items, err := r.reg.Get(context.Background(), t, key)
	if errors.Is(err, atlas.ErrNotFound) || err != nil {
		return nil
	}
	now := time.Now()
	result := make([]Reservation, 0, len(items))
	for _, item := range items {
		if item.Expiry.After(now) {
			result = append(result, Reservation{
				id:       item.Id,
				itemId:   item.ItemId,
				quantity: item.Quantity,
				expiry:   item.Expiry,
			})
		}
	}
	return result
}

func (r *ReservationRegistry) storeReservations(t tenant.Model, key reservationKey, reservations []Reservation) {
	if len(reservations) == 0 {
		_ = r.reg.Remove(context.Background(), t, key)
		return
	}
	items := make([]reservationJSON, 0, len(reservations))
	for _, res := range reservations {
		items = append(items, reservationJSON{
			Id:       res.id,
			ItemId:   res.itemId,
			Quantity: res.quantity,
			Expiry:   res.expiry,
		})
	}
	_ = r.reg.Put(context.Background(), t, key, items)
}

func (r *ReservationRegistry) AddReservation(t tenant.Model, transactionId uuid.UUID, characterId uint32, inventoryType inventory.Type, slot int16, itemId uint32, quantity uint32, expiry time.Duration) (Reservation, error) {
	key := reservationKey{characterId: characterId, inventoryType: inventoryType, slot: slot}

	res := Reservation{
		id:       transactionId,
		itemId:   itemId,
		quantity: quantity,
		expiry:   time.Now().Add(expiry),
	}

	existing := r.loadReservations(t, key)
	existing = append(existing, res)
	r.storeReservations(t, key, existing)

	return res, nil
}

func (r *ReservationRegistry) RemoveReservation(t tenant.Model, transactionId uuid.UUID, characterId uint32, inventoryType inventory.Type, slot int16) (Reservation, error) {
	key := reservationKey{characterId: characterId, inventoryType: inventoryType, slot: slot}

	existing := r.loadReservations(t, key)
	if len(existing) == 0 {
		return Reservation{}, errors.New("does not exist")
	}

	var removed Reservation
	found := false
	newReservations := make([]Reservation, 0, len(existing))
	for _, res := range existing {
		if res.Id() != transactionId {
			newReservations = append(newReservations, res)
		} else {
			removed = res
			found = true
		}
	}

	if !found {
		return Reservation{}, errors.New("does not exist")
	}

	r.storeReservations(t, key, newReservations)
	return removed, nil
}

func (r *ReservationRegistry) SwapReservation(t tenant.Model, characterId uint32, inventoryType inventory.Type, oldSlot int16, newSlot int16) {
	oldKey := reservationKey{characterId: characterId, inventoryType: inventoryType, slot: oldSlot}
	newKey := reservationKey{characterId: characterId, inventoryType: inventoryType, slot: newSlot}

	reservations1 := r.loadReservations(t, oldKey)
	reservations2 := r.loadReservations(t, newKey)

	if len(reservations1) > 0 || len(reservations2) > 0 {
		r.storeReservations(t, oldKey, reservations2)
		r.storeReservations(t, newKey, reservations1)
	}
}

func (r *ReservationRegistry) GetReservedQuantity(t tenant.Model, characterId uint32, inventoryType inventory.Type, slot int16) uint32 {
	key := reservationKey{characterId: characterId, inventoryType: inventoryType, slot: slot}

	reservations := r.loadReservations(t, key)
	if len(reservations) == 0 {
		return 0
	}

	var totalQuantity uint32
	for _, res := range reservations {
		totalQuantity += res.Quantity()
	}

	return totalQuantity
}
