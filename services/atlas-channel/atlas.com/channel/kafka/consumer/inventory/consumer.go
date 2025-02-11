package inventory

import (
	"atlas-channel/character"
	consumer2 "atlas-channel/kafka/consumer"
	_map "atlas-channel/map"
	"atlas-channel/server"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"
	"errors"
	"github.com/Chronicle20/atlas-constants/inventory"
	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/handler"
	"github.com/Chronicle20/atlas-kafka/message"
	"github.com/Chronicle20/atlas-kafka/topic"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("character_inventory_changed_event")(EnvEventInventoryChanged)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
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
			}
		}
	}
}

func handleInventoryAddEvent(sc server.Model, wp writer.Producer) message.Handler[inventoryChangedEvent[inventoryChangedItemAddBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, event inventoryChangedEvent[inventoryChangedItemAddBody]) {
		t := sc.Tenant()
		if !t.Is(tenant.MustFromContext(ctx)) {
			return
		}

		if event.Type != ChangedTypeAdd {
			return
		}

		session.IfPresentByCharacterId(t, sc.WorldId(), sc.ChannelId())(event.CharacterId, addToInventory(l)(ctx)(wp)(event))

	}
}

func addToInventory(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(event inventoryChangedEvent[inventoryChangedItemAddBody]) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(event inventoryChangedEvent[inventoryChangedItemAddBody]) model.Operator[session.Model] {
		return func(wp writer.Producer) func(event inventoryChangedEvent[inventoryChangedItemAddBody]) model.Operator[session.Model] {
			inventoryChangeFunc := session.Announce(l)(ctx)(wp)(writer.CharacterInventoryChange)
			return func(event inventoryChangedEvent[inventoryChangedItemAddBody]) model.Operator[session.Model] {
				return func(s session.Model) error {
					var bp writer.BodyProducer
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
						bp = writer.CharacterInventoryAddEquipableBody(tenant.MustFromContext(ctx))(byte(inventoryType), event.Slot, e, false)
					} else if inventoryType == 2 || inventoryType == 3 || inventoryType == 4 {
						i, err := character.GetItemInSlot(l)(ctx)(s.CharacterId(), byte(inventoryType), event.Slot)()
						if err != nil {
							return err
						}
						bp = writer.CharacterInventoryAddItemBody(tenant.MustFromContext(ctx))(byte(inventoryType), event.Slot, i, false)
					} else if inventoryType == 5 {
						i, err := character.GetItemInSlot(l)(ctx)(s.CharacterId(), byte(inventoryType), event.Slot)()
						if err != nil {
							return err
						}
						bp = writer.CharacterInventoryAddCashItemBody(tenant.MustFromContext(ctx))(byte(inventoryType), event.Slot, i, false)
					}
					err := inventoryChangeFunc(s, bp)
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
	return func(l logrus.FieldLogger, ctx context.Context, event inventoryChangedEvent[inventoryChangedItemUpdateBody]) {
		t := sc.Tenant()
		if !t.Is(tenant.MustFromContext(ctx)) {
			return
		}

		if event.Type != ChangedTypeUpdate {
			return
		}

		session.IfPresentByCharacterId(t, sc.WorldId(), sc.ChannelId())(event.CharacterId, updateInInventory(l)(ctx)(wp)(event))
	}
}

func updateInInventory(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(event inventoryChangedEvent[inventoryChangedItemUpdateBody]) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(event inventoryChangedEvent[inventoryChangedItemUpdateBody]) model.Operator[session.Model] {
		return func(wp writer.Producer) func(event inventoryChangedEvent[inventoryChangedItemUpdateBody]) model.Operator[session.Model] {
			inventoryChangeFunc := session.Announce(l)(ctx)(wp)(writer.CharacterInventoryChange)
			return func(event inventoryChangedEvent[inventoryChangedItemUpdateBody]) model.Operator[session.Model] {
				return func(s session.Model) error {
					inventoryType, ok := inventory.TypeFromItemId(event.Body.ItemId)
					if !ok {
						l.Errorf("Unable to identify inventory type by item [%d].", event.Body.ItemId)
						return errors.New("unable to identify inventory type")
					}

					err := inventoryChangeFunc(s, writer.CharacterInventoryUpdateBody(tenant.MustFromContext(ctx))(byte(inventoryType), event.Slot, event.Body.Quantity, false))
					if err != nil {
						l.WithError(err).Errorf("Unable to update [%d] in slot [%d] for character [%d].", event.Body.ItemId, event.Slot, s.CharacterId())
					}
					return err
				}
			}
		}
	}
}

func handleInventoryMoveEvent(sc server.Model, wp writer.Producer) message.Handler[inventoryChangedEvent[inventoryChangedItemMoveBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, event inventoryChangedEvent[inventoryChangedItemMoveBody]) {
		t := sc.Tenant()
		if !t.Is(tenant.MustFromContext(ctx)) {
			return
		}

		if event.Type != ChangedTypeMove {
			return
		}

		session.IfPresentByCharacterId(t, sc.WorldId(), sc.ChannelId())(event.CharacterId, moveInInventory(l)(ctx)(wp)(event))
	}
}

func moveInInventory(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(event inventoryChangedEvent[inventoryChangedItemMoveBody]) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(event inventoryChangedEvent[inventoryChangedItemMoveBody]) model.Operator[session.Model] {
		return func(wp writer.Producer) func(event inventoryChangedEvent[inventoryChangedItemMoveBody]) model.Operator[session.Model] {
			inventoryChangeFunc := session.Announce(l)(ctx)(wp)(writer.CharacterInventoryChange)
			return func(event inventoryChangedEvent[inventoryChangedItemMoveBody]) model.Operator[session.Model] {
				return func(s session.Model) error {
					errChannels := make(chan error, 2)
					go func() {
						inventoryType, ok := inventory.TypeFromItemId(event.Body.ItemId)
						if !ok {
							l.Errorf("Unable to identify inventory type by item [%d].", event.Body.ItemId)
							return
						}

						err := inventoryChangeFunc(s, writer.CharacterInventoryMoveBody(tenant.MustFromContext(ctx))(byte(inventoryType), event.Slot, event.Body.OldSlot, false))
						if err != nil {
							l.WithError(err).Errorf("Unable to move [%d] in slot [%d] to [%d] for character [%d].", event.Body.ItemId, event.Body.OldSlot, event.Slot, s.CharacterId())
						}
						errChannels <- err
					}()
					go func() {
						c, err := character.GetByIdWithInventory(l)(ctx)()(s.CharacterId())
						if err != nil {
							l.WithError(err).Errorf("Unable to issue appearance update for character [%d] to others in map.", s.CharacterId())
							errChannels <- err
						}
						errChannels <- _map.ForSessionsInMap(l)(ctx)(s.WorldId(), s.ChannelId(), s.MapId(), updateAppearance(l)(ctx)(wp)(c))
					}()

					var err error
					for i := 0; i < 2; i++ {
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
			appearanceUpdateFunc := session.Announce(l)(ctx)(wp)(writer.CharacterAppearanceUpdate)
			return func(c character.Model) model.Operator[session.Model] {
				return func(s session.Model) error {
					err := appearanceUpdateFunc(s, writer.CharacterAppearanceUpdateBody(tenant.MustFromContext(ctx))(c))
					if err != nil {
						l.WithError(err).Errorf("Unable to issue appearance update for character [%d] to others in map.", s.CharacterId())
					}
					return err
				}
			}
		}
	}
}

func handleInventoryRemoveEvent(sc server.Model, wp writer.Producer) message.Handler[inventoryChangedEvent[inventoryChangedItemRemoveBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, event inventoryChangedEvent[inventoryChangedItemRemoveBody]) {
		t := sc.Tenant()
		if !t.Is(tenant.MustFromContext(ctx)) {
			return
		}

		if event.Type != ChangedTypeRemove {
			return
		}

		session.IfPresentByCharacterId(t, sc.WorldId(), sc.ChannelId())(event.CharacterId, removeFromInventory(l)(ctx)(wp)(event))
	}
}

func removeFromInventory(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(event inventoryChangedEvent[inventoryChangedItemRemoveBody]) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(event inventoryChangedEvent[inventoryChangedItemRemoveBody]) model.Operator[session.Model] {
		return func(wp writer.Producer) func(event inventoryChangedEvent[inventoryChangedItemRemoveBody]) model.Operator[session.Model] {
			inventoryChangeFunc := session.Announce(l)(ctx)(wp)(writer.CharacterInventoryChange)
			return func(event inventoryChangedEvent[inventoryChangedItemRemoveBody]) model.Operator[session.Model] {
				return func(s session.Model) error {
					inventoryType, ok := inventory.TypeFromItemId(event.Body.ItemId)
					if !ok {
						l.Errorf("Unable to identify inventory type by item [%d].", event.Body.ItemId)
						return errors.New("unable to identify inventory type")
					}

					err := inventoryChangeFunc(s, writer.CharacterInventoryRemoveBody(tenant.MustFromContext(ctx))(byte(inventoryType), event.Slot, false))
					if err != nil {
						l.WithError(err).Errorf("Unable to remove [%d] in slot [%d] for character [%d].", event.Body.ItemId, event.Slot, s.CharacterId())
					}
					return err
				}
			}
		}
	}
}
