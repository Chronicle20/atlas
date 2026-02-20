package party_quest

import (
	consumer2 "atlas-channel/kafka/consumer"
	pq "atlas-channel/kafka/message/party_quest"
	_map "atlas-channel/map"
	_mapId "github.com/Chronicle20/atlas-constants/map"
	"atlas-channel/server"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/handler"
	"github.com/Chronicle20/atlas-kafka/message"
	"github.com/Chronicle20/atlas-kafka/topic"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("party_quest_status_event")(pq.EnvEventStatusTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser), consumer.SetStartOffset(kafka.LastOffset))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) error {
	return func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) error {
		return func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) error {
			return func(rf func(topic string, handler handler.Handler) (string, error)) error {
				var t string
				t, _ = topic.EnvProvider(l)(pq.EnvEventStatusTopic)()
				if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleStageCleared(sc, wp)))); err != nil {
					return err
				}
				if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleCharacterLeft(sc, wp)))); err != nil {
					return err
				}
				return nil
			}
		}
	}
}

func handleStageCleared(sc server.Model, wp writer.Producer) message.Handler[pq.StatusEvent[pq.StageClearedEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e pq.StatusEvent[pq.StageClearedEventBody]) {
		if e.Type != pq.EventTypeStageCleared {
			return
		}

		if !sc.Is(tenant.MustFromContext(ctx), e.WorldId, e.Body.ChannelId) {
			return
		}

		l.Debugf("Party quest [%s] instance [%s] stage [%d] cleared.", e.QuestId, e.InstanceId, e.Body.StageIndex)

		for i, mid := range e.Body.MapIds {
			instance := uuid.Nil
			if i < len(e.Body.FieldInstances) {
				instance = e.Body.FieldInstances[i]
			}
			f := sc.Field(_mapId.Id(mid), instance)
			err := _map.NewProcessor(l, ctx).ForSessionsInMap(f, announceStageCleared(l, ctx, wp))
			if err != nil {
				l.WithError(err).Errorf("Unable to announce stage clear effects for map [%d].", mid)
			}
		}
	}
}

func announceStageCleared(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) model.Operator[session.Model] {
	return func(s session.Model) error {
		err := session.Announce(l)(ctx)(wp)(writer.FieldEffect)(writer.FieldEffectScreenBody(l)("quest/party/clear"))(s)
		if err != nil {
			return err
		}
		return session.Announce(l)(ctx)(wp)(writer.FieldEffect)(writer.FieldEffectSoundBody(l)("Party1/Clear"))(s)
	}
}

func handleCharacterLeft(sc server.Model, wp writer.Producer) message.Handler[pq.StatusEvent[pq.CharacterLeftEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e pq.StatusEvent[pq.CharacterLeftEventBody]) {
		if e.Type != pq.EventTypeCharacterLeft {
			return
		}

		if !sc.Is(tenant.MustFromContext(ctx), e.WorldId, e.Body.ChannelId) {
			return
		}

		l.Debugf("Character [%d] left party quest [%s] instance [%s]. Reason: %s.", e.Body.CharacterId, e.QuestId, e.InstanceId, e.Body.Reason)

		err := session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(e.Body.CharacterId, announceCharacterLeft(l, ctx, wp))
		if err != nil {
			l.WithError(err).Errorf("Unable to announce PQ departure to character [%d].", e.Body.CharacterId)
		}
	}
}

func announceCharacterLeft(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) model.Operator[session.Model] {
	return func(s session.Model) error {
		return session.Announce(l)(ctx)(wp)(writer.WorldMessage)(writer.WorldMessagePinkTextBody(l)("", "", "You have left the party quest."))(s)
	}
}
