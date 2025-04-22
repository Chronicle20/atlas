package inventory

import (
	"atlas-channel/character"
	"atlas-channel/inventory/compartment/asset"
	consumer2 "atlas-channel/kafka/consumer"
	inventory2 "atlas-channel/kafka/message/inventory"
	_map "atlas-channel/map"
	"atlas-channel/messenger"
	"atlas-channel/server"
	"atlas-channel/session"
	model2 "atlas-channel/socket/model"
	"atlas-channel/socket/writer"
	"context"
	"errors"
	"github.com/Chronicle20/atlas-constants/inventory"
	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/handler"
	"github.com/Chronicle20/atlas-kafka/message"
	"github.com/Chronicle20/atlas-kafka/topic"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("character_inventory_changed_event")(inventory2.EnvEventInventoryChanged)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser), consumer.SetStartOffset(kafka.LastOffset))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) {
		return func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) {
			return func(rf func(topic string, handler handler.Handler) (string, error)) {
				var t string
				t, _ = topic.EnvProvider(l)(inventory2.EnvEventInventoryChanged)()
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleInventoryAddEvent(sc, wp))))
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleInventoryQuantityUpdateEvent(sc, wp))))
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleInventoryAttributeUpdateEvent(sc, wp))))
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleInventoryMoveEvent(sc, wp))))
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleInventoryRemoveEvent(sc, wp))))
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleInventoryItemReservationCancelledEvent(sc, wp))))
			}
		}
	}
}

func handleInventoryAddEvent(sc server.Model, wp writer.Producer) message.Handler[inventory2.InventoryChangedEvent[inventory2.InventoryChangedItemAddBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e inventory2.InventoryChangedEvent[inventory2.InventoryChangedItemAddBody]) {
		t := sc.Tenant()
		if !t.Is(tenant.MustFromContext(ctx)) {
			return
		}

		if e.Type != inventory2.ChangedTypeAdd {
			return
		}

		_ = session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.WorldId(), sc.ChannelId())(e.CharacterId, addToInventory(l)(ctx)(wp)(e))
	}
}

func addToInventory(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(event inventory2.InventoryChangedEvent[inventory2.InventoryChangedItemAddBody]) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(event inventory2.InventoryChangedEvent[inventory2.InventoryChangedItemAddBody]) model.Operator[session.Model] {
		t := tenant.MustFromContext(ctx)
		cp := character.NewProcessor(l, ctx)
		return func(wp writer.Producer) func(event inventory2.InventoryChangedEvent[inventory2.InventoryChangedItemAddBody]) model.Operator[session.Model] {
			return func(event inventory2.InventoryChangedEvent[inventory2.InventoryChangedItemAddBody]) model.Operator[session.Model] {
				return func(s session.Model) error {
					var itemWriter model.Operator[*response.Writer]
					inventoryType, ok := inventory.TypeFromItemId(event.Body.ItemId)
					if !ok {
						l.Errorf("Unable to identify inventory type by item [%d].", event.Body.ItemId)
						return errors.New("unable to identify inventory type")
					}

					i, err := cp.GetItemInSlot(s.CharacterId(), inventoryType, event.Slot)()
					if err != nil {
						return err
					}
					itemWriter = model.FlipOperator(writer.WriteAssetInfo(t)(true))(i)

					bp := writer.CharacterInventoryChangeBody(false, writer.InventoryAddBodyWriter(inventoryType, event.Slot, itemWriter))
					err = session.Announce(l)(ctx)(wp)(writer.CharacterInventoryChange)(bp)(s)
					if err != nil {
						l.WithError(err).Errorf("Unable to add [%d] to slot [%d] for character [%d].", event.Body.ItemId, event.Slot, s.CharacterId())
					}
					return err
				}
			}
		}
	}
}

