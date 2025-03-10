package pet

import (
	consumer2 "atlas-channel/kafka/consumer"
	"atlas-channel/server"
	"atlas-channel/socket/writer"
	"context"
	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/handler"
	"github.com/Chronicle20/atlas-kafka/message"
	"github.com/Chronicle20/atlas-kafka/topic"
	"github.com/Chronicle20/atlas-model/model"
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
			}
		}
	}
}

func handleSpawned(sc server.Model, wp writer.Producer) message.Handler[statusEvent[spawnedStatusEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e statusEvent[spawnedStatusEventBody]) {
		if e.Type != StatusEventTypeSpawned {
			return
		}

		//s, err := session.GetByCharacterId(sc.Tenant(), sc.WorldId(), sc.ChannelId())(e.OwnerId)
		//if err != nil {
		//	return
		//}
		//
		//p := pet.NewModelBuilder(e.PetId, 0, e.Body.TemplateId, e.Body.Name).
		//	SetSlot(e.Body.Slot).
		//	SetLevel(e.Body.Level).
		//	SetTameness(e.Body.Tameness).
		//	SetFullness(e.Body.Fullness).
		//	SetX(e.Body.X).
		//	SetY(e.Body.Y).
		//	SetStance(e.Body.Stance).
		//	Build()
		//
		//go func() {
		//	_ = session.Announce(l)(ctx)(wp)(writer.PetActivated)(writer.PetSpawnBody(l)(sc.Tenant())(s.CharacterId(), p))(s)
		//}()
		//go func() {
		//	_ = _map.ForOtherSessionsInMap(l)(ctx)(s.Map(), s.CharacterId(), session.Announce(l)(ctx)(wp)(writer.PetActivated)(writer.PetSpawnBody(l)(sc.Tenant())(s.CharacterId(), p)))
		//}()
	}
}
