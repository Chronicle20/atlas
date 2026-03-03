package merchant

import (
	consumer2 "atlas-merchant/kafka/consumer"
	"atlas-merchant/frederick"
	"atlas-merchant/kafka/message/asset"
	"atlas-merchant/kafka/message/compartment"
	character "atlas-merchant/kafka/message/character"
	merchant2 "atlas-merchant/kafka/message/merchant"
	"atlas-merchant/kafka/producer"
	"atlas-merchant/listing"
	msg "atlas-merchant/message"
	"atlas-merchant/shop"
	"context"
	"encoding/json"
	"errors"

	"github.com/Chronicle20/atlas-constants/inventory"
	"github.com/Chronicle20/atlas-constants/item"
	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/handler"
	"github.com/Chronicle20/atlas-kafka/message"
	"github.com/Chronicle20/atlas-kafka/topic"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("merchant_command")(merchant2.EnvCommandTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) {
		return func(rf func(topic string, handler handler.Handler) (string, error)) {
			t, _ := topic.EnvProvider(l)(merchant2.EnvCommandTopic)()
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handlePlaceShopCommand(db))))
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleOpenShopCommand(db))))
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleCloseShopCommand(db))))
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleEnterMaintenanceCommand(db))))
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleExitMaintenanceCommand(db))))
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleAddListingCommand(db))))
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleRemoveListingCommand(db))))
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleUpdateListingCommand(db))))
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handlePurchaseBundleCommand(db))))
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleEnterShopCommand(db))))
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleExitShopCommand(db))))
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleSendMessageCommand(db))))
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleRetrieveFrederickCommand(db))))
		}
	}
}

func handlePlaceShopCommand(db *gorm.DB) message.Handler[merchant2.Command[merchant2.CommandPlaceShopBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e merchant2.Command[merchant2.CommandPlaceShopBody]) {
		if e.Type != merchant2.CommandPlaceShop {
			return
		}
		p := shop.NewProcessor(l, ctx, db)
		_, err := p.CreateShop(e.CharacterId, shop.ShopType(e.Body.ShopType), e.Body.Title, e.Body.MapId, e.Body.X, e.Body.Y, e.Body.PermitItemId)
		if err != nil {
			l.WithError(err).Errorf("Error creating shop for character [%d].", e.CharacterId)
		}
	}
}

func handleOpenShopCommand(db *gorm.DB) message.Handler[merchant2.Command[merchant2.CommandOpenShopBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e merchant2.Command[merchant2.CommandOpenShopBody]) {
		if e.Type != merchant2.CommandOpenShop {
			return
		}
		shopId, err := uuid.Parse(e.Body.ShopId)
		if err != nil {
			l.WithError(err).Errorf("Error parsing shopId [%s].", e.Body.ShopId)
			return
		}

		if err := shop.NewProcessor(l, ctx, db).OpenShopAndEmit(shopId, e.CharacterId); err != nil {
			l.WithError(err).Errorf("Error opening shop [%s].", shopId)
		}
	}
}

func handleCloseShopCommand(db *gorm.DB) message.Handler[merchant2.Command[merchant2.CommandCloseShopBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e merchant2.Command[merchant2.CommandCloseShopBody]) {
		if e.Type != merchant2.CommandCloseShop {
			return
		}
		shopId, err := uuid.Parse(e.Body.ShopId)
		if err != nil {
			l.WithError(err).Errorf("Error parsing shopId [%s].", e.Body.ShopId)
			return
		}

		if err := shop.NewProcessor(l, ctx, db).CloseShopAndEmit(shopId, e.CharacterId, shop.CloseReasonManualClose); err != nil {
			l.WithError(err).Errorf("Error closing shop [%s].", shopId)
		}
	}
}

