package pet

import (
	"atlas-channel/character"
	"atlas-channel/inventory/compartment/asset"
	consumer2 "atlas-channel/kafka/consumer"
	_map "atlas-channel/map"
	"atlas-channel/pet"
	"atlas-channel/server"
	"atlas-channel/session"
	model2 "atlas-channel/socket/model"
	"atlas-channel/socket/writer"
	"context"
	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/handler"
	"github.com/Chronicle20/atlas-kafka/message"
	"github.com/Chronicle20/atlas-kafka/topic"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("pet_status_event")(EnvStatusEventTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser), consumer.SetStartOffset(kafka.LastOffset))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) {
		return func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) {
			return func(rf func(topic string, handler handler.Handler) (string, error)) {
				var t string
				t, _ = topic.EnvProvider(l)(EnvStatusEventTopic)()
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleSpawned(sc, wp))))
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleDespawned(sc, wp))))
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleCommandResponse(sc, wp))))
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleClosenessChanged(sc, wp))))
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleFullnessChanged(sc, wp))))
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleLevelChanged(sc, wp))))
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleSlotChanged(sc, wp))))
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleExcludeChanged(sc, wp))))
			}
		}
	}
}

func handleSpawned(sc server.Model, wp writer.Producer) message.Handler[statusEvent[spawnedStatusEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e statusEvent[spawnedStatusEventBody]) {
		if e.Type != StatusEventTypeSpawned {
			return
		}

		s, err := session.GetByCharacterId(sc.Tenant(), sc.WorldId(), sc.ChannelId())(e.OwnerId)
		if err != nil {
			return
		}

		p := pet.NewModelBuilder(e.PetId, 0, e.Body.TemplateId, e.Body.Name).
			SetOwnerID(e.OwnerId).
			SetSlot(e.Body.Slot).
			SetLevel(e.Body.Level).
			SetCloseness(e.Body.Closeness).
			SetFullness(e.Body.Fullness).
			SetX(e.Body.X).
			SetY(e.Body.Y).
			SetStance(e.Body.Stance).
			SetFoothold(e.Body.FH).
			Build()
		_ = announceSpawn(l)(ctx)(wp)(p)(s)
	}
}

func announceSpawn(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(p pet.Model) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(p pet.Model) model.Operator[session.Model] {
		t := tenant.MustFromContext(ctx)
		return func(wp writer.Producer) func(p pet.Model) model.Operator[session.Model] {
			return func(p pet.Model) model.Operator[session.Model] {
				return func(s session.Model) error {
					err := enableActions(l)(ctx)(wp)(s)
					if err != nil {
						l.WithError(err).Errorf("Unable to write [%s] for character [%d].", writer.StatChanged, s.CharacterId())
					}
					err = session.Announce(l)(ctx)(wp)(writer.PetActivated)(writer.PetSpawnBody(l)(t)(p))(s)
					if err != nil {
						l.WithError(err).Errorf("Unable to write pet spawned to character.")
					}
					err = _map.ForOtherSessionsInMap(l)(ctx)(s.Map(), s.CharacterId(), session.Announce(l)(ctx)(wp)(writer.PetActivated)(writer.PetSpawnBody(l)(t)(p)))
					if err != nil {
						l.WithError(err).Errorf("Unable to write pet spawned to other characters.")
					}
					return nil
				}
			}
		}
	}
}

func handleDespawned(sc server.Model, wp writer.Producer) message.Handler[statusEvent[despawnedStatusEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e statusEvent[despawnedStatusEventBody]) {
		if e.Type != StatusEventTypeDespawned {
			return
		}

		s, err := session.GetByCharacterId(sc.Tenant(), sc.WorldId(), sc.ChannelId())(e.OwnerId)
		if err != nil {
			return
		}
		_ = announceDespawn(l)(ctx)(wp)(e.Body.OldSlot, e.Body.Reason)(s)
	}
}

func announceDespawn(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(slot int8, reason string) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(slot int8, reason string) model.Operator[session.Model] {
		return func(wp writer.Producer) func(slot int8, reason string) model.Operator[session.Model] {
			return func(slot int8, reason string) model.Operator[session.Model] {
				return func(s session.Model) error {
					err := enableActions(l)(ctx)(wp)(s)
					if err != nil {
						l.WithError(err).Errorf("Unable to write [%s] for character [%d].", writer.StatChanged, s.CharacterId())
					}
					err = session.Announce(l)(ctx)(wp)(writer.PetActivated)(writer.PetDespawnBody(l)(s.CharacterId(), slot, reason))(s)
					if err != nil {
						l.WithError(err).Errorf("Unable to write pet despawned to character.")
					}
					err = _map.ForOtherSessionsInMap(l)(ctx)(s.Map(), s.CharacterId(), session.Announce(l)(ctx)(wp)(writer.PetActivated)(writer.PetDespawnBody(l)(s.CharacterId(), slot, reason)))
					if err != nil {
						l.WithError(err).Errorf("Unable to write pet despawned to other characters.")
					}
					return nil
				}
			}
		}
	}
}

