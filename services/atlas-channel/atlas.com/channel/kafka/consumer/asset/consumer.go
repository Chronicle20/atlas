package asset

import (
	"atlas-channel/character"
	consumer2 "atlas-channel/kafka/consumer"
	asset2 "atlas-channel/kafka/message/asset"
	_map "atlas-channel/map"
	"atlas-channel/messenger"
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
	"github.com/Chronicle20/atlas-socket/response"
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
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleAssetQuantityUpdatedEvent(sc, wp))))
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleAssetMoveEvent(sc, wp))))
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleAssetDeletedEvent(sc, wp))))
			}
		}
	}
}

func handleAssetCreatedEvent(sc server.Model, wp writer.Producer) message.Handler[asset2.StatusEvent[asset2.CreatedStatusEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e asset2.StatusEvent[asset2.CreatedStatusEventBody]) {
		t := sc.Tenant()
		if !t.Is(tenant.MustFromContext(ctx)) {
			return
		}

		if e.Type != asset2.StatusEventTypeCreated {
			return
		}

		cp := character.NewProcessor(l, ctx)
		_ = session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.WorldId(), sc.ChannelId())(e.CharacterId, func(s session.Model) error {
			var itemWriter model.Operator[*response.Writer]
			inventoryType, ok := inventory.TypeFromItemId(e.TemplateId)
			if !ok {
				l.Errorf("Unable to identify inventory type by item [%d].", e.TemplateId)
				return errors.New("unable to identify inventory type")
			}

			i, err := cp.GetItemInSlot(s.CharacterId(), inventoryType, e.Slot)()
			if err != nil {
				return err
			}
			itemWriter = model.FlipOperator(writer.WriteAssetInfo(t)(true))(i)

			bp := writer.CharacterInventoryChangeBody(false, writer.InventoryAddBodyWriter(inventoryType, e.Slot, itemWriter))
			err = session.Announce(l)(ctx)(wp)(writer.CharacterInventoryChange)(bp)(s)
			if err != nil {
				l.WithError(err).Errorf("Unable to add [%d] to slot [%d] for character [%d].", e.TemplateId, e.Slot, s.CharacterId())
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

		inventoryType, ok := inventory.TypeFromItemId(e.TemplateId)
		if !ok {
			l.Errorf("Unable to identify inventory type by item [%d].", e.TemplateId)
			return
		}

		so := session.Announce(l)(ctx)(wp)(writer.CharacterInventoryChange)(writer.CharacterInventoryChangeBody(false, writer.InventoryQuantityUpdateBodyWriter(inventoryType, e.Slot, e.Body.Quantity)))
		err := session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.WorldId(), sc.ChannelId())(e.CharacterId, so)
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

		_ = session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.WorldId(), sc.ChannelId())(e.CharacterId, moveInCompartment(l)(ctx)(wp)(e))
	}
}

func moveInCompartment(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(e asset2.StatusEvent[asset2.MovedStatusEventBody]) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(e asset2.StatusEvent[asset2.MovedStatusEventBody]) model.Operator[session.Model] {
		cp := character.NewProcessor(l, ctx)
		return func(wp writer.Producer) func(e asset2.StatusEvent[asset2.MovedStatusEventBody]) model.Operator[session.Model] {
			return func(e asset2.StatusEvent[asset2.MovedStatusEventBody]) model.Operator[session.Model] {
				return func(s session.Model) error {
					c, err := cp.GetById(cp.InventoryDecorator, cp.PetModelDecorator)(s.CharacterId())
					if err != nil {
						l.WithError(err).Errorf("Unable to issue appearance update for character [%d] to others in map.", s.CharacterId())
						return err
					}

					errChannels := make(chan error, 3)
					go func() {
						inventoryType, ok := inventory.TypeFromItemId(e.TemplateId)
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
						errChannels <- _map.NewProcessor(l, ctx).ForSessionsInMap(s.Map(), updateAppearance(l)(ctx)(wp)(c))
					}()
					go func() {
						it, ok := inventory.TypeFromItemId(e.TemplateId)
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

func handleAssetDeletedEvent(sc server.Model, wp writer.Producer) message.Handler[asset2.StatusEvent[asset2.DeletedStatusEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e asset2.StatusEvent[asset2.DeletedStatusEventBody]) {
		if e.Type != asset2.StatusEventTypeDeleted {
			return
		}

		t := sc.Tenant()
		if !t.Is(tenant.MustFromContext(ctx)) {
			return
		}

		inventoryType, ok := inventory.TypeFromItemId(e.TemplateId)
		if !ok {
			l.Errorf("Unable to identify inventory type by item [%d].", e.TemplateId)
			return
		}

		af := session.Announce(l)(ctx)(wp)(writer.CharacterInventoryChange)(writer.CharacterInventoryChangeBody(false, writer.InventoryRemoveBodyWriter(inventoryType, e.Slot)))
		err := session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.WorldId(), sc.ChannelId())(e.CharacterId, af)
		if err != nil {
			l.WithError(err).Errorf("Unable to remove [%d] in slot [%d] for character [%d].", e.TemplateId, e.Slot, e.CharacterId)
		}
	}
}