func handleEnterMaintenanceCommand(db *gorm.DB) message.Handler[merchant2.Command[merchant2.CommandEnterMaintenanceBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e merchant2.Command[merchant2.CommandEnterMaintenanceBody]) {
		if e.Type != merchant2.CommandEnterMaintenance {
			return
		}
		shopId, err := uuid.Parse(e.Body.ShopId)
		if err != nil {
			l.WithError(err).Errorf("Error parsing shopId [%s].", e.Body.ShopId)
			return
		}

		if err := shop.NewProcessor(l, ctx, db).EnterMaintenanceAndEmit(shopId, e.CharacterId); err != nil {
			l.WithError(err).Errorf("Error entering maintenance for shop [%s].", shopId)
		}
	}
}

func handleExitMaintenanceCommand(db *gorm.DB) message.Handler[merchant2.Command[merchant2.CommandExitMaintenanceBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e merchant2.Command[merchant2.CommandExitMaintenanceBody]) {
		if e.Type != merchant2.CommandExitMaintenance {
			return
		}
		shopId, err := uuid.Parse(e.Body.ShopId)
		if err != nil {
			l.WithError(err).Errorf("Error parsing shopId [%s].", e.Body.ShopId)
			return
		}

		if err := shop.NewProcessor(l, ctx, db).ExitMaintenanceAndEmit(shopId, e.CharacterId); err != nil {
			l.WithError(err).Errorf("Error exiting maintenance for shop [%s].", shopId)
		}
	}
}

func handleAddListingCommand(db *gorm.DB) message.Handler[merchant2.Command[merchant2.CommandAddListingBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e merchant2.Command[merchant2.CommandAddListingBody]) {
		if e.Type != merchant2.CommandAddListing {
			return
		}
		shopId, err := uuid.Parse(e.Body.ShopId)
		if err != nil {
			l.WithError(err).Errorf("Error parsing shopId [%s].", e.Body.ShopId)
			return
		}

		// Extract flag from item snapshot for trade restriction validation.
		var flag uint16
		if e.Body.ItemSnapshot != nil {
			var snapshot asset.AssetData
			if err := json.Unmarshal(e.Body.ItemSnapshot, &snapshot); err == nil {
				flag = snapshot.Flag
			}
		}

		p := shop.NewProcessor(l, ctx, db)
		created, err := p.AddListingAndEmit(shopId, e.CharacterId, e.Body.ItemId, e.Body.ItemType, e.Body.BundleSize, e.Body.BundleCount, e.Body.PricePerBundle, e.Body.ItemSnapshot, flag, e.Body.InventoryType, e.Body.AssetId)
		if err != nil {
			l.WithError(err).Errorf("Error adding listing to shop [%s].", shopId)
			return
		}

		quantity := uint32(e.Body.BundleSize) * uint32(e.Body.BundleCount)
		l.Infof("Listing [%s] added to shop [%s], releasing asset [%d] (qty %d) from inventory.", created.Id(), shopId, e.Body.AssetId, quantity)
	}
}

func handleRemoveListingCommand(db *gorm.DB) message.Handler[merchant2.Command[merchant2.CommandRemoveListingBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e merchant2.Command[merchant2.CommandRemoveListingBody]) {
		if e.Type != merchant2.CommandRemoveListing {
			return
		}
		shopId, err := uuid.Parse(e.Body.ShopId)
		if err != nil {
			l.WithError(err).Errorf("Error parsing shopId [%s].", e.Body.ShopId)
			return
		}

		p := shop.NewProcessor(l, ctx, db)
		removed, err := p.RemoveListing(shopId, e.Body.ListingIndex)
		if err != nil {
			l.WithError(err).Errorf("Error removing listing from shop [%s].", shopId)
			return
		}

		// Return item to owner's inventory.
		acceptItemToOwner(l, ctx, e.CharacterId, removed)
	}
}

