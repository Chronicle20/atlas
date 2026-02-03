package character

import (
	"atlas-asset-expiration/cashshop"
	"atlas-asset-expiration/data"
	"atlas-asset-expiration/expiration"
	"atlas-asset-expiration/inventory"
	"atlas-asset-expiration/kafka/message/asset"
	"atlas-asset-expiration/kafka/producer"
	"atlas-asset-expiration/storage"
	"context"
	"strconv"
	"time"

	kafkaProducer "github.com/Chronicle20/atlas-kafka/producer"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// CheckAndExpire checks all items for a character and emits expire commands for expired items
func CheckAndExpire(l logrus.FieldLogger) func(pp producer.Provider) func(ctx context.Context) func(characterId, accountId uint32, worldId byte) {
	return func(pp producer.Provider) func(ctx context.Context) func(characterId, accountId uint32, worldId byte) {
		return func(ctx context.Context) func(characterId, accountId uint32, worldId byte) {
			return func(characterId, accountId uint32, worldId byte) {
				now := time.Now()
				l.Infof("Checking expiration for character [%d], account [%d], world [%d].", characterId, accountId, worldId)

				// Check inventory items
				checkInventory(l, pp, ctx, characterId, now)

				// Check storage items
				checkStorage(l, pp, ctx, accountId, worldId, now)

				// Check cashshop items
				checkCashshop(l, pp, ctx, accountId, now)
			}
		}
	}
}

func checkInventory(l logrus.FieldLogger, pp producer.Provider, ctx context.Context, characterId uint32, now time.Time) {
	inv, err := inventory.GetInventory(l)(ctx)(characterId)
	if err != nil {
		l.WithError(err).Warnf("Failed to get inventory for character [%d].", characterId)
		return
	}

	for _, comp := range inv.Compartments {
		assets, err := inventory.GetAssets(l)(ctx)(characterId, comp.Id)
		if err != nil {
			l.WithError(err).Warnf("Failed to get assets for compartment [%s].", comp.Id)
			continue
		}

		for _, a := range assets {
			if expiration.IsExpired(a.Expiration, now) {
				l.Infof("Asset [%s] (template [%d]) is expired for character [%d].", a.Id, a.TemplateId, characterId)

				// Get replacement info
				replaceInfo := data.GetReplaceInfo(l)(ctx)(a.TemplateId)

				// Parse asset ID
				assetId, _ := strconv.ParseUint(a.Id, 10, 32)

				// Emit expire command to compartment topic
				emitCompartmentExpireCommand(l, pp, ctx, characterId, uint32(assetId), a.TemplateId, byte(comp.Type), a.Slot, replaceInfo.ReplaceItemId, replaceInfo.ReplaceMessage)
			}
		}
	}
}

func checkStorage(l logrus.FieldLogger, pp producer.Provider, ctx context.Context, accountId uint32, worldId byte, now time.Time) {
	assets, err := storage.GetAssets(l)(ctx)(accountId, worldId)
	if err != nil {
		l.WithError(err).Warnf("Failed to get storage assets for account [%d], world [%d].", accountId, worldId)
		return
	}

	for _, a := range assets {
		if expiration.IsExpired(a.Expiration, now) {
			l.Infof("Storage asset [%s] (template [%d]) is expired for account [%d].", a.Id, a.TemplateId, accountId)

			// Get replacement info
			replaceInfo := data.GetReplaceInfo(l)(ctx)(a.TemplateId)

			// Emit expire command to storage topic
			emitStorageExpireCommand(l, pp, ctx, accountId, worldId, a.GetAssetId(), a.TemplateId, a.Slot, replaceInfo.ReplaceItemId, replaceInfo.ReplaceMessage)
		}
	}
}

func checkCashshop(l logrus.FieldLogger, pp producer.Provider, ctx context.Context, accountId uint32, now time.Time) {
	items, err := cashshop.GetAllItems(l)(ctx)(accountId)
	if err != nil {
		l.WithError(err).Warnf("Failed to get cashshop items for account [%d].", accountId)
		return
	}

	for _, item := range items {
		if expiration.IsExpired(item.Expiration, now) {
			l.Infof("Cashshop item [%s] (template [%d]) is expired for account [%d].", item.Id, item.TemplateId, accountId)

			// Get replacement info
			replaceInfo := data.GetReplaceInfo(l)(ctx)(item.TemplateId)

			// Emit expire command to cashshop topic
			emitCashShopExpireCommand(l, pp, ctx, accountId, item.GetItemId(), item.TemplateId, replaceInfo.ReplaceItemId, replaceInfo.ReplaceMessage)
		}
	}
}

func emitStorageExpireCommand(l logrus.FieldLogger, pp producer.Provider, ctx context.Context, accountId uint32, worldId byte, assetId uint32, templateId uint32, slot int16, replaceItemId uint32, replaceMessage string) {
	cmd := asset.StorageExpireCommand{
		TransactionId: uuid.New(),
		WorldId:       worldId,
		AccountId:     accountId,
		Type:          asset.CommandTypeExpire,
		Body: asset.StorageExpireBody{
			CharacterId:    0,
			AssetId:        assetId,
			TemplateId:     templateId,
			InventoryType:  0,
			Slot:           slot,
			ReplaceItemId:  replaceItemId,
			ReplaceMessage: replaceMessage,
		},
	}
	err := pp(asset.EnvCommandTopicStorage)(kafkaProducer.SingleMessageProvider(kafkaProducer.CreateKey(int(assetId)), cmd))
	if err != nil {
		l.WithError(err).Errorf("Failed to emit storage expire command for asset [%d] (template [%d]).", assetId, templateId)
	} else {
		l.Infof("Emitted storage expire command for asset [%d] (template [%d]).", assetId, templateId)
	}
}

func emitCashShopExpireCommand(l logrus.FieldLogger, pp producer.Provider, ctx context.Context, accountId uint32, assetId uint32, templateId uint32, replaceItemId uint32, replaceMessage string) {
	cmd := asset.CashShopExpireCommand{
		CharacterId: 0,
		Type:        asset.CommandTypeExpire,
		Body: asset.CashShopExpireBody{
			AccountId:      accountId,
			WorldId:        0,
			AssetId:        assetId,
			TemplateId:     templateId,
			InventoryType:  0,
			Slot:           0,
			ReplaceItemId:  replaceItemId,
			ReplaceMessage: replaceMessage,
		},
	}
	err := pp(asset.EnvCommandTopicCashShop)(kafkaProducer.SingleMessageProvider(kafkaProducer.CreateKey(int(assetId)), cmd))
	if err != nil {
		l.WithError(err).Errorf("Failed to emit cashshop expire command for asset [%d] (template [%d]).", assetId, templateId)
	} else {
		l.Infof("Emitted cashshop expire command for asset [%d] (template [%d]).", assetId, templateId)
	}
}

func emitCompartmentExpireCommand(l logrus.FieldLogger, pp producer.Provider, ctx context.Context, characterId uint32, assetId uint32, templateId uint32, inventoryType byte, slot int16, replaceItemId uint32, replaceMessage string) {
	cmd := asset.CompartmentExpireCommand{
		TransactionId: uuid.New(),
		CharacterId:   characterId,
		InventoryType: inventoryType,
		Type:          asset.CommandTypeExpire,
		Body: asset.CompartmentExpireBody{
			AssetId:        assetId,
			TemplateId:     templateId,
			Slot:           slot,
			ReplaceItemId:  replaceItemId,
			ReplaceMessage: replaceMessage,
		},
	}
	err := pp(asset.EnvCommandTopicCompartment)(kafkaProducer.SingleMessageProvider(kafkaProducer.CreateKey(int(assetId)), cmd))
	if err != nil {
		l.WithError(err).Errorf("Failed to emit compartment expire command for asset [%d] (template [%d]).", assetId, templateId)
	} else {
		l.Infof("Emitted compartment expire command for asset [%d] (template [%d]).", assetId, templateId)
	}
}
