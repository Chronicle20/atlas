package mts

import (
	mtsmsg "atlas-channel/kafka/message/mts"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

// COMMAND_TOPIC_MTS command-message providers. Each mirrors the atlas-mts
// consumer's expected envelope (mtsmsg.Command[E]) keyed on the acting
// character so per-character ordering is preserved (the same keying the
// messenger/cashshop command producers use).

func CreateListingCommandProvider(transactionId uuid.UUID, worldId world.Id, sellerId uint32, sellerAccountId uint32, itemId uint32, quantity uint32, price uint32, isAuction bool, buyNowPrice uint32, durationHours uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(sellerId))
	value := &mtsmsg.Command[mtsmsg.CreateListingCommandBody]{
		TransactionId: transactionId,
		Type:          mtsmsg.CommandCreateListing,
		Body: mtsmsg.CreateListingCommandBody{
			WorldId:         byte(worldId),
			SellerId:        sellerId,
			SellerAccountId: sellerAccountId,
			ItemId:          itemId,
			Quantity:        quantity,
			Price:           price,
			IsAuction:       isAuction,
			BuyNowPrice:     buyNowPrice,
			DurationHours:   durationHours,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func BuyCommandProvider(transactionId uuid.UUID, worldId world.Id, listingId uuid.UUID, buyerId uint32, buyerAccountId uint32, sellerAccountId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(buyerId))
	value := &mtsmsg.Command[mtsmsg.BuyCommandBody]{
		TransactionId: transactionId,
		Type:          mtsmsg.CommandBuy,
		Body: mtsmsg.BuyCommandBody{
			ListingId:       listingId,
			WorldId:         byte(worldId),
			BuyerId:         buyerId,
			BuyerAccountId:  buyerAccountId,
			SellerAccountId: sellerAccountId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func PlaceBidCommandProvider(transactionId uuid.UUID, worldId world.Id, listingId uuid.UUID, bidderId uint32, bidderAccountId uint32, amount uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(bidderId))
	value := &mtsmsg.Command[mtsmsg.PlaceBidCommandBody]{
		TransactionId: transactionId,
		Type:          mtsmsg.CommandPlaceBid,
		Body: mtsmsg.PlaceBidCommandBody{
			ListingId:       listingId,
			WorldId:         byte(worldId),
			BidderId:        bidderId,
			BidderAccountId: bidderAccountId,
			Amount:          amount,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func CancelListingCommandProvider(transactionId uuid.UUID, worldId world.Id, listingId uuid.UUID, sellerId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(sellerId))
	value := &mtsmsg.Command[mtsmsg.CancelListingCommandBody]{
		TransactionId: transactionId,
		Type:          mtsmsg.CommandCancelListing,
		Body: mtsmsg.CancelListingCommandBody{
			ListingId: listingId,
			WorldId:   byte(worldId),
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func TakeHomeCommandProvider(transactionId uuid.UUID, worldId world.Id, holdingId uuid.UUID, characterId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &mtsmsg.Command[mtsmsg.TakeHomeCommandBody]{
		TransactionId: transactionId,
		Type:          mtsmsg.CommandTakeHome,
		Body: mtsmsg.TakeHomeCommandBody{
			HoldingId:   holdingId,
			WorldId:     byte(worldId),
			CharacterId: characterId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func RegisterWishCommandProvider(worldId world.Id, characterId uint32, itemId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &mtsmsg.Command[mtsmsg.RegisterWishCommandBody]{
		Type: mtsmsg.CommandRegisterWish,
		Body: mtsmsg.RegisterWishCommandBody{
			WishId:      uuid.New(),
			WorldId:     byte(worldId),
			CharacterId: characterId,
			ItemId:      itemId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func RemoveWishCommandProvider(worldId world.Id, wishId uuid.UUID, characterId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &mtsmsg.Command[mtsmsg.RemoveWishCommandBody]{
		Type: mtsmsg.CommandRemoveWish,
		Body: mtsmsg.RemoveWishCommandBody{
			WishId:  wishId,
			WorldId: byte(worldId),
		},
	}
	return producer.SingleMessageProvider(key, value)
}
