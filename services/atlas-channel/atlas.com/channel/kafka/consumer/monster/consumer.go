package monster

import (
	consumer2 "atlas-channel/kafka/consumer"
	_map "atlas-channel/map"
	"atlas-channel/monster"
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
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("monster_status_event")(EnvEventTopicStatus)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
			rf(consumer2.NewConfig(l)("monster_movement_event")(EnvEventTopicMovement)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) {
		return func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) {
			return func(rf func(topic string, handler handler.Handler) (string, error)) {
				var t string
				t, _ = topic.EnvProvider(l)(EnvEventTopicStatus)()
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventCreated(sc, wp))))
				t, _ = topic.EnvProvider(l)(EnvEventTopicStatus)()
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventDestroyed(sc, wp))))
				t, _ = topic.EnvProvider(l)(EnvEventTopicStatus)()
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventKilled(sc, wp))))
				t, _ = topic.EnvProvider(l)(EnvEventTopicStatus)()
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventStartControl(sc, wp))))
				t, _ = topic.EnvProvider(l)(EnvEventTopicStatus)()
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventStopControl(sc, wp))))
				t, _ = topic.EnvProvider(l)(EnvEventTopicMovement)()
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleMovementEvent(sc, wp))))
			}
		}
	}
}

func handleStatusEventCreated(sc server.Model, wp writer.Producer) message.Handler[statusEvent[statusEventCreatedBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, event statusEvent[statusEventCreatedBody]) {
		if !sc.Is(tenant.MustFromContext(ctx), event.WorldId, event.ChannelId) {
			return
		}

		if event.Type != EventStatusCreated {
			return
		}

		m, err := monster.GetById(l)(ctx)(event.UniqueId)
		if err != nil {
			l.WithError(err).Errorf("Unable to retrieve the monster [%d] being spawned.", event.UniqueId)
			return
		}

		_map.ForSessionsInMap(l)(ctx)(event.WorldId, event.ChannelId, event.MapId, spawnForSession(l)(ctx)(wp)(m))
	}
}

func spawnForSession(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(m monster.Model) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(m monster.Model) model.Operator[session.Model] {
		return func(wp writer.Producer) func(m monster.Model) model.Operator[session.Model] {
			spawnMonsterFunc := session.Announce(l)(ctx)(wp)(writer.SpawnMonster)
			return func(m monster.Model) model.Operator[session.Model] {
				return func(s session.Model) error {
					l.Debugf("Spawning [%d] monster [%d] for character [%d].", m.MonsterId(), m.UniqueId(), s.CharacterId())
					err := spawnMonsterFunc(s, writer.SpawnMonsterBody(l, tenant.MustFromContext(ctx))(m, false))
					if err != nil {
						l.WithError(err).Errorf("Unable to spawn monster [%d] for character [%d].", m.UniqueId(), s.CharacterId())
					}
					return err
				}
			}
		}
	}
}

func handleStatusEventDestroyed(sc server.Model, wp writer.Producer) message.Handler[statusEvent[statusEventDestroyedBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, event statusEvent[statusEventDestroyedBody]) {
		if !sc.Is(tenant.MustFromContext(ctx), event.WorldId, event.ChannelId) {
			return
		}

		if event.Type != EventStatusDestroyed {
			return
		}

		_map.ForSessionsInMap(l)(ctx)(event.WorldId, event.ChannelId, event.MapId, destroyForSession(l)(ctx)(wp)(event.UniqueId))
	}
}

func destroyForSession(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(uniqueId uint32) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(uniqueId uint32) model.Operator[session.Model] {
		return func(wp writer.Producer) func(uniqueId uint32) model.Operator[session.Model] {
			destroyMonsterFunc := session.Announce(l)(ctx)(wp)(writer.DestroyMonster)
			return func(uniqueId uint32) model.Operator[session.Model] {
				return func(s session.Model) error {
					err := destroyMonsterFunc(s, writer.DestroyMonsterBody(l, tenant.MustFromContext(ctx))(uniqueId, writer.DestroyMonsterTypeFadeOut))
					if err != nil {
						l.WithError(err).Errorf("Unable to destroy monster [%d] for character [%d].", uniqueId, s.CharacterId())
					}
					return err
				}
			}
		}
	}
}

func handleStatusEventKilled(sc server.Model, wp writer.Producer) message.Handler[statusEvent[statusEventKilledBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, event statusEvent[statusEventKilledBody]) {
		if !sc.Is(tenant.MustFromContext(ctx), event.WorldId, event.ChannelId) {
			return
		}

		if event.Type != EventStatusKilled {
			return
		}

		_map.ForSessionsInMap(l)(ctx)(event.WorldId, event.ChannelId, event.MapId, killForSession(l)(ctx)(wp)(event.UniqueId))
	}
}

func killForSession(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(uniqueId uint32) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(uniqueId uint32) model.Operator[session.Model] {
		return func(wp writer.Producer) func(uniqueId uint32) model.Operator[session.Model] {
			destroyMonsterFunc := session.Announce(l)(ctx)(wp)(writer.DestroyMonster)
			return func(uniqueId uint32) model.Operator[session.Model] {
				return func(s session.Model) error {
					err := destroyMonsterFunc(s, writer.DestroyMonsterBody(l, tenant.MustFromContext(ctx))(uniqueId, writer.DestroyMonsterTypeFadeOut))
					if err != nil {
						l.WithError(err).Errorf("Unable to kill monster [%d] for character [%d].", uniqueId, s.CharacterId())
					}
					return err
				}
			}
		}
	}
}

