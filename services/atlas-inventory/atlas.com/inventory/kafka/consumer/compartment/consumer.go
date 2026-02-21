package compartment

import (
	"atlas-inventory/asset"
	"atlas-inventory/compartment"
	consumer2 "atlas-inventory/kafka/consumer"
	compartment2 "atlas-inventory/kafka/message/compartment"
	"context"
	"math"

	"github.com/Chronicle20/atlas-constants/field"
	"github.com/Chronicle20/atlas-constants/inventory"
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
			rf(consumer2.NewConfig(l)("compartment_command")(compartment2.EnvCommandTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) error {
	return func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) error {
		return func(rf func(topic string, handler handler.Handler) (string, error)) error {
			var t string
			t, _ = topic.EnvProvider(l)(compartment2.EnvCommandTopic)()
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleEquipItemCommand(db)))); err != nil {
				return err
			}
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleUnequipItemCommand(db)))); err != nil {
				return err
			}
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleMoveItemCommand(db)))); err != nil {
				return err
			}
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleDropItemCommand(db)))); err != nil {
				return err
			}
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleRequestReserveItemCommand(db)))); err != nil {
				return err
			}
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleConsumeItemCommand(db)))); err != nil {
				return err
			}
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleDestroyItemCommand(db)))); err != nil {
				return err
			}
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleCancelItemReservationCommand(db)))); err != nil {
				return err
			}
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleIncreaseCapacityCommand(db)))); err != nil {
				return err
			}
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleCreateAssetCommand(db)))); err != nil {
				return err
			}
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleRechargeItemCommand(db)))); err != nil {
				return err
			}
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleMergeCommand(db)))); err != nil {
				return err
			}
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleSortCommand(db)))); err != nil {
				return err
			}
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleAcceptCommand(db)))); err != nil {
				return err
			}
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleReleaseCommand(db)))); err != nil {
				return err
			}
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleExpireCommand(db)))); err != nil {
				return err
			}
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleModifyEquipmentCommand(db)))); err != nil {
				return err
			}
			return nil
		}
	}
}

func handleEquipItemCommand(db *gorm.DB) message.Handler[compartment2.Command[compartment2.EquipCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c compartment2.Command[compartment2.EquipCommandBody]) {
		if c.Type != compartment2.CommandEquip {
			return
		}
		_ = compartment.NewProcessor(l, ctx, db).EquipItemAndEmit(c.TransactionId, c.CharacterId, c.Body.Source, c.Body.Destination)
	}
}

func handleUnequipItemCommand(db *gorm.DB) message.Handler[compartment2.Command[compartment2.UnequipCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c compartment2.Command[compartment2.UnequipCommandBody]) {
		if c.Type != compartment2.CommandUnequip {
			return
		}
		_ = compartment.NewProcessor(l, ctx, db).RemoveEquipAndEmit(c.TransactionId, c.CharacterId, c.Body.Source, c.Body.Destination)
	}
}

func handleMoveItemCommand(db *gorm.DB) message.Handler[compartment2.Command[compartment2.MoveCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c compartment2.Command[compartment2.MoveCommandBody]) {
		if c.Type != compartment2.CommandMove {
			return
		}
		_ = compartment.NewProcessor(l, ctx, db).MoveAndEmit(c.TransactionId, c.CharacterId, inventory.Type(c.InventoryType), c.Body.Source, c.Body.Destination)
	}
}

func handleIncreaseCapacityCommand(db *gorm.DB) message.Handler[compartment2.Command[compartment2.IncreaseCapacityCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c compartment2.Command[compartment2.IncreaseCapacityCommandBody]) {
		if c.Type != compartment2.CommandIncreaseCapacity {
			return
		}
		_ = compartment.NewProcessor(l, ctx, db).IncreaseCapacityAndEmit(c.TransactionId, c.CharacterId, inventory.Type(c.InventoryType), c.Body.Amount)
	}
}

func handleDropItemCommand(db *gorm.DB) message.Handler[compartment2.Command[compartment2.DropCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c compartment2.Command[compartment2.DropCommandBody]) {
		if c.Type != compartment2.CommandDrop {
			return
		}

		f := field.NewBuilder(c.Body.WorldId, c.Body.ChannelId, c.Body.MapId).SetInstance(c.Body.Instance).Build()
		_ = compartment.NewProcessor(l, ctx, db).DropAndEmit(c.TransactionId, c.CharacterId, inventory.Type(c.InventoryType), f, c.Body.X, c.Body.Y, c.Body.Source, c.Body.Quantity)
	}
}

func handleRequestReserveItemCommand(db *gorm.DB) message.Handler[compartment2.Command[compartment2.RequestReserveCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c compartment2.Command[compartment2.RequestReserveCommandBody]) {
		if c.Type != compartment2.CommandRequestReserve {
			return
		}
		reserves := make([]compartment.ReservationRequest, 0)
		for _, i := range c.Body.Items {
			reserves = append(reserves, compartment.ReservationRequest{
				Slot:     i.Source,
				ItemId:   i.ItemId,
				Quantity: i.Quantity,
			})
		}

		// TODO producers of this command need to be updated to use main TransactionId and not Body.TransactionId
		transactionId := c.TransactionId
		if transactionId == uuid.Nil {
			transactionId = c.Body.TransactionId
		}
		_ = compartment.NewProcessor(l, ctx, db).RequestReserveAndEmit(transactionId, c.CharacterId, inventory.Type(c.InventoryType), reserves)
	}
}

