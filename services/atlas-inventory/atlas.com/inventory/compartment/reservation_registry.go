package compartment

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/Chronicle20/atlas-constants/inventory"
	"github.com/Chronicle20/atlas-tenant"
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

type ReservationRegistry struct {
	client *goredis.Client
}

var resReg *ReservationRegistry

func InitReservationRegistry(client *goredis.Client) {
	resReg = &ReservationRegistry{client: client}
}

func GetReservationRegistry() *ReservationRegistry {
	return resReg
}

func reservationKey(t tenant.Model, characterId uint32, inventoryType inventory.Type, slot int16) string {
	return fmt.Sprintf("reservation:%s:%d:%d:%d", t.Id().String(), characterId, inventoryType, slot)
}

func (r *ReservationRegistry) loadReservations(key string) []Reservation {
	data, err := r.client.Get(context.Background(), key).Bytes()
	if err != nil {
		return nil
	}
	var items []reservationJSON
	if err := json.Unmarshal(data, &items); err != nil {
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

func (r *ReservationRegistry) storeReservations(key string, reservations []Reservation) {
	if len(reservations) == 0 {
		r.client.Del(context.Background(), key)
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
	data, err := json.Marshal(items)
	if err != nil {
		return
	}
	r.client.Set(context.Background(), key, data, 0)
}

func (r *ReservationRegistry) AddReservation(t tenant.Model, transactionId uuid.UUID, characterId uint32, inventoryType inventory.Type, slot int16, itemId uint32, quantity uint32, expiry time.Duration) (Reservation, error) {
	key := reservationKey(t, characterId, inventoryType, slot)

	res := Reservation{
		id:       transactionId,
		itemId:   itemId,
		quantity: quantity,
		expiry:   time.Now().Add(expiry),
	}

	existing := r.loadReservations(key)
	existing = append(existing, res)
	r.storeReservations(key, existing)

	return res, nil
}

func (r *ReservationRegistry) RemoveReservation(t tenant.Model, transactionId uuid.UUID, characterId uint32, inventoryType inventory.Type, slot int16) (Reservation, error) {
	key := reservationKey(t, characterId, inventoryType, slot)

	existing := r.loadReservations(key)
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

	r.storeReservations(key, newReservations)
	return removed, nil
}

func (r *ReservationRegistry) SwapReservation(t tenant.Model, characterId uint32, inventoryType inventory.Type, oldSlot int16, newSlot int16) {
	oldKey := reservationKey(t, characterId, inventoryType, oldSlot)
	newKey := reservationKey(t, characterId, inventoryType, newSlot)

	reservations1 := r.loadReservations(oldKey)
	reservations2 := r.loadReservations(newKey)

	if len(reservations1) > 0 || len(reservations2) > 0 {
		r.storeReservations(oldKey, reservations2)
		r.storeReservations(newKey, reservations1)
	}
}

func (r *ReservationRegistry) GetReservedQuantity(t tenant.Model, characterId uint32, inventoryType inventory.Type, slot int16) uint32 {
	key := reservationKey(t, characterId, inventoryType, slot)

	reservations := r.loadReservations(key)
	if len(reservations) == 0 {
		return 0
	}

	var totalQuantity uint32
	for _, res := range reservations {
		totalQuantity += res.Quantity()
	}

	return totalQuantity
}

