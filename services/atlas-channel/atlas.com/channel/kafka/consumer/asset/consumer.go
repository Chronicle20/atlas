package asset

import (
	"atlas-channel/asset"
	"atlas-channel/character"
	consumer2 "atlas-channel/kafka/consumer"
	asset2 "atlas-channel/kafka/message/asset"
	_map "atlas-channel/map"
	"atlas-channel/messenger"
	"atlas-channel/pet"
	"atlas-channel/server"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"
	"errors"

	"github.com/Chronicle20/atlas-constants/inventory"
	"github.com/Chronicle20/atlas-constants/item"
	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/handler"
	"github.com/Chronicle20/atlas-kafka/message"
	"github.com/Chronicle20/atlas-kafka/topic"
	"github.com/Chronicle20/atlas-model/model"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("asset_status_event")(asset2.EnvEventTopicStatus)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser), consumer.SetStartOffset(kafka.LastOffset))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) {
		return func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) {
			return func(rf func(topic string, handler handler.Handler) (string, error)) {
				var t string
				t, _ = topic.EnvProvider(l)(asset2.EnvEventTopicStatus)()
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleAssetCreatedEvent(sc, wp))))
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleAssetUpdatedEvent(sc, wp))))
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleAssetQuantityUpdatedEvent(sc, wp))))
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleAssetMoveEvent(sc, wp))))
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleAssetDeletedEvent(sc, wp))))
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleAssetAcceptedEvent(sc, wp))))
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleAssetReleasedEvent(sc, wp))))
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleAssetExpiredEvent(sc, wp))))
			}
		}
	}
}

// enrichPetAsset fetches pet data from atlas-pets for pet assets and enriches the model.
func enrichPetAsset(l logrus.FieldLogger, ctx context.Context, a asset.Model) asset.Model {
	if !a.IsPet() || a.PetId() == 0 {
		return a
	}
	pm, err := pet.NewProcessor(l, ctx).GetById(a.PetId())
	if err != nil {
		l.WithError(err).Debugf("Unable to fetch pet [%d] for asset enrichment.", a.PetId())
		return a
	}
	return asset.Clone(a).
		SetPetName(pm.Name()).
		SetPetLevel(pm.Level()).
		SetCloseness(pm.Closeness()).
		SetFullness(pm.Fullness()).
		SetPetSlot(pm.Slot()).
		MustBuild()
}

func buildAssetFromCreatedBody(e asset2.StatusEvent[asset2.CreatedStatusEventBody]) asset.Model {
	return asset.NewModelBuilder(e.AssetId, e.CompartmentId, e.TemplateId).
		SetSlot(e.Slot).
		SetExpiration(e.Body.Expiration).
		SetCreatedAt(e.Body.CreatedAt).
		SetQuantity(e.Body.Quantity).
		SetOwnerId(e.Body.OwnerId).
		SetFlag(e.Body.Flag).
		SetRechargeable(e.Body.Rechargeable).
		SetStrength(e.Body.Strength).
		SetDexterity(e.Body.Dexterity).
		SetIntelligence(e.Body.Intelligence).
		SetLuck(e.Body.Luck).
		SetHp(e.Body.Hp).
		SetMp(e.Body.Mp).
		SetWeaponAttack(e.Body.WeaponAttack).
		SetMagicAttack(e.Body.MagicAttack).
		SetWeaponDefense(e.Body.WeaponDefense).
		SetMagicDefense(e.Body.MagicDefense).
		SetAccuracy(e.Body.Accuracy).
		SetAvoidability(e.Body.Avoidability).
		SetHands(e.Body.Hands).
		SetSpeed(e.Body.Speed).
		SetJump(e.Body.Jump).
		SetSlots(e.Body.Slots).
		SetFlag(e.Body.Flag).
		SetLevelType(e.Body.LevelType).
		SetLevel(e.Body.Level).
		SetExperience(e.Body.Experience).
		SetHammersApplied(e.Body.HammersApplied).
		SetEquippedSince(e.Body.EquippedSince).
		SetCashId(e.Body.CashId).
		SetCommodityId(e.Body.CommodityId).
		SetPurchaseBy(e.Body.PurchaseBy).
		SetPetId(e.Body.PetId).
		MustBuild()
}

