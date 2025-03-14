package inventory

import (
	"atlas-channel/character"
	consumer2 "atlas-channel/kafka/consumer"
	_map "atlas-channel/map"
	"atlas-channel/messenger"
	"atlas-channel/pet"
	"atlas-channel/server"
	"atlas-channel/session"
	model2 "atlas-channel/socket/model"
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
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("character_inventory_changed_event")(EnvEventInventoryChanged)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser), consumer.SetStartOffset(kafka.LastOffset))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) {
		return func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) {
			return func(rf func(topic string, handler handler.Handler) (string, error)) {
				var t string
				t, _ = topic.EnvProvider(l)(EnvEventInventoryChanged)()
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleInventoryAddEvent(sc, wp))))
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleInventoryUpdateEvent(sc, wp))))
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleInventoryMoveEvent(sc, wp))))
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleInventoryRemoveEvent(sc, wp))))
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleInventoryItemReservationCancelledEvent(sc, wp))))
			}
		}
	}
}

func handleInventoryAddEvent(sc server.Model, wp writer.Producer) message.Handler[inventoryChangedEvent[inventoryChangedItemAddBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e inventoryChangedEvent[inventoryChangedItemAddBody]) {
		t := sc.Tenant()
		if !t.Is(tenant.MustFromContext(ctx)) {
			return
		}

		if e.Type != ChangedTypeAdd {
			return
		}

		session.IfPresentByCharacterId(t, sc.WorldId(), sc.ChannelId())(e.CharacterId, addToInventory(l)(ctx)(wp)(e))

	}
}

func addToInventory(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(event inventoryChangedEvent[inventoryChangedItemAddBody]) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(event inventoryChangedEvent[inventoryChangedItemAddBody]) model.Operator[session.Model] {
		t := tenant.MustFromContext(ctx)
		return func(wp writer.Producer) func(event inventoryChangedEvent[inventoryChangedItemAddBody]) model.Operator[session.Model] {
			return func(event inventoryChangedEvent[inventoryChangedItemAddBody]) model.Operator[session.Model] {
				return func(s session.Model) error {
					var itemWriter model.Operator[*response.Writer]
					inventoryType, ok := inventory.TypeFromItemId(event.Body.ItemId)
					if !ok {
						l.Errorf("Unable to identify inventory type by item [%d].", event.Body.ItemId)
						return errors.New("unable to identify inventory type")
					}

					if inventoryType == 1 {
						e, err := character.GetEquipableInSlot(l)(ctx)(s.CharacterId(), event.Slot)()
						if err != nil {
							return err
						}
						itemWriter = model.FlipOperator(writer.WriteEquipableInfo(t)(true))(e)
					} else if inventoryType == 2 || inventoryType == 3 || inventoryType == 4 {
						i, err := character.GetItemInSlot(l)(ctx)(s.CharacterId(), byte(inventoryType), event.Slot)()
						if err != nil {
							return err
						}
						itemWriter = model.FlipOperator(writer.WriteItemInfo(true))(i)
					} else if inventoryType == 5 {
						i, err := character.GetItemInSlot(l)(ctx)(s.CharacterId(), byte(inventoryType), event.Slot)()
						if err != nil {
							return err
						}

						if item.GetClassification(item.Id(i.ItemId())) == item.Classification(500) {
							p, err := pet.GetByOwnerItem(l)(ctx)(event.CharacterId, i.Id())
							if err != nil {
								return err
							}
							itemWriter = model.FlipOperator(writer.WritePetCashItemInfo(true)(p))(i)
						} else {
							itemWriter = model.FlipOperator(writer.WriteCashItemInfo(true))(i)
						}
					}
					err := session.Announce(l)(ctx)(wp)(writer.CharacterInventoryChange)(writer.CharacterInventoryAddBody(byte(inventoryType), event.Slot, false)(itemWriter))(s)
					if err != nil {
						l.WithError(err).Errorf("Unable to add [%d] to slot [%d] for character [%d].", event.Body.ItemId, event.Slot, s.CharacterId())
					}
					return err
				}
			}
		}
	}
}

