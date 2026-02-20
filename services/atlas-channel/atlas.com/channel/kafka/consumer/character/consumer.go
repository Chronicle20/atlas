package character

import (
	"atlas-channel/character"
	consumer2 "atlas-channel/kafka/consumer"
	character2 "atlas-channel/kafka/message/character"
	_map "atlas-channel/map"
	"atlas-channel/party"
	"atlas-channel/server"
	"atlas-channel/session"
	model2 "atlas-channel/socket/model"
	"atlas-channel/socket/writer"
	"context"

	"github.com/Chronicle20/atlas-constants/field"
	"github.com/Chronicle20/atlas-constants/stat"
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
			rf(consumer2.NewConfig(l)("character_status_event")(character2.EnvEventTopicCharacterStatus)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser), consumer.SetStartOffset(kafka.LastOffset))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) error {
	return func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) error {
		return func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) error {
			return func(rf func(topic string, handler handler.Handler) (string, error)) error {
				var t string
				t, _ = topic.EnvProvider(l)(character2.EnvEventTopicCharacterStatus)()
				if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventStatChanged(sc, wp)))); err != nil {
					return err
				}
				if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventMapChanged(sc, wp)))); err != nil {
					return err
				}
				if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventExperienceChanged(sc, wp)))); err != nil {
					return err
				}
				if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventFameChanged(sc, wp)))); err != nil {
					return err
				}
				if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventMesoChanged(sc, wp)))); err != nil {
					return err
				}
				if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventLevelChanged(sc, wp)))); err != nil {
					return err
				}
				return nil
			}
		}
	}
}

func handleStatusEventStatChanged(sc server.Model, wp writer.Producer) func(l logrus.FieldLogger, ctx context.Context, event character2.StatusEvent[character2.StatusEventStatChangedBody]) {
	return func(l logrus.FieldLogger, ctx context.Context, e character2.StatusEvent[character2.StatusEventStatChangedBody]) {
		if e.Type != character2.StatusEventTypeStatChanged {
			return
		}

		if !sc.IsWorld(tenant.MustFromContext(ctx), e.WorldId) {
			return
		}

		c, err := character.NewProcessor(l, ctx).GetById()(e.CharacterId)
		if err != nil {
			return
		}

		err = session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(e.CharacterId, statChanged(l)(ctx)(wp)(c, e.Body.ExclRequestSent, e.Body.Updates))
		if err != nil {
			l.WithError(err).Errorf("Unable to issue stat changed to character [%d].", e.CharacterId)
		}

		var hpChange bool
		for _, u := range e.Body.Updates {
			if u == stat.TypeHp || u == stat.TypeMaxHp {
				hpChange = true
			}
		}
		if hpChange {
			// TODO - field migration this seems like a bug not using instance
			f := field.NewBuilder(e.WorldId, e.Body.ChannelId, c.MapId()).Build()
			imf := party.OtherMemberInMap(f, c.Id())
			oip := party.MemberToMemberIdMapper(party.FilteredMemberProvider(imf)(party.NewProcessor(l, ctx).ByMemberIdProvider(e.CharacterId)))
			err = session.NewProcessor(l, ctx).ForEachByCharacterId(sc.Channel())(oip, session.Announce(l)(ctx)(wp)(writer.PartyMemberHP)(writer.PartyMemberHPBody(c.Id(), c.Hp(), c.MaxHp())))
			if err != nil {
				l.WithError(err).Errorf("Unable to announce character [%d] health to party members.", c.Id())
			}
		}
	}
}

