package merchant

import (
	consumer2 "atlas-merchant/kafka/consumer"
	merchant2 "atlas-merchant/kafka/message/merchant"
	"atlas-merchant/searchcount"
	"atlas-merchant/shop"
	"context"
	"errors"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
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
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleRecordItemSearchCommand(db))))
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleWithdrawMesoCommand(db))))
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleOrganizeListingsCommand(db))))
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleAddBlacklistCommand(db))))
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleRemoveBlacklistCommand(db))))
		}
	}
}

func handlePlaceShopCommand(db *gorm.DB) message.Handler[merchant2.Command[merchant2.CommandPlaceShopBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e merchant2.Command[merchant2.CommandPlaceShopBody]) {
		if e.Type != merchant2.CommandPlaceShop {
			return
		}
		p := shop.NewProcessor(l, ctx, db)
		_, err := p.CreateShopAndEmit(e.CharacterId, shop.ShopType(e.Body.ShopType), e.Body.Title, e.WorldId, e.ChannelId, e.Body.MapId, e.Body.InstanceId, e.Body.X, e.Body.Y, e.Body.PermitItemId)
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

		p := shop.NewProcessor(l, ctx, db)
		_, err = p.AddListingAndEmit(shopId, e.CharacterId, e.Body.ItemId, e.Body.ItemType, e.Body.BundleSize, e.Body.BundleCount, e.Body.PricePerBundle, e.Body.ItemSnapshot, e.Body.InventoryType, e.Body.AssetId)
		if err != nil {
			l.WithError(err).Errorf("Error adding listing to shop [%s].", shopId)
		}
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

		if _, err := shop.NewProcessor(l, ctx, db).RemoveListingAndEmit(shopId, e.CharacterId, e.Body.ListingIndex); err != nil {
			l.WithError(err).Errorf("Error removing listing from shop [%s].", shopId)
		}
	}
}

func handleWithdrawMesoCommand(db *gorm.DB) message.Handler[merchant2.Command[merchant2.CommandWithdrawMesoBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e merchant2.Command[merchant2.CommandWithdrawMesoBody]) {
		if e.Type != merchant2.CommandWithdrawMeso {
			return
		}
		shopId, err := uuid.Parse(e.Body.ShopId)
		if err != nil {
			l.WithError(err).Errorf("Error parsing shopId [%s].", e.Body.ShopId)
			return
		}
		if err := shop.NewProcessor(l, ctx, db).WithdrawMesoAndEmit(shopId, e.CharacterId); err != nil {
			l.WithError(err).Errorf("Error withdrawing meso for shop [%s].", shopId)
		}
	}
}

func handleOrganizeListingsCommand(db *gorm.DB) message.Handler[merchant2.Command[merchant2.CommandOrganizeListingsBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e merchant2.Command[merchant2.CommandOrganizeListingsBody]) {
		if e.Type != merchant2.CommandOrganizeListings {
			return
		}
		shopId, err := uuid.Parse(e.Body.ShopId)
		if err != nil {
			l.WithError(err).Errorf("Error parsing shopId [%s].", e.Body.ShopId)
			return
		}
		if err := shop.NewProcessor(l, ctx, db).OrganizeListingsAndEmit(shopId, e.CharacterId); err != nil {
			l.WithError(err).Errorf("Error organizing listings for shop [%s].", shopId)
		}
	}
}

func handleAddBlacklistCommand(db *gorm.DB) message.Handler[merchant2.Command[merchant2.CommandBlacklistBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e merchant2.Command[merchant2.CommandBlacklistBody]) {
		if e.Type != merchant2.CommandAddBlacklist {
			return
		}
		shopId, err := uuid.Parse(e.Body.ShopId)
		if err != nil {
			l.WithError(err).Errorf("Error parsing shopId [%s].", e.Body.ShopId)
			return
		}
		if err := shop.NewProcessor(l, ctx, db).AddToBlacklistAndEmit(shopId, e.CharacterId, e.Body.Name, e.Body.BannedCharacterId); err != nil {
			l.WithError(err).Errorf("Error adding [%s] to blacklist for shop [%s].", e.Body.Name, shopId)
		}
	}
}

func handleRemoveBlacklistCommand(db *gorm.DB) message.Handler[merchant2.Command[merchant2.CommandBlacklistBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e merchant2.Command[merchant2.CommandBlacklistBody]) {
		if e.Type != merchant2.CommandRemoveBlacklist {
			return
		}
		shopId, err := uuid.Parse(e.Body.ShopId)
		if err != nil {
			l.WithError(err).Errorf("Error parsing shopId [%s].", e.Body.ShopId)
			return
		}
		if err := shop.NewProcessor(l, ctx, db).RemoveFromBlacklistAndEmit(shopId, e.CharacterId, e.Body.Name); err != nil {
			l.WithError(err).Errorf("Error removing [%s] from blacklist for shop [%s].", e.Body.Name, shopId)
		}
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

		if err := shop.NewProcessor(l, ctx, db).UpdateListing(shopId, e.Body.ListingIndex, e.Body.PricePerBundle, e.Body.BundleSize, e.Body.BundleCount); err != nil {
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

		if err := shop.NewProcessor(l, ctx, db).EnterShopAndEmit(e.CharacterId, shopId, e.Body.VisitorName); err != nil {
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

		if err := shop.NewProcessor(l, ctx, db).SendMessageAndEmit(shopId, e.CharacterId, e.Body.Content); err != nil {
			l.WithError(err).Errorf("Error sending message in shop [%s].", shopId)
		}
	}
}

func handleRetrieveFrederickCommand(db *gorm.DB) message.Handler[merchant2.Command[merchant2.CommandRetrieveFrederickBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e merchant2.Command[merchant2.CommandRetrieveFrederickBody]) {
		if e.Type != merchant2.CommandRetrieveFrederick {
			return
		}

		if err := shop.NewProcessor(l, ctx, db).RetrieveFrederickAndEmit(e.CharacterId, e.WorldId); err != nil {
			l.WithError(err).Errorf("Error retrieving Frederick items for character [%d].", e.CharacterId)
		}
	}
}

func handleRecordItemSearchCommand(db *gorm.DB) message.Handler[merchant2.Command[merchant2.CommandRecordItemSearchBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e merchant2.Command[merchant2.CommandRecordItemSearchBody]) {
		if e.Type != merchant2.CommandRecordItemSearch {
			return
		}
		if err := searchcount.NewProcessor(l, ctx, db).RecordSearch(e.WorldId, e.Body.ItemId); err != nil {
			l.WithError(err).Errorf("Error recording item search for item [%d] in world [%d].", e.Body.ItemId, e.WorldId)
		}
	}
}