func buildAssetFromUpdatedBody(e asset2.StatusEvent[asset2.UpdatedStatusEventBody]) asset.Model {
	return asset.NewModelBuilder(e.AssetId, e.CompartmentId, e.TemplateId).
		SetSlot(e.Slot).
		SetExpiration(e.Body.Expiration).
		SetCreatedAt(e.Body.CreatedAt).
		SetQuantity(e.Body.Quantity).
		SetOwnerId(e.Body.OwnerId).
		SetFlag(e.Body.Flag).
		SetRechargeable(e.Body.Rechargeable).
		SetStrength(e.Body.Strength).
		SetDexterity(e.Body.Dexterity).
		SetIntelligence(e.Body.Intelligence).
		SetLuck(e.Body.Luck).
		SetHp(e.Body.Hp).
		SetMp(e.Body.Mp).
		SetWeaponAttack(e.Body.WeaponAttack).
		SetMagicAttack(e.Body.MagicAttack).
		SetWeaponDefense(e.Body.WeaponDefense).
		SetMagicDefense(e.Body.MagicDefense).
		SetAccuracy(e.Body.Accuracy).
		SetAvoidability(e.Body.Avoidability).
		SetHands(e.Body.Hands).
		SetSpeed(e.Body.Speed).
		SetJump(e.Body.Jump).
		SetSlots(e.Body.Slots).
		SetFlag(e.Body.Flag).
		SetLevelType(e.Body.LevelType).
		SetLevel(e.Body.Level).
		SetExperience(e.Body.Experience).
		SetHammersApplied(e.Body.HammersApplied).
		SetEquippedSince(e.Body.EquippedSince).
		SetCashId(e.Body.CashId).
		SetCommodityId(e.Body.CommodityId).
		SetPurchaseBy(e.Body.PurchaseBy).
		SetPetId(e.Body.PetId).
		MustBuild()
}

func buildAssetFromAcceptedBody(e asset2.StatusEvent[asset2.AcceptedStatusEventBody]) asset.Model {
	return asset.NewModelBuilder(e.AssetId, e.CompartmentId, e.TemplateId).
		SetSlot(e.Slot).
		SetExpiration(e.Body.Expiration).
		SetCreatedAt(e.Body.CreatedAt).
		SetQuantity(e.Body.Quantity).
		SetOwnerId(e.Body.OwnerId).
		SetFlag(e.Body.Flag).
		SetRechargeable(e.Body.Rechargeable).
		SetStrength(e.Body.Strength).
		SetDexterity(e.Body.Dexterity).
		SetIntelligence(e.Body.Intelligence).
		SetLuck(e.Body.Luck).
		SetHp(e.Body.Hp).
		SetMp(e.Body.Mp).
		SetWeaponAttack(e.Body.WeaponAttack).
		SetMagicAttack(e.Body.MagicAttack).
		SetWeaponDefense(e.Body.WeaponDefense).
		SetMagicDefense(e.Body.MagicDefense).
		SetAccuracy(e.Body.Accuracy).
		SetAvoidability(e.Body.Avoidability).
		SetHands(e.Body.Hands).
		SetSpeed(e.Body.Speed).
		SetJump(e.Body.Jump).
		SetSlots(e.Body.Slots).
		SetFlag(e.Body.Flag).
		SetLevelType(e.Body.LevelType).
		SetLevel(e.Body.Level).
		SetExperience(e.Body.Experience).
		SetHammersApplied(e.Body.HammersApplied).
		SetEquippedSince(e.Body.EquippedSince).
		SetCashId(e.Body.CashId).
		SetCommodityId(e.Body.CommodityId).
		SetPurchaseBy(e.Body.PurchaseBy).
		SetPetId(e.Body.PetId).
		MustBuild()
}

