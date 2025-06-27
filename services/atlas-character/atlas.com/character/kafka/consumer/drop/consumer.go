package drop

import (
	"atlas-character/character"
	consumer2 "atlas-character/kafka/consumer"
	drop2 "atlas-character/kafka/message/drop"
	"context"
	"github.com/Chronicle20/atlas-constants/field"
	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/handler"
	"github.com/Chronicle20/atlas-kafka/message"
	"github.com/Chronicle20/atlas-kafka/topic"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("drop_status_event")(drop2.EnvEventTopicDropStatus)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) {
		return func(rf func(topic string, handler handler.Handler) (string, error)) {
			var t string
			t, _ = topic.EnvProvider(l)(drop2.EnvEventTopicDropStatus)()
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleDropReservation(db))))
		}
	}
}

func handleDropReservation(db *gorm.DB) message.Handler[drop2.StatusEvent[drop2.ReservedStatusEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e drop2.StatusEvent[drop2.ReservedStatusEventBody]) {
		if e.Type != drop2.StatusEventTypeReserved {
			return
		}
		if e.Body.Meso > 0 {
			f := field.NewBuilder(e.WorldId, e.ChannelId, e.MapId).Build()
			_ = character.NewProcessor(l, ctx, db).AttemptMesoPickUp(f, e.Body.CharacterId, e.DropId, e.Body.Meso)
			return
		}
	}
}
