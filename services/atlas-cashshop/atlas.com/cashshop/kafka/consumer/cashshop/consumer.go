package cashshop

import (
	cashshop3 "atlas-cashshop/cashshop"
	"atlas-cashshop/cashshop/inventory/asset"
	"atlas-cashshop/coupon"
	consumer2 "atlas-cashshop/kafka/consumer"
	"atlas-cashshop/kafka/message/cashshop"
	cashshop2 "atlas-cashshop/kafka/producer/cashshop"
	"atlas-cashshop/surprise"
	"context"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("cash_shop_command")(cashshop.EnvCommandTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser, consumer.EnvHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) error {
	return func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) error {
		return func(rf func(topic string, handler handler.Handler) (string, error)) error {
			var t string
			t, _ = topic.EnvProvider(l)(cashshop.EnvCommandTopic)()
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleCommandRequestPurchase(db)))); err != nil {
				return err
			}
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleCommandRequestInventoryIncreaseByType(db)))); err != nil {
				return err
			}
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleCommandRequestInventoryIncreaseByItem(db)))); err != nil {
				return err
			}
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleCommandRequestStorageIncrease(db)))); err != nil {
				return err
			}
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleCommandRequestStorageIncreaseByItem(db)))); err != nil {
				return err
			}
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleCommandRequestCharacterSlotIncreaseByItem(db)))); err != nil {
				return err
			}
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleCommandExpire(db)))); err != nil {
				return err
			}
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleOpenSurprise(db)))); err != nil {
				return err
			}
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleCommandRequestCouponRedemption(db)))); err != nil {
				return err
			}
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleCommandRequestLockerRebate(db)))); err != nil {
				return err
			}
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleCommandRequestGiftPurchase(db)))); err != nil {
				return err
			}
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleCommandRequestPackagePurchase(db)))); err != nil {
				return err
			}
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleCommandRequestRingPurchase(db)))); err != nil {
				return err
			}
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleCommandRequestEquipSlotIncrease(db)))); err != nil {
				return err
			}
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleCommandExtendEquipSlot(db)))); err != nil {
				return err
			}
			return nil
		}
	}
}

func handleCommandRequestPurchase(db *gorm.DB) message.Handler[cashshop.Command[cashshop.RequestPurchaseCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c cashshop.Command[cashshop.RequestPurchaseCommandBody]) {
		if c.Type != cashshop.CommandTypeRequestPurchase {
			return
		}
		_ = cashshop3.NewProcessor(l, ctx, db).PurchaseAndEmit(c.CharacterId, c.Body.Currency, c.Body.SerialNumber, c.Body.TransactionId, c.Body.Operation)
	}
}

func handleCommandRequestInventoryIncreaseByType(db *gorm.DB) message.Handler[cashshop.Command[cashshop.RequestInventoryIncreaseByTypeCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c cashshop.Command[cashshop.RequestInventoryIncreaseByTypeCommandBody]) {
		if c.Type != cashshop.CommandTypeRequestInventoryIncreaseByType {
			return
		}
		_ = cashshop3.NewProcessor(l, ctx, db).PurchaseInventoryIncreaseByTypeAndEmit(c.CharacterId, c.Body.Currency, inventory.Type(c.Body.InventoryType))
	}
}

func handleCommandRequestInventoryIncreaseByItem(db *gorm.DB) message.Handler[cashshop.Command[cashshop.RequestInventoryIncreaseByItemCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c cashshop.Command[cashshop.RequestInventoryIncreaseByItemCommandBody]) {
		if c.Type != cashshop.CommandTypeRequestInventoryIncreaseByItem {
			return
		}
		_ = cashshop3.NewProcessor(l, ctx, db).PurchaseInventoryIncreaseByItemAndEmit(c.CharacterId, c.Body.Currency, c.Body.SerialNumber)
	}
}

func handleCommandRequestStorageIncrease(db *gorm.DB) message.Handler[cashshop.Command[cashshop.RequestStorageIncreaseBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c cashshop.Command[cashshop.RequestStorageIncreaseBody]) {
		if c.Type != cashshop.CommandTypeRequestStorageIncrease {
			return
		}
		_ = producer.ProviderImpl(l)(ctx)(cashshop.EnvEventTopicStatus)(cashshop2.ErrorStatusEventProvider(c.CharacterId, "UNKNOWN_ERROR", uuid.Nil))
	}
}

func handleCommandRequestStorageIncreaseByItem(db *gorm.DB) message.Handler[cashshop.Command[cashshop.RequestCharacterSlotIncreaseByItemCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c cashshop.Command[cashshop.RequestCharacterSlotIncreaseByItemCommandBody]) {
		if c.Type != cashshop.CommandTypeRequestStorageIncreaseByItem {
			return
		}
		_ = producer.ProviderImpl(l)(ctx)(cashshop.EnvEventTopicStatus)(cashshop2.ErrorStatusEventProvider(c.CharacterId, "UNKNOWN_ERROR", uuid.Nil))
	}
}