func handleAssetCreatedEvent(sc server.Model, wp writer.Producer) message.Handler[asset2.StatusEvent[asset2.CreatedStatusEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e asset2.StatusEvent[asset2.CreatedStatusEventBody]) {
		if e.Type != asset2.StatusEventTypeCreated {
			return
		}

		t := tenant.MustFromContext(ctx)
		if !t.Is(sc.Tenant()) {
			return
		}

		_ = session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(e.CharacterId, func(s session.Model) error {
			inventoryType, ok := inventory.TypeFromItemId(item.Id(e.TemplateId))
			if !ok {
				l.Errorf("Unable to identify inventory type by item [%d].", e.TemplateId)
				return errors.New("unable to identify inventory type")
			}

			a := buildAssetFromCreatedBody(e)
			a = enrichPetAsset(l, ctx, a)
			itemWriter := model.FlipOperator(writer.WriteAssetInfo(t)(true))(a)
			bp := writer.CharacterInventoryChangeBody(false, writer.InventoryAddBodyWriter(inventoryType, e.Slot, itemWriter))
			err := session.Announce(l)(ctx)(wp)(writer.CharacterInventoryChange)(bp)(s)
			if err != nil {
				l.WithError(err).Errorf("Unable to add [%d] to slot [%d] for character [%d].", e.TemplateId, e.Slot, s.CharacterId())
			}
			return err
		})
	}
}

func handleAssetUpdatedEvent(sc server.Model, wp writer.Producer) message.Handler[asset2.StatusEvent[asset2.UpdatedStatusEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e asset2.StatusEvent[asset2.UpdatedStatusEventBody]) {
		if e.Type != asset2.StatusEventTypeUpdated {
			return
		}

		t := tenant.MustFromContext(ctx)
		if !t.Is(sc.Tenant()) {
			return
		}

		_ = session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(e.CharacterId, func(s session.Model) error {
			inventoryType, ok := inventory.TypeFromItemId(item.Id(e.TemplateId))
			if !ok {
				l.Errorf("Unable to identify inventory type by item [%d].", e.TemplateId)
				return errors.New("unable to identify inventory type")
			}

			a := buildAssetFromUpdatedBody(e)
			so := session.Announce(l)(ctx)(wp)(writer.CharacterInventoryChange)(writer.CharacterInventoryRefreshAsset(sc.Tenant())(inventoryType, a))
			err := session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(e.CharacterId, so)
			if err != nil {
				l.WithError(err).Errorf("Unable to update [%d] in slot [%d] for character [%d].", e.TemplateId, e.Slot, e.CharacterId)
			}
			return err
		})
	}
}

func handleAssetQuantityUpdatedEvent(sc server.Model, wp writer.Producer) message.Handler[asset2.StatusEvent[asset2.QuantityChangedEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e asset2.StatusEvent[asset2.QuantityChangedEventBody]) {
		if e.Type != asset2.StatusEventTypeQuantityChanged {
			return
		}

		t := sc.Tenant()
		if !t.Is(tenant.MustFromContext(ctx)) {
			return
		}

		inventoryType, ok := inventory.TypeFromItemId(item.Id(e.TemplateId))
		if !ok {
			l.Errorf("Unable to identify inventory type by item [%d].", e.TemplateId)
			return
		}

		so := session.Announce(l)(ctx)(wp)(writer.CharacterInventoryChange)(writer.CharacterInventoryChangeBody(false, writer.InventoryQuantityUpdateBodyWriter(inventoryType, e.Slot, e.Body.Quantity)))
		err := session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(e.CharacterId, so)
		if err != nil {
			l.WithError(err).Errorf("Unable to update [%d] in slot [%d] for character [%d].", e.TemplateId, e.Slot, e.CharacterId)
		}
	}
}

func handleAssetMoveEvent(sc server.Model, wp writer.Producer) message.Handler[asset2.StatusEvent[asset2.MovedStatusEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e asset2.StatusEvent[asset2.MovedStatusEventBody]) {
		if e.Type != asset2.StatusEventTypeMoved {
			return
		}

		t := sc.Tenant()
		if !t.Is(tenant.MustFromContext(ctx)) {
			return
		}

		_ = session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(e.CharacterId, moveInCompartment(l)(ctx)(wp)(e))
	}
}