func handleInventoryQuantityUpdateEvent(sc server.Model, wp writer.Producer) message.Handler[inventory2.InventoryChangedEvent[inventory2.InventoryChangedItemQuantityUpdateBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e inventory2.InventoryChangedEvent[inventory2.InventoryChangedItemQuantityUpdateBody]) {
		if e.Type != inventory2.ChangedTypeUpdateQuantity {
			return
		}

		t := sc.Tenant()
		if !t.Is(tenant.MustFromContext(ctx)) {
			return
		}

		inventoryType, ok := inventory.TypeFromItemId(e.Body.ItemId)
		if !ok {
			l.Errorf("Unable to identify inventory type by item [%d].", e.Body.ItemId)
			return
		}

		so := session.Announce(l)(ctx)(wp)(writer.CharacterInventoryChange)(writer.CharacterInventoryChangeBody(false, writer.InventoryQuantityUpdateBodyWriter(inventoryType, e.Slot, e.Body.Quantity)))
		err := session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.WorldId(), sc.ChannelId())(e.CharacterId, so)
		if err != nil {
			l.WithError(err).Errorf("Unable to update [%d] in slot [%d] for character [%d].", e.Body.ItemId, e.Slot, e.CharacterId)
		}
	}
}

func handleInventoryAttributeUpdateEvent(sc server.Model, wp writer.Producer) message.Handler[inventory2.InventoryChangedEvent[inventory2.InventoryChangedItemAttributeUpdateBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e inventory2.InventoryChangedEvent[inventory2.InventoryChangedItemAttributeUpdateBody]) {
		if e.Type != inventory2.ChangedTypeUpdateAttribute {
			return
		}

		t := sc.Tenant()
		if !t.Is(tenant.MustFromContext(ctx)) {
			return
		}

		eqp := asset.NewBuilder[asset.EquipableReferenceData](0, e.Body.ItemId, 0, asset.ReferenceTypeEquipable).
			SetSlot(e.Slot).
			SetExpiration(e.Body.Expiration).
			SetReferenceData(asset.NewEquipableReferenceDataBuilder().
				SetStrength(e.Body.Strength).
				SetDexterity(e.Body.Dexterity).
				SetIntelligence(e.Body.Intelligence).
				SetLuck(e.Body.Luck).
				SetHp(e.Body.HP).
				SetMp(e.Body.MP).
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
				SetOwnerId(0).
				SetLocked(e.Body.Locked).
				SetSpikes(e.Body.Spikes).
				SetKarmaUsed(e.Body.KarmaUsed).
				SetCold(e.Body.Cold).
				SetCanBeTraded(e.Body.CanBeTraded).
				SetLevelType(e.Body.LevelType).
				SetLevel(e.Body.Level).
				SetExperience(e.Body.Experience).
				SetHammersApplied(e.Body.HammersApplied).
				Build()).
			Build()

		so := session.Announce(l)(ctx)(wp)(writer.CharacterInventoryChange)(writer.CharacterInventoryRefreshEquipable(sc.Tenant())(eqp))
		err := session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.WorldId(), sc.ChannelId())(e.CharacterId, so)
		if err != nil {
			l.WithError(err).Errorf("Unable to update [%d] in slot [%d] for character [%d].", e.Body.ItemId, e.Slot, e.CharacterId)
		}
	}
}

func handleInventoryMoveEvent(sc server.Model, wp writer.Producer) message.Handler[inventory2.InventoryChangedEvent[inventory2.InventoryChangedItemMoveBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, event inventory2.InventoryChangedEvent[inventory2.InventoryChangedItemMoveBody]) {
		if event.Type != inventory2.ChangedTypeMove {
			return
		}

		t := sc.Tenant()
		if !t.Is(tenant.MustFromContext(ctx)) {
			return
		}

		_ = session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.WorldId(), sc.ChannelId())(event.CharacterId, moveInInventory(l)(ctx)(wp)(event))
	}
}