func handleCommandResponse(sc server.Model, wp writer.Producer) message.Handler[statusEvent[commandResponseStatusEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e statusEvent[commandResponseStatusEventBody]) {
		if e.Type != StatusEventTypeCommandResponse {
			return
		}

		s, err := session.GetByCharacterId(sc.Tenant(), sc.WorldId(), sc.ChannelId())(e.OwnerId)
		if err != nil {
			return
		}

		go func() {
			err = enableActions(l)(ctx)(wp)(s)
			if err != nil {
				l.WithError(err).Errorf("Unable to write [%s] for character [%d].", writer.StatChanged, s.CharacterId())
			}
		}()
		go func() {
			p := pet.NewModelBuilder(e.PetId, 0, 0, "").
				SetOwnerID(e.OwnerId).
				SetSlot(e.Body.Slot).
				SetCloseness(e.Body.Closeness).
				SetFullness(e.Body.Fullness).
				Build()
			_ = _map.ForSessionsInMap(l)(ctx)(s.Map(), session.Announce(l)(ctx)(wp)(writer.PetCommandResponse)(writer.PetCommandResponseBody(p, e.Body.CommandId, e.Body.Success, false)))
		}()
	}
}

func announcePetStatUpdate(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(p pet.Model) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(p pet.Model) model.Operator[session.Model] {
		cp := character.NewProcessor(l, ctx)
		return func(wp writer.Producer) func(p pet.Model) model.Operator[session.Model] {
			return func(p pet.Model) model.Operator[session.Model] {
				c, err := cp.GetById(cp.InventoryDecorator)(p.OwnerId())
				if err != nil {
					return func(s session.Model) error {
						return err
					}
				}

				var a asset.Model[asset.PetReferenceData]
				for _, ii := range c.Inventory().Cash().Assets() {
					if ii.ReferenceId() == p.InventoryItemId() {
						if prd, ok := ii.ReferenceData().(asset.PetReferenceData); ok {
							a = asset.NewBuilder[asset.PetReferenceData](ii.Id(), ii.TemplateId(), ii.ReferenceId(), ii.ReferenceType()).
								SetSlot(ii.Slot()).
								SetExpiration(ii.Expiration()).
								SetReferenceData(prd).
								Build()
						}
					}
				}
				return session.Announce(l)(ctx)(wp)(writer.CharacterInventoryChange)(writer.CharacterInventoryRefreshPet(a))
			}
		}
	}
}

func handleClosenessChanged(sc server.Model, wp writer.Producer) message.Handler[statusEvent[closenessChangedStatusEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e statusEvent[closenessChangedStatusEventBody]) {
		if e.Type != StatusEventTypeClosenessChanged {
			return
		}

		_ = session.IfPresentByCharacterId(sc.Tenant(), sc.WorldId(), sc.ChannelId())(e.OwnerId, func(s session.Model) error {
			p, err := pet.GetById(l)(ctx)(e.PetId)
			if err != nil {
				return err
			}

			err = announcePetStatUpdate(l)(ctx)(wp)(p)(s)
			if err != nil {
				return err
			}
			return nil
		})
	}
}

func handleFullnessChanged(sc server.Model, wp writer.Producer) message.Handler[statusEvent[fullnessChangedStatusEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e statusEvent[fullnessChangedStatusEventBody]) {
		if e.Type != StatusEventTypeFullnessChanged {
			return
		}

		_ = session.IfPresentByCharacterId(sc.Tenant(), sc.WorldId(), sc.ChannelId())(e.OwnerId, func(s session.Model) error {
			p, err := pet.GetById(l)(ctx)(e.PetId)
			if err != nil {
				return err
			}

			err = announcePetStatUpdate(l)(ctx)(wp)(p)(s)
			if err != nil {
				return err
			}

			return _map.ForSessionsInMap(l)(ctx)(s.Map(), func(os session.Model) error {
				if e.Body.Amount > 0 {
					err := session.Announce(l)(ctx)(wp)(writer.PetCommandResponse)(writer.PetFoodResponseBody(p, 0, true, false))(os)
					if err != nil {
						l.WithError(err).Errorf("Unable to issue pet [%d] response to food.", p.Id())
					}
					return err
				} else {
					return nil
				}
			})
		})
	}
}