func moveInCompartment(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(e asset2.StatusEvent[asset2.MovedStatusEventBody]) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(e asset2.StatusEvent[asset2.MovedStatusEventBody]) model.Operator[session.Model] {
		cp := character.NewProcessor(l, ctx)
		return func(wp writer.Producer) func(e asset2.StatusEvent[asset2.MovedStatusEventBody]) model.Operator[session.Model] {
			return func(e asset2.StatusEvent[asset2.MovedStatusEventBody]) model.Operator[session.Model] {
				return func(s session.Model) error {
					c, err := cp.GetById(cp.InventoryDecorator, cp.PetAssetEnrichmentDecorator, cp.PetModelDecorator)(s.CharacterId())
					if err != nil {
						l.WithError(err).Errorf("Unable to issue appearance update for character [%d] to others in map.", s.CharacterId())
						return err
					}

					errChannels := make(chan error, 3)
					go func() {
						inventoryType, ok := inventory.TypeFromItemId(item.Id(e.TemplateId))
						if !ok {
							l.Errorf("Unable to identify inventory type by item [%d].", e.TemplateId)
							return
						}

						err := session.Announce(l)(ctx)(wp)(writer.CharacterInventoryChange)(writer.CharacterInventoryChangeBody(false, writer.InventoryMoveBodyWriter(inventoryType, e.Slot, e.Body.OldSlot)))(s)
						if err != nil {
							l.WithError(err).Errorf("Unable to move [%d] in slot [%d] to [%d] for character [%d].", e.TemplateId, e.Body.OldSlot, e.Slot, s.CharacterId())
						}
						errChannels <- err
					}()
					go func() {
						errChannels <- _map.NewProcessor(l, ctx).ForSessionsInMap(s.Field(), updateAppearance(l)(ctx)(wp)(c))
					}()
					go func() {
						it, ok := inventory.TypeFromItemId(item.Id(e.TemplateId))
						if !ok || it != inventory.TypeValueEquip {
							return
						}

						if e.Slot > 0 && e.Body.OldSlot > 0 {
							return
						}

						m, err := messenger.NewProcessor(l, ctx).GetByMemberId(e.CharacterId)
						if err != nil {
							return
						}
						um, err := m.FindMember(e.CharacterId)
						if err != nil {
							return
						}

						for _, mm := range m.Members() {
							_ = session.NewProcessor(l, ctx).IfPresentByCharacterId(s.Field().Channel())(mm.Id(), func(os session.Model) error {
								return session.Announce(l)(ctx)(wp)(writer.MessengerOperation)(writer.MessengerOperationUpdateBody(l, ctx)(um.Slot(), c, s.ChannelId()))(os)
							})
						}
						errChannels <- err
					}()

					for i := 0; i < 3; i++ {
						select {
						case <-errChannels:
							err = <-errChannels
						}
					}
					return err
				}
			}
		}
	}
}

func updateAppearance(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(c character.Model) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(c character.Model) model.Operator[session.Model] {
		return func(wp writer.Producer) func(c character.Model) model.Operator[session.Model] {
			return func(c character.Model) model.Operator[session.Model] {
				return session.Announce(l)(ctx)(wp)(writer.CharacterAppearanceUpdate)(writer.CharacterAppearanceUpdateBody(tenant.MustFromContext(ctx))(c))
			}
		}
	}
}

func handleAssetDeletedEvent(sc server.Model, wp writer.Producer) message.Handler[asset2.StatusEvent[asset2.DeletedStatusEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e asset2.StatusEvent[asset2.DeletedStatusEventBody]) {
		if e.Type != asset2.StatusEventTypeDeleted {
			return
		}

		t := sc.Tenant()
		if !t.Is(tenant.MustFromContext(ctx)) {
			return
		}

		inventoryType, ok := inventory.TypeFromItemId(item.Id(e.TemplateId))
		if !ok {
			l.Errorf("Unable to identify inventory type by item [%d].", e.TemplateId)
			return
		}

		af := session.Announce(l)(ctx)(wp)(writer.CharacterInventoryChange)(writer.CharacterInventoryChangeBody(false, writer.InventoryRemoveBodyWriter(inventoryType, e.Slot)))
		err := session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(e.CharacterId, af)
		if err != nil {
			l.WithError(err).Errorf("Unable to remove [%d] in slot [%d] for character [%d].", e.TemplateId, e.Slot, e.CharacterId)
		}
	}
}

