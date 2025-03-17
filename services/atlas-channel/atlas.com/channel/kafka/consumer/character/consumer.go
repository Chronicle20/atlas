package character

import (
	"atlas-channel/character"
	consumer2 "atlas-channel/kafka/consumer"
	_map "atlas-channel/map"
	"atlas-channel/movement"
	"atlas-channel/party"
	"atlas-channel/server"
	"atlas-channel/session"
	model2 "atlas-channel/socket/model"
	"atlas-channel/socket/writer"
	"context"
	"github.com/Chronicle20/atlas-constants/channel"
	_map2 "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
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
			rf(consumer2.NewConfig(l)("character_status_event")(EnvEventTopicCharacterStatus)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser), consumer.SetStartOffset(kafka.LastOffset))
			rf(consumer2.NewConfig(l)("character_movement_event")(EnvEventTopicMovement)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser), consumer.SetStartOffset(kafka.LastOffset))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) {
		return func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) {
			return func(rf func(topic string, handler handler.Handler) (string, error)) {
				var t string
				t, _ = topic.EnvProvider(l)(EnvEventTopicCharacterStatus)()
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventStatChanged(sc, wp))))
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventMapChanged(sc, wp))))
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventExperienceChanged(sc, wp))))
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventFameChanged(sc, wp))))
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventMesoChanged(sc, wp))))
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventLevelChanged(sc, wp))))
				t, _ = topic.EnvProvider(l)(EnvEventTopicMovement)()
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleMovementEvent(sc, wp))))
			}
		}
	}
}

func handleStatusEventStatChanged(sc server.Model, wp writer.Producer) func(l logrus.FieldLogger, ctx context.Context, event statusEvent[statusEventStatChangedBody]) {
	return func(l logrus.FieldLogger, ctx context.Context, e statusEvent[statusEventStatChangedBody]) {
		if e.Type != StatusEventTypeStatChanged {
			return
		}

		if !sc.IsWorld(tenant.MustFromContext(ctx), world.Id(e.WorldId)) {
			return
		}

		c, err := character.GetById(l)(ctx)()(e.CharacterId)
		if err != nil {
			return
		}

		err = session.IfPresentByCharacterId(sc.Tenant(), sc.WorldId(), sc.ChannelId())(e.CharacterId, statChanged(l)(ctx)(wp)(c, e.Body.ExclRequestSent, e.Body.Updates))
		if err != nil {
			l.WithError(err).Errorf("Unable to issue stat changed to character [%d].", e.CharacterId)
		}

		var hpChange bool
		for _, u := range e.Body.Updates {
			if u == writer.StatHp || u == writer.StatMaxHp {
				hpChange = true
			}
		}
		if hpChange {
			imf := party.OtherMemberInMap(sc.WorldId(), sc.ChannelId(), _map2.Id(c.MapId()), c.Id())
			oip := party.MemberToMemberIdMapper(party.FilteredMemberProvider(imf)(party.ByMemberIdProvider(l)(ctx)(e.CharacterId)))
			err = session.ForEachByCharacterId(sc.Tenant(), sc.WorldId(), sc.ChannelId())(oip, session.Announce(l)(ctx)(wp)(writer.PartyMemberHP)(writer.PartyMemberHPBody(c.Id(), c.Hp(), c.MaxHp())))
			if err != nil {
				l.WithError(err).Errorf("Unable to announce character [%d] health to party members.", c.Id())
			}
		}
	}
}

