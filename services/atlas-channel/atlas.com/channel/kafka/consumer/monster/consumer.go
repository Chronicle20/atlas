package monster

import (
	consumer2 "atlas-channel/kafka/consumer"
	_map "atlas-channel/map"
	"atlas-channel/monster"
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
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("monster_status_event")(EnvEventTopicStatus)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser), consumer.SetStartOffset(kafka.LastOffset))
			rf(consumer2.NewConfig(l)("monster_movement_event")(EnvEventTopicMovement)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser), consumer.SetStartOffset(kafka.LastOffset))
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
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventDestroyed(sc, wp))))
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventDamaged(sc, wp))))
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventKilled(sc, wp))))
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventStartControl(sc, wp))))
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventStopControl(sc, wp))))
				t, _ = topic.EnvProvider(l)(EnvEventTopicMovement)()
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleMovementEvent(sc, wp))))
			}
		}
	}
}

func handleStatusEventCreated(sc server.Model, wp writer.Producer) message.Handler[statusEvent[statusEventCreatedBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e statusEvent[statusEventCreatedBody]) {
		if e.Type != EventStatusCreated {
			return
		}

		if !sc.Is(tenant.MustFromContext(ctx), world.Id(e.WorldId), channel.Id(e.ChannelId)) {
			return
		}

		m, err := monster.GetById(l)(ctx)(e.UniqueId)
		if err != nil {
			l.WithError(err).Errorf("Unable to retrieve the monster [%d] being spawned.", e.UniqueId)
			return
		}

		err = _map.ForSessionsInMap(l)(ctx)(sc.Map(_map2.Id(e.MapId)), spawnForSession(l)(ctx)(wp)(m))
		if err != nil {
			l.WithError(err).Errorf("Unable to spawn monster [%d] for characters in map [%d].", m.UniqueId(), e.MapId)
		}
	}
}

func spawnForSession(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(m monster.Model) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(m monster.Model) model.Operator[session.Model] {
		return func(wp writer.Producer) func(m monster.Model) model.Operator[session.Model] {
			return func(m monster.Model) model.Operator[session.Model] {
				return session.Announce(l)(ctx)(wp)(writer.SpawnMonster)(writer.SpawnMonsterBody(l, tenant.MustFromContext(ctx))(m, true))
			}
		}
	}
}

func handleStatusEventDestroyed(sc server.Model, wp writer.Producer) message.Handler[statusEvent[statusEventDestroyedBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e statusEvent[statusEventDestroyedBody]) {
		if e.Type != EventStatusDestroyed {
			return
		}

		if !sc.Is(tenant.MustFromContext(ctx), world.Id(e.WorldId), channel.Id(e.ChannelId)) {
			return
		}

		err := _map.ForSessionsInMap(l)(ctx)(sc.Map(_map2.Id(e.MapId)), destroyForSession(l)(ctx)(wp)(e.UniqueId))
		if err != nil {
			l.WithError(err).Errorf("Unable to destroy monster [%d] for characters in map [%d].", e.UniqueId, e.MapId)
		}
	}
}

func destroyForSession(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(uniqueId uint32) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(uniqueId uint32) model.Operator[session.Model] {
		return func(wp writer.Producer) func(uniqueId uint32) model.Operator[session.Model] {
			return func(uniqueId uint32) model.Operator[session.Model] {
				return session.Announce(l)(ctx)(wp)(writer.DestroyMonster)(writer.DestroyMonsterBody(l, tenant.MustFromContext(ctx))(uniqueId, writer.DestroyMonsterTypeFadeOut))
			}
		}
	}
}

func handleStatusEventDamaged(sc server.Model, wp writer.Producer) message.Handler[statusEvent[statusEventDamagedBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e statusEvent[statusEventDamagedBody]) {
		if e.Type != EventStatusDamaged {
			return
		}

		if !sc.Is(tenant.MustFromContext(ctx), world.Id(e.WorldId), channel.Id(e.ChannelId)) {
			return
		}

		m, err := monster.GetById(l)(ctx)(e.UniqueId)
		if err != nil {
			return
		}

		var idProvider = model.FixedProvider([]uint32{e.Body.ActorId})

		p, err := party.GetByMemberId(l)(ctx)(e.Body.ActorId)
		if err == nil {
			mimf := party.MemberInMap(sc.WorldId(), sc.ChannelId(), _map2.Id(e.MapId))
			mp := party.FilteredMemberProvider(mimf)(model.FixedProvider(p))
			idProvider = party.MemberToMemberIdMapper(mp)
		}

		err = session.ForEachByCharacterId(sc.Tenant(), sc.WorldId(), sc.ChannelId())(idProvider, session.Announce(l)(ctx)(wp)(writer.MonsterHealth)(writer.MonsterHealthBody(m)))
		if err != nil {
			l.WithError(err).Errorf("Unable to announce monster [%d] health.", e.UniqueId)
		}
	}
}

