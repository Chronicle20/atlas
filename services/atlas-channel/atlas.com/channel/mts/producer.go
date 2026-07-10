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

func CreateListingCommandProvider(transactionId uuid.UUID, worldId world.Id, sellerId uint32, sellerAccountId uint32, sellerName string, saleType string, sourceInventoryType byte, assetId uint32, quantity uint32, listValue uint32, buyNowPrice *uint32, durationHours int, minIncrement uint32, category string, subCategory string, offerWishSerial uint32, offerWishOwnerId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(sellerId))
	value := &mtsmsg.Command[mtsmsg.CreateListingCommandBody]{
		TransactionId: transactionId,
		Type:          mtsmsg.CommandCreateListing,
		Body: mtsmsg.CreateListingCommandBody{
			WorldId:             byte(worldId),
			SellerId:            sellerId,
			SellerAccountId:     sellerAccountId,
			SellerName:          sellerName,
			SaleType:            saleType,
			SourceInventoryType: sourceInventoryType,
			AssetId:             assetId,
			Quantity:            quantity,
			ListValue:           listValue,
			BuyNowPrice:         buyNowPrice,
			DurationHours:       durationHours,
			MinIncrement:        minIncrement,
			Category:            category,
			SubCategory:         subCategory,
			OfferWishSerial:     offerWishSerial,
			OfferWishOwnerId:    offerWishOwnerId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func BuyCommandProvider(transactionId uuid.UUID, worldId world.Id, serial uint32, buyerId uint32, buyerAccountId uint32, buyNow bool, resultKind string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(buyerId))
	value := &mtsmsg.Command[mtsmsg.BuyCommandBody]{
		TransactionId: transactionId,
		Type:          mtsmsg.CommandBuy,
		Body: mtsmsg.BuyCommandBody{
			WorldId:        byte(worldId),
			Serial:         serial,
			BuyerId:        buyerId,
			BuyerAccountId: buyerAccountId,
			BuyNow:         buyNow,
			ResultKind:     resultKind,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func PlaceBidCommandProvider(transactionId uuid.UUID, worldId world.Id, serial uint32, bidderId uint32, bidderAccountId uint32, amount uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(bidderId))
	value := &mtsmsg.Command[mtsmsg.PlaceBidCommandBody]{
		TransactionId: transactionId,
		Type:          mtsmsg.CommandPlaceBid,
		Body: mtsmsg.PlaceBidCommandBody{
			WorldId:         byte(worldId),
			Serial:          serial,
			BidderId:        bidderId,
			BidderAccountId: bidderAccountId,
			Amount:          amount,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func CancelListingCommandProvider(transactionId uuid.UUID, worldId world.Id, serial uint32, sellerId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(sellerId))
	value := &mtsmsg.Command[mtsmsg.CancelListingCommandBody]{
		TransactionId: transactionId,
		Type:          mtsmsg.CommandCancelListing,
		Body: mtsmsg.CancelListingCommandBody{
			WorldId:  byte(worldId),
			Serial:   serial,
			SellerId: sellerId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func TakeHomeCommandProvider(transactionId uuid.UUID, worldId world.Id, serial uint32, characterId uint32, inventoryType byte, slot int16) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &mtsmsg.Command[mtsmsg.TakeHomeCommandBody]{
		TransactionId: transactionId,
		Type:          mtsmsg.CommandTakeHome,
		Body: mtsmsg.TakeHomeCommandBody{
			WorldId:       byte(worldId),
			Serial:        serial,
			CharacterId:   characterId,
			InventoryType: inventoryType,
			Slot:          slot,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func RegisterWishCommandProvider(worldId world.Id, characterId uint32, itemId uint32, price uint32, count uint32, origin string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &mtsmsg.Command[mtsmsg.RegisterWishCommandBody]{
		Type: mtsmsg.CommandRegisterWish,
		Body: mtsmsg.RegisterWishCommandBody{
			WishId:      uuid.New(),
			WorldId:     byte(worldId),
			CharacterId: characterId,
			ItemId:      itemId,
			Price:       price,
			Count:       count,
			Origin:      origin,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func RemoveWishCommandProvider(worldId world.Id, wishId uuid.UUID, characterId uint32, origin string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &mtsmsg.Command[mtsmsg.RemoveWishCommandBody]{
		Type: mtsmsg.CommandRemoveWish,
		Body: mtsmsg.RemoveWishCommandBody{
			WishId:  wishId,
			WorldId: byte(worldId),
			Origin:  origin,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