func statChanged(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(c character.Model, exclRequestSent bool, updates []string) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(c character.Model, exclRequestSent bool, updates []string) model.Operator[session.Model] {
		return func(wp writer.Producer) func(c character.Model, exclRequestSent bool, updates []string) model.Operator[session.Model] {
			return func(c character.Model, exclRequestSent bool, updates []string) model.Operator[session.Model] {
				return func(s session.Model) error {
					var su = make([]model2.StatUpdate, 0)
					for _, update := range updates {
						value := int64(0)
						if update == writer.StatSkin {
							value = int64(c.SkinColor())
						} else if update == writer.StatFace {
							value = int64(c.Face())
						} else if update == writer.StatHair {
							value = int64(c.Hair())
						} else if update == writer.StatPetSn1 {
							if len(c.Pets()) > 0 {
								value = int64(c.Pets()[0].Id())
							} else {
								value = int64(0)
							}
						} else if update == writer.StatLevel {
							value = int64(c.Level())
						} else if update == writer.StatJob {
							value = int64(c.JobId())
						} else if update == writer.StatStrength {
							value = int64(c.Strength())
						} else if update == writer.StatDexterity {
							value = int64(c.Dexterity())
						} else if update == writer.StatIntelligence {
							value = int64(c.Intelligence())
						} else if update == writer.StatLuck {
							value = int64(c.Luck())
						} else if update == writer.StatHp {
							value = int64(c.Hp())
						} else if update == writer.StatMaxHp {
							value = int64(c.MaxHp())
						} else if update == writer.StatMp {
							value = int64(c.Mp())
						} else if update == writer.StatMaxMp {
							value = int64(c.MaxMp())
						} else if update == writer.StatAvailableAp {
							value = int64(c.Ap())
						} else if update == writer.StatAvailableSp {
							value = int64(c.Sp()[0])
						} else if update == writer.StatExperience {
							value = int64(c.Experience())
						} else if update == writer.StatFame {
							value = int64(c.Fame())
						} else if update == writer.StatMeso {
							value = int64(c.Meso())
						} else if update == writer.StatPetSn2 {
							if len(c.Pets()) > 1 {
								value = int64(c.Pets()[1].Id())
							} else {
								value = int64(0)
							}
						} else if update == writer.StatPetSn3 {
							if len(c.Pets()) > 2 {
								value = int64(c.Pets()[2].Id())
							} else {
								value = int64(0)
							}
						} else if update == writer.StatGachaponExperience {
							value = int64(c.GachaponExperience())
						}
						su = append(su, model2.NewStatUpdate(update, value))
					}

					err := session.Announce(l)(ctx)(wp)(writer.StatChanged)(writer.StatChangedBody(l)(su, exclRequestSent))(s)
					if err != nil {
						l.WithError(err).Errorf("Unable to write [%s] for character [%d].", writer.StatChanged, s.CharacterId())
						return err
					}
					return nil
				}
			}
		}
	}
}

func handleStatusEventMapChanged(sc server.Model, wp writer.Producer) func(l logrus.FieldLogger, ctx context.Context, event statusEvent[statusEventMapChangedBody]) {
	return func(l logrus.FieldLogger, ctx context.Context, event statusEvent[statusEventMapChangedBody]) {
		if event.Type != StatusEventTypeMapChanged {
			return
		}

		if !sc.Is(tenant.MustFromContext(ctx), world.Id(event.WorldId), channel.Id(event.Body.ChannelId)) {
			return
		}

		session.IfPresentByCharacterId(sc.Tenant(), sc.WorldId(), sc.ChannelId())(event.CharacterId, warpCharacter(l)(ctx)(wp)(event))
	}
}

func warpCharacter(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(event statusEvent[statusEventMapChangedBody]) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(event statusEvent[statusEventMapChangedBody]) model.Operator[session.Model] {
		t := tenant.MustFromContext(ctx)
		return func(wp writer.Producer) func(event statusEvent[statusEventMapChangedBody]) model.Operator[session.Model] {
			return func(event statusEvent[statusEventMapChangedBody]) model.Operator[session.Model] {
				return func(s session.Model) error {
					c, err := character.GetById(l)(ctx)()(s.CharacterId())
					if err != nil {
						l.WithError(err).Errorf("Unable to retrieve character [%d].", s.CharacterId())
						return err
					}

					s = session.SetMapId(_map2.Id(event.Body.TargetMapId))(t.Id(), s.SessionId())

					err = session.Announce(l)(ctx)(wp)(writer.SetField)(writer.WarpToMapBody(l, t)(s.ChannelId(), _map2.Id(event.Body.TargetMapId), event.Body.TargetPortalId, c.Hp()))(s)
					if err != nil {
						l.WithError(err).Errorf("Unable to show set field response for character [%d]", c.Id())
						return err
					}
					return nil
				}
			}
		}
	}
}

func handleStatusEventExperienceChanged(sc server.Model, wp writer.Producer) message.Handler[statusEvent[experienceChangedStatusEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e statusEvent[experienceChangedStatusEventBody]) {
		if e.Type != StatusEventTypeExperienceChanged {
			return
		}

		if !sc.Is(tenant.MustFromContext(ctx), world.Id(e.WorldId), channel.Id(e.Body.ChannelId)) {
			return
		}

		session.IfPresentByCharacterId(sc.Tenant(), sc.WorldId(), sc.ChannelId())(e.CharacterId, announceExperienceGain(l)(ctx)(wp)(e.Body.Distributions))
	}
}