func handleStatusEventStartControl(sc server.Model, wp writer.Producer) message.Handler[statusEvent[statusEventStartControlBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, event statusEvent[statusEventStartControlBody]) {
		if !sc.Is(tenant.MustFromContext(ctx), event.WorldId, event.ChannelId) {
			return
		}

		if event.Type != EventStatusStartControl {
			return
		}

		m, err := monster.GetById(l)(ctx)(event.UniqueId)
		if err != nil {
			l.WithError(err).Errorf("Unable to retrieve the monster [%d] being controlled.", event.UniqueId)
			return
		}

		session.IfPresentByCharacterId(tenant.MustFromContext(ctx), sc.WorldId(), sc.ChannelId())(event.Body.ActorId, startControlForSession(l)(ctx)(wp)(m))
	}
}

func startControlForSession(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(m monster.Model) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(m monster.Model) model.Operator[session.Model] {
		return func(wp writer.Producer) func(m monster.Model) model.Operator[session.Model] {
			controlMonsterFunc := session.Announce(l)(ctx)(wp)(writer.ControlMonster)
			return func(m monster.Model) model.Operator[session.Model] {
				return func(s session.Model) error {
					l.Debugf("Starting control of [%d] monster [%d] for character [%d].", m.MonsterId(), m.UniqueId(), s.CharacterId())
					err := controlMonsterFunc(s, writer.StartControlMonsterBody(l, tenant.MustFromContext(ctx))(m, false))
					if err != nil {
						l.WithError(err).Errorf("Unable to control monster [%d] for character [%d].", m.UniqueId(), s.CharacterId())
					}
					return err
				}
			}
		}
	}
}

func handleStatusEventStopControl(sc server.Model, wp writer.Producer) message.Handler[statusEvent[statusEventStopControlBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, event statusEvent[statusEventStopControlBody]) {
		if !sc.Is(tenant.MustFromContext(ctx), event.WorldId, event.ChannelId) {
			return
		}

		if event.Type != EventStatusStopControl {
			return
		}

		m, err := monster.GetById(l)(ctx)(event.UniqueId)
		if err != nil {
			l.WithError(err).Errorf("Unable to retrieve the monster [%d] being controlled.", event.UniqueId)
			return
		}

		session.IfPresentByCharacterId(tenant.MustFromContext(ctx), sc.WorldId(), sc.ChannelId())(event.Body.ActorId, stopControlForSession(l)(ctx)(wp)(m))
	}
}

func stopControlForSession(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(m monster.Model) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(m monster.Model) model.Operator[session.Model] {
		return func(wp writer.Producer) func(m monster.Model) model.Operator[session.Model] {
			controlMonsterFunc := session.Announce(l)(ctx)(wp)(writer.ControlMonster)
			return func(m monster.Model) model.Operator[session.Model] {
				return func(s session.Model) error {
					l.Debugf("Stopping control of [%d] monster [%d] for character [%d].", m.MonsterId(), m.UniqueId(), s.CharacterId())
					err := controlMonsterFunc(s, writer.StopControlMonsterBody(l, tenant.MustFromContext(ctx))(m))
					if err != nil {
						l.WithError(err).Errorf("Unable to control monster [%d] for character [%d].", m.UniqueId(), s.CharacterId())
					}
					return err
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

		m, err := monster.GetById(l)(ctx)(event.UniqueId)
		if err != nil {
			l.WithError(err).Errorf("Unable to retrieve the monster [%d] moving.", event.UniqueId)
			return
		}

		_map.ForOtherSessionsInMap(l)(ctx)(event.WorldId, event.ChannelId, m.MapId(), event.ObserverId, showMovementForSession(l)(ctx)(wp)(event))
	}
}

func showMovementForSession(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(event movementEvent) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(event movementEvent) model.Operator[session.Model] {
		return func(wp writer.Producer) func(event movementEvent) model.Operator[session.Model] {
			moveMonsterFunc := session.Announce(l)(ctx)(wp)(writer.MoveMonster)
			return func(event movementEvent) model.Operator[session.Model] {
				return func(s session.Model) error {
					l.Debugf("Writing monster [%d] movement for session [%s].", event.UniqueId, s.SessionId().String())
					mt := model2.MultiTargetForBall{}
					for _, t := range event.MultiTarget {
						mt.Targets = append(mt.Targets, model2.NewPosition(t.X, t.Y))
					}
					ra := model2.RandTimeForAreaAttack{}
					for _, t := range event.RandomTimes {
						ra.Times = append(ra.Times, t)
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

					err := moveMonsterFunc(s, writer.MoveMonsterBody(l, tenant.MustFromContext(ctx))(event.UniqueId,
						false, event.SkillPossible, false, event.Skill, event.SkillId,
						event.SkillLevel, mt, ra, mv))
					if err != nil {
						l.WithError(err).Errorf("Unable to move monster [%d] for character [%d].", event.UniqueId, s.CharacterId())
					}
					return err
				}
			}
		}
	}
}
