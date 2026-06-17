package mts

import (
	"atlas-mts/kafka/message/mts"
	"encoding/binary"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

// keyFor derives a stable partition key from a uuid (its first 4 bytes), so all
// events for one transaction land on the same partition in order.
func keyFor(id uuid.UUID) []byte {
	return producer.CreateKey(int(binary.LittleEndian.Uint32(id[:4])))
}

// ListingCreatedStatusEventProvider builds a LISTING_CREATED event.
func ListingCreatedStatusEventProvider(transactionId uuid.UUID, worldId byte, listingId uuid.UUID, sellerId uint32, itemId uint32) model.Provider[[]kafka.Message] {
	value := &mts.StatusEvent[mts.StatusEventListingCreatedBody]{
		TransactionId: transactionId,
		Type:          mts.StatusEventTypeListingCreated,
		Body: mts.StatusEventListingCreatedBody{
			WorldId:   worldId,
			ListingId: listingId,
			SellerId:  sellerId,
			ItemId:    itemId,
		},
	}
	return producer.SingleMessageProvider(keyFor(transactionId), value)
}

// ListingCancelledStatusEventProvider builds a LISTING_CANCELLED event for a
// cancelled listing whose item moved to the seller's holding.
func ListingCancelledStatusEventProvider(transactionId uuid.UUID, worldId byte, listingId uuid.UUID, holdingId uuid.UUID, sellerId uint32, itemId uint32) model.Provider[[]kafka.Message] {
	value := &mts.StatusEvent[mts.StatusEventListingCancelledBody]{
		TransactionId: transactionId,
		Type:          mts.StatusEventTypeListingCancelled,
		Body: mts.StatusEventListingCancelledBody{
			WorldId:   worldId,
			ListingId: listingId,
			HoldingId: holdingId,
			SellerId:  sellerId,
			ItemId:    itemId,
		},
	}
	return producer.SingleMessageProvider(keyFor(transactionId), value)
}

// BidPlacedStatusEventProvider builds a BID_PLACED event.
func BidPlacedStatusEventProvider(transactionId uuid.UUID, worldId byte, listingId uuid.UUID, bidderId uint32, amount uint32) model.Provider[[]kafka.Message] {
	value := &mts.StatusEvent[mts.StatusEventBidPlacedBody]{
		TransactionId: transactionId,
		Type:          mts.StatusEventTypeBidPlaced,
		Body: mts.StatusEventBidPlacedBody{
			WorldId:   worldId,
			ListingId: listingId,
			BidderId:  bidderId,
			Amount:    amount,
		},
	}
	return producer.SingleMessageProvider(keyFor(transactionId), value)
}

// OutbidStatusEventProvider builds an OUTBID event for a displaced high bidder.
func OutbidStatusEventProvider(transactionId uuid.UUID, worldId byte, listingId uuid.UUID, previousBidderId uint32) model.Provider[[]kafka.Message] {
	value := &mts.StatusEvent[mts.StatusEventOutbidBody]{
		TransactionId: transactionId,
		Type:          mts.StatusEventTypeOutbid,
		Body: mts.StatusEventOutbidBody{
			WorldId:          worldId,
			ListingId:        listingId,
			PreviousBidderId: previousBidderId,
		},
	}
	return producer.SingleMessageProvider(keyFor(transactionId), value)
}

// ListingSoldStatusEventProvider builds a LISTING_SOLD event.
func ListingSoldStatusEventProvider(transactionId uuid.UUID, worldId byte, listingId uuid.UUID, buyerId uint32, itemId uint32) model.Provider[[]kafka.Message] {
	value := &mts.StatusEvent[mts.StatusEventListingSoldBody]{
		TransactionId: transactionId,
		Type:          mts.StatusEventTypeListingSold,
		Body: mts.StatusEventListingSoldBody{
			WorldId:   worldId,
			ListingId: listingId,
			BuyerId:   buyerId,
			ItemId:    itemId,
		},
	}
	return producer.SingleMessageProvider(keyFor(transactionId), value)
}

// ListingExpiredStatusEventProvider builds a LISTING_EXPIRED event.
func ListingExpiredStatusEventProvider(transactionId uuid.UUID, worldId byte, listingId uuid.UUID) model.Provider[[]kafka.Message] {
	value := &mts.StatusEvent[mts.StatusEventListingExpiredBody]{
		TransactionId: transactionId,
		Type:          mts.StatusEventTypeListingExpired,
		Body: mts.StatusEventListingExpiredBody{
			WorldId:   worldId,
			ListingId: listingId,
		},
	}
	return producer.SingleMessageProvider(keyFor(transactionId), value)
}

// ItemMovedToHoldingStatusEventProvider builds an ITEM_MOVED_TO_HOLDING event.
func ItemMovedToHoldingStatusEventProvider(transactionId uuid.UUID, worldId byte, holdingId uuid.UUID, ownerId uint32, itemId uint32) model.Provider[[]kafka.Message] {
	value := &mts.StatusEvent[mts.StatusEventItemMovedToHoldingBody]{
		TransactionId: transactionId,
		Type:          mts.StatusEventTypeItemMovedToHolding,
		Body: mts.StatusEventItemMovedToHoldingBody{
			WorldId:   worldId,
			HoldingId: holdingId,
			OwnerId:   ownerId,
			ItemId:    itemId,
		},
	}
	return producer.SingleMessageProvider(keyFor(transactionId), value)
}

// ItemTakenHomeStatusEventProvider builds an ITEM_TAKEN_HOME event.
func ItemTakenHomeStatusEventProvider(transactionId uuid.UUID, worldId byte, holdingId uuid.UUID, characterId uint32, itemId uint32) model.Provider[[]kafka.Message] {
	value := &mts.StatusEvent[mts.StatusEventItemTakenHomeBody]{
		TransactionId: transactionId,
		Type:          mts.StatusEventTypeItemTakenHome,
		Body: mts.StatusEventItemTakenHomeBody{
			WorldId:     worldId,
			HoldingId:   holdingId,
			CharacterId: characterId,
			ItemId:      itemId,
		},
	}
	return producer.SingleMessageProvider(keyFor(transactionId), value)
}

// WishAddedStatusEventProvider builds a WISH_ADDED event.
func WishAddedStatusEventProvider(transactionId uuid.UUID, worldId byte, wishId uuid.UUID, characterId uint32, itemId uint32) model.Provider[[]kafka.Message] {
	value := &mts.StatusEvent[mts.StatusEventWishAddedBody]{
		TransactionId: transactionId,
		Type:          mts.StatusEventTypeWishAdded,
		Body: mts.StatusEventWishAddedBody{
			WorldId:     worldId,
			WishId:      wishId,
			CharacterId: characterId,
			ItemId:      itemId,
		},
	}
	return producer.SingleMessageProvider(keyFor(transactionId), value)
}

// WishRemovedStatusEventProvider builds a WISH_REMOVED event.
func WishRemovedStatusEventProvider(transactionId uuid.UUID, worldId byte, wishId uuid.UUID, characterId uint32) model.Provider[[]kafka.Message] {
	value := &mts.StatusEvent[mts.StatusEventWishRemovedBody]{
		TransactionId: transactionId,
		Type:          mts.StatusEventTypeWishRemoved,
		Body: mts.StatusEventWishRemovedBody{
			WorldId:     worldId,
			WishId:      wishId,
			CharacterId: characterId,
		},
	}
	return producer.SingleMessageProvider(keyFor(transactionId), value)
}