func handleCancelItemReservationCommand(db *gorm.DB) message.Handler[compartment2.Command[compartment2.CancelReservationCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c compartment2.Command[compartment2.CancelReservationCommandBody]) {
		if c.Type != compartment2.CommandCancelReservation {
			return
		}

		// TODO producers of this command need to be updated to use main TransactionId and not Body.TransactionId
		transactionId := c.TransactionId
		if transactionId == uuid.Nil {
			transactionId = c.Body.TransactionId
		}
		_ = compartment.NewProcessor(l, ctx, db).CancelReservationAndEmit(transactionId, c.CharacterId, inventory.Type(c.InventoryType), c.Body.Slot)
	}
}

func handleConsumeItemCommand(db *gorm.DB) message.Handler[compartment2.Command[compartment2.ConsumeCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c compartment2.Command[compartment2.ConsumeCommandBody]) {
		if c.Type != compartment2.CommandConsume {
			return
		}

		// TODO producers of this command need to be updated to use main TransactionId and not Body.TransactionId
		transactionId := c.TransactionId
		if transactionId == uuid.Nil {
			transactionId = c.Body.TransactionId
		}
		_ = compartment.NewProcessor(l, ctx, db).ConsumeAssetAndEmit(transactionId, c.CharacterId, inventory.Type(c.InventoryType), c.Body.Slot)
	}
}

func handleDestroyItemCommand(db *gorm.DB) message.Handler[compartment2.Command[compartment2.DestroyCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c compartment2.Command[compartment2.DestroyCommandBody]) {
		if c.Type != compartment2.CommandDestroy {
			return
		}
		quantity := c.Body.Quantity
		// If RemoveAll is true, remove all instances (use MaxInt32)
		// Otherwise, if quantity is 0, also remove all (backward compatibility)
		if c.Body.RemoveAll || quantity == 0 {
			quantity = math.MaxInt32
		}
		_ = compartment.NewProcessor(l, ctx, db).DestroyAssetAndEmit(c.TransactionId, c.CharacterId, inventory.Type(c.InventoryType), c.Body.Slot, quantity)
	}
}

func handleCreateAssetCommand(db *gorm.DB) message.Handler[compartment2.Command[compartment2.CreateAssetCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c compartment2.Command[compartment2.CreateAssetCommandBody]) {
		if c.Type != compartment2.CommandCreateAsset {
			return
		}
		_ = compartment.NewProcessor(l, ctx, db).CreateAssetAndEmit(c.TransactionId, c.CharacterId, inventory.Type(c.InventoryType), c.Body.TemplateId, c.Body.Quantity, c.Body.Expiration, c.Body.OwnerId, c.Body.Flag, c.Body.Rechargeable)
	}
}

func handleRechargeItemCommand(db *gorm.DB) message.Handler[compartment2.Command[compartment2.RechargeCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c compartment2.Command[compartment2.RechargeCommandBody]) {
		if c.Type != compartment2.CommandRecharge {
			return
		}
		_ = compartment.NewProcessor(l, ctx, db).RechargeAssetAndEmit(c.TransactionId, c.CharacterId, inventory.Type(c.InventoryType), c.Body.Slot, c.Body.Quantity)
	}
}

func handleMergeCommand(db *gorm.DB) message.Handler[compartment2.Command[compartment2.MergeCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c compartment2.Command[compartment2.MergeCommandBody]) {
		if c.Type != compartment2.CommandMerge {
			return
		}
		_ = compartment.NewProcessor(l, ctx, db).MergeAndCompactAndEmit(c.TransactionId, c.CharacterId, inventory.Type(c.InventoryType))
	}
}

func handleSortCommand(db *gorm.DB) message.Handler[compartment2.Command[compartment2.SortCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c compartment2.Command[compartment2.SortCommandBody]) {
		if c.Type != compartment2.CommandSort {
			return
		}
		_ = compartment.NewProcessor(l, ctx, db).CompactAndSortAndEmit(c.TransactionId, c.CharacterId, inventory.Type(c.InventoryType))
	}
}

