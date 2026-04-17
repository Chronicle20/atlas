package merchant

import (
	merchant2 "atlas-channel/kafka/message/merchant"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

func PlaceShopCommandProvider(f field.Model, characterId uint32, shopType byte, title string, permitItemId uint32, x int16, y int16) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &merchant2.Command[merchant2.CommandPlaceShopBody]{
		WorldId:     f.WorldId(),
		ChannelId:   f.ChannelId(),
		CharacterId: characterId,
		Type:        merchant2.CommandPlaceShop,
		Body: merchant2.CommandPlaceShopBody{
			ShopType:     shopType,
			Title:        title,
			MapId:        uint32(f.MapId()),
			InstanceId:   f.Instance(),
			X:            x,
			Y:            y,
			PermitItemId: permitItemId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func OpenShopCommandProvider(characterId uint32, shopId uuid.UUID) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &merchant2.Command[merchant2.CommandOpenShopBody]{
		CharacterId: characterId,
		Type:        merchant2.CommandOpenShop,
		Body: merchant2.CommandOpenShopBody{
			ShopId: shopId.String(),
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func CloseShopCommandProvider(characterId uint32, shopId uuid.UUID) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &merchant2.Command[merchant2.CommandCloseShopBody]{
		CharacterId: characterId,
		Type:        merchant2.CommandCloseShop,
		Body: merchant2.CommandCloseShopBody{
			ShopId: shopId.String(),
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func EnterMaintenanceCommandProvider(characterId uint32, shopId uuid.UUID) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &merchant2.Command[merchant2.CommandEnterMaintenanceBody]{
		CharacterId: characterId,
		Type:        merchant2.CommandEnterMaintenance,
		Body: merchant2.CommandEnterMaintenanceBody{
			ShopId: shopId.String(),
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func ExitMaintenanceCommandProvider(characterId uint32, shopId uuid.UUID) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &merchant2.Command[merchant2.CommandExitMaintenanceBody]{
		CharacterId: characterId,
		Type:        merchant2.CommandExitMaintenance,
		Body: merchant2.CommandExitMaintenanceBody{
			ShopId: shopId.String(),
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func AddListingCommandProvider(characterId uint32, shopId uuid.UUID, inventoryType byte, slot int16, quantity uint16, bundleSize uint16, pricePerBundle uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &merchant2.Command[merchant2.CommandAddListingBody]{
		CharacterId: characterId,
		Type:        merchant2.CommandAddListing,
		Body: merchant2.CommandAddListingBody{
			ShopId:         shopId.String(),
			InventoryType:  inventoryType,
			Slot:           slot,
			BundleSize:     bundleSize,
			BundleCount:    quantity,
			PricePerBundle: pricePerBundle,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func RemoveListingCommandProvider(characterId uint32, shopId uuid.UUID, listingIndex uint16) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &merchant2.Command[merchant2.CommandRemoveListingBody]{
		CharacterId: characterId,
		Type:        merchant2.CommandRemoveListing,
		Body: merchant2.CommandRemoveListingBody{
			ShopId:       shopId.String(),
			ListingIndex: listingIndex,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func EnterShopCommandProvider(characterId uint32, shopId uuid.UUID) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &merchant2.Command[merchant2.CommandEnterShopBody]{
		CharacterId: characterId,
		Type:        merchant2.CommandEnterShop,
		Body: merchant2.CommandEnterShopBody{
			ShopId: shopId.String(),
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func ExitShopCommandProvider(characterId uint32, shopId uuid.UUID) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &merchant2.Command[merchant2.CommandExitShopBody]{
		CharacterId: characterId,
		Type:        merchant2.CommandExitShop,
		Body: merchant2.CommandExitShopBody{
			ShopId: shopId.String(),
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func SendMessageCommandProvider(characterId uint32, shopId uuid.UUID, content string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &merchant2.Command[merchant2.CommandSendMessageBody]{
		CharacterId: characterId,
		Type:        merchant2.CommandSendMessage,
		Body: merchant2.CommandSendMessageBody{
			ShopId:  shopId.String(),
			Content: content,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func PurchaseBundleCommandProvider(characterId uint32, shopId uuid.UUID, listingIndex uint16, bundleCount uint16) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &merchant2.Command[merchant2.CommandPurchaseBundleBody]{
		CharacterId: characterId,
		Type:        merchant2.CommandPurchaseBundle,
		Body: merchant2.CommandPurchaseBundleBody{
			ShopId:       shopId.String(),
			ListingIndex: listingIndex,
			BundleCount:  bundleCount,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