func moveInInventory(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(event inventory2.InventoryChangedEvent[inventory2.InventoryChangedItemMoveBody]) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(event inventory2.InventoryChangedEvent[inventory2.InventoryChangedItemMoveBody]) model.Operator[session.Model] {
		cp := character.NewProcessor(l, ctx)
		return func(wp writer.Producer) func(event inventory2.InventoryChangedEvent[inventory2.InventoryChangedItemMoveBody]) model.Operator[session.Model] {
			return func(e inventory2.InventoryChangedEvent[inventory2.InventoryChangedItemMoveBody]) model.Operator[session.Model] {
				return func(s session.Model) error {
					c, err := cp.GetById(cp.InventoryDecorator, cp.PetModelDecorator)(s.CharacterId())
					if err != nil {
						l.WithError(err).Errorf("Unable to issue appearance update for character [%d] to others in map.", s.CharacterId())
						return err
					}

					errChannels := make(chan error, 3)
					go func() {
						inventoryType, ok := inventory.TypeFromItemId(e.Body.ItemId)
						if !ok {
							l.Errorf("Unable to identify inventory type by item [%d].", e.Body.ItemId)
							return
						}

						err := session.Announce(l)(ctx)(wp)(writer.CharacterInventoryChange)(writer.CharacterInventoryChangeBody(false, writer.InventoryMoveBodyWriter(inventoryType, e.Slot, e.Body.OldSlot)))(s)
						if err != nil {
							l.WithError(err).Errorf("Unable to move [%d] in slot [%d] to [%d] for character [%d].", e.Body.ItemId, e.Body.OldSlot, e.Slot, s.CharacterId())
						}
						errChannels <- err
					}()
					go func() {
						errChannels <- _map.NewProcessor(l, ctx).ForSessionsInMap(s.Map(), updateAppearance(l)(ctx)(wp)(c))
					}()
					go func() {
						it, ok := inventory.TypeFromItemId(e.Body.ItemId)
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
							_ = session.NewProcessor(l, ctx).IfPresentByCharacterId(s.WorldId(), s.ChannelId())(mm.Id(), func(os session.Model) error {
								return session.Announce(l)(ctx)(wp)(writer.MessengerOperation)(writer.MessengerOperationUpdateBody(ctx)(um.Slot(), c, byte(s.ChannelId())))(os)
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

func handleInventoryRemoveEvent(sc server.Model, wp writer.Producer) message.Handler[inventory2.InventoryChangedEvent[inventory2.InventoryChangedItemRemoveBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e inventory2.InventoryChangedEvent[inventory2.InventoryChangedItemRemoveBody]) {
		if e.Type != inventory2.ChangedTypeRemove {
			return
		}

		t := sc.Tenant()
		if !t.Is(tenant.MustFromContext(ctx)) {
			return
		}

		inventoryType, ok := inventory.TypeFromItemId(e.Body.ItemId)
		if !ok {
			l.Errorf("Unable to identify inventory type by item [%d].", e.Body.ItemId)
			return
		}

		err := session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.WorldId(), sc.ChannelId())(e.CharacterId, removeFromInventory(l)(ctx)(wp)(inventoryType, e.Slot))
		if err != nil {
			l.WithError(err).Errorf("Unable to remove [%d] in slot [%d] for character [%d].", e.Body.ItemId, e.Slot, e.CharacterId)
		}
	}
}

func removeFromInventory(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(inventoryType inventory.Type, slot int16) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(inventoryType inventory.Type, slot int16) model.Operator[session.Model] {
		return func(wp writer.Producer) func(inventoryType inventory.Type, slot int16) model.Operator[session.Model] {
			return func(inventoryType inventory.Type, slot int16) model.Operator[session.Model] {
				return session.Announce(l)(ctx)(wp)(writer.CharacterInventoryChange)(writer.CharacterInventoryChangeBody(false, writer.InventoryRemoveBodyWriter(inventoryType, slot)))
			}
		}
	}
}

func handleInventoryItemReservationCancelledEvent(sc server.Model, wp writer.Producer) message.Handler[inventory2.InventoryChangedEvent[inventory2.InventoryChangedItemRemoveBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e inventory2.InventoryChangedEvent[inventory2.InventoryChangedItemRemoveBody]) {
		if e.Type != inventory2.ChangedTypeReservationCancelled {
			return
		}

		t := sc.Tenant()
		if !t.Is(tenant.MustFromContext(ctx)) {
			return
		}

		_ = session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.WorldId(), sc.ChannelId())(e.CharacterId, session.Announce(l)(ctx)(wp)(writer.StatChanged)(writer.StatChangedBody(l)(make([]model2.StatUpdate, 0), true)))
	}
}