// handleAssetAcceptedEvent handles ACCEPTED events when an asset is accepted into inventory (e.g., from storage)
func handleAssetAcceptedEvent(sc server.Model, wp writer.Producer) message.Handler[asset2.StatusEvent[asset2.AcceptedStatusEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e asset2.StatusEvent[asset2.AcceptedStatusEventBody]) {
		if e.Type != asset2.StatusEventTypeAccepted {
			return
		}

		t := tenant.MustFromContext(ctx)
		if !t.Is(sc.Tenant()) {
			return
		}

		_ = session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(e.CharacterId, func(s session.Model) error {
			inventoryType, ok := inventory.TypeFromItemId(item.Id(e.TemplateId))
			if !ok {
				l.Errorf("Unable to identify inventory type by item [%d].", e.TemplateId)
				return errors.New("unable to identify inventory type")
			}

			a := buildAssetFromAcceptedBody(e)
			a = enrichPetAsset(l, ctx, a)
			itemWriter := model.FlipOperator(writer.WriteAssetInfo(t)(true))(a)
			bp := writer.CharacterInventoryChangeBody(false, writer.InventoryAddBodyWriter(inventoryType, e.Slot, itemWriter))
			err := session.Announce(l)(ctx)(wp)(writer.CharacterInventoryChange)(bp)(s)
			if err != nil {
				l.WithError(err).Errorf("Unable to add accepted asset [%d] to slot [%d] for character [%d].", e.TemplateId, e.Slot, s.CharacterId())
				return err
			}
			return nil
		})
	}
}

// handleAssetReleasedEvent handles RELEASED events when an asset is released from inventory (e.g., to storage)
func handleAssetReleasedEvent(sc server.Model, wp writer.Producer) message.Handler[asset2.StatusEvent[asset2.ReleasedStatusEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e asset2.StatusEvent[asset2.ReleasedStatusEventBody]) {
		if e.Type != asset2.StatusEventTypeReleased {
			return
		}

		t := sc.Tenant()
		if !t.Is(tenant.MustFromContext(ctx)) {
			return
		}

		inventoryType, ok := inventory.TypeFromItemId(item.Id(e.TemplateId))
		if !ok {
			l.Errorf("Unable to identify inventory type by item [%d].", e.TemplateId)
			return
		}

		af := session.Announce(l)(ctx)(wp)(writer.CharacterInventoryChange)(writer.CharacterInventoryChangeBody(false, writer.InventoryRemoveBodyWriter(inventoryType, e.Slot)))
		err := session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(e.CharacterId, af)
		if err != nil {
			l.WithError(err).Errorf("Unable to remove released asset [%d] in slot [%d] for character [%d].", e.TemplateId, e.Slot, e.CharacterId)
		}
	}
}

// handleAssetExpiredEvent handles EXPIRED events when an asset has expired
func handleAssetExpiredEvent(sc server.Model, wp writer.Producer) message.Handler[asset2.StatusEvent[asset2.ExpiredStatusEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e asset2.StatusEvent[asset2.ExpiredStatusEventBody]) {
		if e.Type != asset2.StatusEventTypeExpired {
			return
		}

		t := sc.Tenant()
		if !t.Is(tenant.MustFromContext(ctx)) {
			return
		}

		_ = session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(e.CharacterId, func(s session.Model) error {
			// Send appropriate expiration notification based on item type
			if e.Body.ReplaceMessage != "" {
				// Item was replaced - send replacement message
				err := session.Announce(l)(ctx)(wp)(writer.CharacterStatusMessage)(
					writer.CharacterStatusMessageOperationItemExpireReplaceBody(l)([]string{e.Body.ReplaceMessage}),
				)(s)
				if err != nil {
					l.WithError(err).Errorf("Unable to send item expire replace message for character [%d].", e.CharacterId)
					return err
				}
			} else if e.Body.IsCash {
				// Cash item expired
				err := session.Announce(l)(ctx)(wp)(writer.CharacterStatusMessage)(
					writer.CharacterStatusMessageOperationCashItemExpireBody(l)(e.TemplateId),
				)(s)
				if err != nil {
					l.WithError(err).Errorf("Unable to send cash item expire message for character [%d].", e.CharacterId)
					return err
				}
			} else {
				// Regular item expired
				err := session.Announce(l)(ctx)(wp)(writer.CharacterStatusMessage)(
					writer.CharacterStatusMessageOperationGeneralItemExpireBody(l)([]uint32{e.TemplateId}),
				)(s)
				if err != nil {
					l.WithError(err).Errorf("Unable to send general item expire message for character [%d].", e.CharacterId)
					return err
				}
			}
			return nil
		})
	}
}