func handleUpdateListingCommand(db *gorm.DB) message.Handler[merchant2.Command[merchant2.CommandUpdateListingBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e merchant2.Command[merchant2.CommandUpdateListingBody]) {
		if e.Type != merchant2.CommandUpdateListing {
			return
		}
		shopId, err := uuid.Parse(e.Body.ShopId)
		if err != nil {
			l.WithError(err).Errorf("Error parsing shopId [%s].", e.Body.ShopId)
			return
		}

		p := shop.NewProcessor(l, ctx, db)
		err = p.UpdateListing(shopId, e.Body.ListingIndex, e.Body.PricePerBundle, e.Body.BundleSize, e.Body.BundleCount)
		if err != nil {
			l.WithError(err).Errorf("Error updating listing in shop [%s].", shopId)
		}
	}
}

func handlePurchaseBundleCommand(db *gorm.DB) message.Handler[merchant2.Command[merchant2.CommandPurchaseBundleBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e merchant2.Command[merchant2.CommandPurchaseBundleBody]) {
		if e.Type != merchant2.CommandPurchaseBundle {
			return
		}
		shopId, err := uuid.Parse(e.Body.ShopId)
		if err != nil {
			l.WithError(err).Errorf("Error parsing shopId [%s].", e.Body.ShopId)
			return
		}

		p := shop.NewProcessor(l, ctx, db)
		_, err = p.PurchaseBundleAndEmit(e.CharacterId, shopId, e.Body.ListingIndex, e.Body.BundleCount, e.WorldId)
		if err != nil {
			kp := producer.ProviderImpl(l)(ctx)
			reason := "unavailable"
			if errors.Is(err, shop.ErrVersionConflict) {
				reason = "version_conflict"
			} else if errors.Is(err, shop.ErrInsufficientBundles) {
				reason = "insufficient_bundles"
			}
			_ = kp(merchant2.EnvStatusEventTopic)(shop.StatusEventPurchaseFailedProvider(e.CharacterId, shopId, reason))
		}
	}
}

func handleEnterShopCommand(db *gorm.DB) message.Handler[merchant2.Command[merchant2.CommandEnterShopBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e merchant2.Command[merchant2.CommandEnterShopBody]) {
		if e.Type != merchant2.CommandEnterShop {
			return
		}
		shopId, err := uuid.Parse(e.Body.ShopId)
		if err != nil {
			l.WithError(err).Errorf("Error parsing shopId [%s].", e.Body.ShopId)
			return
		}

		if err := shop.NewProcessor(l, ctx, db).EnterShopAndEmit(e.CharacterId, shopId); err != nil {
			l.WithError(err).Errorf("Error entering shop [%s].", shopId)
		}
	}
}

func handleExitShopCommand(db *gorm.DB) message.Handler[merchant2.Command[merchant2.CommandExitShopBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e merchant2.Command[merchant2.CommandExitShopBody]) {
		if e.Type != merchant2.CommandExitShop {
			return
		}
		shopId, err := uuid.Parse(e.Body.ShopId)
		if err != nil {
			l.WithError(err).Errorf("Error parsing shopId [%s].", e.Body.ShopId)
			return
		}

		if err := shop.NewProcessor(l, ctx, db).ExitShopAndEmit(e.CharacterId, shopId); err != nil {
			l.WithError(err).Errorf("Error exiting shop [%s].", shopId)
		}
	}
}

func handleSendMessageCommand(db *gorm.DB) message.Handler[merchant2.Command[merchant2.CommandSendMessageBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e merchant2.Command[merchant2.CommandSendMessageBody]) {
		if e.Type != merchant2.CommandSendMessage {
			return
		}
		shopId, err := uuid.Parse(e.Body.ShopId)
		if err != nil {
			l.WithError(err).Errorf("Error parsing shopId [%s].", e.Body.ShopId)
			return
		}

		mp := msg.NewProcessor(l, ctx, db)
		if err := mp.SendMessage(shopId, e.CharacterId, e.Body.Content); err != nil {
			l.WithError(err).Errorf("Error sending message in shop [%s].", shopId)
		}
	}
}

