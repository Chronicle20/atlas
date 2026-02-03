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

				// Emit expire command
				emitExpireCommand(l, pp, ctx, asset.ExpireCommand{
					TransactionId:  uuid.New(),
					CharacterId:    characterId,
					AccountId:      0,
					AssetId:        uint32(assetId),
					TemplateId:     a.TemplateId,
					InventoryType:  int8(comp.Type),
					Slot:           a.Slot,
					ReplaceItemId:  replaceInfo.ReplaceItemId,
					ReplaceMessage: replaceInfo.ReplaceMessage,
					Source:         "INVENTORY",
				})
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

			// Emit expire command
			emitExpireCommand(l, pp, ctx, asset.ExpireCommand{
				TransactionId:  uuid.New(),
				CharacterId:    0,
				AccountId:      accountId,
				WorldId:        worldId,
				AssetId:        a.GetAssetId(),
				TemplateId:     a.TemplateId,
				InventoryType:  0,
				Slot:           a.Slot,
				ReplaceItemId:  replaceInfo.ReplaceItemId,
				ReplaceMessage: replaceInfo.ReplaceMessage,
				Source:         "STORAGE",
			})
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

			// Emit expire command
			emitExpireCommand(l, pp, ctx, asset.ExpireCommand{
				TransactionId:  uuid.New(),
				CharacterId:    0,
				AccountId:      accountId,
				AssetId:        item.GetItemId(),
				TemplateId:     item.TemplateId,
				InventoryType:  0,
				Slot:           0,
				ReplaceItemId:  replaceInfo.ReplaceItemId,
				ReplaceMessage: replaceInfo.ReplaceMessage,
				Source:         "CASHSHOP",
			})
		}
	}
}

func emitExpireCommand(l logrus.FieldLogger, pp producer.Provider, ctx context.Context, cmd asset.ExpireCommand) {
	err := pp(asset.EnvCommandTopicAssetExpire)(kafkaProducer.SingleMessageProvider(kafkaProducer.CreateKey(int(cmd.AssetId)), cmd))
	if err != nil {
		l.WithError(err).Errorf("Failed to emit expire command for asset [%d] (template [%d]).", cmd.AssetId, cmd.TemplateId)
	} else {
		l.Infof("Emitted expire command for asset [%d] (template [%d]), source [%s].", cmd.AssetId, cmd.TemplateId, cmd.Source)
	}
}
