package mts

import (
	"atlas-mts/kafka/message/mts"
	"encoding/binary"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

// keyFor derives a stable partition key from a uuid (its first 4 bytes), so all
// events for one transaction land on the same partition in order.
func keyFor(id uuid.UUID) []byte {
	return producer.CreateKey(int(binary.LittleEndian.Uint32(id[:4])))
}

// ListingCreatedStatusEventProvider builds a LISTING_CREATED event.
func ListingCreatedStatusEventProvider(transactionId uuid.UUID, worldId byte, listingId uuid.UUID, sellerId uint32, itemId uint32, saleType string) model.Provider[[]kafka.Message] {
	value := &mts.StatusEvent[mts.StatusEventListingCreatedBody]{
		TransactionId: transactionId,
		Type:          mts.StatusEventTypeListingCreated,
		Body: mts.StatusEventListingCreatedBody{
			WorldId:   worldId,
			ListingId: listingId,
			SellerId:  sellerId,
			ItemId:    itemId,
			SaleType:  saleType,
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
func ListingSoldStatusEventProvider(transactionId uuid.UUID, worldId byte, listingId uuid.UUID, sellerId uint32, buyerId uint32, itemId uint32, saleType string, resultKind string, price uint32) model.Provider[[]kafka.Message] {
	value := &mts.StatusEvent[mts.StatusEventListingSoldBody]{
		TransactionId: transactionId,
		Type:          mts.StatusEventTypeListingSold,
		Body: mts.StatusEventListingSoldBody{
			WorldId:    worldId,
			ListingId:  listingId,
			SellerId:   sellerId,
			BuyerId:    buyerId,
			ItemId:     itemId,
			SaleType:   saleType,
			ResultKind: resultKind,
			Price:      price,
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

// WishAddedStatusEventProvider builds a WISH_ADDED event. Origin echoes the
// initiating command's WishOrigin so the channel picks the right clientbound
// result.
func WishAddedStatusEventProvider(transactionId uuid.UUID, worldId byte, wishId uuid.UUID, characterId uint32, itemId uint32, origin string) model.Provider[[]kafka.Message] {
	value := &mts.StatusEvent[mts.StatusEventWishAddedBody]{
		TransactionId: transactionId,
		Type:          mts.StatusEventTypeWishAdded,
		Body: mts.StatusEventWishAddedBody{
			WorldId:     worldId,
			WishId:      wishId,
			CharacterId: characterId,
			ItemId:      itemId,
			Origin:      origin,
		},
	}
	return producer.SingleMessageProvider(keyFor(transactionId), value)
}

// ListingCreateFailedStatusEventProvider builds a LISTING_CREATE_FAILED event for
// a rejected listing creation, carrying the originating sellerId + a fail reason.
func ListingCreateFailedStatusEventProvider(transactionId uuid.UUID, worldId byte, sellerId uint32, reasonKey string) model.Provider[[]kafka.Message] {
	value := &mts.StatusEvent[mts.StatusEventListingCreateFailedBody]{
		TransactionId: transactionId,
		Type:          mts.StatusEventTypeListingCreateFailed,
		Body: mts.StatusEventListingCreateFailedBody{
			WorldId:   worldId,
			SellerId:  sellerId,
			ReasonKey: reasonKey,
		},
	}
	return producer.SingleMessageProvider(keyFor(transactionId), value)
}

// ListingCancelFailedStatusEventProvider builds a LISTING_CANCEL_FAILED event for
// a rejected cancel, carrying the originating sellerId + serial + a fail reason.
func ListingCancelFailedStatusEventProvider(transactionId uuid.UUID, worldId byte, serial uint32, sellerId uint32, reasonKey string) model.Provider[[]kafka.Message] {
	value := &mts.StatusEvent[mts.StatusEventListingCancelFailedBody]{
		TransactionId: transactionId,
		Type:          mts.StatusEventTypeListingCancelFailed,
		Body: mts.StatusEventListingCancelFailedBody{
			WorldId:   worldId,
			Serial:    serial,
			SellerId:  sellerId,
			ReasonKey: reasonKey,
		},
	}
	return producer.SingleMessageProvider(keyFor(transactionId), value)
}

// TakeHomeFailedStatusEventProvider builds a TAKE_HOME_FAILED event for a rejected
// take-home, carrying the originating characterId + serial + a fail reason.
func TakeHomeFailedStatusEventProvider(transactionId uuid.UUID, worldId byte, serial uint32, characterId uint32, reasonKey string) model.Provider[[]kafka.Message] {
	value := &mts.StatusEvent[mts.StatusEventTakeHomeFailedBody]{
		TransactionId: transactionId,
		Type:          mts.StatusEventTypeTakeHomeFailed,
		Body: mts.StatusEventTakeHomeFailedBody{
			WorldId:     worldId,
			Serial:      serial,
			CharacterId: characterId,
			ReasonKey:   reasonKey,
		},
	}
	return producer.SingleMessageProvider(keyFor(transactionId), value)
}

// BuyFailedStatusEventProvider builds a BUY_FAILED event for a rejected buy /
// buy-now, carrying the originating buyerId + serial + a fail reason.
func BuyFailedStatusEventProvider(transactionId uuid.UUID, worldId byte, serial uint32, buyerId uint32, reason string, resultKind string) model.Provider[[]kafka.Message] {
	value := &mts.StatusEvent[mts.StatusEventBuyFailedBody]{
		TransactionId: transactionId,
		Type:          mts.StatusEventTypeBuyFailed,
		Body: mts.StatusEventBuyFailedBody{
			WorldId:    worldId,
			Serial:     serial,
			BuyerId:    buyerId,
			ReasonKey:  reason,
			ResultKind: resultKind,
		},
	}
	return producer.SingleMessageProvider(keyFor(transactionId), value)
}

// BidFailedStatusEventProvider builds a BID_FAILED event for a rejected place-bid,
// carrying the originating bidderId + serial + a fail reason.
func BidFailedStatusEventProvider(transactionId uuid.UUID, worldId byte, serial uint32, bidderId uint32, reason string) model.Provider[[]kafka.Message] {
	value := &mts.StatusEvent[mts.StatusEventBidFailedBody]{
		TransactionId: transactionId,
		Type:          mts.StatusEventTypeBidFailed,
		Body: mts.StatusEventBidFailedBody{
			WorldId:   worldId,
			Serial:    serial,
			BidderId:  bidderId,
			ReasonKey: reason,
		},
	}
	return producer.SingleMessageProvider(keyFor(transactionId), value)
}

// WishRemovedStatusEventProvider builds a WISH_REMOVED event. Origin echoes the
// initiating command's WishOrigin so the channel picks the right clientbound
// result.
func WishRemovedStatusEventProvider(transactionId uuid.UUID, worldId byte, wishId uuid.UUID, characterId uint32, origin string) model.Provider[[]kafka.Message] {
	value := &mts.StatusEvent[mts.StatusEventWishRemovedBody]{
		TransactionId: transactionId,
		Type:          mts.StatusEventTypeWishRemoved,
		Body: mts.StatusEventWishRemovedBody{
			WorldId:     worldId,
			WishId:      wishId,
			CharacterId: characterId,
			Origin:      origin,
		},
	}
	return producer.SingleMessageProvider(keyFor(transactionId), value)
}