func handleInventoryUpdateEvent(sc server.Model, wp writer.Producer) message.Handler[inventoryChangedEvent[inventoryChangedItemUpdateBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e inventoryChangedEvent[inventoryChangedItemUpdateBody]) {
		if e.Type != ChangedTypeUpdate {
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

		err := session.IfPresentByCharacterId(t, sc.WorldId(), sc.ChannelId())(e.CharacterId, updateInInventory(l)(ctx)(wp)(inventoryType, e.Slot, e.Body.Quantity))
		if err != nil {
			l.WithError(err).Errorf("Unable to update [%d] in slot [%d] for character [%d].", e.Body.ItemId, e.Slot, e.CharacterId)
		}
	}
}

func updateInInventory(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(inventoryType inventory.Type, slot int16, quantity uint32) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(inventoryType inventory.Type, slot int16, quantity uint32) model.Operator[session.Model] {
		return func(wp writer.Producer) func(inventoryType inventory.Type, slot int16, quantity uint32) model.Operator[session.Model] {
			return func(inventoryType inventory.Type, slot int16, quantity uint32) model.Operator[session.Model] {
				return session.Announce(l)(ctx)(wp)(writer.CharacterInventoryChange)(writer.CharacterInventoryUpdateBody(tenant.MustFromContext(ctx))(byte(inventoryType), slot, quantity, false))
			}
		}
	}
}

func handleInventoryMoveEvent(sc server.Model, wp writer.Producer) message.Handler[inventoryChangedEvent[inventoryChangedItemMoveBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, event inventoryChangedEvent[inventoryChangedItemMoveBody]) {
		if event.Type != ChangedTypeMove {
			return
		}

		t := sc.Tenant()
		if !t.Is(tenant.MustFromContext(ctx)) {
			return
		}

		session.IfPresentByCharacterId(t, sc.WorldId(), sc.ChannelId())(event.CharacterId, moveInInventory(l)(ctx)(wp)(event))
	}
}

func moveInInventory(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(event inventoryChangedEvent[inventoryChangedItemMoveBody]) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(event inventoryChangedEvent[inventoryChangedItemMoveBody]) model.Operator[session.Model] {
		return func(wp writer.Producer) func(event inventoryChangedEvent[inventoryChangedItemMoveBody]) model.Operator[session.Model] {
			return func(e inventoryChangedEvent[inventoryChangedItemMoveBody]) model.Operator[session.Model] {
				return func(s session.Model) error {
					c, err := character.GetByIdWithInventory(l)(ctx)(character.PetModelDecorator(l)(ctx))(s.CharacterId())
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

						err := session.Announce(l)(ctx)(wp)(writer.CharacterInventoryChange)(writer.CharacterInventoryMoveBody(tenant.MustFromContext(ctx))(byte(inventoryType), e.Slot, e.Body.OldSlot, false))(s)
						if err != nil {
							l.WithError(err).Errorf("Unable to move [%d] in slot [%d] to [%d] for character [%d].", e.Body.ItemId, e.Body.OldSlot, e.Slot, s.CharacterId())
						}
						errChannels <- err
					}()
					go func() {
						errChannels <- _map.ForSessionsInMap(l)(ctx)(s.Map(), updateAppearance(l)(ctx)(wp)(c))
					}()
					go func() {
						it, ok := inventory.TypeFromItemId(e.Body.ItemId)
						if !ok || it != inventory.TypeValueEquip {
							return
						}

						if e.Slot > 0 && e.Body.OldSlot > 0 {
							return
						}

						m, err := messenger.GetByMemberId(l)(ctx)(e.CharacterId)
						if err != nil {
							return
						}
						um, err := m.FindMember(e.CharacterId)
						if err != nil {
							return
						}

						for _, mm := range m.Members() {
							_ = session.IfPresentByCharacterId(s.Tenant(), s.WorldId(), s.ChannelId())(mm.Id(), func(os session.Model) error {
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

func handleInventoryRemoveEvent(sc server.Model, wp writer.Producer) message.Handler[inventoryChangedEvent[inventoryChangedItemRemoveBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e inventoryChangedEvent[inventoryChangedItemRemoveBody]) {
		if e.Type != ChangedTypeRemove {
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

		err := session.IfPresentByCharacterId(t, sc.WorldId(), sc.ChannelId())(e.CharacterId, removeFromInventory(l)(ctx)(wp)(inventoryType, e.Slot))
		if err != nil {
			l.WithError(err).Errorf("Unable to remove [%d] in slot [%d] for character [%d].", e.Body.ItemId, e.Slot, e.CharacterId)
		}
	}
}

func removeFromInventory(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(inventoryType inventory.Type, slot int16) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(inventoryType inventory.Type, slot int16) model.Operator[session.Model] {
		return func(wp writer.Producer) func(inventoryType inventory.Type, slot int16) model.Operator[session.Model] {
			return func(inventoryType inventory.Type, slot int16) model.Operator[session.Model] {
				return session.Announce(l)(ctx)(wp)(writer.CharacterInventoryChange)(writer.CharacterInventoryRemoveBody(tenant.MustFromContext(ctx))(byte(inventoryType), slot, false))
			}
		}
	}
}

func handleInventoryItemReservationCancelledEvent(sc server.Model, wp writer.Producer) message.Handler[inventoryChangedEvent[inventoryChangedItemRemoveBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e inventoryChangedEvent[inventoryChangedItemRemoveBody]) {
		if e.Type != ChangedTypeReservationCancelled {
			return
		}

		t := sc.Tenant()
		if !t.Is(tenant.MustFromContext(ctx)) {
			return
		}

		_ = session.IfPresentByCharacterId(sc.Tenant(), sc.WorldId(), sc.ChannelId())(e.CharacterId, session.Announce(l)(ctx)(wp)(writer.StatChanged)(writer.StatChangedBody(l)(make([]model2.StatUpdate, 0), true)))
	}
}