func statChanged(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(c character.Model, exclRequestSent bool, updates []stat.Type) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(c character.Model, exclRequestSent bool, updates []stat.Type) model.Operator[session.Model] {
		return func(wp writer.Producer) func(c character.Model, exclRequestSent bool, updates []stat.Type) model.Operator[session.Model] {
			return func(c character.Model, exclRequestSent bool, updates []stat.Type) model.Operator[session.Model] {
				return func(s session.Model) error {
					var su = make([]model2.StatUpdate, 0)
					for _, update := range updates {
						value := int64(0)
						if update == stat.TypeSkin {
							value = int64(c.SkinColor())
						} else if update == stat.TypeFace {
							value = int64(c.Face())
						} else if update == stat.TypeHair {
							value = int64(c.Hair())
						} else if update == stat.TypePetSn1 {
							if len(c.Pets()) > 0 {
								value = int64(c.Pets()[0].Id())
							} else {
								value = int64(0)
							}
						} else if update == stat.TypeLevel {
							value = int64(c.Level())
						} else if update == stat.TypeJob {
							value = int64(c.JobId())
						} else if update == stat.TypeStrength {
							value = int64(c.Strength())
						} else if update == stat.TypeDexterity {
							value = int64(c.Dexterity())
						} else if update == stat.TypeIntelligence {
							value = int64(c.Intelligence())
						} else if update == stat.TypeLuck {
							value = int64(c.Luck())
						} else if update == stat.TypeHp {
							value = int64(c.Hp())
						} else if update == stat.TypeMaxHp {
							value = int64(c.MaxHp())
						} else if update == stat.TypeMp {
							value = int64(c.Mp())
						} else if update == stat.TypeMaxMp {
							value = int64(c.MaxMp())
						} else if update == stat.TypeAvailableAP {
							value = int64(c.Ap())
						} else if update == stat.TypeAvailableSP {
							value = int64(c.Sp()[0])
						} else if update == stat.TypeExperience {
							value = int64(c.Experience())
						} else if update == stat.TypeFame {
							value = int64(c.Fame())
						} else if update == stat.TypeMeso {
							value = int64(c.Meso())
						} else if update == stat.TypePetSn2 {
							if len(c.Pets()) > 1 {
								value = int64(c.Pets()[1].Id())
							} else {
								value = int64(0)
							}
						} else if update == stat.TypePetSn3 {
							if len(c.Pets()) > 2 {
								value = int64(c.Pets()[2].Id())
							} else {
								value = int64(0)
							}
						} else if update == stat.TypeGachaponExperience {
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

func handleStatusEventMapChanged(sc server.Model, wp writer.Producer) func(l logrus.FieldLogger, ctx context.Context, event character2.StatusEvent[character2.StatusEventMapChangedBody]) {
	return func(l logrus.FieldLogger, ctx context.Context, event character2.StatusEvent[character2.StatusEventMapChangedBody]) {
		if event.Type != character2.StatusEventTypeMapChanged {
			return
		}

		if !sc.Is(tenant.MustFromContext(ctx), event.WorldId, event.Body.ChannelId) {
			return
		}

		session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(event.CharacterId, warpCharacter(l)(ctx)(wp)(event))
	}
}

func warpCharacter(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(event character2.StatusEvent[character2.StatusEventMapChangedBody]) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(event character2.StatusEvent[character2.StatusEventMapChangedBody]) model.Operator[session.Model] {
		t := tenant.MustFromContext(ctx)
		return func(wp writer.Producer) func(event character2.StatusEvent[character2.StatusEventMapChangedBody]) model.Operator[session.Model] {
			return func(event character2.StatusEvent[character2.StatusEventMapChangedBody]) model.Operator[session.Model] {
				return func(s session.Model) error {
					c, err := character.NewProcessor(l, ctx).GetById()(s.CharacterId())
					if err != nil {
						l.WithError(err).Errorf("Unable to retrieve character [%d].", s.CharacterId())
						return err
					}

					targetField := field.NewBuilder(event.WorldId, event.Body.ChannelId, event.Body.TargetMapId).SetInstance(event.Body.TargetInstance).Build()
					s = session.NewProcessor(l, ctx).SetField(s.SessionId(), targetField)

					err = session.Announce(l)(ctx)(wp)(writer.SetField)(writer.WarpToMapBody(t)(s.ChannelId(), event.Body.TargetMapId, event.Body.TargetPortalId, c.Hp()))(s)
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

func handleStatusEventExperienceChanged(sc server.Model, wp writer.Producer) message.Handler[character2.StatusEvent[character2.ExperienceChangedStatusEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e character2.StatusEvent[character2.ExperienceChangedStatusEventBody]) {
		if e.Type != character2.StatusEventTypeExperienceChanged {
			return
		}

		if !sc.Is(tenant.MustFromContext(ctx), e.WorldId, e.Body.ChannelId) {
			return
		}

		session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(e.CharacterId, announceExperienceGain(l)(ctx)(wp)(e.Body.Distributions))
	}
}

func announceExperienceGain(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(distributions []character2.ExperienceDistributions) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(distributions []character2.ExperienceDistributions) model.Operator[session.Model] {
		t := tenant.MustFromContext(ctx)
		return func(wp writer.Producer) func(distributions []character2.ExperienceDistributions) model.Operator[session.Model] {
			return func(distributions []character2.ExperienceDistributions) model.Operator[session.Model] {
				return func(s session.Model) error {
					c := model2.IncreaseExperienceConfig{}

					for _, d := range distributions {
						if d.ExperienceType == character2.ExperienceDistributionTypeWhite {
							c.White = true
							c.Amount = int32(d.Amount)
						} else if d.ExperienceType == character2.ExperienceDistributionTypeYellow {
							c.White = false
							c.Amount = int32(d.Amount)
						} else if d.ExperienceType == character2.ExperienceDistributionTypeChat {
							c.InChat = true
							c.Amount = int32(d.Amount)
						} else if d.ExperienceType == character2.ExperienceDistributionTypeMonsterBook {
							c.MonsterBookBonus = int32(d.Amount)
						} else if d.ExperienceType == character2.ExperienceDistributionTypeMonsterEvent {
							c.MobEventBonusPercentage = byte(d.Amount)
						} else if d.ExperienceType == character2.ExperienceDistributionTypePlayTime {
							c.MobEventBonusPercentage = byte(d.Amount)
							c.PlayTimeHour = byte(d.Attr1)
						} else if d.ExperienceType == character2.ExperienceDistributionTypeWedding {
							c.WeddingBonusEXP = int32(d.Amount)
						} else if d.ExperienceType == character2.ExperienceDistributionTypeSpiritWeek {
							c.QuestBonusRate = byte(d.Amount)
						} else if d.ExperienceType == character2.ExperienceDistributionTypeParty {
							c.PartyBonusExp = int32(d.Amount)
							c.PartyBonusEventRate = byte(d.Attr1)
						} else if d.ExperienceType == character2.ExperienceDistributionTypeItem {
							c.ItemBonusEXP = int32(d.Amount)
						} else if d.ExperienceType == character2.ExperienceDistributionTypeInternetCafe {
							c.PremiumIPExp = int32(d.Amount)
						} else if d.ExperienceType == character2.ExperienceDistributionTypeRainbowWeek {
							c.RainbowWeekEventEXP = int32(d.Amount)
						} else if d.ExperienceType == character2.ExperienceDistributionTypePartyRing {
							c.PartyEXPRingEXP = int32(d.Amount)
						} else if d.ExperienceType == character2.ExperienceDistributionTypeCakePie {
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

func handleStatusEventFameChanged(sc server.Model, wp writer.Producer) message.Handler[character2.StatusEvent[character2.FameChangedStatusEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e character2.StatusEvent[character2.FameChangedStatusEventBody]) {
		if e.Type != character2.StatusEventTypeFameChanged {
			return
		}

		if !sc.IsWorld(tenant.MustFromContext(ctx), e.WorldId) {
			return
		}

		cp := character.NewProcessor(l, ctx)
		c, err := cp.GetById()(e.CharacterId)
		if err != nil {
			return
		}

		if e.Body.ActorType == character2.StatusEventActorTypeCharacter {
			ac, err := cp.GetById()(e.Body.ActorId)
			if err != nil {
				return
			}

			err = session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(e.CharacterId, receiveFame(l)(ctx)(wp)(ac.Name(), e.Body.Amount))
			if err != nil {
				l.WithError(err).Errorf("Unable to notify character [%d] they received fame [%d] from [%s].", e.CharacterId, e.Body.Amount, ac.Name())
			}
			err = session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(e.Body.ActorId, giveFame(l)(ctx)(wp)(c.Name(), e.Body.Amount, c.Fame()))
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

func handleStatusEventMesoChanged(sc server.Model, wp writer.Producer) message.Handler[character2.StatusEvent[character2.MesoChangedStatusEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e character2.StatusEvent[character2.MesoChangedStatusEventBody]) {
		if e.Type != character2.StatusEventTypeMesoChanged {
			return
		}

		if !sc.IsWorld(tenant.MustFromContext(ctx), e.WorldId) {
			return
		}

		err := session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(e.CharacterId, mesoChanged(l)(ctx)(wp)(e.Body.Amount))
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

func handleStatusEventLevelChanged(sc server.Model, wp writer.Producer) message.Handler[character2.StatusEvent[character2.LevelChangedStatusEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e character2.StatusEvent[character2.LevelChangedStatusEventBody]) {
		if e.Type != character2.StatusEventTypeLevelChanged {
			return
		}

		if !sc.Is(tenant.MustFromContext(ctx), e.WorldId, e.Body.ChannelId) {
			return
		}

		session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(e.CharacterId, func(s session.Model) error {
			_ = session.Announce(l)(ctx)(wp)(writer.CharacterEffect)(writer.CharacterLevelUpEffectBody(l)())(s)
			_ = _map.NewProcessor(l, ctx).ForOtherSessionsInMap(s.Field(), s.CharacterId(), session.Announce(l)(ctx)(wp)(writer.CharacterEffectForeign)(writer.CharacterLevelUpEffectForeignBody(l)(s.CharacterId())))
			return nil
		})

	}
}