func handleCommandRequestCharacterSlotIncreaseByItem(db *gorm.DB) message.Handler[cashshop.Command[cashshop.RequestCharacterSlotIncreaseByItemCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c cashshop.Command[cashshop.RequestCharacterSlotIncreaseByItemCommandBody]) {
		if c.Type != cashshop.CommandTypeRequestCharacterSlotIncreaseByItem {
			return
		}
		_ = producer.ProviderImpl(l)(ctx)(cashshop.EnvEventTopicStatus)(cashshop2.ErrorStatusEventProvider(c.CharacterId, "UNKNOWN_ERROR", uuid.Nil))
	}
}

func handleCommandExpire(db *gorm.DB) message.Handler[cashshop.Command[cashshop.ExpireCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c cashshop.Command[cashshop.ExpireCommandBody]) {
		if c.Type != cashshop.CommandTypeExpire {
			return
		}

		l.Debugf("Received EXPIRE command for account [%d], asset [%d], template [%d].",
			c.Body.AccountId, c.Body.AssetId, c.Body.TemplateId)

		err := asset.NewProcessor(l, ctx, db).ExpireAndEmit(
			c.Body.AssetId,
			c.Body.ReplaceItemId,
			c.Body.ReplaceMessage,
		)
		if err != nil {
			l.WithError(err).Errorf("Failed to expire cashshop asset [%d] for account [%d].", c.Body.AssetId, c.Body.AccountId)
		}
	}
}

func handleOpenSurprise(db *gorm.DB) message.Handler[cashshop.Command[cashshop.OpenSurpriseCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c cashshop.Command[cashshop.OpenSurpriseCommandBody]) {
		// COMMAND_TOPIC_CASH_SHOP is shared across handlers: without this
		// guard another command's body unmarshals into OpenSurpriseCommandBody
		// and produces a garbage open request.
		if c.Type != cashshop.CommandTypeOpenSurprise {
			return
		}
		err := surprise.NewProcessor(l, ctx, db).OpenAndEmit(c.Body.TransactionId, c.Body.AccountId, c.CharacterId, c.Body.CashId)
		if err != nil {
			l.WithError(err).Errorf("Unable to open surprise box [%d] for character [%d].", c.Body.CashId, c.CharacterId)
		}
	}
}

func handleCommandRequestCouponRedemption(db *gorm.DB) message.Handler[cashshop.Command[cashshop.RequestCouponRedemptionCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c cashshop.Command[cashshop.RequestCouponRedemptionCommandBody]) {
		if c.Type != cashshop.CommandTypeRequestCouponRedemption {
			return
		}
		// RedeemAndEmit owns the whole outcome, including emitting the failure
		// event on the direct producer path, so a returned error has already
		// been reported to the player and only needs logging here.
		if err := coupon.NewProcessor(l, ctx, db).RedeemAndEmit(c.CharacterId, c.Body.Code); err != nil {
			l.WithError(err).Debugf("Coupon redemption for character [%d] did not succeed.", c.CharacterId)
		}
	}
}

func handleCommandRequestLockerRebate(db *gorm.DB) message.Handler[cashshop.Command[cashshop.RequestLockerRebateCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c cashshop.Command[cashshop.RequestLockerRebateCommandBody]) {
		if c.Type != cashshop.CommandTypeRequestLockerRebate {
			return
		}
		// RebateAndEmit owns the whole outcome, including emitting the
		// LOCKER_REBATED / ERROR event on the appropriate path, so a
		// returned error has already been reported to the player and only
		// needs logging here.
		if err := cashshop3.NewProcessor(l, ctx, db).RebateAndEmit(c.CharacterId, c.Body.AccountId, c.Body.CashId, c.Body.TransactionId); err != nil {
			l.WithError(err).Errorf("Locker rebate for character [%d], cashId [%d] did not succeed.", c.CharacterId, c.Body.CashId)
		}
	}
}

func handleCommandRequestGiftPurchase(db *gorm.DB) message.Handler[cashshop.Command[cashshop.RequestGiftPurchaseCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c cashshop.Command[cashshop.RequestGiftPurchaseCommandBody]) {
		if c.Type != cashshop.CommandTypeRequestGiftPurchase {
			return
		}
		// GiftAndEmit owns the whole outcome, including emitting the
		// GIFT_PURCHASED / ERROR event on the appropriate path, so a
		// returned error has already been reported to the player and only
		// needs logging here.
		if err := cashshop3.NewProcessor(l, ctx, db).GiftAndEmit(c.CharacterId, c.Body.TransactionId, c.Body.SerialNumber, c.Body.RecipientCharacterId, c.Body.SenderName, c.Body.Message); err != nil {
			l.WithError(err).Errorf("Gift purchase for character [%d], recipient [%d] did not succeed.", c.CharacterId, c.Body.RecipientCharacterId)
		}
	}
}

