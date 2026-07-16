package merchant

import (
	"atlas-channel/asset"
	assetmsg "atlas-channel/kafka/message/asset"
	merchant2 "atlas-channel/kafka/message/merchant"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	inventory2 "github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
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

// itemTypeFromInventoryType maps the client's inventory/compartment type to
// the merchant listing ItemType byte convention: 1 for equipment, 2 for
// everything else. Matches socket/writer/shop_scanner.go's ShopScannerRecords,
// which treats ItemType==1 as the signal to attach an equip GW_ItemSlotBase.
func itemTypeFromInventoryType(inventoryType byte) byte {
	if inventory2.Type(inventoryType) == inventory2.TypeValueEquip {
		return 1
	}
	return 2
}

// snapshotFromAsset builds the full point-in-time asset snapshot carried on
// the AddListing command, mirroring atlas-merchant's own
// kafka/message/asset.AssetData shape (task-127 fix: without this the
// listing persists with an empty snapshot and the owl item-id search can
// never render an equip preview or correct stackable quantity for it).
func snapshotFromAsset(a asset.Model) assetmsg.AssetData {
	return assetmsg.AssetData{
		Expiration:     a.Expiration(),
		CreatedAt:      a.CreatedAt(),
		Quantity:       a.Quantity(),
		OwnerId:        a.OwnerId(),
		Flag:           a.Flag(),
		Rechargeable:   a.Rechargeable(),
		Strength:       a.Strength(),
		Dexterity:      a.Dexterity(),
		Intelligence:   a.Intelligence(),
		Luck:           a.Luck(),
		Hp:             a.Hp(),
		Mp:             a.Mp(),
		WeaponAttack:   a.WeaponAttack(),
		MagicAttack:    a.MagicAttack(),
		WeaponDefense:  a.WeaponDefense(),
		MagicDefense:   a.MagicDefense(),
		Accuracy:       a.Accuracy(),
		Avoidability:   a.Avoidability(),
		Hands:          a.Hands(),
		Speed:          a.Speed(),
		Jump:           a.Jump(),
		Slots:          a.Slots(),
		LevelType:      a.LevelType(),
		Level:          a.Level(),
		Experience:     a.Experience(),
		HammersApplied: a.HammersApplied(),
		EquippedSince:  a.EquippedSince(),
		CashId:         a.CashId(),
		CommodityId:    a.CommodityId(),
		PurchaseBy:     a.PurchaseBy(),
		PetId:          a.PetId(),
	}
}

// AddListingCommandProvider builds the ADD_LISTING command carrying the
// resolved asset's ItemId, ItemType, AssetId and full ItemSnapshot. a is the
// asset resolved from the character's inventory slot by the caller (see
// Processor.AddListing); without it atlas-merchant persists the listing with
// itemId=0/itemType=0/assetId=0/empty snapshot, which is unreachable by the
// item-id-keyed owl search (task-127).
func AddListingCommandProvider(characterId uint32, shopId uuid.UUID, inventoryType byte, slot int16, quantity uint16, bundleSize uint16, pricePerBundle uint32, a asset.Model) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &merchant2.Command[merchant2.CommandAddListingBody]{
		CharacterId: characterId,
		Type:        merchant2.CommandAddListing,
		Body: merchant2.CommandAddListingBody{
			ShopId:         shopId.String(),
			ItemId:         a.TemplateId(),
			ItemType:       itemTypeFromInventoryType(inventoryType),
			InventoryType:  inventoryType,
			Slot:           slot,
			BundleSize:     bundleSize,
			BundleCount:    quantity,
			PricePerBundle: pricePerBundle,
			AssetId:        a.Id(),
			ItemSnapshot:   snapshotFromAsset(a),
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

func EnterShopCommandProvider(characterId uint32, shopId uuid.UUID, visitorName string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &merchant2.Command[merchant2.CommandEnterShopBody]{
		CharacterId: characterId,
		Type:        merchant2.CommandEnterShop,
		Body: merchant2.CommandEnterShopBody{
			VisitorName: visitorName,
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

func RecordItemSearchCommandProvider(f field.Model, characterId uint32, itemId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &merchant2.Command[merchant2.CommandRecordItemSearchBody]{
		WorldId:     f.WorldId(),
		ChannelId:   f.ChannelId(),
		CharacterId: characterId,
		Type:        merchant2.CommandRecordItemSearch,
		Body: merchant2.CommandRecordItemSearchBody{
			ItemId: itemId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func WithdrawMesoCommandProvider(characterId uint32, shopId uuid.UUID) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &merchant2.Command[merchant2.CommandWithdrawMesoBody]{
		CharacterId: characterId,
		Type:        merchant2.CommandWithdrawMeso,
		Body:        merchant2.CommandWithdrawMesoBody{ShopId: shopId.String()},
	}
	return producer.SingleMessageProvider(key, value)
}

func OrganizeListingsCommandProvider(characterId uint32, shopId uuid.UUID) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &merchant2.Command[merchant2.CommandOrganizeListingsBody]{
		CharacterId: characterId,
		Type:        merchant2.CommandOrganizeListings,
		Body:        merchant2.CommandOrganizeListingsBody{ShopId: shopId.String()},
	}
	return producer.SingleMessageProvider(key, value)
}

func AddBlacklistCommandProvider(characterId uint32, shopId uuid.UUID, name string, bannedCharacterId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &merchant2.Command[merchant2.CommandBlacklistBody]{
		CharacterId: characterId,
		Type:        merchant2.CommandAddBlacklist,
		Body:        merchant2.CommandBlacklistBody{ShopId: shopId.String(), Name: name, BannedCharacterId: bannedCharacterId},
	}
	return producer.SingleMessageProvider(key, value)
}

func RemoveBlacklistCommandProvider(characterId uint32, shopId uuid.UUID, name string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &merchant2.Command[merchant2.CommandBlacklistBody]{
		CharacterId: characterId,
		Type:        merchant2.CommandRemoveBlacklist,
		Body:        merchant2.CommandBlacklistBody{ShopId: shopId.String(), Name: name},
	}
	return producer.SingleMessageProvider(key, value)
}