func announceExperienceGain(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(distributions []experienceDistributions) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(distributions []experienceDistributions) model.Operator[session.Model] {
		t := tenant.MustFromContext(ctx)
		return func(wp writer.Producer) func(distributions []experienceDistributions) model.Operator[session.Model] {
			return func(distributions []experienceDistributions) model.Operator[session.Model] {
				return func(s session.Model) error {
					c := model2.IncreaseExperienceConfig{}

					for _, d := range distributions {
						if d.ExperienceType == ExperienceDistributionTypeWhite {
							c.White = true
							c.Amount = int32(d.Amount)
						} else if d.ExperienceType == ExperienceDistributionTypeYellow {
							c.White = false
							c.Amount = int32(d.Amount)
						} else if d.ExperienceType == ExperienceDistributionTypeChat {
							c.InChat = true
							c.Amount = int32(d.Amount)
						} else if d.ExperienceType == ExperienceDistributionTypeMonsterBook {
							c.MonsterBookBonus = int32(d.Amount)
						} else if d.ExperienceType == ExperienceDistributionTypeMonsterEvent {
							c.MobEventBonusPercentage = byte(d.Amount)
						} else if d.ExperienceType == ExperienceDistributionTypePlayTime {
							c.MobEventBonusPercentage = byte(d.Amount)
							c.PlayTimeHour = byte(d.Attr1)
						} else if d.ExperienceType == ExperienceDistributionTypeWedding {
							c.WeddingBonusEXP = int32(d.Amount)
						} else if d.ExperienceType == ExperienceDistributionTypeSpiritWeek {
							c.QuestBonusRate = byte(d.Amount)
						} else if d.ExperienceType == ExperienceDistributionTypeParty {
							c.PartyBonusExp = int32(d.Amount)
							c.PartyBonusEventRate = byte(d.Attr1)
						} else if d.ExperienceType == ExperienceDistributionTypeItem {
							c.ItemBonusEXP = int32(d.Amount)
						} else if d.ExperienceType == ExperienceDistributionTypeInternetCafe {
							c.PremiumIPExp = int32(d.Amount)
						} else if d.ExperienceType == ExperienceDistributionTypeRainbowWeek {
							c.RainbowWeekEventEXP = int32(d.Amount)
						} else if d.ExperienceType == ExperienceDistributionTypePartyRing {
							c.PartyEXPRingEXP = int32(d.Amount)
						} else if d.ExperienceType == ExperienceDistributionTypeCakePie {
							c.CakePieEventBonus = int32(d.Amount)
						}
					}

					err := session.Announce(l)(ctx)(wp)(writer.CharacterStatusMessage)(writer.CharacterStatusMessageOperationIncreaseExperienceBody(l, t)(c))(s)
					if err != nil {
						l.WithError(err).Errorf("Unable to announce experience gain to character [%d].", s.CharacterId())
						return err
					}
					return nil
				}
			}
		}
	}
}

func handleStatusEventFameChanged(sc server.Model, wp writer.Producer) message.Handler[statusEvent[fameChangedStatusEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e statusEvent[fameChangedStatusEventBody]) {
		if e.Type != StatusEventTypeFameChanged {
			return
		}

		if !sc.IsWorld(tenant.MustFromContext(ctx), world.Id(e.WorldId)) {
			return
		}

		c, err := character.GetById(l)(ctx)()(e.CharacterId)
		if err != nil {
			return
		}

		if e.Body.ActorType == StatusEventActorTypeCharacter {
			ac, err := character.GetById(l)(ctx)()(e.Body.ActorId)
			if err != nil {
				return
			}

			err = session.IfPresentByCharacterId(sc.Tenant(), sc.WorldId(), sc.ChannelId())(e.CharacterId, receiveFame(l)(ctx)(wp)(ac.Name(), e.Body.Amount))
			if err != nil {
				l.WithError(err).Errorf("Unable to notify character [%d] they received fame [%d] from [%s].", e.CharacterId, e.Body.Amount, ac.Name())
			}
			err = session.IfPresentByCharacterId(sc.Tenant(), sc.WorldId(), sc.ChannelId())(e.Body.ActorId, giveFame(l)(ctx)(wp)(c.Name(), e.Body.Amount, c.Fame()))
			if err != nil {
				l.WithError(err).Errorf("Unable to notify character [%d] they received fame [%d] from [%s].", e.Body.ActorId, e.Body.Amount, c.Name())
			}
		}
	}
}