func handleAcceptCommand(db *gorm.DB) message.Handler[compartment2.Command[compartment2.AcceptCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c compartment2.Command[compartment2.AcceptCommandBody]) {
		if c.Type != compartment2.CommandAccept {
			return
		}

		// TODO producers of this command need to be updated to use main TransactionId and not Body.TransactionId
		transactionId := c.TransactionId
		if transactionId == uuid.Nil {
			transactionId = c.Body.TransactionId
		}
		m := asset.NewBuilder(uuid.Nil, c.Body.TemplateId).
			SetExpiration(c.Body.Expiration).
			SetQuantity(c.Body.Quantity).
			SetOwnerId(c.Body.OwnerId).
			SetFlag(c.Body.Flag).
			SetRechargeable(c.Body.Rechargeable).
			SetStrength(c.Body.Strength).
			SetDexterity(c.Body.Dexterity).
			SetIntelligence(c.Body.Intelligence).
			SetLuck(c.Body.Luck).
			SetHp(c.Body.Hp).
			SetMp(c.Body.Mp).
			SetWeaponAttack(c.Body.WeaponAttack).
			SetMagicAttack(c.Body.MagicAttack).
			SetWeaponDefense(c.Body.WeaponDefense).
			SetMagicDefense(c.Body.MagicDefense).
			SetAccuracy(c.Body.Accuracy).
			SetAvoidability(c.Body.Avoidability).
			SetHands(c.Body.Hands).
			SetSpeed(c.Body.Speed).
			SetJump(c.Body.Jump).
			SetSlots(c.Body.Slots).
			SetFlag(c.Body.Flag).
			SetLevelType(c.Body.LevelType).
			SetLevel(c.Body.Level).
			SetExperience(c.Body.Experience).
			SetHammersApplied(c.Body.HammersApplied).
			SetEquippedSince(c.Body.EquippedSince).
			SetCashId(c.Body.CashId).
			SetCommodityId(c.Body.CommodityId).
			SetPurchaseBy(c.Body.PurchaseBy).
			SetPetId(c.Body.PetId).
			Build()
		_ = compartment.NewProcessor(l, ctx, db).AcceptAndEmit(transactionId, c.CharacterId, inventory.Type(c.InventoryType), m)
	}
}

func handleReleaseCommand(db *gorm.DB) message.Handler[compartment2.Command[compartment2.ReleaseCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c compartment2.Command[compartment2.ReleaseCommandBody]) {
		if c.Type != compartment2.CommandRelease {
			return
		}

		// TODO producers of this command need to be updated to use main TransactionId and not Body.TransactionId
		transactionId := c.TransactionId
		if transactionId == uuid.Nil {
			transactionId = c.Body.TransactionId
		}
		_ = compartment.NewProcessor(l, ctx, db).ReleaseAndEmit(transactionId, c.CharacterId, inventory.Type(c.InventoryType), c.Body.AssetId, c.Body.Quantity)
	}
}

func handleExpireCommand(db *gorm.DB) message.Handler[compartment2.Command[compartment2.ExpireCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c compartment2.Command[compartment2.ExpireCommandBody]) {
		if c.Type != compartment2.CommandExpire {
			return
		}

		l.Debugf("Received EXPIRE command for character [%d], asset [%d], template [%d], slot [%d].",
			c.CharacterId, c.Body.AssetId, c.Body.TemplateId, c.Body.Slot)

		// Determine if this is a cash item based on inventory type
		isCash := inventory.Type(c.InventoryType) == inventory.TypeValueCash

		err := compartment.NewProcessor(l, ctx, db).ExpireAssetAndEmit(
			c.TransactionId,
			c.CharacterId,
			inventory.Type(c.InventoryType),
			c.Body.Slot,
			isCash,
			c.Body.ReplaceItemId,
			c.Body.ReplaceMessage,
		)
		if err != nil {
			l.WithError(err).Errorf("Failed to expire asset [%d] for character [%d].", c.Body.AssetId, c.CharacterId)
		}
	}
}

func handleModifyEquipmentCommand(db *gorm.DB) message.Handler[compartment2.Command[compartment2.ModifyEquipmentCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c compartment2.Command[compartment2.ModifyEquipmentCommandBody]) {
		if c.Type != compartment2.CommandModifyEquipment {
			return
		}

		stats := asset.NewBuilder(uuid.Nil, 0).
			SetStrength(c.Body.Strength).
			SetDexterity(c.Body.Dexterity).
			SetIntelligence(c.Body.Intelligence).
			SetLuck(c.Body.Luck).
			SetHp(c.Body.Hp).
			SetMp(c.Body.Mp).
			SetWeaponAttack(c.Body.WeaponAttack).
			SetMagicAttack(c.Body.MagicAttack).
			SetWeaponDefense(c.Body.WeaponDefense).
			SetMagicDefense(c.Body.MagicDefense).
			SetAccuracy(c.Body.Accuracy).
			SetAvoidability(c.Body.Avoidability).
			SetHands(c.Body.Hands).
			SetSpeed(c.Body.Speed).
			SetJump(c.Body.Jump).
			SetSlots(c.Body.Slots).
			SetFlag(c.Body.Flag).
			SetLevelType(c.Body.LevelType).
			SetLevel(c.Body.Level).
			SetExperience(c.Body.Experience).
			SetHammersApplied(c.Body.HammersApplied).
			SetExpiration(c.Body.Expiration).
			Build()
		_ = compartment.NewProcessor(l, ctx, db).ModifyEquipmentAndEmit(c.TransactionId, c.CharacterId, c.Body.AssetId, stats)
	}
}