func handleCommandRequestPackagePurchase(db *gorm.DB) message.Handler[cashshop.Command[cashshop.RequestPackagePurchaseCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c cashshop.Command[cashshop.RequestPackagePurchaseCommandBody]) {
		if c.Type != cashshop.CommandTypeRequestPackagePurchase {
			return
		}
		// PurchasePackageAndEmit owns the whole outcome, including emitting
		// the PACKAGE_PURCHASED / ERROR event on the appropriate path, so a
		// returned error has already been reported to the player and only
		// needs logging here.
		if err := cashshop3.NewProcessor(l, ctx, db).PurchasePackageAndEmit(c.CharacterId, c.Body.TransactionId, c.Body.Currency, c.Body.SerialNumber, c.Body.RecipientCharacterId, c.Body.SenderName); err != nil {
			l.WithError(err).Errorf("Package purchase for character [%d], recipient [%d] did not succeed.", c.CharacterId, c.Body.RecipientCharacterId)
		}
	}
}

func handleCommandRequestRingPurchase(db *gorm.DB) message.Handler[cashshop.Command[cashshop.RequestRingPurchaseCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c cashshop.Command[cashshop.RequestRingPurchaseCommandBody]) {
		if c.Type != cashshop.CommandTypeRequestRingPurchase {
			return
		}
		// PurchaseRingAndEmit owns the whole outcome, including emitting the
		// RING_PURCHASED / ERROR event on the appropriate path, so a
		// returned error has already been reported to the player and only
		// needs logging here.
		if err := cashshop3.NewProcessor(l, ctx, db).PurchaseRingAndEmit(c.CharacterId, c.Body.TransactionId, c.Body.Currency, c.Body.SerialNumber, c.Body.PartnerCharacterId, c.Body.SenderName, c.Body.Message, c.Body.RingType); err != nil {
			l.WithError(err).Errorf("Ring purchase for character [%d], partner [%d] did not succeed.", c.CharacterId, c.Body.PartnerCharacterId)
		}
	}
}

func handleCommandRequestEquipSlotIncrease(db *gorm.DB) message.Handler[cashshop.Command[cashshop.RequestEquipSlotIncreaseCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c cashshop.Command[cashshop.RequestEquipSlotIncreaseCommandBody]) {
		if c.Type != cashshop.CommandTypeRequestEquipSlotIncrease {
			return
		}
		// PurchaseEquipSlotAndEmit owns the whole outcome, including
		// emitting the EQUIP_SLOT_INCREASED / ERROR event on the
		// appropriate path, so a returned error has already been reported
		// to the player and only needs logging here.
		if err := cashshop3.NewProcessor(l, ctx, db).PurchaseEquipSlotAndEmit(c.CharacterId, c.Body.Currency, c.Body.SerialNumber, c.Body.TransactionId); err != nil {
			l.WithError(err).Errorf("Equip slot purchase for character [%d] did not succeed.", c.CharacterId)
		}
	}
}

// handleCommandExtendEquipSlot consumes the internal EXTEND_EQUIP_SLOT
// follow-up command PurchaseEquipSlotAndEmit mints via the outbox (task-240
// task 24c): by the time this fires, the purchase's wallet debit and
// purchase record have already durably committed, so the atlas-character
// write only ever happens after the charge is final -- never before.
func handleCommandExtendEquipSlot(db *gorm.DB) message.Handler[cashshop.Command[cashshop.ExtendEquipSlotCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c cashshop.Command[cashshop.ExtendEquipSlotCommandBody]) {
		if c.Type != cashshop.CommandTypeExtendEquipSlot {
			return
		}
		// CompleteEquipSlotExtension owns the whole outcome, including
		// emitting EQUIP_SLOT_INCREASED on success; a failure here means the
		// atlas-character write itself failed and is logged for
		// reconciliation, not retried and not reported to the player as a
		// purchase failure (the charge already stands).
		if err := cashshop3.NewProcessor(l, ctx, db).CompleteEquipSlotExtension(c.CharacterId, c.Body.SlotIndex, c.Body.Days, c.Body.TransactionId); err != nil {
			l.WithError(err).Errorf("Equip slot extension for character [%d] (transaction [%s]) did not succeed.", c.CharacterId, c.Body.TransactionId)
		}
	}
}