func receiveFame(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(fromName string, amount int8) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(fromName string, amount int8) model.Operator[session.Model] {
		return func(wp writer.Producer) func(fromName string, amount int8) model.Operator[session.Model] {
			return func(fromName string, amount int8) model.Operator[session.Model] {
				return session.Announce(l)(ctx)(wp)(writer.FameResponse)(writer.ReceiveFameResponseBody(l)(fromName, amount))
			}
		}
	}
}

func giveFame(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(toName string, amount int8, total int16) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(toName string, amount int8, total int16) model.Operator[session.Model] {
		return func(wp writer.Producer) func(toName string, amount int8, total int16) model.Operator[session.Model] {
			return func(toName string, amount int8, total int16) model.Operator[session.Model] {
				return session.Announce(l)(ctx)(wp)(writer.FameResponse)(writer.GiveFameResponseBody(l)(toName, amount, total))
			}
		}
	}
}

func handleStatusEventMesoChanged(sc server.Model, wp writer.Producer) message.Handler[statusEvent[mesoChangedStatusEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e statusEvent[mesoChangedStatusEventBody]) {
		if e.Type != StatusEventTypeMesoChanged {
			return
		}

		if !sc.IsWorld(tenant.MustFromContext(ctx), world.Id(e.WorldId)) {
			return
		}

		err := session.IfPresentByCharacterId(sc.Tenant(), sc.WorldId(), sc.ChannelId())(e.CharacterId, mesoChanged(l)(ctx)(wp)(e.Body.Amount))
		if err != nil {
			l.WithError(err).Errorf("Unable to notify character [%d] they received meso [%d].", e.CharacterId, e.Body.Amount)
		}
	}
}

func mesoChanged(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(amount int32) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(amount int32) model.Operator[session.Model] {
		return func(wp writer.Producer) func(amount int32) model.Operator[session.Model] {
			return func(amount int32) model.Operator[session.Model] {
				return session.Announce(l)(ctx)(wp)(writer.CharacterStatusMessage)(writer.CharacterStatusMessageOperationIncreaseMesoBody(l)(amount))
			}
		}
	}
}

func handleStatusEventLevelChanged(sc server.Model, wp writer.Producer) message.Handler[statusEvent[levelChangedStatusEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e statusEvent[levelChangedStatusEventBody]) {
		if e.Type != StatusEventTypeLevelChanged {
			return
		}

		if !sc.Is(tenant.MustFromContext(ctx), world.Id(e.WorldId), channel.Id(e.Body.ChannelId)) {
			return
		}

		session.IfPresentByCharacterId(sc.Tenant(), sc.WorldId(), sc.ChannelId())(e.CharacterId, func(s session.Model) error {
			_ = session.Announce(l)(ctx)(wp)(writer.CharacterEffect)(writer.CharacterLevelUpEffectBody(l)())(s)
			_ = _map.ForOtherSessionsInMap(l)(ctx)(s.Map(), s.CharacterId(), session.Announce(l)(ctx)(wp)(writer.CharacterEffectForeign)(writer.CharacterLevelUpEffectForeignBody(l)(s.CharacterId())))
			return nil
		})

	}
}

func handleMovementEvent(sc server.Model, wp writer.Producer) message.Handler[movementEvent] {
	return func(l logrus.FieldLogger, ctx context.Context, e movementEvent) {
		if !sc.Is(tenant.MustFromContext(ctx), world.Id(e.WorldId), channel.Id(e.ChannelId)) {
			return
		}

		mv := movement.ProduceMovementForSocket(e.Movement)
		err := _map.ForOtherSessionsInMap(l)(ctx)(sc.Map(_map2.Id(e.MapId)), e.CharacterId, showMovementForSession(l)(ctx)(wp)(e.CharacterId, *mv))
		if err != nil {
			l.WithError(err).Errorf("Unable to move character [%d] for characters in map [%d].", e.CharacterId, e.MapId)
		}
	}
}

func showMovementForSession(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(characterId uint32, m model2.Movement) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(characterId uint32, m model2.Movement) model.Operator[session.Model] {
		return func(wp writer.Producer) func(characterId uint32, m model2.Movement) model.Operator[session.Model] {
			return func(characterId uint32, m model2.Movement) model.Operator[session.Model] {
				return session.Announce(l)(ctx)(wp)(writer.CharacterMovement)(writer.CharacterMovementBody(l, tenant.MustFromContext(ctx))(characterId, m))
			}
		}
	}
}