func acceptItemToOwner(l logrus.FieldLogger, ctx context.Context, characterId uint32, li listing.Model) {
	if li.ItemSnapshot() == nil {
		l.Warnf("Listing [%s] has no item snapshot, cannot return to inventory.", li.Id())
		return
	}

	var ad asset.AssetData
	if err := json.Unmarshal(li.ItemSnapshot(), &ad); err != nil {
		l.WithError(err).Errorf("Error unmarshaling item snapshot for listing [%s].", li.Id())
		return
	}

	ad.Quantity = uint32(li.Quantity())

	invType, ok := inventory.TypeFromItemId(item.Id(li.ItemId()))
	if !ok {
		l.Errorf("Unable to determine inventory type for item [%d].", li.ItemId())
		return
	}

	transactionId := uuid.New()
	kp := producer.ProviderImpl(l)(ctx)
	_ = kp(compartment.EnvCommandTopic)(shop.AcceptAssetCommandProvider(transactionId, characterId, byte(invType), li.ItemId(), ad))
	l.Infof("Returning item [%d] (qty %d) to character [%d] inventory.", li.ItemId(), li.Quantity(), characterId)
}

func handleRetrieveFrederickCommand(db *gorm.DB) message.Handler[merchant2.Command[merchant2.CommandRetrieveFrederickBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e merchant2.Command[merchant2.CommandRetrieveFrederickBody]) {
		if e.Type != merchant2.CommandRetrieveFrederick {
			return
		}

		fp := frederick.NewProcessor(l, ctx, db)

		items, err := fp.GetItems(e.CharacterId)
		if err != nil {
			l.WithError(err).Errorf("Error retrieving Frederick items for character [%d].", e.CharacterId)
			return
		}

		mesos, err := fp.GetMesos(e.CharacterId)
		if err != nil {
			l.WithError(err).Errorf("Error retrieving Frederick mesos for character [%d].", e.CharacterId)
			return
		}

		if len(items) == 0 && len(mesos) == 0 {
			l.Debugf("No items or mesos at Frederick for character [%d].", e.CharacterId)
			return
		}

		kp := producer.ProviderImpl(l)(ctx)

		// Transfer items to character's inventory.
		for _, fi := range items {
			if fi.ItemSnapshot() == nil {
				continue
			}

			var ad asset.AssetData
			if err := json.Unmarshal(fi.ItemSnapshot(), &ad); err != nil {
				l.WithError(err).Errorf("Error unmarshaling item snapshot for Frederick item [%s].", fi.Id())
				continue
			}

			ad.Quantity = uint32(fi.Quantity())

			invType, ok := inventory.TypeFromItemId(item.Id(fi.ItemId()))
			if !ok {
				l.Errorf("Unable to determine inventory type for Frederick item [%d].", fi.ItemId())
				continue
			}

			transactionId := uuid.New()
			_ = kp(compartment.EnvCommandTopic)(shop.AcceptAssetCommandProvider(transactionId, e.CharacterId, byte(invType), fi.ItemId(), ad))
		}

		// Transfer mesos to character.
		var totalMesos uint32
		for _, fm := range mesos {
			totalMesos += fm.Amount()
		}
		if totalMesos > 0 {
			transactionId := uuid.New()
			_ = kp(character.EnvCommandTopic)(shop.ChangeMesoCommandProvider(transactionId, e.WorldId, e.CharacterId, 0, "FREDERICK", int32(totalMesos)))
		}

		// Clear Frederick storage and notifications after dispatching transfers.
		if err := fp.ClearItems(e.CharacterId); err != nil {
			l.WithError(err).Errorf("Error clearing Frederick items for character [%d].", e.CharacterId)
		}
		if err := fp.ClearMesos(e.CharacterId); err != nil {
			l.WithError(err).Errorf("Error clearing Frederick mesos for character [%d].", e.CharacterId)
		}
		if err := fp.ClearNotifications(e.CharacterId); err != nil {
			l.WithError(err).Errorf("Error clearing Frederick notifications for character [%d].", e.CharacterId)
		}

		l.Infof("Retrieved %d items and %d meso records from Frederick for character [%d].", len(items), len(mesos), e.CharacterId)
	}
}
