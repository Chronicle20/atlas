package buff

import (
	"atlas-channel/character/buff"
	"atlas-channel/character/buff/stat"
	consumer2 "atlas-channel/kafka/consumer"
	buff2 "atlas-channel/kafka/message/buff"
	"atlas-channel/listener"
	_map "atlas-channel/map"
	"atlas-channel/server"
	"atlas-channel/session"
	socketHandler "atlas-channel/socket/handler"
	"atlas-channel/socket/writer"
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	charpkt "github.com/Chronicle20/atlas/libs/atlas-packet/character/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("character_buff_status_event")(buff2.EnvEventStatusTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser), consumer.SetStartOffset(kafka.LastOffset))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
	return func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
		return func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
			return func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
				var t string
				var handles []listener.HandlerHandle
				t, _ = topic.EnvProvider(l)(buff2.EnvEventStatusTopic)()
				id, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventApplied(sc, wp))))
				if err != nil {
					return nil, err
				}
				handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})
				id, err = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventExpired(sc, wp))))
				if err != nil {
					return nil, err
				}
				handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})
				id, err = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventBerserk(sc, wp))))
				if err != nil {
					return nil, err
				}
				handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})
				return handles, nil
			}
		}
	}
}

func handleStatusEventApplied(sc server.Model, wp writer.Producer) message.Handler[buff2.StatusEvent[buff2.AppliedStatusEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e buff2.StatusEvent[buff2.AppliedStatusEventBody]) {
		if e.Type != buff2.EventStatusTypeBuffApplied {
			return
		}

		if !sc.IsWorld(tenant.MustFromContext(ctx), e.WorldId) {
			return
		}

		session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(e.CharacterId, func(s session.Model) error {
			bs := make([]buff.Model, 0)
			changes := make([]stat.Model, 0)
			for _, cm := range e.Body.Changes {
				changes = append(changes, stat.NewStat(cm.Type, cm.Amount))
			}
			bs = append(bs, buff.NewBuff(e.Body.SourceId, e.Body.Level, e.Body.Duration, changes, e.Body.CreatedAt, e.Body.ExpiresAt))

			err := session.Announce(l)(ctx)(wp)(charpkt.CharacterBuffGiveWriter)(writer.CharacterBuffGiveBody(bs))(s)
			if err != nil {
				l.WithError(err).Errorf("Unable to write new character [%d] buffs.", e.CharacterId)
			}

			_ = _map.NewProcessor(l, ctx).ForOtherSessionsInMap(s.Field(), s.CharacterId(), func(os session.Model) error {
				err = session.Announce(l)(ctx)(wp)(charpkt.CharacterBuffGiveForeignWriter)(writer.CharacterBuffGiveForeignBody(e.CharacterId, bs))(os)
				if err != nil {
					l.WithError(err).Errorf("Unable to write new character [%d] buffs.", e.CharacterId)
					return err
				}
				return nil
			})
			return nil
		})
	}
}

func handleStatusEventExpired(sc server.Model, wp writer.Producer) message.Handler[buff2.StatusEvent[buff2.ExpiredStatusEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e buff2.StatusEvent[buff2.ExpiredStatusEventBody]) {
		if e.Type != buff2.EventStatusTypeBuffExpired {
			return
		}

		if !sc.IsWorld(tenant.MustFromContext(ctx), e.WorldId) {
			return
		}

		session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(e.CharacterId, func(s session.Model) error {
			ebs := make([]buff.Model, 0)
			changes := make([]stat.Model, 0)
			for _, cm := range e.Body.Changes {
				changes = append(changes, stat.NewStat(cm.Type, cm.Amount))
			}
			ebs = append(ebs, buff.NewBuff(e.Body.SourceId, e.Body.Level, e.Body.Duration, changes, e.Body.CreatedAt, e.Body.ExpiresAt))

			err := session.Announce(l)(ctx)(wp)(charpkt.CharacterBuffCancelWriter)(writer.CharacterBuffCancelBody(ebs))(s)
			if err != nil {
				l.WithError(err).Errorf("Unable to write character [%d] cancelled buffs.", e.CharacterId)
			}

			_ = _map.NewProcessor(l, ctx).ForOtherSessionsInMap(s.Field(), s.CharacterId(), func(os session.Model) error {
				err = session.Announce(l)(ctx)(wp)(charpkt.CharacterBuffCancelForeignWriter)(writer.CharacterBuffCancelForeignBody(e.CharacterId, ebs))(os)
				if err != nil {
					l.WithError(err).Errorf("Unable to write new character [%d] buffs.", e.CharacterId)
					return err
				}
				return nil
			})
			return nil
		})
	}
}

// handleStatusEventBerserk translates one berserk broadcast tick into the own
// + foreign EffectSkillUse packets (task-154). Stateless by design (D4):
// atlas-buffs owns the schedule; the periodic re-broadcast covers late-joining
// observers, so there is no map-enter hook. No session means the character
// transferred or logged out between emit and consume — the next tick
// self-corrects.
func handleStatusEventBerserk(sc server.Model, wp writer.Producer) message.Handler[buff2.StatusEvent[buff2.BerserkStatusEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e buff2.StatusEvent[buff2.BerserkStatusEventBody]) {
		if e.Type != buff2.EventStatusTypeBerserk {
			return
		}

		if !sc.Is(tenant.MustFromContext(ctx), e.WorldId, e.Body.ChannelId) {
			return
		}

		_ = session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(e.CharacterId, func(s session.Model) error {
			if err := socketHandler.AnnounceBerserkEffect(l)(ctx)(wp)(e.Body.SkillId, e.Body.CharacterLevel, e.Body.SkillLevel, e.Body.Active)(s); err != nil {
				l.WithError(err).Errorf("Unable to write berserk effect for character [%d].", e.CharacterId)
			}

			_ = _map.NewProcessor(l, ctx).ForOtherSessionsInMap(s.Field(), s.CharacterId(), socketHandler.AnnounceForeignBerserkEffect(l)(ctx)(wp)(e.CharacterId, e.Body.SkillId, e.Body.CharacterLevel, e.Body.SkillLevel, e.Body.Active))
			return nil
		})
	}
}