func handleLevelChanged(sc server.Model, wp writer.Producer) message.Handler[statusEvent[levelChangedStatusEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e statusEvent[levelChangedStatusEventBody]) {
		if e.Type != StatusEventTypeLevelChanged {
			return
		}

		_ = session.IfPresentByCharacterId(sc.Tenant(), sc.WorldId(), sc.ChannelId())(e.OwnerId, func(s session.Model) error {
			p, err := pet.GetById(l)(ctx)(e.PetId)
			if err != nil {
				return err
			}

			err = announcePetStatUpdate(l)(ctx)(wp)(p)(s)
			if err != nil {
				return err
			}

			return _map.ForSessionsInMap(l)(ctx)(s.Map(), func(os session.Model) error {
				if s.CharacterId() == os.CharacterId() {
					err = session.Announce(l)(ctx)(wp)(writer.CharacterEffect)(writer.CharacterPetEffectBody(l)(byte(e.Body.Slot), 0))(os)
					if err != nil {
						l.WithError(err).Errorf("Unable to issue pet [%d] level up.", p.Id())
					}
					return err
				} else {
					err = session.Announce(l)(ctx)(wp)(writer.CharacterEffectForeign)(writer.CharacterPetEffectForeignBody(l)(s.CharacterId(), byte(e.Body.Slot), 0))(os)
					if err != nil {
						l.WithError(err).Errorf("Unable to issue pet [%d] level up.", p.Id())
					}
					return err
				}
			})
		})
	}
}

func handleSlotChanged(sc server.Model, wp writer.Producer) message.Handler[statusEvent[slotChangedStatusEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e statusEvent[slotChangedStatusEventBody]) {
		if e.Type != StatusEventTypeSlotChanged {
			return
		}

		_ = session.IfPresentByCharacterId(sc.Tenant(), sc.WorldId(), sc.ChannelId())(e.OwnerId, func(s session.Model) error {
			stat := ""
			sn := int64(0)
			if e.Body.NewSlot < 0 {
				sn = 0
				if e.Body.OldSlot == 0 {
					stat = writer.StatPetSn1
				} else if e.Body.OldSlot == 1 {
					stat = writer.StatPetSn2
				} else if e.Body.OldSlot == 2 {
					stat = writer.StatPetSn3
				}
			} else if e.Body.NewSlot == 0 {
				stat = writer.StatPetSn1
				sn = int64(e.PetId)
			} else if e.Body.NewSlot == 1 {
				stat = writer.StatPetSn2
				sn = int64(e.PetId)
			} else {
				stat = writer.StatPetSn3
				sn = int64(e.PetId)
			}

			err := session.Announce(l)(ctx)(wp)(writer.StatChanged)(writer.StatChangedBody(l)([]model2.StatUpdate{model2.NewStatUpdate(stat, sn)}, true))(s)
			if err != nil {
				return err
			}

			if e.Body.OldSlot >= 0 && e.Body.NewSlot >= 0 {
				p, err := pet.GetById(l)(ctx)(e.PetId)
				if err != nil {
					return err
				}

				if e.Body.OldSlot > e.Body.NewSlot {
					_ = announceDespawn(l)(ctx)(wp)(e.Body.OldSlot, writer.PetDespawnModeNormal)(s)
				}
				_ = announceSpawn(l)(ctx)(wp)(p)(s)
			}
			return nil
		})
	}
}

func enableActions(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(s session.Model) error {
	return func(ctx context.Context) func(wp writer.Producer) func(s session.Model) error {
		return func(wp writer.Producer) func(s session.Model) error {
			return session.Announce(l)(ctx)(wp)(writer.StatChanged)(writer.StatChangedBody(l)(make([]model2.StatUpdate, 0), true))
		}
	}
}

func handleExcludeChanged(sc server.Model, wp writer.Producer) message.Handler[statusEvent[excludeChangedStatusEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e statusEvent[excludeChangedStatusEventBody]) {
		if e.Type != StatusEventTypeExcludeChanged {
			return
		}

		_ = session.IfPresentByCharacterId(sc.Tenant(), sc.WorldId(), sc.ChannelId())(e.OwnerId, func(s session.Model) error {
			p, err := pet.GetById(l)(ctx)(e.PetId)
			if err != nil {
				return err
			}
			if p.Slot() < 0 {
				return nil
			}
			return session.Announce(l)(ctx)(wp)(writer.PetExcludeResponse)(writer.PetExcludeResponseBody(p))(s)
		})
	}
}
