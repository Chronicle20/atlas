package shop

import (
	merchant "atlas-merchant/kafka/message/merchant"

	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

func StatusEventShopOpenedProvider(characterId uint32, shopId uuid.UUID, m Model) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &merchant.StatusEvent[merchant.StatusEventShopOpenedBody]{
		CharacterId: characterId,
		Type:        merchant.StatusEventShopOpened,
		Body: merchant.StatusEventShopOpenedBody{
			ShopId:     shopId.String(),
			ShopType:   byte(m.ShopType()),
			WorldId:    m.WorldId(),
			ChannelId:  m.ChannelId(),
			MapId:      m.MapId(),
			InstanceId: m.InstanceId(),
			Title:      m.Title(),
			X:          m.X(),
			Y:          m.Y(),
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func StatusEventShopClosedProvider(characterId uint32, shopId uuid.UUID, reason CloseReason) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &merchant.StatusEvent[merchant.StatusEventShopClosedBody]{
		CharacterId: characterId,
		Type:        merchant.StatusEventShopClosed,
		Body: merchant.StatusEventShopClosedBody{
			ShopId:      shopId.String(),
			CloseReason: byte(reason),
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func StatusEventMaintenanceEnteredProvider(characterId uint32, shopId uuid.UUID) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &merchant.StatusEvent[merchant.StatusEventVisitorBody]{
		CharacterId: characterId,
		Type:        merchant.StatusEventMaintenanceEntered,
		Body: merchant.StatusEventVisitorBody{
			ShopId:      shopId.String(),
			CharacterId: characterId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func StatusEventMaintenanceExitedProvider(characterId uint32, shopId uuid.UUID) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &merchant.StatusEvent[merchant.StatusEventVisitorBody]{
		CharacterId: characterId,
		Type:        merchant.StatusEventMaintenanceExited,
		Body: merchant.StatusEventVisitorBody{
			ShopId:      shopId.String(),
			CharacterId: characterId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func StatusEventVisitorEnteredProvider(characterId uint32, shopId uuid.UUID) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &merchant.StatusEvent[merchant.StatusEventVisitorBody]{
		CharacterId: characterId,
		Type:        merchant.StatusEventVisitorEntered,
		Body: merchant.StatusEventVisitorBody{
			ShopId:      shopId.String(),
			CharacterId: characterId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func StatusEventVisitorExitedProvider(characterId uint32, shopId uuid.UUID) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &merchant.StatusEvent[merchant.StatusEventVisitorBody]{
		CharacterId: characterId,
		Type:        merchant.StatusEventVisitorExited,
		Body: merchant.StatusEventVisitorBody{
			ShopId:      shopId.String(),
			CharacterId: characterId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func StatusEventVisitorEjectedProvider(characterId uint32, shopId uuid.UUID) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &merchant.StatusEvent[merchant.StatusEventVisitorBody]{
		CharacterId: characterId,
		Type:        merchant.StatusEventVisitorEjected,
		Body: merchant.StatusEventVisitorBody{
			ShopId:      shopId.String(),
			CharacterId: characterId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func StatusEventCapacityFullProvider(characterId uint32, shopId uuid.UUID) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &merchant.StatusEvent[merchant.StatusEventCapacityFullBody]{
		CharacterId: characterId,
		Type:        merchant.StatusEventCapacityFull,
		Body: merchant.StatusEventCapacityFullBody{
			ShopId: shopId.String(),
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func StatusEventPurchaseFailedProvider(characterId uint32, shopId uuid.UUID, reason string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &merchant.StatusEvent[merchant.StatusEventPurchaseFailedBody]{
		CharacterId: characterId,
		Type:        merchant.StatusEventPurchaseFailed,
		Body: merchant.StatusEventPurchaseFailedBody{
			ShopId: shopId.String(),
			Reason: reason,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func StatusEventMessageSentProvider(characterId uint32, shopId uuid.UUID, slot byte, content string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &merchant.StatusEvent[merchant.StatusEventMessageSentBody]{
		CharacterId: characterId,
		Type:        merchant.StatusEventMessageSent,
		Body: merchant.StatusEventMessageSentBody{
			ShopId:      shopId.String(),
			CharacterId: characterId,
			Slot:        slot,
			Content:     content,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func ListingEventPurchasedProvider(shopId uuid.UUID, listingIndex uint16, buyerCharacterId uint32, bundleCount uint16, bundlesRemaining uint16) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(buyerCharacterId))
	value := &merchant.ListingEvent[merchant.ListingEventPurchasedBody]{
		ShopId: shopId.String(),
		Type:   merchant.ListingEventPurchased,
		Body: merchant.ListingEventPurchasedBody{
			ListingIndex:     listingIndex,
			BuyerCharacterId: buyerCharacterId,
			BundleCount:      bundleCount,
			BundlesRemaining: bundlesRemaining,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
