package character

import (
	"atlas-channel/character"
	consumer2 "atlas-channel/kafka/consumer"
	_map "atlas-channel/map"
	"atlas-channel/server"
	"atlas-channel/session"
	model2 "atlas-channel/socket/model"
	"atlas-channel/socket/writer"
	"atlas-channel/tenant"
	"context"
	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/handler"
	"github.com/Chronicle20/atlas-kafka/message"
	"github.com/Chronicle20/atlas-kafka/topic"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/sirupsen/logrus"
)

const consumerStatusEvent = "character_status"
const consumerMovementEvent = "character_movement"

func StatusEventConsumer(l logrus.FieldLogger) func(groupId string) consumer.Config {
	return func(groupId string) consumer.Config {
		return consumer2.NewConfig(l)(consumerStatusEvent)(EnvEventTopicCharacterStatus)(groupId)
	}
}

func StatusEventStatChangedRegister(sc server.Model, wp writer.Producer) func(l logrus.FieldLogger) (string, handler.Handler) {
	return func(l logrus.FieldLogger) (string, handler.Handler) {
		t, _ := topic.EnvProvider(l)(EnvEventTopicCharacterStatus)()
		return t, message.AdaptHandler(message.PersistentConfig(handleStatusEventStatChanged(sc, wp)))
	}
}

func StatusEventMapChangedRegister(sc server.Model, wp writer.Producer) func(l logrus.FieldLogger) (string, handler.Handler) {
	return func(l logrus.FieldLogger) (string, handler.Handler) {
		t, _ := topic.EnvProvider(l)(EnvEventTopicCharacterStatus)()
		return t, message.AdaptHandler(message.PersistentConfig(handleStatusEventMapChanged(sc, wp)))
	}
}

func handleStatusEventStatChanged(sc server.Model, wp writer.Producer) func(l logrus.FieldLogger, ctx context.Context, event statusEvent[statusEventStatChangedBody]) {
	return func(l logrus.FieldLogger, ctx context.Context, event statusEvent[statusEventStatChangedBody]) {
		if !sc.Is(event.Tenant, event.WorldId, event.Body.ChannelId) {
			return
		}

		if event.Type != EventCharacterStatusTypeStatChanged {
			return
		}

		session.IfPresentByCharacterId(event.Tenant, sc.WorldId(), sc.ChannelId())(event.CharacterId, statChanged(l, ctx, event.Tenant, wp)(event.Body.ExclRequestSent))
	}
}

func statChanged(l logrus.FieldLogger, _ context.Context, _ tenant.Model, wp writer.Producer) func(exclRequestSent bool) model.Operator[session.Model] {
	statChangedFunc := session.Announce(l)(wp)(writer.StatChanged)
	return func(exclRequestSent bool) model.Operator[session.Model] {
		return func(s session.Model) error {
			err := statChangedFunc(s, writer.StatChangedBody(l)(exclRequestSent))
			if err != nil {
				l.WithError(err).Errorf("Unable to write [%s] for character [%d].", writer.StatChanged, s.CharacterId())
				return err
			}
			return nil
		}
	}
}

func handleStatusEventMapChanged(sc server.Model, wp writer.Producer) func(l logrus.FieldLogger, ctx context.Context, event statusEvent[statusEventMapChangedBody]) {
	return func(l logrus.FieldLogger, ctx context.Context, event statusEvent[statusEventMapChangedBody]) {
		if !sc.Is(event.Tenant, event.WorldId, event.Body.ChannelId) {
			return
		}

		if event.Type != EventCharacterStatusTypeMapChanged {
			return
		}

		session.IfPresentByCharacterId(event.Tenant, sc.WorldId(), sc.ChannelId())(event.CharacterId, warpCharacter(l, ctx, event.Tenant, wp)(event))
	}
}

func warpCharacter(l logrus.FieldLogger, ctx context.Context, t tenant.Model, wp writer.Producer) func(event statusEvent[statusEventMapChangedBody]) model.Operator[session.Model] {
	setFieldFunc := session.Announce(l)(wp)(writer.SetField)
	return func(event statusEvent[statusEventMapChangedBody]) model.Operator[session.Model] {
		return func(s session.Model) error {
			c, err := character.GetById(l, ctx, t)(s.CharacterId())
			if err != nil {
				l.WithError(err).Errorf("Unable to retrieve character [%d].", s.CharacterId())
				return err
			}

			s = session.SetMapId(event.Body.TargetMapId)(s.Tenant().Id, s.SessionId())

			err = setFieldFunc(s, writer.WarpToMapBody(l, s.Tenant())(s.ChannelId(), event.Body.TargetMapId, event.Body.TargetPortalId, c.Hp()))
			if err != nil {
				l.WithError(err).Errorf("Unable to show set field response for character [%d]", c.Id())
				return err
			}
			return nil
		}
	}
}

func MovementEventConsumer(l logrus.FieldLogger) func(groupId string) consumer.Config {
	return func(groupId string) consumer.Config {
		return consumer2.NewConfig(l)(consumerMovementEvent)(EnvEventTopicMovement)(groupId)
	}
}

func MovementEventRegister(sc server.Model, wp writer.Producer) func(l logrus.FieldLogger) (string, handler.Handler) {
	return func(l logrus.FieldLogger) (string, handler.Handler) {
		t, _ := topic.EnvProvider(l)(EnvEventTopicMovement)()
		return t, message.AdaptHandler(message.PersistentConfig(handleMovementEvent(sc, wp)))
	}
}

func handleMovementEvent(sc server.Model, wp writer.Producer) message.Handler[movementEvent] {
	return func(l logrus.FieldLogger, ctx context.Context, event movementEvent) {
		if !sc.Is(event.Tenant, event.WorldId, event.ChannelId) {
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

		_map.ForOtherSessionsInMap(l, ctx, sc.Tenant())(sc.WorldId(), sc.ChannelId(), event.MapId, event.CharacterId, showMovementForSession(l, wp)(event.CharacterId, mv))
	}
}

func showMovementForSession(l logrus.FieldLogger, wp writer.Producer) func(characterId uint32, m model2.Movement) model.Operator[session.Model] {
	moveCharacterFunc := session.Announce(l)(wp)(writer.CharacterMovement)
	return func(characterId uint32, m model2.Movement) model.Operator[session.Model] {
		return func(s session.Model) error {
			err := moveCharacterFunc(s, writer.CharacterMovementBody(l, s.Tenant())(characterId, m))
			if err != nil {
				l.WithError(err).Errorf("Unable to move character [%d] for character [%d].", characterId, s.CharacterId())
			}
			return err
		}
	}
}
