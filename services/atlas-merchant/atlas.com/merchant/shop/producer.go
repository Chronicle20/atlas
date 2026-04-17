package shop

import (
	asset2 "atlas-merchant/kafka/message/asset"
	character "atlas-merchant/kafka/message/character"
	"atlas-merchant/kafka/message/compartment"
	merchant "atlas-merchant/kafka/message/merchant"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
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

func StatusEventVisitorEnteredProvider(characterId uint32, shopId uuid.UUID, slot byte) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &merchant.StatusEvent[merchant.StatusEventVisitorBody]{
		CharacterId: characterId,
		Type:        merchant.StatusEventVisitorEntered,
		Body: merchant.StatusEventVisitorBody{
			ShopId:      shopId.String(),
			CharacterId: characterId,
			Slot:        slot,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func StatusEventVisitorExitedProvider(characterId uint32, shopId uuid.UUID, slot byte) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &merchant.StatusEvent[merchant.StatusEventVisitorBody]{
		CharacterId: characterId,
		Type:        merchant.StatusEventVisitorExited,
		Body: merchant.StatusEventVisitorBody{
			ShopId:      shopId.String(),
			CharacterId: characterId,
			Slot:        slot,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func StatusEventVisitorEjectedProvider(characterId uint32, shopId uuid.UUID, slot byte) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &merchant.StatusEvent[merchant.StatusEventVisitorBody]{
		CharacterId: characterId,
		Type:        merchant.StatusEventVisitorEjected,
		Body: merchant.StatusEventVisitorBody{
			ShopId:      shopId.String(),
			CharacterId: characterId,
			Slot:        slot,
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

func ReleaseAssetCommandProvider(transactionId uuid.UUID, characterId uint32, inventoryType byte, assetId uint32, quantity uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &compartment.Command[compartment.ReleaseCommandBody]{
		TransactionId: transactionId,
		CharacterId:   characterId,
		InventoryType: inventoryType,
		Type:          compartment.CommandRelease,
		Body: compartment.ReleaseCommandBody{
			TransactionId: transactionId,
			AssetId:       assetId,
			Quantity:      quantity,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func AcceptAssetCommandProvider(transactionId uuid.UUID, characterId uint32, inventoryType byte, templateId uint32, assetData asset2.AssetData) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &compartment.Command[compartment.AcceptCommandBody]{
		TransactionId: transactionId,
		CharacterId:   characterId,
		InventoryType: inventoryType,
		Type:          compartment.CommandAccept,
		Body: compartment.AcceptCommandBody{
			TransactionId: transactionId,
			TemplateId:    templateId,
			AssetData:     assetData,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func ChangeMesoCommandProvider(transactionId uuid.UUID, worldId world.Id, characterId uint32, actorId uint32, actorType string, amount int32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &character.Command[character.RequestChangeMesoBody]{
		TransactionId: transactionId,
		WorldId:       worldId,
		CharacterId:   characterId,
		Type:          character.CommandRequestChangeMeso,
		Body: character.RequestChangeMesoBody{
			ActorId:   actorId,
			ActorType: actorType,
			Amount:    amount,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
