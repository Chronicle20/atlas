// Package compartment produces the COMMAND_TOPIC_COMPARTMENT reserve and
// cancel commands that back trade staging. Under the reserve-at-staging model
// (design §5.3) nothing leaves the owner's inventory when an item is staged —
// atlas-inventory simply marks the quantity reserved, which is enough to block
// the competing move / merge / drop paths until settlement or teardown.
package compartment

import (
	compartmentmsg "atlas-trades/kafka/message/compartment"
	"time"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"

	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory/slot"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

// requestReserveCommandProvider asks atlas-inventory to hold `quantity` of the
// asset in `sourceSlot` for `expiry`.
//
// reservationId is set on BOTH the envelope and the body: atlas-inventory's
// handler prefers the envelope and falls back to the body when the envelope is
// nil (kafka/consumer/compartment/consumer.go:164-168), and the reservation
// registry keys the reservation by exactly that id — it is the handle
// CANCEL_RESERVATION later needs, which is why StagedItem carries it.
//
// The wire quantity is int16 (compartmentmsg.ItemBody.Quantity), narrower than
// the uint16 the client's PUT_ITEM decodes, so callers must have already
// rejected anything above math.MaxInt16: a negative would be widened to a huge
// uint32 by atlas-inventory's AddReservation and lock the whole stack.
func requestReserveCommandProvider(reservationId uuid.UUID, characterId character.Id, inventoryType inventory.Type, sourceSlot slot.Position, templateId item.Id, quantity int16, expiry time.Duration) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &compartmentmsg.Command[compartmentmsg.RequestReserveCommandBody]{
		TransactionId: reservationId,
		CharacterId:   uint32(characterId),
		InventoryType: byte(inventoryType),
		Type:          compartmentmsg.CommandRequestReserve,
		Body: compartmentmsg.RequestReserveCommandBody{
			TransactionId: reservationId,
			ExpirySeconds: uint32(expiry.Seconds()),
			Items: []compartmentmsg.ItemBody{{
				Source:   int16(sourceSlot),
				ItemId:   uint32(templateId),
				Quantity: quantity,
			}},
		},
	}
	return producer.SingleMessageProvider(key, value)
}

// cancelReservationCommandProvider releases the reservation identified by
// reservationId on (inventoryType, sourceSlot). atlas-inventory treats an
// unknown reservation as a no-op (compartment/processor.go:822-825), so a
// cancel that races an expiry is harmless.
func cancelReservationCommandProvider(reservationId uuid.UUID, characterId character.Id, inventoryType inventory.Type, sourceSlot slot.Position) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &compartmentmsg.Command[compartmentmsg.CancelReservationCommandBody]{
		TransactionId: reservationId,
		CharacterId:   uint32(characterId),
		InventoryType: byte(inventoryType),
		Type:          compartmentmsg.CommandCancelReservation,
		Body: compartmentmsg.CancelReservationCommandBody{
			TransactionId: reservationId,
			Slot:          int16(sourceSlot),
		},
	}
	return producer.SingleMessageProvider(key, value)
}