func handleStatusEventKilled(sc server.Model, wp writer.Producer) message.Handler[statusEvent[statusEventKilledBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e statusEvent[statusEventKilledBody]) {
		if e.Type != EventStatusKilled {
			return
		}

		if !sc.Is(tenant.MustFromContext(ctx), world.Id(e.WorldId), channel.Id(e.ChannelId)) {
			return
		}

		err := _map.ForSessionsInMap(l)(ctx)(sc.Map(_map2.Id(e.MapId)), killForSession(l)(ctx)(wp)(e.UniqueId))
		if err != nil {
			l.WithError(err).Errorf("Unable to kill monster [%d] for characters in map [%d].", e.UniqueId, e.MapId)
		}
	}
}

func killForSession(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(uniqueId uint32) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(uniqueId uint32) model.Operator[session.Model] {
		return func(wp writer.Producer) func(uniqueId uint32) model.Operator[session.Model] {
			return func(uniqueId uint32) model.Operator[session.Model] {
				return session.Announce(l)(ctx)(wp)(writer.DestroyMonster)(writer.DestroyMonsterBody(l, tenant.MustFromContext(ctx))(uniqueId, writer.DestroyMonsterTypeFadeOut))
			}
		}
	}
}

func handleStatusEventStartControl(sc server.Model, wp writer.Producer) message.Handler[statusEvent[statusEventStartControlBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e statusEvent[statusEventStartControlBody]) {
		if e.Type != EventStatusStartControl {
			return
		}

		if !sc.Is(tenant.MustFromContext(ctx), world.Id(e.WorldId), channel.Id(e.ChannelId)) {
			return
		}

		m, err := monster.GetById(l)(ctx)(e.UniqueId)
		if err != nil {
			l.WithError(err).Errorf("Unable to start control of monster [%d] for character [%d].", e.UniqueId, e.Body.ActorId)
			return
		}

		err = session.IfPresentByCharacterId(tenant.MustFromContext(ctx), sc.WorldId(), sc.ChannelId())(e.Body.ActorId, startControlForSession(l)(ctx)(wp)(m))
		if err != nil {
			l.WithError(err).Errorf("Unable to start control of monster [%d] for character [%d].", e.UniqueId, e.Body.ActorId)
		}
	}
}

func startControlForSession(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(m monster.Model) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(m monster.Model) model.Operator[session.Model] {
		return func(wp writer.Producer) func(m monster.Model) model.Operator[session.Model] {
			return func(m monster.Model) model.Operator[session.Model] {
				return session.Announce(l)(ctx)(wp)(writer.ControlMonster)(writer.StartControlMonsterBody(l, tenant.MustFromContext(ctx))(m, false))
			}
		}
	}
}

func handleStatusEventStopControl(sc server.Model, wp writer.Producer) message.Handler[statusEvent[statusEventStopControlBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e statusEvent[statusEventStopControlBody]) {
		if e.Type != EventStatusStopControl {
			return
		}

		if !sc.Is(tenant.MustFromContext(ctx), world.Id(e.WorldId), channel.Id(e.ChannelId)) {
			return
		}

		m, err := monster.GetById(l)(ctx)(e.UniqueId)
		if err != nil {
			l.WithError(err).Errorf("Unable to stop control of monster [%d] for character [%d].", e.UniqueId, e.Body.ActorId)
			return
		}

		err = session.IfPresentByCharacterId(tenant.MustFromContext(ctx), sc.WorldId(), sc.ChannelId())(e.Body.ActorId, stopControlForSession(l)(ctx)(wp)(m))
		if err != nil {
			l.WithError(err).Errorf("Unable to stop control of monster [%d] for character [%d].", e.UniqueId, e.Body.ActorId)
		}
	}
}

func stopControlForSession(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(m monster.Model) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(m monster.Model) model.Operator[session.Model] {
		return func(wp writer.Producer) func(m monster.Model) model.Operator[session.Model] {
			return func(m monster.Model) model.Operator[session.Model] {
				return session.Announce(l)(ctx)(wp)(writer.ControlMonster)(writer.StopControlMonsterBody(l, tenant.MustFromContext(ctx))(m))
			}
		}
	}
}

func handleMovementEvent(sc server.Model, wp writer.Producer) message.Handler[movementEvent] {
	return func(l logrus.FieldLogger, ctx context.Context, e movementEvent) {
		if !sc.Is(tenant.MustFromContext(ctx), world.Id(e.WorldId), channel.Id(e.ChannelId)) {
			return
		}

		m, err := monster.GetById(l)(ctx)(e.UniqueId)
		if err != nil {
			l.WithError(err).Errorf("Unable to retrieve the monster [%d] moving.", e.UniqueId)
			return
		}

		err = _map.ForOtherSessionsInMap(l)(ctx)(sc.Map(m.MapId()), e.ObserverId, showMovementForSession(l)(ctx)(wp)(e))
		if err != nil {
			l.WithError(err).Errorf("Unable to move monster [%d] for characters in map [%d].", e.UniqueId, m.MapId())
		}
	}
}

func showMovementForSession(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(event movementEvent) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(event movementEvent) model.Operator[session.Model] {
		return func(wp writer.Producer) func(event movementEvent) model.Operator[session.Model] {
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

					return session.Announce(l)(ctx)(wp)(writer.MoveMonster)(writer.MoveMonsterBody(l, tenant.MustFromContext(ctx))(event.UniqueId,
						false, event.SkillPossible, false, event.Skill, event.SkillId,
						event.SkillLevel, mt, ra, mv))(s)
				}
			}
		}
	}
}
