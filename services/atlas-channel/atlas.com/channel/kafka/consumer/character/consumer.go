package character

import (
	"atlas-channel/character"
	consumer2 "atlas-channel/kafka/consumer"
	_map "atlas-channel/map"
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
	"github.com/sirupsen/logrus"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("character_status_event")(EnvEventTopicCharacterStatus)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
			rf(consumer2.NewConfig(l)("character_movement_event")(EnvEventTopicMovement)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
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
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventFameChanged(sc, wp))))
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventMesoChanged(sc, wp))))
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

		t := sc.Tenant()
		if !t.Is(tenant.MustFromContext(ctx)) {
			return
		}

		if sc.WorldId() != e.WorldId {
			return
		}

		session.IfPresentByCharacterId(sc.Tenant(), sc.WorldId(), sc.ChannelId())(e.CharacterId, statChanged(l)(ctx)(wp)(e.Body.ExclRequestSent, e.Body.Updates))
	}
}

func statChanged(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(exclRequestSent bool, updates []string) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(exclRequestSent bool, updates []string) model.Operator[session.Model] {
		return func(wp writer.Producer) func(exclRequestSent bool, updates []string) model.Operator[session.Model] {
			return func(exclRequestSent bool, updates []string) model.Operator[session.Model] {
				return func(s session.Model) error {
					c, err := character.GetById(l)(ctx)()(s.CharacterId())
					if err != nil {
						return err
					}
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
							value = int64(0)
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
							value = int64(0)
						} else if update == writer.StatPetSn3 {
							value = int64(0)
						} else if update == writer.StatGachaponExperience {
							value = int64(c.GachaponExperience())
						}
						su = append(su, model2.NewStatUpdate(update, value))
					}

					err = session.Announce(l)(ctx)(wp)(writer.StatChanged)(s, writer.StatChangedBody(l)(su, exclRequestSent))
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

		if !sc.Is(tenant.MustFromContext(ctx), event.WorldId, event.Body.ChannelId) {
			return
		}

		session.IfPresentByCharacterId(sc.Tenant(), sc.WorldId(), sc.ChannelId())(event.CharacterId, warpCharacter(l)(ctx)(wp)(event))
	}
}

func warpCharacter(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(event statusEvent[statusEventMapChangedBody]) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(event statusEvent[statusEventMapChangedBody]) model.Operator[session.Model] {
		t := tenant.MustFromContext(ctx)
		return func(wp writer.Producer) func(event statusEvent[statusEventMapChangedBody]) model.Operator[session.Model] {
			setFieldFunc := session.Announce(l)(ctx)(wp)(writer.SetField)
			return func(event statusEvent[statusEventMapChangedBody]) model.Operator[session.Model] {
				return func(s session.Model) error {
					c, err := character.GetById(l)(ctx)()(s.CharacterId())
					if err != nil {
						l.WithError(err).Errorf("Unable to retrieve character [%d].", s.CharacterId())
						return err
					}

					s = session.SetMapId(event.Body.TargetMapId)(t.Id(), s.SessionId())

					err = setFieldFunc(s, writer.WarpToMapBody(l, t)(s.ChannelId(), event.Body.TargetMapId, event.Body.TargetPortalId, c.Hp()))
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

func handleStatusEventFameChanged(sc server.Model, wp writer.Producer) message.Handler[statusEvent[fameChangedStatusEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e statusEvent[fameChangedStatusEventBody]) {
		if e.Type != StatusEventTypeFameChanged {
			return
		}

		t := sc.Tenant()
		if !t.Is(tenant.MustFromContext(ctx)) {
			return
		}

		if sc.WorldId() != e.WorldId {
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

			session.IfPresentByCharacterId(sc.Tenant(), sc.WorldId(), sc.ChannelId())(e.CharacterId, receiveFame(l)(ctx)(wp)(ac.Name(), e.Body.Amount))
			session.IfPresentByCharacterId(sc.Tenant(), sc.WorldId(), sc.ChannelId())(e.Body.ActorId, giveFame(l)(ctx)(wp)(c.Name(), e.Body.Amount, c.Fame()))
		}
	}
}

func receiveFame(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(fromName string, amount int8) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(fromName string, amount int8) model.Operator[session.Model] {
		return func(wp writer.Producer) func(fromName string, amount int8) model.Operator[session.Model] {
			fameResponseFunc := session.Announce(l)(ctx)(wp)(writer.FameResponse)
			return func(fromName string, amount int8) model.Operator[session.Model] {
				return func(s session.Model) error {
					err := fameResponseFunc(s, writer.ReceiveFameResponseBody(l)(fromName, amount))
					if err != nil {
						l.WithError(err).Errorf("Unable to notify character [%d] they received fame [%d] from [%s].", s.CharacterId(), amount, fromName)
						return err
					}
					return nil
				}
			}
		}
	}
}

func giveFame(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(toName string, amount int8, total int16) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(toName string, amount int8, total int16) model.Operator[session.Model] {
		return func(wp writer.Producer) func(toName string, amount int8, total int16) model.Operator[session.Model] {
			fameResponseFunc := session.Announce(l)(ctx)(wp)(writer.FameResponse)
			return func(toName string, amount int8, total int16) model.Operator[session.Model] {
				return func(s session.Model) error {
					err := fameResponseFunc(s, writer.GiveFameResponseBody(l)(toName, amount, total))
					if err != nil {
						l.WithError(err).Errorf("Unable to notify character [%d] they received fame [%d] from [%s].", s.CharacterId(), amount, toName)
						return err
					}
					return nil
				}
			}
		}
	}
}

func handleStatusEventMesoChanged(sc server.Model, wp writer.Producer) message.Handler[statusEvent[mesoChangedStatusEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e statusEvent[mesoChangedStatusEventBody]) {
		if e.Type != StatusEventTypeMesoChanged {
			return
		}

		t := sc.Tenant()
		if !t.Is(tenant.MustFromContext(ctx)) {
			return
		}

		if sc.WorldId() != e.WorldId {
			return
		}

		session.IfPresentByCharacterId(sc.Tenant(), sc.WorldId(), sc.ChannelId())(e.CharacterId, mesoChanged(l)(ctx)(wp)(e.Body.Amount))
	}
}

func mesoChanged(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(amount int32) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(amount int32) model.Operator[session.Model] {
		return func(wp writer.Producer) func(amount int32) model.Operator[session.Model] {
			characterStatusMessageFunc := session.Announce(l)(ctx)(wp)(writer.CharacterStatusMessage)
			return func(amount int32) model.Operator[session.Model] {
				return func(s session.Model) error {
					err := characterStatusMessageFunc(s, writer.CharacterStatusMessageOperationIncreaseMesoBody(l)(amount))
					if err != nil {
						l.WithError(err).Errorf("Unable to notify character [%d] they received meso [%d].", s.CharacterId(), amount)
						return err
					}
					return nil
				}
			}
		}
	}
}

func handleMovementEvent(sc server.Model, wp writer.Producer) message.Handler[movementEvent] {
	return func(l logrus.FieldLogger, ctx context.Context, event movementEvent) {
		if !sc.Is(tenant.MustFromContext(ctx), event.WorldId, event.ChannelId) {
			return
		}

		mv := model2.Movement{StartX: event.Movement.StartX, StartY: event.Movement.StartY}
		for _, elem := range event.Movement.Elements {
			if elem.TypeStr == MovementTypeNormal {
				mv.Elements = append(mv.Elements, &model2.NormalElement{
					Element: model2.Element{
						StartX:      elem.StartX,
						StartY:      elem.StartY,
						BMoveAction: elem.MoveAction,
						BStat:       elem.Stat,
						X:           elem.X,
						Y:           elem.Y,
						Vx:          elem.VX,
						Vy:          elem.VY,
						Fh:          elem.FH,
						FhFallStart: elem.FHFallStart,
						XOffset:     elem.XOffset,
						YOffset:     elem.YOffset,
						TElapse:     elem.TimeElapsed,
						ElemType:    elem.TypeVal,
					},
				})
			} else if elem.TypeStr == MovementTypeTeleport {
				mv.Elements = append(mv.Elements, &model2.TeleportElement{
					Element: model2.Element{
						StartX:      elem.StartX,
						StartY:      elem.StartY,
						BMoveAction: elem.MoveAction,
						BStat:       elem.Stat,
						X:           elem.X,
						Y:           elem.Y,
						Vx:          elem.VX,
						Vy:          elem.VY,
						Fh:          elem.FH,
						FhFallStart: elem.FHFallStart,
						XOffset:     elem.XOffset,
						YOffset:     elem.YOffset,
						TElapse:     elem.TimeElapsed,
						ElemType:    elem.TypeVal,
					},
				})
			} else if elem.TypeStr == MovementTypeStartFallDown {
				mv.Elements = append(mv.Elements, &model2.StartFallDownElement{
					Element: model2.Element{
						StartX:      elem.StartX,
						StartY:      elem.StartY,
						BMoveAction: elem.MoveAction,
						BStat:       elem.Stat,
						X:           elem.X,
						Y:           elem.Y,
						Vx:          elem.VX,
						Vy:          elem.VY,
						Fh:          elem.FH,
						FhFallStart: elem.FHFallStart,
						XOffset:     elem.XOffset,
						YOffset:     elem.YOffset,
						TElapse:     elem.TimeElapsed,
						ElemType:    elem.TypeVal,
					},
				})
			} else if elem.TypeStr == MovementTypeFlyingBlock {
				mv.Elements = append(mv.Elements, &model2.FlyingBlockElement{
					Element: model2.Element{
						StartX:      elem.StartX,
						StartY:      elem.StartY,
						BMoveAction: elem.MoveAction,
						BStat:       elem.Stat,
						X:           elem.X,
						Y:           elem.Y,
						Vx:          elem.VX,
						Vy:          elem.VY,
						Fh:          elem.FH,
						FhFallStart: elem.FHFallStart,
						XOffset:     elem.XOffset,
						YOffset:     elem.YOffset,
						TElapse:     elem.TimeElapsed,
						ElemType:    elem.TypeVal,
					},
				})
			} else if elem.TypeStr == MovementTypeJump {
				mv.Elements = append(mv.Elements, &model2.JumpElement{
					Element: model2.Element{
						StartX:      elem.StartX,
						StartY:      elem.StartY,
						BMoveAction: elem.MoveAction,
						BStat:       elem.Stat,
						X:           elem.X,
						Y:           elem.Y,
						Vx:          elem.VX,
						Vy:          elem.VY,
						Fh:          elem.FH,
						FhFallStart: elem.FHFallStart,
						XOffset:     elem.XOffset,
						YOffset:     elem.YOffset,
						TElapse:     elem.TimeElapsed,
						ElemType:    elem.TypeVal,
					},
				})
			} else if elem.TypeStr == MovementTypeStatChange {
				mv.Elements = append(mv.Elements, &model2.StatChangeElement{
					Element: model2.Element{
						StartX:      elem.StartX,
						StartY:      elem.StartY,
						BMoveAction: elem.MoveAction,
						BStat:       elem.Stat,
						X:           elem.X,
						Y:           elem.Y,
						Vx:          elem.VX,
						Vy:          elem.VY,
						Fh:          elem.FH,
						FhFallStart: elem.FHFallStart,
						XOffset:     elem.XOffset,
						YOffset:     elem.YOffset,
						TElapse:     elem.TimeElapsed,
						ElemType:    elem.TypeVal,
					},
				})
			} else {
				mv.Elements = append(mv.Elements, &model2.Element{
					StartX:      elem.StartX,
					StartY:      elem.StartY,
					BMoveAction: elem.MoveAction,
					BStat:       elem.Stat,
					X:           elem.X,
					Y:           elem.Y,
					Vx:          elem.VX,
					Vy:          elem.VY,
					Fh:          elem.FH,
					FhFallStart: elem.FHFallStart,
					XOffset:     elem.XOffset,
					YOffset:     elem.YOffset,
					TElapse:     elem.TimeElapsed,
					ElemType:    elem.TypeVal,
				})
			}
		}

		_map.ForOtherSessionsInMap(l)(ctx)(sc.WorldId(), sc.ChannelId(), event.MapId, event.CharacterId, showMovementForSession(l)(ctx)(wp)(event.CharacterId, mv))
	}
}

func showMovementForSession(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(characterId uint32, m model2.Movement) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(characterId uint32, m model2.Movement) model.Operator[session.Model] {
		return func(wp writer.Producer) func(characterId uint32, m model2.Movement) model.Operator[session.Model] {
			moveCharacterFunc := session.Announce(l)(ctx)(wp)(writer.CharacterMovement)
			return func(characterId uint32, m model2.Movement) model.Operator[session.Model] {
				return func(s session.Model) error {
					err := moveCharacterFunc(s, writer.CharacterMovementBody(l, tenant.MustFromContext(ctx))(characterId, m))
					if err != nil {
						l.WithError(err).Errorf("Unable to move character [%d] for character [%d].", characterId, s.CharacterId())
					}
					return err
				}
			}
		}
	}
}
